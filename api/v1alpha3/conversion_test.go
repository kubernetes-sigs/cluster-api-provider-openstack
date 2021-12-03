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
	v1beta1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
)

func TestFuzzyConversion(t *testing.T) {
	g := gomega.NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(AddToScheme(scheme)).To(gomega.Succeed())
	g.Expect(v1beta1.AddToScheme(scheme)).To(gomega.Succeed())

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
			func(v1alpha3MachineSpec *OpenStackMachineSpec, c fuzz.Continue) {
				c.FuzzNoCustom(v1alpha3MachineSpec)

				v1alpha3MachineSpec.UserDataSecret = nil

				if v1alpha3MachineSpec.CloudsSecret != nil {
					// In switching to IdentityRef, fetching the cloud secret
					// from a different namespace is no longer supported
					v1alpha3MachineSpec.CloudsSecret.Namespace = ""
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

			// Don't test hub-spoke-hub conversion of v1beta1 fields which are not in v1alpha3
			func(v1beta1ClusterSpec *v1beta1.OpenStackClusterSpec, c fuzz.Continue) {
				c.FuzzNoCustom(v1beta1ClusterSpec)

				v1beta1ClusterSpec.APIServerFixedIP = ""
				v1beta1ClusterSpec.AllowAllInClusterTraffic = false
				v1beta1ClusterSpec.DisableAPIServerFloatingIP = false
			},
			func(v1beta1MachineSpec *v1beta1.OpenStackMachineSpec, c fuzz.Continue) {
				c.FuzzNoCustom(v1beta1MachineSpec)

				v1beta1MachineSpec.Ports = nil
			},
			func(v1beta1Network *v1beta1.Network, c fuzz.Continue) {
				c.FuzzNoCustom(v1beta1Network)

				v1beta1Network.PortOpts = nil
			},
			func(v1beta1ClusterStatus *v1beta1.OpenStackClusterStatus, c fuzz.Continue) {
				c.FuzzNoCustom(v1beta1ClusterStatus)

				v1beta1ClusterStatus.FailureMessage = nil
				v1beta1ClusterStatus.FailureReason = nil
			},
			func(v1beta1OpenStackIdentityRef *v1beta1.OpenStackIdentityReference, c fuzz.Continue) {
				c.FuzzNoCustom(v1beta1OpenStackIdentityRef)

				// IdentityRef was assumed to be a Secret in v1alpha3
				v1beta1OpenStackIdentityRef.Kind = "Secret"
			},
		}
	}

	t.Run("for OpenStackCluster", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &v1beta1.OpenStackCluster{},
		Spoke:       &OpenStackCluster{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzerFuncs},
	}))

	t.Run("for OpenStackMachine", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &v1beta1.OpenStackMachine{},
		Spoke:       &OpenStackMachine{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzerFuncs},
	}))

	t.Run("for OpenStackMachineTemplate", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &v1beta1.OpenStackMachineTemplate{},
		Spoke:       &OpenStackMachineTemplate{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzerFuncs},
	}))
}
