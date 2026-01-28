/*
Copyright 2026 The Kubernetes Authors.

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
	"fmt"

	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
	v1beta1conditions "sigs.k8s.io/cluster-api/util/deprecated/v1beta1/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1alpha1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

var _ = Describe("OpenStackFloatingIPPool controller", func() {
	var (
		testPool        *infrav1alpha1.OpenStackFloatingIPPool
		testNamespace   string
		poolReconciler  *OpenStackFloatingIPPoolReconciler
		poolMockCtrl    *gomock.Controller
		poolMockFactory *scope.MockScopeFactory
		testNum         int
	)

	BeforeEach(func() {
		testNum++
		testNamespace = fmt.Sprintf("pool-test-%d", testNum)

		testPool = &infrav1alpha1.OpenStackFloatingIPPool{
			TypeMeta: metav1.TypeMeta{
				APIVersion: infrav1alpha1.SchemeGroupVersion.Group + "/" + infrav1alpha1.SchemeGroupVersion.Version,
				Kind:       "OpenStackFloatingIPPool",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pool",
				Namespace: testNamespace,
			},
			Spec: infrav1alpha1.OpenStackFloatingIPPoolSpec{
				IdentityRef: infrav1.OpenStackIdentityReference{
					Name:      "test-creds",
					CloudName: "openstack",
				},
				ReclaimPolicy: infrav1alpha1.ReclaimDelete,
			},
		}

		input := framework.CreateNamespaceInput{
			Creator: k8sClient,
			Name:    testNamespace,
		}
		framework.CreateNamespace(ctx, input)

		poolMockCtrl = gomock.NewController(GinkgoT())
		poolMockFactory = scope.NewMockScopeFactory(poolMockCtrl, "")
		poolReconciler = &OpenStackFloatingIPPoolReconciler{
			Client:       k8sClient,
			ScopeFactory: poolMockFactory,
		}
	})

	AfterEach(func() {
		orphan := metav1.DeletePropagationOrphan
		deleteOptions := client.DeleteOptions{
			PropagationPolicy: &orphan,
		}

		// Remove finalizers and delete openstackfloatingippool
		patchHelper, err := patch.NewHelper(testPool, k8sClient)
		Expect(err).To(BeNil())
		testPool.SetFinalizers([]string{})
		err = patchHelper.Patch(ctx, testPool)
		Expect(err).To(BeNil())
		err = k8sClient.Delete(ctx, testPool, &deleteOptions)
		Expect(err).To(BeNil())
	})

	It("should set OpenStackAuthenticationSucceededCondition to False when credentials secret is missing", func() {
		testPool.SetName("missing-pool-credentials")
		testPool.Spec.IdentityRef = infrav1.OpenStackIdentityReference{
			Type:      "Secret",
			Name:      "non-existent-secret",
			CloudName: "openstack",
		}

		err := k8sClient.Create(ctx, testPool)
		Expect(err).To(BeNil())

		credentialsErr := fmt.Errorf("secret not found: non-existent-secret")
		poolMockFactory.SetClientScopeCreateError(credentialsErr)

		req := reconcile.Request{
			NamespacedName: client.ObjectKey{
				Name:      testPool.Name,
				Namespace: testPool.Namespace,
			},
		}
		result, err := poolReconciler.Reconcile(ctx, req)

		Expect(err).To(MatchError(credentialsErr))
		Expect(result).To(Equal(reconcile.Result{}))

		// Fetch the updated OpenStackFloatingIPPool to verify the condition was set
		updatedPool := &infrav1alpha1.OpenStackFloatingIPPool{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: testPool.Name, Namespace: testPool.Namespace}, updatedPool)).To(Succeed())

		// Verify OpenStackAuthenticationSucceededCondition is set to False
		Expect(v1beta1conditions.IsFalse(updatedPool, infrav1.OpenStackAuthenticationSucceeded)).To(BeTrue())
		condition := v1beta1conditions.Get(updatedPool, infrav1.OpenStackAuthenticationSucceeded)
		Expect(condition).ToNot(BeNil())
		Expect(condition.Reason).To(Equal(infrav1.OpenStackAuthenticationFailedReason))
		Expect(condition.Severity).To(Equal(clusterv1beta1.ConditionSeverityError))
		Expect(condition.Message).To(ContainSubstring("Failed to create OpenStack client scope"))
	})

	It("should set OpenStackAuthenticationSucceededCondition to False when namespace is denied access to ClusterIdentity", func() {
		testPool.SetName("identity-access-denied-pool")
		testPool.Spec.IdentityRef = infrav1.OpenStackIdentityReference{
			Type:      "ClusterIdentity",
			Name:      "test-cluster-identity",
			CloudName: "openstack",
		}

		err := k8sClient.Create(ctx, testPool)
		Expect(err).To(BeNil())

		identityAccessErr := &scope.IdentityAccessDeniedError{
			IdentityName:       "test-cluster-identity",
			RequesterNamespace: testNamespace,
		}
		poolMockFactory.SetClientScopeCreateError(identityAccessErr)

		req := reconcile.Request{
			NamespacedName: client.ObjectKey{
				Name:      testPool.Name,
				Namespace: testPool.Namespace,
			},
		}
		result, err := poolReconciler.Reconcile(ctx, req)

		Expect(err).To(MatchError(identityAccessErr))
		Expect(result).To(Equal(reconcile.Result{}))

		// Fetch the updated OpenStackFloatingIPPool to verify the condition was set
		updatedPool := &infrav1alpha1.OpenStackFloatingIPPool{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: testPool.Name, Namespace: testPool.Namespace}, updatedPool)).To(Succeed())

		// Verify OpenStackAuthenticationSucceededCondition is set to False
		Expect(v1beta1conditions.IsFalse(updatedPool, infrav1.OpenStackAuthenticationSucceeded)).To(BeTrue())
		condition := v1beta1conditions.Get(updatedPool, infrav1.OpenStackAuthenticationSucceeded)
		Expect(condition).ToNot(BeNil())
		Expect(condition.Reason).To(Equal(infrav1.OpenStackAuthenticationFailedReason))
		Expect(condition.Severity).To(Equal(clusterv1beta1.ConditionSeverityError))
		Expect(condition.Message).To(ContainSubstring("Failed to create OpenStack client scope"))
	})
})
