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

	. "github.com/onsi/gomega"
	"k8s.io/utils/pointer"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

func TestOpenStackCluster_ValidateUpdate(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name        string
		oldTemplate *infrav1.OpenStackCluster
		newTemplate *infrav1.OpenStackCluster
		wantErr     bool
	}{
		{
			name: "Changing OpenStackCluster.Spec.IdentityRef.Name is allowed",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobarbaz",
						CloudName: "foobar",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Changing OpenStackCluster.Spec.IdentityRef.CloudName is allowed",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobarbaz",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Changing OpenStackCluster.Spec.Bastion is allowed",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					Bastion: &infrav1.Bastion{
						Spec: &infrav1.OpenStackMachineSpec{
							Image:  infrav1.ImageFilter{Name: pointer.String("foobar")},
							Flavor: "minimal",
						},
						Enabled: true,
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Bastion: &infrav1.BastionStatus{
						Name: "foobar",
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					Bastion: &infrav1.Bastion{
						Spec: &infrav1.OpenStackMachineSpec{
							Image:  infrav1.ImageFilter{Name: pointer.String("foobarbaz")},
							Flavor: "medium",
						},
						Enabled: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Changing security group rules on the OpenStackCluster.Spec.ManagedSecurityGroups.AllNodesSecurityGroupRules is allowed",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					ManagedSecurityGroups: &infrav1.ManagedSecurityGroups{
						AllNodesSecurityGroupRules: []infrav1.SecurityGroupRuleSpec{
							{
								Name:                "foobar",
								Description:         pointer.String("foobar"),
								PortRangeMin:        pointer.Int(80),
								PortRangeMax:        pointer.Int(80),
								Protocol:            pointer.String("tcp"),
								RemoteManagedGroups: []infrav1.ManagedSecurityGroupName{"controlplane"},
							},
						},
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					ManagedSecurityGroups: &infrav1.ManagedSecurityGroups{
						AllNodesSecurityGroupRules: []infrav1.SecurityGroupRuleSpec{
							{
								Name:                "foobar",
								Description:         pointer.String("foobar"),
								PortRangeMin:        pointer.Int(80),
								PortRangeMax:        pointer.Int(80),
								Protocol:            pointer.String("tcp"),
								RemoteManagedGroups: []infrav1.ManagedSecurityGroupName{"controlplane", "worker"},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Changing CIDRs on the OpenStackCluster.Spec.APIServerLoadBalancer.AllowedCIDRs is allowed",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
						Enabled: pointer.Bool(true),
						AllowedCIDRs: []string{
							"0.0.0.0/0",
							"192.168.10.0/24",
						},
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
						Enabled: pointer.Bool(true),
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
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
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
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					ControlPlaneAvailabilityZones: []string{
						"alice",
						"bob",
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
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
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					ControlPlaneAvailabilityZones: []string{
						"alice",
						"bob",
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Modifying OpenstackCluster.Spec.ControlPlaneOmitAvailabilityZone is allowed",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					ControlPlaneOmitAvailabilityZone: pointer.Bool(true),
				},
			},
			wantErr: false,
		},
		{
			name: "Changing OpenStackCluster.Spec.APIServerFixedIP is allowed when API Server Floating IP is disabled",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					DisableAPIServerFloatingIP: pointer.Bool(true),
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					DisableAPIServerFloatingIP: pointer.Bool(true),
					APIServerFixedIP:           pointer.String("20.1.56.1"),
				},
			},
			wantErr: false,
		},
		{
			name: "Changing OpenStackCluster.Spec.APIServerFixedIP is not allowed",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					DisableAPIServerFloatingIP: pointer.Bool(false),
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					DisableAPIServerFloatingIP: pointer.Bool(false),
					APIServerFixedIP:           pointer.String("20.1.56.1"),
				},
			},
			wantErr: true,
		},

		{
			name: "Changing OpenStackCluster.Spec.APIServerPort is allowed when API Server Floating IP is disabled",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					DisableAPIServerFloatingIP: pointer.Bool(true),
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					DisableAPIServerFloatingIP: pointer.Bool(true),
					APIServerPort:              pointer.Int(8443),
				},
			},
			wantErr: false,
		},
		{
			name: "Changing OpenStackCluster.Spec.APIServerPort is not allowed",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					DisableAPIServerFloatingIP: pointer.Bool(false),
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					DisableAPIServerFloatingIP: pointer.Bool(false),
					APIServerPort:              pointer.Int(8443),
				},
			},
			wantErr: true,
		},
		{
			name: "Changing OpenStackCluster.Spec.APIServerFloatingIP is allowed when it matches the current api server loadbalancer IP",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					APIServerLoadBalancer: &infrav1.LoadBalancer{
						IP: "1.2.3.4",
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					APIServerFloatingIP: pointer.String("1.2.3.4"),
				},
				Status: infrav1.OpenStackClusterStatus{
					APIServerLoadBalancer: &infrav1.LoadBalancer{
						IP: "1.2.3.4",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Changing OpenStackCluster.Spec.APIServerFloatingIP is not allowed when it doesn't matches the current api server loadbalancer IP",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					APIServerLoadBalancer: &infrav1.LoadBalancer{
						IP: "1.2.3.4",
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					APIServerFloatingIP: pointer.String("5.6.7.8"),
				},
				Status: infrav1.OpenStackClusterStatus{
					APIServerLoadBalancer: &infrav1.LoadBalancer{
						IP: "1.2.3.4",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Removing OpenStackCluster.Spec.Bastion when it is enabled is not allowed",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					Bastion: &infrav1.Bastion{
						Enabled: true,
						Spec: &infrav1.OpenStackMachineSpec{
							Flavor: "m1.small",
							Image:  infrav1.ImageFilter{Name: pointer.String("ubuntu")},
						},
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Removing OpenStackCluster.Spec.Bastion when it is disabled is allowed",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					Bastion: &infrav1.Bastion{
						Enabled: false,
						Spec: &infrav1.OpenStackMachineSpec{
							Flavor: "m1.small",
							Image:  infrav1.ImageFilter{Name: pointer.String("ubuntu")},
						},
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			webhook := &openStackClusterWebhook{}
			warn, err := webhook.ValidateUpdate(ctx, tt.oldTemplate, tt.newTemplate)
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
		template *infrav1.OpenStackCluster
		wantErr  bool
	}{
		{
			name: "OpenStackCluster.Spec.IdentityRef with correct spec on create",
			template: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "OpenStackCluster.Spec.ManagedSecurityGroups.AllNodesSecurityGroupRules with correct spec on create",
			template: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					ManagedSecurityGroups: &infrav1.ManagedSecurityGroups{
						AllNodesSecurityGroupRules: []infrav1.SecurityGroupRuleSpec{
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
			template: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "foobar",
						CloudName: "foobar",
					},
					ManagedSecurityGroups: &infrav1.ManagedSecurityGroups{
						AllNodesSecurityGroupRules: []infrav1.SecurityGroupRuleSpec{
							{
								Name:                "foobar",
								Description:         pointer.String("foobar"),
								PortRangeMin:        pointer.Int(80),
								PortRangeMax:        pointer.Int(80),
								Protocol:            pointer.String("tcp"),
								RemoteManagedGroups: []infrav1.ManagedSecurityGroupName{"controlplane"},
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
			ctx := context.TODO()
			webhook := &openStackClusterWebhook{}
			warn, err := webhook.ValidateCreate(ctx, tt.template)
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
