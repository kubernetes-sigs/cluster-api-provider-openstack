/*
Copyright 2023 The Kubernetes Authors.

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

package webhooks

import (
	"context"
	"testing"

	. "github.com/onsi/gomega" //nolint:revive
	"k8s.io/utils/ptr"

	infrav1beta2 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
)

func TestOpenStackMachine_ValidateCreate(t *testing.T) {
	tests := []struct {
		name    string
		machine *infrav1beta2.OpenStackMachine
		wantErr bool
	}{
		{
			name: "RootVolume and AdditionalBlockDevices with conflicting name",
			machine: &infrav1beta2.OpenStackMachine{
				Spec: infrav1beta2.OpenStackMachineSpec{
					Flavor: ptr.To("m1.small"),
					Image: infrav1beta2.ImageParam{
						Filter: &infrav1beta2.ImageFilter{
							Name: ptr.To("ubuntu"),
						},
					},
					RootVolume: &infrav1beta2.RootVolume{
						SizeGiB: 50,
					},
					AdditionalBlockDevices: []infrav1beta2.AdditionalBlockDevice{
						{
							Name:    "root",
							SizeGiB: 10,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Port security disabled with security groups",
			machine: &infrav1beta2.OpenStackMachine{
				Spec: infrav1beta2.OpenStackMachineSpec{
					Flavor: ptr.To("m1.small"),
					Image: infrav1beta2.ImageParam{
						Filter: &infrav1beta2.ImageFilter{
							Name: ptr.To("ubuntu"),
						},
					},
					Ports: []infrav1beta2.PortOpts{
						{
							SecurityGroups: []infrav1beta2.SecurityGroupParam{{ID: ptr.To("sg-1")}},
							ResolvedPortSpecFields: infrav1beta2.ResolvedPortSpecFields{
								DisablePortSecurity: ptr.To(true),
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Valid machine spec",
			machine: &infrav1beta2.OpenStackMachine{
				Spec: infrav1beta2.OpenStackMachineSpec{
					Flavor: ptr.To("m1.small"),
					Image: infrav1beta2.ImageParam{
						Filter: &infrav1beta2.ImageFilter{
							Name: ptr.To("ubuntu"),
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			ctx := context.TODO()

			webhook := &openStackMachineWebhook{}
			warn, err := webhook.ValidateCreate(ctx, tt.machine)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
			g.Expect(warn).To(BeEmpty())
		})
	}
}

func TestOpenStackMachine_ValidateUpdate(t *testing.T) {
	tests := []struct {
		name       string
		oldMachine *infrav1beta2.OpenStackMachine
		newMachine *infrav1beta2.OpenStackMachine
		wantErr    bool
	}{
		{
			name: "ProviderID is immutable once set",
			oldMachine: &infrav1beta2.OpenStackMachine{
				Spec: infrav1beta2.OpenStackMachineSpec{
					Flavor:     ptr.To("m1.small"),
					ProviderID: ptr.To("openstack:///old-id"),
					Image: infrav1beta2.ImageParam{
						Filter: &infrav1beta2.ImageFilter{
							Name: ptr.To("ubuntu"),
						},
					},
				},
			},
			newMachine: &infrav1beta2.OpenStackMachine{
				Spec: infrav1beta2.OpenStackMachineSpec{
					Flavor:     ptr.To("m1.small"),
					ProviderID: ptr.To("openstack:///new-id"),
					Image: infrav1beta2.ImageParam{
						Filter: &infrav1beta2.ImageFilter{
							Name: ptr.To("ubuntu"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "ProviderID can be set for the first time",
			oldMachine: &infrav1beta2.OpenStackMachine{
				Spec: infrav1beta2.OpenStackMachineSpec{
					Flavor: ptr.To("m1.small"),
					Image: infrav1beta2.ImageParam{
						Filter: &infrav1beta2.ImageFilter{
							Name: ptr.To("ubuntu"),
						},
					},
				},
			},
			newMachine: &infrav1beta2.OpenStackMachine{
				Spec: infrav1beta2.OpenStackMachineSpec{
					Flavor:     ptr.To("m1.small"),
					ProviderID: ptr.To("openstack:///new-id"),
					Image: infrav1beta2.ImageParam{
						Filter: &infrav1beta2.ImageFilter{
							Name: ptr.To("ubuntu"),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "IdentityRef change is allowed",
			oldMachine: &infrav1beta2.OpenStackMachine{
				Spec: infrav1beta2.OpenStackMachineSpec{
					Flavor: ptr.To("m1.small"),
					Image: infrav1beta2.ImageParam{
						Filter: &infrav1beta2.ImageFilter{
							Name: ptr.To("ubuntu"),
						},
					},
					IdentityRef: &infrav1beta2.OpenStackIdentityReference{
						Name:      "old-ref",
						CloudName: "old-cloud",
					},
				},
			},
			newMachine: &infrav1beta2.OpenStackMachine{
				Spec: infrav1beta2.OpenStackMachineSpec{
					Flavor: ptr.To("m1.small"),
					Image: infrav1beta2.ImageParam{
						Filter: &infrav1beta2.ImageFilter{
							Name: ptr.To("ubuntu"),
						},
					},
					IdentityRef: &infrav1beta2.OpenStackIdentityReference{
						Name:      "new-ref",
						CloudName: "new-cloud",
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			ctx := context.TODO()

			webhook := &openStackMachineWebhook{}
			warn, err := webhook.ValidateUpdate(ctx, tt.oldMachine, tt.newMachine)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
			g.Expect(warn).To(BeEmpty())
		})
	}
}
