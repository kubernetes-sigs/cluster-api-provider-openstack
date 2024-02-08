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

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/servergroups"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha8"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

var (
	reconcilerServerGroup *OpenStackServerGroupReconciler
	//ctx                   context.Context
	testServerGroup *infrav1.OpenStackServerGroup
	//testNamespace         string
	//mockCtrl              *gomock.Controller
	//mockScopeFactory      *scope.MockScopeFactory
)

var _ = Describe("OpenStackServerGroup controller", func() {
	testServerGroupName := "test-servergroup"
	testNum := 0

	BeforeEach(func() {
		ctx = context.TODO()
		testNum++
		testNamespace = fmt.Sprintf("testservergroup-%d", testNum)

		// Create a standard ServerGroup definition for all tests
		testServerGroup = &infrav1.OpenStackServerGroup{
			TypeMeta: metav1.TypeMeta{
				APIVersion: infrav1.GroupVersion.Group + "/" + infrav1.GroupVersion.Version,
				Kind:       "OpenStackServerGroup",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testServerGroupName,
				Namespace: testNamespace,
			},
			Spec:   infrav1.OpenStackServerGroupSpec{},
			Status: infrav1.OpenStackServerGroupStatus{},
		}
		// Set finalizers, so the first reconcile doesn't need to by default.
		testServerGroup.SetFinalizers([]string{infrav1.ServerGroupFinalizer})

		input := framework.CreateNamespaceInput{
			Creator: k8sClient,
			Name:    testNamespace,
		}
		framework.CreateNamespace(ctx, input)

		mockCtrl = gomock.NewController(GinkgoT())
		mockScopeFactory = scope.NewMockScopeFactory(mockCtrl, "", logr.Discard())
		reconcilerServerGroup = func() *OpenStackServerGroupReconciler {
			return &OpenStackServerGroupReconciler{
				Client:       k8sClient,
				ScopeFactory: mockScopeFactory,
			}
		}()

	})

	AfterEach(func() {
		orphan := metav1.DeletePropagationOrphan
		deleteOptions := client.DeleteOptions{
			PropagationPolicy: &orphan,
		}

		// Delete OpenstackServerGroup
		patchHelper, err := patch.NewHelper(testServerGroup, k8sClient)
		Expect(err).To(BeNil())
		err = patchHelper.Patch(ctx, testServerGroup)
		Expect(err).To(BeNil())
		err = k8sClient.Delete(ctx, testServerGroup, &deleteOptions)
		Expect(err).To(BeNil())
		input := framework.DeleteNamespaceInput{
			Deleter: k8sClient,
			Name:    testNamespace,
		}
		framework.DeleteNamespace(ctx, input)
	})

	It("should do nothing when servergroup resource is paused", func() {
		testServerGroup.SetName("paused")
		annotations.AddAnnotations(testServerGroup, map[string]string{clusterv1.PausedAnnotation: "true"})
		testServerGroup.SetFinalizers([]string{})

		err := k8sClient.Create(ctx, testServerGroup)
		Expect(err).To(BeNil())
		req := createRequestFromServerGroup(testServerGroup)

		result, err := reconcilerServerGroup.Reconcile(ctx, req)
		Expect(err).To(BeNil())
		Expect(result).To(Equal(reconcile.Result{}))

		// Ensure Finalizer was not set by paused reconcile
		err = k8sClient.Get(ctx, req.NamespacedName, testServerGroup)
		Expect(err).To(BeNil())
		Expect(testServerGroup.GetFinalizers()).To(BeNil())
	})
	It("should do nothing when unable to get OS client", func() {
		testServerGroup.SetName("no-openstack-client")

		err := k8sClient.Create(ctx, testServerGroup)
		Expect(err).To(BeNil())
		req := createRequestFromServerGroup(testServerGroup)

		clientCreateErr := fmt.Errorf("Test failure")
		mockScopeFactory.SetClientScopeCreateError(clientCreateErr)

		result, err := reconcilerServerGroup.Reconcile(ctx, req)
		Expect(err).To(MatchError(clientCreateErr))
		Expect(result).To(Equal(reconcile.Result{}))
	})
	It("should add a finalizer on the first reconcile", func() {
		testServerGroup.SetName("finalizer-add")
		testServerGroup.SetFinalizers([]string{})

		err := k8sClient.Create(ctx, testServerGroup)
		Expect(err).To(BeNil())

		// Reconcile our resource and make sure the finalizer was set
		req := createRequestFromServerGroup(testServerGroup)
		result, err := reconcilerServerGroup.Reconcile(ctx, req)
		Expect(err).To(BeNil())
		Expect(result).To(Equal(reconcile.Result{}))

		// Retrieve the server group from K8s client
		err = k8sClient.Get(ctx, req.NamespacedName, testServerGroup)
		Expect(err).To(BeNil())

		Expect(testServerGroup.GetFinalizers()).To(Equal([]string{infrav1.ServerGroupFinalizer}))
	})
	It("should adopt an existing servergroup even if its uuid is not stored in status", func() {
		testServerGroup.SetName("adopt-existing-servergroup")

		// Set up servergroup spec, and status with no uuid
		testServerGroup.Spec = infrav1.OpenStackServerGroupSpec{
			Policy: "anti-affinity",
		}
		err := k8sClient.Create(ctx, testServerGroup)
		Expect(err).To(BeNil())
		testServerGroup.Status = infrav1.OpenStackServerGroupStatus{
			ID:    "",
			Ready: false,
		}
		// Write the test resource to k8s client
		err = k8sClient.Status().Update(ctx, testServerGroup)
		Expect(err).To(BeNil())

		// Define and record the existing resource the reconcile will see.
		servergroups := []servergroups.ServerGroup{
			{
				Name:     "adopt-existing-servergroup",
				ID:       "adopted-servergroup-uuid",
				Policies: []string{"anti-affinity"},
			},
		}
		computeClientRecorder := mockScopeFactory.ComputeClient.EXPECT()
		computeClientRecorder.ListServerGroups().Return(servergroups, nil)

		// Reconcile our resource, and make sure it adopted the expected resource.
		req := createRequestFromServerGroup(testServerGroup)
		result, err := reconcilerServerGroup.Reconcile(ctx, req)
		Expect(err).To(BeNil())
		Expect(result).To(Equal(reconcile.Result{}))

		// Retrieve the server group from K8s client
		err = k8sClient.Get(ctx, req.NamespacedName, testServerGroup)
		Expect(err).To(BeNil())

		Expect(testServerGroup.Status.ID).To(Equal("adopted-servergroup-uuid"))
		Expect(testServerGroup.Status.Ready).To(BeTrue())
	})

	It("should delete an existing servergroup even if its uuid is not stored in status", func() {
		testServerGroup.SetName("delete-existing-servergroup-no-uuid")

		// Set up servergroup spec, and status with no uuid.
		testServerGroup.Spec = infrav1.OpenStackServerGroupSpec{
			Policy: "anti-affinity",
		}
		err := k8sClient.Create(ctx, testServerGroup)
		Expect(err).To(BeNil())
		testServerGroup.Status = infrav1.OpenStackServerGroupStatus{
			ID:    "",
			Ready: false,
		}
		// Write the test resource to k8s client
		err = k8sClient.Status().Update(ctx, testServerGroup)
		Expect(err).To(BeNil())

		// Define and record the existing resource the reconcile will see.
		servergroups := []servergroups.ServerGroup{
			{
				Name:     "delete-existing-servergroup-no-uuid",
				ID:       "existing-servergroup-uuid",
				Policies: []string{"anti-affinity"},
			},
		}
		computeClientRecorder := mockScopeFactory.ComputeClient.EXPECT()
		computeClientRecorder.ListServerGroups().Return(servergroups, nil)
		computeClientRecorder.DeleteServerGroup("existing-servergroup-uuid").Return(nil)

		// Reconcile our resource, and make sure it finds the expected resource, then deletes it.
		scope, err := mockScopeFactory.NewClientScopeFromServerGroup(ctx, k8sClient, testServerGroup, nil, logr.Discard())
		Expect(err).To(BeNil())
		result, err := reconcilerServerGroup.reconcileDelete(scope, testServerGroup)
		Expect(err).To(BeNil())
		Expect(result).To(Equal(reconcile.Result{}))

		// Finalizer should now be removed.
		Expect(testServerGroup.GetFinalizers()).To(Equal([]string{}))
	})

	It("should succeed reconcile delete even if the servergroup does not exist", func() {
		testServerGroup.SetName("delete-servergroup-not-exist")

		// Set up servergroup spec, and status with no uuid.
		testServerGroup.Spec = infrav1.OpenStackServerGroupSpec{
			Policy: "anti-affinity",
		}
		err := k8sClient.Create(ctx, testServerGroup)
		Expect(err).To(BeNil())
		testServerGroup.Status = infrav1.OpenStackServerGroupStatus{
			ID:    "",
			Ready: false,
		}
		// Write the test resource to k8s client
		err = k8sClient.Status().Update(ctx, testServerGroup)
		Expect(err).To(BeNil())

		// Define and record the existing resource the reconcile will see.
		servergroups := []servergroups.ServerGroup{}
		computeClientRecorder := mockScopeFactory.ComputeClient.EXPECT()
		computeClientRecorder.ListServerGroups().Return(servergroups, nil)

		// Reconcile our resource, and make sure it finds the expected resource, then deletes it.
		scope, err := mockScopeFactory.NewClientScopeFromServerGroup(ctx, k8sClient, testServerGroup, nil, logr.Discard())
		Expect(err).To(BeNil())
		result, err := reconcilerServerGroup.reconcileDelete(scope, testServerGroup)
		Expect(err).To(BeNil())
		Expect(result).To(Equal(reconcile.Result{}))

		// Finalizer should now be removed.
		Expect(testServerGroup.GetFinalizers()).To(Equal([]string{}))
	})

	It("should requeue if the service returns temporary errors", func() {
		testServerGroup.SetName("requeue-on-openstack-error")

		// Set up servergroup spec
		testServerGroup.Spec = infrav1.OpenStackServerGroupSpec{
			Policy: "anti-affinity",
		}
		err := k8sClient.Create(ctx, testServerGroup)
		Expect(err).To(BeNil())
		testServerGroup.Status = infrav1.OpenStackServerGroupStatus{
			ID:    "",
			Ready: false,
		}
		// Write the test resource to k8s client
		err = k8sClient.Status().Update(ctx, testServerGroup)
		Expect(err).To(BeNil())

		// Define and record the existing resource the reconcile will see.
		servergroups := []servergroups.ServerGroup{}
		computeClientRecorder := mockScopeFactory.ComputeClient.EXPECT()
		computeClientRecorder.ListServerGroups().Return(servergroups, nil)
		expected_error := gophercloud.ErrDefault500{}
		computeClientRecorder.CreateServerGroup("requeue-on-openstack-error", "anti-affinity").Return(nil, expected_error)

		// Reconcile our resource, and make sure it adopted the expected resource.
		scope, err := mockScopeFactory.NewClientScopeFromServerGroup(ctx, k8sClient, testServerGroup, nil, logr.Discard())
		Expect(err).To(BeNil())
		result, err := reconcilerServerGroup.reconcileNormal(ctx, scope, testServerGroup)
		// Expect error to surface with empty result.
		// Due to the error, the reconcile should be re-tried with default backoff by ControllerRuntime
		Expect(err).To(Equal(expected_error))
		Expect(result).To(Equal(reconcile.Result{}))
	})

})

func createRequestFromServerGroup(openStackServerGroup *infrav1.OpenStackServerGroup) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      openStackServerGroup.GetName(),
			Namespace: openStackServerGroup.GetNamespace(),
		},
	}
}
