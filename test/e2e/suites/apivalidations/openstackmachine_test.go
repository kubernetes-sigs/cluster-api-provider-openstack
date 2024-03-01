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
	"k8s.io/utils/pointer"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

var _ = Describe("OpenStackMachine API validations", func() {
	var namespace *corev1.Namespace
	var machine *infrav1.OpenStackMachine

	BeforeEach(func() {
		namespace = createNamespace()

		// Initialise a basic machine object in the correct namespace
		machine = &infrav1.OpenStackMachine{}
		machine.Namespace = namespace.Name
		machine.GenerateName = "machine-"
	})

	It("should allow the smallest permissible machine spec", func() {
		Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "OpenStackMachine creation should succeed")
	})

	It("should only allow the providerID to be set once", func() {
		By("Creating a bare machine")
		Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "OpenStackMachine creation should succeed")

		By("Setting the providerID")
		machine.Spec.ProviderID = pointer.String("foo")
		Expect(k8sClient.Update(ctx, machine)).To(Succeed(), "Setting providerID should succeed")

		By("Modifying the providerID")
		machine.Spec.ProviderID = pointer.String("bar")
		Expect(k8sClient.Update(ctx, machine)).NotTo(Succeed(), "Updating providerID should fail")
	})
})
