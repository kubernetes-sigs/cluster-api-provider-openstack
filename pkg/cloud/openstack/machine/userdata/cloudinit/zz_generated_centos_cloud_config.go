package cloudinit

/*
This file is auto-generated DO NOT TOUCH!
*/

const (
	centosCloudConfig = `#cloud-config

yum_repos:
  # The name of the repository
  kubernetes:
    # Any repository configuration options
    # See: man yum.conf
    #
    # This one is required!
    baseurl: https://packages.cloud.google.com/yum/repos/kubernetes-el7-x86_64
    enabled: true
    failovermethod: priority
    gpgcheck: true
    repo_gpgcheck: true
    gpgkey: https://packages.cloud.google.com/yum/doc/yum-key.gpg https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
    name: Kubernetes

packages:
  - [kubeadm, {{.Version}}]
  - [kubectl, {{.Version}}]
  - [kubelet, {{.Version}}]
  - docker
  - yum-versionlock

write_files:
  - path: /etc/sysconfig/docker
    permissions: "0444"
    content: |
      OPTIONS="--selinux-enabled --log-driver=journald --signature-verification=false --iptables=false --ip-masq=false"
  - path: /etc/sysconfig/kubelet-cluster-api-provider-openstack
    permissions: "0444"
    content: |
      KUBELET_KUBEADM_EXTRA_ARGS=--cgroup-driver=systemd
  - path: /var/lib/bootstrap/hacks
    permissions: "0555"
    content: |
      #!/bin/bash
      hostnamectl set-hostname $(hostname -s)
      setenforce 0
  - path: /etc/selinux/config
    permissions: "0444"
    content: |
      SELINUX=permissive
      SELINUXTYPE=targeted
  - path: /etc/systemd/system/kubelet.service.d/20-add-environment.conf
    permissons: "0444"
    content: |
      [Service]
      EnvironmentFile=-/etc/sysconfig/kubelet-cluster-api-provider-openstack

runcmd:
  - [/var/lib/bootstrap/hacks]
  - [yum, versionlock, add, kubelet-{{.Version}}-*]
  - [systemctl, daemon-reload]

`
)
