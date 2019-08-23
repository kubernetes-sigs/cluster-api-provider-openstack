#!/usr/bin/env bash
set -e
set -x
(
KUBELET_VERSION={{ .Machine.Spec.Version }}
NAMESPACE={{ .Machine.ObjectMeta.Namespace }}
MACHINE=$NAMESPACE
MACHINE+="/"
MACHINE+={{ .Machine.ObjectMeta.Name }}
ARCH=amd64
swapoff -a
# disable swap in fstab
sed -i.bak -r 's/(.+ swap .+)/#\1/' /etc/fstab

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

if [[ $(getenforce) != 'Disabled' ]]; then
  setenforce 0
fi

yum install -y kubelet-$KUBELET_VERSION kubeadm-$KUBELET_VERSION kubectl-$KUBELET_VERSION --disableexcludes=kubernetes

function install_configure_docker () {
    # prevent docker from auto-starting
    echo "exit 101" > /usr/sbin/policy-rc.d
    chmod +x /usr/sbin/policy-rc.d
    trap "rm /usr/sbin/policy-rc.d" RETURN
    yum install -y docker
    echo 'OPTIONS="--log-driver=journald --signature-verification=false --iptables=false --ip-masq=false"' >> /etc/sysconfig/docker
    systemctl daemon-reload
    systemctl enable docker
    systemctl start docker
}

install_configure_docker

systemctl enable kubelet.service

modprobe br_netfilter
echo '1' > /proc/sys/net/bridge/bridge-nf-call-iptables
echo '1' > /proc/sys/net/ipv4/ip_forward

echo $OPENSTACK_CLOUD_PROVIDER_CONF | base64 -d > /etc/kubernetes/cloud.conf
mkdir /etc/certs
echo $OPENSTACK_CLOUD_CACERT_CONFIG | base64 -d > /etc/certs/cacert


# Setup certificates
mkdir /etc/kubernetes/pki /etc/kubernetes/pki/etcd
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
/usr/bin/sed -i "s#\${OPENSTACK_IPV4_LOCAL}#${OPENSTACK_IPV4_LOCAL}#" /etc/kubernetes/kubeadm_config.yaml

kubeadm init -v 10 --config /etc/kubernetes/kubeadm_config.yaml
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
