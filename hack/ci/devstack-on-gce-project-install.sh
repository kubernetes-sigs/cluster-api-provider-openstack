#!/usr/bin/env bash

# Copyright 2021 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# hack script for preparing GCP to run cluster-api-provider-openstack e2e

set -o errexit -o nounset -o pipefail

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/../../
cd "${REPO_ROOT}" || exit 1
REPO_ROOT_ABSOLUTE=$(pwd)

CLUSTER_NAME=${CLUSTER_NAME:-"capo-e2e"}

GOOGLE_APPLICATION_CREDENTIALS=${GOOGLE_APPLICATION_CREDENTIALS:-""}
GCP_PROJECT=${GCP_PROJECT:-""}
GCP_REGION=${GCP_REGION:-"us-east4"}
GCP_ZONE=${GCP_ZONE:-"us-east4-a"}
GCP_MACHINE_MIN_CPU_PLATFORM=${GCP_MACHINE_MIN_CPU_PLATFORM:-"Intel Cascade Lake"}
GCP_MACHINE_TYPE=${GCP_MACHINE_TYPE:-"n2-standard-16"}
GCP_NETWORK_NAME=${GCP_NETWORK_NAME:-"${CLUSTER_NAME}-mynetwork"}

OPENSTACK_RELEASE=${OPENSTACK_RELEASE:-"victoria"}
OPENSTACK_ENABLE_HORIZON=${OPENSTACK_ENABLE_HORIZON:-"false"}
# Flavors are default or preinstalled:
# * default: installs devstack via cloud-init
#   * OPENSTACK_RELEASE only works on default
# * preinstalled: uses a already installed devstack
FLAVOR=${FLAVOR:="preinstalled"}

echo "Using: GCP_PROJECT: ${GCP_PROJECT} GCP_REGION: ${GCP_REGION} GCP_NETWORK_NAME: ${GCP_NETWORK_NAME}"

# retry $1 times with $2 sleep in between
function retry {
  attempt=0
  max_attempts=${1}
  interval=${2}
  shift; shift
  until [[ ${attempt} -ge "${max_attempts}" ]] ; do
    attempt=$((attempt+1))
    set +e
    eval "$*" && return || echo "failed ${attempt} times: $*"
    set -e
    sleep "${interval}"
  done
  echo "error: reached max attempts at retry($*)"
  return 1
}

function init_networks() {
  if [[ ${GCP_NETWORK_NAME} != "default" ]]; then
    if ! gcloud compute networks describe "${GCP_NETWORK_NAME}" --project "${GCP_PROJECT}" > /dev/null;
    then
      gcloud compute networks create --project "$GCP_PROJECT" "${GCP_NETWORK_NAME}" --subnet-mode custom
      gcloud compute networks subnets create "${GCP_NETWORK_NAME}" --project "$GCP_PROJECT" --network="${GCP_NETWORK_NAME}" --range="10.0.0.0/20" --region "${GCP_REGION}"

      gcloud compute firewall-rules create "${GCP_NETWORK_NAME}"-allow-http --project "$GCP_PROJECT" \
        --allow tcp:80 --network "${GCP_NETWORK_NAME}" --quiet
      gcloud compute firewall-rules create "${GCP_NETWORK_NAME}"-allow-https --project "$GCP_PROJECT" \
        --allow tcp:443 --network "${GCP_NETWORK_NAME}" --quiet
      gcloud compute firewall-rules create "${GCP_NETWORK_NAME}"-allow-icmp --project "$GCP_PROJECT" \
        --allow icmp --network "${GCP_NETWORK_NAME}" --priority 65534 --quiet
      gcloud compute firewall-rules create "${GCP_NETWORK_NAME}"-allow-internal --project "$GCP_PROJECT" \
        --allow "tcp:0-65535,udp:0-65535,icmp" --network "${GCP_NETWORK_NAME}" --priority 65534 --quiet
      gcloud compute firewall-rules create "${GCP_NETWORK_NAME}"-allow-rdp --project "$GCP_PROJECT" \
        --allow "tcp:3389" --network "${GCP_NETWORK_NAME}" --priority 65534 --quiet
      gcloud compute firewall-rules create "${GCP_NETWORK_NAME}"-allow-ssh --project "$GCP_PROJECT" \
        --allow "tcp:22" --network "${GCP_NETWORK_NAME}" --priority 65534 --quiet
    fi
  fi

  gcloud compute firewall-rules list --project "$GCP_PROJECT"
  gcloud compute networks list --project="${GCP_PROJECT}"
  gcloud compute networks describe "${GCP_NETWORK_NAME}" --project="${GCP_PROJECT}"

  if ! gcloud compute routers describe "${CLUSTER_NAME}-myrouter" --project="${GCP_PROJECT}" --region="${GCP_REGION}" > /dev/null;
  then
    gcloud compute routers create "${CLUSTER_NAME}-myrouter" --project="${GCP_PROJECT}" \
    --region="${GCP_REGION}" --network="${GCP_NETWORK_NAME}"
  fi
  if ! gcloud compute routers nats describe --router="${CLUSTER_NAME}-myrouter" "${CLUSTER_NAME}-mynat" --project="${GCP_PROJECT}" --region="${GCP_REGION}" > /dev/null;
  then
  gcloud compute routers nats create "${CLUSTER_NAME}-mynat" --project="${GCP_PROJECT}" \
    --router-region="${GCP_REGION}" --router="${CLUSTER_NAME}-myrouter" \
    --nat-all-subnet-ip-ranges --auto-allocate-nat-external-ips
  fi
}

main() {
  # Initialize the necessary network requirements
  if [[ -n "${SKIP_INIT_NETWORK:-}" ]]; then
    echo "Skipping network initialization..."
  else
    init_networks
  fi

  if [[ ${FLAVOR} = "default" ]]
  then
    if ! gcloud compute disks describe devstack-${FLAVOR} --project "${GCP_PROJECT}" --zone "${GCP_ZONE}" > /dev/null;
    then
      gcloud compute disks create devstack-${FLAVOR} \
        --project "${GCP_PROJECT}" \
        --image-project ubuntu-os-cloud --image-family ubuntu-2004-lts \
        --zone "${GCP_ZONE}"
    fi

    if ! gcloud compute images describe devstack-${FLAVOR} --project "${GCP_PROJECT}" > /dev/null;
    then
      gcloud compute images create devstack-${FLAVOR} \
        --project "${GCP_PROJECT}" \
        --source-disk devstack-${FLAVOR} --source-disk-zone "${GCP_ZONE}" \
        --licenses "https://www.googleapis.com/compute/v1/projects/vm-options/global/licenses/enable-vmx"
    fi
  fi

  if [[ ${FLAVOR} = "preinstalled" ]]
  then
    if ! gcloud compute images describe devstack-${FLAVOR} --project "${GCP_PROJECT}" > /dev/null;
    then
      gcloud compute images create devstack-${FLAVOR} \
        --project "${GCP_PROJECT}" \
        --source-uri gs://artifacts.k8s-staging-capi-openstack.appspot.com/test/devstack/2021-03-28/devstack.raw.tar.gz \
        --licenses "https://www.googleapis.com/compute/v1/projects/vm-options/global/licenses/enable-vmx"
    fi
  fi

  if ! gcloud compute instances describe openstack --project "${GCP_PROJECT}" --zone "${GCP_ZONE}" > /dev/null;
  then
    < ./hack/ci/devstack-${FLAVOR}-cloud-init.yaml.tpl \
	    sed "s|\${OPENSTACK_ENABLE_HORIZON}|${OPENSTACK_ENABLE_HORIZON}|" | \
      sed "s|\${OPENSTACK_RELEASE}|${OPENSTACK_RELEASE}|" \
	    > ./hack/ci/devstack-${FLAVOR}-cloud-init.yaml

    gcloud compute instances create openstack \
      --project "${GCP_PROJECT}" \
      --zone "${GCP_ZONE}" \
      --image devstack-${FLAVOR} \
      --boot-disk-size 200G \
      --boot-disk-type pd-ssd \
      --can-ip-forward \
      --tags http-server,https-server,novnc,openstack-apis \
      --min-cpu-platform "${GCP_MACHINE_MIN_CPU_PLATFORM}" \
      --machine-type "${GCP_MACHINE_TYPE}" \
      --network-interface="private-network-ip=10.0.2.15,network=${CLUSTER_NAME}-mynetwork,subnet=${CLUSTER_NAME}-mynetwork" \
      --metadata-from-file user-data=./hack/ci/devstack-${FLAVOR}-cloud-init.yaml
  fi

  # Install some local dependencies we later need in the meantime (we have to wait for cloud init anyway)
  if ! command -v sshuttle;
  then
    # Install sshuttle from source because we need: https://github.com/sshuttle/sshuttle/pull/661
    # TODO(sbueringer) install via pip after the next release after 1.0.5 via:
    # pip3 install sshuttle
    cd /tmp
    git clone https://github.com/sshuttle/sshuttle.git
    cd sshuttle
    pip3 install .
    cd "${REPO_ROOT_ABSOLUTE}" || exit 1
  fi
  if ! command -v openstack;
  then
    apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y python3-dev
    # install PyYAML first because otherwise we get an error because pip3 doesn't upgrade PyYAML to the correct version
    # ERROR: Cannot uninstall 'PyYAML'. It is a distutils installed project and thus we cannot accurately determine which
    # files belong to it which would lead to only a partial uninstall.
    pip3 install --ignore-installed PyYAML
    pip3 install python-cinderclient python-glanceclient python-keystoneclient python-neutronclient python-novaclient python-openstackclient python-octaviaclient
  fi

  PUBLIC_IP=$(gcloud compute instances describe openstack --project "${GCP_PROJECT}" --zone "${GCP_ZONE}" --format='get(networkInterfaces[0].accessConfigs[0].natIP)')
  PRIVATE_IP=$(gcloud compute instances describe openstack --project "${GCP_PROJECT}" --zone "${GCP_ZONE}" --format='get(networkInterfaces[0].networkIP)')

  # Wait until cloud-init is done
  retry 120 30 "gcloud compute ssh --project ${GCP_PROJECT} --zone ${GCP_ZONE} openstack -- cat /var/lib/cloud/instance/boot-finished"

  # Open tunnel
  echo "Opening tunnel to ${PRIVATE_IP} via ${PUBLIC_IP}"
  sshuttle -r "${PUBLIC_IP}" "${PRIVATE_IP}/32" 172.24.4.0/24 --ssh-cmd='ssh -i ~/.ssh/google_compute_engine -o "StrictHostKeyChecking no" -o "UserKnownHostsFile=/dev/null" -o "IdentitiesOnly=yes"' -l 0.0.0.0 -D

  export OS_REGION_NAME=RegionOne
  export OS_PROJECT_DOMAIN_ID=default
  export OS_AUTH_URL=http://${PRIVATE_IP}/identity
  export OS_TENANT_NAME=admin
  export OS_USER_DOMAIN_ID=default
  export OS_USERNAME=admin
  export OS_PROJECT_NAME=admin
  export OS_PASSWORD=secretadmin
  export OS_IDENTITY_API_VERSION=3

  # Wait until the OpenStack API is reachable
  retry 120 30 "openstack versions show"

  nova hypervisor-stats
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

  # the flavors are created in a way that we can execute at least 2 e2e tests in parallel (overall we have 32 vCPUs)
  openstack flavor delete m1.tiny
  openstack flavor create --ram 512 --disk 1 --vcpus 1 --public --id 1 m1.tiny --property hw_rng:allowed='True'
  openstack flavor delete m1.small
  openstack flavor create --ram 4192 --disk 10 --vcpus 2 --public --id 2 m1.small --property hw_rng:allowed='True'
  openstack flavor delete m1.medium
  openstack flavor create --ram 6144 --disk 10 --vcpus 4 --public --id 3 m1.medium --property hw_rng:allowed='True'

  # Adjust the CPU quota
  openstack quota set --cores 32 demo
  openstack quota set --secgroups 50 demo

  export OS_TENANT_NAME=demo
  export OS_USERNAME=demo
  export OS_PROJECT_NAME=demo

  cat << EOF > "${REPO_ROOT_ABSOLUTE}/clouds.yaml"
clouds:
  ${CLUSTER_NAME}:
    auth:
      username: ${OS_USERNAME}
      password: ${OS_PASSWORD}
      user_domain_id: ${OS_USER_DOMAIN_ID}
      auth_url: ${OS_AUTH_URL}
      domain_id: default
      project_name: demo
    verify: false
    region_name: RegionOne
EOF
  echo "${REPO_ROOT_ABSOLUTE}/clouds.yaml:"
  cat "${REPO_ROOT_ABSOLUTE}/clouds.yaml"
}

main "$@"
