/*
This file is auto-generated DO NOT TOUCH!
*/
package userdata

const (
	workerKubeadmCloudConfig = `apiVersion: kubeadm.k8s.io/v1alpha3
kind: JoinConfiguration
{{- if (ne .CloudConf "") }}
nodeRegistration:
  kubeletExtraArgs:
    cloud-provider: "openstack"
    cloud-config: "/etc/kubernetes/cloud.conf"
{{- end }}
token: {{.Token}}
discoveryTokenAPIServers:
  - {{.ControlPlaneEndpoint}}
discoveryTokenUnsafeSkipCAVerification: true
`
)
