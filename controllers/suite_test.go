/*
Copyright 2020 The Kubernetes Authors.

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

package controllers

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/compute"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
	"sigs.k8s.io/cluster-api-provider-openstack/test/helpers/external"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

// See https://github.com/onsi/ginkgo/blob/ver2/docs/MIGRATING_TO_V2.md#removed-async-testing
var _ = BeforeSuite(func() {
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "config", "crd", "bases"),
		},
		// Add fake CAPI CRDs that we reference
		CRDs: []*apiextensionsv1.CustomResourceDefinition{
			external.TestClusterCRD.DeepCopy(),
			external.TestMachineCRD.DeepCopy(),
		},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = infrav1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	framework.TryAddDefaultSchemes(scheme.Scheme)

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("EnvTest sanity check", func() {
	ctx = context.TODO()
	It("should be able to create a namespace", func() {
		testNamespace := "capo-test"
		namespacedName := types.NamespacedName{
			Name: testNamespace,
		}
		namespaceInput := framework.CreateNamespaceInput{
			Creator: k8sClient,
			Name:    testNamespace,
		}

		// Create the namespace
		namespace := framework.CreateNamespace(ctx, namespaceInput)
		// Check the result
		namespaceResult := &corev1.Namespace{}
		err := k8sClient.Get(ctx, namespacedName, namespaceResult)
		Expect(err).To(BeNil())
		Expect(namespaceResult).To(Equal(namespace))

		// Clean up
		foregroundDeletePropagation := metav1.DeletePropagationForeground
		err = k8sClient.Delete(ctx, namespace, &client.DeleteOptions{PropagationPolicy: &foregroundDeletePropagation})
		Expect(err).To(BeNil())
		// Note: Since the controller-manager is not part of envtest the namespace
		// will actually stay in "Terminating" state and never be completely gone.
	})
})

var _ = Describe("When calling getOrCreate", func() {
	logger := GinkgoLogr

	var (
		reconsiler       OpenStackMachineReconciler
		mockCtrl         *gomock.Controller
		mockScopeFactory *scope.MockScopeFactory
		computeService   *compute.Service
		err              error
	)

	BeforeEach(func() {
		ctx = context.Background()
		reconsiler = OpenStackMachineReconciler{}
		mockCtrl = gomock.NewController(GinkgoT())
		mockScopeFactory = scope.NewMockScopeFactory(mockCtrl, "1234")
		computeService, err = compute.NewService(scope.NewWithLogger(mockScopeFactory, logger))
		Expect(err).NotTo(HaveOccurred())
	})

	It("should return an error if unable to get instance", func() {
		openStackCluster := &infrav1.OpenStackCluster{}
		machine := &clusterv1.Machine{}
		openStackMachine := &infrav1.OpenStackMachine{
			Status: infrav1.OpenStackMachineStatus{
				InstanceID: ptr.To("machine-uuid"),
			},
		}

		mockScopeFactory.ComputeClient.EXPECT().GetServer(gomock.Any()).Return(nil, errors.New("Test error when getting server"))
		instanceStatus, err := reconsiler.getOrCreateInstance(logger, openStackCluster, machine, openStackMachine, computeService, "", []string{})
		Expect(err).To(HaveOccurred())
		Expect(instanceStatus).To(BeNil())
		conditions := openStackMachine.GetConditions()
		Expect(len(conditions) > 0).To(BeTrue())
		for i := range conditions {
			if conditions[i].Type == infrav1.InstanceReadyCondition {
				Expect(conditions[i].Reason).To(Equal(infrav1.OpenStackErrorReason))
				break
			}
		}
	})

	It("should retrieve instance by name if no ID is stored", func() {
		openStackCluster := &infrav1.OpenStackCluster{}
		machine := &clusterv1.Machine{}
		openStackMachine := &infrav1.OpenStackMachine{}
		servers := make([]clients.ServerExt, 1)
		servers[0].ID = "machine-uuid"

		mockScopeFactory.ComputeClient.EXPECT().ListServers(gomock.Any()).Return(servers, nil)
		instanceStatus, err := reconsiler.getOrCreateInstance(logger, openStackCluster, machine, openStackMachine, computeService, "", []string{})
		Expect(err).ToNot(HaveOccurred())
		Expect(instanceStatus).ToNot(BeNil())
		Expect(instanceStatus.ID()).To(Equal("machine-uuid"))
	})
})
