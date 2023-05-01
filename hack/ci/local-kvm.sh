#!/usr/bin/env bash

# Copyright 2023 The Kubernetes Authors.
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
  if [[ ! -f "${devstackdir}/id_rsa" ]]; then
    ssh-keygen -t rsa -f "${devstackdir}/id_rsa" -N ''
    chmod 600 "${devstackdir}/id_rsa"
  fi

  ubuntuImage="${devstackdir}/focal-server-cloudimg-amd64.img"
  ubuntuImageURL="https://storage.googleapis.com/artifacts.k8s-staging-capi-openstack.appspot.com/test/ubuntu/2023-01-14/focal-server-cloudimg-amd64.img"
  ubuntuImageSha256Sum="2aa7fb737962f00da3ac21f542dff7f8d966ee10a4e99de9422d406107e74d92"
  if [[ ! -f "${ubuntuImage}" ]] || [[ ! "$(sha256sum "${ubuntuImage}")" =~ ${ubuntuImageSha256Sum} ]]; then
    wget -O "${ubuntuImage}" "$ubuntuImageURL"
  fi

  KVM_VCPUS_controller=${KVM_VCPUS:-"16"}
  KVM_VCPUS_worker=${KVM_VCPUS:-"8"}
  KVM_MEM_controller=${KVM_MEM:-"16384"}
  KVM_MEM_worker=${KVM_MEM:-"8192"}
}

function init_infrastructure() {
  echo "Creating CAPO devstack private network."
  ADDRESS=$(ipcalc -n "${PRIVATE_NETWORK_CIDR}" | awk '/HostMin/{print $2}')
  NETMASK=$(ipcalc -n "${PRIVATE_NETWORK_CIDR}" | awk '/Netmask/{print $2}')
  HOSTMAX=$(ipcalc -n "${PRIVATE_NETWORK_CIDR}" | awk '/HostMax/{print $2}')
  DHCP_FIRST="${ADDRESS}0"

  # create network only if does not exist
  if ! virsh net-uuid "${CLUSTER_NAME}" &>/dev/null; then
    # from http://blog.zencoffee.org/2016/06/static-mac-generator-kvm/
    CONTROLLER_MAC=$(date +%s -d '10 seconds ago' | md5sum | head -c 6 | sed -e 's/\([0-9A-Fa-f]\{2\}\)/\1:/g' -e 's/\(.*\):$/\1/' | sed -e 's/^/52:54:00:/')
    WORKER_MAC=$(date +%s | md5sum | head -c 6 | sed -e 's/\([0-9A-Fa-f]\{2\}\)/\1:/g' -e 's/\(.*\):$/\1/' | sed -e 's/^/52:54:00:/')
    export CLUSTER_NAME ADDRESS NETMASK HOSTMAX CONTROLLER_IP CONTROLLER_MAC WORKER_IP WORKER_MAC DHCP_FIRST
    envsubst <"${scriptdir}/local-kvm/private-net.xml" | virsh net-create /dev/stdin
  else
    CONTROLLER_MAC=$(virsh net-dumpxml capo-e2e | grep "name='controller'" | sed -r "s/.* mac='([^']+)' .*/\1/")
    WORKER_MAC=$(virsh net-dumpxml capo-e2e | grep "name='worker'" | sed -r "s/.* mac='([^']+)' .*/\1/")
  fi
}

function create_vm {
  local name=$1 && shift
  local ip=$1 && shift
  local userdata=$1 && shift
  local public=$1 && shift # Unused by KVM

  vmName="${CLUSTER_NAME}-${name}"

  if virsh domuuid "${vmName}" &>/dev/null; then
    echo "Instance ${vmName} already edployed"
    return
  fi

  echo "Creating cloud-init iso"
  mkdir -p "${devstackdir}/cloud-init-${name}"
  rm -f "${devstackdir}/cloud-init-${name}/cidata.iso"
  cp "${userdata}" "${devstackdir}/cloud-init-${name}/user-data"
  printf "instance-id: %s\nlocal-hostname: %s\n" "${name}" "${name}" >"${devstackdir}/cloud-init-${name}/meta-data"
  genisoimage -output "${devstackdir}/cloud-init-${name}/cidata.iso" -V cidata -r -J "${devstackdir}/cloud-init-${name}/user-data" "${devstackdir}/cloud-init-${name}/meta-data"

  echo "Creating image for ${name}"
  rm -f "${devstackdir}/data-${name}.img"
  qemu-img create -b "${devstackdir}/focal-server-cloudimg-amd64.img" -f qcow2 -F qcow2 "${devstackdir}/data-${name}.img" 200G

  local vcpus="KVM_VCPUS_${name}"
  vcpus=${!vcpus}
  local memory="KVM_MEM_${name}"
  memory=${!memory}

  local mac="${name^^}_MAC"
  mac=${!mac}

  virt-install --name "${vmName}" \
    --vcpus "${vcpus}" --cpu host \
    --memory "${memory}" \
    --import \
    --disk "path=${devstackdir}/data-${name}.img,format=qcow2" \
    --disk "path=${devstackdir}/cloud-init-${name}/cidata.iso,device=cdrom" \
    --os-variant=ubuntu20.04 \
    --network "bridge=br-${CLUSTER_NAME},model=virtio,mac=${mac}" \
    --graphics none --noautoconsole
}

function get_public_ip {
  echo "${CONTROLLER_IP}"
}

function get_mtu {
  echo 1500
}

function get_ssh_public_key_file {
  echo "${devstackdir}/id_rsa.pub"
}

function get_ssh_private_key_file {
  echo "${devstackdir}/id_rsa"
}

function cloud_cleanup {
  virsh destroy --graceful "${CLUSTER_NAME}-controller" || true
  virsh destroy --graceful "${CLUSTER_NAME}-worker" || true
  virsh undefine "${CLUSTER_NAME}-controller" || true
  virsh undefine "${CLUSTER_NAME}-worker" || true
  kind delete cluster --name "${CLUSTER_NAME}" || true
}
