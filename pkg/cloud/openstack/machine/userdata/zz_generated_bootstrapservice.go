package userdata

/*
This file is auto-generated DO NOT TOUCH!
*/

const (
	bootstrapService = `[Unit]
Description=Bootstrap a Kubernetes Node

[Service]
Type=oneshot
RemainAfterExit=yes
EnvironmentFile=-/etc/default/bootstrap-kubernetes
ExecStart=/usr/local/bin/bootstrap-kubernetes
`
)
