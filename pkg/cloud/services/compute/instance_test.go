/*
Copyright 2018 The Kubernetes Authors.

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
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	. "github.com/onsi/gomega"
	"k8s.io/utils/pointer"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha4"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking/mock_networking"
)

func Test_getPortName(t *testing.T) {
	type args struct {
		instanceName string
		opts         *infrav1.PortOpts
		netIndex     int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "with nil PortOpts",
			args: args{"test-1-instance", nil, 2},
			want: "test-1-instance-2",
		},
		{
			name: "with PortOpts name suffix",
			args: args{"test-1-instance", &infrav1.PortOpts{NameSuffix: "foo"}, 4},
			want: "test-1-instance-foo",
		},
		{
			name: "without PortOpts name suffix",
			args: args{"test-1-instance", &infrav1.PortOpts{}, 4},
			want: "test-1-instance-4",
		},
		{
			name: "with PortOpts name suffix",
			args: args{"test-1-instance", &infrav1.PortOpts{NameSuffix: "foo2", NetworkID: "bar", DisablePortSecurity: pointer.Bool(true)}, 4},
			want: "test-1-instance-foo2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPortName(tt.args.instanceName, tt.args.opts, tt.args.netIndex); got != tt.want {
				t.Errorf("getPortName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_getServerNetworks(t *testing.T) {
	const testClusterTag = "cluster=mycluster"

	// Network A:
	//  Network is tagged
	//  Has 3 subnets
	//  Subnets A1 and A2 are tagged
	//  Subnet A3 is not tagged
	// Network B:
	//  Network is tagged
	//  Has 1 subnet, B1, which is also tagged
	// Network C:
	//  Network is not tagged
	//  Has 1 subnet, C1, which is also not tagged

	networkAUUID := "7f0a7cc9-d7c8-41d2-87a2-2fc7f5ec544e"
	networkBUUID := "607559d9-a5a4-4a0b-a92d-75eba89e3343"
	networkCUUID := "9d7b0284-b22e-4bc7-b90e-28a652cac7cc"
	subnetA1UUID := "869f6790-17a9-44d5-83a1-89e180514515"
	subnetA2UUID := "bd926900-5277-47a5-bd71-c6f713165dbd"
	subnetA3UUID := "79dfde1b-07f1-48a0-97fd-07e2f6018c46"
	subnetB1UUID := "efc2cc7d-c6e0-45c6-8147-0e08b8530664"
	subnetC1UUID := "b33271f4-6bb1-430a-88bf-789394815aaf"

	testNetworkA := networks.Network{
		ID:      networkAUUID,
		Name:    "network-a",
		Subnets: []string{subnetA1UUID, subnetA2UUID},
		Tags:    []string{testClusterTag},
	}
	testNetworkB := networks.Network{
		ID:      networkBUUID,
		Name:    "network-b",
		Subnets: []string{subnetB1UUID},
		Tags:    []string{testClusterTag},
	}

	testSubnetA1 := subnets.Subnet{
		ID:        subnetA1UUID,
		Name:      "subnet-a1",
		NetworkID: networkAUUID,
		Tags:      []string{testClusterTag},
	}
	testSubnetA2 := subnets.Subnet{
		ID:        subnetA2UUID,
		Name:      "subnet-a2",
		NetworkID: networkAUUID,
		Tags:      []string{testClusterTag},
	}
	testSubnetB1 := subnets.Subnet{
		ID:        subnetB1UUID,
		Name:      "subnet-b1",
		NetworkID: networkBUUID,
		Tags:      []string{testClusterTag},
	}

	// Define arbitrary test network and subnet filters for use in multiple tests,
	// the gophercloud ListOpts they should translate to, and the arbitrary returned networks/subnets.
	testNetworkFilter := infrav1.Filter{Tags: testClusterTag}
	testNetworkListOpts := networks.ListOpts{Tags: testClusterTag}
	testSubnetFilter := infrav1.SubnetFilter{Tags: testClusterTag}
	testSubnetListOpts := subnets.ListOpts{Tags: testClusterTag}

	tests := []struct {
		name          string
		networkParams []infrav1.NetworkParam
		want          []infrav1.Network
		expect        func(m *mock_networking.MockNetworkClientMockRecorder)
		wantErr       bool
	}{
		{
			name: "Network UUID without subnet",
			networkParams: []infrav1.NetworkParam{
				{UUID: networkAUUID},
			},
			want: []infrav1.Network{
				{ID: networkAUUID, Subnet: &infrav1.Subnet{}},
			},
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
			},
			wantErr: false,
		},
		{
			name: "Network filter without subnet",
			networkParams: []infrav1.NetworkParam{
				{Filter: testNetworkFilter},
			},
			want: []infrav1.Network{
				{ID: networkAUUID, Subnet: &infrav1.Subnet{}},
				{ID: networkBUUID, Subnet: &infrav1.Subnet{}},
			},
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
				// List tagged networks (A & B)
				m.ListNetwork(&testNetworkListOpts).
					Return([]networks.Network{testNetworkA, testNetworkB}, nil)
			},
			wantErr: false,
		},
		{
			name: "Subnet by filter without network",
			networkParams: []infrav1.NetworkParam{
				{
					Subnets: []infrav1.SubnetParam{{Filter: testSubnetFilter}},
				},
			},
			want: []infrav1.Network{
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA1UUID}},
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA2UUID}},
				{ID: networkBUUID, Subnet: &infrav1.Subnet{ID: subnetB1UUID}},
			},
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
				// List all tagged subnets in any network (A1, A2, and B1)
				m.ListSubnet(&testSubnetListOpts).
					Return([]subnets.Subnet{testSubnetA1, testSubnetA2, testSubnetB1}, nil)
			},
			wantErr: false,
		},
		{
			name: "Network UUID and subnet filter",
			networkParams: []infrav1.NetworkParam{
				{
					UUID: networkAUUID,
					Subnets: []infrav1.SubnetParam{
						{Filter: testSubnetFilter},
					},
				},
			},
			want: []infrav1.Network{
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA1UUID}},
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA2UUID}},
			},
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
				// List tagged subnets in network A (A1 & A2)
				networkAFilter := testSubnetListOpts
				networkAFilter.NetworkID = networkAUUID
				m.ListSubnet(&networkAFilter).
					Return([]subnets.Subnet{testSubnetA1, testSubnetA2}, nil)
			},
			wantErr: false,
		},
		{
			name: "Network UUID and subnet UUID",
			networkParams: []infrav1.NetworkParam{
				{
					UUID: networkAUUID,
					Subnets: []infrav1.SubnetParam{
						{UUID: subnetA1UUID},
					},
				},
			},
			want: []infrav1.Network{
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA1UUID}},
			},
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
			},
			wantErr: false,
		},
		{
			name: "Network UUID and multiple subnet params",
			networkParams: []infrav1.NetworkParam{
				{
					UUID: networkAUUID,
					Subnets: []infrav1.SubnetParam{
						{UUID: subnetA3UUID},
						{Filter: testSubnetFilter},
					},
				},
			},
			want: []infrav1.Network{
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA3UUID}},
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA1UUID}},
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA2UUID}},
			},
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
				// List tagged subnets in network A
				networkAFilter := testSubnetListOpts
				networkAFilter.NetworkID = networkAUUID
				m.ListSubnet(&networkAFilter).
					Return([]subnets.Subnet{testSubnetA1, testSubnetA2}, nil)
			},
			wantErr: false,
		},
		{
			name: "Multiple network params",
			networkParams: []infrav1.NetworkParam{
				{
					UUID: networkCUUID,
					Subnets: []infrav1.SubnetParam{
						{UUID: subnetC1UUID},
					},
				},
				{
					Filter: testNetworkFilter,
					Subnets: []infrav1.SubnetParam{
						{Filter: testSubnetFilter},
					},
				},
			},
			want: []infrav1.Network{
				{ID: networkCUUID, Subnet: &infrav1.Subnet{ID: subnetC1UUID}},
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA1UUID}},
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA2UUID}},
				{ID: networkBUUID, Subnet: &infrav1.Subnet{ID: subnetB1UUID}},
			},
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
				// List tagged networks (A & B)
				m.ListNetwork(&testNetworkListOpts).
					Return([]networks.Network{testNetworkA, testNetworkB}, nil)

				// List tagged subnets in network A (A1 & A2)
				networkAFilter := testSubnetListOpts
				networkAFilter.NetworkID = networkAUUID
				m.ListSubnet(&networkAFilter).
					Return([]subnets.Subnet{testSubnetA1, testSubnetA2}, nil)

				// List tagged subnets in network B (B1)
				networkBFilter := testSubnetListOpts
				networkBFilter.NetworkID = networkBUUID
				m.ListSubnet(&networkBFilter).
					Return([]subnets.Subnet{testSubnetB1}, nil)
			},
			wantErr: false,
		},
		{
			// Expect an error if a network filter doesn't match any networks
			name: "Network filter matches no networks",
			networkParams: []infrav1.NetworkParam{
				{Filter: testNetworkFilter},
			},
			want: nil,
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
				// List tagged networks (none for this test)
				m.ListNetwork(&testNetworkListOpts).Return([]networks.Network{}, nil)
			},
			wantErr: true,
		},
		{
			// Expect an error if a subnet filter doesn't match any subnets
			name: "Subnet filter matches no subnets",
			networkParams: []infrav1.NetworkParam{
				{
					UUID: networkAUUID,
					Subnets: []infrav1.SubnetParam{
						{Filter: testSubnetFilter},
					},
				},
			},
			want: nil,
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
				// List tagged subnets in network A
				networkAFilter := testSubnetListOpts
				networkAFilter.NetworkID = networkAUUID
				m.ListSubnet(&networkAFilter).Return([]subnets.Subnet{}, nil)
			},
			wantErr: true,
		},
		{
			name: "Subnet UUID without network",
			networkParams: []infrav1.NetworkParam{
				{Subnets: []infrav1.SubnetParam{
					{UUID: subnetA1UUID},
				}},
			},
			want: nil,
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			mockNetworkClient := mock_networking.NewMockNetworkClient(mockCtrl)
			tt.expect(mockNetworkClient.EXPECT())

			networkingService := networking.NewTestService(
				"", mockNetworkClient, logr.Discard(),
			)
			s := &Service{
				networkingService: networkingService,
			}

			got, err := s.getServerNetworks(tt.networkParams)
			g := NewWithT(t)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
			g.Expect(got).To(Equal(tt.want))
		})
	}
}
