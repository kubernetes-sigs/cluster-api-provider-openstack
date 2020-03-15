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

OPENSTACK_CLOUD_YAML_FILE=${OPENSTACK_CLOUD_YAML_FILE:-"/tmp/clouds.yaml"}
OPENSTACK_SSH_AUTHORIZED_KEY_PATH=${OPENSTACK_SSH_AUTHORIZED_KEY_PATH:-"/tmp/id_rsa.pub"}
OPENSTACK_SSH_PRIVATE_KEY_PATH=${OPENSTACK_SSH_PRIVATE_KEY_PATH:-"/tmp/id_rsa"}
OPENSTACK_IMAGE_NAME="ubuntu-1910-kube-v1.17.3"
CLUSTER_NAME=${CLUSTER_NAME:-"capi-quickstart"}
KUBERNETES_VERSION=${KUBERNETES_VERSION:-"v1.17.3"}
TIMESTAMP=$(date +"%Y-%m-%dT%H:%M:%SZ")

ARTIFACTS="${ARTIFACTS:-${PWD}/_artifacts}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"

# dump logs from kind and all the nodes
dump-logs() {

  echo "Dump logs"
  mkdir -p "${ARTIFACTS}/logs"

  # log version information
  echo "=== versions ==="
  echo "kind : $(kind version)" || true
  echo "bootstrap cluster:"
  kubectl version || true
  echo "deployed cluster:"
  kubectl --kubeconfig=${PWD}/kubeconfig version || true
  echo ""

  # dump all the info from the CAPI related CRDs
  kubectl get \
    clusters,openstackclusters,machines,openstackmachines,kubeadmconfigs,machinedeployments,openstackmachinetemplates,kubeadmconfigtemplates,machinesets \
    --all-namespaces -o yaml >> "${ARTIFACTS}/logs/capo.info" || true

  # dump images info
  echo "images in docker" >> "${ARTIFACTS}/logs/images.info"
  docker images >> "${ARTIFACTS}/logs/images.info"
  echo "images from bootstrap using containerd CLI" >> "${ARTIFACTS}/logs/images.info"
  docker exec clusterapi-control-plane ctr -n k8s.io images list >> "${ARTIFACTS}/logs/images.info" || true
  echo "images in bootstrap cluster using kubectl CLI" >> "${ARTIFACTS}/logs/images.info"
  (kubectl get pods --all-namespaces -o json \
   | jq --raw-output '.items[].spec.containers[].image' | sort)  >> "${ARTIFACTS}/logs/images.info" || true
  echo "images in deployed cluster using kubectl CLI" >> "${ARTIFACTS}/logs/images.info"
  (kubectl --kubeconfig="${PWD}"/kubeconfig get pods --all-namespaces -o json \
   | jq --raw-output '.items[].spec.containers[].image' | sort)  >> "${ARTIFACTS}/logs/images.info" || true

  # dump cluster info for kind
  kubectl cluster-info dump > "${ARTIFACTS}/logs/kind-cluster.info" || true

  # export all logs from kind
  kind "export" logs --name="clusterapi" "${ARTIFACTS}/logs" || true

  # dump OpenStack info
  echo "" > "${ARTIFACTS}/logs/openstack-cluster.info"
  echo "=== OpenStack compute instances list ===" >> "${ARTIFACTS}/logs/openstack-cluster.info" || true
  openstack server list >> "${ARTIFACTS}/logs/openstack-cluster.info" || true
  echo "=== OpenStack compute instances show ===" >> "${ARTIFACTS}/logs/openstack-cluster.info" || true
  openstack server list -f value -c Name | xargs -I% openstack server show % >> "${ARTIFACTS}/logs/openstack-cluster.info" || true
  echo "=== cluster-info dump ===" >> "${ARTIFACTS}/logs/openstack-cluster.info" || true
  kubectl --kubeconfig=${PWD}/kubeconfig cluster-info dump >> "${ARTIFACTS}/logs/openstack-cluster.info" || true

  set -x
  jump_node_name=$(openstack server list -f value -c Name | grep ${CLUSTER_NAME}-control-plane | head -n 1)
  jump_node=$(openstack server show ${jump_node_name} -f value -c addresses | awk '{print $2}')
  for node in $(openstack server list -f value -c Name)
  do
    echo "collecting logs from ${node} using jump host "
    dir="${ARTIFACTS}/logs/${node}"
    mkdir -p ${dir}

    openstack console log show "${node}" > "${dir}/console.log" || true

    ssh-to-node "${node}" "${jump_node}" "sudo chmod -R a+r /var/log" || true

    PROXY_COMMAND="ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=30 -x -W %h:22 -i ${OPENSTACK_SSH_PRIVATE_KEY_PATH} capo@${jump_node}"
    node=$(openstack port show ${node}  -f json -c fixed_ips | jq '.fixed_ips[0].ip_address' -r)
    scp -r -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=30 -o ProxyCommand="${PROXY_COMMAND}" -i ${OPENSTACK_SSH_PRIVATE_KEY_PATH} \
      "capo@${node}:/var/log/cloud-init.log" "capo@${node}:/var/log/cloud-init-output.log" \
      "capo@${node}:/var/log/pods" "capo@${node}:/var/log/containers" \
      "${dir}" || true

    ssh-to-node "${node}" "${jump_node}" "sudo journalctl --output=short-precise -k" > "${dir}/kern.log" || true
    ssh-to-node "${node}" "${jump_node}" "sudo journalctl --output=short-precise" > "${dir}/systemd.log" || true
    ssh-to-node "${node}" "${jump_node}" "sudo crictl version && sudo crictl info" > "${dir}/containerd.info" || true
    ssh-to-node "${node}" "${jump_node}" "sudo journalctl --no-pager -u cloud-final" > "${dir}/cloud-final.log" || true
    ssh-to-node "${node}" "${jump_node}" "sudo journalctl --no-pager -u kubelet.service" > "${dir}/kubelet.log" || true
    ssh-to-node "${node}" "${jump_node}" "sudo journalctl --no-pager -u containerd.service" > "${dir}/containerd.log" || true
    ssh-to-node "${node}" "${jump_node}" "sudo top -b -n 1" > "${dir}/top.txt" || true
    ssh-to-node "${node}" "${jump_node}" "sudo crictl ps" > "${dir}/crictl-ps.log" || true
    ssh-to-node "${node}" "${jump_node}" "sudo crictl pods" > "${dir}/crictl-pods.log" || true
  done
  set +x
}

# SSH to a node by name ($1) via jump server ($2) and run a command ($3).
function ssh-to-node() {
  local node="$1"
  local jump="$2"
  local cmd="$3"

  ssh_params="-o LogLevel=quiet -o ConnectTimeout=30 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no"
  scp $ssh_params -i "${OPENSTACK_SSH_PRIVATE_KEY_PATH}" "${OPENSTACK_SSH_PRIVATE_KEY_PATH}" "capo@${jump}:${OPENSTACK_SSH_PRIVATE_KEY_PATH}"
  ssh $ssh_params -i "${OPENSTACK_SSH_PRIVATE_KEY_PATH}" \
    -o "ProxyCommand ssh $ssh_params -W %h:%p -i ${OPENSTACK_SSH_PRIVATE_KEY_PATH} capo@${jump}" \
    capo@"${node}" "${cmd}"
}

# cleanup all resources we use
cleanup() {
  # KIND_IS_UP is true once we: kind create
  if [[ "${KIND_IS_UP:-}" = true ]]; then
    timeout 600 kubectl \
      delete cluster "${CLUSTER_NAME}" || true
     timeout 600 kubectl \
      wait --for=delete -n ${CLUSTER_NAME} cluster/"${CLUSTER_NAME}" || true
    make kind-reset || true
  fi
  # clean up e2e.test symlink
  (cd "$(go env GOPATH)/src/k8s.io/kubernetes" && rm -f _output/bin/e2e.test) || true
}

# our exit handler (trap)
exit-handler() {
  dump-logs
  cleanup
}

upload_image() {
  echo "Upload image"

  # Remove old image if we don't want to reuse it
  if [[ "${REUSE_OLD_IMAGES:-true}" == "false" ]]; then
    image_id=$(openstack image list --name=${OPENSTACK_IMAGE_NAME} -f value -c ID)
    if [[ ! -z "$image_id" ]]; then
        echo "Deleting old image ${OPENSTACK_IMAGE_NAME} with id: ${image_id}"
        openstack image delete ${image_id}
    fi
  fi

  image=$(openstack image list --name=${OPENSTACK_IMAGE_NAME} -f value -c Name)
  if [[ ! -z "$image" ]]; then
    echo "Image ${OPENSTACK_IMAGE_NAME} already exists"
    return
  fi

  tmp_source_image=/tmp/ubuntu-1910.ova.qcow2
  wget -q https://github.com/sbueringer/image-builder/releases/download/v1.17.3-04/ubuntu-1910-kube-v1.17.3.qcow2 -O ${tmp_source_image}

  echo "Uploading image ${tmp_source_image} as ${OPENSTACK_IMAGE_NAME}"
  openstack image create --disk-format qcow2 \
    --private \
    --container-format bare \
    --file "${tmp_source_image}" ${OPENSTACK_IMAGE_NAME}
}

install_prereqs() {
    # Install Docker
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
    sudo apt-key fingerprint 0EBFCD88
    sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
    sudo apt-get update
    sudo apt-get install -y docker-ce docker-ce-cli containerd.io jq
    # docker socket already works because OpenLab runs via root

    # Install go
    sudo snap install go --classic
    
    # Install yq
	GO111MODULE=on go get github.com/mikefarah/yq/v2

    source "${REPO_ROOT}/hack/ensure-go.sh"
    source "${REPO_ROOT}/hack/ensure-kubectl.sh"
    source "${REPO_ROOT}/hack/ensure-kind.sh"
    source "${REPO_ROOT}/hack/ensure-kustomize.sh"

    # Kubernetes does not build with modern bazel
    #sudo apt install curl
    #curl https://bazel.build/bazel-release.pub.gpg | sudo apt-key add -
    #echo "deb [arch=amd64] https://storage.googleapis.com/bazel-apt stable jdk1.8" | sudo tee /etc/apt/sources.list.d/bazel.list
    #sudo apt update && sudo apt install bazel
    wget -q https://github.com/bazelbuild/bazel/releases/download/0.28.1/bazel-0.28.1-installer-linux-x86_64.sh
    chmod +x bazel-0.28.1-installer-linux-x86_64.sh
    ./bazel-0.28.1-installer-linux-x86_64.sh --user

    # Bazel is installed in /root/bin
    export PATH="/root/bin:${PATH}"
}

# build kubernetes / node image, e2e binaries
build() {
  # possibly enable bazel build caching before building kubernetes
  if [[ "${BAZEL_REMOTE_CACHE_ENABLED:-false}" == "true" ]]; then
    create_bazel_cache_rcs.sh || true
  fi

  if [[ ! -d "$(go env GOPATH)/src/k8s.io/kubernetes" ]]; then
    mkdir -p $(go env GOPATH)/src/k8s.io
    cd $(go env GOPATH)/src/k8s.io

    git clone https://github.com/kubernetes/kubernetes.git
    cd kubernetes
    git checkout release-1.17
  fi

  pushd "$(go env GOPATH)/src/k8s.io/kubernetes"

  # make sure we have e2e requirements
  bazel build //cmd/kubectl //vendor/github.com/onsi/ginkgo/ginkgo

  # ensure the e2e script will find our binaries ...
  mkdir -p "${PWD}/_output/bin/"
  rm -f "${PWD}/_output/bin/e2e.test"
  if bazel --version | grep "0.28.1"
  then
    bazel build //test/e2e:e2e.test
    cp "${PWD}/bazel-bin/test/e2e/e2e.test" "${PWD}/_output/bin/e2e.test"
  else
    go test -c ./test/e2e/
    cp "./e2e.test" "${PWD}/_output/bin/e2e.test"
  fi
  PATH="$(dirname "$(find "${PWD}/bazel-bin/" -name kubectl -type f)"):${PATH}"
  export PATH

  # attempt to release some memory after building
  sync || true
  sudo sh -c "echo 1 > /proc/sys/vm/drop_caches" || true

  popd
}

# generate manifests needed for creating the GCP cluster to run the tests
generate_manifests() {
  cd ${REPO_ROOT}
  if ! command -v kustomize >/dev/null 2>&1; then
    (cd ./hack/tools/ && GO111MODULE=on go install sigs.k8s.io/kustomize/kustomize/v3)
  fi

  echo "Build Docker Images & Generating Manifests"

  # Enable the bits to inject a script that can pull newer versions of kubernetes
  if [[ -n ${CI_VERSION:-} || -n ${USE_CI_ARTIFACTS:-} ]]; then
    if ! grep -i -wq "patchesStrategicMerge" "templates/kustomization.yaml"; then
      echo "patchesStrategicMerge:" >> "templates/kustomization.yaml"
      echo "- kustomizeversions.yaml" >> "templates/kustomization.yaml"
    fi
  fi

  make modules docker-build
}

# up a cluster with kind
create_cluster() {
  # actually create the cluster
  KIND_IS_UP=true

  if [[ ! -f ${OPENSTACK_SSH_AUTHORIZED_KEY_PATH} ]]
  then
    ssh-keygen -t rsa -f ${OPENSTACK_SSH_PRIVATE_KEY_PATH}  -N ""
    chmod 0400 ${OPENSTACK_SSH_AUTHORIZED_KEY_PATH}
  fi

  # exports the b64 env vars used below
  source ${REPO_ROOT}/templates/env.rc ${OPENSTACK_CLOUD_YAML_FILE} ${CLUSTER_NAME}

  OPENSTACK_CLOUD_CACERT_B64=${OPENSTACK_CLOUD_CACERT_B64} \
  OPENSTACK_CLOUD_PROVIDER_CONF_B64=${OPENSTACK_CLOUD_PROVIDER_CONF_B64} \
  OPENSTACK_CLOUD_YAML_B64=${OPENSTACK_CLOUD_YAML_B64} \
  OPENSTACK_EXTERNAL_NETWORK_ID=$(openstack network show "public" -f value -c id) \
  OPENSTACK_IMAGE_NAME=${OPENSTACK_IMAGE_NAME} \
  OPENSTACK_SSH_AUTHORIZED_KEY="$(cat ${OPENSTACK_SSH_AUTHORIZED_KEY_PATH})" \
  KUBERNETES_VERSION=${KUBERNETES_VERSION} \
    make create-cluster

  # Wait till all machines are running (bail out at 30 mins)
  attempt=0
  while true; do
    kubectl get machines -n "${CLUSTER_NAME}"
    read running total <<< $(kubectl get machines -n "${CLUSTER_NAME}" \
      -o json | jq -r '.items[].status.phase' | awk 'BEGIN{count=0} /(r|R)unning/{count++} END{print count " " NR}') ;
    if [[ ${total} ==  ${running} ]]; then
      return 0
    fi
    read failed total <<< $(kubectl get machines -n "${CLUSTER_NAME}" \
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

  # wait for all the nodes to be ready
  kubectl wait --for=condition=Ready node --kubeconfig="$KUBECONFIG" --all || true

  # setting this env prevents ginkgo e2e from trying to run provider setup
  export KUBERNETES_CONFORMANCE_TEST="y"
  # run the tests
  (cd "$(go env GOPATH)/src/k8s.io/kubernetes" && ./hack/ginkgo-e2e.sh \
    '--provider=skeleton' "--num-nodes=${NUM_NODES}" \
    "--ginkgo.focus=${FOCUS}" "--ginkgo.skip=${SKIP}" \
    "--report-dir=${ARTIFACTS}" '--disable-log-dump=true')

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
    if [[ "$arg" == "--clean" ]]; then
      cleanup
      return 0
    fi
    if [[ "$arg" == "--install-prereqs" ]]; then
      INSTALL_PREREQS="1"
    fi
    if [[ "$arg" == "--skip-cleanup" ]]; then
      SKIP_CLEANUP="1"
    fi
    if [[ "$arg" == "--use-ci-artifacts" ]]; then
      USE_CI_ARTIFACTS="1"
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
  done

  # create temp dir and setup cleanup
  SKIP_CLEANUP=${SKIP_CLEANUP:-""}
  if [[ -z "${SKIP_CLEANUP}" ]]; then
    trap exit-handler EXIT
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
    upload_image
  fi

  build
  generate_manifests
  create_cluster

  if [[ -z "${SKIP_RUN_TESTS:-}" ]]; then
    echo "Running tests..."
    run_tests
  fi
}

main "$@"
