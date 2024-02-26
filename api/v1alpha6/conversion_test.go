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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	fuzz "github.com/google/gofuzz"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/utils/pointer"
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

	filterInvalidTags := func(tags []infrav1.NeutronTag) []infrav1.NeutronTag {
		var ret []infrav1.NeutronTag
		for i := range tags {
			s := string(tags[i])
			if len(s) > 0 && !strings.Contains(s, ",") {
				ret = append(ret, tags[i])
			}
		}
		return ret
	}

	fuzzerFuncs := func(_ runtimeserializer.CodecFactory) []interface{} {
		return []interface{}{
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

			func(spec *infrav1.OpenStackClusterSpec, c fuzz.Continue) {
				c.FuzzNoCustom(spec)

				// The fuzzer only seems to generate Subnets of
				// length 1, but we need to also test length 2.
				// Ensure it is occasionally generated.
				if len(spec.Subnets) == 1 && c.RandBool() {
					subnet := infrav1.SubnetFilter{}
					c.FuzzNoCustom(&subnet)
					spec.Subnets = append(spec.Subnets, subnet)
				}
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

			func(spec *infrav1.SubnetSpec, c fuzz.Continue) {
				c.FuzzNoCustom(spec)

				// CIDR is required and API validates that it's present, so
				// we force it to always be set.
				for spec.CIDR == "" {
					spec.CIDR = c.RandString()
				}
			},

			func(pool *infrav1.AllocationPool, c fuzz.Continue) {
				c.FuzzNoCustom(pool)

				// Start and End are required properties, let's make sure both are set
				for pool.Start == "" {
					pool.Start = c.RandString()
				}

				for pool.End == "" {
					pool.End = c.RandString()
				}
			},

			// v1beta1 filter tags cannot contain commas and can't be empty.

			func(filter *infrav1.SubnetFilter, c fuzz.Continue) {
				c.FuzzNoCustom(filter)

				filter.Tags = filterInvalidTags(filter.Tags)
				filter.TagsAny = filterInvalidTags(filter.TagsAny)
				filter.NotTags = filterInvalidTags(filter.NotTags)
				filter.NotTagsAny = filterInvalidTags(filter.NotTagsAny)
			},

			func(filter *infrav1.NetworkFilter, c fuzz.Continue) {
				c.FuzzNoCustom(filter)

				filter.Tags = filterInvalidTags(filter.Tags)
				filter.TagsAny = filterInvalidTags(filter.TagsAny)
				filter.NotTags = filterInvalidTags(filter.NotTags)
				filter.NotTagsAny = filterInvalidTags(filter.NotTagsAny)
			},

			func(filter *infrav1.RouterFilter, c fuzz.Continue) {
				c.FuzzNoCustom(filter)

				filter.Tags = filterInvalidTags(filter.Tags)
				filter.TagsAny = filterInvalidTags(filter.TagsAny)
				filter.NotTags = filterInvalidTags(filter.NotTags)
				filter.NotTagsAny = filterInvalidTags(filter.NotTagsAny)
			},

			func(filter *infrav1.SecurityGroupFilter, c fuzz.Continue) {
				c.FuzzNoCustom(filter)

				filter.Tags = filterInvalidTags(filter.Tags)
				filter.TagsAny = filterInvalidTags(filter.TagsAny)
				filter.NotTags = filterInvalidTags(filter.NotTags)
				filter.NotTagsAny = filterInvalidTags(filter.NotTagsAny)
			},
		}
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	g := gomega.NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(AddToScheme(scheme)).To(gomega.Succeed())
	g.Expect(infrav1.AddToScheme(scheme)).To(gomega.Succeed())

	// Variables used in the tests
	uuids := []string{"abc123", "123abc"}
	securityGroupsUuids := []infrav1.SecurityGroupFilter{
		{ID: uuids[0]},
		{ID: uuids[1]},
	}
	securityGroupFilter := []SecurityGroupParam{
		{Name: "one"},
		{UUID: "654cba"},
	}
	securityGroupFilterMerged := []infrav1.SecurityGroupFilter{
		{Name: "one"},
		{ID: "654cba"},
		{ID: uuids[0]},
		{ID: uuids[1]},
	}
	legacyPortProfile := map[string]string{
		"capabilities": "[\"switchdev\"]",
		"trusted":      "true",
	}
	convertedPortProfile := infrav1.BindingProfile{
		OVSHWOffload: pointer.Bool(true),
		TrustedVF:    pointer.Bool(true),
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
				Profile:        &convertedPortProfile,
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
				Profile:        &convertedPortProfile,
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

	g := gomega.NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(AddToScheme(scheme)).To(gomega.Succeed())
	g.Expect(infrav1.AddToScheme(scheme)).To(gomega.Succeed())

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
		testAfter         func(*OpenStackMachine)
		expectNetworkDiff bool
	}{
		{
			name: "No change",
		},
		{
			name: "Non-ignored change",
			modifyUp: func(up *infrav1.OpenStackMachine) {
				up.Spec.Flavor = "new-flavor"
			},
			testAfter: func(after *OpenStackMachine) {
				g.Expect(after.Spec.Flavor).To(gomega.Equal("new-flavor"))
			},
			expectNetworkDiff: true,
		},
		{
			name: "Set ProviderID",
			modifyUp: func(up *infrav1.OpenStackMachine) {
				up.Spec.ProviderID = pointer.String("new-provider-id")
			},
			testAfter: func(after *OpenStackMachine) {
				g.Expect(after.Spec.ProviderID).To(gomega.Equal(pointer.String("new-provider-id")))
			},
			expectNetworkDiff: false,
		},
		{
			name: "Set InstanceID",
			modifyUp: func(up *infrav1.OpenStackMachine) {
				up.Spec.InstanceID = pointer.String("new-instance-id")
			},
			testAfter: func(after *OpenStackMachine) {
				g.Expect(after.Spec.InstanceID).To(gomega.Equal(pointer.String("new-instance-id")))
			},
			expectNetworkDiff: false,
		},
		{
			name: "Set ProviderID and non-ignored change",
			modifyUp: func(up *infrav1.OpenStackMachine) {
				up.Spec.ProviderID = pointer.String("new-provider-id")
				up.Spec.Flavor = "new-flavor"
			},
			testAfter: func(after *OpenStackMachine) {
				g.Expect(after.Spec.ProviderID).To(gomega.Equal(pointer.String("new-provider-id")))
				g.Expect(after.Spec.Flavor).To(gomega.Equal("new-flavor"))
			},
			expectNetworkDiff: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := testMachine()

			up := infrav1.OpenStackMachine{}
			g.Expect(before.ConvertTo(&up)).To(gomega.Succeed())

			if tt.modifyUp != nil {
				tt.modifyUp(&up)
			}

			after := OpenStackMachine{}
			g.Expect(after.ConvertFrom(&up)).To(gomega.Succeed())

			if tt.testAfter != nil {
				tt.testAfter(&after)
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
