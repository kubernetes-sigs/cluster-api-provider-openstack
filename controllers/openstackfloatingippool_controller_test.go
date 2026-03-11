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
	"context"
	"fmt"

	floatingips "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	networks "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	"go.uber.org/mock/gomock"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	ipamv1 "sigs.k8s.io/cluster-api/api/ipam/v1beta2"
	"sigs.k8s.io/cluster-api/test/framework"
	v1beta1conditions "sigs.k8s.io/cluster-api/util/deprecated/v1beta1/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
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

	// Context: full IP lifecycle — tests IP allocation/release together with cluster pause/unpause.
	//
	// Resources:
	//   - Cluster A (clusterv1.Cluster) in testNamespace
	//
	// Scenario 1 — cluster NOT paused:
	//   1. Create IPAddressClaim → reconcile → IPAddress created, pool.Status.ClaimedIPs updated
	//   2. Delete IPAddressClaim → reconcile → floating IP deleted, pool.Status.ClaimedIPs updated
	//
	// Scenario 2 — cluster PAUSED then UNPAUSED:
	//   3. Pause cluster, create IPAddressClaim → reconcile is skipped (no IP allocated)
	//   4. Unpause cluster → reconcile runs → IPAddress created, pool.Status.ClaimedIPs updated
	//   5. Pause cluster, delete IPAddressClaim → reconcile is skipped (IP not released)
	//   6. Unpause cluster → reconcile runs → IP released, pool.Status.ClaimedIPs updated
	Context("IPAddressClaim lifecycle with cluster pause/unpause", func() {
		const (
			testClusterName = "test-cluster-a"
			poolName        = "lifecycle-pool"
			ip1             = "192.168.100.1"
			ip1ID           = "fip-id-1"
			ip2             = "192.168.100.2"
			ip2ID           = "fip-id-2"
			networkID       = "ext-net-id"
		)

		var (
			mgrCancel   context.CancelFunc
			mgrDone     chan struct{}
			mgrClient   client.Client
			testCluster *clusterv1.Cluster
		)

		BeforeEach(func() {
			testPool.SetName(poolName)

			var mgrCtx context.Context
			mgrCtx, mgrCancel = context.WithCancel(context.Background())
			mgrDone = make(chan struct{})

			// Build a manager so we can register field indexers that MatchingFields relies on.
			mgr, err := ctrl.NewManager(cfg, ctrl.Options{
				Scheme:                 k8sClient.Scheme(),
				Metrics:                metricsserver.Options{BindAddress: "0"},
				HealthProbeBindAddress: "0",
			})
			Expect(err).ToNot(HaveOccurred())

			// Register field indexers identical to those in SetupWithManager.
			Expect(mgr.GetFieldIndexer().IndexField(
				mgrCtx, &ipamv1.IPAddressClaim{},
				infrav1alpha1.OpenStackFloatingIPPoolNameIndex,
				func(rawObj client.Object) []string {
					c := rawObj.(*ipamv1.IPAddressClaim)
					if c.Spec.PoolRef.Kind != openStackFloatingIPPool {
						return nil
					}
					return []string{c.Spec.PoolRef.Name}
				},
			)).To(Succeed())

			Expect(mgr.GetFieldIndexer().IndexField(
				mgrCtx, &ipamv1.IPAddress{},
				infrav1alpha1.OpenStackFloatingIPPoolNameIndex,
				func(rawObj client.Object) []string {
					a := rawObj.(*ipamv1.IPAddress)
					if a.Spec.PoolRef.Kind != openStackFloatingIPPool {
						return nil
					}
					return []string{a.Spec.PoolRef.Name}
				},
			)).To(Succeed())

			mgrClient = mgr.GetClient()
			// Redirect the reconciler to use the manager's cached client.
			poolReconciler.Client = mgrClient

			go func() {
				defer close(mgrDone)
				_ = mgr.Start(mgrCtx)
			}()
			Expect(mgr.GetCache().WaitForCacheSync(mgrCtx)).To(BeTrue())

			// Cluster A – used as the owning cluster for all IPAddressClaims in this context.
			testCluster = &clusterv1.Cluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: clusterv1.GroupVersion.String(),
					Kind:       "Cluster",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      testClusterName,
					Namespace: testNamespace,
				},
			}
			Expect(mgrClient.Create(ctx, testCluster)).To(Succeed())
			Eventually(func() error {
				return mgrClient.Get(ctx, client.ObjectKey{Name: testClusterName, Namespace: testNamespace}, &clusterv1.Cluster{})
			}, "5s").Should(Succeed())
		})

		AfterEach(func() {
			orphan := metav1.DeletePropagationOrphan
			_ = mgrClient.Delete(ctx, testCluster, &client.DeleteOptions{PropagationPolicy: &orphan})
			mgrCancel()
			Eventually(mgrDone, "10s").Should(BeClosed())
		})

		It("should allocate and release IPs, respecting cluster pause state", func() {
			tag := testPool.GetFloatingIPTag()

			// ─── Network mock expectations ───────────────────────────────────────
			net := poolMockFactory.NetworkClient.EXPECT()

			// Network discovery: GetNetworkByParam calls ListNetwork (may repeat).
			net.ListNetwork(gomock.Any()).
				Return([]networks.Network{{ID: networkID, Name: "external"}}, nil).
				AnyTimes()

			// GetFloatingIPsByTag: no pre-tagged IPs exist.
			net.ListFloatingIP(floatingips.ListOpts{Tags: tag}).
				Return([]floatingips.FloatingIP{}, nil).
				AnyTimes()

			// ip1 lifecycle (scenario 1 – not paused).
			net.CreateFloatingIP(gomock.Any()).
				Return(&floatingips.FloatingIP{FloatingIP: ip1, ID: ip1ID}, nil).
				Times(1)
			net.ListFloatingIP(floatingips.ListOpts{FloatingIP: ip1}).
				Return([]floatingips.FloatingIP{{FloatingIP: ip1, ID: ip1ID}}, nil).
				AnyTimes()
			net.ReplaceAllAttributesTags("floatingips", ip1ID, gomock.Any()).
				Return([]string{tag}, nil)
			net.DeleteFloatingIP(ip1ID).Return(nil)

			// ip2 lifecycle (scenario 2 – paused then unpaused).
			net.CreateFloatingIP(gomock.Any()).
				Return(&floatingips.FloatingIP{FloatingIP: ip2, ID: ip2ID}, nil).
				Times(1)
			net.ListFloatingIP(floatingips.ListOpts{FloatingIP: ip2}).
				Return([]floatingips.FloatingIP{{FloatingIP: ip2, ID: ip2ID}}, nil).
				AnyTimes()
			net.ReplaceAllAttributesTags("floatingips", ip2ID, gomock.Any()).
				Return([]string{tag}, nil)
			net.DeleteFloatingIP(ip2ID).Return(nil)

			// ─── Helpers ─────────────────────────────────────────────────────────
			poolKey := client.ObjectKey{Name: poolName, Namespace: testNamespace}
			reconcilePool := func() {
				req := reconcile.Request{NamespacedName: poolKey}
				_, err := poolReconciler.Reconcile(ctx, req)
				Expect(err).ToNot(HaveOccurred())
			}
			getPool := func() *infrav1alpha1.OpenStackFloatingIPPool {
				p := &infrav1alpha1.OpenStackFloatingIPPool{}
				Expect(mgrClient.Get(ctx, poolKey, p)).To(Succeed())
				return p
			}
			newClaim := func(name string) *ipamv1.IPAddressClaim {
				return &ipamv1.IPAddressClaim{
					TypeMeta: metav1.TypeMeta{
						APIVersion: ipamv1.GroupVersion.String(),
						Kind:       "IPAddressClaim",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: testNamespace,
						// ClusterNameLabel lets GetClusterFromMetadata locate cluster A.
						Labels: map[string]string{clusterv1.ClusterNameLabel: testClusterName},
					},
					Spec: ipamv1.IPAddressClaimSpec{
						PoolRef: ipamv1.IPPoolReference{
							APIGroup: infrav1alpha1.SchemeGroupVersion.Group,
							Kind:     openStackFloatingIPPool,
							Name:     poolName,
						},
					},
				}
			}
			pauseCluster := func() {
				patched := testCluster.DeepCopy()
				patched.Spec.Paused = ptr.To(true)
				Expect(mgrClient.Patch(ctx, patched, client.MergeFrom(testCluster))).To(Succeed())
				testCluster = patched
				Eventually(func() bool {
					c := &clusterv1.Cluster{}
					_ = mgrClient.Get(ctx, client.ObjectKey{Name: testClusterName, Namespace: testNamespace}, c)
					return ptr.Deref(c.Spec.Paused, false)
				}, "5s").Should(BeTrue())
			}
			unpauseCluster := func() {
				patched := testCluster.DeepCopy()
				patched.Spec.Paused = nil
				Expect(mgrClient.Patch(ctx, patched, client.MergeFrom(testCluster))).To(Succeed())
				testCluster = patched
				Eventually(func() bool {
					c := &clusterv1.Cluster{}
					_ = mgrClient.Get(ctx, client.ObjectKey{Name: testClusterName, Namespace: testNamespace}, c)
					return !ptr.Deref(c.Spec.Paused, false)
				}, "5s").Should(BeTrue())
			}

			// ─────────────────────────────────────────────────────────────────────
			// SCENARIO 1: cluster NOT paused
			// ─────────────────────────────────────────────────────────────────────
			By("Scenario 1: create pool and first claim (cluster not paused)")

			Expect(mgrClient.Create(ctx, testPool)).To(Succeed())
			Eventually(func() error {
				return mgrClient.Get(ctx, poolKey, &infrav1alpha1.OpenStackFloatingIPPool{})
			}, "5s").Should(Succeed())

			claim1 := newClaim("claim-1")
			Expect(mgrClient.Create(ctx, claim1)).To(Succeed())
			Eventually(func() error {
				return mgrClient.Get(ctx, client.ObjectKey{Name: "claim-1", Namespace: testNamespace}, &ipamv1.IPAddressClaim{})
			}, "5s").Should(Succeed())

			// Pass 1: pool gets its finalizer (early return).
			reconcilePool()
			// Wait for the pool finalizer to be in the cache before the next pass.
			Eventually(func() bool {
				p := &infrav1alpha1.OpenStackFloatingIPPool{}
				if err := mgrClient.Get(ctx, poolKey, p); err != nil {
					return false
				}
				return controllerutil.ContainsFinalizer(p, infrav1alpha1.OpenStackFloatingIPPoolFinalizer)
			}, "5s").Should(BeTrue(), "pool should have its finalizer in cache")

			// Pass 2: network discovered, claim gets its finalizer (early return via continue).
			reconcilePool()
			// Wait for claim1 to have its finalizer in the cache before the next pass.
			// This prevents stale-cache issues in subsequent reconcile passes.
			Eventually(func() bool {
				c := &ipamv1.IPAddressClaim{}
				if err := mgrClient.Get(ctx, client.ObjectKey{Name: "claim-1", Namespace: testNamespace}, c); err != nil {
					return false
				}
				return controllerutil.ContainsFinalizer(c, infrav1alpha1.OpenStackFloatingIPPoolFinalizer)
			}, "5s").Should(BeTrue(), "claim-1 should have its finalizer in cache")

			// Pass 3: IP allocated, IPAddress created, claim.Status.AddressRef set.
			reconcilePool()

			By("Scenario 1: verify IP is claimed")
			Eventually(func() []string {
				return getPool().Status.ClaimedIPs
			}, "5s").Should(ContainElement(ip1))

			// Also wait for the cache to reflect claim1.Status.AddressRef so the
			// claim deletion reconcile path can see it and call deleteIPAddress.
			Eventually(func() string {
				c := &ipamv1.IPAddressClaim{}
				_ = mgrClient.Get(ctx, client.ObjectKey{Name: "claim-1", Namespace: testNamespace}, c)
				return c.Status.AddressRef.Name
			}, "5s").Should(Equal("claim-1"))

			ipAddress1 := &ipamv1.IPAddress{}
			Expect(mgrClient.Get(ctx, client.ObjectKey{Name: "claim-1", Namespace: testNamespace}, ipAddress1)).To(Succeed())
			Expect(ipAddress1.Spec.Address).To(Equal(ip1))

			By("Scenario 1: delete claim → IP released")
			Expect(mgrClient.Delete(ctx, claim1)).To(Succeed())
			Eventually(func() bool {
				c := &ipamv1.IPAddressClaim{}
				if err := mgrClient.Get(ctx, client.ObjectKey{Name: "claim-1", Namespace: testNamespace}, c); err != nil {
					return false
				}
				return !c.DeletionTimestamp.IsZero()
			}, "5s").Should(BeTrue())

			// Reconcile: processes claim deletion — removes DeleteFloatingIPFinalizer
			// from IPAddress and removes pool finalizer from claim.
			reconcilePool()

			// Verify the DeleteFloatingIPFinalizer was removed from the IPAddress by
			// waiting for the cache to reflect the update.
			Eventually(func() bool {
				fresh := &ipamv1.IPAddress{}
				if err := k8sClient.Get(ctx, client.ObjectKey{Name: "claim-1", Namespace: testNamespace}, fresh); err != nil {
					return true // already gone
				}
				return len(fresh.Finalizers) == 0
			}, "5s").Should(BeTrue(), "DeleteFloatingIPFinalizer should be removed")

			// Simulate GC: manually delete the IPAddress now that its finalizer is removed.
			// Use the direct k8sClient and default (background) propagation — Orphan
			// propagation adds an internal GC finalizer that is never removed in envtest.
			freshIPAddr1 := &ipamv1.IPAddress{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: "claim-1", Namespace: testNamespace}, freshIPAddr1); err == nil {
				Expect(k8sClient.Delete(ctx, freshIPAddr1)).To(Succeed())
			}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Name: "claim-1", Namespace: testNamespace}, &ipamv1.IPAddress{})
			}, "5s").ShouldNot(Succeed(), "IPAddress should be deleted")

			// Wait for both claim1 and ipAddress1 to be gone from the manager cache
			// before the next reconcile — otherwise the reconciler would see stale
			// objects and try to re-add finalizers (causing a 422 or double-delete).
			Eventually(func() bool {
				return apierrors.IsNotFound(mgrClient.Get(ctx, client.ObjectKey{Name: "claim-1", Namespace: testNamespace}, &ipamv1.IPAddressClaim{}))
			}, "5s").Should(BeTrue(), "claim-1 should be gone from cache")
			Eventually(func() bool {
				return apierrors.IsNotFound(mgrClient.Get(ctx, client.ObjectKey{Name: "claim-1", Namespace: testNamespace}, &ipamv1.IPAddress{}))
			}, "5s").Should(BeTrue(), "ipAddress1 should be gone from cache")

			// Reconcile: pool status rebuilt without ip1.
			reconcilePool()
			Expect(getPool().Status.ClaimedIPs).NotTo(ContainElement(ip1))

			// ─────────────────────────────────────────────────────────────────────
			// SCENARIO 2a: cluster PAUSED – create claim is skipped
			// ─────────────────────────────────────────────────────────────────────
			By("Scenario 2a: pause cluster, create claim → reconcile skipped")

			pauseCluster()

			claim2 := newClaim("claim-2")
			Expect(mgrClient.Create(ctx, claim2)).To(Succeed())
			Eventually(func() error {
				return mgrClient.Get(ctx, client.ObjectKey{Name: "claim-2", Namespace: testNamespace}, &ipamv1.IPAddressClaim{})
			}, "5s").Should(Succeed())

			// Cluster is paused — reconcile runs but skips the claim entirely.
			reconcilePool()

			// Claim must still have no finalizer and no IP.
			fetchedClaim2 := &ipamv1.IPAddressClaim{}
			Expect(mgrClient.Get(ctx, client.ObjectKey{Name: "claim-2", Namespace: testNamespace}, fetchedClaim2)).To(Succeed())
			Expect(fetchedClaim2.Finalizers).To(BeEmpty())
			Expect(fetchedClaim2.Status.AddressRef.Name).To(BeEmpty())
			Expect(getPool().Status.ClaimedIPs).NotTo(ContainElement(ip2))

			// ─────────────────────────────────────────────────────────────────────
			// SCENARIO 2b: unpause → IP allocated
			// ─────────────────────────────────────────────────────────────────────
			By("Scenario 2b: unpause cluster → IP allocated")

			unpauseCluster()

			// Pass 1: claim gets its finalizer.
			reconcilePool()
			// Wait for claim2's finalizer to appear in the cache before the next pass.
			Eventually(func() bool {
				c := &ipamv1.IPAddressClaim{}
				if err := mgrClient.Get(ctx, client.ObjectKey{Name: "claim-2", Namespace: testNamespace}, c); err != nil {
					return false
				}
				return controllerutil.ContainsFinalizer(c, infrav1alpha1.OpenStackFloatingIPPoolFinalizer)
			}, "5s").Should(BeTrue(), "claim-2 should have its finalizer in cache")
			// Pass 2: IP allocated.
			reconcilePool()

			Eventually(func() []string {
				return getPool().Status.ClaimedIPs
			}, "5s").Should(ContainElement(ip2))

			// Wait for claim2's AddressRef to be visible in cache.
			Eventually(func() string {
				c := &ipamv1.IPAddressClaim{}
				_ = mgrClient.Get(ctx, client.ObjectKey{Name: "claim-2", Namespace: testNamespace}, c)
				return c.Status.AddressRef.Name
			}, "5s").Should(Equal("claim-2"))

			ipAddress2 := &ipamv1.IPAddress{}
			Expect(mgrClient.Get(ctx, client.ObjectKey{Name: "claim-2", Namespace: testNamespace}, ipAddress2)).To(Succeed())
			Expect(ipAddress2.Spec.Address).To(Equal(ip2))

			// ─────────────────────────────────────────────────────────────────────
			// SCENARIO 2c: cluster PAUSED again – delete claim is skipped
			// ─────────────────────────────────────────────────────────────────────
			By("Scenario 2c: pause cluster, delete claim → deletion skipped")

			pauseCluster()

			Expect(mgrClient.Delete(ctx, claim2)).To(Succeed())
			Eventually(func() bool {
				c := &ipamv1.IPAddressClaim{}
				if err := mgrClient.Get(ctx, client.ObjectKey{Name: "claim-2", Namespace: testNamespace}, c); err != nil {
					return false
				}
				return !c.DeletionTimestamp.IsZero()
			}, "5s").Should(BeTrue())

			// Cluster paused – reconcile should skip claim deletion.
			reconcilePool()

			// ip2 is STILL claimed; IPAddress still exists.
			Expect(getPool().Status.ClaimedIPs).To(ContainElement(ip2))
			Expect(mgrClient.Get(ctx, client.ObjectKey{Name: "claim-2", Namespace: testNamespace}, &ipamv1.IPAddress{})).To(Succeed())

			// ─────────────────────────────────────────────────────────────────────
			// SCENARIO 2d: unpause → IP released
			// ─────────────────────────────────────────────────────────────────────
			By("Scenario 2d: unpause cluster → IP released")

			unpauseCluster()

			// Reconcile: processes claim deletion and calls DeleteFloatingIP on OpenStack.
			reconcilePool()

			// Verify the DeleteFloatingIPFinalizer was removed from the IPAddress.
			Eventually(func() bool {
				fresh := &ipamv1.IPAddress{}
				if err := k8sClient.Get(ctx, client.ObjectKey{Name: "claim-2", Namespace: testNamespace}, fresh); err != nil {
					return true // already gone
				}
				return len(fresh.Finalizers) == 0
			}, "5s").Should(BeTrue(), "DeleteFloatingIPFinalizer should be removed")

			// Simulate GC: manually delete the IPAddress now that its finalizer is removed.
			// Use the direct k8sClient and default (background) propagation.
			freshIPAddr2 := &ipamv1.IPAddress{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: "claim-2", Namespace: testNamespace}, freshIPAddr2); err == nil {
				Expect(k8sClient.Delete(ctx, freshIPAddr2)).To(Succeed())
			}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Name: "claim-2", Namespace: testNamespace}, &ipamv1.IPAddress{})
			}, "5s").ShouldNot(Succeed(), "IPAddress should be deleted")

			// Wait for claim2 and ipAddress2 to be gone from the manager cache
			// before the final reconcile — prevents stale-object finalizer conflicts.
			Eventually(func() bool {
				return apierrors.IsNotFound(mgrClient.Get(ctx, client.ObjectKey{Name: "claim-2", Namespace: testNamespace}, &ipamv1.IPAddressClaim{}))
			}, "5s").Should(BeTrue(), "claim-2 should be gone from cache")
			Eventually(func() bool {
				return apierrors.IsNotFound(mgrClient.Get(ctx, client.ObjectKey{Name: "claim-2", Namespace: testNamespace}, &ipamv1.IPAddress{}))
			}, "5s").Should(BeTrue(), "ipAddress2 should be gone from cache")

			// Final reconcile: pool ClaimedIPs no longer contains ip2.
			reconcilePool()
			Expect(getPool().Status.ClaimedIPs).NotTo(ContainElement(ip2))
		})
	})
})
