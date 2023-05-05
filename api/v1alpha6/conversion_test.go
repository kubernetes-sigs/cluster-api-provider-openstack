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

package v1alpha6

import (
	"testing"

	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha7"
)

func TestFuzzyConversion(t *testing.T) {
	// The test already ignores the data annotation added on up-conversion.
	// Also ignore the data annotation added on down-conversion.
	ignoreDataAnnotation := func(hub ctrlconversion.Hub) {
		obj := hub.(metav1.Object)
		delete(obj.GetAnnotations(), utilconversion.DataAnnotation)
	}

	t.Run("for OpenStackCluster", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:              &infrav1.OpenStackCluster{},
		Spoke:            &OpenStackCluster{},
		HubAfterMutation: ignoreDataAnnotation,
	}))

	t.Run("for OpenStackClusterTemplate", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:              &infrav1.OpenStackClusterTemplate{},
		Spoke:            &OpenStackClusterTemplate{},
		HubAfterMutation: ignoreDataAnnotation,
	}))

	t.Run("for OpenStackMachine", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:              &infrav1.OpenStackMachine{},
		Spoke:            &OpenStackMachine{},
		HubAfterMutation: ignoreDataAnnotation,
	}))

	t.Run("for OpenStackMachineTemplate", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:              &infrav1.OpenStackMachineTemplate{},
		Spoke:            &OpenStackMachineTemplate{},
		HubAfterMutation: ignoreDataAnnotation,
	}))
}

// Test that mutation of a converted object survives a subsequent conversion.
func TestMutation(t *testing.T) {
	g := gomega.NewWithT(t)

	t.Run("mutation during up-conversion", func(t *testing.T) {
		// Initialise an object with 2 values set
		before := OpenStackCluster{
			Spec: OpenStackClusterSpec{
				CloudName:                  "cloud",
				DisableAPIServerFloatingIP: false,
			},
		}

		// Up-convert the object
		var up infrav1.OpenStackCluster
		g.Expect(before.ConvertTo(&up)).To(gomega.Succeed())

		// Modify one of the values
		up.Spec.DisableAPIServerFloatingIP = true

		// Down-convert the object
		var after OpenStackCluster
		g.Expect(after.ConvertFrom(&up)).To(gomega.Succeed())

		// Ensure that the down-converted values are as expected
		g.Expect(after.Spec.CloudName).To(gomega.Equal("cloud"))
		g.Expect(after.Spec.DisableAPIServerFloatingIP).To(gomega.Equal(true))
	})

	t.Run("mutation during down-conversion", func(t *testing.T) {
		// Initialise an object with 2 values set
		before := infrav1.OpenStackCluster{
			Spec: infrav1.OpenStackClusterSpec{
				CloudName:                  "cloud",
				DisableAPIServerFloatingIP: false,
			},
		}

		// Down-convert the object
		var down OpenStackCluster
		g.Expect(down.ConvertFrom(&before)).To(gomega.Succeed())

		// Modify one of the values
		down.Spec.DisableAPIServerFloatingIP = true

		// Up-convert the object
		var after infrav1.OpenStackCluster
		g.Expect(down.ConvertTo(&after)).To(gomega.Succeed())

		// Ensure that the up-converted values are as expected
		g.Expect(after.Spec.CloudName).To(gomega.Equal("cloud"))
		g.Expect(after.Spec.DisableAPIServerFloatingIP).To(gomega.Equal(true))
	})
}

func TestNetworksToPorts(t *testing.T) {
	g := gomega.NewWithT(t)

	const (
		networkuuid = "55e18f0a-d89a-4d69-b18f-160fb142cb5d"
		subnetuuid  = "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d"
	)

	tests := []struct {
		name              string
		beforeMachineSpec OpenStackMachineSpec
		afterMachineSpec  infrav1.OpenStackMachineSpec
	}{
		{
			name: "Network by UUID, no subnets",
			beforeMachineSpec: OpenStackMachineSpec{
				Networks: []NetworkParam{
					{
						UUID: networkuuid,
					},
				},
			},
			afterMachineSpec: infrav1.OpenStackMachineSpec{
				Ports: []infrav1.PortOpts{
					{
						Network: &infrav1.NetworkFilter{
							ID: networkuuid,
						},
					},
				},
			},
		},
		{
			name: "Network by filter, no subnets",
			beforeMachineSpec: OpenStackMachineSpec{
				Networks: []NetworkParam{
					{
						Filter: NetworkFilter{
							Name:        "network-name",
							Description: "network-description",
							ProjectID:   "project-id",
							Tags:        "tags",
							TagsAny:     "tags-any",
							NotTags:     "not-tags",
							NotTagsAny:  "not-tags-any",
						},
					},
				},
			},
			afterMachineSpec: infrav1.OpenStackMachineSpec{
				Ports: []infrav1.PortOpts{
					{
						Network: &infrav1.NetworkFilter{
							Name:        "network-name",
							Description: "network-description",
							ProjectID:   "project-id",
							Tags:        "tags",
							TagsAny:     "tags-any",
							NotTags:     "not-tags",
							NotTagsAny:  "not-tags-any",
						},
					},
				},
			},
		},
		{
			name: "Subnet by UUID",
			beforeMachineSpec: OpenStackMachineSpec{
				Networks: []NetworkParam{
					{
						UUID: networkuuid,
						Subnets: []SubnetParam{
							{
								UUID: subnetuuid,
							},
						},
					},
				},
			},
			afterMachineSpec: infrav1.OpenStackMachineSpec{
				Ports: []infrav1.PortOpts{
					{
						Network: &infrav1.NetworkFilter{
							ID: networkuuid,
						},
						FixedIPs: []infrav1.FixedIP{
							{
								Subnet: &infrav1.SubnetFilter{
									ID: subnetuuid,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Subnet by filter",
			beforeMachineSpec: OpenStackMachineSpec{
				Networks: []NetworkParam{
					{
						UUID: networkuuid,
						Subnets: []SubnetParam{
							{
								Filter: SubnetFilter{
									Name:            "subnet-name",
									Description:     "subnet-description",
									ProjectID:       "project-id",
									IPVersion:       6,
									GatewayIP:       "x.x.x.x",
									CIDR:            "y.y.y.y",
									IPv6AddressMode: "address-mode",
									IPv6RAMode:      "ra-mode",
									Tags:            "tags",
									TagsAny:         "tags-any",
									NotTags:         "not-tags",
									NotTagsAny:      "not-tags-any",
								},
							},
						},
					},
				},
			},
			afterMachineSpec: infrav1.OpenStackMachineSpec{
				Ports: []infrav1.PortOpts{
					{
						Network: &infrav1.NetworkFilter{
							ID: networkuuid,
						},
						FixedIPs: []infrav1.FixedIP{
							{
								Subnet: &infrav1.SubnetFilter{
									Name:            "subnet-name",
									Description:     "subnet-description",
									ProjectID:       "project-id",
									IPVersion:       6,
									GatewayIP:       "x.x.x.x",
									CIDR:            "y.y.y.y",
									IPv6AddressMode: "address-mode",
									IPv6RAMode:      "ra-mode",
									Tags:            "tags",
									TagsAny:         "tags-any",
									NotTags:         "not-tags",
									NotTagsAny:      "not-tags-any",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Multiple subnets",
			beforeMachineSpec: OpenStackMachineSpec{
				Networks: []NetworkParam{
					{
						UUID: networkuuid,
						Subnets: []SubnetParam{
							{
								UUID: subnetuuid,
							},
							{
								Filter: SubnetFilter{
									Name:            "subnet-name",
									Description:     "subnet-description",
									ProjectID:       "project-id",
									IPVersion:       6,
									GatewayIP:       "x.x.x.x",
									CIDR:            "y.y.y.y",
									IPv6AddressMode: "address-mode",
									IPv6RAMode:      "ra-mode",
									Tags:            "tags",
									TagsAny:         "tags-any",
									NotTags:         "not-tags",
									NotTagsAny:      "not-tags-any",
								},
							},
						},
					},
				},
			},
			afterMachineSpec: infrav1.OpenStackMachineSpec{
				Ports: []infrav1.PortOpts{
					{
						Network: &infrav1.NetworkFilter{
							ID: networkuuid,
						},
						FixedIPs: []infrav1.FixedIP{
							{
								Subnet: &infrav1.SubnetFilter{
									ID: subnetuuid,
								},
							},
						},
					},
					{
						Network: &infrav1.NetworkFilter{
							ID: networkuuid,
						},
						FixedIPs: []infrav1.FixedIP{
							{
								Subnet: &infrav1.SubnetFilter{
									Name:            "subnet-name",
									Description:     "subnet-description",
									ProjectID:       "project-id",
									IPVersion:       6,
									GatewayIP:       "x.x.x.x",
									CIDR:            "y.y.y.y",
									IPv6AddressMode: "address-mode",
									IPv6RAMode:      "ra-mode",
									Tags:            "tags",
									TagsAny:         "tags-any",
									NotTags:         "not-tags",
									NotTagsAny:      "not-tags-any",
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := &OpenStackMachine{
				Spec: tt.beforeMachineSpec,
			}
			after := infrav1.OpenStackMachine{}

			g.Expect(before.ConvertTo(&after)).To(gomega.Succeed())
			g.Expect(after.Spec).To(gomega.Equal(tt.afterMachineSpec))
		})
	}
}
