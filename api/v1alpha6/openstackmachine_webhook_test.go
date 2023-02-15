/*
Copyright 2022 The Kubernetes Authors.

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

package v1alpha6

import (
	"testing"
)

func TestOpenStackMachine_ValidateUpdate(t *testing.T) {
	emptyFailureDomain := func() *FailureDomain {
		return &FailureDomain{}
	}
	populatedFailureDomain := func() *FailureDomain {
		return &FailureDomain{
			ComputeAvailabilityZone: "nova-az",
			StorageAvailabilityZone: "nova-az",
			Ports: []PortOpts{
				{
					Network: &NetworkFilter{ID: "0de96287-ecdf-434e-97d0-4a1a305cfd2e"},
					FixedIPs: []FixedIP{
						{
							Subnet: &SubnetFilter{
								ID: "47e505f9-fa48-43f9-8cb9-e5016f60a54b",
							},
						},
					},
				},
			},
		}
	}

	tests := []struct {
		name    string
		old     OpenStackMachine
		new     OpenStackMachine
		wantErr bool
	}{
		{
			name:    "Permit no change to the Failure Domain",
			old:     OpenStackMachine{},
			new:     OpenStackMachine{},
			wantErr: false,
		},
		{
			name: "Permit initially setting empty Failure Domain",
			old:  OpenStackMachine{},
			new: OpenStackMachine{
				Status: OpenStackMachineStatus{
					FailureDomain: emptyFailureDomain(),
				},
			},
		},
		{
			name: "Permit initially setting populated Failure Domain",
			old:  OpenStackMachine{},
			new: OpenStackMachine{
				Status: OpenStackMachineStatus{
					FailureDomain: populatedFailureDomain(),
				},
			},
		},
		{
			name: "Do not permit changing empty Failure Domain",
			old: OpenStackMachine{
				Status: OpenStackMachineStatus{
					FailureDomain: emptyFailureDomain(),
				},
			},
			new: OpenStackMachine{
				Status: OpenStackMachineStatus{
					FailureDomain: populatedFailureDomain(),
				},
			},
			wantErr: true,
		},
		{
			name: "Do not permit changing populated Failure Domain",
			old: OpenStackMachine{
				Status: OpenStackMachineStatus{
					FailureDomain: populatedFailureDomain(),
				},
			},
			new: OpenStackMachine{
				Status: OpenStackMachineStatus{
					FailureDomain: emptyFailureDomain(),
				},
			},
			wantErr: true,
		},
		{
			name: "Do not permit unsetting empty Failure Domain",
			old: OpenStackMachine{
				Status: OpenStackMachineStatus{
					FailureDomain: emptyFailureDomain(),
				},
			},
			new:     OpenStackMachine{},
			wantErr: true,
		},
		{
			name: "Do not permit unsetting populated Failure Domain",
			old: OpenStackMachine{
				Status: OpenStackMachineStatus{
					FailureDomain: populatedFailureDomain(),
				},
			},
			new:     OpenStackMachine{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.new.ValidateUpdate(&tt.old); (err != nil) != tt.wantErr {
				t.Errorf("OpenStackMachine.ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
