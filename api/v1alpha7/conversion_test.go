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

package v1alpha7

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
		v1alpha7FuzzerFuncs := []interface{}{
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

			func(identityRef *infrav1.OpenStackIdentityReference, c fuzz.Continue) {
				c.FuzzNoCustom(identityRef)

				// None of the following identityRef fields have ever been set in v1alpha7
				identityRef.Region = ""
			},
		}

		return slices.Concat(v1alpha7FuzzerFuncs, testhelpers.InfraV1FuzzerFuncs())
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

func TestMachineConversionControllerSpecFields(t *testing.T) {
	// This tests that we still do field restoration when the controller modifies ProviderID and InstanceID in the spec

	// Define an initial state which cannot be converted losslessly. We add
	// an IdentityRef with a Kind, which has been removed in v1beta1.
	testMachine := func() *OpenStackMachine {
		return &OpenStackMachine{
			Spec: OpenStackMachineSpec{
				IdentityRef: &OpenStackIdentityReference{
					Kind: "InvalidKind",
					Name: "test-name",
				},
			},
		}
	}

	tests := []struct {
		name                  string
		modifyUp              func(*infrav1.OpenStackMachine)
		testAfter             func(gomega.Gomega, *OpenStackMachine)
		expectIdentityRefDiff bool
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
			expectIdentityRefDiff: true,
		},
		{
			name: "Set ProviderID",
			modifyUp: func(up *infrav1.OpenStackMachine) {
				up.Spec.ProviderID = ptr.To("new-provider-id")
			},
			testAfter: func(g gomega.Gomega, after *OpenStackMachine) {
				g.Expect(after.Spec.ProviderID).To(gomega.Equal(ptr.To("new-provider-id")))
			},
			expectIdentityRefDiff: false,
		},
		{
			name: "Set InstanceID",
			modifyUp: func(up *infrav1.OpenStackMachine) {
				up.Status.InstanceID = ptr.To("new-instance-id")
			},
			testAfter: func(g gomega.Gomega, after *OpenStackMachine) {
				g.Expect(after.Spec.InstanceID).To(gomega.Equal(ptr.To("new-instance-id")))
			},
			expectIdentityRefDiff: false,
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
			expectIdentityRefDiff: true,
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

			g.Expect(after.Spec.IdentityRef).ToNot(gomega.BeNil())
			if tt.expectIdentityRefDiff {
				g.Expect(after.Spec.IdentityRef.Kind).ToNot(gomega.Equal("InvalidKind"))
			} else {
				g.Expect(after.Spec.IdentityRef.Kind).To(gomega.Equal("InvalidKind"))
			}
		})
	}
}

func TestConvert_v1alpha7_OpenStackClusterSpec_To_v1beta1_OpenStackClusterSpec(t *testing.T) {
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
			err := Convert_v1alpha7_OpenStackClusterSpec_To_v1beta1_OpenStackClusterSpec(tt.in.DeepCopy(), out, nil)
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
			err := Convert_v1alpha7_OpenStackClusterTemplateSpec_To_v1beta1_OpenStackClusterTemplateSpec(in, out, nil)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(&out.Template.Spec).To(gomega.Equal(tt.expectedOut), cmp.Diff(&out.Template.Spec, tt.expectedOut))
		})
	}
}

func TestConvert_v1alpha7_OpenStackMachineSpec_To_v1beta1_OpenStackMachineSpec(t *testing.T) {
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
			err := Convert_v1alpha7_OpenStackMachineSpec_To_v1beta1_OpenStackMachineSpec(tt.in.DeepCopy(), out, nil)
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
			err := Convert_v1alpha7_OpenStackMachineTemplateSpec_To_v1beta1_OpenStackMachineTemplateSpec(in, out, nil)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(&out.Template.Spec).To(gomega.Equal(tt.expectedOut), cmp.Diff(&out.Template.Spec, tt.expectedOut))
		})
	}
}

func Test_FuzzRestorers(t *testing.T) {
	/* Cluster */
	testhelpers.FuzzRestorer(t, "restorev1alpha7ClusterSpec", restorev1alpha7ClusterSpec)
	testhelpers.FuzzRestorer(t, "restorev1beta1ClusterSpec", restorev1beta1ClusterSpec)
	testhelpers.FuzzRestorer(t, "restorev1alpha7ClusterStatus", restorev1alpha7ClusterStatus)
	testhelpers.FuzzRestorer(t, "restorev1beta1ClusterStatus", restorev1beta1ClusterStatus)
	testhelpers.FuzzRestorer(t, "restorev1alpha7Bastion", restorev1alpha7Bastion)
	testhelpers.FuzzRestorer(t, "restorev1beta1Bastion", restorev1beta1Bastion)
	testhelpers.FuzzRestorer(t, "restorev1beta1BastionStatus", restorev1beta1BastionStatus)

	/* ClusterTemplate */
	testhelpers.FuzzRestorer(t, "restorev1alpha7ClusterTemplateSpec", restorev1alpha7ClusterTemplateSpec)
	testhelpers.FuzzRestorer(t, "restorev1alpha7ClusterTemplateSpec", restorev1alpha7ClusterTemplateSpec)

	/* Machine */
	testhelpers.FuzzRestorer(t, "restorev1alpha7MachineSpec", restorev1alpha7MachineSpec)
	testhelpers.FuzzRestorer(t, "restorev1beta1MachineSpec", restorev1beta1MachineSpec)

	/* MachineTemplate */
	testhelpers.FuzzRestorer(t, "restorev1alpha7MachineTemplateSpec", restorev1alpha7MachineTemplateSpec)

	/* Types */
	testhelpers.FuzzRestorer(t, "restorev1alpha7SecurityGroupFilter", restorev1alpha7SecurityGroupFilter)
	testhelpers.FuzzRestorer(t, "restorev1alpha7SecurityGroup", restorev1alpha7SecurityGroup)
	testhelpers.FuzzRestorer(t, "restorev1beta1SecurityGroupParam", restorev1beta1SecurityGroupParam)
	testhelpers.FuzzRestorer(t, "restorev1alpha7NetworkFilter", restorev1alpha7NetworkFilter)
	testhelpers.FuzzRestorer(t, "restorev1beta1NetworkParam", restorev1beta1NetworkParam)
	testhelpers.FuzzRestorer(t, "restorev1alpha7SubnetFilter", restorev1alpha7SubnetFilter)
	testhelpers.FuzzRestorer(t, "restorev1beta1SubnetParam", restorev1beta1SubnetParam)
	testhelpers.FuzzRestorer(t, "restorev1alpha7RouterFilter", restorev1alpha7RouterFilter)
	testhelpers.FuzzRestorer(t, "restorev1beta1RouterParam", restorev1beta1RouterParam)
	testhelpers.FuzzRestorer(t, "restorev1alpha7Port", restorev1alpha7Port)
	testhelpers.FuzzRestorer(t, "restorev1beta1Port", restorev1beta1Port)
	testhelpers.FuzzRestorer(t, "restorev1beta1APIServerLoadBalancer", restorev1beta1APIServerLoadBalancer)
	testhelpers.FuzzRestorer(t, "restorev1beta1BlockDeviceVolume", restorev1beta1BlockDeviceVolume)
}
