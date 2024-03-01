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

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

var _ = Describe("OpenStackCluster API validations", func() {
	var cluster *infrav1.OpenStackCluster
	var namespace *corev1.Namespace

	BeforeEach(func() {
		namespace = createNamespace()

		// Initialise a basic cluster object in the correct namespace
		cluster = &infrav1.OpenStackCluster{}
		cluster.Namespace = namespace.Name
		cluster.GenerateName = "cluster-"
	})

	It("should allow the smallest permissible cluster spec", func() {
		Expect(k8sClient.Create(ctx, cluster)).To(Succeed(), "OpenStackCluster creation should succeed")
	})

	It("should only allow controlPlaneEndpoint to be set once", func() {
		By("Creating a bare cluster")
		Expect(k8sClient.Create(ctx, cluster)).To(Succeed(), "OpenStackCluster creation should succeed")

		By("Setting the control plane endpoint")
		cluster.Spec.ControlPlaneEndpoint.Host = "foo"
		cluster.Spec.ControlPlaneEndpoint.Port = 1234
		Expect(k8sClient.Update(ctx, cluster)).To(Succeed(), "Setting control plane endpoint should succeed")

		By("Modifying the control plane endpoint")
		cluster.Spec.ControlPlaneEndpoint.Host = "bar"
		Expect(k8sClient.Update(ctx, cluster)).NotTo(Succeed(), "Updating control plane endpoint should fail")
	})

	It("should allow an empty managed security groups definition", func() {
		cluster.Spec.ManagedSecurityGroups = &infrav1.ManagedSecurityGroups{}
		Expect(k8sClient.Create(ctx, cluster)).To(Succeed(), "OpenStackCluster creation should succeed")
	})
})
