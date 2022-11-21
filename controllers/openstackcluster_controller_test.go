/*
Copyright 2022 The Kubernetes Authors.

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
	"time"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha6"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/simulator"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
	"sigs.k8s.io/cluster-api-provider-openstack/test"
)

var (
	reconciler       *OpenStackClusterReconciler
	ctx              context.Context
	testCluster      *infrav1.OpenStackCluster
	capiCluster      *clusterv1.Cluster
	testNamespace    string
	mockCtrl         *gomock.Controller
	mockScopeFactory *scope.MockScopeFactory
)

var _ = Describe("OpenStackCluster controller", func() {
	capiClusterName := "capi-cluster"
	testClusterName := "test-cluster"
	testNum := 0

	BeforeEach(func() {
		ctx = context.TODO()
		testNum++
		testNamespace = fmt.Sprintf("test-%d", testNum)

		testCluster = &infrav1.OpenStackCluster{
			TypeMeta: metav1.TypeMeta{
				APIVersion: infrav1.GroupVersion.Group + "/" + infrav1.GroupVersion.Version,
				Kind:       "OpenStackCluster",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testClusterName,
				Namespace: testNamespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: clusterv1.GroupVersion.Group + "/" + clusterv1.GroupVersion.Version,
						Kind:       "Cluster",
						Name:       capiClusterName,
						UID:        types.UID("cluster-uid"),
					},
				},
			},
			Spec:   infrav1.OpenStackClusterSpec{},
			Status: infrav1.OpenStackClusterStatus{},
		}
		capiCluster = &clusterv1.Cluster{
			TypeMeta: metav1.TypeMeta{
				APIVersion: clusterv1.GroupVersion.Group + "/" + clusterv1.GroupVersion.Version,
				Kind:       "Cluster",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      capiClusterName,
				Namespace: testNamespace,
			},
		}

		input := framework.CreateNamespaceInput{
			Creator: k8sClient,
			Name:    testNamespace,
		}
		framework.CreateNamespace(ctx, input)

		mockCtrl = gomock.NewController(GinkgoT())
		mockScopeFactory = scope.NewMockScopeFactory(mockCtrl)
		reconciler = func() *OpenStackClusterReconciler {
			return &OpenStackClusterReconciler{
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

		// Remove finalizers and delete openstackcluster
		patchHelper, err := patch.NewHelper(testCluster, k8sClient)
		Expect(err).To(BeNil())
		testCluster.SetFinalizers([]string{})
		err = patchHelper.Patch(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Delete(ctx, testCluster, &deleteOptions)
		Expect(err).To(BeNil())
		// Remove finalizers and delete cluster
		patchHelper, err = patch.NewHelper(capiCluster, k8sClient)
		Expect(err).To(BeNil())
		capiCluster.SetFinalizers([]string{})
		err = patchHelper.Patch(ctx, capiCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Delete(ctx, capiCluster, &deleteOptions)
		Expect(err).To(BeNil())
		input := framework.DeleteNamespaceInput{
			Deleter: k8sClient,
			Name:    testNamespace,
		}
		framework.DeleteNamespace(ctx, input)
	})

	It("should do nothing when owner is missing", func() {
		testCluster.SetName("missing-owner")
		testCluster.SetOwnerReferences([]metav1.OwnerReference{})

		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())
		req := createRequestFromOSCluster(testCluster)

		result, err := reconciler.Reconcile(ctx, req)
		// Expect no error and empty result
		Expect(err).To(BeNil())
		Expect(result).To(Equal(reconcile.Result{}))
	})
	It("should do nothing when paused", func() {
		testCluster.SetName("paused")
		annotations.AddAnnotations(testCluster, map[string]string{clusterv1.PausedAnnotation: "true"})

		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())
		req := createRequestFromOSCluster(testCluster)

		result, err := reconciler.Reconcile(ctx, req)
		// Expect no error and empty result
		Expect(err).To(BeNil())
		Expect(result).To(Equal(reconcile.Result{}))
	})
	It("should do nothing when unable to get OS client", func() {
		testCluster.SetName("no-openstack-client")
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())
		req := createRequestFromOSCluster(testCluster)

		clientCreateErr := fmt.Errorf("Test failure")
		mockScopeFactory.SetClientScopeCreateError(clientCreateErr)

		result, err := reconciler.Reconcile(ctx, req)
		// Expect error for getting OS client and empty result
		Expect(err).To(MatchError(clientCreateErr))
		Expect(result).To(Equal(reconcile.Result{}))
	})
	It("should be able to reconcile when bastion is disabled and does not exist", func() {
		testCluster.SetName("no-bastion")
		testCluster.Spec = infrav1.OpenStackClusterSpec{
			Bastion: &infrav1.Bastion{
				Enabled: false,
			},
		}
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())
		scope, err := mockScopeFactory.NewClientScopeFromCluster(ctx, k8sClient, testCluster, logr.Discard())
		Expect(err).To(BeNil())

		computeClientRecorder := mockScopeFactory.ComputeClient.EXPECT()
		computeClientRecorder.ListServers(servers.ListOpts{
			Name: "^capi-cluster-bastion$",
		}).Return([]clients.ServerExt{}, nil)

		networkClientRecorder := mockScopeFactory.NetworkClient.EXPECT()
		networkClientRecorder.ListSecGroup(gomock.Any()).Return([]groups.SecGroup{}, nil)

		err = deleteBastion(scope, capiCluster, testCluster)
		Expect(err).To(BeNil())
	})
})

func createRequestFromOSCluster(openStackCluster *infrav1.OpenStackCluster) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      openStackCluster.GetName(),
			Namespace: openStackCluster.GetNamespace(),
		},
	}
}

var _ = Describe("OpenStackCluster controller Bastion", func() {
	const (
		externalNetworkUUID = "6c22ea3c-5589-4651-b559-ca8124e99d5e"
		imageUUID           = "9391d6ac-3f2c-4f0c-a840-d29fcec19bab"
		workerFlavorUUID    = "4d8bef18-a149-4b9a-be7e-0386173b9b08"
		projectID           = "60ff3abf-1f3d-41ba-aa3c-40a54c6790a7"
	)

	var (
		templateVars test.TemplateVars
		templateName string

		objs []runtime.Object
		sim  *simulator.OpenStackSimulator

		cleanupMgr func()

		capiCluster      *clusterv1.Cluster
		openStackCluster *infrav1.OpenStackCluster

		testNamespace string
		testNum       int

		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.TODO()

		testNum++
		testNamespace = fmt.Sprintf("cluster-bastion-test-%d", testNum)
		namespace := &corev1.Namespace{}
		namespace.Name = testNamespace
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

		templateVars = test.TemplateVars{
			ClusterName:                        "test-cluster",
			CNIResources:                       "{}",
			KubernetesVersion:                  "1.25.0",
			OpenStackBastionImageName:          "test-bastion-image",
			OpenStackBastionMachineFlavor:      "test-bastion-flavor",
			OpenStackCloud:                     "openstack",
			OpenStackCloudCACert:               "-",
			OpenStackCloudYAML:                 "---\nopenstack:",
			OpenStackCloudProviderConf:         "-",
			OpenStackDNSNameservers:            "8.8.8.8",
			OpenStackExternalNetworkID:         externalNetworkUUID,
			OpenStackFailureDomain:             "test-failuredomain",
			OpenStackImageName:                 "test-image",
			OpenStackSSHKeyName:                "test-keypair",
			OpenStackControlPlaneMachineFlavor: "test-controlplaneflavor",
			OpenStackNodeMachineFlavor:         "test-workerflavor",
			ControlPlaneMachineCount:           "3",
			WorkerMachineCount:                 "2",
		}
		templateName = "cluster-template.yaml"

		sim = simulator.NewOpenStackSimulator()
		sim.Network.SimAddNetworkWithSubnet("test-external-network", externalNetworkUUID, "192.168.0.0/24", true)
		sim.Compute.SimAddAvailabilityZone("test-az")
		sim.Image.SimAddImage("test-bastion-image", imageUUID)
		sim.Compute.SimAddFlavor("test-bastion-flavor", workerFlavorUUID)

		var mgr manager.Manager
		var runMgr func()
		mgr, runMgr, cleanupMgr = getManager(ctx, logger, scheme.Scheme)
		scopeFactory := scope.NewSimulatorScopeFactory(sim, projectID, logger)
		addClusterController(mgr, k8sClient, scopeFactory)

		go runMgr()
	})

	JustBeforeEach(func() {
		var err error
		objs, err = test.ReadTemplatedObjects(templateName, templateVars, scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		capiCluster = func() *clusterv1.Cluster {
			for _, obj := range objs {
				if capiCluster, ok := obj.(*clusterv1.Cluster); ok {
					return capiCluster
				}
			}
			return nil
		}()
		Expect(capiCluster).ToNot(BeNil())
		capiCluster.Namespace = testNamespace

		openStackCluster = func() *infrav1.OpenStackCluster {
			for _, obj := range objs {
				if openstackCluster, ok := obj.(*infrav1.OpenStackCluster); ok {
					return openstackCluster
				}
			}
			return nil
		}()
		Expect(openStackCluster).ToNot(BeNil())
		openStackCluster.Namespace = testNamespace
	})

	AfterEach(func() {
		// Move this to DeferCleanup() when we upgrade to ginkgo v2
		cleanupMgr()
	})

	Context("when the OpenStackCluster has an initial bastion", func() {
		It("should provision and deprovision successfully", func() {
			openStackCluster.Spec.Bastion = &infrav1.Bastion{
				Enabled: true,
				Instance: infrav1.OpenStackMachineSpec{
					Flavor:     "test-bastion-flavor",
					Image:      "test-bastion-image",
					SSHKeyName: "test-key",
					/*
						RootVolume: &infrav1.RootVolume{
							Size:             100,
							VolumeType:       "test-volumetype",
							AvailabilityZone: "test-az2",
						},
					*/
				},
				AvailabilityZone: "test-az2",
			}

			By("creating the cluster")
			createClusters(ctx, k8sClient, capiCluster, openStackCluster)

			By("waiting for bastion to exist")
			Eventually(func() *infrav1.Instance {
				Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: testNamespace, Name: openStackCluster.Name}, openStackCluster)).To(Succeed())
				return openStackCluster.Status.Bastion
			}).WithTimeout(30*time.Second).ShouldNot(BeNil(), "Bastion should exist")

			By("deleting the cluster")
			Expect(k8sClient.Delete(ctx, openStackCluster)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKey{Namespace: openStackCluster.Name, Name: openStackCluster.Namespace}, openStackCluster)
				return apierrors.IsNotFound(err)
			}).Should(BeTrue(), "OpenStackCluster should be deleted")
		})
	})

	Context("when the OpenStackCluster has no initial bastion", func() {
		It("should provision and deprovision successfully", func() {
			// The E2E templates have an initial bastion, so we need to remove it
			openStackCluster.Spec.Bastion = nil

			By("creating the cluster")
			createClusters(ctx, k8sClient, capiCluster, openStackCluster)

			By("waiting for the cluster to be ready")
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: testNamespace, Name: openStackCluster.Name}, openStackCluster)).To(Succeed())
				return openStackCluster.Status.Ready
			}).WithTimeout(30*time.Second).Should(BeTrue(), "OpenStackCluster should be ready")

			By("adding a bastion")
			// Retry because we can race with the controller
			Eventually(func() error {
				Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: testNamespace, Name: openStackCluster.Name}, openStackCluster)).To(Succeed())
				openStackCluster.Spec.Bastion = &infrav1.Bastion{
					Enabled: true,
					Instance: infrav1.OpenStackMachineSpec{
						Flavor:     "test-bastion-flavor",
						Image:      "test-bastion-image",
						SSHKeyName: "test-key",
						/*
							RootVolume: &infrav1.RootVolume{
								Size:             100,
								VolumeType:       "test-volumetype",
								AvailabilityZone: "test-az2",
							},
						*/
					},
					AvailabilityZone: "test-az2",
				}
				return k8sClient.Update(ctx, openStackCluster)
			}).Should(Succeed())

			By("waiting for bastion to exist")
			Eventually(func() *infrav1.Instance {
				Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: testNamespace, Name: openStackCluster.Name}, openStackCluster)).To(Succeed())
				return openStackCluster.Status.Bastion
			}).WithTimeout(30*time.Second).ShouldNot(BeNil(), "Bastion should exist")

			By("the bastion should have been created correctly")
			bastion := openStackCluster.Status.Bastion

			// We don't populate very much in APIStatus, so check it directly in the simulator
			simServer := sim.Compute.SimGetServer(bastion.ID)
			Expect(simServer).ToNot(BeNil(), "bastion should exist in simulator")
			Expect(simServer.Image).To(HaveKeyWithValue("name", "test-bastion-image"), "bastion image should be correct")
			Expect(simServer.AvailabilityZone).To(Equal("test-az2"), "bastion availability zone should be correct")

			Expect(bastion.SSHKeyName).To(Equal("test-key"), "bastion SSH key name should be correct")
			Expect(bastion.FloatingIP).ToNot(BeEmpty(), "bastion should have a floating IP")

			By("deleting the cluster")
			Expect(k8sClient.Delete(ctx, openStackCluster)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKey{Namespace: openStackCluster.Name, Name: openStackCluster.Namespace}, openStackCluster)
				return apierrors.IsNotFound(err)
			}).Should(BeTrue(), "OpenStackCluster should be deleted")
		})
	})
})

func createClusters(ctx context.Context, k8sClient client.Client, capiCluster *clusterv1.Cluster, openStackCluster *infrav1.OpenStackCluster) {
	Expect(k8sClient.Create(ctx, capiCluster)).To(Succeed())

	openStackCluster.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: clusterv1.GroupVersion.String(),
			Kind:       "Cluster",
			Name:       capiCluster.Name,
			UID:        capiCluster.UID,
		},
	}
	Expect(k8sClient.Create(ctx, openStackCluster)).To(Succeed())
}
