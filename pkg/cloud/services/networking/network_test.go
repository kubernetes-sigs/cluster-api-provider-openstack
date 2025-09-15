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
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/google/go-cmp/cmp"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	. "github.com/onsi/gomega" //nolint:revive
	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/names"
)

const (
	clusterResourceName = "test-cluster"
)

func Test_updateSubnetDNSNameservers(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	const subnetID = "subnet-123"

	tests := []struct {
		name               string
		currentNameservers []string
		desiredNameservers []string
		expect             func(m *mock.MockNetworkClientMockRecorder)
	}{
		{
			name:               "no changes needed",
			currentNameservers: []string{"8.8.8.8", "8.8.4.4"},
			desiredNameservers: []string{"8.8.8.8", "8.8.4.4"},
			expect:             func(*mock.MockNetworkClientMockRecorder) {},
		},
		{
			name:               "different nameservers",
			currentNameservers: []string{"8.8.8.8", "8.8.4.4"},
			desiredNameservers: []string{"1.1.1.1", "1.0.0.1"},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.UpdateSubnet(subnetID, subnets.UpdateOpts{
					DNSNameservers: &[]string{"1.1.1.1", "1.0.0.1"},
				}).Return(&subnets.Subnet{
					ID:             subnetID,
					DNSNameservers: []string{"1.1.1.1", "1.0.0.1"},
				}, nil)
			},
		},
		{
			name:               "different count",
			currentNameservers: []string{"8.8.8.8"},
			desiredNameservers: []string{"8.8.8.8", "8.8.4.4"},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.UpdateSubnet(subnetID, subnets.UpdateOpts{
					DNSNameservers: &[]string{"8.8.8.8", "8.8.4.4"},
				}).Return(&subnets.Subnet{
					ID:             subnetID,
					DNSNameservers: []string{"8.8.8.8", "8.8.4.4"},
				}, nil)
			},
		},
		{
			name:               "same nameservers but different order",
			currentNameservers: []string{"8.8.8.8", "8.8.4.4"},
			desiredNameservers: []string{"8.8.4.4", "8.8.8.8"},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.UpdateSubnet(subnetID, subnets.UpdateOpts{
					DNSNameservers: &[]string{"8.8.4.4", "8.8.8.8"},
				}).Return(&subnets.Subnet{
					ID:             subnetID,
					DNSNameservers: []string{"8.8.4.4", "8.8.8.8"},
				}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			mockClient := mock.NewMockNetworkClient(mockCtrl)
			tt.expect(mockClient.EXPECT())

			cluster := &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ManagedSubnets: []infrav1.SubnetSpec{
						{
							DNSNameservers: tt.desiredNameservers,
						},
					},
				},
			}
			subnet := &subnets.Subnet{
				ID:             subnetID,
				DNSNameservers: tt.currentNameservers,
			}

			scopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			log := testr.New(t)
			s := Service{
				client: mockClient,
				scope:  scope.NewWithLogger(scopeFactory, log),
			}

			err := s.updateSubnetDNSNameservers(cluster, subnet)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(subnet.DNSNameservers).To(Equal(tt.desiredNameservers))
		})
	}
}

func Test_ReconcileNetwork(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNetworkName := getNetworkName(clusterResourceName)
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
					DisablePortSecurity: ptr.To(true),
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
					NetworkMTU: ptr.To(1500),
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
						MTU:          ptr.To(1500),
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

			scopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			log := testr.New(t)
			s := Service{
				client: mockClient,
				scope:  scope.NewWithLogger(scopeFactory, log),
			}
			err := s.ReconcileNetwork(tt.openStackCluster, clusterResourceName)
			g.Expect(err).ShouldNot(HaveOccurred())
		})
	}
}

func Test_ReconcileExternalNetwork(t *testing.T) {
	fakeNetworkID := "d08803fc-2fa5-4179-b9f7-8c43d0af2fe6"
	fakeNetworkname := "external-network"

	// Use gomega to match the ListOptsBuilder argument
	getExternalNetwork := func(g Gomega, listOpts networks.ListOpts, ret []networks.Network) func(networks.ListOptsBuilder) ([]networks.Network, error) {
		return func(opts networks.ListOptsBuilder) ([]networks.Network, error) {
			expected := &external.ListOptsExt{
				ListOptsBuilder: listOpts,
				External:        ptr.To(true),
			}
			g.Expect(opts).To(Equal(expected), cmp.Diff(opts, expected))

			return ret, nil
		}
	}

	tests := []struct {
		name             string
		openStackCluster *infrav1.OpenStackCluster
		expect           func(g Gomega, m *mock.MockNetworkClientMockRecorder)
		want             *infrav1.OpenStackCluster
		wantErr          bool
	}{
		{
			name: "reconcile external network by ID",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ExternalNetwork: &infrav1.NetworkParam{
						ID: ptr.To(fakeNetworkID),
					},
				},
			},
			expect: func(_ Gomega, m *mock.MockNetworkClientMockRecorder) {
				m.GetNetwork(fakeNetworkID).Return(&networks.Network{
					ID:   fakeNetworkID,
					Name: fakeNetworkname,
				}, nil)
			},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ExternalNetwork: &infrav1.NetworkParam{
						ID: ptr.To(fakeNetworkID),
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
					ExternalNetwork: &infrav1.NetworkParam{
						Filter: &infrav1.NetworkFilter{Name: fakeNetworkname},
					},
				},
			},
			expect: func(g Gomega, m *mock.MockNetworkClientMockRecorder) {
				m.ListNetwork(gomock.Any()).
					DoAndReturn(getExternalNetwork(g, networks.ListOpts{Name: fakeNetworkname}, []networks.Network{
						{
							ID:   fakeNetworkID,
							Name: fakeNetworkname,
						},
					}))
			},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ExternalNetwork: &infrav1.NetworkParam{
						Filter: &infrav1.NetworkFilter{Name: fakeNetworkname},
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
			name: "reconcile external network by ID when external network by id not found",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ExternalNetwork: &infrav1.NetworkParam{
						ID: ptr.To(fakeNetworkID),
					},
				},
			},
			expect: func(_ Gomega, m *mock.MockNetworkClientMockRecorder) {
				m.GetNetwork(fakeNetworkID).Return(nil, gophercloud.ErrUnexpectedResponseCode{Actual: 404})
			},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ExternalNetwork: &infrav1.NetworkParam{
						ID: ptr.To(fakeNetworkID),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "reconcile external network by ID when external network by name not found",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ExternalNetwork: &infrav1.NetworkParam{
						Filter: &infrav1.NetworkFilter{Name: fakeNetworkname},
					},
				},
			},
			expect: func(g Gomega, m *mock.MockNetworkClientMockRecorder) {
				m.ListNetwork(gomock.Any()).
					DoAndReturn(getExternalNetwork(g, networks.ListOpts{Name: fakeNetworkname}, []networks.Network{}))
			},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ExternalNetwork: &infrav1.NetworkParam{
						Filter: &infrav1.NetworkFilter{Name: fakeNetworkname},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "not reconcile external network when external network disabled",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					DisableExternalNetwork: ptr.To(true),
				},
			},
			expect: func(Gomega, *mock.MockNetworkClientMockRecorder) {},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					DisableExternalNetwork: ptr.To(true),
				},
				Status: infrav1.OpenStackClusterStatus{
					ExternalNetwork: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "reconcile external network with no filter when zero external networks found",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{},
			},
			expect: func(g Gomega, m *mock.MockNetworkClientMockRecorder) {
				m.ListNetwork(gomock.Any()).
					DoAndReturn(getExternalNetwork(g, networks.ListOpts{}, []networks.Network{}))
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
			expect: func(g Gomega, m *mock.MockNetworkClientMockRecorder) {
				m.ListNetwork(gomock.Any()).
					DoAndReturn(getExternalNetwork(g, networks.ListOpts{}, []networks.Network{
						{
							ID:   fakeNetworkID,
							Name: fakeNetworkname,
						},
					}))
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
			expect: func(g Gomega, m *mock.MockNetworkClientMockRecorder) {
				m.
					ListNetwork(gomock.Any()).
					DoAndReturn(func(opts networks.ListOptsBuilder) ([]networks.Network, error) {
						expected := &external.ListOptsExt{
							ListOptsBuilder: networks.ListOpts{},
							External:        ptr.To(true),
						}
						g.Expect(opts).To(Equal(expected), cmp.Diff(opts, expected))

						return []networks.Network{
							{
								ID:   fakeNetworkID,
								Name: fakeNetworkname,
							},
							{
								ID:   "d08803fc-2fa5-4179-b9f7-8c43d0af2fe7",
								Name: "external-network-2",
							},
						}, nil
					})
			},
			want:    &infrav1.OpenStackCluster{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			g := NewWithT(t)
			mockClient := mock.NewMockNetworkClient(mockCtrl)
			tt.expect(g, mockClient.EXPECT())

			scopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			log := testr.New(t)
			s := Service{
				client: mockClient,
				scope:  scope.NewWithLogger(scopeFactory, log),
			}
			err := s.ReconcileExternalNetwork(tt.openStackCluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileExternalNetwork() error = %v, wantErr %v", err, tt.wantErr)
			}
			g.Expect(tt.openStackCluster).To(Equal(tt.want), cmp.Diff(tt.openStackCluster, tt.want))
		})
	}
}

func Test_ReconcileSubnet(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedSubnetName := getSubnetName(clusterResourceName)
	expectedSubnetDesc := names.GetDescription(clusterResourceName)
	fakeSubnetID := "d08803fc-2fa5-4179-b9d7-8c43d0af2fe6"
	fakeCIDR := "10.0.0.0/24"
	fakeNetworkID := "d08803fc-2fa5-4279-b9f7-8c45d0ff2fe6"
	fakeDNS1 := "10.0.10.200"
	fakeDNS2 := "10.0.10.201"

	tests := []struct {
		name             string
		openStackCluster *infrav1.OpenStackCluster
		expect           func(m *mock.MockNetworkClientMockRecorder)
		want             *infrav1.OpenStackClusterStatus
	}{
		{
			name: "ensures status set when reconciling an existing subnet",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ManagedSubnets: []infrav1.SubnetSpec{
						{
							CIDR: fakeCIDR,
						},
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						NetworkStatus: infrav1.NetworkStatus{
							ID: fakeNetworkID,
						},
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListSubnet(subnets.ListOpts{NetworkID: fakeNetworkID, CIDR: fakeCIDR}).
					Return([]subnets.Subnet{
						{
							ID:   fakeSubnetID,
							Name: expectedSubnetName,
							CIDR: fakeCIDR,
						},
					}, nil)
			},
			want: &infrav1.OpenStackClusterStatus{
				Network: &infrav1.NetworkStatusWithSubnets{
					NetworkStatus: infrav1.NetworkStatus{
						ID: fakeNetworkID,
					},
					Subnets: []infrav1.Subnet{
						{
							Name: expectedSubnetName,
							ID:   fakeSubnetID,
							CIDR: fakeCIDR,
						},
					},
				},
			},
		},
		{
			name: "creation without any parameter",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ManagedSubnets: []infrav1.SubnetSpec{
						{
							CIDR: fakeCIDR,
						},
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						NetworkStatus: infrav1.NetworkStatus{
							ID: fakeNetworkID,
						},
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListSubnet(subnets.ListOpts{NetworkID: fakeNetworkID, CIDR: fakeCIDR}).
					Return([]subnets.Subnet{}, nil)

				m.
					CreateSubnet(subnets.CreateOpts{
						NetworkID:   fakeNetworkID,
						Name:        expectedSubnetName,
						IPVersion:   4,
						CIDR:        fakeCIDR,
						Description: expectedSubnetDesc,
					}).
					Return(&subnets.Subnet{
						ID:   fakeSubnetID,
						Name: expectedSubnetName,
						CIDR: fakeCIDR,
					}, nil)
			},
			want: &infrav1.OpenStackClusterStatus{
				Network: &infrav1.NetworkStatusWithSubnets{
					NetworkStatus: infrav1.NetworkStatus{
						ID: fakeNetworkID,
					},
					Subnets: []infrav1.Subnet{
						{
							Name: expectedSubnetName,
							ID:   fakeSubnetID,
							CIDR: fakeCIDR,
						},
					},
				},
			},
		},
		{
			name: "creation with DNSNameservers",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ManagedSubnets: []infrav1.SubnetSpec{
						{
							CIDR:           fakeCIDR,
							DNSNameservers: []string{fakeDNS1},
						},
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						NetworkStatus: infrav1.NetworkStatus{
							ID: fakeNetworkID,
						},
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListSubnet(subnets.ListOpts{NetworkID: fakeNetworkID, CIDR: fakeCIDR}).
					Return([]subnets.Subnet{}, nil)

				m.
					CreateSubnet(subnets.CreateOpts{
						NetworkID:      fakeNetworkID,
						Name:           expectedSubnetName,
						IPVersion:      4,
						CIDR:           fakeCIDR,
						Description:    expectedSubnetDesc,
						DNSNameservers: []string{fakeDNS1},
					}).
					Return(&subnets.Subnet{
						ID:   fakeSubnetID,
						Name: expectedSubnetName,
						CIDR: fakeCIDR,
					}, nil)
			},
			want: &infrav1.OpenStackClusterStatus{
				Network: &infrav1.NetworkStatusWithSubnets{
					NetworkStatus: infrav1.NetworkStatus{
						ID: fakeNetworkID,
					},
					Subnets: []infrav1.Subnet{
						{
							Name: expectedSubnetName,
							ID:   fakeSubnetID,
							CIDR: fakeCIDR,
						},
					},
				},
			},
		},
		{
			name: "creation with allocationPools",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ManagedSubnets: []infrav1.SubnetSpec{
						{
							CIDR: fakeCIDR,
							AllocationPools: []infrav1.AllocationPool{
								{
									Start: "10.0.0.1",
									End:   "10.0.0.10",
								},
								{
									Start: "10.0.0.20",
									End:   "10.0.0.254",
								},
							},
						},
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						NetworkStatus: infrav1.NetworkStatus{
							ID: fakeNetworkID,
						},
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListSubnet(subnets.ListOpts{NetworkID: fakeNetworkID, CIDR: fakeCIDR}).
					Return([]subnets.Subnet{}, nil)

				m.
					CreateSubnet(subnets.CreateOpts{
						NetworkID:   fakeNetworkID,
						Name:        expectedSubnetName,
						IPVersion:   4,
						CIDR:        fakeCIDR,
						Description: expectedSubnetDesc,
						AllocationPools: []subnets.AllocationPool{
							{
								Start: "10.0.0.1",
								End:   "10.0.0.10",
							},
							{
								Start: "10.0.0.20",
								End:   "10.0.0.254",
							},
						},
					}).
					Return(&subnets.Subnet{
						ID:   fakeSubnetID,
						Name: expectedSubnetName,
						CIDR: fakeCIDR,
					}, nil)
			},
			want: &infrav1.OpenStackClusterStatus{
				Network: &infrav1.NetworkStatusWithSubnets{
					NetworkStatus: infrav1.NetworkStatus{
						ID: fakeNetworkID,
					},
					Subnets: []infrav1.Subnet{
						{
							Name: expectedSubnetName,
							ID:   fakeSubnetID,
							CIDR: fakeCIDR,
						},
					},
				},
			},
		},
		{
			name: "existing subnet with different DNS nameservers - update needed",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ManagedSubnets: []infrav1.SubnetSpec{
						{
							CIDR:           fakeCIDR,
							DNSNameservers: []string{fakeDNS1},
						},
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						NetworkStatus: infrav1.NetworkStatus{
							ID: fakeNetworkID,
						},
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListSubnet(subnets.ListOpts{NetworkID: fakeNetworkID, CIDR: fakeCIDR}).
					Return([]subnets.Subnet{
						{
							ID:             fakeSubnetID,
							Name:           expectedSubnetName,
							CIDR:           fakeCIDR,
							DNSNameservers: []string{fakeDNS2},
						},
					}, nil)

				updateOpts := subnets.UpdateOpts{
					DNSNameservers: &[]string{fakeDNS1},
				}

				m.UpdateSubnet(fakeSubnetID, updateOpts).
					Return(&subnets.Subnet{
						ID:             fakeSubnetID,
						Name:           expectedSubnetName,
						CIDR:           fakeCIDR,
						DNSNameservers: []string{fakeDNS1},
					}, nil).
					Times(1)
			},
			want: &infrav1.OpenStackClusterStatus{
				Network: &infrav1.NetworkStatusWithSubnets{
					NetworkStatus: infrav1.NetworkStatus{
						ID: fakeNetworkID,
					},
					Subnets: []infrav1.Subnet{
						{
							Name: expectedSubnetName,
							ID:   fakeSubnetID,
							CIDR: fakeCIDR,
						},
					},
				},
			},
		},
		{
			name: "existing subnet with same DNS nameservers - no update needed",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ManagedSubnets: []infrav1.SubnetSpec{
						{
							CIDR:           fakeCIDR,
							DNSNameservers: []string{fakeDNS1},
						},
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						NetworkStatus: infrav1.NetworkStatus{
							ID: fakeNetworkID,
						},
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListSubnet(subnets.ListOpts{NetworkID: fakeNetworkID, CIDR: fakeCIDR}).
					Return([]subnets.Subnet{
						{
							ID:             fakeSubnetID,
							Name:           expectedSubnetName,
							CIDR:           fakeCIDR,
							DNSNameservers: []string{fakeDNS1},
						},
					}, nil)
			},
			want: &infrav1.OpenStackClusterStatus{
				Network: &infrav1.NetworkStatusWithSubnets{
					NetworkStatus: infrav1.NetworkStatus{
						ID: fakeNetworkID,
					},
					Subnets: []infrav1.Subnet{
						{
							Name: expectedSubnetName,
							ID:   fakeSubnetID,
							CIDR: fakeCIDR,
						},
					},
				},
			},
		},
		{
			name: "existing subnet with multiple different DNS nameservers - update needed",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ManagedSubnets: []infrav1.SubnetSpec{
						{
							CIDR:           fakeCIDR,
							DNSNameservers: []string{fakeDNS1, fakeDNS2},
						},
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						NetworkStatus: infrav1.NetworkStatus{
							ID: fakeNetworkID,
						},
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListSubnet(subnets.ListOpts{NetworkID: fakeNetworkID, CIDR: fakeCIDR}).
					Return([]subnets.Subnet{
						{
							ID:             fakeSubnetID,
							Name:           expectedSubnetName,
							CIDR:           fakeCIDR,
							DNSNameservers: []string{fakeDNS1},
						},
					}, nil)

				updateOpts := subnets.UpdateOpts{
					DNSNameservers: &[]string{fakeDNS1, fakeDNS2},
				}

				m.UpdateSubnet(fakeSubnetID, updateOpts).
					Return(&subnets.Subnet{
						ID:             fakeSubnetID,
						Name:           expectedSubnetName,
						CIDR:           fakeCIDR,
						DNSNameservers: []string{fakeDNS1, fakeDNS2},
					}, nil).
					Times(1)
			},
			want: &infrav1.OpenStackClusterStatus{
				Network: &infrav1.NetworkStatusWithSubnets{
					NetworkStatus: infrav1.NetworkStatus{
						ID: fakeNetworkID,
					},
					Subnets: []infrav1.Subnet{
						{
							Name: expectedSubnetName,
							ID:   fakeSubnetID,
							CIDR: fakeCIDR,
						},
					},
				},
			},
		},
		{
			name: "existing subnet with multiple (invert) different DNS nameservers - update needed",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ManagedSubnets: []infrav1.SubnetSpec{
						{
							CIDR:           fakeCIDR,
							DNSNameservers: []string{fakeDNS2, fakeDNS1},
						},
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						NetworkStatus: infrav1.NetworkStatus{
							ID: fakeNetworkID,
						},
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListSubnet(subnets.ListOpts{NetworkID: fakeNetworkID, CIDR: fakeCIDR}).
					Return([]subnets.Subnet{
						{
							ID:             fakeSubnetID,
							Name:           expectedSubnetName,
							CIDR:           fakeCIDR,
							DNSNameservers: []string{fakeDNS2},
						},
					}, nil)

				updateOpts := subnets.UpdateOpts{
					DNSNameservers: &[]string{fakeDNS2, fakeDNS1},
				}

				m.UpdateSubnet(fakeSubnetID, updateOpts).
					Return(&subnets.Subnet{
						ID:             fakeSubnetID,
						Name:           expectedSubnetName,
						CIDR:           fakeCIDR,
						DNSNameservers: []string{fakeDNS2, fakeDNS1},
					}, nil).
					Times(1)
			},
			want: &infrav1.OpenStackClusterStatus{
				Network: &infrav1.NetworkStatusWithSubnets{
					NetworkStatus: infrav1.NetworkStatus{
						ID: fakeNetworkID,
					},
					Subnets: []infrav1.Subnet{
						{
							Name: expectedSubnetName,
							ID:   fakeSubnetID,
							CIDR: fakeCIDR,
						},
					},
				},
			},
		},
		{
			name: "existing subnet with no DNS nameservers initially - add DNS nameservers",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ManagedSubnets: []infrav1.SubnetSpec{
						{
							CIDR:           fakeCIDR,
							DNSNameservers: []string{fakeDNS1},
						},
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						NetworkStatus: infrav1.NetworkStatus{
							ID: fakeNetworkID,
						},
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListSubnet(subnets.ListOpts{NetworkID: fakeNetworkID, CIDR: fakeCIDR}).
					Return([]subnets.Subnet{
						{
							ID:             fakeSubnetID,
							Name:           expectedSubnetName,
							CIDR:           fakeCIDR,
							DNSNameservers: []string{},
						},
					}, nil)

				updateOpts := subnets.UpdateOpts{
					DNSNameservers: &[]string{fakeDNS1},
				}

				m.UpdateSubnet(fakeSubnetID, updateOpts).
					Return(&subnets.Subnet{
						ID:             fakeSubnetID,
						Name:           expectedSubnetName,
						CIDR:           fakeCIDR,
						DNSNameservers: []string{fakeDNS1},
					}, nil).
					Times(1)
			},
			want: &infrav1.OpenStackClusterStatus{
				Network: &infrav1.NetworkStatusWithSubnets{
					NetworkStatus: infrav1.NetworkStatus{
						ID: fakeNetworkID,
					},
					Subnets: []infrav1.Subnet{
						{
							Name: expectedSubnetName,
							ID:   fakeSubnetID,
							CIDR: fakeCIDR,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			log := testr.New(t)
			mockClient := mock.NewMockNetworkClient(mockCtrl)
			tt.expect(mockClient.EXPECT())
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			s := Service{
				client: mockClient,
				scope:  scope.NewWithLogger(mockScopeFactory, log),
			}
			err := s.ReconcileSubnet(tt.openStackCluster, clusterResourceName)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(tt.openStackCluster.Status).To(Equal(*tt.want))
		})
	}
}
