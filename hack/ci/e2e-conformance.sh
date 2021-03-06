#!/usr/bin/env bash

# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# hack script for running a cluster-api-provider-openstack e2e

set -o errexit -o nounset -o pipefail

OPENSTACK_CLOUD=${OPENSTACK_CLOUD:-"capi-quickstart"}
OPENSTACK_CLOUD_YAML_FILE=${OPENSTACK_CLOUD_YAML_FILE:-"/tmp/clouds.yaml"}
IMAGE_URL="https://github.com/kubernetes-sigs/cluster-api-provider-openstack/releases/download/v0.3.0/ubuntu-1910-kube-v1.17.3.qcow2"
CIRROS_URL="http://download.cirros-cloud.net/0.5.1/cirros-0.5.1-x86_64-disk.img"
OPENSTACK_IMAGE_NAME=${OPENSTACK_IMAGE_NAME:-"ubuntu-1910-kube-v1.17.3"}
OPENSTACK_BASTION_IMAGE_NAME=${OPENSTACK_BASTION_IMAGE_NAME:-"cirros"}
OPENSTACK_FAILURE_DOMAIN=${OPENSTACK_FAILURE_DOMAIN:-"nova"}
OPENSTACK_DNS_NAMESERVERS=${OPENSTACK_DNS_NAMESERVERS:-"192.168.200.1"}
OPENSTACK_NODE_MACHINE_FLAVOR=${OPENSTACK_NODE_MACHINE_FLAVOR:-"m1.small"}
WORKER_MACHINE_COUNT=${WORKER_MACHINE_COUNT:-"3"}
OPENSTACK_CONTROL_PLANE_MACHINE_FLAVOR=${OPENSTACK_CONTROL_PLANE_MACHINE_FLAVOR:-"m1.medium"}
CONTROL_PLANE_MACHINE_COUNT=${CONTROL_PLANE_MACHINE_COUNT:-"1"}
OPENSTACK_BASTION_MACHINE_FLAVOR=${OPENSTACK_BASTION_MACHINE_FLAVOR:-"m1.tiny"}
OPENSTACK_CLUSTER_TEMPLATE=${OPENSTACK_CLUSTER_TEMPLATE:-"./templates/cluster-template-without-lb.yaml"}
CLUSTER_NAME=${CLUSTER_NAME:-"capi-quickstart"}
OPENSTACK_SSH_KEY_NAME=${OPENSTACK_SSH_KEY_NAME:-"${CLUSTER_NAME}-key"}
KUBERNETES_VERSION=${KUBERNETES_VERSION:-"v1.19.7"}
USE_CI_ARTIFACTS=${USE_CI_ARTIFACTS:-"true"}
IMAGE_REPOSITORY=${IMAGE_REPOSITORY:-"k8s.gcr.io"}

ARTIFACTS="${ARTIFACTS:-${PWD}/_artifacts}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"

LOGS_KIND_DUMPED=false
LOGS_CAPO_DUMPED=false

dump_kind_logs() {
  set -x

  if [[ "${LOGS_KIND_DUMPED}" == "true" ]];
  then
    echo "kind logs already dumped"
    return 0
  fi
  LOGS_KIND_DUMPED=true

  echo "Dump logs"
  mkdir -p "${ARTIFACTS}/logs"

  echo "=== versions ==="
  echo "kind : $(kind version)" || true
  echo "bootstrap cluster:"
  kubectl version || true
  echo ""

  # dump all the info from the CAPI related CRDs
  kubectl get clusters,openstackclusters,machines,openstackmachines,kubeadmconfigs,machinedeployments,openstackmachinetemplates,kubeadmconfigtemplates,machinesets --all-namespaces -o yaml > "${ARTIFACTS}/logs/kind-capo.txt" || true

  # dump cluster info for kind
  kubectl cluster-info dump > "${ARTIFACTS}/logs/kind-cluster.txt" || true
  kubectl get secrets -o yaml -A > "${ARTIFACTS}/logs/kind-cluster-secrets.txt" || true

  # dump images info
  echo "images in docker" >> "${ARTIFACTS}/logs/images.txt"
  docker images >> "${ARTIFACTS}/logs/images.txt"
  echo "images from bootstrap using containerd CLI" >> "${ARTIFACTS}/logs/images.txt"
  docker exec clusterapi-control-plane ctr -n k8s.io images list >> "${ARTIFACTS}/logs/images.txt" || true
  echo "images in bootstrap cluster using kubectl CLI" >> "${ARTIFACTS}/logs/images.txt"
  (kubectl get pods --all-namespaces -o json \
   | jq --raw-output '.items[].spec.containers[].image' | sort)  >> "${ARTIFACTS}/logs/images.txt" || true
  sudo journalctl --output=short-precise -k -b all > "${ARTIFACTS}/logs/kern.txt" || true
  sudo journalctl --output=short-precise > "${ARTIFACTS}/logs/systemd.txt" || true
  sudo last -x > "${ARTIFACTS}/logs/last-x.txt" || true

  # export all logs from kind
  kind "export" logs --name="clusterapi" "${ARTIFACTS}/logs" || true
  set +x
}

dump_capo_logs() {
  set -x

  if [[ "${LOGS_CAPO_DUMPED}" == "true" ]];
  then
    echo "capo logs already dumped"
    return 0
  fi
  LOGS_CAPO_DUMPED=true

  echo "Dump logs"
  mkdir -p "${ARTIFACTS}/logs"

  echo "=== versions ==="
  echo "capo cluster:"
  kubectl --kubeconfig=${PWD}/kubeconfig version || true
  echo ""

  # dump images info
  echo "images in deployed cluster using kubectl CLI" >> "${ARTIFACTS}/logs/images.txt"
  (kubectl --kubeconfig="${PWD}"/kubeconfig get pods --all-namespaces -o json \
   | jq --raw-output '.items[].spec.containers[].image' | sort)  >> "${ARTIFACTS}/logs/images.txt" || true

  # dump OpenStack info
  echo "" > "${ARTIFACTS}/logs/openstack-cluster.txt"
  echo "=== OpenStack compute images list ===" >> "${ARTIFACTS}/logs/openstack-cluster.txt" || true
  openstack image list >> "${ARTIFACTS}/logs/openstack-cluster.txt" || true
  echo "=== OpenStack compute instances list ===" >> "${ARTIFACTS}/logs/openstack-cluster.txt" || true
  openstack server list >> "${ARTIFACTS}/logs/openstack-cluster.txt" || true
  echo "=== OpenStack compute instances show ===" >> "${ARTIFACTS}/logs/openstack-cluster.txt" || true
  openstack server list -f value -c Name | xargs -I% openstack server show % >> "${ARTIFACTS}/logs/openstack-cluster.txt" || true
  echo "=== cluster-info dump ===" >> "${ARTIFACTS}/logs/openstack-cluster.txt" || true
  kubectl --kubeconfig=${PWD}/kubeconfig cluster-info dump >> "${ARTIFACTS}/logs/openstack-cluster.txt" || true
  kubectl --kubeconfig=${PWD}/kubeconfig get secrets -o yaml -A > "${ARTIFACTS}/logs/openstack-cluster-secrets.txt" || true

  jump_node_name=$(openstack server list -f value -c Name | grep ${CLUSTER_NAME}-bastion | head -n 1)
  jump_node=$(openstack server show ${jump_node_name} -f value -c addresses | awk '{print $2}')
  for node in $(openstack server list -f value -c Name | grep -v bastion)
  do
    echo "collecting logs from ${node} using jump host "
    dir="${ARTIFACTS}/logs/${node}"
    mkdir -p ${dir}

    openstack console log show "${node}" > "${dir}/console.log" || true

    ssh_key_pem="/tmp/${OPENSTACK_SSH_KEY_NAME}.pem"
    PROXY_COMMAND="ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=30 -x -W %h:22 -i ${ssh_key_pem} cirros@${jump_node}"
    node=$(openstack port show ${node}  -f json -c fixed_ips | jq '.fixed_ips[0].ip_address' -r)

    ssh-to-node "${node}" "${jump_node}" "sudo chmod -R a+r /var/log" || true
    scp -r -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=30 -o ProxyCommand="${PROXY_COMMAND}" -i ${ssh_key_pem} \
      "ubuntu@${node}:/var/log/cloud-init.log" "ubuntu@${node}:/var/log/cloud-init-output.log" \
      "ubuntu@${node}:/var/log/pods" "ubuntu@${node}:/var/log/containers" \
      "${dir}" || true

    ssh-to-node "${node}" "${jump_node}" "sudo journalctl --output=short-precise -k" > "${dir}/kern.log" || true
    ssh-to-node "${node}" "${jump_node}" "sudo journalctl --output=short-precise" > "${dir}/systemd.log" || true
    ssh-to-node "${node}" "${jump_node}" "sudo crictl version && sudo crictl info" > "${dir}/containerd.txt" || true
    ssh-to-node "${node}" "${jump_node}" "sudo journalctl --no-pager -u cloud-final" > "${dir}/cloud-final.log" || true
    ssh-to-node "${node}" "${jump_node}" "sudo journalctl --no-pager -u kubelet.service" > "${dir}/kubelet.log" || true
    ssh-to-node "${node}" "${jump_node}" "sudo journalctl --no-pager -u containerd.service" > "${dir}/containerd.log" || true
    ssh-to-node "${node}" "${jump_node}" "sudo top -b -n 1" > "${dir}/top.txt" || true
    ssh-to-node "${node}" "${jump_node}" "sudo crictl ps" > "${dir}/crictl-ps.log" || true
    ssh-to-node "${node}" "${jump_node}" "sudo crictl pods" > "${dir}/crictl-pods.log" || true
  done
  set +x
}

function dump_logs() {
  dump_kind_logs
  dump_capo_logs
}

# SSH to a node by name ($1) via jump server ($2) and run a command ($3).
function ssh-to-node() {
  local node="$1"
  local jump="$2"
  local cmd="$3"

  ssh_key_pem="/tmp/${OPENSTACK_SSH_KEY_NAME}.pem"
  ssh_params="-o LogLevel=quiet -o ConnectTimeout=30 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no"
  scp $ssh_params -i $ssh_key_pem $ssh_key_pem "cirros@${jump}:$ssh_key_pem"
  ssh $ssh_params -i $ssh_key_pem \
    -o "ProxyCommand ssh $ssh_params -W %h:%p -i $ssh_key_pem cirros@${jump}" \
    ubuntu@"${node}" "${cmd}"
}

create_key_pair() {
  echo "Create key pair"

  if [[ -f /tmp/${OPENSTACK_SSH_KEY_NAME}.pem ]] && [[ -f /tmp/${OPENSTACK_SSH_KEY_NAME}.pem.pub ]]
  then
    echo "Skipping, key pair already exists"
    return 0
  fi

  ssh-keygen -t rsa -f "/tmp/${OPENSTACK_SSH_KEY_NAME}.pem"  -N ""
  chmod 0400 "/tmp/${OPENSTACK_SSH_KEY_NAME}.pem"

  openstack keypair create --public-key "/tmp/${OPENSTACK_SSH_KEY_NAME}.pem.pub" ${OPENSTACK_SSH_KEY_NAME}
}

upload_image() {
  # $1: image name
  # $2: image url

  echo "Upload image"

  # Remove old image if we don't want to reuse it
  if [[ "${REUSE_OLD_IMAGES:-true}" == "false" ]]; then
    image_id=$(openstack image list --name=$1 -f value -c ID)
    if [[ ! -z "$image_id" ]]; then
        echo "Deleting old image $1  with id: ${image_id}"
        openstack image delete ${image_id}
    fi
  fi

  image=$(openstack image list --name=$1 -f value -c Name)
  if [[ ! -z "$image" ]]; then
    echo "Image $1 already exists"
    return
  fi

  tmpfile=$(mktemp)
  echo "Download image $1 from $2"
  wget -q -c $2 -O ${tmpfile}

  echo "Uploading image $1"
  openstack image create --disk-format qcow2 --private --container-format bare --file "${tmpfile}" $1
}

install_prereqs() {
  # Install Docker
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
  sudo apt-key fingerprint 0EBFCD88
  sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
  sudo apt-get update
  sudo apt-get install -y docker-ce docker-ce-cli containerd.io jq
  # docker socket already works because OpenLab runs via root
    
    # Install yq
	GO111MODULE=on go get github.com/mikefarah/yq/v2
	go get -u github.com/go-bindata/go-bindata/...

  source "${REPO_ROOT}/hack/ensure-kubectl.sh"
  source "${REPO_ROOT}/hack/ensure-kind.sh"
}

# build kubernetes / node image, e2e binaries
build() {
  if [[ ! -d "$(go env GOPATH)/src/k8s.io/kubernetes" ]]; then
    mkdir -p $(go env GOPATH)/src/k8s.io
    cd $(go env GOPATH)/src/k8s.io

    git clone https://github.com/kubernetes/kubernetes.git
    cd kubernetes
    git checkout -b ${KUBERNETES_VERSION} "refs/tags/${KUBERNETES_VERSION}"
  fi

  pushd "$(go env GOPATH)/src/k8s.io/kubernetes"

  # re-create _output/bin folder
  rm -rf "${PWD}/_output/bin"
  mkdir -p "${PWD}/_output/bin/"

  go build -o ./_output/bin/kubectl ./cmd/kubectl

  ./hack/generate-bindata.sh
  go test -o ./_output/bin/e2e.test -c ./test/e2e/

  go build -o ./_output/bin/ginkgo ./vendor/github.com/onsi/ginkgo/ginkgo

  PATH="$(go env GOPATH)/src/k8s.io/kubernetes/_output/bin:${PATH}"
  export PATH
  popd

  # attempt to release some memory after building
  sync || true
  sudo sh -c "echo 1 > /proc/sys/vm/drop_caches" || true

  cd ${REPO_ROOT}
  echo "Build Docker Images"
  make modules hack/tools/bin/clusterctl hack/tools/bin/kustomize hack/tools/bin/envsubst docker-build
}

# CAPI/CAPO versions
# See: https://console.cloud.google.com/storage/browser/artifacts.k8s-staging-cluster-api.appspot.com/components?project=k8s-staging-cluster-api
E2E_CAPI_DOWNLOAD_VERSION=nightly_master_20210305
E2E_CAPI_VERSION=0.4.0
E2E_CAPO_VERSION=0.4.0

# up a cluster with kind
create_cluster() {
  # actually create the cluster
  KIND_IS_UP=true

  # exports the b64 env vars used below
  source ${REPO_ROOT}/templates/env.rc ${OPENSTACK_CLOUD_YAML_FILE} ${CLUSTER_NAME}

  cd ${REPO_ROOT}

	# Create local repository to use local OpenStack provider and nightly CAPI
	mkdir -p ./out
  cat ./hack/ci/e2e-conformance/clusterctl.yaml.tpl | \
	  sed "s|\${PWD}|${PWD}|" | \
	  sed "s|\${E2E_CAPO_VERSION}|${E2E_CAPO_VERSION}|" | \
	  sed "s|\${E2E_CAPI_VERSION}|${E2E_CAPI_VERSION}|" \
	   > ./out/clusterctl.yaml

	mkdir -p ./out/infrastructure-openstack/${E2E_CAPO_VERSION}
	MANIFEST_IMG=gcr.io/k8s-staging-capi-openstack/capi-openstack-controller-amd64 MANIFEST_TAG=dev make set-manifest-image
	./hack/tools/bin/kustomize build config/default > ./out/infrastructure-openstack/${E2E_CAPO_VERSION}/infrastructure-components.yaml
  cp ./hack/ci/e2e-conformance/metadata.yaml ./out/infrastructure-openstack/${E2E_CAPO_VERSION}/metadata.yaml

	mkdir -p ./out/cluster-api/v${E2E_CAPI_VERSION}
	wget https://storage.googleapis.com/artifacts.k8s-staging-cluster-api.appspot.com/components/${E2E_CAPI_DOWNLOAD_VERSION}/core-components.yaml -O ./out/cluster-api/v${E2E_CAPI_VERSION}/core-components.yaml
	cp ./hack/ci/e2e-conformance/metadata.yaml ./out/cluster-api/v${E2E_CAPI_VERSION}/metadata.yaml

	mkdir -p ./out/bootstrap-kubeadm/v${E2E_CAPI_VERSION}
	wget https://storage.googleapis.com/artifacts.k8s-staging-cluster-api.appspot.com/components/${E2E_CAPI_DOWNLOAD_VERSION}/bootstrap-components.yaml -O ./out/bootstrap-kubeadm/v${E2E_CAPI_VERSION}/bootstrap-components.yaml
	cp ./hack/ci/e2e-conformance/metadata.yaml ./out/bootstrap-kubeadm/v${E2E_CAPI_VERSION}/metadata.yaml

	mkdir -p ./out/control-plane-kubeadm/v${E2E_CAPI_VERSION}
	wget https://storage.googleapis.com/artifacts.k8s-staging-cluster-api.appspot.com/components/${E2E_CAPI_DOWNLOAD_VERSION}/control-plane-components.yaml -O ./out/control-plane-kubeadm/v${E2E_CAPI_VERSION}/control-plane-components.yaml
	cp ./hack/ci/e2e-conformance/metadata.yaml ./out/control-plane-kubeadm/v${E2E_CAPI_VERSION}/metadata.yaml

  # Generate cluster.yaml
	OPENSTACK_FAILURE_DOMAIN=${OPENSTACK_FAILURE_DOMAIN} \
	OPENSTACK_CLOUD=${OPENSTACK_CLOUD} \
	OPENSTACK_CLOUD_CACERT_B64=${OPENSTACK_CLOUD_CACERT_B64} \
	OPENSTACK_CLOUD_PROVIDER_CONF_B64=${OPENSTACK_CLOUD_PROVIDER_CONF_B64} \
	OPENSTACK_CLOUD_YAML_B64=${OPENSTACK_CLOUD_YAML_B64} \
	OPENSTACK_DNS_NAMESERVERS=${OPENSTACK_DNS_NAMESERVERS} \
	OPENSTACK_IMAGE_NAME=${OPENSTACK_IMAGE_NAME} \
	OPENSTACK_SSH_KEY_NAME=${OPENSTACK_SSH_KEY_NAME} \
	OPENSTACK_NODE_MACHINE_FLAVOR=${OPENSTACK_NODE_MACHINE_FLAVOR} \
	OPENSTACK_CONTROL_PLANE_MACHINE_FLAVOR=${OPENSTACK_CONTROL_PLANE_MACHINE_FLAVOR} \
	  ./hack/tools/bin/clusterctl config cluster "${CLUSTER_NAME}" \
	    --from="${OPENSTACK_CLUSTER_TEMPLATE}" \
	    --kubernetes-version "${KUBERNETES_VERSION}" \
	    --target-namespace "${CLUSTER_NAME}" \
	    --control-plane-machine-count="${CONTROL_PLANE_MACHINE_COUNT}" \
	    --worker-machine-count="${WORKER_MACHINE_COUNT}" > ./hack/ci/e2e-conformance/cluster.yaml

  # Patch cluster.yaml
	cat ./hack/ci/e2e-conformance/e2e-conformance_patch.yaml.tpl | \
	  sed "s|\${OPENSTACK_CLOUD_PROVIDER_CONF_B64}|${OPENSTACK_CLOUD_PROVIDER_CONF_B64}|" | \
	  sed "s|\${OPENSTACK_CLOUD_CACERT_B64}|${OPENSTACK_CLOUD_CACERT_B64}|" | \
	  sed "s|\${USE_CI_ARTIFACTS}|${USE_CI_ARTIFACTS}|" | \
	  sed "s|\${IMAGE_REPOSITORY}|${IMAGE_REPOSITORY}|" | \
	  sed "s|\${KUBERNETES_VERSION}|${KUBERNETES_VERSION}|" | \
	  sed "s|\${CLUSTER_NAME}|${CLUSTER_NAME}|" | \
	  sed "s|\${OPENSTACK_BASTION_MACHINE_FLAVOR}|${OPENSTACK_BASTION_MACHINE_FLAVOR}|" | \
	  sed "s|\${OPENSTACK_BASTION_IMAGE_NAME}|${OPENSTACK_BASTION_IMAGE_NAME}|" | \
	  sed "s|\${OPENSTACK_SSH_KEY_NAME}|${OPENSTACK_SSH_KEY_NAME}|" \
	   > ./hack/ci/e2e-conformance/e2e-conformance_patch.yaml

	./hack/tools/bin/kustomize build --reorder=none hack/ci/e2e-conformance  > ./out/cluster.yaml

	if ! kind get clusters | grep -q clusterapi
	then
		kind create cluster --name=clusterapi
	fi

  echo "loading capi image into kind cluster ..."
  kind --name=clusterapi load docker-image "gcr.io/k8s-staging-capi-openstack/capi-openstack-controller-amd64:dev"

	# Delete already deployed provider
	set -x
	./hack/tools/bin/clusterctl delete --all
	./hack/tools/bin/clusterctl delete --infrastructure openstack --include-namespace --namespace capo-system || true
	kubectl wait --for=delete --timeout=5m ns/capo-system || true

	# Deploy provider
	./hack/tools/bin/clusterctl init --config ./out/clusterctl.yaml --infrastructure openstack --core cluster-api:v${E2E_CAPI_VERSION} --bootstrap kubeadm:v${E2E_CAPI_VERSION} --control-plane kubeadm:v${E2E_CAPI_VERSION}

	# Wait for CAPI pods
	kubectl wait --for=condition=Ready --timeout=5m -n capi-system pod --all
	kubectl wait --for=condition=Ready --timeout=5m -n capi-kubeadm-bootstrap-system pod --all
	kubectl wait --for=condition=Ready --timeout=5m -n capi-kubeadm-control-plane-system pod --all
	kubectl wait --for=condition=Ready --timeout=5m -n capo-system pod --all

	# Wait for CAPO CRDs
	kubectl wait --for condition=established --timeout=60s crds/openstackmachines.infrastructure.cluster.x-k8s.io
	kubectl wait --for condition=established --timeout=60s crds/openstackmachinetemplates.infrastructure.cluster.x-k8s.io
	kubectl wait --for condition=established --timeout=60s crds/openstackclusters.infrastructure.cluster.x-k8s.io

  # Wait until everything is really ready, as we had some problems with pods being ready but not yet
  # available when deploying the cluster.
	sleep 5

	# Deploy cluster
  kubectl create ns "${CLUSTER_NAME}" || true
	kubectl apply -f ./out/cluster.yaml

	# Wait for the kubeconfig to become available.

	timeout 300 bash -c "while ! kubectl -n ${CLUSTER_NAME} get secrets | grep ${CLUSTER_NAME}-kubeconfig; do kubectl -n ${CLUSTER_NAME} get secrets; sleep 10; done"
	# Get kubeconfig and store it locally.
	kubectl -n "${CLUSTER_NAME}" get secrets "${CLUSTER_NAME}-kubeconfig" -o json | jq -r .data.value | base64 --decode > ./kubeconfig
	timeout 900 bash -c "while ! kubectl --kubeconfig=./kubeconfig get nodes | grep control-plane; do kubectl --kubeconfig=./kubeconfig get nodes; sleep 10; done"

	# Deploy calico
	curl https://docs.projectcalico.org/manifests/calico.yaml | sed "s/veth_mtu:.*/veth_mtu: \"1430\"/g" | \
		kubectl --kubeconfig=./kubeconfig apply -f -

  # Wait till all machines are running (bail out at 30 mins)
  attempt=0
  while true; do
    kubectl -n "${CLUSTER_NAME}" get machines
    read running total <<< $(kubectl -n "${CLUSTER_NAME}" get machines \
      -o json | jq -r '.items[].status.phase' | awk 'BEGIN{count=0} /(r|R)unning/{count++} END{print count " " NR}') ;
    if [[ ${total} ==  ${running} ]]; then
      return 0
    fi
    read failed total <<< $(kubectl -n "${CLUSTER_NAME}" get machines \
      -o json | jq -r '.items[].status.phase' | awk 'BEGIN{count=0} /(f|F)ailed/{count++} END{print count " " NR}') ;
    if [[ ! ${failed} -eq 0 ]]; then
      echo "$failed machines (out of $total) in cluster failed ... bailing out"
      exit 1
    fi
    timestamp=$(date +"[%H:%M:%S]")
    if [[ ${attempt} -gt 180 ]]; then
      echo "cluster did not start in 30 mins ... bailing out!"
      exit 1
    fi
    echo "$timestamp Total machines : $total / Running : $running .. waiting for 10 seconds"
    sleep 10
    attempt=$((attempt+1))
  done

  # Wait till all pods and nodes are ready
  kubectl --kubeconfig=./kubeconfig wait --for=condition=Ready --timeout=15m pods -n kube-system --all
  kubectl --kubeconfig=./kubeconfig wait --for=condition=Ready --timeout=5m node --all
}

delete_cluster() {
  kubectl -n "${CLUSTER_NAME}" delete cluster --all --ignore-not-found
	kubectl -n "${CLUSTER_NAME}" get machinedeployment,kubeadmcontrolplane,cluster

	if [[ $(kubectl -n "${CLUSTER_NAME}" get machinedeployment,kubeadmcontrolplane,cluster | wc -l) -gt 0 ]]
	then
	  echo "Error: not all resources have been deleted correctly"
	  exit 1
	fi
}

# run e2es with kubetest
run_tests() {
  # export the KUBECONFIG
  KUBECONFIG="${PWD}/kubeconfig"
  export KUBECONFIG

  # ginkgo regexes
  SKIP="${SKIP:-}"
  FOCUS="${FOCUS:-"\\[Conformance\\]"}"
  # if we set PARALLEL=true, skip serial tests set --ginkgo-parallel
  if [[ "${PARALLEL:-false}" == "true" ]]; then
    export GINKGO_PARALLEL=y
    export GINKGO_PARALLEL_NODES=5
    echo "Running tests in parallel"
    if [[ -z "${SKIP}" ]]; then
      SKIP="\\[Serial\\]"
    else
      SKIP="\\[Serial\\]|${SKIP}"
    fi
  fi

  # get the number of worker nodes
  # TODO(bentheelder): this is kinda gross
  NUM_NODES="$(kubectl get nodes --kubeconfig="$KUBECONFIG" \
    -o=jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.taints}{"\n"}{end}' \
    | grep -cv "node-role.kubernetes.io/master" )"

  # setting this env prevents ginkgo e2e from trying to run provider setup
  export KUBERNETES_CONFORMANCE_TEST="y"
  # run the tests
  (cd "$(go env GOPATH)/src/k8s.io/kubernetes" && ./hack/ginkgo-e2e.sh \
    '--provider=skeleton' "--num-nodes=${NUM_NODES}" \
    "--ginkgo.focus=${FOCUS}" "--ginkgo.skip=${SKIP}" \
    "--report-dir=${ARTIFACTS}" '--disable-log-dump=true' | tee ${ARTIFACTS}/e2e.log)

  unset KUBECONFIG
  unset KUBERNETES_CONFORMANCE_TEST
}

# setup kind, build kubernetes, create a cluster, run the e2es
main() {
  for arg in "$@"
  do
    if [[ "$arg" == "--verbose" ]]; then
      set -o xtrace
    fi
    if [[ "$arg" == "--install-prereqs" ]]; then
      INSTALL_PREREQS="1"
    fi
    if [[ "$arg" == "--skip-cleanup" ]]; then
      SKIP_CLEANUP="1"
    fi
    if [[ "$arg" == "--use-ci-artifacts" ]]; then
      USE_CI_ARTIFACTS="true"
    fi
    if [[ "$arg" == "--skip-upload-image" ]]; then
      SKIP_UPLOAD_IMAGE="1"
    fi
    if [[ "$arg" == "--skip-run-tests" ]]; then
      SKIP_RUN_TESTS="1"
    fi
    if [[ "$arg" == "--run-tests-parallel" ]]; then
      export PARALLEL="true"
    fi
    if [[ "$arg" == "--delete-cluster" ]]; then
      DELETE_CLUSTER="1"
    fi
  done

  # create temp dir and setup cleanup
  SKIP_CLEANUP=${SKIP_CLEANUP:-""}
  if [[ -z "${SKIP_CLEANUP}" ]]; then
    trap dump_logs EXIT
  fi
  # ensure artifacts exists when not in CI
  export ARTIFACTS
  mkdir -p "${ARTIFACTS}/logs"

  source "${REPO_ROOT}/hack/ensure-go.sh"
  source "${REPO_ROOT}/hack/ensure-kind.sh"

  export GOPATH=${GOPATH:-/home/ubuntu/go}
  export PATH=$PATH:${GOPATH}/bin:/snap/bin:${HOME}/bin

  INSTALL_PREREQS=${INSTALL_PREREQS:-""}
  if [[ "${INSTALL_PREREQS}" == "yes" || "${INSTALL_PREREQS}" == "1" ]]; then
    echo "Install prereqs..."
    install_prereqs
  fi
  if [[ -z "${SKIP_UPLOAD_IMAGE:-}" ]]; then
    echo "Uploading image..."
    upload_image ${OPENSTACK_IMAGE_NAME} ${IMAGE_URL}
    upload_image ${OPENSTACK_BASTION_IMAGE_NAME} ${CIRROS_URL}
  fi

  build
  create_key_pair
  create_cluster

  if [[ -z "${SKIP_RUN_TESTS:-}" ]]; then
    echo "Running tests..."
    run_tests
  fi

  DELETE_CLUSTER=${DELETE_CLUSTER:-""}
  if [[ "${DELETE_CLUSTER}" == "yes" || "${DELETE_CLUSTER}" == "1" ]]; then
    echo "Dumping logs"
    dump_logs

    echo "Deleting cluster..."
    delete_cluster
  fi
}

main "$@"
