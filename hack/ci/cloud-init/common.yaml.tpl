#cloud-config
runcmd:
- sysctl -p /etc/sysctl.d/devstack.conf
- /root/devstack.sh
final_message: "The system is finally up, after $UPTIME seconds"
users:
- name: cloud
  lock_passwd: true
  sudo: ALL=(ALL) NOPASSWD:ALL
  ssh_authorized_keys:
  - ${SSH_PUBLIC_KEY}
# Infrastructure packages required:
#   python3 - required by sshuttle
#   git - required to obtain devstack
#   jq - required by devstack-common.sh
packages:
- python3
- git
- jq
package_upgrade: true
write_files:
- path: /etc/sysctl.d/devstack.conf
  permissions: 0644
  content: |
    net.ipv4.ip_forward=1
    net.ipv4.conf.default.rp_filter=0
    net.ipv4.conf.all.rp_filter=0
- path: /tmp/devstack-common.sh
  permissions: 0644
  content: |
    # ensure nested virtualization
    function ensure_kvm {
      sudo modprobe kvm-intel
      if [ ! -c /dev/kvm ]; then
          echo /dev/kvm is not present
          exit 1
      fi
    }

    function run_devstack {
      su - stack -c "TERM=vt100 /opt/stack/devstack/stack.sh"
    }

    function upload_images {
      # Add environment variables for auth/endpoints
      echo 'source /opt/stack/devstack/openrc admin admin' >> /opt/stack/.bashrc

      # Upload the images so we don't have to upload them from Prow
      su - stack -c "source /opt/stack/devstack/openrc admin admin && /opt/stack/devstack/tools/upload_image.sh https://storage.googleapis.com/artifacts.k8s-staging-capi-openstack.appspot.com/test/ubuntu/2021-03-27/ubuntu-2004-kube-v1.18.15.qcow2"
      su - stack -c "source /opt/stack/devstack/openrc admin admin && /opt/stack/devstack/tools/upload_image.sh https://storage.googleapis.com/artifacts.k8s-staging-capi-openstack.appspot.com/test/cirros/2021-03-27/cirros-0.5.1-x86_64-disk.img"
    }
