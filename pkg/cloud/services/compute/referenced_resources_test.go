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
	"testing"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/servergroups"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions"
	. "github.com/onsi/gomega"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha8"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

func Test_ResolveReferencedMachineResources(t *testing.T) {
	constFalse := false
	const serverGroupID1 = "ce96e584-7ebc-46d6-9e55-987d72e3806c"
	const imageID1 = "de96e584-7ebc-46d6-9e55-987d72e3806c"

	minimumReferences := &infrav1.ReferencedMachineResources{
		ImageID: imageID1,
	}

	tests := []struct {
		testName          string
		serverGroupFilter *infrav1.ServerGroupFilter
		imageFilter       *infrav1.ImageFilter
		portsOpts         *[]infrav1.PortOpts
		clusterStatus     *infrav1.OpenStackClusterStatus
		expectComputeMock func(m *mock.MockComputeClientMockRecorder)
		expectImageMock   func(m *mock.MockImageClientMockRecorder)
		expectNetworkMock func(m *mock.MockNetworkClientMockRecorder)
		want              *infrav1.ReferencedMachineResources
		wantErr           bool
	}{
		{
			testName:          "Resources ID passed",
			serverGroupFilter: &infrav1.ServerGroupFilter{ID: serverGroupID1},
			imageFilter:       &infrav1.ImageFilter{ID: imageID1},
			expectComputeMock: func(m *mock.MockComputeClientMockRecorder) {},
			expectImageMock:   func(m *mock.MockImageClientMockRecorder) {},
			expectNetworkMock: func(m *mock.MockNetworkClientMockRecorder) {},
			want:              &infrav1.ReferencedMachineResources{ImageID: imageID1, ServerGroupID: serverGroupID1},
			wantErr:           false,
		},
		{
			testName:          "Server group filter nil",
			serverGroupFilter: nil,
			expectComputeMock: func(m *mock.MockComputeClientMockRecorder) {},
			expectImageMock:   func(m *mock.MockImageClientMockRecorder) {},
			expectNetworkMock: func(m *mock.MockNetworkClientMockRecorder) {},
			want:              minimumReferences,
			wantErr:           false,
		},
		{
			testName:          "Server group ID empty",
			serverGroupFilter: &infrav1.ServerGroupFilter{},
			expectComputeMock: func(m *mock.MockComputeClientMockRecorder) {},
			expectImageMock:   func(m *mock.MockImageClientMockRecorder) {},
			expectNetworkMock: func(m *mock.MockNetworkClientMockRecorder) {},
			want:              minimumReferences,
			wantErr:           false,
		},
		{
			testName:          "Server group by Name not found",
			serverGroupFilter: &infrav1.ServerGroupFilter{Name: "test-server-group"},
			expectComputeMock: func(m *mock.MockComputeClientMockRecorder) {
				m.ListServerGroups().Return(
					[]servergroups.ServerGroup{},
					nil)
			},
			expectImageMock:   func(m *mock.MockImageClientMockRecorder) {},
			expectNetworkMock: func(m *mock.MockNetworkClientMockRecorder) {},
			want:              &infrav1.ReferencedMachineResources{},
			wantErr:           true,
		},
		{
			testName:          "Image by Name not found",
			imageFilter:       &infrav1.ImageFilter{Name: "test-image"},
			expectComputeMock: func(m *mock.MockComputeClientMockRecorder) {},
			expectImageMock: func(m *mock.MockImageClientMockRecorder) {
				m.ListImages(images.ListOpts{Name: "test-image"}).Return(
					[]images.Image{},
					nil)
			},
			expectNetworkMock: func(m *mock.MockNetworkClientMockRecorder) {},
			want:              &infrav1.ReferencedMachineResources{},
			wantErr:           true,
		},
		{
			testName: "PortsOpts set",
			clusterStatus: &infrav1.OpenStackClusterStatus{
				Network: &infrav1.NetworkStatusWithSubnets{
					Subnets: []infrav1.Subnet{
						{
							ID: "test-subnet-id",
						},
					},
				},
			},
			portsOpts: &[]infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID: "test-network-id",
					},
					Trunk: &constFalse,
				},
			},
			expectComputeMock: func(m *mock.MockComputeClientMockRecorder) {},
			expectImageMock:   func(m *mock.MockImageClientMockRecorder) {},
			expectNetworkMock: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListExtensions().Return([]extensions.Extension{}, nil)
			},
			want: &infrav1.ReferencedMachineResources{
				ImageID: imageID1,
				PortsOpts: []infrav1.PortOpts{
					{
						Network: &infrav1.NetworkFilter{
							ID: "test-network-id",
						},
						Trunk: &constFalse,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			g := NewWithT(t)
			mockCtrl := gomock.NewController(t)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "", logr.Discard())

			tt.expectComputeMock(mockScopeFactory.ComputeClient.EXPECT())
			tt.expectImageMock(mockScopeFactory.ImageClient.EXPECT())
			tt.expectNetworkMock(mockScopeFactory.NetworkClient.EXPECT())

			// Set defaults for required fields
			imageFilter := &infrav1.ImageFilter{ID: imageID1}
			if tt.imageFilter != nil {
				imageFilter = tt.imageFilter
			}
			portsOpts := &[]infrav1.PortOpts{}
			if tt.portsOpts != nil {
				portsOpts = tt.portsOpts
			}

			openStackCluster := &infrav1.OpenStackCluster{}
			if tt.clusterStatus != nil {
				openStackCluster.Status = *tt.clusterStatus
			}

			machineSpec := &infrav1.OpenStackMachineSpec{
				ServerGroup: tt.serverGroupFilter,
				Image:       *imageFilter,
				Ports:       *portsOpts,
			}

			resources := &infrav1.ReferencedMachineResources{}

			_, err := ResolveReferencedMachineResources(mockScopeFactory, openStackCluster, machineSpec, resources)
			if tt.wantErr {
				g.Expect(err).Error()
				return
			}

			g.Expect(resources).To(Equal(tt.want), cmp.Diff(resources, tt.want))
		})
	}
}
