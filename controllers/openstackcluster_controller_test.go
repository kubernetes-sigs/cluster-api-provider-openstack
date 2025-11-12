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
	"reflect"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/util/annotations"
	v1beta1conditions "sigs.k8s.io/cluster-api/util/deprecated/v1beta1/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1alpha1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
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
	bastionSpec := infrav1.OpenStackMachineSpec{
		Flavor: ptr.To("flavor-name"),
		Image: infrav1.ImageParam{
			Filter: &infrav1.ImageFilter{
				Name: ptr.To("fake-name"),
			},
		},
	}

	BeforeEach(func() {
		ctx = context.TODO()
		testNum++
		testNamespace = fmt.Sprintf("test-%d", testNum)

		testCluster = &infrav1.OpenStackCluster{
			TypeMeta: metav1.TypeMeta{
				APIVersion: infrav1.SchemeGroupVersion.Group + "/" + infrav1.SchemeGroupVersion.Version,
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
			Spec: infrav1.OpenStackClusterSpec{
				IdentityRef: infrav1.OpenStackIdentityReference{
					Name:      "test-creds",
					CloudName: "openstack",
				},
			},
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
		mockScopeFactory = scope.NewMockScopeFactory(mockCtrl, "")
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

	It("should create OpenStackClusterIdentity (CRD present)", func() {
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())

		id := &infrav1alpha1.OpenStackClusterIdentity{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("id-%d", GinkgoRandomSeed()),
			},
			Spec: infrav1alpha1.OpenStackClusterIdentitySpec{
				SecretRef: infrav1alpha1.OpenStackCredentialSecretReference{
					Name:      "creds",
					Namespace: "capo-system",
				},
			},
		}
		Expect(k8sClient.Create(ctx, id)).To(Succeed())

		// Cleanup cluster-scoped resource since it won't be deleted with namespace
		DeferCleanup(func() {
			Expect(k8sClient.Delete(ctx, id)).To(Succeed())
		})
	})

	It("should successfully create OpenStackCluster with valid identityRef", func() {
		testCluster.Spec.IdentityRef = infrav1.OpenStackIdentityReference{
			Name:      "creds",
			CloudName: "openstack",
			// Type should default to "Secret"
		}
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())

		// Verify the object was created and Type was defaulted
		created := &infrav1.OpenStackCluster{}
		err = k8sClient.Get(ctx, client.ObjectKey{Name: testCluster.Name, Namespace: testCluster.Namespace}, created)
		Expect(err).To(Succeed())
		Expect(created.Spec.IdentityRef.Type).To(Equal("Secret"))
		Expect(created.Spec.IdentityRef.Name).To(Equal("creds"))
		Expect(created.Spec.IdentityRef.CloudName).To(Equal("openstack"))
	})

	It("should successfully create OpenStackCluster with ClusterIdentity type", func() {
		testCluster.Spec.IdentityRef = infrav1.OpenStackIdentityReference{
			Type:      "ClusterIdentity",
			Name:      "global-creds",
			CloudName: "openstack",
			Region:    "RegionOne",
		}
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())

		// Verify all fields are preserved
		created := &infrav1.OpenStackCluster{}
		err = k8sClient.Get(ctx, client.ObjectKey{Name: testCluster.Name, Namespace: testCluster.Namespace}, created)
		Expect(err).To(Succeed())
		Expect(created.Spec.IdentityRef.Type).To(Equal("ClusterIdentity"))
		Expect(created.Spec.IdentityRef.Name).To(Equal("global-creds"))
		Expect(created.Spec.IdentityRef.CloudName).To(Equal("openstack"))
		Expect(created.Spec.IdentityRef.Region).To(Equal("RegionOne"))
	})

	It("should fail when namespace is denied access to ClusterIdentity", func() {
		testCluster.SetName("identity-access-denied")
		testCluster.Spec.IdentityRef = infrav1.OpenStackIdentityReference{
			Type:      "ClusterIdentity",
			Name:      "test-cluster-identity",
			CloudName: "openstack",
		}

		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())

		identityAccessErr := &scope.IdentityAccessDeniedError{
			IdentityName:       "test-cluster-identity",
			RequesterNamespace: testNamespace,
		}
		mockScopeFactory.SetClientScopeCreateError(identityAccessErr)

		req := createRequestFromOSCluster(testCluster)
		result, err := reconciler.Reconcile(ctx, req)

		Expect(err).To(MatchError(identityAccessErr))
		Expect(result).To(Equal(reconcile.Result{}))
	})

	It("should reject updates that modify identityRef.region (immutable)", func() {
		testCluster.Spec = infrav1.OpenStackClusterSpec{
			IdentityRef: infrav1.OpenStackIdentityReference{
				Type:      "Secret",
				Name:      "creds",
				CloudName: "openstack",
				Region:    "RegionOne",
			},
		}
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())

		// Try to update region
		fetched := &infrav1.OpenStackCluster{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testClusterName, Namespace: testNamespace}, fetched)).To(Succeed())
		fetched.Spec.IdentityRef.Region = "RegionTwo"
		Expect(k8sClient.Update(ctx, fetched)).ToNot(Succeed())
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
	It("should be able to reconcile when bastion is explicitly disabled and does not exist", func() {
		testCluster.SetName("no-bastion-explicit")
		testCluster.Spec = infrav1.OpenStackClusterSpec{
			IdentityRef: infrav1.OpenStackIdentityReference{
				Name:      "test-creds",
				CloudName: "openstack",
			},
			Bastion: &infrav1.Bastion{Enabled: ptr.To(false)},
		}
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())
		testCluster.Status = infrav1.OpenStackClusterStatus{
			Bastion: &infrav1.BastionStatus{
				ID: "bastion-uuid",
			},
		}
		err = k8sClient.Status().Update(ctx, testCluster)
		Expect(err).To(BeNil())
		log := GinkgoLogr
		clientScope, err := mockScopeFactory.NewClientScopeFromObject(ctx, k8sClient, nil, log, testCluster)
		Expect(err).To(BeNil())
		scope := scope.NewWithLogger(clientScope, log)

		err = reconciler.deleteBastion(ctx, scope, capiCluster, testCluster)
		Expect(err).To(BeNil())
		Expect(testCluster.Status.Bastion).To(BeNil())
	})
	It("should delete an existing bastion even if its uuid is not stored in status", func() {
		testCluster.SetName("delete-existing-bastion")
		testCluster.Spec = infrav1.OpenStackClusterSpec{
			IdentityRef: infrav1.OpenStackIdentityReference{
				Name:      "test-creds",
				CloudName: "openstack",
			},
		}
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())
		testCluster.Status = infrav1.OpenStackClusterStatus{
			Network: &infrav1.NetworkStatusWithSubnets{
				NetworkStatus: infrav1.NetworkStatus{
					ID: "network-id",
				},
			},
		}
		err = k8sClient.Status().Update(ctx, testCluster)
		Expect(err).To(BeNil())

		log := GinkgoLogr
		clientScope, err := mockScopeFactory.NewClientScopeFromObject(ctx, k8sClient, nil, log, testCluster)
		Expect(err).To(BeNil())
		scope := scope.NewWithLogger(clientScope, log)

		err = reconciler.deleteBastion(ctx, scope, capiCluster, testCluster)
		Expect(err).To(BeNil())
	})

	It("should implicitly filter cluster subnets by cluster network", func() {
		const externalNetworkID = "a42211a2-4d2c-426f-9413-830e4b4abbbc"
		const clusterNetworkID = "6c90b532-7ba0-418a-a276-5ae55060b5b0"
		const clusterSubnetID = "cad5a91a-36de-4388-823b-b0cc82cadfdc"

		testCluster.SetName("subnet-filtering")
		testCluster.Spec = infrav1.OpenStackClusterSpec{
			IdentityRef: infrav1.OpenStackIdentityReference{
				Name:      "test-creds",
				CloudName: "openstack",
			},
			Bastion: &infrav1.Bastion{
				Enabled: ptr.To(true),
				Spec:    &bastionSpec,
			},
			DisableAPIServerFloatingIP: ptr.To(true),
			APIServerFixedIP:           ptr.To("10.0.0.1"),
			ExternalNetwork: &infrav1.NetworkParam{
				ID: ptr.To(externalNetworkID),
			},
			Network: &infrav1.NetworkParam{
				ID: ptr.To(clusterNetworkID),
			},
		}
		testCluster.Status = infrav1.OpenStackClusterStatus{
			Bastion: &infrav1.BastionStatus{
				Resources: &infrav1.MachineResources{
					Ports: []infrav1.PortStatus{
						{
							ID: "port-id",
						},
					},
				},
			},
		}
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())

		log := GinkgoLogr
		clientScope, err := mockScopeFactory.NewClientScopeFromObject(ctx, k8sClient, nil, log, testCluster)
		Expect(err).To(BeNil())
		scope := scope.NewWithLogger(clientScope, log)

		networkClientRecorder := mockScopeFactory.NetworkClient.EXPECT()

		// Fetch external network
		networkClientRecorder.GetNetwork(externalNetworkID).Return(&networks.Network{
			ID:   externalNetworkID,
			Name: "external-network",
		}, nil)

		// Fetch cluster network
		networkClientRecorder.GetNetwork(clusterNetworkID).Return(&networks.Network{
			ID:   clusterNetworkID,
			Name: "cluster-network",
		}, nil)

		// Fetching cluster subnets should be filtered by cluster network id
		networkClientRecorder.ListSubnet(subnets.ListOpts{
			NetworkID: clusterNetworkID,
		}).Return([]subnets.Subnet{
			{
				ID:   clusterSubnetID,
				Name: "cluster-subnet",
				CIDR: "192.168.0.0/24",
			},
		}, nil)

		err = reconcileNetworkComponents(scope, capiCluster, testCluster)
		Expect(err).To(BeNil())

		// Verify conditions are set correctly
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.NetworkReadyCondition)).To(BeTrue())
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.SecurityGroupsReadyCondition)).To(BeTrue())
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.APIEndpointReadyCondition)).To(BeTrue())
	})

	It("should allow two subnets for the cluster network", func() {
		const externalNetworkID = "a42211a2-4d2c-426f-9413-830e4b4abbbc"
		const clusterNetworkID = "6c90b532-7ba0-418a-a276-5ae55060b5b0"
		clusterSubnets := []string{"cad5a91a-36de-4388-823b-b0cc82cadfdc", "e2407c18-c4e7-4d3d-befa-8eec5d8756f2"}

		testCluster.SetName("subnet-filtering")
		testCluster.Spec = infrav1.OpenStackClusterSpec{
			IdentityRef: infrav1.OpenStackIdentityReference{
				Name:      "test-creds",
				CloudName: "openstack",
			},
			Bastion: &infrav1.Bastion{
				Enabled: ptr.To(true),
				Spec:    &bastionSpec,
			},
			DisableAPIServerFloatingIP: ptr.To(true),
			APIServerFixedIP:           ptr.To("10.0.0.1"),
			ExternalNetwork: &infrav1.NetworkParam{
				ID: ptr.To(externalNetworkID),
			},
			Network: &infrav1.NetworkParam{
				ID: ptr.To(clusterNetworkID),
			},
			Subnets: []infrav1.SubnetParam{
				{ID: &clusterSubnets[0]},
				{ID: &clusterSubnets[1]},
			},
		}
		testCluster.Status = infrav1.OpenStackClusterStatus{
			Bastion: &infrav1.BastionStatus{
				Resources: &infrav1.MachineResources{
					Ports: []infrav1.PortStatus{
						{
							ID: "port-id",
						},
					},
				},
			},
		}
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())

		log := GinkgoLogr
		clientScope, err := mockScopeFactory.NewClientScopeFromObject(ctx, k8sClient, nil, log, testCluster)
		Expect(err).To(BeNil())
		scope := scope.NewWithLogger(clientScope, log)

		networkClientRecorder := mockScopeFactory.NetworkClient.EXPECT()

		// Fetch external network
		networkClientRecorder.GetNetwork(externalNetworkID).Return(&networks.Network{
			ID:   externalNetworkID,
			Name: "external-network",
		}, nil)

		// Fetch cluster network
		networkClientRecorder.GetNetwork(clusterNetworkID).Return(&networks.Network{
			ID:   clusterNetworkID,
			Name: "cluster-network",
		}, nil)

		networkClientRecorder.GetSubnet(clusterSubnets[0]).Return(&subnets.Subnet{
			ID:        clusterSubnets[0],
			NetworkID: clusterNetworkID,
			Name:      "cluster-subnet",
			CIDR:      "192.168.0.0/24",
		}, nil)

		networkClientRecorder.GetSubnet(clusterSubnets[1]).Return(&subnets.Subnet{
			ID:        clusterSubnets[1],
			NetworkID: clusterNetworkID,
			Name:      "cluster-subnet-v6",
			CIDR:      "2001:db8:2222:5555::/64",
		}, nil)

		err = reconcileNetworkComponents(scope, capiCluster, testCluster)
		Expect(err).To(BeNil())
		Expect(len(testCluster.Status.Network.Subnets)).To(Equal(2))

		// Verify conditions are set correctly
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.NetworkReadyCondition)).To(BeTrue())
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.SecurityGroupsReadyCondition)).To(BeTrue())
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.APIEndpointReadyCondition)).To(BeTrue())
	})

	It("should allow fetch network by subnet", func() {
		const clusterNetworkID = "6c90b532-7ba0-418a-a276-5ae55060b5b0"
		const clusterSubnetID = "cad5a91a-36de-4388-823b-b0cc82cadfdc"

		testCluster.SetName("subnet-filtering")
		testCluster.Spec = infrav1.OpenStackClusterSpec{
			IdentityRef: infrav1.OpenStackIdentityReference{
				Name:      "test-creds",
				CloudName: "openstack",
			},
			DisableAPIServerFloatingIP: ptr.To(true),
			APIServerFixedIP:           ptr.To("10.0.0.1"),
			DisableExternalNetwork:     ptr.To(true),
			Subnets: []infrav1.SubnetParam{
				{ID: ptr.To(clusterSubnetID)},
			},
		}
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())

		log := GinkgoLogr
		clientScope, err := mockScopeFactory.NewClientScopeFromObject(ctx, k8sClient, nil, log, testCluster)
		Expect(err).To(BeNil())
		scope := scope.NewWithLogger(clientScope, log)

		networkClientRecorder := mockScopeFactory.NetworkClient.EXPECT()

		// Fetching cluster subnets should be filtered by cluster network id
		networkClientRecorder.GetSubnet(clusterSubnetID).Return(&subnets.Subnet{
			ID:        clusterSubnetID,
			CIDR:      "192.168.0.0/24",
			NetworkID: clusterNetworkID,
		}, nil)

		// Fetch cluster network using the NetworkID from the filtered Subnets
		networkClientRecorder.GetNetwork(clusterNetworkID).Return(&networks.Network{
			ID: clusterNetworkID,
		}, nil)

		err = reconcileNetworkComponents(scope, capiCluster, testCluster)
		Expect(err).To(BeNil())
		Expect(testCluster.Status.Network.ID).To(Equal(clusterNetworkID))

		// Verify conditions are set correctly
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.NetworkReadyCondition)).To(BeTrue())
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.SecurityGroupsReadyCondition)).To(BeTrue())
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.APIEndpointReadyCondition)).To(BeTrue())
	})

	It("reconcile pre-existing network components by id", func() {
		const clusterNetworkID = "6c90b532-7ba0-418a-a276-5ae55060b5b0"
		const clusterSubnetID = "cad5a91a-36de-4388-823b-b0cc82cadfdc"
		const clusterRouterID = "e2407c18-c4e7-4d3d-befa-8eec5d8756f2"

		testCluster.SetName("pre-existing-network-components-by-id")
		testCluster.Spec = infrav1.OpenStackClusterSpec{
			IdentityRef: infrav1.OpenStackIdentityReference{
				Name:      "test-creds",
				CloudName: "openstack",
			},
			Network: &infrav1.NetworkParam{
				ID: ptr.To(clusterNetworkID),
			},
			Subnets: []infrav1.SubnetParam{
				{
					ID: ptr.To(clusterSubnetID),
				},
			},
			ManagedSubnets: []infrav1.SubnetSpec{},
			Router: &infrav1.RouterParam{
				ID: ptr.To(clusterRouterID),
			},
		}
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())

		log := GinkgoLogr
		clientScope, err := mockScopeFactory.NewClientScopeFromObject(ctx, k8sClient, nil, log, testCluster)
		Expect(err).To(BeNil())
		scope := scope.NewWithLogger(clientScope, log)

		networkClientRecorder := mockScopeFactory.NetworkClient.EXPECT()

		networkClientRecorder.GetSubnet(clusterSubnetID).Return(&subnets.Subnet{
			ID:        clusterSubnetID,
			CIDR:      "192.168.0.0/24",
			NetworkID: clusterNetworkID,
		}, nil)

		networkClientRecorder.GetNetwork(clusterNetworkID).Return(&networks.Network{
			ID: clusterNetworkID,
		}, nil)

		networkClientRecorder.GetRouter(clusterRouterID).Return(&routers.Router{
			ID: clusterRouterID,
		}, nil)

		networkingService, err := networking.NewService(scope)
		Expect(err).To(BeNil())

		err = reconcilePreExistingNetworkComponents(scope, networkingService, testCluster)
		Expect(err).To(BeNil())
		Expect(testCluster.Status.Network.ID).To(Equal(clusterNetworkID))
		Expect(testCluster.Status.Network.Subnets[0].ID).To(Equal(clusterSubnetID))
		Expect(testCluster.Status.Router.ID).To(Equal(clusterRouterID))

		// Verify conditions are set correctly
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.NetworkReadyCondition)).To(BeTrue())
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.RouterReadyCondition)).To(BeTrue())
	})

	It("reconcile pre-existing network components by name", func() {
		const clusterNetworkID = "6c90b532-7ba0-418a-a276-5ae55060b5b0"
		const clusterNetworkName = "capo"
		const clusterSubnetID = "cad5a91a-36de-4388-823b-b0cc82cadfdc"
		const clusterSubnetName = "capo"
		const clusterRouterID = "e2407c18-c4e7-4d3d-befa-8eec5d8756f2"
		const clusterRouterName = "capo"

		testCluster.SetName("pre-existing-network-components-by-id")
		testCluster.Spec = infrav1.OpenStackClusterSpec{
			IdentityRef: infrav1.OpenStackIdentityReference{
				Name:      "test-creds",
				CloudName: "openstack",
			},
			Network: &infrav1.NetworkParam{
				Filter: &infrav1.NetworkFilter{
					Name: clusterNetworkName,
				},
			},
			Subnets: []infrav1.SubnetParam{
				{
					Filter: &infrav1.SubnetFilter{
						Name: clusterSubnetName,
					},
				},
			},
			ManagedSubnets: []infrav1.SubnetSpec{},
			Router: &infrav1.RouterParam{
				Filter: &infrav1.RouterFilter{
					Name: clusterRouterName,
				},
			},
		}
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())

		log := GinkgoLogr
		clientScope, err := mockScopeFactory.NewClientScopeFromObject(ctx, k8sClient, nil, log, testCluster)
		Expect(err).To(BeNil())
		scope := scope.NewWithLogger(clientScope, log)

		networkClientRecorder := mockScopeFactory.NetworkClient.EXPECT()

		networkClientRecorder.ListNetwork(networks.ListOpts{
			Name: clusterNetworkName,
		}).Return([]networks.Network{
			{
				ID: clusterNetworkID,
			},
		}, nil)

		networkClientRecorder.ListSubnet(subnets.ListOpts{
			Name:      clusterSubnetName,
			NetworkID: clusterNetworkID,
		}).Return([]subnets.Subnet{
			{
				ID:        clusterSubnetID,
				CIDR:      "192.168.0.0/24",
				NetworkID: clusterNetworkID,
			},
		}, nil)

		networkClientRecorder.ListRouter(routers.ListOpts{
			Name: clusterRouterName,
		}).Return([]routers.Router{
			{
				ID: clusterRouterID,
			},
		}, nil)

		networkingService, err := networking.NewService(scope)
		Expect(err).To(BeNil())

		err = reconcilePreExistingNetworkComponents(scope, networkingService, testCluster)
		Expect(err).To(BeNil())
		Expect(testCluster.Status.Network.ID).To(Equal(clusterNetworkID))
		Expect(testCluster.Status.Network.Subnets[0].ID).To(Equal(clusterSubnetID))
		Expect(testCluster.Status.Router.ID).To(Equal(clusterRouterID))

		// Verify conditions are set correctly
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.NetworkReadyCondition)).To(BeTrue())
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.RouterReadyCondition)).To(BeTrue())
	})

	It("should reconcile API endpoint with floating IP and set condition", func() {
		const externalNetworkID = "a42211a2-4d2c-426f-9413-830e4b4abbbc"
		const clusterNetworkID = "6c90b532-7ba0-418a-a276-5ae55060b5b0"
		const clusterSubnetID = "cad5a91a-36de-4388-823b-b0cc82cadfdc"
		const floatingIP = "203.0.113.10"

		testCluster.SetName("api-endpoint-floating-ip")
		testCluster.Spec = infrav1.OpenStackClusterSpec{
			IdentityRef: infrav1.OpenStackIdentityReference{
				Name:      "test-creds",
				CloudName: "openstack",
			},
			ExternalNetwork: &infrav1.NetworkParam{
				ID: ptr.To(externalNetworkID),
			},
			Network: &infrav1.NetworkParam{
				ID: ptr.To(clusterNetworkID),
			},
			// When DisableAPIServerFloatingIP is not set and external network is configured,
			// a floating IP should be created for the API server
		}
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())

		log := GinkgoLogr
		clientScope, err := mockScopeFactory.NewClientScopeFromObject(ctx, k8sClient, nil, log, testCluster)
		Expect(err).To(BeNil())
		scope := scope.NewWithLogger(clientScope, log)

		networkClientRecorder := mockScopeFactory.NetworkClient.EXPECT()

		// Fetch external network
		networkClientRecorder.GetNetwork(externalNetworkID).Return(&networks.Network{
			ID:   externalNetworkID,
			Name: "external-network",
		}, nil)

		// Fetch cluster network
		networkClientRecorder.GetNetwork(clusterNetworkID).Return(&networks.Network{
			ID:   clusterNetworkID,
			Name: "cluster-network",
		}, nil)

		// Fetching cluster subnets
		networkClientRecorder.ListSubnet(subnets.ListOpts{
			NetworkID: clusterNetworkID,
		}).Return([]subnets.Subnet{
			{
				ID:   clusterSubnetID,
				Name: "cluster-subnet",
				CIDR: "192.168.0.0/24",
			},
		}, nil)

		// Mock floating IP creation for API server
		// When no specific IP is requested, it will just create a new floating IP
		networkClientRecorder.CreateFloatingIP(gomock.Any()).Return(&floatingips.FloatingIP{
			FloatingIP: floatingIP,
			ID:         "floating-ip-id",
		}, nil)

		err = reconcileNetworkComponents(scope, capiCluster, testCluster)
		Expect(err).To(BeNil())

		// Verify API endpoint was set
		Expect(testCluster.Spec.ControlPlaneEndpoint).ToNot(BeNil())
		Expect(testCluster.Spec.ControlPlaneEndpoint.Host).To(Equal(floatingIP))
		Expect(testCluster.Spec.ControlPlaneEndpoint.Port).To(Equal(int32(6443)))

		// Verify conditions are set correctly
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.NetworkReadyCondition)).To(BeTrue())
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.SecurityGroupsReadyCondition)).To(BeTrue())
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.APIEndpointReadyCondition)).To(BeTrue())
	})

	It("should reconcile API endpoint with fixed IP and set condition", func() {
		const clusterNetworkID = "6c90b532-7ba0-418a-a276-5ae55060b5b0"
		const clusterSubnetID = "cad5a91a-36de-4388-823b-b0cc82cadfdc"
		const fixedIP = "192.168.0.10"

		testCluster.SetName("api-endpoint-fixed-ip")
		testCluster.Spec = infrav1.OpenStackClusterSpec{
			IdentityRef: infrav1.OpenStackIdentityReference{
				Name:      "test-creds",
				CloudName: "openstack",
			},
			Network: &infrav1.NetworkParam{
				ID: ptr.To(clusterNetworkID),
			},
			DisableExternalNetwork:     ptr.To(true),
			DisableAPIServerFloatingIP: ptr.To(true),
			APIServerFixedIP:           ptr.To(fixedIP),
		}
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())

		log := GinkgoLogr
		clientScope, err := mockScopeFactory.NewClientScopeFromObject(ctx, k8sClient, nil, log, testCluster)
		Expect(err).To(BeNil())
		scope := scope.NewWithLogger(clientScope, log)

		networkClientRecorder := mockScopeFactory.NetworkClient.EXPECT()

		// Fetch cluster network
		networkClientRecorder.GetNetwork(clusterNetworkID).Return(&networks.Network{
			ID:   clusterNetworkID,
			Name: "cluster-network",
		}, nil)

		// Fetching cluster subnets
		networkClientRecorder.ListSubnet(subnets.ListOpts{
			NetworkID: clusterNetworkID,
		}).Return([]subnets.Subnet{
			{
				ID:   clusterSubnetID,
				Name: "cluster-subnet",
				CIDR: "192.168.0.0/24",
			},
		}, nil)

		err = reconcileNetworkComponents(scope, capiCluster, testCluster)
		Expect(err).To(BeNil())

		// Verify API endpoint was set with fixed IP
		Expect(testCluster.Spec.ControlPlaneEndpoint).ToNot(BeNil())
		Expect(testCluster.Spec.ControlPlaneEndpoint.Host).To(Equal(fixedIP))
		Expect(testCluster.Spec.ControlPlaneEndpoint.Port).To(Equal(int32(6443)))

		// Verify conditions are set correctly
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.NetworkReadyCondition)).To(BeTrue())
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.SecurityGroupsReadyCondition)).To(BeTrue())
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.APIEndpointReadyCondition)).To(BeTrue())
	})

	It("should set NetworkReadyCondition to False when network lookup fails", func() {
		const clusterNetworkID = "6c90b532-7ba0-418a-a276-5ae55060b5b0"

		testCluster.SetName("network-lookup-failure")
		testCluster.Spec = infrav1.OpenStackClusterSpec{
			IdentityRef: infrav1.OpenStackIdentityReference{
				Name:      "test-creds",
				CloudName: "openstack",
			},
			Network: &infrav1.NetworkParam{
				ID: ptr.To(clusterNetworkID),
			},
			DisableExternalNetwork:     ptr.To(true),
			DisableAPIServerFloatingIP: ptr.To(true),
			APIServerFixedIP:           ptr.To("192.168.0.10"),
		}
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())

		log := GinkgoLogr
		clientScope, err := mockScopeFactory.NewClientScopeFromObject(ctx, k8sClient, nil, log, testCluster)
		Expect(err).To(BeNil())
		scope := scope.NewWithLogger(clientScope, log)

		networkClientRecorder := mockScopeFactory.NetworkClient.EXPECT()

		// Simulate network lookup failure
		networkClientRecorder.GetNetwork(clusterNetworkID).Return(nil, fmt.Errorf("unable to get network"))

		err = reconcileNetworkComponents(scope, capiCluster, testCluster)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("error fetching cluster network"))

		// Verify NetworkReadyCondition is set to False
		Expect(v1beta1conditions.IsFalse(testCluster, infrav1.NetworkReadyCondition)).To(BeTrue())
		condition := v1beta1conditions.Get(testCluster, infrav1.NetworkReadyCondition)
		Expect(condition).ToNot(BeNil())
		Expect(condition.Reason).To(Equal(infrav1.OpenStackErrorReason))
		Expect(condition.Severity).To(Equal(clusterv1beta1.ConditionSeverityError))
		Expect(condition.Message).To(ContainSubstring("Failed to find network"))
	})

	It("should set NetworkReadyCondition to False when subnet lookup fails", func() {
		const clusterNetworkID = "6c90b532-7ba0-418a-a276-5ae55060b5b0"

		testCluster.SetName("subnet-lookup-failure")
		testCluster.Spec = infrav1.OpenStackClusterSpec{
			IdentityRef: infrav1.OpenStackIdentityReference{
				Name:      "test-creds",
				CloudName: "openstack",
			},
			Network: &infrav1.NetworkParam{
				ID: ptr.To(clusterNetworkID),
			},
			DisableExternalNetwork:     ptr.To(true),
			DisableAPIServerFloatingIP: ptr.To(true),
			APIServerFixedIP:           ptr.To("192.168.0.10"),
		}
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())

		log := GinkgoLogr
		clientScope, err := mockScopeFactory.NewClientScopeFromObject(ctx, k8sClient, nil, log, testCluster)
		Expect(err).To(BeNil())
		scope := scope.NewWithLogger(clientScope, log)

		networkClientRecorder := mockScopeFactory.NetworkClient.EXPECT()

		// Network lookup succeeds
		networkClientRecorder.GetNetwork(clusterNetworkID).Return(&networks.Network{
			ID:   clusterNetworkID,
			Name: "cluster-network",
		}, nil)

		// Subnet list lookup fails
		networkClientRecorder.ListSubnet(subnets.ListOpts{
			NetworkID: clusterNetworkID,
		}).Return(nil, fmt.Errorf("failed to list subnets"))

		err = reconcileNetworkComponents(scope, capiCluster, testCluster)
		Expect(err).ToNot(BeNil())

		// Verify NetworkReadyCondition is set to False
		Expect(v1beta1conditions.IsFalse(testCluster, infrav1.NetworkReadyCondition)).To(BeTrue())
		condition := v1beta1conditions.Get(testCluster, infrav1.NetworkReadyCondition)
		Expect(condition).ToNot(BeNil())
		Expect(condition.Reason).To(Equal(infrav1.OpenStackErrorReason))
		Expect(condition.Severity).To(Equal(clusterv1beta1.ConditionSeverityError))
	})

	It("should set RouterReadyCondition to False when router lookup fails", func() {
		const clusterNetworkID = "6c90b532-7ba0-418a-a276-5ae55060b5b0"
		const clusterSubnetID = "cad5a91a-36de-4388-823b-b0cc82cadfdc"
		const clusterRouterID = "a0e2b0a5-4d2f-4e8d-9a1c-6b3e7f8c9d0e"

		testCluster.SetName("router-lookup-failure")
		testCluster.Spec = infrav1.OpenStackClusterSpec{
			IdentityRef: infrav1.OpenStackIdentityReference{
				Name:      "test-creds",
				CloudName: "openstack",
			},
			Network: &infrav1.NetworkParam{
				ID: ptr.To(clusterNetworkID),
			},
			Router: &infrav1.RouterParam{
				ID: ptr.To(clusterRouterID),
			},
			DisableExternalNetwork:     ptr.To(true),
			DisableAPIServerFloatingIP: ptr.To(true),
			APIServerFixedIP:           ptr.To("192.168.0.10"),
		}
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())

		log := GinkgoLogr
		clientScope, err := mockScopeFactory.NewClientScopeFromObject(ctx, k8sClient, nil, log, testCluster)
		Expect(err).To(BeNil())
		scope := scope.NewWithLogger(clientScope, log)

		networkClientRecorder := mockScopeFactory.NetworkClient.EXPECT()

		// Network lookup succeeds
		networkClientRecorder.GetNetwork(clusterNetworkID).Return(&networks.Network{
			ID:   clusterNetworkID,
			Name: "cluster-network",
		}, nil)

		// Subnet lookup succeeds
		networkClientRecorder.ListSubnet(subnets.ListOpts{
			NetworkID: clusterNetworkID,
		}).Return([]subnets.Subnet{
			{
				ID:   clusterSubnetID,
				Name: "cluster-subnet",
				CIDR: "192.168.0.0/24",
			},
		}, nil)

		// Router lookup fails
		networkClientRecorder.GetRouter(clusterRouterID).Return(nil, fmt.Errorf("unable to get router"))

		err = reconcileNetworkComponents(scope, capiCluster, testCluster)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("error fetching cluster router"))

		// Verify RouterReadyCondition is set to False
		Expect(v1beta1conditions.IsFalse(testCluster, infrav1.RouterReadyCondition)).To(BeTrue())
		condition := v1beta1conditions.Get(testCluster, infrav1.RouterReadyCondition)
		Expect(condition).ToNot(BeNil())
		Expect(condition.Reason).To(Equal(infrav1.OpenStackErrorReason))
		Expect(condition.Severity).To(Equal(clusterv1beta1.ConditionSeverityError))
		Expect(condition.Message).To(ContainSubstring("Failed to find router"))

		// NetworkReadyCondition should still be True since network succeeded
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.NetworkReadyCondition)).To(BeTrue())
	})

	It("should set SecurityGroupsReadyCondition to False when security group reconciliation fails", func() {
		const clusterNetworkID = "6c90b532-7ba0-418a-a276-5ae55060b5b0"
		const clusterSubnetID = "cad5a91a-36de-4388-823b-b0cc82cadfdc"

		testCluster.SetName("security-group-failure")
		testCluster.Spec = infrav1.OpenStackClusterSpec{
			IdentityRef: infrav1.OpenStackIdentityReference{
				Name:      "test-creds",
				CloudName: "openstack",
			},
			Network: &infrav1.NetworkParam{
				ID: ptr.To(clusterNetworkID),
			},
			DisableExternalNetwork:     ptr.To(true),
			DisableAPIServerFloatingIP: ptr.To(true),
			APIServerFixedIP:           ptr.To("192.168.0.10"),
			ManagedSecurityGroups: &infrav1.ManagedSecurityGroups{
				AllNodesSecurityGroupRules: []infrav1.SecurityGroupRuleSpec{
					{
						Direction: "ingress",
						Protocol:  ptr.To("tcp"),
						RemoteManagedGroups: []infrav1.ManagedSecurityGroupName{
							"worker",
						},
					},
				},
			},
		}
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())

		log := GinkgoLogr
		clientScope, err := mockScopeFactory.NewClientScopeFromObject(ctx, k8sClient, nil, log, testCluster)
		Expect(err).To(BeNil())
		scope := scope.NewWithLogger(clientScope, log)

		networkClientRecorder := mockScopeFactory.NetworkClient.EXPECT()

		// Network lookup succeeds
		networkClientRecorder.GetNetwork(clusterNetworkID).Return(&networks.Network{
			ID:   clusterNetworkID,
			Name: "cluster-network",
		}, nil)

		// Subnet lookup succeeds
		networkClientRecorder.ListSubnet(subnets.ListOpts{
			NetworkID: clusterNetworkID,
		}).Return([]subnets.Subnet{
			{
				ID:   clusterSubnetID,
				Name: "cluster-subnet",
				CIDR: "192.168.0.0/24",
			},
		}, nil)

		// Security group creation fails - this will trigger an error in getOrCreateSecurityGroup
		networkClientRecorder.ListSecGroup(gomock.Any()).Return([]groups.SecGroup{}, nil).AnyTimes()
		networkClientRecorder.CreateSecGroup(gomock.Any()).Return(nil, fmt.Errorf("quota exceeded")).AnyTimes()

		err = reconcileNetworkComponents(scope, capiCluster, testCluster)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("failed to reconcile security groups"))

		// Verify SecurityGroupsReadyCondition is set to False
		Expect(v1beta1conditions.IsFalse(testCluster, infrav1.SecurityGroupsReadyCondition)).To(BeTrue())
		condition := v1beta1conditions.Get(testCluster, infrav1.SecurityGroupsReadyCondition)
		Expect(condition).ToNot(BeNil())
		Expect(condition.Reason).To(Equal(infrav1.SecurityGroupReconcileFailedReason))
		Expect(condition.Severity).To(Equal(clusterv1beta1.ConditionSeverityError))
		Expect(condition.Message).To(ContainSubstring("Failed to reconcile security groups"))

		// NetworkReadyCondition should still be True since network succeeded
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.NetworkReadyCondition)).To(BeTrue())
	})

	It("should set APIEndpointReadyCondition to False when floating IP creation fails", func() {
		const externalNetworkID = "a42211a2-4d2c-426f-9413-830e4b4abbbc"
		const clusterNetworkID = "6c90b532-7ba0-418a-a276-5ae55060b5b0"
		const clusterSubnetID = "cad5a91a-36de-4388-823b-b0cc82cadfdc"

		testCluster.SetName("floating-ip-failure")
		testCluster.Spec = infrav1.OpenStackClusterSpec{
			IdentityRef: infrav1.OpenStackIdentityReference{
				Name:      "test-creds",
				CloudName: "openstack",
			},
			ExternalNetwork: &infrav1.NetworkParam{
				ID: ptr.To(externalNetworkID),
			},
			Network: &infrav1.NetworkParam{
				ID: ptr.To(clusterNetworkID),
			},
			// When DisableAPIServerFloatingIP is not set and external network is configured,
			// a floating IP should be created for the API server
		}
		err := k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())

		log := GinkgoLogr
		clientScope, err := mockScopeFactory.NewClientScopeFromObject(ctx, k8sClient, nil, log, testCluster)
		Expect(err).To(BeNil())
		scope := scope.NewWithLogger(clientScope, log)

		networkClientRecorder := mockScopeFactory.NetworkClient.EXPECT()

		// Fetch external network
		networkClientRecorder.GetNetwork(externalNetworkID).Return(&networks.Network{
			ID:   externalNetworkID,
			Name: "external-network",
		}, nil)

		// Fetch cluster network
		networkClientRecorder.GetNetwork(clusterNetworkID).Return(&networks.Network{
			ID:   clusterNetworkID,
			Name: "cluster-network",
		}, nil)

		// Fetching cluster subnets
		networkClientRecorder.ListSubnet(subnets.ListOpts{
			NetworkID: clusterNetworkID,
		}).Return([]subnets.Subnet{
			{
				ID:   clusterSubnetID,
				Name: "cluster-subnet",
				CIDR: "192.168.0.0/24",
			},
		}, nil)

		// Mock floating IP creation failure
		networkClientRecorder.CreateFloatingIP(gomock.Any()).Return(nil, fmt.Errorf("quota exceeded"))

		err = reconcileNetworkComponents(scope, capiCluster, testCluster)
		Expect(err).ToNot(BeNil())

		// Verify APIEndpointReadyCondition is set to False
		Expect(v1beta1conditions.IsFalse(testCluster, infrav1.APIEndpointReadyCondition)).To(BeTrue())
		condition := v1beta1conditions.Get(testCluster, infrav1.APIEndpointReadyCondition)
		Expect(condition).ToNot(BeNil())
		Expect(condition.Reason).To(Equal(infrav1.APIEndpointConfigFailedReason))
		Expect(condition.Severity).To(Equal(clusterv1beta1.ConditionSeverityError))
		Expect(condition.Message).To(ContainSubstring("Failed to reconcile control plane endpoint"))

		// NetworkReadyCondition and SecurityGroupsReadyCondition should still be True
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.NetworkReadyCondition)).To(BeTrue())
		Expect(v1beta1conditions.IsTrue(testCluster, infrav1.SecurityGroupsReadyCondition)).To(BeTrue())
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

func Test_setClusterNetwork(t *testing.T) {
	openStackCluster := &infrav1.OpenStackCluster{}
	openStackCluster.Status.Network = &infrav1.NetworkStatusWithSubnets{}

	filterednetwork := &networks.Network{
		ID:   "network1",
		Name: "network1",
		Tags: []string{"tag1", "tag2"},
	}

	setClusterNetwork(openStackCluster, filterednetwork)
	expected := infrav1.NetworkStatus{
		ID:   "network1",
		Name: "network1",
		Tags: []string{"tag1", "tag2"},
	}

	if !reflect.DeepEqual(openStackCluster.Status.Network.NetworkStatus, expected) {
		t.Errorf("setClusterNetwork() = %v, want %v", openStackCluster.Status.Network.NetworkStatus, expected)
	}
}

func Test_getAPIServerPort(t *testing.T) {
	tests := []struct {
		name             string
		openStackCluster *infrav1.OpenStackCluster
		want             int32
	}{
		{
			name:             "default",
			openStackCluster: &infrav1.OpenStackCluster{},
			want:             6443,
		},
		{
			name: "with a control plane endpoint",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ControlPlaneEndpoint: &clusterv1beta1.APIEndpoint{
						Host: "192.168.0.1",
						Port: 6444,
					},
				},
			},
			want: 6444,
		},
		{
			name: "with API server port",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerPort: ptr.To(uint16(6445)),
				},
			},
			want: 6445,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getAPIServerPort(tt.openStackCluster); got != tt.want {
				t.Errorf("getAPIServerPort() = %v, want %v", got, tt.want)
			}
		})
	}
}
