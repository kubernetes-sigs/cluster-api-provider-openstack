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

OPENSTACK_RELEASE=${OPENSTACK_RELEASE:-"2026.1"}
OPENSTACK_ENABLE_HORIZON=${OPENSTACK_ENABLE_HORIZON:-"false"}

# Devstack will create a provider network using this range
# We create a route to it with sshuttle
FLOATING_RANGE=${FLOATING_RANGE:-"172.24.4.0/24"}

# That will be the default DNS server for the Neutron subnets
OPENSTACK_DNS_NAMESERVERS=${OPENSTACK_DNS_NAMESERVERS:-"8.8.8.8"}

# Servers will be directly attached to the private network
# We create a route to it with sshuttle
PRIVATE_NETWORK_CIDR=${PRIVATE_NETWORK_CIDR:-"10.0.3.0/24"}
CONTROLLER_IP=${CONTROLLER_IP:-"10.0.3.15"}
WORKER_IP=${WORKER_IP:-"10.0.3.16"}

SKIP_INIT_INFRA=${SKIP_INIT_INFRA:-}
SKIP_SECONDARY_AZ=${SKIP_SECONDARY_AZ:-}

PRIMARY_AZ=testaz1
SECONDARY_AZ=testaz2

# Maximum time to wait for cloud-init (and therefore devstack) to complete on a
# single VM. Healthy runs have been observed to take between 15 and 70 minutes.
# This must stay well below the Prow job timeout (5h) to leave room for the
# tests themselves. Without it a hung bootstrap is only stopped by the Prow job
# timeout.
DEVSTACK_WAIT_TIMEOUT_MINUTES=${DEVSTACK_WAIT_TIMEOUT_MINUTES:-90}

# For apt-get
export DEBIAN_FRONTEND=noninteractive

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/../../
cd "${REPO_ROOT}" || exit 1
REPO_ROOT_ABSOLUTE=$(pwd)
ARTIFACTS=${ARTIFACTS:-/tmp/${CLUSTER_NAME}-artifacts}
devstackdir="${ARTIFACTS}/devstack"
mkdir -p "$devstackdir"

source "${REPO_ROOT}/hack/ci/common.sh"

function ensure_openstack_client {
    if ! command -v openstack;
    then
        # Install virtualenv to install the openstack client and curl to fetch
        # the build constraints.
        apt-get install -y python3-virtualenv curl
        python3 -m virtualenv -p $(which python3) /tmp/openstack-venv
        VIRTUAL_ENV_DISABLE_PROMPT=1 source /tmp/openstack-venv/bin/activate

        # We explicitly pin to the stable branch version of openstackclient.
        curl -L https://releases.openstack.org/constraints/upper/${OPENSTACK_RELEASE} -o /tmp/openstack-constraints

        pip install -c /tmp/openstack-constraints \
                python-openstackclient python-cinderclient \
                python-glanceclient python-keystoneclient \
                python-neutronclient python-novaclient python-octaviaclient

        # Ensure openstack cli can load fully including all plugins
        openstack help >/dev/null
    fi
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

# Run a command on the VM, saving its stdout to a file. This never fails and is
# bounded in time so that it can safely be used on a broken or overloaded VM.
# stderr is saved next to it with an .err suffix, if there is any.
function capture_remote {
    local ip=$1 && shift
    local outfile=$1 && shift

    local ssh_cmd
    ssh_cmd="$(get_ssh_cmd) -o LogLevel=ERROR -o ConnectTimeout=30 -o ServerAliveInterval=15 -o ServerAliveCountMax=4"

    local rc=0
    # shellcheck disable=SC2086 # ssh_cmd must be word split
    timeout 300 $ssh_cmd -n "$ip" -- "$@" > "$outfile" 2> "${outfile}.err" || rc=$?
    if [ "$rc" -ne 0 ]; then
        echo "WARNING: capturing ${outfile} failed with exit code ${rc}"
    fi
    [ -s "${outfile}.err" ] || rm -f "${outfile}.err"
    return 0
}

# Like capture_remote, but the command is a shell script which is run as root.
function capture_remote_script {
    local ip=$1 && shift
    local outfile=$1 && shift
    local script=$1 && shift

    local ssh_cmd
    ssh_cmd="$(get_ssh_cmd) -o LogLevel=ERROR -o ConnectTimeout=30 -o ServerAliveInterval=15 -o ServerAliveCountMax=4"

    local rc=0
    # shellcheck disable=SC2086 # ssh_cmd must be word split
    printf '%s\n' "$script" | timeout 300 $ssh_cmd "$ip" -- sudo sh -s > "$outfile" 2> "${outfile}.err" || rc=$?
    if [ "$rc" -ne 0 ]; then
        echo "WARNING: capturing ${outfile} failed with exit code ${rc}"
    fi
    [ -s "${outfile}.err" ] || rm -f "${outfile}.err"
    return 0
}

# Collect as much information as possible from the VM to help debug
# infrastructure problems. This is called on success as well as on failure so
# that we have a baseline to compare against. It must never fail.
function collect_diagnostics {
    local name=$1 && shift
    local ip=$1 && shift

    local dir="${devstackdir}/${name}-diagnostics"
    mkdir -p "$dir"

    echo "Collecting diagnostics from ${name} (${ip}) into ${dir}"

    # The full journal, including all kernel messages.
    capture_remote "$ip" "${dir}/journal.log" "sudo journalctl -b -a --no-pager"
    capture_remote "$ip" "${dir}/kernel.log" "sudo journalctl -k -b -a --no-pager"
    capture_remote "$ip" "${dir}/dmesg.log" "sudo dmesg -T"

    # Log files which are not in the journal: stack.sh log, OVN/OVS logs (when
    # built from source they are installed under /usr/local), cloud-init logs
    # and the resource sampler started from cloud-init.
    capture_remote_script "$ip" "${dir}/logs.tar.gz" '
        tar czf - --ignore-failed-read -C / \
            opt/stack/logs \
            var/log/ovn var/log/openvswitch \
            usr/local/var/log/ovn usr/local/var/log/openvswitch \
            var/log/capo-sampler.log var/log/cloud-init.log var/log/cloud-init-output.log 2>/dev/null'

    # Current state of the machine
    capture_remote_script "$ip" "${dir}/state.txt" '
        run() { echo "=== $*"; "$@" 2>&1; }
        run date -u
        run uptime
        run uname -a
        run cat /proc/cmdline
        run lscpu
        run free -m
        run grep -E "Dirty|Writeback" /proc/meminfo
        run df -h
        run tail -n 3 /proc/pressure/cpu /proc/pressure/io /proc/pressure/memory
        run dpkg-query -W -f "\${Package} \${Version}\n" "linux-image-*" systemd rsyslog cloud-init openssh-server sudo "google-*"
        run systemctl show cloud-final -p StandardOutput -p StandardError
        run systemd-analyze cat-config systemd/journald.conf
        run journalctl --disk-usage
        run stty -F /dev/ttyS0 -a
        run systemctl list-units --failed --no-pager
        run systemctl status --no-pager --full "devstack@*" "ovn*" "ovs*" "openvswitch*" "ovsdb*"
        run ss -tanp
        run ps aux --sort=-%cpu
        run systemd-analyze blame --no-pager'
}

# Wait until cloud-final has completed on the VM, collecting logs from it.
# Returns 0 on success, non-zero if cloud-init failed or did not complete within
# DEVSTACK_WAIT_TIMEOUT_MINUTES. Logs and diagnostics are collected in all cases.
#
# NOTE: Don't rely on errexit in this function. retry (common.sh) leaves it
# disabled, and it is not applied in conditional contexts anyway.
function wait_for_devstack {
    local name=$1 && shift
    local ip=$1 && shift

    local ssh_cmd
    ssh_cmd="$(get_ssh_cmd) -o LogLevel=ERROR"

    # Wait until ssh is up
    if ! wait_for_ssh "$ip"; then
        echo "ERROR: ${name} (${ip}) did not become reachable by ssh"
        return 1
    fi

    echo "Waiting for cloud-final to complete on ${name} (timeout: ${DEVSTACK_WAIT_TIMEOUT_MINUTES} minutes)"
    local start deadline status now poll_start
    start=$(date -u +%s)
    deadline=$((start + DEVSTACK_WAIT_TIMEOUT_MINUTES * 60))

    # Don't trace the polling loop, it would add a dozen lines per poll.
    { set +x; } 2>/dev/null
    while true; do
        poll_start=$(date -u +%s)
        # Exit codes: 0 done, 10 cloud-final failed, 11 still running.
        # Anything else (e.g. 255) is a problem connecting, which we tolerate
        # until the deadline.
        status=0
        # shellcheck disable=SC2086 # ssh_cmd must be word split
        $ssh_cmd -o ConnectTimeout=30 -o ServerAliveInterval=15 -o ServerAliveCountMax=4 "$ip" -- "
            systemctl --quiet is-failed cloud-final && exit 10
            systemctl --quiet is-active cloud-final && exit 0
            exit 11" || status=$?
        now=$(date -u +%s)

        if [ "$status" -eq 0 ]; then
            echo "cloud-final completed on ${name} after $(((now - start) / 60)) minutes"
            break
        elif [ "$status" -eq 10 ]; then
            echo "ERROR: cloud-final failed on ${name}"
            break
        fi

        if [ "$now" -ge "$deadline" ]; then
            echo "ERROR: Timed out waiting for cloud-final on ${name} after ${DEVSTACK_WAIT_TIMEOUT_MINUTES} minutes"
            status=124
            break
        fi
        # A slow poll (or a non-11 status) means the VM was not responding
        echo "Waited $(((now - start) / 60)) minutes (ssh exit status ${status}, poll took $((now - poll_start))s)"
        sleep 30
    done
    set -x

    # Everything below is best effort, we always want the logs.

    # Flush the journal to ensure we get the final gasp of cloud-final if it
    # died
    $ssh_cmd -o ConnectTimeout=30 "$ip" -- sudo journalctl --flush || true

    # Continuously capture devstack logs until killed
    $ssh_cmd "$ip" -- sudo journalctl --no-tail -a -b -u 'devstack@*' -f > "${devstackdir}/${name}-devstack.log" &

    # Capture cloud-init logs
    # Devstack logs are in cloud-final
    local service
    for service in cloud-config cloud-final cloud-init-local cloud-init; do
        capture_remote "$ip" "${devstackdir}/${name}-${service}.log" "sudo journalctl -a -b -u ${service}"

        # Fail if any cloud-init service failed
        $ssh_cmd -o ConnectTimeout=30 "$ip" -- sudo systemctl status --full "$service" || [ "$status" -ne 0 ] || status=1
    done

    collect_diagnostics "$name" "$ip"

    return "$status"
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
	    OPENSTACK_DNS_NAMESERVERS="$OPENSTACK_DNS_NAMESERVERS" \
        MTU="$(get_mtu)" \
        PRIMARY_AZ="$PRIMARY_AZ" SECONDARY_AZ="$SECONDARY_AZ" \
            envsubst '${SSH_PUBLIC_KEY} ${OPENSTACK_ADDITIONAL_SERVICES}
                    ${OPENSTACK_RELEASE} ${HOST_IP} ${FLOATING_RANGE} ${OPENSTACK_DNS_NAMESERVERS}
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

    if ! wait_for_devstack controller "$CONTROLLER_IP"; then
        echo "ERROR: Failed to create a working devstack controller"
        exit 1
    fi

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
    if [[ -n "${SKIP_SECONDARY_AZ:-}" ]]; then
        echo "Skipping worker creation..."
    else
        create_worker
    fi

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
