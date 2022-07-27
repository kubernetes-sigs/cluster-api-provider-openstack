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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha6"
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
		{
			name: "OpenStackClusterTemplate",
			spoke: &OpenStackClusterTemplate{
				Spec: OpenStackClusterTemplateSpec{},
			},
			hub: &infrav1.OpenStackClusterTemplate{},
			want: &infrav1.OpenStackClusterTemplate{
				Spec: infrav1.OpenStackClusterTemplateSpec{},
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
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"cluster.x-k8s.io/conversion-data": "{\"spec\":{\"cloudName\":\"\",\"flavor\":\"\",\"ports\":[{\"fixedIPs\":[{\"subnet\":{\"id\":\"986f5848-127f-4357-944e-5dd75472def8\"}}]}]},\"status\":{\"ready\":false}}",
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
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"cluster.x-k8s.io/conversion-data": "{\"spec\":{\"allowAllInClusterTraffic\":false,\"apiServerLoadBalancer\":{\"additionalPorts\":[80,443],\"enabled\":true},\"cloudName\":\"\",\"controlPlaneEndpoint\":{\"host\":\"\",\"port\":0},\"disableAPIServerFloatingIP\":false,\"managedSecurityGroups\":false,\"network\":{},\"subnet\":{}},\"status\":{\"ready\":false}}",
					},
				},
			},
		},
		{
			name:  "OpenStackClusterTemplate",
			spoke: &OpenStackClusterTemplate{},
			hub: &infrav1.OpenStackClusterTemplate{
				Spec: infrav1.OpenStackClusterTemplateSpec{},
			},
			want: &OpenStackClusterTemplate{
				Spec: OpenStackClusterTemplateSpec{},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"cluster.x-k8s.io/conversion-data": "{\"spec\":{\"template\":{\"spec\":{\"allowAllInClusterTraffic\":false,\"apiServerLoadBalancer\":{},\"cloudName\":\"\",\"controlPlaneEndpoint\":{\"host\":\"\",\"port\":0},\"disableAPIServerFloatingIP\":false,\"managedSecurityGroups\":false,\"network\":{},\"subnet\":{}}}}}",
					},
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
			func(v1alpha4Cluster *OpenStackCluster, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha4Cluster)

				v1alpha4Cluster.ObjectMeta.Annotations = map[string]string{}

				if v1alpha4Cluster.Spec.Bastion != nil {
					v1alpha4Cluster.Spec.Bastion.Instance.Image = ""
				}
			},
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

				// In v1alpha6 only DeviceType="disk" and SourceType="image" are supported
				v1alpha4RootVolume.DeviceType = "disk"
				v1alpha4RootVolume.SourceType = "image"
			},
			func(v1alpha4Machine *OpenStackMachine, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha4Machine)

				v1alpha4Machine.ObjectMeta.Annotations = map[string]string{}

				if v1alpha4Machine.Spec.RootVolume != nil {
					// OpenStackMachineSpec.Image is ignored in v1alpha4 if RootVolume is set
					v1alpha4Machine.Spec.Image = ""
				}
			},
			func(v1alpha4MachineTemplate *OpenStackMachineTemplate, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha4MachineTemplate)

				v1alpha4MachineTemplate.ObjectMeta.Annotations = map[string]string{}

				v1alpha4MachineTemplate.Spec.Template.Spec.Image = ""
			},
			func(v1alpha4Instance *Instance, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha4Instance)

				if v1alpha4Instance.RootVolume != nil {
					// OpenStackInstance.Image is ignored in v1alpha4 if RootVolume is set
					v1alpha4Instance.Image = ""
				}
			},
			func(v1alpha4ClusterTemplate *OpenStackClusterTemplate, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha4ClusterTemplate)

				v1alpha4ClusterTemplate.ObjectMeta.Annotations = map[string]string{}

				if v1alpha4ClusterTemplate.Spec.Template.Spec.Bastion != nil {
					v1alpha4ClusterTemplate.Spec.Template.Spec.Bastion.Instance.Image = ""
				}
			},

			// Don't test hub-spoke-hub conversion of infrav1 fields which are not in v1alpha4
			func(v1alpha6PortOpts *infrav1.PortOpts, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha6PortOpts)

				// v1alpha4 PortOpts has only NetworkID, so only Network.ID filter can be translated
				if v1alpha6PortOpts.Network != nil {
					v1alpha6PortOpts.Network = &infrav1.NetworkFilter{ID: v1alpha6PortOpts.Network.ID}

					// We have no way to differentiate between a nil NetworkFilter and an
					// empty NetworkFilter after conversion because they both translate into an
					// empty string in v1alpha4
					if *v1alpha6PortOpts.Network == (infrav1.NetworkFilter{}) {
						v1alpha6PortOpts.Network = nil
					}
				}
				v1alpha6PortOpts.SecurityGroupFilters = nil
			},
			func(v1alpha6FixedIP *infrav1.FixedIP, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha6FixedIP)

				// v1alpha4 only supports subnet specified by ID
				if v1alpha6FixedIP.Subnet != nil {
					v1alpha6FixedIP.Subnet = &infrav1.SubnetFilter{ID: v1alpha6FixedIP.Subnet.ID}

					// We have no way to differentiate between a nil SubnetFilter and an
					// empty SubnetFilter after conversion because they both translate into an
					// empty string in v1alpha4
					if *v1alpha6FixedIP.Subnet == (infrav1.SubnetFilter{}) {
						v1alpha6FixedIP.Subnet = nil
					}
				}
			},
			func(v1alpha6Cluster *infrav1.OpenStackCluster, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha6Cluster)

				v1alpha6Cluster.ObjectMeta.Annotations = map[string]string{}

				v1alpha6Cluster.Spec.APIServerLoadBalancer.AllowedCIDRs = nil

				if v1alpha6Cluster.Spec.Bastion != nil {
					v1alpha6Cluster.Spec.Bastion.Instance.Image = ""
				}

				if v1alpha6Cluster.Status.Bastion != nil {
					v1alpha6Cluster.Status.Bastion.ImageUUID = ""
					v1alpha6Cluster.Status.Bastion.Image = ""
					v1alpha6Cluster.Status.Bastion.Networks = nil
				}

				if v1alpha6Cluster.Status.Network != nil {
					if v1alpha6Cluster.Status.Network.APIServerLoadBalancer != nil {
						v1alpha6Cluster.Status.Network.APIServerLoadBalancer.AllowedCIDRs = nil
					}
					if v1alpha6Cluster.Status.Network.Router != nil {
						v1alpha6Cluster.Status.Network.Router.IPs = []string{}
					}
				}

				if v1alpha6Cluster.Status.ExternalNetwork != nil {
					if v1alpha6Cluster.Status.ExternalNetwork.APIServerLoadBalancer != nil {
						v1alpha6Cluster.Status.ExternalNetwork.APIServerLoadBalancer.AllowedCIDRs = nil
					}
					if v1alpha6Cluster.Status.ExternalNetwork.Router != nil {
						v1alpha6Cluster.Status.ExternalNetwork.Router.IPs = []string{}
					}
				}
			},
			func(v1alpha6Machine *infrav1.OpenStackMachine, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha6Machine)

				v1alpha6Machine.ObjectMeta.Annotations = map[string]string{}

				// In v1alpha4 boot from volume only supports
				// image by UUID, and boot from local only
				// suppots image by name
				if v1alpha6Machine.Spec.RootVolume != nil && v1alpha6Machine.Spec.RootVolume.Size > 0 {
					v1alpha6Machine.Spec.Image = ""
				} else {
					v1alpha6Machine.Spec.ImageUUID = ""
				}
			},
			func(v1alpha6MachineTemplate *infrav1.OpenStackMachineTemplate, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha6MachineTemplate)

				v1alpha6MachineTemplate.ObjectMeta.Annotations = map[string]string{}

				v1alpha6MachineTemplate.Spec.Template.Spec.Image = ""
			},
			func(v1alpha6Instance *infrav1.Instance, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha6Instance)

				// In v1alpha4 boot from volume only supports
				// image by UUID, and boot from local only
				// suppots image by name
				if v1alpha6Instance.RootVolume != nil && v1alpha6Instance.RootVolume.Size > 0 {
					v1alpha6Instance.Image = ""
				} else {
					v1alpha6Instance.ImageUUID = ""
				}
			},
			func(v1alpha6RootVolume *infrav1.RootVolume, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha6RootVolume)

				v1alpha6RootVolume.VolumeType = ""
				v1alpha6RootVolume.AvailabilityZone = ""
			},
			func(v1alpha6ClusterTemplate *infrav1.OpenStackClusterTemplate, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha6ClusterTemplate)

				v1alpha6ClusterTemplate.ObjectMeta.Annotations = map[string]string{}

				v1alpha6ClusterTemplate.Spec.Template.Spec.APIServerLoadBalancer.AllowedCIDRs = nil

				if v1alpha6ClusterTemplate.Spec.Template.Spec.Bastion != nil {
					v1alpha6ClusterTemplate.Spec.Template.Spec.Bastion.Instance.Image = ""
				}
			},
		}
	}

	t.Run("for OpenStackCluster", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &infrav1.OpenStackCluster{},
		Spoke:       &OpenStackCluster{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzerFuncs},
	}))

	t.Run("for OpenStackClusterTemplate", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &infrav1.OpenStackClusterTemplate{},
		Spoke:       &OpenStackClusterTemplate{},
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
