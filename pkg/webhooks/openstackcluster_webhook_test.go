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

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha7"
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
			name: "OpenStackCluster.Spec.IdentityRef.Kind must always be Secret",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &infrav1.OpenStackIdentityReference{
						Kind: "Secret",
						Name: "foobar",
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &infrav1.OpenStackIdentityReference{
						Kind: "foobar",
						Name: "foobar",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Changing OpenStackCluster.Spec.IdentityRef.Name is allowed",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &infrav1.OpenStackIdentityReference{
						Kind: "Secret",
						Name: "foobar",
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &infrav1.OpenStackIdentityReference{
						Kind: "Secret",
						Name: "foobarbaz",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "OpenStackCluster.Spec.IdentityRef can be changed if it was unset",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					CloudName: "foobar",
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &infrav1.OpenStackIdentityReference{
						Kind: "Secret",
						Name: "foobar",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "OpenStackCluster.Spec.IdentityRef must not be removed",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &infrav1.OpenStackIdentityReference{
						Kind: "Secret",
						Name: "foobar",
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					CloudName: "foobar",
				},
			},
			wantErr: true,
		},
		{
			name: "Changing OpenStackCluster.Spec.Bastion is allowed",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					CloudName: "foobar",
					Bastion: &infrav1.Bastion{
						Instance: infrav1.OpenStackMachineSpec{
							CloudName: "foobar",
							Image:     "foobar",
							Flavor:    "minimal",
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
					CloudName: "foobar",
					Bastion: &infrav1.Bastion{
						Instance: infrav1.OpenStackMachineSpec{
							CloudName: "foobarbaz",
							Image:     "foobarbaz",
							Flavor:    "medium",
						},
						Enabled: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Changing CIDRs on the OpenStackCluster.Spec.APIServerLoadBalancer.AllowedCIDRs is allowed",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					CloudName: "foobar",
					APIServerLoadBalancer: infrav1.APIServerLoadBalancer{
						Enabled: true,
						AllowedCIDRs: []string{
							"0.0.0.0/0",
							"192.168.10.0/24",
						},
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					CloudName: "foobar",
					APIServerLoadBalancer: infrav1.APIServerLoadBalancer{
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
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					CloudName: "foobar",
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
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
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					CloudName: "foobar",
					ControlPlaneAvailabilityZones: []string{
						"alice",
						"bob",
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
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
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					CloudName: "foobar",
					ControlPlaneAvailabilityZones: []string{
						"alice",
						"bob",
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					CloudName: "foobar",
				},
			},
			wantErr: false,
		},
		{
			name: "Changing OpenStackCluster.Spec.APIServerFixedIP is allowed when API Server Floating IP is disabled",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					DisableAPIServerFloatingIP: true,
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					DisableAPIServerFloatingIP: true,
					APIServerFixedIP:           "20.1.56.1",
				},
			},
			wantErr: false,
		},
		{
			name: "Changing OpenStackCluster.Spec.APIServerFixedIP is not allowed",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					DisableAPIServerFloatingIP: false,
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					DisableAPIServerFloatingIP: false,
					APIServerFixedIP:           "20.1.56.1",
				},
			},
			wantErr: true,
		},

		{
			name: "Changing OpenStackCluster.Spec.APIServerPort is allowed when API Server Floating IP is disabled",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					DisableAPIServerFloatingIP: true,
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					DisableAPIServerFloatingIP: true,
					APIServerPort:              8443,
				},
			},
			wantErr: false,
		},
		{
			name: "Changing OpenStackCluster.Spec.APIServerPort is not allowed",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					DisableAPIServerFloatingIP: false,
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					DisableAPIServerFloatingIP: false,
					APIServerPort:              8443,
				},
			},
			wantErr: true,
		},
		{
			name: "Changing OpenStackCluster.Spec.APIServerFloatingIP is allowed when it matches the current api server loadbalancer IP",
			oldTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerFloatingIP: "",
				},
				Status: infrav1.OpenStackClusterStatus{
					APIServerLoadBalancer: &infrav1.LoadBalancer{
						IP: "1.2.3.4",
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerFloatingIP: "1.2.3.4",
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
					APIServerFloatingIP: "",
				},
				Status: infrav1.OpenStackClusterStatus{
					APIServerLoadBalancer: &infrav1.LoadBalancer{
						IP: "1.2.3.4",
					},
				},
			},
			newTemplate: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerFloatingIP: "5.6.7.8",
				},
				Status: infrav1.OpenStackClusterStatus{
					APIServerLoadBalancer: &infrav1.LoadBalancer{
						IP: "1.2.3.4",
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
					CloudName: "foobar",
					IdentityRef: &infrav1.OpenStackIdentityReference{
						Kind: "Secret",
						Name: "foobar",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "OpenStackCluster.Spec.IdentityRef with faulty spec on create",
			template: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &infrav1.OpenStackIdentityReference{
						Kind: "foobar",
						Name: "foobar",
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
