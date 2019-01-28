package userdata

/*
This file is auto-generated DO NOT TOUCH!
*/

const (
	masterKubeadmCloudConfig = `apiVersion: kubeadm.k8s.io/v1alpha3
kind: InitConfiguration
apiEndpoint:
  bindPort: 443
nodeRegistration:
---
apiVersion: kubeadm.k8s.io/v1alpha3
kind: ClusterConfiguration
kubernetesVersion: v{{.ControlPlaneVersion}}
networking:
  serviceSubnet: {{.ServiceCIDR}}
clusterName: kubernetes
controlPlaneEndpoint: {{.ControlPlaneEndpoint}}
controllerManagerExtraArgs:
  cluster-cidr: {{.PodCIDR}}
  service-cluster-ip-range: {{.ServiceCIDR}}
  allocate-node-cidrs: "true"

`
)
