#!/usr/bin/env bash

# Copyright 2024 The Kubernetes Authors.
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

# hack script for preparing libvirt to run cluster-api-provider-openstack e2e

set -x -o errexit -o nounset -o pipefail

# Required environment variables:
# SSH_PUBLIC_KEY_FILE
# SSH_PRIVATE_KEY_FILE
# LIBVIRT_NETWORK_NAME

function cloud_init {
    LIBVIRT_NETWORK_NAME=${LIBVIRT_NETWORK_NAME:-${CLUSTER_NAME}-network}
    LIBVIRT_IMAGE_NAME=${LIBVIRT_IMAGE_NAME:-ubuntu-2404-lts}

    LIBVIRT_MEMORY=${LIBVIRT_MEMORY:-8192}
    LIBVIRT_MEMORY_controller=${LIBVIRT_MEMORY_controller:-$LIBVIRT_MEMORY}
    LIBVIRT_MEMORY_worker=${LIBVIRT_MEMORY_worker:-$LIBVIRT_MEMORY}

    LIBVIRT_VCPU=${LIBVIRT_VCPU:-4}
    LIBVIRT_VCPU_controller=${LIBVIRT_VCPU_controller:-$LIBVIRT_VCPU}
    LIBVIRT_VCPU_worker=${LIBVIRT_VCPU_worker:-$LIBVIRT_VCPU}

    LIBVIRT_MAC_controller="00:60:2f:32:81:00"
    LIBVIRT_MAC_worker="00:60:2f:32:81:01"
}

function init_infrastructure() {
    if ! virsh net-info "${LIBVIRT_NETWORK_NAME}" &>/dev/null; then
        virsh net-define <(cat <<EOF
<network>
  <name>${LIBVIRT_NETWORK_NAME}</name>
  <forward mode='nat'>
    <nat>
      <port start='1024' end='65535'/>
    </nat>
  </forward>
  <bridge name="capobr0" stp="on" delay="0"/>
  <ip address="${PRIVATE_NETWORK_CIDR%/*}" netmask="255.255.255.0">
    <dhcp>
      <range start="${PRIVATE_NETWORK_CIDR%.*}.10" end="${PRIVATE_NETWORK_CIDR%.*}.199"/>
      <host mac="${LIBVIRT_MAC_controller}" name='controller' ip="${CONTROLLER_IP}"/>
      <host mac="${LIBVIRT_MAC_worker}" name='worker' ip="${WORKER_IP}"/>
    </dhcp>
  </ip>
</network>
EOF
)
        virsh net-start "${LIBVIRT_NETWORK_NAME}"
        virsh net-autostart "${LIBVIRT_NETWORK_NAME}"
    fi

    if [ ! -f "/tmp/${LIBVIRT_IMAGE_NAME}.qcow2" ]; then
        curl -o "/tmp/${LIBVIRT_IMAGE_NAME}.qcow2" https://cloud-images.ubuntu.com/releases/noble/release/ubuntu-24.04-server-cloudimg-amd64.img
    fi
}

function create_vm {
    local name=$1 && shift
    local ip=$1 && shift
    local userdata=$1 && shift
    local public=$1 && shift

    local memory=LIBVIRT_MEMORY_${name}
    memory=${!memory}
    local vcpu=LIBVIRT_VCPU_${name}
    vcpu=${!vcpu}
    local servername="${CLUSTER_NAME}-${name}"
    local mac=LIBVIRT_MAC_${name}
    mac=${!mac}

    # Values which weren't initialised if we skipped init_infrastructure. Use names instead.
    networkid=${networkid:-${LIBVIRT_NETWORK_NAME}}
    volumeid=${volumeid:-${LIBVIRT_IMAGE_NAME}_${name}.qcow2}

    sudo cp "/tmp/${LIBVIRT_IMAGE_NAME}.qcow2" "/var/lib/libvirt/images/${volumeid}"
    sudo qemu-img resize "/var/lib/libvirt/images/${volumeid}" +200G

    local serverid
    local serverid
    if ! virsh dominfo "${servername}" &>/dev/null; then
        sudo virt-install \
            --name "${servername}" \
            --memory "${memory}" \
            --vcpus "${vcpu}" \
            --import \
            --disk "/var/lib/libvirt/images/${volumeid},format=qcow2,bus=virtio" \
            --network network="${networkid}",mac="${mac}" \
            --os-variant=ubuntu22.04 \
            --graphics none \
            --cloud-init user-data="${userdata}" \
            --noautoconsole
    fi
}

function get_public_ip {
    echo "${CONTROLLER_IP}"
}

function get_mtu {
    # Set MTU statically for libvirt
    echo 1500
}

function get_ssh_public_key_file {
    echo "${SSH_PUBLIC_KEY_FILE}"
}

function get_ssh_private_key_file {
    # Allow this to be unbound. This is handled in create_devstack.sh
    echo "${SSH_PRIVATE_KEY_FILE:-}"
}

function cloud_cleanup {
    for serverid in $(virsh list --all --name | grep -E "${CLUSTER_NAME}-controller|${CLUSTER_NAME}-worker"); do
        virsh destroy "${serverid}"
        virsh undefine "${serverid}" --remove-all-storage
    done

    for networkid in $(virsh net-list --name | grep -E "${CLUSTER_NAME}"); do
        virsh net-destroy "${networkid}"
        virsh net-undefine "${networkid}"
    done

    true
}
