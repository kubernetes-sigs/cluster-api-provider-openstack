//go:build e2e

/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package shared

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Test_commandsForMachine can be used to develop and test
// executeCommands locally.
func Test_commandsForMachine(t *testing.T) {
	t.Skip()

	type args struct {
		machineIP string
		bastionIP string
		commands  []command
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "",
			args: args{
				machineIP: "10.6.0.230",
				bastionIP: "172.24.4.58",
				commands: []command{
					{
						title: "systemd",
						cmd:   "journalctl --no-pager --output=short-precise | grep -v  'audit:\\|audit\\['",
					},
					{
						title: "kern",
						cmd:   "journalctl --no-pager --output=short-precise -k",
					},
					{
						title: "containerd-info",
						cmd:   "crictl info",
					},
					{
						title: "containerd-containers",
						cmd:   "crictl ps",
					},
					{
						title: "containerd-pods",
						cmd:   "crictl pods",
					},
					{
						title: "cloud-final",
						cmd:   "journalctl --no-pager -u cloud-final",
					},
					{
						title: "kubelet",
						cmd:   "journalctl --no-pager -u kubelet.service",
					},
					{
						title: "containerd",
						cmd:   "journalctl --no-pager -u containerd.service",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			workingDir, err := os.Getwd()
			if err != nil {
				panic(err)
			}
			executeCommands(context.Background(), filepath.Join(workingDir, "..", "..", "..", "_artifacts"), true, "/tmp/", tt.args.machineIP, tt.args.bastionIP, "ubuntu", tt.args.commands)
		})
	}
}
