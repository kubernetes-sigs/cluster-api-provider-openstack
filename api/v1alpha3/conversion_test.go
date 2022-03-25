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

package v1alpha3

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	runtime "k8s.io/apimachinery/pkg/runtime"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

func TestConvertTo(t *testing.T) {
	g := gomega.NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(AddToScheme(scheme)).To(gomega.Succeed())
	g.Expect(infrav1.AddToScheme(scheme)).To(gomega.Succeed())

	tests := []struct {
		name  string
		spoke ctrlconversion.Convertible
		hub   ctrlconversion.Hub
		want  ctrlconversion.Hub
	}{
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

	tests := []struct {
		name  string
		spoke ctrlconversion.Convertible
		hub   ctrlconversion.Hub
		want  ctrlconversion.Convertible
	}{
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
			// Don't test spoke-hub-spoke conversion of v1alpha3 fields which are not in v1beta1
			func(v1alpha3ClusterSpec *OpenStackClusterSpec, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha3ClusterSpec)

				v1alpha3ClusterSpec.UseOctavia = false

				if v1alpha3ClusterSpec.CloudsSecret != nil {
					// In switching to IdentityRef, fetching the cloud secret
					// from a different namespace is no longer supported
					v1alpha3ClusterSpec.CloudsSecret.Namespace = ""
				}
			},
			func(v1alpha3RootVolume *RootVolume, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha3RootVolume)

				// In v1beta1 only DeviceType="disk" and SourceType="image" are supported
				v1alpha3RootVolume.DeviceType = "disk"
				v1alpha3RootVolume.SourceType = "image"
			},
			func(v1alpha3MachineSpec *OpenStackMachineSpec, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha3MachineSpec)

				v1alpha3MachineSpec.UserDataSecret = nil

				if v1alpha3MachineSpec.CloudsSecret != nil {
					// In switching to IdentityRef, fetching the cloud secret
					// from a different namespace is no longer supported
					v1alpha3MachineSpec.CloudsSecret.Namespace = ""
				}

				if v1alpha3MachineSpec.RootVolume != nil {
					// OpenStackMachineSpec.Image is ignored in v1alpha3 if RootVolume is set
					v1alpha3MachineSpec.Image = ""
				}
			},
			func(v1alpha3Instance *Instance, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha3Instance)

				if v1alpha3Instance.RootVolume != nil {
					// OpenStackInstance.Image is ignored in v1alpha3 if RootVolume is set
					v1alpha3Instance.Image = ""
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

				// TenantID and ProjectID are the same thing, so TenantID is removed in v1beta1
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

				// TenantID and ProjectID are the same thing, so TenantID is removed in v1beta1
				// Test that we restore TenantID from ProjectID
				v1alpha3Filter.TenantID = v1alpha3Filter.ProjectID
			},

			// Don't test hub-spoke-hub conversion of v1beta1 fields which are not in v1alpha3
			func(v1beta1ClusterSpec *infrav1.OpenStackClusterSpec, c fuzz.Continue) {
				c.FuzzNoCustom(v1beta1ClusterSpec)

				v1beta1ClusterSpec.APIServerFixedIP = ""
				v1beta1ClusterSpec.AllowAllInClusterTraffic = false
				v1beta1ClusterSpec.DisableAPIServerFloatingIP = false
			},
			func(v1beta1MachineSpec *infrav1.OpenStackMachineSpec, c fuzz.Continue) {
				c.FuzzNoCustom(v1beta1MachineSpec)

				v1beta1MachineSpec.Ports = nil
				v1beta1MachineSpec.ImageUUID = ""
			},
			func(v1beta1Network *infrav1.Network, c fuzz.Continue) {
				c.FuzzNoCustom(v1beta1Network)

				v1beta1Network.PortOpts = nil
			},
			func(v1beta1ClusterStatus *infrav1.OpenStackClusterStatus, c fuzz.Continue) {
				c.FuzzNoCustom(v1beta1ClusterStatus)

				v1beta1ClusterStatus.FailureMessage = nil
				v1beta1ClusterStatus.FailureReason = nil
				if v1beta1ClusterStatus.Bastion != nil {
					v1beta1ClusterStatus.Bastion.ImageUUID = ""
				}
			},
			func(v1beta1OpenStackIdentityRef *infrav1.OpenStackIdentityReference, c fuzz.Continue) {
				c.FuzzNoCustom(v1beta1OpenStackIdentityRef)

				// IdentityRef was assumed to be a Secret in v1alpha3
				v1beta1OpenStackIdentityRef.Kind = "Secret"
			},
			func(v1beta1RootVolume *infrav1.RootVolume, c fuzz.Continue) {
				c.FuzzNoCustom(v1beta1RootVolume)

				v1beta1RootVolume.VolumeType = ""
				v1beta1RootVolume.AvailabilityZone = ""
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
