#!/usr/bin/env bash
set -e
set -x
(
KUBELET_VERSION={{ .Machine.Spec.Version }}
VERSION=v${KUBELET_VERSION}
NAMESPACE={{ .Machine.ObjectMeta.Namespace }}
MACHINE=$NAMESPACE
MACHINE+="/"
MACHINE+={{ .Machine.ObjectMeta.Name }}
ARCH=amd64
swapoff -a
# disable swap in fstab
sed -i.bak -r 's/(.+ swap .+)/#\1/' /etc/fstab
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
touch /etc/apt/sources.list.d/kubernetes.list
sh -c 'echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list'
apt-get update -y
apt-get install -y \
    prips

# Getting local ip from the metadata of the node.
echo "Getting local ip from metadata"
for i in $(seq 60); do
    echo "trying to get local-ipv4 $i / 60"
    OPENSTACK_IPV4_LOCAL=$(curl --fail -s http://169.254.169.254/latest/meta-data/local-ipv4)
    if [[ $? == 0 ]] && [[ -n "$OPENSTACK_IPV4_LOCAL" ]]; then
        break
    fi
    sleep 1
done

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
KUBECTL=$(getversion kubectl ${KUBELET_VERSION}-)
apt-get install -y \
    kubelet=${KUBELET} \
    kubeadm=${KUBEADM} \
    kubectl=${KUBECTL}

mv /usr/bin/kubeadm.dl /usr/bin/kubeadm
chmod a+rx /usr/bin/kubeadm

echo W0dsb2JhbF0KYXV0aC11cmw9bnVsbAp1c2VybmFtZT0ibnVsbCIKcGFzc3dvcmQ9Im51bGwiCnJlZ2lvbj0ibnVsbCIKdGVuYW50LWlkPSJudWxsIgpkb21haW4tbmFtZT0ibnVsbCIKCg== | base64 -d > /etc/kubernetes/cloud.conf
chmod 600 /etc/kubernetes/cloud.conf
mkdir /etc/certs
echo  | base64 -d > /etc/certs/cacert

systemctl daemon-reload
systemctl restart kubelet.service
systemctl disable ufw
systemctl mask ufw

# Setup certificates
mkdir - /etc/kubernetes/pki /etc/kubernetes/pki/etcd
cat > /etc/kubernetes/pki/ca.crt <<EOF
{{ .CACert }}
EOF

cat > /etc/kubernetes/pki/ca.key <<EOF
{{ .CAKey }}
EOF

cat > /etc/kubernetes/pki/etcd/ca.crt <<EOF
{{ .EtcdCACert }}
EOF

cat > /etc/kubernetes/pki/etcd/ca.key <<EOF
{{ .EtcdCAKey }}
EOF

cat > /etc/kubernetes/pki/front-proxy-ca.crt <<EOF
{{ .FrontProxyCACert }}
EOF

cat > /etc/kubernetes/pki/front-proxy-ca.key <<EOF
{{ .FrontProxyCAKey }}
EOF

cat > /etc/kubernetes/pki/sa.pub <<EOF
{{ .SaCert }}
EOF

cat > /etc/kubernetes/pki/sa.key <<EOF
{{ .SaKey }}
EOF

# Set up kubeadm config file to pass parameters to kubeadm init.
cat > /etc/kubernetes/kubeadm_config.yaml <<EOF
{{ .KubeadmConfig }}
EOF

echo "Replacing OPENSTACK_IPV4_LOCAL in kubeadm_config through ${OPENSTACK_IPV4_LOCAL}"
/bin/sed -i "s#\${OPENSTACK_IPV4_LOCAL}#${OPENSTACK_IPV4_LOCAL}#" /etc/kubernetes/kubeadm_config.yaml

# Create and set bridge-nf-call-iptables to 1 to pass the kubeadm preflight check.
# Workaround was found here:
# http://zeeshanali.com/sysadmin/fixed-sysctl-cannot-stat-procsysnetbridgebridge-nf-call-iptables/
modprobe br_netfilter

kubeadm init -v 10 --config /etc/kubernetes/kubeadm_config.yaml
for tries in $(seq 1 60); do
    kubectl --kubeconfig /etc/kubernetes/kubelet.conf annotate --overwrite node $(hostname) machine=${MACHINE} && break
    sleep 1
done
# By default, use calico for container network plugin, should make this configurable.
kubectl --kubeconfig /etc/kubernetes/admin.conf apply -f https://docs.projectcalico.org/v3.5/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/1.7/calico.yaml
echo done.
) 2>&1 | tee /var/log/startup.log
