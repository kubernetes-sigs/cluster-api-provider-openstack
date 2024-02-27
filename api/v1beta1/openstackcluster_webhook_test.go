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

package v1beta1

import (
	"testing"

	. "github.com/onsi/gomega"
	"k8s.io/utils/pointer"
)

func TestOpenStackCluster_ValidateUpdate(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name        string
		oldTemplate *OpenStackCluster
		newTemplate *OpenStackCluster
		wantErr     bool
	}{
		{
			name: "Changing OpenStackCluster.Spec.IdentityRef.Name is allowed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &OpenStackIdentityReference{
						Name: "foobar",
					},
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &OpenStackIdentityReference{
						Name: "foobarbaz",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "OpenStackCluster.Spec.IdentityRef can be changed if it was unset",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &OpenStackIdentityReference{
						Name: "foobar",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "OpenStackCluster.Spec.IdentityRef must not be removed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &OpenStackIdentityReference{
						Name: "foobar",
					},
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
				},
			},
			wantErr: true,
		},
		{
			name: "Changing OpenStackCluster.Spec.Bastion is allowed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					Bastion: &Bastion{
						Instance: OpenStackMachineSpec{
							CloudName: "foobar",
							Image:     ImageFilter{Name: "foobar"},
							Flavor:    "minimal",
						},
						Enabled: true,
					},
				},
				Status: OpenStackClusterStatus{
					Bastion: &BastionStatus{
						Name: "foobar",
					},
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					Bastion: &Bastion{
						Instance: OpenStackMachineSpec{
							CloudName: "foobarbaz",
							Image:     ImageFilter{Name: "foobarbaz"},
							Flavor:    "medium",
						},
						Enabled: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Changing security group rules on the OpenStackCluster.Spec.ManagedSecurityGroups.AllNodesSecurityGroupRules is allowed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					ManagedSecurityGroups: &ManagedSecurityGroups{
						AllNodesSecurityGroupRules: []SecurityGroupRuleSpec{
							{
								Name:                "foobar",
								Description:         pointer.String("foobar"),
								PortRangeMin:        pointer.Int(80),
								PortRangeMax:        pointer.Int(80),
								Protocol:            pointer.String("tcp"),
								RemoteManagedGroups: []ManagedSecurityGroupName{"controlplane"},
							},
						},
					},
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					ManagedSecurityGroups: &ManagedSecurityGroups{
						AllNodesSecurityGroupRules: []SecurityGroupRuleSpec{
							{
								Name:                "foobar",
								Description:         pointer.String("foobar"),
								PortRangeMin:        pointer.Int(80),
								PortRangeMax:        pointer.Int(80),
								Protocol:            pointer.String("tcp"),
								RemoteManagedGroups: []ManagedSecurityGroupName{"controlplane", "worker"},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Changing CIDRs on the OpenStackCluster.Spec.APIServerLoadBalancer.AllowedCIDRs is allowed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					APIServerLoadBalancer: APIServerLoadBalancer{
						Enabled: true,
						AllowedCIDRs: []string{
							"0.0.0.0/0",
							"192.168.10.0/24",
						},
					},
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					APIServerLoadBalancer: APIServerLoadBalancer{
						Enabled: true,
						AllowedCIDRs: []string{
							"0.0.0.0/0",
							"192.168.10.0/24",
							"10.6.0.0/16",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Adding OpenStackCluster.Spec.ControlPlaneAvailabilityZones is allowed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					ControlPlaneAvailabilityZones: []string{
						"alice",
						"bob",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Modifying OpenStackCluster.Spec.ControlPlaneAvailabilityZones is allowed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					ControlPlaneAvailabilityZones: []string{
						"alice",
						"bob",
					},
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					ControlPlaneAvailabilityZones: []string{
						"alice",
						"bob",
						"eve",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Removing OpenStackCluster.Spec.ControlPlaneAvailabilityZones is allowed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					ControlPlaneAvailabilityZones: []string{
						"alice",
						"bob",
					},
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
				},
			},
			wantErr: false,
		},
		{
			name: "Modifying OpenstackCluster.Spec.ControlPlaneOmitAvailabilityZone is allowed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName:                        "foobar",
					ControlPlaneOmitAvailabilityZone: true,
				},
			},
			wantErr: false,
		},
		{
			name: "Changing OpenStackCluster.Spec.APIServerFixedIP is allowed when API Server Floating IP is disabled",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					DisableAPIServerFloatingIP: true,
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					DisableAPIServerFloatingIP: true,
					APIServerFixedIP:           "20.1.56.1",
				},
			},
			wantErr: false,
		},
		{
			name: "Changing OpenStackCluster.Spec.APIServerFixedIP is not allowed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					DisableAPIServerFloatingIP: false,
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					DisableAPIServerFloatingIP: false,
					APIServerFixedIP:           "20.1.56.1",
				},
			},
			wantErr: true,
		},

		{
			name: "Changing OpenStackCluster.Spec.APIServerPort is allowed when API Server Floating IP is disabled",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					DisableAPIServerFloatingIP: true,
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					DisableAPIServerFloatingIP: true,
					APIServerPort:              8443,
				},
			},
			wantErr: false,
		},
		{
			name: "Changing OpenStackCluster.Spec.APIServerPort is not allowed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					DisableAPIServerFloatingIP: false,
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					DisableAPIServerFloatingIP: false,
					APIServerPort:              8443,
				},
			},
			wantErr: true,
		},
		{
			name: "Changing OpenStackCluster.Spec.APIServerFloatingIP is allowed when it matches the current api server loadbalancer IP",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					APIServerFloatingIP: "",
				},
				Status: OpenStackClusterStatus{
					APIServerLoadBalancer: &LoadBalancer{
						IP: "1.2.3.4",
					},
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					APIServerFloatingIP: "1.2.3.4",
				},
				Status: OpenStackClusterStatus{
					APIServerLoadBalancer: &LoadBalancer{
						IP: "1.2.3.4",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Changing OpenStackCluster.Spec.APIServerFloatingIP is not allowed when it doesn't matches the current api server loadbalancer IP",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					APIServerFloatingIP: "",
				},
				Status: OpenStackClusterStatus{
					APIServerLoadBalancer: &LoadBalancer{
						IP: "1.2.3.4",
					},
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					APIServerFloatingIP: "5.6.7.8",
				},
				Status: OpenStackClusterStatus{
					APIServerLoadBalancer: &LoadBalancer{
						IP: "1.2.3.4",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Removing OpenStackCluster.Spec.Bastion when it is enabled is not allowed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					Bastion: &Bastion{
						Enabled: true,
						Instance: OpenStackMachineSpec{
							Flavor: "m1.small",
							Image:  ImageFilter{Name: "ubuntu"},
						},
					},
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{},
			},
			wantErr: true,
		},
		{
			name: "Removing OpenStackCluster.Spec.Bastion when it is disabled is allowed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					Bastion: &Bastion{
						Enabled: false,
						Instance: OpenStackMachineSpec{
							Flavor: "m1.small",
							Image:  ImageFilter{Name: "ubuntu"},
						},
					},
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warn, err := tt.newTemplate.ValidateUpdate(tt.oldTemplate)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
			// Nothing emits warnings yet
			g.Expect(warn).To(BeEmpty())
		})
	}
}

func TestOpenStackCluster_ValidateCreate(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name     string
		template *OpenStackCluster
		wantErr  bool
	}{
		{
			name: "OpenStackCluster.Spec.IdentityRef with correct spec on create",
			template: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &OpenStackIdentityReference{
						Name: "foobar",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "OpenStackCluster.Spec.ManagedSecurityGroups.AllNodesSecurityGroupRules with correct spec on create",
			template: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					ManagedSecurityGroups: &ManagedSecurityGroups{
						AllNodesSecurityGroupRules: []SecurityGroupRuleSpec{
							{
								Name:         "foobar",
								Description:  pointer.String("foobar"),
								PortRangeMin: pointer.Int(80),
								PortRangeMax: pointer.Int(80),
								Protocol:     pointer.String("tcp"),
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "OpenStackCluster.Spec.ManagedSecurityGroups.AllNodesSecurityGroupRules with mutually exclusive fields on create",
			template: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					ManagedSecurityGroups: &ManagedSecurityGroups{
						AllNodesSecurityGroupRules: []SecurityGroupRuleSpec{
							{
								Name:                "foobar",
								Description:         pointer.String("foobar"),
								PortRangeMin:        pointer.Int(80),
								PortRangeMax:        pointer.Int(80),
								Protocol:            pointer.String("tcp"),
								RemoteManagedGroups: []ManagedSecurityGroupName{"controlplane"},
								RemoteGroupID:       pointer.String("foobar"),
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warn, err := tt.template.ValidateCreate()
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
			// Nothing emits warnings yet
			g.Expect(warn).To(BeEmpty())
		})
	}
}
