#!/bin/bash
set -e
set -x
(
KUBELET_VERSION={{ .Machine.Spec.Versions.Kubelet }}
TOKEN={{ .Token }}
MASTER={{ call .GetMasterEndpoint }}
NAMESPACE={{ .Machine.ObjectMeta.Namespace }}
MACHINE=$NAMESPACE
MACHINE+="/"
MACHINE+={{ .Machine.ObjectMeta.Name }}
CLUSTER_DNS_DOMAIN={{ .Cluster.Spec.ClusterNetwork.ServiceDomain }}
POD_CIDR={{ .PodCIDR }}
SERVICE_CIDR={{ .ServiceCIDR }}

apt-get update
apt-get install -y apt-transport-https prips
apt-key adv --keyserver hkp://keyserver.ubuntu.com --recv-keys F76221572C52609D
cat <<EOF > /etc/apt/sources.list.d/k8s.list
deb [arch=amd64] https://apt.dockerproject.org/repo ubuntu-xenial main
EOF
apt-get update

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

curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
cat <<EOF > /etc/apt/sources.list.d/kubernetes.list
deb http://apt.kubernetes.io/ kubernetes-xenial main
EOF
apt-get update

# Needed for the node and kubeadm preflights
modprobe ip_vs_sh ip_vs ip_vs_rr ip_vs_wrr

mkdir -p /etc/kubernetes/
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
apt-get install -y kubelet=${KUBELET} kubeadm=${KUBEADM} kubectl=${KUBECTL}
# kubeadm uses 10th IP as DNS server
CLUSTER_DNS_SERVER=$(prips ${SERVICE_CIDR} | head -n 11 | tail -n 1)

# Write the cloud.conf so that the kubelet can use it.
echo $OPENSTACK_CLOUD_PROVIDER_CONF | base64 -d > /etc/kubernetes/cloud.conf

# Set up kubeadm config file to pass to kubeadm join.
cat > /etc/kubernetes/kubeadm_config.yaml <<EOF
apiVersion: kubeadm.k8s.io/v1alpha3
kind: JoinConfiguration
nodeRegistration:
  kubeletExtraArgs:
    cloud-provider: "openstack"
    cloud-config: "/etc/kubernetes/cloud.conf"
token: ${TOKEN}
discoveryTokenAPIServers:
  - ${MASTER}
discoveryTokenUnsafeSkipCAVerification: true
EOF

# Override network args to use kubenet instead of cni, override Kubelet DNS args and
# add cloud provider args.
cat > /etc/systemd/system/kubelet.service.d/20-kubenet.conf <<EOF
[Service]
Environment="KUBELET_DNS_ARGS=--cluster-dns=${CLUSTER_DNS_SERVER} --cluster-domain=${CLUSTER_DNS_DOMAIN}"
EOF
systemctl daemon-reload
systemctl restart kubelet.service
systemctl disable ufw 
systemctl mask ufw

kubeadm join --ignore-preflight-errors=all --config /etc/kubernetes/kubeadm_config.yaml
for tries in $(seq 1 60); do
	kubectl --kubeconfig /etc/kubernetes/kubelet.conf annotate --overwrite node $(hostname) machine=${MACHINE} && break
	sleep 1
done
echo done.
) 2>&1 | tee /var/log/startup.log

