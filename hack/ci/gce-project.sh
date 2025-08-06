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

# hack script for preparing GCP to run cluster-api-provider-openstack e2e

set -x -o errexit -o nounset -o pipefail

function cloud_init {
  GOOGLE_APPLICATION_CREDENTIALS=${GOOGLE_APPLICATION_CREDENTIALS:-""}
  GCP_PROJECT=${GCP_PROJECT:-""}
  GCP_REGION=${GCP_REGION:-"us-east4"}
  GCP_MACHINE_MIN_CPU_PLATFORM=${GCP_MACHINE_MIN_CPU_PLATFORM:-"Intel Cascade Lake"}
  GCP_NETWORK_NAME=${GCP_NETWORK_NAME:-"${CLUSTER_NAME}-mynetwork"}

  # We have a quota of 24 vCPUs
  GCP_MACHINE_TYPE_controller=${GCP_MACHINE_TYPE:-"n2-standard-16"}
  GCP_MACHINE_TYPE_worker=${GCP_MACHINE_TYPE:-"n2-standard-8"}

  echo "Using: GCP_PROJECT: ${GCP_PROJECT} GCP_REGION: ${GCP_REGION} GCP_NETWORK_NAME: ${GCP_NETWORK_NAME}"

  # Generate local ssh configuration
  # NOTE(mdbooth): This command successfully populates ssh config and then
  # fails for some reason I don't understand. We ignore the failure.
  gcloud compute config-ssh >/dev/null 2>&1 || true
}

function init_infrastructure() {
  if [[ ${GCP_NETWORK_NAME} != "default" ]]; then
    if ! gcloud compute networks describe "$GCP_NETWORK_NAME" --project "$GCP_PROJECT" >/dev/null 2>&1; then
      gcloud compute networks create --project "$GCP_PROJECT" "$GCP_NETWORK_NAME" --subnet-mode custom
      gcloud compute networks subnets create "$GCP_NETWORK_NAME" --project "$GCP_PROJECT" \
        --network="$GCP_NETWORK_NAME" --range="$PRIVATE_NETWORK_CIDR" --region "$GCP_REGION"

      gcloud compute firewall-rules create "${GCP_NETWORK_NAME}-allow-http" --project "$GCP_PROJECT" \
        --allow tcp:80 --direction=INGRESS --network "$GCP_NETWORK_NAME" --quiet
      # As of Victoria, neutron is the only service which isn't multiplexed by
      # apached on port 80
      gcloud compute firewall-rules create "${GCP_NETWORK_NAME}-allow-neutron" --project "$GCP_PROJECT" \
        --allow tcp:9696 --direction=INGRESS --network "$GCP_NETWORK_NAME" --quiet
      gcloud compute firewall-rules create "${GCP_NETWORK_NAME}-allow-icmp" --project "$GCP_PROJECT" \
        --allow icmp --direction=INGRESS --network "$GCP_NETWORK_NAME" --priority 65534 --quiet
      gcloud compute firewall-rules create "${GCP_NETWORK_NAME}-allow-ssh" --project "$GCP_PROJECT" \
        --allow "tcp:22" --direction=INGRESS --network "$GCP_NETWORK_NAME" --priority 65534 --quiet
      gcloud compute firewall-rules create "${GCP_NETWORK_NAME}-allow-internal" --project "$GCP_PROJECT" \
        --allow "tcp:0-65535,udp:0-65535,icmp" --source-ranges="$PRIVATE_NETWORK_CIDR" \
        --direction=INGRESS --network "$GCP_NETWORK_NAME" --priority 65534 --quiet
    fi
  fi

  gcloud compute firewall-rules list --project "$GCP_PROJECT"
  gcloud compute networks list --project="$GCP_PROJECT"
  gcloud compute networks describe "$GCP_NETWORK_NAME" --project="$GCP_PROJECT"

  if ! gcloud compute routers describe "${CLUSTER_NAME}-myrouter" --project="$GCP_PROJECT" --region="$GCP_REGION" >/dev/null 2>&1; then
    gcloud compute routers create "${CLUSTER_NAME}-myrouter" --project="$GCP_PROJECT" \
      --region="$GCP_REGION" --network="$GCP_NETWORK_NAME"
  fi
  if ! gcloud compute routers nats describe --router="$CLUSTER_NAME-myrouter" "$CLUSTER_NAME-mynat" \
    --project="$GCP_PROJECT" --region="${GCP_REGION}" >/dev/null 2>&1; then
    gcloud compute routers nats create "${CLUSTER_NAME}-mynat" --project="$GCP_PROJECT" \
      --router-region="$GCP_REGION" --router="${CLUSTER_NAME}-myrouter" \
      --nat-all-subnet-ip-ranges --auto-allocate-nat-external-ips
  fi
}

function create_vm {
  local name=$1 && shift
  local ip=$1 && shift
  local userdata=$1 && shift
  local public=$1 && shift # Unused by GCE

  local machine_type="GCP_MACHINE_TYPE_${name}"
  machine_type=${!machine_type}
  local servername="${CLUSTER_NAME}-${name}"

  # Loop over all zones in the GCP region to ignore a full zone.
  # We are not able to use 'gcloud compute zones list' as the gcloud.compute.zones.list permission is missing.
  for GCP_ZONE in "${GCP_REGION}-a" "${GCP_REGION}-b" "${GCP_REGION}-c"; do
    if ! gcloud compute instances describe "$servername" --project "$GCP_PROJECT" --zone "$GCP_ZONE" >/dev/null 2>&1; then
      if gcloud compute instances create "$servername" \
        --project "$GCP_PROJECT" \
        --zone "$GCP_ZONE" \
        --enable-nested-virtualization \
        --image-project ubuntu-os-cloud \
        --image-family ubuntu-2404-lts-amd64 \
        --boot-disk-size 500G \
        --boot-disk-type pd-ssd \
        --can-ip-forward \
        --tags http-server,https-server,novnc,openstack-apis \
        --min-cpu-platform "$GCP_MACHINE_MIN_CPU_PLATFORM" \
        --machine-type "$machine_type" \
        --network-interface="private-network-ip=${ip},network=${CLUSTER_NAME}-mynetwork,subnet=${CLUSTER_NAME}-mynetwork" \
        --metadata-from-file user-data="$userdata"; then
        # return function create_vm if the instance have been created successfully.
        return
      fi
    fi
  done
  echo "No free GCP zone could be found to create instance $servername."
  exit 1
}

function get_public_ip {
  local ip
  while ! ip=$(gcloud compute instances describe "${CLUSTER_NAME}-controller" \
    --project "$GCP_PROJECT" --zone "$GCP_ZONE" \
    --format='get(networkInterfaces[0].accessConfigs[0].natIP)'); do
    echo "Waiting for a public IP"
    sleep 5
  done
  echo "$ip"
}

function get_mtu {
  # According to this documentation it's 1460. Hardcoded?
  #   https://cloud.google.com/network-connectivity/docs/vpn/concepts/mtu-considerations
  echo 1460
}

function get_ssh_public_key_file {
  echo /root/.ssh/google_compute_engine.pub
}

function get_ssh_private_key_file {
  echo /root/.ssh/google_compute_engine
}

function cloud_cleanup {
  # Not implemented
  exit 1
}
