apiVersion: controlplane.cluster.x-k8s.io/v1alpha3
kind: KubeadmControlPlane
metadata:
  name: "${CLUSTER_NAME}-control-plane"
spec:
  kubeadmConfigSpec:
    clusterConfiguration:
      kubernetesVersion: "${KUBERNETES_VERSION}"
    initConfiguration:
      nodeRegistration:
        kubeletExtraArgs:
          v: "8"
    joinConfiguration:
      nodeRegistration:
        kubeletExtraArgs:
          v: "8"
    verbosity: 8
    preKubeadmCommands:
    - bash -c /tmp/kubeadm-bootstrap.sh
    files:
    - path: /etc/kubernetes/cloud.conf
      owner: root
      permissions: "0600"
      content: ${OPENSTACK_CLOUD_PROVIDER_CONF_B64}
      encoding: base64
    - path: /etc/certs/cacert
      owner: root
      permissions: "0600"
      content: ${OPENSTACK_CLOUD_CACERT_B64}
      encoding: base64
    - path: /tmp/kubeadm-bootstrap.sh
      owner: "root:root"
      permissions: "0744"
      content: |
        #!/bin/bash

        set -o nounset
        set -o pipefail
        set -o errexit
        set -e

        [[ $(id -u) != 0 ]] && SUDO="sudo" || SUDO=""

        # This script installs binaries and containers that are a result of the CI and release builds.
        # It runs '... --version' commands to verify that the binaries are correctly installed

        LINE_SEPARATOR="*************************************************"
        echo "$LINE_SEPARATOR"

        echo "$(date): stopping kubelet"
        ${SUDO} systemctl stop kubelet

        echo "$(date): debug output"
        ps aux
        top -b -n 1

        CI_VERSION=${CI_VERSION:-"${KUBERNETES_VERSION}"}
        if [[ "${CI_VERSION}" != "" ]]; then
          CI_DIR=/tmp/k8s-ci
          mkdir -p $CI_DIR
          # replace + with %2B for the URL
          CI_URL="https://storage.googleapis.com/kubernetes-release-dev/ci/${CI_VERSION//+/%2B}/bin/linux/amd64"
          declare -a BINARIES_TO_TEST=("kubectl" "kubelet" "kubeadm")
          declare -a CONTAINERS_TO_TEST=("kube-apiserver" "kube-controller-manager" "kube-scheduler" "kube-proxy")
          CONTAINER_EXT="tar"

          echo "* testing CI version $CI_VERSION"

          for CI_BINARY in "${BINARIES_TO_TEST[@]}"; do
            echo "$(date): downloading binary $CI_URL/$CI_BINARY"
            # move old binary away to avoid err "Text file busy"
            ${SUDO} mv /usr/bin/${CI_BINARY} /usr/bin/${CI_BINARY}.bak
            ${SUDO} curl --retry 5 -sS "${CI_URL}/${CI_BINARY}" -o "${CI_DIR}/${CI_BINARY}"
            ${SUDO} cp ${CI_DIR}/${CI_BINARY} /usr/bin/${CI_BINARY}
            ${SUDO} chmod +x /usr/bin/${CI_BINARY}
            echo "$(date): downloading binary $CI_URL/$CI_BINARY finished"
          done

          for CI_CONTAINER in "${CONTAINERS_TO_TEST[@]}"; do
            echo "$(date): downloading container $CI_URL/$CI_CONTAINER.$CONTAINER_EXT"
            ${SUDO} curl --retry 5 -sS "${CI_URL}/$CI_CONTAINER.$CONTAINER_EXT" -o "$CI_DIR/$CI_CONTAINER.$CONTAINER_EXT"
            ${SUDO} ctr -n k8s.io images import "$CI_DIR/$CI_CONTAINER.$CONTAINER_EXT" || echo "* ignoring expected 'ctr images import' result"
            ${SUDO} ctr -n k8s.io images tag k8s.gcr.io/$CI_CONTAINER-amd64:"${CI_VERSION//+/_}" k8s.gcr.io/$CI_CONTAINER:"${CI_VERSION//+/_}"
            ${SUDO} ctr -n k8s.io images tag k8s.gcr.io/$CI_CONTAINER-amd64:"${CI_VERSION//+/_}" gcr.io/kubernetes-ci-images/$CI_CONTAINER:"${CI_VERSION//+/_}"
            echo "$(date): downloading container $CI_URL/$CI_CONTAINER.$CONTAINER_EXT finished"
          done
        fi

        echo "$(date): checking binary versions"

        echo "ctr version: " $(ctr version)
        echo "kubeadm version: " $(kubeadm version -o=short)
        echo "kubectl version: " $(kubectl version --client=true --short=true)
        echo "kubelet version: " $(kubelet --version)

        echo "$LINE_SEPARATOR"
  version: "${KUBERNETES_VERSION}"
---
apiVersion: bootstrap.cluster.x-k8s.io/v1alpha3
kind: KubeadmConfigTemplate
metadata:
  name: ${CLUSTER_NAME}-md-0
spec:
  template:
    spec:
      verbosity: 8
      preKubeadmCommands:
        - bash -c /tmp/kubeadm-bootstrap.sh
      files:
        - content: ${OPENSTACK_CLOUD_PROVIDER_CONF_B64}
          encoding: base64
          owner: root
          path: /etc/kubernetes/cloud.conf
          permissions: "0600"
        - content: ${OPENSTACK_CLOUD_CACERT_B64}
          encoding: base64
          owner: root
          path: /etc/certs/cacert
          permissions: "0600"
        - path: /tmp/kubeadm-bootstrap.sh
          owner: "root:root"
          permissions: "0744"
          content: |
            #!/bin/bash

            set -o nounset
            set -o pipefail
            set -o errexit
            set -e

            [[ $(id -u) != 0 ]] && SUDO="sudo" || SUDO=""

            # This script installs binaries and containers that are a result of the CI and release builds.
            # It runs '... --version' commands to verify that the binaries are correctly installed

            LINE_SEPARATOR="*************************************************"
            echo "$LINE_SEPARATOR"

            echo "$(date): stopping kubelet"
            ${SUDO} systemctl stop kubelet

            echo "$(date): debug output"
            ps aux
            top -b -n 1

            CI_VERSION=${CI_VERSION:-"${KUBERNETES_VERSION}"}
            if [[ "${CI_VERSION}" != "" ]]; then
              CI_DIR=/tmp/k8s-ci
              mkdir -p $CI_DIR
              # replace + with %2B for the URL
              CI_URL="https://storage.googleapis.com/kubernetes-release-dev/ci/${CI_VERSION//+/%2B}/bin/linux/amd64"
              declare -a BINARIES_TO_TEST=("kubectl" "kubelet" "kubeadm")
              declare -a CONTAINERS_TO_TEST=("kube-proxy")
              CONTAINER_EXT="tar"

              echo "* testing CI version $CI_VERSION"

              for CI_BINARY in "${BINARIES_TO_TEST[@]}"; do
                echo "$(date): downloading binary $CI_URL/$CI_BINARY"
                # move old binary away to avoid err "Text file busy"
                ${SUDO} mv /usr/bin/${CI_BINARY} /usr/bin/${CI_BINARY}.bak
                ${SUDO} curl --retry 5 -sS "${CI_URL}/${CI_BINARY}" -o "${CI_DIR}/${CI_BINARY}"
                ${SUDO} cp ${CI_DIR}/${CI_BINARY} /usr/bin/${CI_BINARY}
                ${SUDO} chmod +x /usr/bin/${CI_BINARY}
                echo "$(date): downloading binary $CI_URL/$CI_BINARY finished"
              done

              for CI_CONTAINER in "${CONTAINERS_TO_TEST[@]}"; do
                echo "$(date): downloading container $CI_URL/$CI_CONTAINER.$CONTAINER_EXT"
                ${SUDO} curl --retry 5 -sS "${CI_URL}/$CI_CONTAINER.$CONTAINER_EXT" -o "$CI_DIR/$CI_CONTAINER.$CONTAINER_EXT"
                ${SUDO} ctr -n k8s.io images import "$CI_DIR/$CI_CONTAINER.$CONTAINER_EXT" || echo "* ignoring expected 'ctr images import' result"
                ${SUDO} ctr -n k8s.io images tag k8s.gcr.io/$CI_CONTAINER-amd64:"${CI_VERSION//+/_}" k8s.gcr.io/$CI_CONTAINER:"${CI_VERSION//+/_}"
                ${SUDO} ctr -n k8s.io images tag k8s.gcr.io/$CI_CONTAINER-amd64:"${CI_VERSION//+/_}" gcr.io/kubernetes-ci-images/$CI_CONTAINER:"${CI_VERSION//+/_}"
                echo "$(date): downloading container $CI_URL/$CI_CONTAINER.$CONTAINER_EXT finished"
              done
            fi

            echo "$(date): checking binary versions"

            echo "ctr version: " $(ctr version)
            echo "kubeadm version: " $(kubeadm version -o=short)
            echo "kubectl version: " $(kubectl version --client=true --short=true)
            echo "kubelet version: " $(kubelet --version)

            echo "$LINE_SEPARATOR"
---
apiVersion: cluster.x-k8s.io/v1alpha3
kind: MachineDeployment
metadata:
  name: "${CLUSTER_NAME}-md-0"
spec:
  clusterName: "${CLUSTER_NAME}"
  template:
    spec:
      version: "${KUBERNETES_VERSION}"
