package userdata

/*
This file is auto-generated DO NOT TOUCH!
*/

const (
	workerKubeadmCloudConfig = `apiVersion: kubeadm.k8s.io/v1alpha3
kind: JoinConfiguration
token: {{.Token}}
discoveryTokenAPIServers:
  - {{.ControlPlaneEndpoint}}
discoveryTokenUnsafeSkipCAVerification: true

`
)
