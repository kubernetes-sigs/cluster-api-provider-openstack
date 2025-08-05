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

package compute

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/google/go-cmp/cmp"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servergroups"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	. "github.com/onsi/gomega" //nolint:revive
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infrav1alpha1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

func Test_ResolveServerSpec(t *testing.T) {
	const (
		serverGroupID1 = "ce96e584-7ebc-46d6-9e55-987d72e3806c"
		imageID1       = "de96e584-7ebc-46d6-9e55-987d72e3806c"
		networkID1     = "23ab8b71-89d4-425f-ac81-4eb83b35125a"
		networkID2     = "cc8f75ce-6ce4-4b8a-836e-e5dac91cc9c8"
		subnetID       = "32dc0e7f-34b6-4544-a69b-248955618736"
		flavorID       = "661c21bc-be52-44e3-9d2e-8d1e11623b59"
	)

	defaultPortSpec := []infrav1.ResolvedPortSpec{
		{
			Name:        "test-instance-0",
			Description: "Created by cluster-api-provider-openstack cluster test-cluster",
			NetworkID:   networkID1,
		},
	}

	defaultPortOpts := []infrav1.PortOpts{
		{
			Network: &infrav1.NetworkParam{
				ID: ptr.To(networkID1),
			},
		},
	}

	tests := []struct {
		testName             string
		spec                 infrav1alpha1.OpenStackServerSpec
		managedSecurityGroup *string
		expectComputeMock    func(m *mock.MockComputeClientMockRecorder)
		expectImageMock      func(m *mock.MockImageClientMockRecorder)
		expectNetworkMock    func(m *mock.MockNetworkClientMockRecorder)
		before               *infrav1alpha1.ResolvedServerSpec
		want                 *infrav1alpha1.ResolvedServerSpec
		wantErr              bool
	}{
		{
			testName: "Resources ID passed",
			spec: infrav1alpha1.OpenStackServerSpec{
				ServerGroup: &infrav1.ServerGroupParam{ID: ptr.To(serverGroupID1)},
				Image:       infrav1.ImageParam{ID: ptr.To(imageID1)},
				FlavorID:    ptr.To(flavorID),
				Ports:       defaultPortOpts,
			},
			want: &infrav1alpha1.ResolvedServerSpec{
				ImageID:       imageID1,
				FlavorID:      flavorID,
				ServerGroupID: serverGroupID1,
				Ports:         defaultPortSpec,
			},
		},
		{
			testName: "Server group by Name not found",
			spec: infrav1alpha1.OpenStackServerSpec{
				ServerGroup: &infrav1.ServerGroupParam{Filter: &infrav1.ServerGroupFilter{Name: ptr.To("test-server-group")}},
				Image:       infrav1.ImageParam{ID: ptr.To(imageID1)},
				FlavorID:    ptr.To(flavorID),
				Ports:       defaultPortOpts,
			},
			want: &infrav1alpha1.ResolvedServerSpec{},
			expectComputeMock: func(m *mock.MockComputeClientMockRecorder) {
				m.ListServerGroups().Return(
					[]servergroups.ServerGroup{},
					nil)
			},
			wantErr: true,
		},
		{
			testName: "Image by Name not found",
			spec: infrav1alpha1.OpenStackServerSpec{
				Image: infrav1.ImageParam{
					Filter: &infrav1.ImageFilter{
						Name: ptr.To("test-image"),
					},
				},
				FlavorID: ptr.To(flavorID),
				Ports:    defaultPortOpts,
			},
			expectImageMock: func(m *mock.MockImageClientMockRecorder) {
				m.ListImages(images.ListOpts{Name: "test-image"}).Return([]images.Image{}, nil)
			},
			want:    &infrav1alpha1.ResolvedServerSpec{},
			wantErr: true,
		},
		{
			testName: "Resolved ports length mismatch",
			spec: infrav1alpha1.OpenStackServerSpec{
				Image:    infrav1.ImageParam{ID: ptr.To(imageID1)},
				FlavorID: ptr.To(flavorID),
				Ports:    slices.Concat(defaultPortOpts, defaultPortOpts),
			},
			want:    &infrav1alpha1.ResolvedServerSpec{},
			wantErr: true,
		},
		{
			testName: "Flavor by Name not found",
			spec: infrav1alpha1.OpenStackServerSpec{
				Image:  infrav1.ImageParam{ID: ptr.To(imageID1)},
				Flavor: ptr.To("foo"),
				Ports:  defaultPortOpts,
			},
			expectComputeMock: func(m *mock.MockComputeClientMockRecorder) {
				m.ListFlavors().Return([]flavors.Flavor{}, nil)
			},
			wantErr: true,
		},
	}
	for i, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			tt := &tests[i]
			g := NewWithT(t)
			log := testr.New(t)
			mockCtrl := gomock.NewController(t)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")

			if tt.expectComputeMock != nil {
				tt.expectComputeMock(mockScopeFactory.ComputeClient.EXPECT())
			}
			if tt.expectImageMock != nil {
				tt.expectImageMock(mockScopeFactory.ImageClient.EXPECT())
			}
			if tt.expectNetworkMock != nil {
				tt.expectNetworkMock(mockScopeFactory.NetworkClient.EXPECT())
			}

			resources := tt.before
			if resources == nil {
				resources = &infrav1alpha1.ResolvedServerSpec{}
			}
			openStackServer := &infrav1alpha1.OpenStackServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-instance",
					Labels: map[string]string{
						clusterv1.ClusterNameLabel: "test-cluster",
					},
				},
				Spec: tt.spec,
				Status: infrav1alpha1.OpenStackServerStatus{
					Resolved: resources,
				},
			}

			fakeClient := fake.NewFakeClient()

			scope := scope.NewWithLogger(mockScopeFactory, log)
			_, _, err := ResolveServerSpec(context.TODO(), scope, fakeClient, openStackServer)
			if tt.wantErr {
				g.Expect(err).Error()
				return
			}

			g.Expect(err).To(BeNil())
			g.Expect(resources).To(Equal(tt.want), cmp.Diff(resources, tt.want))
		})
	}
}

func Test_getInstanceTags(t *testing.T) {
	tests := []struct {
		name             string
		spec             func() *infrav1.OpenStackMachineSpec
		openStackCluster func() *infrav1.OpenStackCluster
		wantMachineTags  []string
	}{
		{
			name: "No tags",
			spec: func() *infrav1.OpenStackMachineSpec {
				return &infrav1.OpenStackMachineSpec{}
			},
			openStackCluster: func() *infrav1.OpenStackCluster {
				return &infrav1.OpenStackCluster{
					Spec: infrav1.OpenStackClusterSpec{},
				}
			},
			wantMachineTags: []string{},
		},
		{
			name: "Machine tags only",
			spec: func() *infrav1.OpenStackMachineSpec {
				return &infrav1.OpenStackMachineSpec{
					Tags: []string{"machine-tag1", "machine-tag2"},
				}
			},
			openStackCluster: func() *infrav1.OpenStackCluster {
				return &infrav1.OpenStackCluster{
					Spec: infrav1.OpenStackClusterSpec{},
				}
			},
			wantMachineTags: []string{"machine-tag1", "machine-tag2"},
		},
		{
			name: "Cluster tags only",
			spec: func() *infrav1.OpenStackMachineSpec {
				return &infrav1.OpenStackMachineSpec{}
			},
			openStackCluster: func() *infrav1.OpenStackCluster {
				return &infrav1.OpenStackCluster{
					Spec: infrav1.OpenStackClusterSpec{
						Tags: []string{"cluster-tag1", "cluster-tag2"},
					},
				}
			},
			wantMachineTags: []string{"cluster-tag1", "cluster-tag2"},
		},
		{
			name: "Machine and cluster tags",
			spec: func() *infrav1.OpenStackMachineSpec {
				return &infrav1.OpenStackMachineSpec{
					Tags: []string{"machine-tag1", "machine-tag2"},
				}
			},
			openStackCluster: func() *infrav1.OpenStackCluster {
				return &infrav1.OpenStackCluster{
					Spec: infrav1.OpenStackClusterSpec{
						Tags: []string{"cluster-tag1", "cluster-tag2"},
					},
				}
			},
			wantMachineTags: []string{"machine-tag1", "machine-tag2", "cluster-tag1", "cluster-tag2"},
		},
		{
			name: "Duplicate tags",
			spec: func() *infrav1.OpenStackMachineSpec {
				return &infrav1.OpenStackMachineSpec{
					Tags: []string{"tag1", "tag2", "tag1"},
				}
			},
			openStackCluster: func() *infrav1.OpenStackCluster {
				return &infrav1.OpenStackCluster{
					Spec: infrav1.OpenStackClusterSpec{
						Tags: []string{"tag2", "tag3", "tag3"},
					},
				}
			},
			wantMachineTags: []string{"tag1", "tag2", "tag3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMachineTags := InstanceTags(tt.spec(), tt.openStackCluster())
			if !reflect.DeepEqual(gotMachineTags, tt.wantMachineTags) {
				t.Errorf("getInstanceTags() = %v, want %v", gotMachineTags, tt.wantMachineTags)
			}
		})
	}
}
