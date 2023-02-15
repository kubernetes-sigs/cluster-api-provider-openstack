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
	"testing"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
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

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha6"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/compute"
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
		mockScopeFactory = scope.NewMockScopeFactory(mockCtrl, "", logr.Discard())
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
		scope, err := mockScopeFactory.NewClientScopeFromCluster(ctx, k8sClient, testCluster, nil, logr.Discard())
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

func Test_reconcileFailureDomains(t *testing.T) {
	g := NewWithT(t)
	mockCtrl := gomock.NewController(t)

	tests := []struct {
		name                  string
		novaAvailabilityZones []string
		openStackClusterSpec  infrav1.OpenStackClusterSpec
		want                  clusterv1.FailureDomains
		wantErr               bool
	}{
		{
			name: "Only Nova AZs",
			novaAvailabilityZones: []string{
				"nova-az1",
				"nova-az2",
			},
			openStackClusterSpec: infrav1.OpenStackClusterSpec{},
			want: map[string]clusterv1.FailureDomainSpec{
				"nova-az1": {
					ControlPlane: true,
					Attributes:   map[string]string{"Type": "AvailabilityZone"},
				},
				"nova-az2": {
					ControlPlane: true,
					Attributes:   map[string]string{"Type": "AvailabilityZone"},
				},
			},
			wantErr: false,
		},
		{
			name: "Error fetching Nova AZs",
			novaAvailabilityZones: []string{
				"nova-az1",
				"nova-az2",
			},
			openStackClusterSpec: infrav1.OpenStackClusterSpec{},
			want:                 nil,
			wantErr:              true,
		},
		{
			name: "ControlPlaneAvailabilityZones specified",
			novaAvailabilityZones: []string{
				"nova-az1",
				"nova-az2",
			},
			openStackClusterSpec: infrav1.OpenStackClusterSpec{
				ControlPlaneAvailabilityZones: []string{"nova-az1"},
			},
			want: map[string]clusterv1.FailureDomainSpec{
				"nova-az1": {
					ControlPlane: true,
					Attributes:   map[string]string{"Type": "AvailabilityZone"},
				},
				"nova-az2": {
					ControlPlane: false,
					Attributes:   map[string]string{"Type": "AvailabilityZone"},
				},
			},
			wantErr: false,
		},
		{
			name: "ControlPlaneOmitAvailabilityZone specified",
			novaAvailabilityZones: []string{
				"nova-az1",
				"nova-az2",
			},
			openStackClusterSpec: infrav1.OpenStackClusterSpec{
				ControlPlaneOmitAvailabilityZone: true,
			},
			want: map[string]clusterv1.FailureDomainSpec{
				"nova-az1": {
					ControlPlane: false,
					Attributes:   map[string]string{"Type": "AvailabilityZone"},
				},
				"nova-az2": {
					ControlPlane: false,
					Attributes:   map[string]string{"Type": "AvailabilityZone"},
				},
			},
			wantErr: false,
		},
		{
			name:                  "Only failure domains specified",
			novaAvailabilityZones: []string{},
			openStackClusterSpec: infrav1.OpenStackClusterSpec{
				FailureDomains: []infrav1.FailureDomainDefinition{
					{
						Name:             "fd-control-plane-1",
						MachinePlacement: "All",
						FailureDomain:    infrav1.FailureDomain{},
					},
					{
						Name:             "fd-worker-only",
						MachinePlacement: "NoControlPlane",
						FailureDomain:    infrav1.FailureDomain{},
					},
					{
						Name:             "fd-control-plane-2",
						MachinePlacement: "All",
						FailureDomain:    infrav1.FailureDomain{},
					},
				},
			},
			want: map[string]clusterv1.FailureDomainSpec{
				"fd-control-plane-1": {
					ControlPlane: true,
					Attributes:   map[string]string{"Type": "Cluster"},
				},
				"fd-control-plane-2": {
					ControlPlane: true,
					Attributes:   map[string]string{"Type": "Cluster"},
				},
				"fd-worker-only": {
					ControlPlane: false,
					Attributes:   map[string]string{"Type": "Cluster"},
				},
			},
			wantErr: false,
		},
		{
			name: "Both AZs and failure domains specified",
			novaAvailabilityZones: []string{
				"nova-az1",
				"nova-az2",
			},
			openStackClusterSpec: infrav1.OpenStackClusterSpec{
				ControlPlaneAvailabilityZones: []string{"nova-az1"},
				FailureDomains: []infrav1.FailureDomainDefinition{
					{
						Name:             "fd-control-plane",
						MachinePlacement: "All",
						FailureDomain:    infrav1.FailureDomain{},
					},
					{
						Name:             "fd-worker-only",
						MachinePlacement: "NoControlPlane",
						FailureDomain:    infrav1.FailureDomain{},
					},
				},
			},
			want: map[string]clusterv1.FailureDomainSpec{
				"nova-az1": {
					ControlPlane: true,
					Attributes:   map[string]string{"Type": "AvailabilityZone"},
				},
				"nova-az2": {
					ControlPlane: false,
					Attributes:   map[string]string{"Type": "AvailabilityZone"},
				},
				"fd-control-plane": {
					ControlPlane: true,
					Attributes:   map[string]string{"Type": "Cluster"},
				},
				"fd-worker-only": {
					ControlPlane: false,
					Attributes:   map[string]string{"Type": "Cluster"},
				},
			},
			wantErr: false,
		},
		{
			name: "Override AZ with failure domain",
			novaAvailabilityZones: []string{
				"nova-az1",
				"nova-az2",
			},
			openStackClusterSpec: infrav1.OpenStackClusterSpec{
				FailureDomains: []infrav1.FailureDomainDefinition{
					{
						Name:             "nova-az2",
						MachinePlacement: "All",
						FailureDomain:    infrav1.FailureDomain{},
					},
				},
			},
			want: map[string]clusterv1.FailureDomainSpec{
				"nova-az1": {
					ControlPlane: true,
					Attributes:   map[string]string{"Type": "AvailabilityZone"},
				},
				"nova-az2": {
					ControlPlane: true,
					Attributes:   map[string]string{"Type": "Cluster"},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "", logr.Discard())

			azs := make([]availabilityzones.AvailabilityZone, len(tt.novaAvailabilityZones))
			for i, novaAZ := range tt.novaAvailabilityZones {
				azs[i] = availabilityzones.AvailabilityZone{ZoneName: novaAZ}
			}
			wantErr := func() error {
				if tt.wantErr {
					return fmt.Errorf("error")
				}
				return nil
			}()
			mockScopeFactory.ComputeClient.EXPECT().ListAvailabilityZones().Return(azs, wantErr)

			s, err := compute.NewService(mockScopeFactory)
			g.Expect(err).NotTo(HaveOccurred())

			openStackCluster := &infrav1.OpenStackCluster{
				Spec: tt.openStackClusterSpec,
			}
			got, err := reconcileFailureDomains(s, openStackCluster)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
				g.Expect(got).To(BeNil())
			} else {
				g.Expect(got).To(Equal(tt.want))
				g.Expect(err).NotTo(HaveOccurred())
			}
		})
	}
}
