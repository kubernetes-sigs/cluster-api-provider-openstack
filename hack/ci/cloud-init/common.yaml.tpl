#cloud-config
runcmd:
- sysctl -p /etc/sysctl.d/devstack.conf
- /root/devstack.sh
bootcmd:
# Stop cloud-final from writing to the serial console. cloud-final.service
# uses StandardOutput=journal+console and every line that devstack prints
# during stack.sh is written to ttyS0, which is a slow serial port on GCE. The
# output is still kept in the journal and in /var/log/cloud-init-output.log.
- |
  mkdir -p /etc/systemd/system/cloud-final.service.d
  cat > /etc/systemd/system/cloud-final.service.d/10-capo-no-console.conf <<'EOF'
  [Service]
  StandardOutput=journal
  StandardError=journal
  EOF
  systemctl daemon-reload
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
write_files:
- path: /etc/sysctl.d/devstack.conf
  permissions: "0644"
  content: |
    net.ipv4.ip_forward=1
    net.ipv4.conf.default.rp_filter=0
    net.ipv4.conf.all.rp_filter=0
- path: /tmp/devstack-common.sh
  permissions: "0644"
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
