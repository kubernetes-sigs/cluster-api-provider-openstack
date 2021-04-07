#cloud-config
hostname: localhost
users:
- name: ubuntu
  lock_passwd: true
write_files:
- content: |
    net.ipv4.ip_forward=1
    net.ipv4.conf.default.rp_filter=0
    net.ipv4.conf.all.rp_filter=0
  path: /etc/sysctl.d/devstack.conf
- content: |
    #!/bin/bash

    set -o errexit -o nounset -o pipefail

    kvm-ok
    sudo modprobe kvm-intel

    # Resize disk
    lsblk
    df -h
    for disk in $(lsblk -d -o NAME | grep -v "loop\|NAME")
    do
      sudo growpart /dev/${disk} 1
    done
    # Resize root disk
    sudo resize2fs $(df -hT | grep /$ | awk '{print $1}')
    lsblk
    df -h

    if [[ ${OPENSTACK_ENABLE_HORIZON} = "false" ]]
    then
      sed -i '/horizon/d' /opt/stack/devstack/local.conf
    fi

    # Stack that stack!
    su - stack -c /opt/stack/devstack/stack.sh

    # Upload the images so we don't have to upload them from Prow
    su - stack -c "source /opt/stack/devstack/openrc admin admin && /opt/stack/devstack/tools/upload_image.sh https://storage.googleapis.com/artifacts.k8s-staging-capi-openstack.appspot.com/test/ubuntu/2021-03-27/ubuntu-2004-kube-v1.18.15.qcow2"
    su - stack -c "source /opt/stack/devstack/openrc admin admin && /opt/stack/devstack/tools/upload_image.sh https://storage.googleapis.com/artifacts.k8s-staging-capi-openstack.appspot.com/test/cirros/2021-03-27/cirros-0.5.1-x86_64-disk.img"

    # Use the network interface of the private ip
    INTERFACE=$(ifconfig | grep -B1 10.0.2.15 | grep -o "^\w*")
    sudo iptables -t nat -I POSTROUTING -o ${INTERFACE} -s 172.24.4.0/24 -j MASQUERADE
    sudo iptables -I FORWARD -s 172.24.4.0/24 -j ACCEPT

  path: /root/devstack.sh
  permissions: '0777'
runcmd:
- sysctl -p /etc/sysctl.d/devstack.conf
- /root/devstack.sh
final_message: "The system is finally up, after $UPTIME seconds"
