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
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	fuzz "github.com/google/gofuzz"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/utils/ptr"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	testhelpers "sigs.k8s.io/cluster-api-provider-openstack/test/helpers"
)

// Setting this to false to avoid running tests in parallel. Only for use in development.
const parallel = true

func runParallel(f func(t *testing.T)) func(t *testing.T) {
	if parallel {
		return func(t *testing.T) {
			t.Helper()
			t.Parallel()
			f(t)
		}
	}
	return f
}

func TestFuzzyConversion(t *testing.T) {
	// The test already ignores the data annotation added on up-conversion.
	// Also ignore the data annotation added on down-conversion.
	ignoreDataAnnotation := func(hub conversion.Hub) {
		obj := hub.(metav1.Object)
		delete(obj.GetAnnotations(), utilconversion.DataAnnotation)
	}

	fuzzerFuncs := func(_ runtimeserializer.CodecFactory) []interface{} {
		v1alpha6FuzzerFuncs := []interface{}{
			func(instance *Instance, c fuzz.Continue) {
				c.FuzzNoCustom(instance)

				// None of the following status fields have ever been set in v1alpha6
				instance.Trunk = false
				instance.FailureDomain = ""
				instance.SecurityGroups = nil
				instance.Networks = nil
				instance.Subnet = ""
				instance.Tags = nil
				instance.Image = ""
				instance.ImageUUID = ""
				instance.Flavor = ""
				instance.UserData = ""
				instance.Metadata = nil
				instance.ConfigDrive = nil
				instance.RootVolume = nil
				instance.ServerGroupID = ""
			},

			func(status *OpenStackClusterStatus, c fuzz.Continue) {
				c.FuzzNoCustom(status)

				// None of the following status fields have ever been set in v1alpha6
				if status.ExternalNetwork != nil {
					status.ExternalNetwork.Subnet = nil
					status.ExternalNetwork.PortOpts = nil
					status.ExternalNetwork.Router = nil
					status.ExternalNetwork.APIServerLoadBalancer = nil
				}
			},

			func(identityRef *infrav1.OpenStackIdentityReference, c fuzz.Continue) {
				c.FuzzNoCustom(identityRef)

				// None of the following identityRef fields have ever been set in v1alpha6
				identityRef.Region = ""
			},

			func(spec *OpenStackMachineSpec, c fuzz.Continue) {
				c.FuzzNoCustom(spec)

				// RandString() generates strings up to 20
				// characters long. To exercise truncation of
				// long server metadata keys and values we need
				// the possibility of strings > 255 chars.
				genLongString := func() string {
					var ret string
					for len(ret) < 255 {
						ret += c.RandString()
					}
					return ret
				}

				// Existing server metadata keys will be short. Add a random number of long ones.
				for c.RandBool() {
					if spec.ServerMetadata == nil {
						spec.ServerMetadata = map[string]string{}
					}
					spec.ServerMetadata[genLongString()] = c.RandString()
				}

				// Randomly make some server metadata values long.
				for k := range spec.ServerMetadata {
					if c.RandBool() {
						spec.ServerMetadata[k] = genLongString()
					}
				}
			},
		}

		return slices.Concat(v1alpha6FuzzerFuncs, testhelpers.InfraV1FuzzerFuncs())
	}

	t.Run("for OpenStackCluster", runParallel(utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:              &infrav1.OpenStackCluster{},
		Spoke:            &OpenStackCluster{},
		HubAfterMutation: ignoreDataAnnotation,
		FuzzerFuncs:      []fuzzer.FuzzerFuncs{fuzzerFuncs},
	})))

	t.Run("for OpenStackCluster with mutate", runParallel(testhelpers.FuzzMutateTestFunc(testhelpers.FuzzMutateTestFuncInput{
		FuzzTestFuncInput: utilconversion.FuzzTestFuncInput{
			Hub:              &infrav1.OpenStackCluster{},
			Spoke:            &OpenStackCluster{},
			HubAfterMutation: ignoreDataAnnotation,
			FuzzerFuncs:      []fuzzer.FuzzerFuncs{fuzzerFuncs},
		},
		MutateFuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzerFuncs},
	})))

	t.Run("for OpenStackClusterTemplate", runParallel(utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:              &infrav1.OpenStackClusterTemplate{},
		Spoke:            &OpenStackClusterTemplate{},
		HubAfterMutation: ignoreDataAnnotation,
		FuzzerFuncs:      []fuzzer.FuzzerFuncs{fuzzerFuncs},
	})))

	t.Run("for OpenStackClusterTemplate with mutate", runParallel(testhelpers.FuzzMutateTestFunc(testhelpers.FuzzMutateTestFuncInput{
		FuzzTestFuncInput: utilconversion.FuzzTestFuncInput{
			Hub:              &infrav1.OpenStackClusterTemplate{},
			Spoke:            &OpenStackClusterTemplate{},
			HubAfterMutation: ignoreDataAnnotation,
			FuzzerFuncs:      []fuzzer.FuzzerFuncs{fuzzerFuncs},
		},
		MutateFuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzerFuncs},
	})))

	t.Run("for OpenStackMachine", runParallel(utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:              &infrav1.OpenStackMachine{},
		Spoke:            &OpenStackMachine{},
		HubAfterMutation: ignoreDataAnnotation,
		FuzzerFuncs:      []fuzzer.FuzzerFuncs{fuzzerFuncs},
	})))

	t.Run("for OpenStackMachine with mutate", runParallel(testhelpers.FuzzMutateTestFunc(testhelpers.FuzzMutateTestFuncInput{
		FuzzTestFuncInput: utilconversion.FuzzTestFuncInput{
			Hub:              &infrav1.OpenStackMachine{},
			Spoke:            &OpenStackMachine{},
			HubAfterMutation: ignoreDataAnnotation,
			FuzzerFuncs:      []fuzzer.FuzzerFuncs{fuzzerFuncs},
		},
		MutateFuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzerFuncs},
	})))

	t.Run("for OpenStackMachineTemplate", runParallel(utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:              &infrav1.OpenStackMachineTemplate{},
		Spoke:            &OpenStackMachineTemplate{},
		HubAfterMutation: ignoreDataAnnotation,
		FuzzerFuncs:      []fuzzer.FuzzerFuncs{fuzzerFuncs},
	})))

	t.Run("for OpenStackMachineTemplate with mutate", runParallel(testhelpers.FuzzMutateTestFunc(testhelpers.FuzzMutateTestFuncInput{
		FuzzTestFuncInput: utilconversion.FuzzTestFuncInput{
			Hub:              &infrav1.OpenStackMachineTemplate{},
			Spoke:            &OpenStackMachineTemplate{},
			HubAfterMutation: ignoreDataAnnotation,
			FuzzerFuncs:      []fuzzer.FuzzerFuncs{fuzzerFuncs},
		},
		MutateFuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzerFuncs},
	})))
}

func TestNetworksToPorts(t *testing.T) {
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
						Network: &infrav1.NetworkParam{
							ID: ptr.To(networkuuid),
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
						Network: &infrav1.NetworkParam{
							Filter: &infrav1.NetworkFilter{
								Name:        "network-name",
								Description: "network-description",
								ProjectID:   "project-id",
								FilterByNeutronTags: infrav1.FilterByNeutronTags{
									Tags:       []infrav1.NeutronTag{"tags"},
									TagsAny:    []infrav1.NeutronTag{"tags-any"},
									NotTags:    []infrav1.NeutronTag{"not-tags"},
									NotTagsAny: []infrav1.NeutronTag{"not-tags-any"},
								},
							},
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
						Network: &infrav1.NetworkParam{
							ID: ptr.To(networkuuid),
						},
						FixedIPs: []infrav1.FixedIP{
							{
								Subnet: &infrav1.SubnetParam{
									ID: ptr.To(subnetuuid),
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
						Network: &infrav1.NetworkParam{
							ID: ptr.To(networkuuid),
						},
						FixedIPs: []infrav1.FixedIP{
							{
								Subnet: &infrav1.SubnetParam{
									Filter: &infrav1.SubnetFilter{
										Name:            "subnet-name",
										Description:     "subnet-description",
										ProjectID:       "project-id",
										IPVersion:       6,
										GatewayIP:       "x.x.x.x",
										CIDR:            "y.y.y.y",
										IPv6AddressMode: "address-mode",
										IPv6RAMode:      "ra-mode",
										FilterByNeutronTags: infrav1.FilterByNeutronTags{
											Tags:       []infrav1.NeutronTag{"tags"},
											TagsAny:    []infrav1.NeutronTag{"tags-any"},
											NotTags:    []infrav1.NeutronTag{"not-tags"},
											NotTagsAny: []infrav1.NeutronTag{"not-tags-any"},
										},
									},
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
						Network: &infrav1.NetworkParam{
							ID: ptr.To(networkuuid),
						},
						FixedIPs: []infrav1.FixedIP{
							{
								Subnet: &infrav1.SubnetParam{
									ID: ptr.To(subnetuuid),
								},
							},
						},
					},
					{
						Network: &infrav1.NetworkParam{
							ID: ptr.To(networkuuid),
						},
						FixedIPs: []infrav1.FixedIP{
							{
								Subnet: &infrav1.SubnetParam{
									Filter: &infrav1.SubnetFilter{
										Name:            "subnet-name",
										Description:     "subnet-description",
										ProjectID:       "project-id",
										IPVersion:       6,
										GatewayIP:       "x.x.x.x",
										CIDR:            "y.y.y.y",
										IPv6AddressMode: "address-mode",
										IPv6RAMode:      "ra-mode",
										FilterByNeutronTags: infrav1.FilterByNeutronTags{
											Tags:       []infrav1.NeutronTag{"tags"},
											TagsAny:    []infrav1.NeutronTag{"tags-any"},
											NotTags:    []infrav1.NeutronTag{"not-tags"},
											NotTagsAny: []infrav1.NeutronTag{"not-tags-any"},
										},
									},
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
			g := gomega.NewWithT(t)

			before := &OpenStackMachine{
				Spec: tt.beforeMachineSpec,
			}
			after := infrav1.OpenStackMachine{}

			g.Expect(before.ConvertTo(&after)).To(gomega.Succeed())
			g.Expect(after.Spec).To(gomega.Equal(tt.afterMachineSpec), cmp.Diff(after.Spec, tt.afterMachineSpec))
		})
	}
}

// TestPortOptsConvertTo checks conversion TO the hub version.
// This is useful to ensure that the SecurityGroups are properly
// converted to SecurityGroupFilters, and merged with any existing
// SecurityGroupFilters.
func TestPortOptsConvertTo(t *testing.T) {
	// Variables used in the tests
	uuids := []string{"abc123", "123abc"}
	securityGroupsUuids := []infrav1.SecurityGroupParam{
		{ID: &uuids[0]},
		{ID: &uuids[1]},
	}
	securityGroupFilter := []SecurityGroupParam{
		{Name: "one"},
		{UUID: "654cba"},
	}
	securityGroupFilterMerged := []infrav1.SecurityGroupParam{
		{Filter: &infrav1.SecurityGroupFilter{Name: "one"}},
		{ID: ptr.To("654cba")},
		{ID: &uuids[0]},
		{ID: &uuids[1]},
	}
	legacyPortProfile := map[string]string{
		"capabilities": "[\"switchdev\"]",
		"trusted":      "true",
	}
	convertedPortProfile := infrav1.BindingProfile{
		OVSHWOffload: ptr.To(true),
		TrustedVF:    ptr.To(true),
	}

	tests := []struct {
		name string
		// spokePortOpts are the PortOpts in the spoke version
		spokePortOpts []PortOpts
		// hubPortOpts are the PortOpts in the hub version that should be expected after conversion
		hubPortOpts []infrav1.PortOpts
	}{
		{
			// The list of security group UUIDs should be translated to proper SecurityGroupParams
			name: "SecurityGroups to SecurityGroupFilters",
			spokePortOpts: []PortOpts{{
				Profile:        legacyPortProfile,
				SecurityGroups: uuids,
			}},
			hubPortOpts: []infrav1.PortOpts{{
				ResolvedPortSpecFields: infrav1.ResolvedPortSpecFields{
					Profile: &convertedPortProfile,
				},
				SecurityGroups: securityGroupsUuids,
			}},
		},
		{
			name: "Merge SecurityGroups and SecurityGroupFilters",
			spokePortOpts: []PortOpts{{
				Profile:              legacyPortProfile,
				SecurityGroups:       uuids,
				SecurityGroupFilters: securityGroupFilter,
			}},
			hubPortOpts: []infrav1.PortOpts{{
				ResolvedPortSpecFields: infrav1.ResolvedPortSpecFields{
					Profile: &convertedPortProfile,
				},
				SecurityGroups: securityGroupFilterMerged,
			}},
		},
		{
			name:          "Empty port",
			spokePortOpts: []PortOpts{{}},
			hubPortOpts:   []infrav1.PortOpts{{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			scheme := runtime.NewScheme()
			g.Expect(AddToScheme(scheme)).To(gomega.Succeed())
			g.Expect(infrav1.AddToScheme(scheme)).To(gomega.Succeed())

			// The spoke machine template with added PortOpts
			spokeMachineTemplate := OpenStackMachineTemplate{
				Spec: OpenStackMachineTemplateSpec{
					Template: OpenStackMachineTemplateResource{
						Spec: OpenStackMachineSpec{
							Ports: tt.spokePortOpts,
						},
					},
				},
			}
			// The hub machine template with added PortOpts
			hubMachineTemplate := infrav1.OpenStackMachineTemplate{
				Spec: infrav1.OpenStackMachineTemplateSpec{
					Template: infrav1.OpenStackMachineTemplateResource{
						Spec: infrav1.OpenStackMachineSpec{
							Ports: tt.hubPortOpts,
						},
					},
				},
			}
			convertedHub := infrav1.OpenStackMachineTemplate{}

			err := spokeMachineTemplate.ConvertTo(&convertedHub)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			// Comparing spec only here since the conversion will also add annotations that we don't care about for the test
			g.Expect(convertedHub.Spec).To(gomega.Equal(hubMachineTemplate.Spec), cmp.Diff(convertedHub.Spec, hubMachineTemplate.Spec))
		})
	}
}

func TestMachineConversionControllerSpecFields(t *testing.T) {
	// This tests that we still do field restoration when the controller modifies ProviderID and InstanceID in the spec

	// This test machine contains a network definition. If we restore it on
	// down-conversion it will still have a network definition. If we don't,
	// the network definition will have become a port definition.
	testMachine := func() *OpenStackMachine {
		return &OpenStackMachine{
			Spec: OpenStackMachineSpec{
				Networks: []NetworkParam{
					{
						UUID: "network-uuid",
					},
				},
			},
		}
	}

	tests := []struct {
		name              string
		modifyUp          func(*infrav1.OpenStackMachine)
		testAfter         func(gomega.Gomega, *OpenStackMachine)
		expectNetworkDiff bool
	}{
		{
			name: "No change",
		},
		{
			name: "Non-ignored change",
			modifyUp: func(up *infrav1.OpenStackMachine) {
				up.Spec.Flavor = ptr.To("new-flavor")
			},
			testAfter: func(g gomega.Gomega, after *OpenStackMachine) {
				g.Expect(after.Spec.Flavor).To(gomega.Equal(ptr.To("new-flavor")))
			},
			expectNetworkDiff: true,
		},
		{
			name: "Set ProviderID",
			modifyUp: func(up *infrav1.OpenStackMachine) {
				up.Spec.ProviderID = ptr.To("new-provider-id")
			},
			testAfter: func(g gomega.Gomega, after *OpenStackMachine) {
				g.Expect(after.Spec.ProviderID).To(gomega.Equal(ptr.To("new-provider-id")))
			},
			expectNetworkDiff: false,
		},
		{
			name: "Set InstanceID",
			modifyUp: func(up *infrav1.OpenStackMachine) {
				up.Status.InstanceID = ptr.To("new-instance-id")
			},
			testAfter: func(g gomega.Gomega, after *OpenStackMachine) {
				g.Expect(after.Spec.InstanceID).To(gomega.Equal(ptr.To("new-instance-id")))
			},
			expectNetworkDiff: false,
		},
		{
			name: "Set ProviderID and non-ignored change",
			modifyUp: func(up *infrav1.OpenStackMachine) {
				up.Spec.ProviderID = ptr.To("new-provider-id")
				up.Spec.Flavor = ptr.To("new-flavor")
			},
			testAfter: func(g gomega.Gomega, after *OpenStackMachine) {
				g.Expect(after.Spec.ProviderID).To(gomega.Equal(ptr.To("new-provider-id")))
				g.Expect(after.Spec.Flavor).To(gomega.Equal(ptr.To("new-flavor")))
			},
			expectNetworkDiff: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			scheme := runtime.NewScheme()
			g.Expect(AddToScheme(scheme)).To(gomega.Succeed())
			g.Expect(infrav1.AddToScheme(scheme)).To(gomega.Succeed())
			before := testMachine()

			up := infrav1.OpenStackMachine{}
			g.Expect(before.ConvertTo(&up)).To(gomega.Succeed())

			if tt.modifyUp != nil {
				tt.modifyUp(&up)
			}

			after := OpenStackMachine{}
			g.Expect(after.ConvertFrom(&up)).To(gomega.Succeed())

			if tt.testAfter != nil {
				tt.testAfter(g, &after)
			}

			if !tt.expectNetworkDiff {
				g.Expect(after.Spec.Networks).To(gomega.HaveLen(1))
				g.Expect(after.Spec.Ports).To(gomega.HaveLen(0))
			} else {
				g.Expect(after.Spec.Networks).To(gomega.HaveLen(0))
				g.Expect(after.Spec.Ports).To(gomega.HaveLen(1))
			}
		})
	}
}

func TestConvert_v1alpha6_OpenStackClusterSpec_To_v1beta1_OpenStackClusterSpec(t *testing.T) {
	tests := []struct {
		name        string
		in          *OpenStackClusterSpec
		expectedOut *infrav1.OpenStackClusterSpec
	}{
		{
			name:        "empty",
			in:          &OpenStackClusterSpec{},
			expectedOut: &infrav1.OpenStackClusterSpec{},
		},
		{
			name: "with managed security groups and not allow all in cluster traffic",
			in: &OpenStackClusterSpec{
				ManagedSecurityGroups:    true,
				AllowAllInClusterTraffic: false,
			},
			expectedOut: &infrav1.OpenStackClusterSpec{
				ManagedSecurityGroups: &infrav1.ManagedSecurityGroups{
					AllNodesSecurityGroupRules: infrav1.LegacyCalicoSecurityGroupRules(),
				},
			},
		},
		{
			name: "with managed security groups and allow all in cluster traffic",
			in: &OpenStackClusterSpec{
				ManagedSecurityGroups:    true,
				AllowAllInClusterTraffic: true,
			},
			expectedOut: &infrav1.OpenStackClusterSpec{
				ManagedSecurityGroups: &infrav1.ManagedSecurityGroups{
					AllowAllInClusterTraffic: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			out := &infrav1.OpenStackClusterSpec{}
			err := Convert_v1alpha6_OpenStackClusterSpec_To_v1beta1_OpenStackClusterSpec(tt.in.DeepCopy(), out, nil)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(out).To(gomega.Equal(tt.expectedOut), cmp.Diff(out, tt.expectedOut))
		})

		t.Run("template_"+tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			in := &OpenStackClusterTemplateSpec{
				Template: OpenStackClusterTemplateResource{
					Spec: *(tt.in.DeepCopy()),
				},
			}
			out := &infrav1.OpenStackClusterTemplateSpec{}
			err := Convert_v1alpha6_OpenStackClusterTemplateSpec_To_v1beta1_OpenStackClusterTemplateSpec(in, out, nil)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(&out.Template.Spec).To(gomega.Equal(tt.expectedOut), cmp.Diff(&out.Template.Spec, tt.expectedOut))
		})
	}
}

func TestConvert_v1alpha6_OpenStackMachineSpec_To_v1beta1_OpenStackMachineSpec(t *testing.T) {
	tests := []struct {
		name        string
		in          *OpenStackMachineSpec
		expectedOut *infrav1.OpenStackMachineSpec
	}{
		{
			name:        "empty",
			in:          &OpenStackMachineSpec{},
			expectedOut: &infrav1.OpenStackMachineSpec{},
		},
		{
			name: "empty port",
			in: &OpenStackMachineSpec{
				Ports: []PortOpts{{}},
			},
			expectedOut: &infrav1.OpenStackMachineSpec{
				Ports: []infrav1.PortOpts{{}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			out := &infrav1.OpenStackMachineSpec{}
			err := Convert_v1alpha6_OpenStackMachineSpec_To_v1beta1_OpenStackMachineSpec(tt.in.DeepCopy(), out, nil)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(out).To(gomega.Equal(tt.expectedOut), cmp.Diff(out, tt.expectedOut))
		})

		t.Run("template_"+tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			in := &OpenStackMachineTemplateSpec{
				Template: OpenStackMachineTemplateResource{
					Spec: *(tt.in.DeepCopy()),
				},
			}
			out := &infrav1.OpenStackMachineTemplateSpec{}
			err := Convert_v1alpha6_OpenStackMachineTemplateSpec_To_v1beta1_OpenStackMachineTemplateSpec(in, out, nil)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(&out.Template.Spec).To(gomega.Equal(tt.expectedOut), cmp.Diff(&out.Template.Spec, tt.expectedOut))
		})
	}
}

func Test_FuzzRestorers(t *testing.T) {
	/* Cluster */
	testhelpers.FuzzRestorer(t, "restorev1alpha6ClusterSpec", restorev1alpha6ClusterSpec)
	testhelpers.FuzzRestorer(t, "restorev1beta1ClusterSpec", restorev1beta1ClusterSpec)
	testhelpers.FuzzRestorer(t, "restorev1alpha6ClusterStatus", restorev1alpha6ClusterStatus)
	testhelpers.FuzzRestorer(t, "restorev1beta1ClusterStatus", restorev1beta1ClusterStatus)
	testhelpers.FuzzRestorer(t, "restorev1beta1Bastion", restorev1beta1Bastion)
	testhelpers.FuzzRestorer(t, "restorev1beta1BastionStatus", restorev1beta1BastionStatus)

	/* ClusterTemplate */
	testhelpers.FuzzRestorer(t, "restorev1beta1ClusterTemplateSpec", restorev1beta1ClusterTemplateSpec)

	/* Machine */
	testhelpers.FuzzRestorer(t, "restorev1alpha6MachineSpec", restorev1alpha6MachineSpec)
	testhelpers.FuzzRestorer(t, "restorev1beta1MachineSpec", restorev1beta1MachineSpec)

	/* MachineTemplate */
	testhelpers.FuzzRestorer(t, "restorev1alpha6MachineTemplateMachineSpec", restorev1alpha6MachineTemplateMachineSpec)

	/* Types */
	testhelpers.FuzzRestorer(t, "restorev1alpha6SecurityGroupFilter", restorev1alpha6SecurityGroupFilter)
	testhelpers.FuzzRestorer(t, "restorev1beta1SecurityGroupParam", restorev1beta1SecurityGroupParam)
	testhelpers.FuzzRestorer(t, "restorev1alpha6NetworkFilter", restorev1alpha6NetworkFilter)
	testhelpers.FuzzRestorer(t, "restorev1beta1NetworkParam", restorev1beta1NetworkParam)
	testhelpers.FuzzRestorer(t, "restorev1alpha6SubnetFilter", restorev1alpha6SubnetFilter)
	testhelpers.FuzzRestorer(t, "restorev1alpha6SubnetParam", restorev1alpha6SubnetParam)
	testhelpers.FuzzRestorer(t, "restorev1beta1SubnetParam", restorev1beta1SubnetParam)
	testhelpers.FuzzRestorer(t, "restorev1alpha6Port", restorev1alpha6Port)
	testhelpers.FuzzRestorer(t, "restorev1beta1BlockDeviceVolume", restorev1beta1BlockDeviceVolume)
	testhelpers.FuzzRestorer(t, "restorev1alpha6SecurityGroup", restorev1alpha6SecurityGroup)
	testhelpers.FuzzRestorer(t, "restorev1beta1APIServerLoadBalancer", restorev1beta1APIServerLoadBalancer)
}
