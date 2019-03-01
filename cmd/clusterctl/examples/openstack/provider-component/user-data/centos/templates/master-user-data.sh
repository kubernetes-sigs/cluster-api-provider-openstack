#!/bin/bash
set -e
set -x
(
KUBELET_VERSION={{ .Machine.Spec.Versions.Kubelet }}
VERSION=v${KUBELET_VERSION}
NAMESPACE={{ .Machine.ObjectMeta.Namespace }}
MACHINE=$NAMESPACE
MACHINE+="/"
MACHINE+={{ .Machine.ObjectMeta.Name }}
CONTROL_PLANE_VERSION={{ .Machine.Spec.Versions.ControlPlane }}
CLUSTER_DNS_DOMAIN={{ .Cluster.Spec.ClusterNetwork.ServiceDomain }}
POD_CIDR={{ .PodCIDR }}
SERVICE_CIDR={{ .ServiceCIDR }}
ARCH=amd64

# Getting master ip from the metadata of the node. By default we try the public-ipv4
# If we don't get any, we fall back to local-ipv4 and in the worst case to localhost
MASTER=""
for i in $(seq 60); do
    echo "trying to get public-ipv4 $i / 60"
    MASTER=$(curl --fail -s http://169.254.169.254/2009-04-04/meta-data/public-ipv4)
    if [[ $? == 0 ]] && [[ -n "$MASTER" ]]; then
        break
    fi
    sleep 1
done

if [[ -z "$MASTER" ]]; then
    echo "falling back to local-ipv4"
    for i in $(seq 60); do
        echo "trying to get local-ipv4 $i / 60"
        MASTER=$(curl --fail -s http://169.254.169.254/2009-04-04/meta-data/local-ipv4)
        if [[ $? == 0 ]] && [[ -n "$MASTER" ]]; then
            break
        fi
        sleep 1
    done
fi

if [[ -z "$MASTER" ]]; then
    echo "falling back to localhost"
    MASTER="localhost"
fi
MASTER="${MASTER}:443"

cat <<EOF > /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://packages.cloud.google.com/yum/repos/kubernetes-el7-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
exclude=kube*
EOF

setenforce 0
yum install -y kubelet-$CONTROL_PLANE_VERSION kubeadm-$CONTROL_PLANE_VERSION kubectl-$CONTROL_PLANE_VERSION --disableexcludes=kubernetes

function install_configure_docker () {
    # prevent docker from auto-starting
    echo "exit 101" > /usr/sbin/policy-rc.d
    chmod +x /usr/sbin/policy-rc.d
    trap "rm /usr/sbin/policy-rc.d" RETURN
    yum install -y docker
    echo 'OPTIONS="--selinux-enabled --log-driver=journald --signature-verification=false --iptables=false --ip-masq=false"' >> /etc/sysconfig/docker
    systemctl daemon-reload
    systemctl enable docker
    systemctl start docker
}

install_configure_docker

cat <<EOF > /etc/default/kubelet
KUBELET_KUBEADM_EXTRA_ARGS=--cgroup-driver=systemd
EOF

systemctl enable kubelet.service

modprobe br_netfilter
echo '1' > /proc/sys/net/bridge/bridge-nf-call-iptables
echo '1' > /proc/sys/net/ipv4/ip_forward

echo $OPENSTACK_CLOUD_PROVIDER_CONF | base64 -d > /etc/kubernetes/cloud.conf

# Set up kubeadm config file to pass parameters to kubeadm init.
cat > /etc/kubernetes/kubeadm_config.yaml <<EOF
apiVersion: kubeadm.k8s.io/v1alpha3
kind: InitConfiguration
bootstrapTokens:
- token: ${TOKEN}
apiEndpoint:
  bindPort: 443
nodeRegistration:
  name: $(hostname -s)
  kubeletExtraArgs:
    cloud-provider: "openstack"
    cloud-config: "/etc/kubernetes/cloud.conf"
---
apiVersion: kubeadm.k8s.io/v1alpha3
kind: ClusterConfiguration
kubernetesVersion: v${CONTROL_PLANE_VERSION}
networking:
  serviceSubnet: ${SERVICE_CIDR}
clusterName: kubernetes
apiServerExtraArgs:
  cloud-provider: "openstack"
  cloud-config: "/etc/kubernetes/cloud.conf"
apiServerExtraVolumes:
- name: cloud
  hostPath: "/etc/kubernetes/cloud.conf"
  mountPath: "/etc/kubernetes/cloud.conf"
controlPlaneEndpoint: ${MASTER}
controllerManagerExtraArgs:
  cluster-cidr: ${POD_CIDR}
  service-cluster-ip-range: ${SERVICE_CIDR}
  allocate-node-cidrs: "true"
  cloud-provider: "openstack"
  cloud-config: "/etc/kubernetes/cloud.conf"
controllerManagerExtraVolumes:
- name: cloud
  hostPath: "/etc/kubernetes/cloud.conf"
  mountPath: "/etc/kubernetes/cloud.conf"
EOF

kubeadm init --config /etc/kubernetes/kubeadm_config.yaml
for tries in $(seq 1 60); do
    kubectl --kubeconfig /etc/kubernetes/kubelet.conf annotate --overwrite node $(hostname -s) machine=${MACHINE} && break
    sleep 1
done

# By default, use calico for container network plugin, should make this configurable.
kubectl --kubeconfig /etc/kubernetes/admin.conf apply -f https://docs.projectcalico.org/v3.5/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/1.7/calico.yaml

mkdir -p /root/.kube
cp -i /etc/kubernetes/admin.conf /root/.kube/config
chown $(id -u):$(id -g) /root/.kube/config

echo done.
) 2>&1 | tee /var/log/startup.log

