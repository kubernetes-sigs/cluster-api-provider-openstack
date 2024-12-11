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

# hack script for preparing Openstack to run cluster-api-provider-openstack e2e

set -x -o errexit -o nounset -o pipefail

# Required environment variables:
# OS_CLOUD
# SSH_PUBLIC_KEY_FILE
# SSH_PRIVATE_KEY_FILE
# OPENSTACK_PUBLIC_NETWORK
# OPENSTACK_PUBLIC_IP (optional, will be created on OPENSTACK_PUBLIC_NETWORK if not defined)
# USE_VOLUMES (optional, default true)

function cloud_init {
    OPENSTACK_NETWORK_NAME=${OPENSTACK_NETWORK_NAME:-${CLUSTER_NAME}-network}
    OPENSTACK_SUBNET_NAME=${OPENSTACK_SUBNET_NAME:-${CLUSTER_NAME}-subnet}
    OPENSTACK_SECGROUP_NAME=${OPENSTACK_SECGROUP_NAME:-${CLUSTER_NAME}-secgroup}
    OPENSTACK_ROUTER_NAME=${OPENSTACK_ROUTER_NAME:-${CLUSTER_NAME}-router}
    OPENSTACK_IMAGE_NAME=${OPENSTACK_IMAGE_NAME:-ubuntu-2404-lts}

    OPENSTACK_FLAVOR=${OPENSTACK_FLAVOR:-m1.xlarge}
    OPENSTACK_FLAVOR_controller=${OPENSTACK_FLAVOR_controller:-$OPENSTACK_FLAVOR}
    OPENSTACK_FLAVOR_worker=${OPENSTACK_FLAVOR_worker:-$OPENSTACK_FLAVOR}

    ensure_openstack_client
}

function init_infrastructure() {
    if ! networkid=$(openstack network show "$OPENSTACK_NETWORK_NAME" -f value -c id 2> /dev/null)
    then
        network=$(openstack network create --tag "$CLUSTER_NAME" "$OPENSTACK_NETWORK_NAME" -f json)
        networkid=$(jq -re '.id' <<< "$network")
        mtu=$(jq -re '.mtu' <<< "$network")
    fi

    if ! subnetid=$(openstack subnet show "$OPENSTACK_SUBNET_NAME" -f value -c id 2>/dev/null)
    then
        subnetid=$(openstack subnet create --network "$networkid" --tag "$CLUSTER_NAME" \
            --subnet-range "$PRIVATE_NETWORK_CIDR" \
            "$OPENSTACK_SUBNET_NAME" -f value -c id)
    fi

    if ! secgroupid=$(openstack security group show "$OPENSTACK_SECGROUP_NAME" -f value -c id 2>/dev/null)
    then
        secgroupid=$(openstack security group create --tag "$CLUSTER_NAME" "$OPENSTACK_SECGROUP_NAME" -f value -c id)
        openstack security group rule create --description="${CLUSTER_NAME}-allow-http" \
            --ingress --protocol tcp --dst-port 80 "$secgroupid"
        # As of Victoria, neutron is the only service which isn't multiplexed
        # by apache on port 80
        openstack security group rule create --description="${CLUSTER_NAME}-allow-neutron" \
            --ingress --protocol tcp --dst-port 9696 "$secgroupid"
        openstack security group rule create --description="${CLUSTER_NAME}-allow-icmp" \
            --ingress --protocol icmp "$secgroupid"
        openstack security group rule create --description="${CLUSTER_NAME}-allow-ssh" \
            --ingress --protocol tcp --dst-port 22 "$secgroupid"
        openstack security group rule create --description="${CLUSTER_NAME}-allow-internal" \
            --ingress --remote-ip "$PRIVATE_NETWORK_CIDR" "$secgroupid"
    fi

    if ! routerid=$(openstack router show "$OPENSTACK_ROUTER_NAME" -f value -c id 2>/dev/null)
    then
        routerid=$(openstack router create --tag "$CLUSTER_NAME" "$OPENSTACK_ROUTER_NAME" -f value -c id)
        openstack router set "$routerid" --external-gateway "$OPENSTACK_PUBLIC_NETWORK"
        openstack router add subnet "$routerid" "$subnetid"
    fi

    # If OPENSTACK_PUBLIC_IP is not set, look for an existing unattached tagged floating ip before creating one
    [ -z "${OPENSTACK_PUBLIC_IP:-}" ] && \
        OPENSTACK_PUBLIC_IP=$(openstack floating ip list --tags "${CLUSTER_NAME}" -f value -c "Floating IP Address" | head -n 1)
    [ -z "${OPENSTACK_PUBLIC_IP:-}" ] && \
        OPENSTACK_PUBLIC_IP=$(openstack floating ip create --tag "$CLUSTER_NAME" \
                "$OPENSTACK_PUBLIC_NETWORK" -f value -c floating_ip_address)

    # We don't tag the image with the cluster name as we expect it to be shared
    if ! imageid=$(openstack image show "$OPENSTACK_IMAGE_NAME" -f value -c id 2>/dev/null)
    then
        curl -o /tmp/ubuntu-2204.qcow2 https://cloud-images.ubuntu.com/releases/jammy/release/ubuntu-22.04-server-cloudimg-amd64.img
        imageid=$(openstack image create --disk-format qcow2 --file /tmp/ubuntu-2204.qcow2 "$OPENSTACK_IMAGE_NAME" -f value -c id)
        rm /tmp/ubuntu-2204.qcow2
    fi
}

function create_vm {
    local name=$1 && shift
    local ip=$1 && shift
    local userdata=$1 && shift
    local public=$1 && shift

    local flavor=OPENSTACK_FLAVOR_${name}
    flavor=${!flavor}
    local servername="${CLUSTER_NAME}-${name}"

    # Values which weren't initialised if we skipped init_infrastructure. Use names instead.
    networkid=${networkid:-${OPENSTACK_NETWORK_NAME}}
    secgroupid=${secgroupid:-${OPENSTACK_SECGROUP_NAME}}
    imageid=${imageid:-${OPENSTACK_IMAGE_NAME}}

    local storage_medium_flag="--image=$imageid"

    if [ "${USE_VOLUMES:-true}" == "true" ]; then
      local volumename="${CLUSTER_NAME}-${name}"
      local volumeid
      if ! volumeid=$(openstack volume show "$volumename" -f value -c id 2>/dev/null)
      then
          volumeid=$(openstack volume create -f value -c id --size 200  \
              --bootable --image "$imageid" "$volumename")
          while [ "$(openstack volume show "$volumename" -f value -c status 2>/dev/null)" != "available" ]; do
              echo "Waiting for volume to become available"
              sleep 5
          done
      fi
      storage_medium_flag="--volume=$volumeid"
    fi

    local serverid
    if ! serverid=$(openstack server show "$servername" -f value -c id 2>/dev/null)
    then
        serverid=$(openstack server create -f value -c id \
            --os-compute-api-version 2.52 --tag "$CLUSTER_NAME" \
            --flavor "$flavor" "$storage_medium_flag" \
            --nic net-id="$networkid",v4-fixed-ip="$ip" \
            --security-group "$secgroupid" \
            --user-data "$userdata" \
            --wait "$servername" | xargs echo) # Output suffers from additional newline

        if [ "$public" == "public" ]; then
            openstack server add floating ip "$serverid" "$OPENSTACK_PUBLIC_IP"
        fi
    fi
}

function get_public_ip {
    echo "$OPENSTACK_PUBLIC_IP"
}

function get_mtu {
    # If we just created the network then mtu is already set. If not, fetch it.
    mtu=${mtu:-$(openstack network show "$OPENSTACK_NETWORK_NAME" -f value -c mtu)}
    echo "$mtu"
}

function get_ssh_public_key_file {
    echo "$SSH_PUBLIC_KEY_FILE"
}

function get_ssh_private_key_file {
    # Allow this to be unbound. This is handled in create_devstack.sh
    echo "${SSH_PRIVATE_KEY_FILE:-}"
}

function cloud_cleanup {
    for floating_ip in $(openstack floating ip list --tag "$CLUSTER_NAME" -f value -c ID); do
        openstack floating ip delete "$floating_ip"
    done

    # List server by tags may be broken: https://bugzilla.redhat.com/show_bug.cgi?id=2012910
    for name in controller worker; do
        name="${CLUSTER_NAME}-${name}"
        if ! openstack server delete --wait "$name"; then
            openstack server show "$name" && exit 1
        fi
        if ! openstack volume delete "$name"; then
            openstack volume show "$name" && exit 1
        fi
    done

    for routerid in $(openstack router list --tag "$CLUSTER_NAME" -f value -c ID); do
        for subnetid in $(openstack router show "$routerid" -f json | jq -r '.interfaces_info[] | .subnet_id'); do
            openstack router remove subnet "$routerid" "$subnetid"
        done
        openstack router delete "$routerid"
    done

    for networkid in $(openstack network list --tag "$CLUSTER_NAME" -f value -c ID); do
        for portid in $(openstack port list --network "$networkid" -f value -c id -c status | awk '$2 == "DOWN" {print $1}'); do
            openstack port delete "$portid"
        done
        openstack network delete "$networkid"
    done

    for subnetid in $(openstack subnet list --tag "$CLUSTER_NAME" -f value -c ID); do
        openstack subnet delete "$subnetid"
    done

    for secgroupid in $(openstack security group list --tag "$CLUSTER_NAME" -f value -c ID); do
        openstack security group delete "$secgroupid"
    done

    true
}
