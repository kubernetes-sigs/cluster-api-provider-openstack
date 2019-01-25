/*
This file is auto-generated DO NOT TOUCH!
*/
package userdata

const (
	workerCloudConfig = `#cloud-config

write_files:
  - path: /etc/kubernetes/kubeadm_config.yaml
    permissions: "0444"
    encoding: b64
    content: {{.KubeadmConfig}}
{{- if (ne .CloudConf "") }}
  - path: /etc/kubernetes/cloud.conf
    encoding: b64
    permissions: "0444"
    content: {{.CloudConf}}
{{- end }}

merge_how: "list(append)+dict(recurse_array)+str()"

`
)
