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

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
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
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

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
		testCluster.Spec = infrav1.OpenStackClusterSpec{}
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
	})

	It("should allow two subnets for the cluster network", func() {
		const externalNetworkID = "a42211a2-4d2c-426f-9413-830e4b4abbbc"
		const clusterNetworkID = "6c90b532-7ba0-418a-a276-5ae55060b5b0"
		clusterSubnets := []string{"cad5a91a-36de-4388-823b-b0cc82cadfdc", "e2407c18-c4e7-4d3d-befa-8eec5d8756f2"}

		testCluster.SetName("subnet-filtering")
		testCluster.Spec = infrav1.OpenStackClusterSpec{
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
	})

	It("should allow fetch network by subnet", func() {
		const clusterNetworkID = "6c90b532-7ba0-418a-a276-5ae55060b5b0"
		const clusterSubnetID = "cad5a91a-36de-4388-823b-b0cc82cadfdc"

		testCluster.SetName("subnet-filtering")
		testCluster.Spec = infrav1.OpenStackClusterSpec{
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
	})

	It("reconcile pre-existing network components by id", func() {
		const clusterNetworkID = "6c90b532-7ba0-418a-a276-5ae55060b5b0"
		const clusterSubnetID = "cad5a91a-36de-4388-823b-b0cc82cadfdc"
		const clusterRouterID = "e2407c18-c4e7-4d3d-befa-8eec5d8756f2"

		testCluster.SetName("pre-existing-network-components-by-id")
		testCluster.Spec = infrav1.OpenStackClusterSpec{
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
