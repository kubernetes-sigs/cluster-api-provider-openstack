/*
Copyright 2021 The Kubernetes Authors.

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

package v1alpha4

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	runtime "k8s.io/apimachinery/pkg/runtime"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha5"
)

func TestConvertTo(t *testing.T) {
	g := gomega.NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(AddToScheme(scheme)).To(gomega.Succeed())
	g.Expect(infrav1.AddToScheme(scheme)).To(gomega.Succeed())

	const subnetID = "986f5848-127f-4357-944e-5dd75472def8"

	tests := []struct {
		name  string
		spoke ctrlconversion.Convertible
		hub   ctrlconversion.Hub
		want  ctrlconversion.Hub
	}{
		{
			name: "FixedIP with SubnetID",
			spoke: &OpenStackMachine{
				Spec: OpenStackMachineSpec{
					Ports: []PortOpts{
						{
							FixedIPs: []FixedIP{
								{SubnetID: subnetID},
							},
						},
					},
				},
			},
			hub: &infrav1.OpenStackMachine{},
			want: &infrav1.OpenStackMachine{
				Spec: infrav1.OpenStackMachineSpec{
					Ports: []infrav1.PortOpts{
						{
							FixedIPs: []infrav1.FixedIP{
								{Subnet: &infrav1.SubnetFilter{ID: subnetID}},
							},
						},
					},
				},
			},
		},
		{
			name: "APIServer LoadBalancer Configuration",
			spoke: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					ManagedAPIServerLoadBalancer:         true,
					APIServerLoadBalancerAdditionalPorts: []int{80, 443},
				},
			},
			hub: &infrav1.OpenStackCluster{},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerLoadBalancer: infrav1.APIServerLoadBalancer{
						Enabled:         true,
						AdditionalPorts: []int{80, 443},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spoke.ConvertTo(tt.hub)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(tt.hub).To(gomega.Equal(tt.want))
		})
	}
}

func TestConvertFrom(t *testing.T) {
	g := gomega.NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(AddToScheme(scheme)).To(gomega.Succeed())
	g.Expect(infrav1.AddToScheme(scheme)).To(gomega.Succeed())

	const subnetID = "986f5848-127f-4357-944e-5dd75472def8"

	tests := []struct {
		name  string
		spoke ctrlconversion.Convertible
		hub   ctrlconversion.Hub
		want  ctrlconversion.Convertible
	}{
		{
			name:  "FixedIP with SubnetFilter.ID",
			spoke: &OpenStackMachine{},
			hub: &infrav1.OpenStackMachine{
				Spec: infrav1.OpenStackMachineSpec{
					Ports: []infrav1.PortOpts{
						{
							FixedIPs: []infrav1.FixedIP{
								{Subnet: &infrav1.SubnetFilter{ID: subnetID}},
							},
						},
					},
				},
			},
			want: &OpenStackMachine{
				Spec: OpenStackMachineSpec{
					Ports: []PortOpts{
						{
							FixedIPs: []FixedIP{
								{SubnetID: subnetID},
							},
						},
					},
				},
			},
		},
		{
			name:  "APIServer LoadBalancer Configuration",
			spoke: &OpenStackCluster{},
			hub: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerLoadBalancer: infrav1.APIServerLoadBalancer{
						Enabled:         true,
						AdditionalPorts: []int{80, 443},
					},
				},
			},
			want: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					ManagedAPIServerLoadBalancer:         true,
					APIServerLoadBalancerAdditionalPorts: []int{80, 443},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spoke.ConvertFrom(tt.hub)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(tt.spoke).To(gomega.Equal(tt.want))
		})
	}
}

func TestFuzzyConversion(t *testing.T) {
	g := gomega.NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(AddToScheme(scheme)).To(gomega.Succeed())
	g.Expect(infrav1.AddToScheme(scheme)).To(gomega.Succeed())

	fuzzerFuncs := func(_ runtimeserializer.CodecFactory) []interface{} {
		return []interface{}{
			// Don't test spoke-hub-spoke conversion of v1alpha4 fields which are not in infrav1
			func(v1alpha3SubnetFilter *SubnetFilter, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha3SubnetFilter)
				v1alpha3SubnetFilter.EnableDHCP = nil
				v1alpha3SubnetFilter.NetworkID = ""
				v1alpha3SubnetFilter.SubnetPoolID = ""
				v1alpha3SubnetFilter.Limit = 0
				v1alpha3SubnetFilter.Marker = ""
				v1alpha3SubnetFilter.SortKey = ""
				v1alpha3SubnetFilter.SortDir = ""

				// TenantID and ProjectID are the same thing, so TenantID is removed in infrav1
				// Test that we restore TenantID from ProjectID
				v1alpha3SubnetFilter.TenantID = v1alpha3SubnetFilter.ProjectID
			},
			func(v1alpha3Filter *Filter, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha3Filter)
				v1alpha3Filter.Status = ""
				v1alpha3Filter.AdminStateUp = nil
				v1alpha3Filter.Shared = nil
				v1alpha3Filter.Marker = ""
				v1alpha3Filter.Limit = 0
				v1alpha3Filter.SortKey = ""
				v1alpha3Filter.SortDir = ""

				// TenantID and ProjectID are the same thing, so TenantID is removed in infrav1
				// Test that we restore TenantID from ProjectID
				v1alpha3Filter.TenantID = v1alpha3Filter.ProjectID
			},
			func(v1alpha4RootVolume *RootVolume, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha4RootVolume)

				// In v1alpha5 only DeviceType="disk" and SourceType="image" are supported
				v1alpha4RootVolume.DeviceType = "disk"
				v1alpha4RootVolume.SourceType = "image"
			},
			func(v1alpha4MachineSpec *OpenStackMachineSpec, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha4MachineSpec)

				if v1alpha4MachineSpec.RootVolume != nil {
					// OpenStackMachineSpec.Image is ignored in v1alpha4 if RootVolume is set
					v1alpha4MachineSpec.Image = ""
				}
			},
			func(v1alpha4Instance *Instance, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha4Instance)

				if v1alpha4Instance.RootVolume != nil {
					// OpenStackInstance.Image is ignored in v1alpha4 if RootVolume is set
					v1alpha4Instance.Image = ""
				}
			},

			// Don't test hub-spoke-hub conversion of infrav1 fields which are not in v1alpha4
			func(v1alpha5PortOpts *infrav1.PortOpts, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha5PortOpts)

				// v1alpha4 PortOpts has only NetworkID, so only Network.ID filter can be translated
				if v1alpha5PortOpts.Network != nil {
					v1alpha5PortOpts.Network = &infrav1.NetworkFilter{ID: v1alpha5PortOpts.Network.ID}

					// We have no way to differentiate between a nil NetworkFilter and an
					// empty NetworkFilter after conversion because they both translate into an
					// empty string in v1alpha4
					if *v1alpha5PortOpts.Network == (infrav1.NetworkFilter{}) {
						v1alpha5PortOpts.Network = nil
					}
				}
			},
			func(v1alpha5FixedIP *infrav1.FixedIP, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha5FixedIP)

				// v1alpha4 only supports subnet specified by ID
				if v1alpha5FixedIP.Subnet != nil {
					v1alpha5FixedIP.Subnet = &infrav1.SubnetFilter{ID: v1alpha5FixedIP.Subnet.ID}

					// We have no way to differentiate between a nil SubnetFilter and an
					// empty SubnetFilter after conversion because they both translate into an
					// empty string in v1alpha4
					if *v1alpha5FixedIP.Subnet == (infrav1.SubnetFilter{}) {
						v1alpha5FixedIP.Subnet = nil
					}
				}
			},
			func(v1alpha5ClusterStatus *infrav1.OpenStackClusterStatus, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha5ClusterStatus)

				if v1alpha5ClusterStatus.Bastion != nil {
					v1alpha5ClusterStatus.Bastion.ImageUUID = ""
				}
			},
			func(v1alpha5MachineSpec *infrav1.OpenStackMachineSpec, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha5MachineSpec)

				// In v1alpha4 boot from volume only supports
				// image by UUID, and boot from local only
				// suppots image by name
				if v1alpha5MachineSpec.RootVolume != nil && v1alpha5MachineSpec.RootVolume.Size > 0 {
					v1alpha5MachineSpec.Image = ""
				} else {
					v1alpha5MachineSpec.ImageUUID = ""
				}
			},
			func(v1alpha5Instance *infrav1.Instance, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha5Instance)

				// In v1alpha4 boot from volume only supports
				// image by UUID, and boot from local only
				// suppots image by name
				if v1alpha5Instance.RootVolume != nil && v1alpha5Instance.RootVolume.Size > 0 {
					v1alpha5Instance.Image = ""
				} else {
					v1alpha5Instance.ImageUUID = ""
				}
			},
			func(v1alpha5RootVolume *infrav1.RootVolume, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha5RootVolume)

				v1alpha5RootVolume.VolumeType = ""
				v1alpha5RootVolume.AvailabilityZone = ""
			},
		}
	}

	t.Run("for OpenStackCluster", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &infrav1.OpenStackCluster{},
		Spoke:       &OpenStackCluster{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzerFuncs},
	}))

	t.Run("for OpenStackMachine", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &infrav1.OpenStackMachine{},
		Spoke:       &OpenStackMachine{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzerFuncs},
	}))

	t.Run("for OpenStackMachineTemplate", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &infrav1.OpenStackMachineTemplate{},
		Spoke:       &OpenStackMachineTemplate{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzerFuncs},
	}))
}
