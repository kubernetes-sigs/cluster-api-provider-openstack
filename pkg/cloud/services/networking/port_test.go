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
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsecurity"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	. "github.com/onsi/gomega"
	"k8s.io/utils/pointer"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

func Test_CreatePort(t *testing.T) {
	// Arbitrary GUIDs used in the tests
	netID := "7fd24ceb-788a-441f-ad0a-d8e2f5d31a1d"
	subnetID1 := "d9c88a6d-0b8c-48ff-8f0e-8d85a078c194"
	subnetID2 := "d9c2346d-05gc-48er-9ut4-ig83ayt8c7h4"
	portID1 := "50214c48-c09e-4a54-914f-97b40fd22802"
	hostID := "825c1b11-3dca-4bfe-a2d8-a3cc1964c8d5"
	trunkID := "eb7541fa-5e2a-4cca-b2c3-dfa409b917ce"
	portSecurityGroupID := "f51d1206-fc5a-4f7a-a5c0-2e03e44e4dc0"

	// Other arbitrary variables passed in to the tests
	instanceSecurityGroups := []string{"instance-secgroup"}
	securityGroupUUIDs := []string{portSecurityGroupID}
	portSecurityGroupFilters := []infrav1.SecurityGroupFilter{{ID: portSecurityGroupID, Name: "port-secgroup"}}
	valueSpecs := map[string]string{"key": "value"}

	tests := []struct {
		name                   string
		portName               string
		port                   infrav1.PortOpts
		instanceSecurityGroups []string
		tags                   []string
		expect                 func(m *mock.MockNetworkClientMockRecorder)
		// Note the 'wanted' port isn't so important, since it will be whatever we tell ListPort or CreatePort to return.
		// Mostly in this test suite, we're checking that CreatePort is called with the expected port opts.
		want    *ports.Port
		wantErr bool
	}{
		{
			"creates port with defaults (description and secgroups) if not specified in portOpts",
			"foo-port-1",
			infrav1.PortOpts{
				Network: &infrav1.NetworkFilter{
					ID: netID,
				},
			},
			instanceSecurityGroups,
			[]string{},
			func(m *mock.MockNetworkClientMockRecorder) {
				m.
					CreatePort(portsbinding.CreateOptsExt{
						CreateOptsBuilder: ports.CreateOpts{
							Name:                "foo-port-1",
							Description:         "Created by cluster-api-provider-openstack cluster test-cluster",
							SecurityGroups:      &instanceSecurityGroups,
							NetworkID:           netID,
							AllowedAddressPairs: []ports.AddressPair{},
						},
					}).Return(&ports.Port{ID: portID1}, nil)
			},
			&ports.Port{ID: portID1},
			false,
		},
		{
			"creates port with specified portOpts if no matching port exists",
			"foo-port-bar",
			infrav1.PortOpts{
				Network: &infrav1.NetworkFilter{
					ID: netID,
				},
				NameSuffix:   pointer.String("bar"),
				Description:  pointer.String("this is a test port"),
				MACAddress:   pointer.String("fe:fe:fe:fe:fe:fe"),
				AdminStateUp: pointer.Bool(true),
				FixedIPs: []infrav1.FixedIP{
					{
						Subnet: &infrav1.SubnetFilter{
							Name: "subnetFoo",
						},
						IPAddress: pointer.String("192.168.0.50"),
					}, {
						IPAddress: pointer.String("192.168.1.50"),
					},
				},
				SecurityGroups: portSecurityGroupFilters,
				AllowedAddressPairs: []infrav1.AddressPair{{
					IPAddress:  "10.10.10.10",
					MACAddress: pointer.String("f1:f1:f1:f1:f1:f1"),
				}},
				HostID:   pointer.String(hostID),
				VNICType: pointer.String("direct"),
				Profile: &infrav1.BindingProfile{
					OVSHWOffload: pointer.Bool(true),
					TrustedVF:    pointer.Bool(true),
				},
				DisablePortSecurity: pointer.Bool(false),
				Tags:                []string{"my-port-tag"},
			},
			nil,
			nil,
			func(m *mock.MockNetworkClientMockRecorder) {
				portCreateOpts := ports.CreateOpts{
					NetworkID:    netID,
					Name:         "foo-port-bar",
					Description:  "this is a test port",
					AdminStateUp: pointer.Bool(true),
					MACAddress:   "fe:fe:fe:fe:fe:fe",
					FixedIPs: []ports.IP{
						{
							SubnetID:  subnetID1,
							IPAddress: "192.168.0.50",
						}, {
							IPAddress: "192.168.1.50",
						},
					},
					SecurityGroups: &securityGroupUUIDs,
					AllowedAddressPairs: []ports.AddressPair{{
						IPAddress:  "10.10.10.10",
						MACAddress: "f1:f1:f1:f1:f1:f1",
					}},
				}
				portsecurityCreateOptsExt := portsecurity.PortCreateOptsExt{
					CreateOptsBuilder:   portCreateOpts,
					PortSecurityEnabled: pointer.Bool(true),
				}
				portbindingCreateOptsExt := portsbinding.CreateOptsExt{
					// Note for the test matching, the order in which the builders are composed
					// must be the same as in the function we are testing.
					CreateOptsBuilder: portsecurityCreateOptsExt,
					HostID:            hostID,
					VNICType:          "direct",
					Profile: map[string]interface{}{
						"capabilities": []string{"switchdev"},
						"trusted":      true,
					},
				}
				m.
					CreatePort(portbindingCreateOptsExt).
					Return(&ports.Port{
						ID: portID1,
					}, nil)
				m.ReplaceAllAttributesTags("ports", portID1, attributestags.ReplaceAllOpts{Tags: []string{"my-port-tag"}}).Return([]string{"my-port-tag"}, nil)
				m.
					ListSubnet(subnets.ListOpts{
						Name:      "subnetFoo",
						NetworkID: netID,
					}).Return([]subnets.Subnet{
					{
						ID:        subnetID1,
						Name:      "subnetFoo",
						NetworkID: netID,
					},
				}, nil)
			},
			&ports.Port{
				ID: portID1,
			},
			false,
		},
		{
			"fails to create port with specified portOpts if subnet query returns more than one subnet",
			"foo-port-bar",
			infrav1.PortOpts{
				Network: &infrav1.NetworkFilter{
					ID: netID,
				},
				NameSuffix:  pointer.String("foo-port-bar"),
				Description: pointer.String("this is a test port"),
				FixedIPs: []infrav1.FixedIP{{
					Subnet: &infrav1.SubnetFilter{
						FilterByNeutronTags: infrav1.FilterByNeutronTags{
							Tags: []infrav1.NeutronTag{"Foo"},
						},
					},
					IPAddress: pointer.String("192.168.0.50"),
				}},
			},
			nil,
			nil,
			func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListSubnet(subnets.ListOpts{
						Tags:      "Foo",
						NetworkID: netID,
					}).Return([]subnets.Subnet{
					{
						ID:        subnetID1,
						NetworkID: netID,
						Name:      "subnetFoo",
					},
					{
						ID:        subnetID2,
						NetworkID: netID,
						Name:      "subnetBar",
					},
				}, nil)
			},
			nil,
			true,
		},
		{
			"overrides default (instance) security groups if port security groups are specified",
			"foo-port-1",
			infrav1.PortOpts{
				Network: &infrav1.NetworkFilter{
					ID: netID,
				},
				SecurityGroups: portSecurityGroupFilters,
			},
			instanceSecurityGroups,
			[]string{},
			func(m *mock.MockNetworkClientMockRecorder) {
				m.
					CreatePort(portsbinding.CreateOptsExt{
						CreateOptsBuilder: ports.CreateOpts{
							Name:                "foo-port-1",
							Description:         "Created by cluster-api-provider-openstack cluster test-cluster",
							SecurityGroups:      &securityGroupUUIDs,
							NetworkID:           netID,
							AllowedAddressPairs: []ports.AddressPair{},
						},
					},
					).Return(&ports.Port{ID: portID1}, nil)
			},
			&ports.Port{ID: portID1},
			false,
		},
		{
			"creates port with instance tags when port tags aren't specified",
			"foo-port-1",
			infrav1.PortOpts{
				Network: &infrav1.NetworkFilter{
					ID: netID,
				},
			},
			nil,
			[]string{"my-instance-tag"},
			func(m *mock.MockNetworkClientMockRecorder) {
				m.CreatePort(portsbinding.CreateOptsExt{
					CreateOptsBuilder: ports.CreateOpts{
						Name:                "foo-port-1",
						Description:         "Created by cluster-api-provider-openstack cluster test-cluster",
						NetworkID:           netID,
						AllowedAddressPairs: []ports.AddressPair{},
					},
				}).Return(&ports.Port{ID: portID1}, nil)
				m.ReplaceAllAttributesTags("ports", portID1, attributestags.ReplaceAllOpts{Tags: []string{"my-instance-tag"}}).Return([]string{"my-instance-tag"}, nil)
			},
			&ports.Port{ID: portID1},
			false,
		},
		{
			"creates port with port specific tags appending to instance tags",
			"foo-port-1",
			infrav1.PortOpts{
				Network: &infrav1.NetworkFilter{
					ID: netID,
				},
				Tags: []string{"my-port-tag"},
			},
			nil,
			[]string{"my-instance-tag"},
			func(m *mock.MockNetworkClientMockRecorder) {
				m.CreatePort(portsbinding.CreateOptsExt{
					CreateOptsBuilder: ports.CreateOpts{
						Name:                "foo-port-1",
						Description:         "Created by cluster-api-provider-openstack cluster test-cluster",
						NetworkID:           netID,
						AllowedAddressPairs: []ports.AddressPair{},
					},
				}).Return(&ports.Port{ID: portID1}, nil)
				m.
					ReplaceAllAttributesTags("ports", portID1, attributestags.ReplaceAllOpts{Tags: []string{"my-instance-tag", "my-port-tag"}}).
					Return([]string{"my-instance-tag", "my-port-tag"}, nil)
			},
			&ports.Port{ID: portID1},
			false,
		},
		{
			"creates port and trunk (with tags) if they aren't found",
			"foo-port-1",
			infrav1.PortOpts{
				Network: &infrav1.NetworkFilter{
					ID: netID,
				},
				Trunk: pointer.Bool(true),
			},
			nil,
			[]string{"my-tag"},
			func(m *mock.MockNetworkClientMockRecorder) {
				m.
					CreatePort(portsbinding.CreateOptsExt{
						CreateOptsBuilder: ports.CreateOpts{
							Name:                "foo-port-1",
							Description:         "Created by cluster-api-provider-openstack cluster test-cluster",
							NetworkID:           netID,
							AllowedAddressPairs: []ports.AddressPair{},
						},
					}).Return(&ports.Port{Name: "foo-port-1", ID: portID1}, nil)
				m.
					ListTrunk(trunks.ListOpts{
						Name:   "foo-port-1",
						PortID: portID1,
					}).Return([]trunks.Trunk{}, nil)
				m.
					CreateTrunk(trunks.CreateOpts{
						Name:        "foo-port-1",
						PortID:      portID1,
						Description: "Created by cluster-api-provider-openstack cluster test-cluster",
					}).Return(&trunks.Trunk{ID: trunkID}, nil)

				m.ReplaceAllAttributesTags("ports", portID1, attributestags.ReplaceAllOpts{Tags: []string{"my-tag"}}).Return([]string{"my-tag"}, nil)
				m.ReplaceAllAttributesTags("trunks", trunkID, attributestags.ReplaceAllOpts{Tags: []string{"my-tag"}}).Return([]string{"my-tag"}, nil)
			},
			&ports.Port{Name: "foo-port-1", ID: portID1},
			false,
		},
		{
			"creates port with value_specs",
			"foo-port-1",
			infrav1.PortOpts{
				Network: &infrav1.NetworkFilter{
					ID: netID,
				},
				ValueSpecs: []infrav1.ValueSpec{
					{
						Name:  "Not important",
						Key:   "key",
						Value: "value",
					},
				},
			},
			nil,
			nil,
			func(m *mock.MockNetworkClientMockRecorder) {
				m.
					CreatePort(portsbinding.CreateOptsExt{
						CreateOptsBuilder: ports.CreateOpts{
							Name:                "foo-port-1",
							Description:         "Created by cluster-api-provider-openstack cluster test-cluster",
							NetworkID:           netID,
							AllowedAddressPairs: []ports.AddressPair{},
							ValueSpecs:          &valueSpecs,
						},
					}).Return(&ports.Port{ID: portID1}, nil)
			},
			&ports.Port{ID: portID1},
			false,
		},
		{
			"creates port with propagate uplink status",
			"foo-port-1",
			infrav1.PortOpts{
				Network: &infrav1.NetworkFilter{
					ID: netID,
				},
				PropagateUplinkStatus: pointer.Bool(true),
			},
			instanceSecurityGroups,
			[]string{},
			func(m *mock.MockNetworkClientMockRecorder) {
				m.
					CreatePort(portsbinding.CreateOptsExt{
						CreateOptsBuilder: ports.CreateOpts{
							Name:                  "foo-port-1",
							Description:           "Created by cluster-api-provider-openstack cluster test-cluster",
							SecurityGroups:        &instanceSecurityGroups,
							NetworkID:             netID,
							AllowedAddressPairs:   []ports.AddressPair{},
							PropagateUplinkStatus: pointer.Bool(true),
						},
					}).Return(&ports.Port{ID: portID1, PropagateUplinkStatus: true}, nil)
			},
			&ports.Port{ID: portID1, PropagateUplinkStatus: true},
			false,
		},
	}

	eventObject := &infrav1.OpenStackMachine{}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			g := NewWithT(t)
			mockClient := mock.NewMockNetworkClient(mockCtrl)
			tt.expect(mockClient.EXPECT())
			s := Service{
				client: mockClient,
			}
			got, err := s.CreatePort(
				eventObject,
				"test-cluster",
				tt.portName,
				&tt.port,
				tt.instanceSecurityGroups,
				tt.tags,
			)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
			g.Expect(got).To(Equal(tt.want))
		})
	}
}

func TestService_normalizePorts(t *testing.T) {
	const (
		defaultNetworkID = "3c66f3ca-2d26-4d9d-ae3b-568f54129773"
		defaultSubnetID  = "d8dbba89-8c39-4192-a571-e702fca35bac"

		networkID = "afa54944-1443-4132-9ef5-ce37eb4d6ab6"
		subnetID  = "d786e715-c299-4a97-911d-640c10fc0392"
	)

	openStackCluster := &infrav1.OpenStackCluster{
		Status: infrav1.OpenStackClusterStatus{
			Network: &infrav1.NetworkStatusWithSubnets{
				NetworkStatus: infrav1.NetworkStatus{
					ID: defaultNetworkID,
				},
				Subnets: []infrav1.Subnet{
					{ID: defaultSubnetID},
				},
			},
		},
	}

	tests := []struct {
		name          string
		ports         []infrav1.PortOpts
		instanceTrunk bool
		expectNetwork func(m *mock.MockNetworkClientMockRecorder)
		want          []infrav1.PortOpts
		wantErr       bool
	}{
		{
			name:  "No ports: no ports",
			ports: []infrav1.PortOpts{},
			want:  []infrav1.PortOpts{},
		},
		{
			name: "Nil network, no fixed IPs: cluster defaults",
			ports: []infrav1.PortOpts{
				{
					Network:  nil,
					FixedIPs: nil,
				},
			},
			want: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID: defaultNetworkID,
					},
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								ID: defaultSubnetID,
							},
						},
					},
					Trunk: pointer.Bool(false),
				},
			},
		},
		{
			name: "Empty network, no fixed IPs: cluster defaults",
			ports: []infrav1.PortOpts{
				{
					Network:  &infrav1.NetworkFilter{},
					FixedIPs: nil,
				},
			},
			want: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID: defaultNetworkID,
					},
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								ID: defaultSubnetID,
							},
						},
					},
					Trunk: pointer.Bool(false),
				},
			},
		},
		{
			name: "Port inherits trunk from instance",
			ports: []infrav1.PortOpts{
				{
					Network:  &infrav1.NetworkFilter{},
					FixedIPs: nil,
				},
			},
			instanceTrunk: true,
			want: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID: defaultNetworkID,
					},
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								ID: defaultSubnetID,
							},
						},
					},
					Trunk: pointer.Bool(true),
				},
			},
		},
		{
			name: "Port overrides trunk from instance",
			ports: []infrav1.PortOpts{
				{
					Network:  &infrav1.NetworkFilter{},
					FixedIPs: nil,
					Trunk:    pointer.Bool(true),
				},
			},
			want: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID: defaultNetworkID,
					},
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								ID: defaultSubnetID,
							},
						},
					},
					Trunk: pointer.Bool(true),
				},
			},
		},
		{
			name: "Network defined by ID: unchanged",
			ports: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID: networkID,
					},
				},
			},
			want: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID: networkID,
					},
					Trunk: pointer.Bool(false),
				},
			},
		},
		{
			name: "Network defined by filter: add ID from network lookup",
			ports: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						Name: "test-network",
					},
				},
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListNetwork(networks.ListOpts{Name: "test-network"}).Return([]networks.Network{
					{ID: networkID},
				}, nil)
			},
			want: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID:   networkID,
						Name: "test-network",
					},
					Trunk: pointer.Bool(false),
				},
			},
		},
		{
			name: "No network, fixed IP has subnet by ID: add ID from subnet",
			ports: []infrav1.PortOpts{
				{
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								ID: subnetID,
							},
						},
					},
				},
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.GetSubnet(subnetID).Return(&subnets.Subnet{ID: subnetID, NetworkID: networkID}, nil)
			},
			want: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID: networkID,
					},
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								ID: subnetID,
							},
						},
					},
					Trunk: pointer.Bool(false),
				},
			},
		},
		{
			name: "No network, fixed IP has subnet by filter: add ID from subnet",
			ports: []infrav1.PortOpts{
				{
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								Name: "test-subnet",
							},
						},
					},
				},
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListSubnet(subnets.ListOpts{Name: "test-subnet"}).Return([]subnets.Subnet{
					{ID: subnetID, NetworkID: networkID},
				}, nil)
			},
			want: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID: networkID,
					},
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								ID:   subnetID,
								Name: "test-subnet",
							},
						},
					},
					Trunk: pointer.Bool(false),
				},
			},
		},
		{
			name: "No network, fixed IP subnet returns no matches: error",
			ports: []infrav1.PortOpts{
				{
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								Name: "test-subnet",
							},
						},
					},
				},
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListSubnet(subnets.ListOpts{Name: "test-subnet"}).Return([]subnets.Subnet{}, nil)
			},
			wantErr: true,
		},
		{
			name: "No network, only fixed IP subnet returns multiple matches: error",
			ports: []infrav1.PortOpts{
				{
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								Name: "test-subnet",
							},
						},
					},
				},
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListSubnet(subnets.ListOpts{Name: "test-subnet"}).Return([]subnets.Subnet{
					{ID: subnetID, NetworkID: networkID},
					{ID: "8008494c-301e-4e5c-951b-a8ab568447fd", NetworkID: "5d48bfda-db28-42ee-8374-50e13d1fe5ea"},
				}, nil)
			},
			wantErr: true,
		},
		{
			name: "No network, first fixed IP subnet returns multiple matches: used ID from second fixed IP",
			ports: []infrav1.PortOpts{
				{
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								Name: "test-subnet1",
							},
						},
						{
							Subnet: &infrav1.SubnetFilter{
								Name: "test-subnet2",
							},
						},
					},
				},
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListSubnet(subnets.ListOpts{Name: "test-subnet1"}).Return([]subnets.Subnet{
					{ID: subnetID, NetworkID: networkID},
					{ID: "8008494c-301e-4e5c-951b-a8ab568447fd", NetworkID: "5d48bfda-db28-42ee-8374-50e13d1fe5ea"},
				}, nil)
				m.ListSubnet(subnets.ListOpts{Name: "test-subnet2"}).Return([]subnets.Subnet{
					{ID: subnetID, NetworkID: networkID},
				}, nil)
			},
			want: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID: networkID,
					},
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								Name: "test-subnet1",
							},
						},
						{
							Subnet: &infrav1.SubnetFilter{
								ID:   subnetID,
								Name: "test-subnet2",
							},
						},
					},
					Trunk: pointer.Bool(false),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			log := testr.New(t)

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockClient := mock.NewMockNetworkClient(mockCtrl)
			if tt.expectNetwork != nil {
				tt.expectNetwork(mockClient.EXPECT())
			}
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			s := Service{
				client: mockClient,
				scope:  scope.NewWithLogger(mockScopeFactory, log),
			}

			got, err := s.normalizePorts(tt.ports, openStackCluster, tt.instanceTrunk)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
				return
			}

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(got).To(Equal(tt.want), cmp.Diff(got, tt.want))
		})
	}
}

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
			args: args{"test-1-instance", &infrav1.PortOpts{NameSuffix: pointer.String("foo")}, 4},
			want: "test-1-instance-foo",
		},
		{
			name: "without PortOpts name suffix",
			args: args{"test-1-instance", &infrav1.PortOpts{}, 4},
			want: "test-1-instance-4",
		},
		{
			name: "with PortOpts name suffix",
			args: args{"test-1-instance", &infrav1.PortOpts{NameSuffix: pointer.String("foo2"), Network: &infrav1.NetworkFilter{ID: "bar"}, DisablePortSecurity: pointer.Bool(true)}, 4},
			want: "test-1-instance-foo2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetPortName(tt.args.instanceName, tt.args.opts, tt.args.netIndex); got != tt.want {
				t.Errorf("getPortName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_AdoptPorts(t *testing.T) {
	const (
		networkID1 = "5e8e0d3b-7f3d-4f3e-8b3f-3e3e3e3e3e3e"
		networkID2 = "0a4ff38e-1e03-4b4e-994c-c8ae38a2915e"
		networkID3 = "bd22ea65-53de-4585-bb6f-b0a84d0085d1"
		portID1    = "78e0d3b-7f3d-4f3e-8b3f-3e3e3e3e3e3e"
		portID2    = "a838209b-389a-47a0-9161-3d6919891074"
	)

	tests := []struct {
		testName           string
		desiredPorts       []infrav1.PortOpts
		dependentResources infrav1.DependentMachineResources
		expect             func(*mock.MockNetworkClientMockRecorder)
		want               infrav1.DependentMachineResources
		wantErr            bool
	}{
		{
			testName: "No desired ports",
		},
		{
			testName: "desired port already in status: no-op",
			desiredPorts: []infrav1.PortOpts{
				{Network: &infrav1.NetworkFilter{ID: networkID1}},
			},
			dependentResources: infrav1.DependentMachineResources{
				Ports: []infrav1.PortStatus{
					{
						ID: portID1,
					},
				},
			},
			want: infrav1.DependentMachineResources{
				Ports: []infrav1.PortStatus{
					{
						ID: portID1,
					},
				},
			},
		},
		{
			testName: "desired port not in status, exists: adopt",
			desiredPorts: []infrav1.PortOpts{
				{Network: &infrav1.NetworkFilter{ID: networkID1}},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListPort(ports.ListOpts{Name: "test-machine-0", NetworkID: networkID1}).
					Return([]ports.Port{{ID: portID1}}, nil)
			},
			want: infrav1.DependentMachineResources{
				Ports: []infrav1.PortStatus{
					{
						ID: portID1,
					},
				},
			},
		},
		{
			testName: "desired port not in status, does not exist: ignore",
			desiredPorts: []infrav1.PortOpts{
				{Network: &infrav1.NetworkFilter{ID: networkID1}},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListPort(ports.ListOpts{Name: "test-machine-0", NetworkID: networkID1}).
					Return(nil, nil)
			},
			want: infrav1.DependentMachineResources{},
		},
		{
			testName: "2 desired ports, first in status, second exists: adopt second",
			desiredPorts: []infrav1.PortOpts{
				{Network: &infrav1.NetworkFilter{ID: networkID1}},
				{Network: &infrav1.NetworkFilter{ID: networkID2}},
			},
			dependentResources: infrav1.DependentMachineResources{
				Ports: []infrav1.PortStatus{
					{
						ID: portID1,
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListPort(ports.ListOpts{Name: "test-machine-1", NetworkID: networkID2}).
					Return([]ports.Port{{ID: portID2}}, nil)
			},
			want: infrav1.DependentMachineResources{
				Ports: []infrav1.PortStatus{
					{ID: portID1},
					{ID: portID2},
				},
			},
		},
		{
			testName: "3 desired ports, first in status, second does not exist: ignore, do no look for third",
			desiredPorts: []infrav1.PortOpts{
				{Network: &infrav1.NetworkFilter{ID: networkID1}},
				{Network: &infrav1.NetworkFilter{ID: networkID2}},
				{Network: &infrav1.NetworkFilter{ID: networkID3}},
			},
			dependentResources: infrav1.DependentMachineResources{
				Ports: []infrav1.PortStatus{
					{
						ID: portID1,
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListPort(ports.ListOpts{Name: "test-machine-1", NetworkID: networkID2}).
					Return(nil, nil)
			},
			want: infrav1.DependentMachineResources{
				Ports: []infrav1.PortStatus{
					{ID: portID1},
				},
			},
		},
	}
	for i := range tests {
		tt := &tests[i]
		t.Run(tt.testName, func(t *testing.T) {
			g := NewWithT(t)
			log := testr.New(t)

			mockCtrl := gomock.NewController(t)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			mockClient := mock.NewMockNetworkClient(mockCtrl)
			if tt.expect != nil {
				tt.expect(mockClient.EXPECT())
			}

			s := Service{
				client: mockClient,
			}

			err := s.AdoptPorts(scope.NewWithLogger(mockScopeFactory, log),
				"test-machine",
				tt.desiredPorts, &tt.dependentResources)
			if tt.wantErr {
				g.Expect(err).Error()
				return
			}

			g.Expect(tt.dependentResources).To(Equal(tt.want), cmp.Diff(&tt.dependentResources, tt.want))
		})
	}
}
