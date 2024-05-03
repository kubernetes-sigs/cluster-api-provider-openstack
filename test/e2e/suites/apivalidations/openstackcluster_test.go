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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1alpha6 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha6"
	infrav1alpha7 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha7"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

var _ = Describe("OpenStackCluster API validations", func() {
	var namespace *corev1.Namespace

	create := func(obj client.Object) error {
		err := k8sClient.Create(ctx, obj)
		if err == nil {
			DeferCleanup(func() error {
				return k8sClient.Delete(ctx, obj)
			})
		}
		return err
	}

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
			Expect(create(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")
		})

		It("should only allow controlPlaneEndpoint to be set once", func() {
			By("Creating a bare cluster")
			Expect(create(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")

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
			Expect(create(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")
		})

		It("should default enabled to true if APIServerLoadBalancer is specified without enabled=true", func() {
			cluster.Spec.APIServerLoadBalancer = &infrav1.APIServerLoadBalancer{}
			Expect(create(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")

			// Fetch the cluster and check the defaulting
			fetchedCluster := &infrav1.OpenStackCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, fetchedCluster)).To(Succeed(), "OpenStackCluster fetch should succeed")

			Expect(fetchedCluster.Spec.APIServerLoadBalancer.Enabled).ToNot(BeNil(), "APIServerLoadBalancer.Enabled should have been defaulted")
			Expect(*fetchedCluster.Spec.APIServerLoadBalancer.Enabled).To(BeTrue(), "APIServerLoadBalancer.Enabled should default to true")
		})

		It("should not default APIServerLoadBalancer if it is not specifid", func() {
			Expect(create(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")

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
					Image: infrav1.ImageParam{
						Filter: &infrav1.ImageFilter{
							Name: ptr.To("fake-image"),
						},
					},
				},
			}
			Expect(create(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")
		})

		It("should not allow bastion.enabled=true without a spec", func() {
			cluster.Spec.Bastion = &infrav1.Bastion{
				Enabled: ptr.To(true),
			}
			Expect(create(cluster)).NotTo(Succeed(), "OpenStackCluster creation should not succeed")
		})

		It("should not allow an empty Bastion", func() {
			cluster.Spec.Bastion = &infrav1.Bastion{}
			Expect(create(cluster)).NotTo(Succeed(), "OpenStackCluster creation should not succeed")
		})

		It("should default bastion.enabled=true", func() {
			cluster.Spec.Bastion = &infrav1.Bastion{
				Spec: &infrav1.OpenStackMachineSpec{
					Image: infrav1.ImageParam{
						Filter: &infrav1.ImageFilter{
							Name: ptr.To("fake-image"),
						},
					},
				},
			}
			Expect(create(cluster)).To(Succeed(), "OpenStackCluster creation should not succeed")

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
					Image: infrav1.ImageParam{
						Filter: &infrav1.ImageFilter{
							Name: ptr.To("fake-image"),
						},
					},
				},
				FloatingIP: ptr.To("10.0.0.0"),
			}
			Expect(create(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")
		})

		It("should not allow non-IPv4 as bastion floating IP", func() {
			cluster.Spec.Bastion = &infrav1.Bastion{
				Spec: &infrav1.OpenStackMachineSpec{
					Image: infrav1.ImageParam{
						Filter: &infrav1.ImageFilter{
							Name: ptr.To("fake-image"),
						},
					},
				},
				FloatingIP: ptr.To("foobar"),
			}
			Expect(create(cluster)).NotTo(Succeed(), "OpenStackCluster creation should not succeed")
		})
	})

	Context("v1alpha7", func() {
		var cluster *infrav1alpha7.OpenStackCluster

		BeforeEach(func() {
			// Initialise a basic cluster object in the correct namespace
			cluster = &infrav1alpha7.OpenStackCluster{}
			cluster.Namespace = namespace.Name
			cluster.GenerateName = clusterNamePrefix
		})

		It("should restore cluster spec idempotently after controller writes to controlPlaneEndpoint", func() {
			// Set identityRef.Kind, as it will be lost if the restorer does not execute
			cluster.Spec.IdentityRef = &infrav1alpha7.OpenStackIdentityReference{
				Kind: "FakeKind",
				Name: "identity-ref",
			}
			Expect(create(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")

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
			cluster = &infrav1alpha7.OpenStackCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: infrav1Cluster.Name, Namespace: infrav1Cluster.Namespace}, cluster)).To(Succeed(), "OpenStackCluster fetch should succeed")
			Expect(cluster.Spec.ControlPlaneEndpoint).To(Equal(*infrav1Cluster.Spec.ControlPlaneEndpoint), "Control plane endpoint should be restored")
			Expect(cluster.Spec.IdentityRef.Kind).To(Equal("FakeKind"), "IdentityRef.Kind should be restored")
		})

		It("should not enable an explicitly disabled bastion when converting to v1beta1", func() {
			cluster.Spec.Bastion = &infrav1alpha7.Bastion{Enabled: false}
			Expect(create(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")

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
	})

	Context("v1alpha6", func() {
		var cluster *infrav1alpha6.OpenStackCluster //nolint:staticcheck

		BeforeEach(func() {
			// Initialise a basic cluster object in the correct namespace
			cluster = &infrav1alpha6.OpenStackCluster{} //nolint:staticcheck
			cluster.Namespace = namespace.Name
			cluster.GenerateName = clusterNamePrefix
		})

		It("should restore cluster spec idempotently after controller writes to controlPlaneEndpoint", func() {
			// Set identityRef.Kind, as it will be lost if the restorer does not execute
			cluster.Spec.IdentityRef = &infrav1alpha6.OpenStackIdentityReference{
				Kind: "FakeKind",
				Name: "identity-ref",
			}
			Expect(create(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")

			// Fetch the infrav1 version of the cluster
			infrav1Cluster := &infrav1.OpenStackCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, infrav1Cluster)).To(Succeed(), "OpenStackCluster fetch should succeed")

			// Update the infrav1 cluster to set the control plane endpoint
			infrav1Cluster.Spec.ControlPlaneEndpoint = &clusterv1.APIEndpoint{
				Host: "foo",
				Port: 1234,
			}
			Expect(k8sClient.Update(ctx, infrav1Cluster)).To(Succeed(), "Setting control plane endpoint should succeed")

			// Fetch the v1alpha6 version of the cluster and ensure that both the new control plane endpoint and the identityRef.Kind are present
			cluster = &infrav1alpha6.OpenStackCluster{} //nolint:staticcheck
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: infrav1Cluster.Name, Namespace: infrav1Cluster.Namespace}, cluster)).To(Succeed(), "OpenStackCluster fetch should succeed")
			Expect(cluster.Spec.ControlPlaneEndpoint).To(Equal(*infrav1Cluster.Spec.ControlPlaneEndpoint), "Control plane endpoint should be restored")
			Expect(cluster.Spec.IdentityRef.Kind).To(Equal("FakeKind"), "IdentityRef.Kind should be restored")
		})

		It("should not enable an explicitly disabled bastion when converting to v1beta1", func() {
			cluster.Spec.Bastion = &infrav1alpha6.Bastion{Enabled: false}
			Expect(create(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")

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
	})
})
