package cloudinit

/*
This file is auto-generated DO NOT TOUCH!
*/

const (
	genericCloudConfig = `#cloud-config

## FIXME
package_update: false
package_upgrade: false

write_files:
  - path: /etc/default/bootstrap-kubernetes
    permissions: "0444"
    content: |
      # Environment variables needed for bootstrapping kubernetes
      NAMESPACE={{.Namespace}}
      MACHINE_NAME={{.Name}}
  - path: /etc/modules-load.d/kubernetes.conf
    permissions: "0444"
    content: |
      br_netfilter
  - path: /etc/systemd/system/bootstrap-kubernetes.service
    permissions: "0444"
    encoding: b64
    content: {{.BootstrapService}}
  - path: /usr/local/bin/bootstrap-kubernetes
    permissions: "0555"
    encoding: b64
    content: {{.BootstrapScript}}
  - path: /etc/sysctl.d/10-kubeadm.conf
    paermissions: "0444"
    content: |
      net.bridge.bridge-nf-call-iptables = 1
      net.ipv4.ip_forward = 1

runcmd:
  - [systemctl, daemon-reload]
  - [systemctl, restart, systemd-modules-load.service]
  - [systemctl, restart, systemd-sysctl.service]
  - [systemctl, enable, docker.service]
  - [systemctl, start, --no-block, docker.service]
  - [systemctl, enable, kubelet.service]
  - [systemctl, start, kubelet.service]
  - [systemctl, start, bootstrap-kubernetes.service]

merge_how: "list(append)+dict(recurse_array)+str()"

`
)
