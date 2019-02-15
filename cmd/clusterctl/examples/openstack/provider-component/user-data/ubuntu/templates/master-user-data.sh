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
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
touch /etc/apt/sources.list.d/kubernetes.list
sh -c 'echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list'
apt-get update -y
apt-get install -y \
    prips

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

function install_configure_docker () {
    # prevent docker from auto-starting
    echo "exit 101" > /usr/sbin/policy-rc.d
    chmod +x /usr/sbin/policy-rc.d
    trap "rm /usr/sbin/policy-rc.d" RETURN
    apt-get install -y docker.io
    echo 'DOCKER_OPTS="--iptables=false --ip-masq=false"' > /etc/default/docker

    # Reset iptables config
    mkdir -p /etc/systemd/system/docker.service.d
    cat > /etc/systemd/system/docker.service.d/10-iptables.conf <<EOF
[Service]
EnvironmentFile=/etc/default/docker
ExecStart=
ExecStart=/usr/bin/dockerd -H fd:// \$DOCKER_OPTS
EOF

    systemctl daemon-reload
    systemctl enable docker
    systemctl start docker
}
install_configure_docker

curl -sSL https://dl.k8s.io/release/${VERSION}/bin/linux/${ARCH}/kubeadm > /usr/bin/kubeadm.dl
chmod a+rx /usr/bin/kubeadm.dl
# kubeadm uses 10th IP as DNS server
CLUSTER_DNS_SERVER=$(prips ${SERVICE_CIDR} | head -n 11 | tail -n 1)
# Our Debian packages have versions like "1.8.0-00" or "1.8.0-01". Do a prefix
# search based on our SemVer to find the right (newest) package version.
function getversion() {
    name=$1
    prefix=$2
    version=$(apt-cache madison $name | awk '{ print $3 }' | grep ^$prefix | head -n1)
    if [[ -z "$version" ]]; then
        echo Can\'t find package $name with prefix $prefix
        exit 1
    fi
    echo $version
}
KUBELET=$(getversion kubelet ${KUBELET_VERSION}-)
KUBEADM=$(getversion kubeadm ${KUBELET_VERSION}-)
apt-get install -y \
    kubelet=${KUBELET} \
    kubeadm=${KUBEADM}
mv /usr/bin/kubeadm.dl /usr/bin/kubeadm
chmod a+rx /usr/bin/kubeadm

cat > /etc/systemd/system/kubelet.service.d/20-kubenet.conf <<EOF
[Service]
Environment="KUBELET_DNS_ARGS=--cluster-dns=${CLUSTER_DNS_SERVER} --cluster-domain=${CLUSTER_DNS_DOMAIN}"
EOF

echo $OPENSTACK_CLOUD_PROVIDER_CONF | base64 -d > /etc/kubernetes/cloud.conf

systemctl daemon-reload
systemctl restart kubelet.service
systemctl disable ufw
systemctl mask ufw

# Set up kubeadm config file to pass parameters to kubeadm init.
# We're using 443 until this bug is fixed
# https://github.com/kubernetes-sigs/cluster-api-provider-openstack/issues/64
cat > /etc/kubernetes/kubeadm_config.yaml <<EOF
apiVersion: kubeadm.k8s.io/v1alpha3
kind: InitConfiguration
bootstrapTokens:
- token: ${TOKEN}
apiEndpoint:
  bindPort: 443
nodeRegistration:
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
controlPlaneEndpoint: ${MASTER}
apiServerExtraArgs:
  cloud-provider: "openstack"
  cloud-config: "/etc/kubernetes/cloud.conf"
apiServerExtraVolumes:
- name: cloud
  hostPath: "/etc/kubernetes/cloud.conf"
  mountPath: "/etc/kubernetes/cloud.conf"
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

# Create and set bridge-nf-call-iptables to 1 to pass the kubeadm preflight check.
# Workaround was found here:
# http://zeeshanali.com/sysadmin/fixed-sysctl-cannot-stat-procsysnetbridgebridge-nf-call-iptables/
modprobe br_netfilter

kubeadm init --config /etc/kubernetes/kubeadm_config.yaml
for tries in $(seq 1 60); do
    kubectl --kubeconfig /etc/kubernetes/kubelet.conf annotate --overwrite node $(hostname) machine=${MACHINE} && break
    sleep 1
done
# By default, use calico for container network plugin, should make this configurable.
kubectl --kubeconfig /etc/kubernetes/admin.conf apply -f https://docs.projectcalico.org/v3.1/getting-started/kubernetes/installation/hosted/rbac-kdd.yaml
kubectl --kubeconfig /etc/kubernetes/admin.conf apply -f https://docs.projectcalico.org/v3.1/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/1.7/calico.yaml
echo done.
) 2>&1 | tee /var/log/startup.log

