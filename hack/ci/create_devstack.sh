#!/usr/bin/env bash

# Copyright 2021 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# 	http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# hack script for preparing devstack to run cluster-api-provider-openstack e2e
# This script invokes the ${RESOURCE_TYPE}.sh scripts to set up the infrastructure
# needed for running devstack.

set -x
set -o errexit -o nounset -o pipefail

if [ -z "${RESOURCE_TYPE}" ]; then
    echo "RESOURCE_TYPE must be defined"
    exit 1
fi

scriptdir=$(dirname "${BASH_SOURCE[0]}")
source "${scriptdir}/${RESOURCE_TYPE}.sh"

CLUSTER_NAME=${CLUSTER_NAME:-"capo-e2e"}

OPENSTACK_RELEASE=${OPENSTACK_RELEASE:-"zed"}
OPENSTACK_ENABLE_HORIZON=${OPENSTACK_ENABLE_HORIZON:-"false"}

# Devstack will create a provider network using this range
# We create a route to it with sshuttle
FLOATING_RANGE=${FLOATING_RANGE:-"172.24.4.0/24"}

# Servers will be directly attached to the private network
# We create a route to it with sshuttle
PRIVATE_NETWORK_CIDR=${PRIVATE_NETWORK_CIDR:-"10.0.3.0/24"}
CONTROLLER_IP=${CONTROLLER_IP:-"10.0.3.15"}
WORKER_IP=${WORKER_IP:-"10.0.3.16"}

PRIMARY_AZ=testaz1
SECONDARY_AZ=testaz2

# For apt-get
export DEBIAN_FRONTEND=noninteractive

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/../../
cd "${REPO_ROOT}" || exit 1
REPO_ROOT_ABSOLUTE=$(pwd)
ARTIFACTS=${ARTIFACTS:-/tmp/${CLUSTER_NAME}-artifacts}
devstackdir="${ARTIFACTS}/devstack"
mkdir -p "$devstackdir"

# retry $1 times with $2 sleep in between
function retry {
    attempt=0
    max_attempts=${1}
    interval=${2}
    shift; shift
    until [[ "$attempt" -ge "$max_attempts" ]] ; do
        attempt=$((attempt+1))
        set +e
        eval "$*" && return || echo "failed $attempt times: $*"
        set -e
        sleep "$interval"
    done
    echo "error: reached max attempts at retry($*)"
    return 1
}

function ensure_openstack_client {
    if ! command -v openstack;
    then
        # We are running in a Debian Buster image with python 3.7. Python 3.7
        # is starting to show its age in upstream support. Ideally we would
        # move to a newer image with a newer python version.
        #
        # Until then, this script tries to carefully navigate around current
        # issues running openstack client on python 3.7.
        #
        # We explicitly pin the zed version of openstackclient. This is the
        # last version of openstackclient which will support python 3.7.

        # Install virtualenv to install the openstack client and curl to fetch
        # the build constraints.
        apt-get install -y python3-virtualenv curl
        python3 -m virtualenv -p $(which python3) /tmp/openstack-venv
        VIRTUAL_ENV_DISABLE_PROMPT=1 source /tmp/openstack-venv/bin/activate

        # openstackclient has never actually supported python 3.7, only 3.6 and
        # 3.8. Here we download the zed constraints file and modify all the
        # 3.8 constraints to be 3.7 constraints.
        # curl -L https://releases.openstack.org/constraints/upper/zed -o /tmp/zed-constraints
        # sed -i "s/python_version=='3.8'/python_version=='3.7'/" /tmp/zed-constraints

        pip install \
                python-openstackclient python-cinderclient \
                python-glanceclient python-keystoneclient \
                python-neutronclient python-novaclient python-octaviaclient
    fi
}

function wait_for_ssh {
    local ip=$1 && shift

    retry 10 30 "$(get_ssh_cmd) ${ip} -- true"
}

function start_sshuttle {
    if ! command -v sshuttle;
    then
        pip3 install sshuttle
    fi

    kill_sshuttle

    # Open tunnel
    public_ip=$(get_public_ip)
    wait_for_ssh "$public_ip"
    echo "Opening tunnel to ${PRIVATE_NETWORK_CIDR} and ${FLOATING_RANGE} via ${public_ip}"
    # sshuttle won't succeed until ssh is up and python is installed on the destination
    retry 50 30 sshuttle -r "$public_ip" "$PRIVATE_NETWORK_CIDR" "$FLOATING_RANGE" --ssh-cmd=\""$(get_ssh_cmd)"\" -l 0.0.0.0 -D

    # Give sshuttle a few seconds to be fully up
    sleep 5
}

function kill_sshuttle {
    sshuttle_pidfile="${REPO_ROOT_ABSOLUTE}/sshuttle.pid"
    if [ -f "$sshuttle_pidfile" ]; then
        sshuttle_pid=$(cat "$sshuttle_pidfile")
        kill "$sshuttle_pid"
        while [ -d "/proc/$sshuttle_pid" ]; do
            echo "Waiting for sshuttle pid $sshuttle_pid to die"
            sleep 1
        done
    fi
}

function wait_for_devstack {
    local name=$1 && shift
    local ip=$1 && shift

    # Wait until cloud-init is done
    wait_for_ssh "$ip"

    ssh_cmd=$(get_ssh_cmd)

    $ssh_cmd "$ip" -- "
    echo Waiting for cloud-final to complete
    start=\$(date -u +%s)
    while true; do
       systemctl --quiet is-failed cloud-final && exit 1
       systemctl --quiet is-active cloud-final && exit 0
       echo Waited \$(((\$(date -u +%s)-\$start)/60)) minutes
       sleep 30
    done"

    # Flush the journal to ensure we get the final gasp of cloud-final if it
    # died
    $ssh_cmd "$ip" -- sudo journalctl --flush

    # Continuously capture devstack logs until killed
    $ssh_cmd "$ip" -- sudo journalctl --no-tail -a -b -u 'devstack@*' -f > "${devstackdir}/${name}-devstack.log" &

    $ssh_cmd "$ip" -- sudo journalctl --no-tail -a -b -u 'devstack@neutron-api.service' -f > "${devstackdir}/${name}-neutron.log" &

    # Capture cloud-init logs
    # Devstack logs are in cloud-final
    for service in cloud-config cloud-final cloud-init-local cloud-init; do
        $ssh_cmd "$ip" -- sudo journalctl -a -b -u "$service" > "${devstackdir}/${name}-${service}.log"

        # Fail early if any cloud-init service failed
        $ssh_cmd "$ip" -- sudo systemctl status --full "$service" || exit 1
    done
}

function get_ssh_cmd {
    local private_key_file=$(get_ssh_private_key_file)
    if [ -z "$private_key_file" ]; then
        # If there's no private key file use the public key instead
        # This allows us to specify a private key which is held only on a
        # hardware device and therefore has no key file
        private_key_file=$(get_ssh_public_key_file)
    fi

    echo "ssh -i ${private_key_file} -l cloud " \
         "-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o IdentitiesOnly=yes -o PasswordAuthentication=no "
}

function create_devstack {
    local name=$1 && shift
    local ip=$1 && shift
    local public=${1:-} && shift

    cloud_init="${devstackdir}/cloud-init-${name}.yaml"

    if [[ "$OPENSTACK_ENABLE_HORIZON" = "true" ]]
    then
        OPENSTACK_ADDITIONAL_SERVICES="${OPENSTACK_ADDITIONAL_SERVICES},horizon"
    fi

    if ! command -v envsubst;
    then
        apt-get update && apt-get install -y gettext
    fi

    # Ensure cloud-init exists and is empty
    truncate --size 0 "$cloud_init"

    for tpl in common "$name"; do
        SSH_PUBLIC_KEY="$(cat $(get_ssh_public_key_file))" \
        OPENSTACK_ADDITIONAL_SERVICES="${OPENSTACK_ADDITIONAL_SERVICES:-}" \
        OPENSTACK_RELEASE="$OPENSTACK_RELEASE" \
        HOST_IP="$ip" \
        CONTROLLER_IP="$CONTROLLER_IP" \
        FLOATING_RANGE="$FLOATING_RANGE" \
        MTU="$(get_mtu)" \
        PRIMARY_AZ="$PRIMARY_AZ" SECONDARY_AZ="$SECONDARY_AZ" \
            envsubst '${SSH_PUBLIC_KEY} ${OPENSTACK_ADDITIONAL_SERVICES}
                    ${OPENSTACK_RELEASE} ${HOST_IP} ${FLOATING_RANGE}
                    ${CONTROLLER_IP} ${MTU} ${PRIMARY_AZ} ${SECONDARY_AZ}' \
                < "./hack/ci/cloud-init/${tpl}.yaml.tpl" >> "$cloud_init"
    done

    create_vm "$name" "$ip" "$cloud_init" "$public"
}

function cleanup {
    kill_sshuttle
    cloud_cleanup
    exit 0
}

function create_worker {
    # Create the worker machine synchronously
    create_devstack worker "$WORKER_IP"

    # Wait and run post-install tasks asynchronously
    wait_for_devstack worker "$WORKER_IP" > "${devstackdir}/worker-build.log" 2>&1 &
}

function main() {
    if [ "${1:-}" == "cleanup" ]; then
        cleanup
    fi

    # Initialize the necessary infrastructure requirements
    cloud_init
    if [[ -n "${SKIP_INIT_INFRA:-}" ]]; then
        echo "Skipping infrastructure initialization..."
    else
        init_infrastructure
    fi

    # Create devstack VM.
    # devstack initialisation proceeds asynchronously in the VM.
    create_devstack controller "$CONTROLLER_IP" public

    # Install some local dependencies we later need in the meantime (we have to wait for cloud init anyway)
    ensure_openstack_client
    start_sshuttle

    wait_for_devstack controller "$CONTROLLER_IP"

    # At this point the controller is a fully functional OpenStack capable of
    # running tests which only require a single availability zone. Here we
    # create the worker VM, but we don't wait for devstack to finish installing
    # on it. This allows us to start running tests while the worker is still
    # installing, which takes some time to complete. The worker will be
    # automatically added as an additional compute and volume service in its own
    # availability zone when it completes.
    #
    # For robustness, tests which require multi-AZ MUST check that the second AZ
    # is available, and wait if it is not.
    #
    # For efficiency, tests which require multi-AZ SHOULD run as late as possible.
    create_worker

    public_ip=$(get_public_ip)
    cat << EOF > "${REPO_ROOT_ABSOLUTE}/clouds.yaml"
clouds:
  capo-e2e:
    auth:
      username: demo
      password: secretadmin
      user_domain_id: default
      auth_url: http://${public_ip}/identity
      domain_id: default
      project_name: demo
    verify: false
    region_name: RegionOne
  capo-e2e-admin:
    auth:
      username: admin
      password: secretadmin
      user_domain_id: default
      auth_url: http://${public_ip}/identity
      domain_id: default
      project_name: admin
    verify: false
    region_name: RegionOne
EOF

    export OS_CLOUD="capo-e2e-admin"

    # Wait until the OpenStack API is reachable
    retry 5 30 "openstack versions show"

    # Log some useful info
    openstack hypervisor stats show
    openstack host list
    openstack usage list
    openstack project list
    openstack network list
    openstack subnet list
    openstack image list
    openstack flavor list
    openstack server list
    openstack availability zone list
    openstack domain list

    echo "${REPO_ROOT_ABSOLUTE}/clouds.yaml:"
    cat "${REPO_ROOT_ABSOLUTE}/clouds.yaml"
}

main "$@"
