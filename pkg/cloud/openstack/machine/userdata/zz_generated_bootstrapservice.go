package userdata

/*
This file is auto-generated DO NOT TOUCH!
*/

const (
	bootstrapService = `[Unit]
Description=Bootstrap a Kubernetes Node

[Service]
Type=simple
Restart=on-failure
RestartSec=10s
EnvironmentFile=-/etc/default/bootstrap-kubernetes
ExecStart=/usr/local/bin/bootstrap-kubernetes
`
)
