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

	fuzz "github.com/google/gofuzz"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	runtime "k8s.io/apimachinery/pkg/runtime"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha7"
)

// TestConvertPortOpts checks both conversions to and from the hub version.
// NOTE: Since the conversion from v1alpha7 to v1alpha6 can be done in multiple ways,
// you need to be careful with what you expect here! The conversion function prefers the v1alpha6 SecurityGroupFilter
// over SecurityGroups since v1alpha7 SecurityGroups are equivalent to v1alpha6 SecurityGroupFilters.
// So you should not expect to get v1alpha6 SecurityGroups out of the conversion.
func TestConvertPortOpts(t *testing.T) {
	g := gomega.NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(AddToScheme(scheme)).To(gomega.Succeed())
	g.Expect(infrav1.AddToScheme(scheme)).To(gomega.Succeed())

	// Variables used in the tests
	uuids := []string{"abc123", "123abc"}
	spokeSecurityGroupParams := []SecurityGroupParam{
		{
			UUID: uuids[0],
		},
		{
			Name: "securityGroup1",
		},
		{
			Filter: SecurityGroupFilter{Tags: "tag"},
		},
	}
	hubSecurityGroupParams := []infrav1.SecurityGroupParam{
		{
			UUID: uuids[0],
		},
		{
			Name: "securityGroup1",
		},
		{
			Filter: infrav1.SecurityGroupFilter{Tags: "tag"},
		},
	}

	tests := []struct {
		name string
		// spokePortOpts are the PortOpts in the spoke version
		spokePortOpts []PortOpts
		// hubPortOpts are the PortOpts in the hub version
		hubPortOpts []infrav1.PortOpts
	}{
		{
			name:          "Empty PortOpts",
			spokePortOpts: []PortOpts{},
			hubPortOpts:   []infrav1.PortOpts{},
		},
		{
			// SecurityGroupFilters have basically just been renamed
			name: "Spoke SecurityGroupFilter",
			spokePortOpts: []PortOpts{
				{
					SecurityGroupFilters: spokeSecurityGroupParams,
				},
			},
			hubPortOpts: []infrav1.PortOpts{
				{
					SecurityGroups: &hubSecurityGroupParams,
				},
			},
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
			convertedSpoke := OpenStackMachineTemplate{}

			err := spokeMachineTemplate.ConvertTo(&convertedHub)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(convertedHub).To(gomega.Equal(hubMachineTemplate))

			err = convertedSpoke.ConvertFrom(&hubMachineTemplate)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			// Comparing spec only here since the conversion will also add annotations that we don't care about for the test
			g.Expect(convertedSpoke.Spec).To(gomega.Equal(spokeMachineTemplate.Spec))
		})
	}
}

// TestPortOptsConvertTo checks conversion TO the hub version.
func TestPortOptsConvertTo(t *testing.T) {
	g := gomega.NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(AddToScheme(scheme)).To(gomega.Succeed())
	g.Expect(infrav1.AddToScheme(scheme)).To(gomega.Succeed())

	// Variables used in the tests
	uuids := []string{"abc123", "123abc"}
	securityGroupsUuids := []infrav1.SecurityGroupParam{
		{UUID: uuids[0]},
		{UUID: uuids[1]},
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
			name: "Spoke SecurityGroups to Hub SecurityGroups",
			spokePortOpts: []PortOpts{{
				SecurityGroups: &uuids,
			}},
			hubPortOpts: []infrav1.PortOpts{{
				SecurityGroups: &securityGroupsUuids,
			}},
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
			g.Expect(convertedHub).To(gomega.Equal(hubMachineTemplate))
		})
	}
}

func TestFuzzyConversion(t *testing.T) {
	t.Run("for OpenStackCluster", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &infrav1.OpenStackCluster{},
		Spoke:       &OpenStackCluster{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{PortOptsFuzzFunc},
	}))

	t.Run("for OpenStackClusterTemplate", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &infrav1.OpenStackClusterTemplate{},
		Spoke:       &OpenStackClusterTemplate{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{PortOptsFuzzFunc},
	}))

	t.Run("for OpenStackMachine", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &infrav1.OpenStackMachine{},
		Spoke:       &OpenStackMachine{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{PortOptsFuzzFunc},
	}))

	t.Run("for OpenStackMachineTemplate", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &infrav1.OpenStackMachineTemplate{},
		Spoke:       &OpenStackMachineTemplate{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{PortOptsFuzzFunc},
	}))
}

func PortOptsFuzzFunc(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		PortOptsFuzzer,
	}
}

func PortOptsFuzzer(in *PortOpts, c fuzz.Continue) {
	c.FuzzNoCustom(in)

	// SecurityGroups has been removed in v1alpha7
	// Conversion is possible (see tests above), but when doing spoke-hub-spoke, we get back
	// SecurityGroupFilters instead of SecurityGroups since they are equivalent to
	// the v1alpha7 SecurityGroups and thus better for conversion.
	in.SecurityGroups = nil
}
