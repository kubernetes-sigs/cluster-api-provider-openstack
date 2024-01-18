/*
Copyright 2021 The Kubernetes Authors.

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

package networking

import (
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	. "github.com/onsi/gomega"
	"k8s.io/utils/pointer"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha8"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

func Test_ReconcileNetwork(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	clusterName := "test-cluster"
	expectedNetworkName := getNetworkName(clusterName)
	fakeNetworkID := "d08803fc-2fa5-4179-b9f7-8c43d0af2fe6"

	tests := []struct {
		name             string
		openStackCluster *infrav1.OpenStackCluster
		expect           func(m *mock.MockNetworkClientMockRecorder)
		want             *infrav1.OpenStackCluster
	}{
		{
			name: "ensures status set when reconciling an existing network",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListNetwork(networks.ListOpts{Name: expectedNetworkName}).
					Return([]networks.Network{
						{
							ID:   fakeNetworkID,
							Name: expectedNetworkName,
						},
					}, nil)
			},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						NetworkStatus: infrav1.NetworkStatus{
							ID:   fakeNetworkID,
							Name: expectedNetworkName,
							Tags: []string{},
						},
					},
				},
			},
		},
		{
			name: "creation without any parameter",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListNetwork(networks.ListOpts{Name: expectedNetworkName}).
					Return([]networks.Network{}, nil)

				m.
					CreateNetwork(createOpts{
						AdminStateUp: gophercloud.Enabled,
						Name:         expectedNetworkName,
					}).
					Return(&networks.Network{
						ID:   fakeNetworkID,
						Name: expectedNetworkName,
					}, nil)
			},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						NetworkStatus: infrav1.NetworkStatus{
							ID:   fakeNetworkID,
							Name: expectedNetworkName,
							Tags: []string{},
						},
					},
				},
			},
		},
		{
			name: "creation with disabled port security",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					DisablePortSecurity: true,
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListNetwork(networks.ListOpts{Name: expectedNetworkName}).
					Return([]networks.Network{}, nil)

				m.
					CreateNetwork(createOpts{
						AdminStateUp:        gophercloud.Enabled,
						Name:                expectedNetworkName,
						PortSecurityEnabled: gophercloud.Disabled,
					}).
					Return(&networks.Network{
						ID:   fakeNetworkID,
						Name: expectedNetworkName,
					}, nil)
			},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						NetworkStatus: infrav1.NetworkStatus{
							ID:   fakeNetworkID,
							Name: expectedNetworkName,
							Tags: []string{},
						},
					},
				},
			},
		},
		{
			name: "creation with mtu set",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					NetworkMTU: 1500,
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListNetwork(networks.ListOpts{Name: expectedNetworkName}).
					Return([]networks.Network{}, nil)

				m.
					CreateNetwork(createOpts{
						AdminStateUp: gophercloud.Enabled,
						Name:         expectedNetworkName,
						MTU:          pointer.Int(1500),
					}).
					Return(&networks.Network{
						ID:   fakeNetworkID,
						Name: expectedNetworkName,
					}, nil)
			},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						NetworkStatus: infrav1.NetworkStatus{
							ID:   fakeNetworkID,
							Name: expectedNetworkName,
							Tags: []string{},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			mockClient := mock.NewMockNetworkClient(mockCtrl)
			tt.expect(mockClient.EXPECT())
			s := Service{
				client: mockClient,
				scope:  scope.NewMockScopeFactory(mockCtrl, "", logr.Discard()),
			}
			err := s.ReconcileNetwork(tt.openStackCluster, clusterName)
			g.Expect(err).ShouldNot(HaveOccurred())
		})
	}
}

func Test_ReconcileExternalNetwork(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	fakeNetworkID := "d08803fc-2fa5-4179-b9f7-8c43d0af2fe6"
	fakeNetworkname := "external-network"
	isAutodetecting := true

	tests := []struct {
		name             string
		openStackCluster *infrav1.OpenStackCluster
		expect           func(m *mock.MockNetworkClientMockRecorder)
		want             *infrav1.OpenStackCluster
		wantErr          bool
	}{
		{
			name: "reconcile external network by ID",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ExternalNetwork: infrav1.NetworkFilter{
						ID: fakeNetworkID,
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListNetwork(external.ListOptsExt{
						ListOptsBuilder: networks.ListOpts{ID: fakeNetworkID},
					}).
					Return([]networks.Network{
						{
							ID:   fakeNetworkID,
							Name: fakeNetworkname,
						},
					}, nil)
			},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ExternalNetwork: infrav1.NetworkFilter{
						ID: fakeNetworkID,
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					ExternalNetwork: &infrav1.NetworkStatus{
						ID:   fakeNetworkID,
						Name: fakeNetworkname,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "reconcile external network by name",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ExternalNetwork: infrav1.NetworkFilter{
						Name: fakeNetworkname,
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListNetwork(external.ListOptsExt{
						ListOptsBuilder: networks.ListOpts{Name: fakeNetworkname},
					}).
					Return([]networks.Network{
						{
							ID:   fakeNetworkID,
							Name: fakeNetworkname,
						},
					}, nil)
			},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ExternalNetwork: infrav1.NetworkFilter{
						Name: fakeNetworkname,
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					ExternalNetwork: &infrav1.NetworkStatus{
						ID:   fakeNetworkID,
						Name: fakeNetworkname,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "reconcile external network by ID when no external network found",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ExternalNetwork: infrav1.NetworkFilter{
						ID: fakeNetworkID,
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListNetwork(external.ListOptsExt{
						ListOptsBuilder: networks.ListOpts{ID: fakeNetworkID},
					}).
					Return([]networks.Network{}, nil)
			},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ExternalNetwork: infrav1.NetworkFilter{
						ID: fakeNetworkID,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "not reconcile external network when external network disabled",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					DisableExternalNetwork: true,
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					DisableExternalNetwork: true,
				},
				Status: infrav1.OpenStackClusterStatus{
					ExternalNetwork: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "reconcile external network with no filter when zero external network found",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListNetwork(external.ListOptsExt{
						ListOptsBuilder: networks.ListOpts{},
						External:        &isAutodetecting,
					}).
					Return([]networks.Network{}, nil)
			},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{},
				Status: infrav1.OpenStackClusterStatus{
					ExternalNetwork: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "reconcile external network with no filter when one external network found",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListNetwork(external.ListOptsExt{
						ListOptsBuilder: networks.ListOpts{},
						External:        &isAutodetecting,
					}).
					Return([]networks.Network{
						{
							ID:   fakeNetworkID,
							Name: fakeNetworkname,
						},
					}, nil)
			},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{},
				Status: infrav1.OpenStackClusterStatus{
					ExternalNetwork: &infrav1.NetworkStatus{
						ID:   fakeNetworkID,
						Name: fakeNetworkname,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "reconcile external network with no filter when more than one external network found",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListNetwork(external.ListOptsExt{
						ListOptsBuilder: networks.ListOpts{},
						External:        &isAutodetecting,
					}).
					Return([]networks.Network{
						{
							ID:   fakeNetworkID,
							Name: fakeNetworkname,
						},
						{
							ID:   "d08803fc-2fa5-4179-b9f7-8c43d0af2fe7",
							Name: "external-network-2",
						},
					}, nil)
			},
			want:    &infrav1.OpenStackCluster{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			mockClient := mock.NewMockNetworkClient(mockCtrl)
			tt.expect(mockClient.EXPECT())
			s := Service{
				client: mockClient,
				scope:  scope.NewMockScopeFactory(mockCtrl, "", logr.Discard()),
			}
			err := s.ReconcileExternalNetwork(tt.openStackCluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileExternalNetwork() error = %v, wantErr %v", err, tt.wantErr)
			}
			g.Expect(tt.openStackCluster).To(Equal(tt.want))
		})
	}
}

func Test_ConvertOpenStackSubnetToCAPOSubnet(t *testing.T) {
	caposubnets := []infrav1.Subnet{
		{
			ID:   "subnet1",
			Name: "subnet1",
			CIDR: "10.0.0.0/24",
			Tags: []string{"tag1", "tag2"},
		},
	}

	filteredSubnet := &subnets.Subnet{
		ID:   "subnet2",
		Name: "subnet2",
		CIDR: "192.168.0.0/24",
		Tags: []string{"tag3", "tag4"},
	}

	s := Service{}
	result := s.ConvertOpenStackSubnetToCAPOSubnet(caposubnets, filteredSubnet)

	expected := []infrav1.Subnet{
		{
			ID:   "subnet1",
			Name: "subnet1",
			CIDR: "10.0.0.0/24",
			Tags: []string{"tag1", "tag2"},
		},
		{
			ID:   "subnet2",
			Name: "subnet2",
			CIDR: "192.168.0.0/24",
			Tags: []string{"tag3", "tag4"},
		},
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ConvertOpenStackSubnetToCAPOSubnet() = %v, want %v", result, expected)
	}
}
