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

package image

import (
	"context"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
)

func TestController(t *testing.T) {
	RegisterFailHandler(Fail)

	suiteConfig, reporterConfig := GinkgoConfiguration()
	// Display logs written to GinkgoLogr
	reporterConfig.Verbose = true

	RunSpecs(t, "ORC Image Controller Suite", suiteConfig, reporterConfig)
}

var _ = BeforeSuite(func() {
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "config", "crd", "bases"),
		},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = orcv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{})
	Expect(err).ToNot(HaveOccurred(), "initialise controller-runtime cluster")
	Expect(k8sClient).ToNot(BeNil(), "initialise controller-runtime cluster")
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("EnvTest sanity check", func() {
	It("should be able to create a namespace", func() {
		ctx := context.TODO()
		namespace := &corev1.Namespace{}
		namespace.SetGenerateName("test-")

		// Create the namespace
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed(), "create namespace")
		DeferCleanup(func() {
			Expect(k8sClient.Delete(ctx, namespace)).To(Succeed(), "delete namespace")
		})

		// Check the result
		namespaceResult := &corev1.Namespace{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: namespace.GetName()}, namespaceResult)).To(Succeed(), "get namespace")
		Expect(namespaceResult).To(Equal(namespace))
	})
})
