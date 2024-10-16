/*
Copyright 2024 The Kubernetes Authors.

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

package apivalidations

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1alpha7 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha7"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

var _ = Describe("OpenStackCluster API validations", func() {
	var namespace *corev1.Namespace

	BeforeEach(func() {
		namespace = createNamespace()
	})

	Context("infrav1", func() {
		var cluster *infrav1.OpenStackCluster

		BeforeEach(func() {
			// Initialise a basic cluster object in the correct namespace
			cluster = &infrav1.OpenStackCluster{}
			cluster.Namespace = namespace.Name
			cluster.GenerateName = clusterNamePrefix
		})

		It("should allow the smallest permissible cluster spec", func() {
			Expect(createObj(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")
		})

		It("should only allow controlPlaneEndpoint to be set once", func() {
			By("Creating a bare cluster")
			Expect(createObj(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")

			By("Setting the control plane endpoint")
			cluster.Spec.ControlPlaneEndpoint = &clusterv1.APIEndpoint{
				Host: "foo",
				Port: 1234,
			}
			Expect(k8sClient.Update(ctx, cluster)).To(Succeed(), "Setting control plane endpoint should succeed")

			By("Modifying the control plane endpoint")
			cluster.Spec.ControlPlaneEndpoint.Host = "bar"
			Expect(k8sClient.Update(ctx, cluster)).NotTo(Succeed(), "Updating control plane endpoint should fail")
		})

		It("should allow an empty managed security groups definition", func() {
			cluster.Spec.ManagedSecurityGroups = &infrav1.ManagedSecurityGroups{}
			Expect(createObj(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")
		})

		It("should default enabled to true if APIServerLoadBalancer is specified without enabled=true", func() {
			cluster.Spec.APIServerLoadBalancer = &infrav1.APIServerLoadBalancer{}
			Expect(createObj(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")

			// Fetch the cluster and check the defaulting
			fetchedCluster := &infrav1.OpenStackCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, fetchedCluster)).To(Succeed(), "OpenStackCluster fetch should succeed")

			Expect(fetchedCluster.Spec.APIServerLoadBalancer.Enabled).ToNot(BeNil(), "APIServerLoadBalancer.Enabled should have been defaulted")
			Expect(*fetchedCluster.Spec.APIServerLoadBalancer.Enabled).To(BeTrue(), "APIServerLoadBalancer.Enabled should default to true")
		})

		It("should not default APIServerLoadBalancer if it is not specifid", func() {
			Expect(createObj(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")

			// Fetch the cluster and check the defaulting
			fetchedCluster := &infrav1.OpenStackCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, fetchedCluster)).To(Succeed(), "OpenStackCluster fetch should succeed")

			Expect(fetchedCluster.Spec.APIServerLoadBalancer).To(BeNil(), "APIServerLoadBalancer should not have been defaulted")
			Expect(fetchedCluster.Spec.APIServerLoadBalancer.IsEnabled()).To(BeFalse(), "APIServerLoadBalancer.Enabled should not have been defaulted")
		})

		It("should allow bastion.enabled=true with a spec", func() {
			cluster.Spec.Bastion = &infrav1.Bastion{
				Enabled: ptr.To(true),
				Spec: &infrav1.OpenStackMachineSpec{
					Flavor: ptr.To("flavor-name"),
					Image: infrav1.ImageParam{
						Filter: &infrav1.ImageFilter{
							Name: ptr.To("fake-image"),
						},
					},
				},
			}
			Expect(createObj(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")
		})

		It("should not allow bastion.enabled=true without a spec", func() {
			cluster.Spec.Bastion = &infrav1.Bastion{
				Enabled: ptr.To(true),
			}
			Expect(createObj(cluster)).NotTo(Succeed(), "OpenStackCluster creation should not succeed")
		})

		It("should not allow an empty Bastion", func() {
			cluster.Spec.Bastion = &infrav1.Bastion{}
			Expect(createObj(cluster)).NotTo(Succeed(), "OpenStackCluster creation should not succeed")
		})

		It("should default bastion.enabled=true", func() {
			cluster.Spec.Bastion = &infrav1.Bastion{
				Spec: &infrav1.OpenStackMachineSpec{
					Flavor: ptr.To("flavor-name"),
					Image: infrav1.ImageParam{
						Filter: &infrav1.ImageFilter{
							Name: ptr.To("fake-image"),
						},
					},
				},
			}
			Expect(createObj(cluster)).To(Succeed(), "OpenStackCluster creation should not succeed")

			// Fetch the cluster and check the defaulting
			fetchedCluster := &infrav1.OpenStackCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, fetchedCluster)).To(Succeed(), "OpenStackCluster fetch should succeed")
			Expect(fetchedCluster.Spec.Bastion.Enabled).ToNot(BeNil(), "Bastion.Enabled should have been defaulted")
			Expect(*fetchedCluster.Spec.Bastion.Enabled).To(BeTrueBecause("Bastion.Enabled should default to true"))
		})

		It("should allow IPv4 as bastion floatingIP", func() {
			cluster.Spec.Bastion = &infrav1.Bastion{
				Enabled: ptr.To(true),
				Spec: &infrav1.OpenStackMachineSpec{
					Flavor: ptr.To("flavor-name"),
					Image: infrav1.ImageParam{
						Filter: &infrav1.ImageFilter{
							Name: ptr.To("fake-image"),
						},
					},
				},
				FloatingIP: ptr.To("10.0.0.0"),
			}
			Expect(createObj(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")
		})

		It("should not allow non-IPv4 as bastion floating IP", func() {
			cluster.Spec.Bastion = &infrav1.Bastion{
				Spec: &infrav1.OpenStackMachineSpec{
					Flavor: ptr.To("flavor-name"),
					Image: infrav1.ImageParam{
						Filter: &infrav1.ImageFilter{
							Name: ptr.To("fake-image"),
						},
					},
				},
				FloatingIP: ptr.To("foobar"),
			}
			Expect(createObj(cluster)).NotTo(Succeed(), "OpenStackCluster creation should not succeed")
		})

		// We must use unstructured to set values which can't be marshalled by the Go type
		unstructuredClusterWithAPIPort := func(apiServerPort any) *unstructured.Unstructured {
			obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cluster)
			Expect(err).NotTo(HaveOccurred(), "converting cluster to unstructured")

			u := &unstructured.Unstructured{}
			u.Object = obj
			u.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   infrav1.SchemeGroupVersion.Group,
				Version: infrav1.SchemeGroupVersion.Version,
				Kind:    "OpenStackCluster",
			})
			spec := obj["spec"].(map[string]any)
			spec["apiServerPort"] = apiServerPort

			return u
		}

		It("should not allow apiServerPort greater than MaxUInt16", func() {
			u := unstructuredClusterWithAPIPort(math.MaxUint16 + 1)
			Expect(createObj(u)).NotTo(Succeed(), "OpenStackCluster creation should not succeed")
		})

		It("should not allow apiServerPort less than zero", func() {
			u := unstructuredClusterWithAPIPort(-1)
			Expect(createObj(u)).NotTo(Succeed(), "OpenStackCluster creation should not succeed")
		})

		It("should allow apiServerPort zero", func() {
			u := unstructuredClusterWithAPIPort(0)
			Expect(createObj(u)).To(Succeed(), "OpenStackCluster creation should succeed")
		})

		It("should allow apiServerPort 65535", func() {
			u := unstructuredClusterWithAPIPort(math.MaxUint16)
			Expect(createObj(u)).To(Succeed(), "OpenStackCluster creation should succeed")
		})
	})

	Context("v1alpha7", func() {
		var cluster *infrav1alpha7.OpenStackCluster //nolint: staticcheck

		BeforeEach(func() {
			// Initialise a basic cluster object in the correct namespace
			cluster = &infrav1alpha7.OpenStackCluster{} //nolint: staticcheck
			cluster.Namespace = namespace.Name
			cluster.GenerateName = clusterNamePrefix
		})

		It("should restore cluster spec idempotently after controller writes to controlPlaneEndpoint", func() {
			// Set identityRef.Kind, as it will be lost if the restorer does not execute
			cluster.Spec.IdentityRef = &infrav1alpha7.OpenStackIdentityReference{
				Kind: "FakeKind",
				Name: "identity-ref",
			}
			Expect(createObj(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")

			// Fetch the infrav1 version of the cluster
			infrav1Cluster := &infrav1.OpenStackCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, infrav1Cluster)).To(Succeed(), "OpenStackCluster fetch should succeed")

			// Update the infrav1 cluster to set the control plane endpoint
			infrav1Cluster.Spec.ControlPlaneEndpoint = &clusterv1.APIEndpoint{
				Host: "foo",
				Port: 1234,
			}
			Expect(k8sClient.Update(ctx, infrav1Cluster)).To(Succeed(), "Setting control plane endpoint should succeed")

			// Fetch the v1alpha7 version of the cluster and ensure that both the new control plane endpoint and the identityRef.Kind are present
			cluster = &infrav1alpha7.OpenStackCluster{} //nolint:staticcheck
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: infrav1Cluster.Name, Namespace: infrav1Cluster.Namespace}, cluster)).To(Succeed(), "OpenStackCluster fetch should succeed")
			Expect(cluster.Spec.ControlPlaneEndpoint).To(Equal(*infrav1Cluster.Spec.ControlPlaneEndpoint), "Control plane endpoint should be restored")
			Expect(cluster.Spec.IdentityRef.Kind).To(Equal("FakeKind"), "IdentityRef.Kind should be restored")
		})

		It("should not enable an explicitly disabled bastion when converting to v1beta1", func() {
			cluster.Spec.Bastion = &infrav1alpha7.Bastion{Enabled: false}
			Expect(createObj(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")

			// Fetch the infrav1 version of the cluster
			infrav1Cluster := &infrav1.OpenStackCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, infrav1Cluster)).To(Succeed(), "OpenStackCluster fetch should succeed")

			infrav1Bastion := infrav1Cluster.Spec.Bastion

			// NOTE(mdbooth): It may be reasonable to remove the
			// bastion if it is disabled with no other properties.
			// It would be reasonable to update the assertions
			// accordingly if we did that.
			Expect(infrav1Bastion).ToNot(BeNil(), "Bastion should not have been removed")
			Expect(infrav1Bastion.Enabled).To(Equal(ptr.To(false)), "Bastion should remain disabled")
		})

		It("should downgrade cleanly from infrav1", func() {
			infrav1Cluster := &infrav1.OpenStackCluster{}
			infrav1Cluster.Namespace = namespace.Name
			infrav1Cluster.GenerateName = clusterNamePrefix
			infrav1Cluster.Spec.IdentityRef.CloudName = "test-cloud"
			infrav1Cluster.Spec.IdentityRef.Name = "test-credentials"
			Expect(createObj(infrav1Cluster)).To(Succeed(), "infrav1 OpenStackCluster creation should succeed")

			// Just fetching the object as v1alpha6 doesn't trigger
			// validation failure, so we first fetch it and then
			// patch the object with identical contents. The patch
			// triggers a validation failure.
			cluster := &infrav1alpha7.OpenStackCluster{} //nolint: staticcheck
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: infrav1Cluster.Name, Namespace: infrav1Cluster.Namespace}, cluster)).To(Succeed(), "OpenStackCluster fetch should succeed")

			setObjectGVK(cluster)
			cluster.ManagedFields = nil
			Expect(k8sClient.Patch(ctx, cluster, client.Apply, client.FieldOwner("test"), client.ForceOwnership)).To(Succeed(), format.Object(cluster, 4))
		})
	})
})
