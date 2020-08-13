apiVersion: controlplane.cluster.x-k8s.io/v1alpha3
kind: KubeadmControlPlane
metadata:
  name: "${CLUSTER_NAME}-control-plane"
spec:
  kubeadmConfigSpec:
    initConfiguration:
      nodeRegistration:
        kubeletExtraArgs:
          v: "8"
    joinConfiguration:
      nodeRegistration:
        kubeletExtraArgs:
          v: "8"
    verbosity: 8
---
apiVersion: bootstrap.cluster.x-k8s.io/v1alpha3
kind: KubeadmConfigTemplate
metadata:
  name: ${CLUSTER_NAME}-md-0
spec:
  template:
    spec:
      verbosity: 8
