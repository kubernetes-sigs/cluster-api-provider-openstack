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
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsecurity"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"k8s.io/utils/pointer"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

func Test_CreatePort(t *testing.T) {
	// Arbitrary values used in the tests
	const (
		netID               = "7fd24ceb-788a-441f-ad0a-d8e2f5d31a1d"
		subnetID1           = "d9c88a6d-0b8c-48ff-8f0e-8d85a078c194"
		subnetID2           = "d9c2346d-05gc-48er-9ut4-ig83ayt8c7h4"
		portID              = "50214c48-c09e-4a54-914f-97b40fd22802"
		hostID              = "825c1b11-3dca-4bfe-a2d8-a3cc1964c8d5"
		trunkID             = "eb7541fa-5e2a-4cca-b2c3-dfa409b917ce"
		portSecurityGroupID = "f51d1206-fc5a-4f7a-a5c0-2e03e44e4dc0"
		ipAddress1          = "192.0.2.1"
		ipAddress2          = "198.51.100.1"
		macAddress          = "de:ad:be:ef:fe:ed"
	)

	tests := []struct {
		name   string
		port   infrav1.ResolvedPortSpec
		expect func(m *mock.MockNetworkClientMockRecorder, g Gomega)
		// Note the 'wanted' port isn't so important, since it will be whatever we tell ListPort or CreatePort to return.
		// Mostly in this test suite, we're checking that CreatePort is called with the expected port opts.
		want    *ports.Port
		wantErr bool
	}{
		{
			name: "creates port correctly with all options specified except tags, trunk and disablePortSecurity",
			port: infrav1.ResolvedPortSpec{
				Name:        "foo-port-1",
				Description: "Created by cluster-api-provider-openstack cluster test-cluster",
				NetworkID:   netID,
				FixedIPs: []infrav1.ResolvedFixedIP{
					{
						SubnetID:  pointer.String(subnetID1),
						IPAddress: pointer.String(ipAddress1),
					},
					{
						IPAddress: pointer.String(ipAddress2),
					},
					{
						SubnetID: pointer.String(subnetID2),
					},
				},
				SecurityGroups: []string{portSecurityGroupID},
				ResolvedPortSpecFields: infrav1.ResolvedPortSpecFields{
					AdminStateUp: pointer.Bool(true),
					MACAddress:   pointer.String(macAddress),
					AllowedAddressPairs: []infrav1.AddressPair{
						{
							IPAddress:  ipAddress1,
							MACAddress: pointer.String(macAddress),
						},
						{
							IPAddress: ipAddress2,
						},
					},
					HostID:   pointer.String(hostID),
					VNICType: pointer.String("normal"),
					Profile: &infrav1.BindingProfile{
						OVSHWOffload: pointer.Bool(true),
						TrustedVF:    pointer.Bool(true),
					},
					PropagateUplinkStatus: pointer.Bool(true),
					ValueSpecs: []infrav1.ValueSpec{
						{
							Name:  "test-valuespec",
							Key:   "test-key",
							Value: "test-value",
						},
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder, g Gomega) {
				var expectedCreateOpts ports.CreateOptsBuilder
				expectedCreateOpts = ports.CreateOpts{
					Name:         "foo-port-1",
					Description:  "Created by cluster-api-provider-openstack cluster test-cluster",
					NetworkID:    netID,
					AdminStateUp: pointer.Bool(true),
					MACAddress:   macAddress,
					FixedIPs: []ports.IP{
						{
							SubnetID:  subnetID1,
							IPAddress: ipAddress1,
						},
						{
							IPAddress: ipAddress2,
						},
						{
							SubnetID: subnetID2,
						},
					},
					SecurityGroups: &[]string{portSecurityGroupID},
					AllowedAddressPairs: []ports.AddressPair{
						{
							IPAddress:  ipAddress1,
							MACAddress: macAddress,
						},
						{
							IPAddress: ipAddress2,
						},
					},
					PropagateUplinkStatus: pointer.Bool(true),
					ValueSpecs: &map[string]string{
						"test-key": "test-value",
					},
				}
				expectedCreateOpts = portsbinding.CreateOptsExt{
					CreateOptsBuilder: expectedCreateOpts,
					HostID:            hostID,
					VNICType:          "normal",
					Profile: map[string]interface{}{
						"capabilities": []string{"switchdev"},
						"trusted":      true,
					},
				}

				// The following allows us to use gomega to
				// compare the argument instead of gomock.
				// Gomock's output in the case of a mismatch is
				// not usable for this struct.
				m.CreatePort(gomock.Any()).DoAndReturn(func(builder ports.CreateOptsBuilder) (*ports.Port, error) {
					gotCreateOpts := builder.(portsbinding.CreateOptsExt)
					g.Expect(gotCreateOpts).To(Equal(expectedCreateOpts), cmp.Diff(gotCreateOpts, expectedCreateOpts))
					return &ports.Port{ID: portID}, nil
				})
			},
			want: &ports.Port{ID: portID},
		},
		{
			name: "creates minimum port correctly",
			port: infrav1.ResolvedPortSpec{
				Name:      "test-port",
				NetworkID: netID,
			},
			expect: func(m *mock.MockNetworkClientMockRecorder, g Gomega) {
				var expectedCreateOpts ports.CreateOptsBuilder
				expectedCreateOpts = ports.CreateOpts{
					NetworkID: netID,
					Name:      "test-port",
				}
				expectedCreateOpts = portsbinding.CreateOptsExt{
					CreateOptsBuilder: expectedCreateOpts,
				}
				m.CreatePort(gomock.Any()).DoAndReturn(func(builder ports.CreateOptsBuilder) (*ports.Port, error) {
					gotCreateOpts := builder.(portsbinding.CreateOptsExt)
					g.Expect(gotCreateOpts).To(Equal(expectedCreateOpts), cmp.Diff(gotCreateOpts, expectedCreateOpts))
					return &ports.Port{ID: portID}, nil
				})
			},
			want: &ports.Port{ID: portID},
		},
		{
			name: "disable port security also ignores allowed address pairs",
			port: infrav1.ResolvedPortSpec{
				Name:      "test-port",
				NetworkID: netID,
				ResolvedPortSpecFields: infrav1.ResolvedPortSpecFields{
					DisablePortSecurity: pointer.Bool(true),
					AllowedAddressPairs: []infrav1.AddressPair{
						{
							IPAddress:  ipAddress1,
							MACAddress: pointer.String(macAddress),
						},
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder, g Gomega) {
				var expectedCreateOpts ports.CreateOptsBuilder
				expectedCreateOpts = ports.CreateOpts{
					NetworkID: netID,
					Name:      "test-port",
				}
				expectedCreateOpts = portsecurity.PortCreateOptsExt{
					CreateOptsBuilder:   expectedCreateOpts,
					PortSecurityEnabled: pointer.Bool(false),
				}
				expectedCreateOpts = portsbinding.CreateOptsExt{
					CreateOptsBuilder: expectedCreateOpts,
				}
				m.CreatePort(gomock.Any()).DoAndReturn(func(builder ports.CreateOptsBuilder) (*ports.Port, error) {
					gotCreateOpts := builder.(portsbinding.CreateOptsExt)
					g.Expect(gotCreateOpts).To(Equal(expectedCreateOpts), cmp.Diff(gotCreateOpts, expectedCreateOpts))
					return &ports.Port{ID: portID}, nil
				})
			},
			want: &ports.Port{ID: portID},
		},
		{
			name: "disable port security explicitly false includes allowed address pairs",
			port: infrav1.ResolvedPortSpec{
				Name:      "test-port",
				NetworkID: netID,
				ResolvedPortSpecFields: infrav1.ResolvedPortSpecFields{
					DisablePortSecurity: pointer.Bool(false),
					AllowedAddressPairs: []infrav1.AddressPair{
						{
							IPAddress:  ipAddress1,
							MACAddress: pointer.String(macAddress),
						},
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder, g types.Gomega) {
				var expectedCreateOpts ports.CreateOptsBuilder
				expectedCreateOpts = ports.CreateOpts{
					NetworkID: netID,
					Name:      "test-port",
					AllowedAddressPairs: []ports.AddressPair{
						{
							IPAddress:  ipAddress1,
							MACAddress: macAddress,
						},
					},
				}
				expectedCreateOpts = portsecurity.PortCreateOptsExt{
					CreateOptsBuilder:   expectedCreateOpts,
					PortSecurityEnabled: pointer.Bool(true),
				}
				expectedCreateOpts = portsbinding.CreateOptsExt{
					CreateOptsBuilder: expectedCreateOpts,
				}
				m.CreatePort(gomock.Any()).DoAndReturn(func(builder ports.CreateOptsBuilder) (*ports.Port, error) {
					gotCreateOpts := builder.(portsbinding.CreateOptsExt)
					g.Expect(gotCreateOpts).To(Equal(expectedCreateOpts), cmp.Diff(gotCreateOpts, expectedCreateOpts))
					return &ports.Port{ID: portID}, nil
				})
			},
			want: &ports.Port{ID: portID},
		},
		{
			name: "tags and trunk",
			port: infrav1.ResolvedPortSpec{
				Name:      "test-port",
				NetworkID: netID,
				Tags:      []string{"tag1", "tag2"},
				Trunk:     pointer.Bool(true),
			},
			expect: func(m *mock.MockNetworkClientMockRecorder, g types.Gomega) {
				var expectedCreateOpts ports.CreateOptsBuilder
				expectedCreateOpts = ports.CreateOpts{
					NetworkID: netID,
					Name:      "test-port",
				}
				expectedCreateOpts = portsbinding.CreateOptsExt{
					CreateOptsBuilder: expectedCreateOpts,
				}

				// Create the port
				m.CreatePort(gomock.Any()).DoAndReturn(func(builder ports.CreateOptsBuilder) (*ports.Port, error) {
					gotCreateOpts := builder.(portsbinding.CreateOptsExt)
					g.Expect(gotCreateOpts).To(Equal(expectedCreateOpts), cmp.Diff(gotCreateOpts, expectedCreateOpts))
					return &ports.Port{ID: portID, Name: "test-port"}, nil
				})

				// Tag the port
				m.ReplaceAllAttributesTags("ports", portID, attributestags.ReplaceAllOpts{
					Tags: []string{"tag1", "tag2"},
				})

				// Look for existing trunk
				m.ListTrunk(trunks.ListOpts{
					PortID: portID,
					Name:   "test-port",
				}).Return([]trunks.Trunk{}, nil)

				// Create the trunk
				m.CreateTrunk(trunks.CreateOpts{
					PortID: portID,
					Name:   "test-port",
				}).Return(&trunks.Trunk{ID: trunkID}, nil)

				// Tag the trunk
				m.ReplaceAllAttributesTags("trunks", trunkID, attributestags.ReplaceAllOpts{
					Tags: []string{"tag1", "tag2"},
				})
			},
			want: &ports.Port{ID: portID, Name: "test-port"},
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
			tt.expect(mockClient.EXPECT(), g)
			s := Service{
				client: mockClient,
			}
			got, err := s.CreatePort(
				eventObject,
				&tt.port,
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

func TestService_ConstructPorts(t *testing.T) {
	const (
		defaultNetworkID = "3c66f3ca-2d26-4d9d-ae3b-568f54129773"
		defaultSubnetID  = "d8dbba89-8c39-4192-a571-e702fca35bac"

		networkID        = "afa54944-1443-4132-9ef5-ce37eb4d6ab6"
		subnetID1        = "d786e715-c299-4a97-911d-640c10fc0392"
		subnetID2        = "41ad8201-5b2f-4e0e-b29d-3d82fad6ef10"
		securityGroupID1 = "044f6d31-3938-4f09-ad45-47b661e2ba1c"
		securityGroupID2 = "427b77ee-40b7-4f1b-b025-72ad1a42ee51"

		defaultDescription = "Created by cluster-api-provider-openstack cluster test-cluster"
	)

	expectListExtensions := func(m *mock.MockNetworkClientMockRecorder) {
		trunkExtension := extensions.Extension{}
		trunkExtension.Alias = "trunk"
		m.ListExtensions().Return([]extensions.Extension{trunkExtension}, nil)
	}

	tests := []struct {
		name                 string
		spec                 infrav1.OpenStackMachineSpec
		managedSecurityGroup *string
		expectNetwork        func(m *mock.MockNetworkClientMockRecorder)
		want                 []infrav1.ResolvedPortSpec
		wantErr              bool
	}{
		{
			name: "No ports creates port on default network",
			spec: infrav1.OpenStackMachineSpec{},
			want: []infrav1.ResolvedPortSpec{
				{
					Name:        "test-instance-0",
					Description: defaultDescription,
					Tags:        []string{"test-tag"},
					NetworkID:   defaultNetworkID,
					FixedIPs: []infrav1.ResolvedFixedIP{
						{SubnetID: pointer.String(defaultSubnetID)},
					},
				},
			},
		},
		{
			name: "Nil network, no fixed IPs: cluster defaults",
			spec: infrav1.OpenStackMachineSpec{
				Ports: []infrav1.PortOpts{
					{
						NameSuffix: pointer.String("custom"),
						Network:    nil,
						FixedIPs:   nil,
					},
				},
			},
			want: []infrav1.ResolvedPortSpec{
				{
					Name:        "test-instance-custom",
					Description: defaultDescription,
					NetworkID:   defaultNetworkID,
					FixedIPs: []infrav1.ResolvedFixedIP{
						{
							SubnetID: pointer.String(defaultSubnetID),
						},
					},
					Tags: []string{"test-tag"},
				},
			},
		},
		{
			name: "Port inherits trunk from instance",
			spec: infrav1.OpenStackMachineSpec{
				Ports: []infrav1.PortOpts{
					{
						NameSuffix: pointer.String("custom"),
						Network:    nil,
						FixedIPs:   nil,
					},
				},
				Trunk: true,
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				expectListExtensions(m)
			},
			want: []infrav1.ResolvedPortSpec{
				{
					Name:        "test-instance-custom",
					Description: defaultDescription,
					NetworkID:   defaultNetworkID,
					FixedIPs: []infrav1.ResolvedFixedIP{
						{SubnetID: pointer.String(defaultSubnetID)},
					},
					Tags:  []string{"test-tag"},
					Trunk: pointer.Bool(true),
				},
			},
			wantErr: false,
		},
		{
			name: "Port overrides trunk from instance",
			spec: infrav1.OpenStackMachineSpec{
				Ports: []infrav1.PortOpts{
					{
						Trunk: pointer.Bool(true),
					},
				},
				Trunk: false,
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				expectListExtensions(m)
			},
			want: []infrav1.ResolvedPortSpec{
				{
					Name:        "test-instance-0",
					Description: defaultDescription,
					NetworkID:   defaultNetworkID,
					FixedIPs: []infrav1.ResolvedFixedIP{
						{SubnetID: pointer.String(defaultSubnetID)},
					},
					Tags:  []string{"test-tag"},
					Trunk: pointer.Bool(true),
				},
			},
		},
		{
			name: "Network defined by ID: no lookup",
			spec: infrav1.OpenStackMachineSpec{
				Ports: []infrav1.PortOpts{
					{
						Network: &infrav1.NetworkFilter{
							ID: networkID,
						},
					},
				},
			},
			want: []infrav1.ResolvedPortSpec{
				{
					NetworkID: networkID,

					// Defaults
					Name:        "test-instance-0",
					Description: defaultDescription,
					Tags:        []string{"test-tag"},
				},
			},
		},
		{
			name: "Network defined by filter: add ID from network lookup",
			spec: infrav1.OpenStackMachineSpec{
				Ports: []infrav1.PortOpts{
					{
						Network: &infrav1.NetworkFilter{
							Name: "test-network",
						},
					},
				},
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListNetwork(networks.ListOpts{Name: "test-network"}).Return([]networks.Network{
					{ID: networkID},
				}, nil)
			},
			want: []infrav1.ResolvedPortSpec{
				{
					NetworkID: networkID,

					// Defaults
					Name:        "test-instance-0",
					Description: defaultDescription,
					Tags:        []string{"test-tag"},
				},
			},
		},
		{
			name: "No network, fixed IP has subnet by ID: add ID from subnet",
			spec: infrav1.OpenStackMachineSpec{
				Ports: []infrav1.PortOpts{
					{
						FixedIPs: []infrav1.FixedIP{
							{
								Subnet: &infrav1.SubnetFilter{
									ID: subnetID1,
								},
							},
						},
					},
				},
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.GetSubnet(subnetID1).Return(&subnets.Subnet{ID: subnetID1, NetworkID: networkID}, nil)
			},
			want: []infrav1.ResolvedPortSpec{
				{
					NetworkID: networkID,
					FixedIPs: []infrav1.ResolvedFixedIP{
						{SubnetID: pointer.String(subnetID1)},
					},

					// Defaults
					Name:        "test-instance-0",
					Description: defaultDescription,
					Tags:        []string{"test-tag"},
				},
			},
		},
		{
			name: "No network, fixed IP has subnet by filter: add ID from subnet",
			spec: infrav1.OpenStackMachineSpec{
				Ports: []infrav1.PortOpts{
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
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListSubnet(subnets.ListOpts{Name: "test-subnet"}).Return([]subnets.Subnet{
					{ID: subnetID1, NetworkID: networkID},
				}, nil)
			},
			want: []infrav1.ResolvedPortSpec{
				{
					NetworkID: networkID,
					FixedIPs: []infrav1.ResolvedFixedIP{
						{
							SubnetID: pointer.String(subnetID1),
						},
					},

					// Defaults
					Name:        "test-instance-0",
					Description: defaultDescription,
					Tags:        []string{"test-tag"},
				},
			},
		},
		{
			name: "No network, fixed IP subnet returns no matches: error",
			spec: infrav1.OpenStackMachineSpec{
				Ports: []infrav1.PortOpts{
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
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListSubnet(subnets.ListOpts{Name: "test-subnet"}).Return([]subnets.Subnet{}, nil)
			},
			wantErr: true,
		},
		{
			name: "No network, only fixed IP subnet returns multiple matches: error",
			spec: infrav1.OpenStackMachineSpec{
				Ports: []infrav1.PortOpts{
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
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListSubnet(subnets.ListOpts{Name: "test-subnet"}).Return([]subnets.Subnet{
					{ID: subnetID1, NetworkID: networkID},
					{ID: "8008494c-301e-4e5c-951b-a8ab568447fd", NetworkID: "5d48bfda-db28-42ee-8374-50e13d1fe5ea"},
				}, nil)
			},
			wantErr: true,
		},
		{
			name: "No network, first fixed IP subnet returns multiple matches: used ID from second fixed IP",
			spec: infrav1.OpenStackMachineSpec{
				Ports: []infrav1.PortOpts{
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
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListSubnet(subnets.ListOpts{Name: "test-subnet1"}).Return([]subnets.Subnet{
					{ID: subnetID1, NetworkID: networkID},
					{ID: "8008494c-301e-4e5c-951b-a8ab568447fd", NetworkID: "5d48bfda-db28-42ee-8374-50e13d1fe5ea"},
				}, nil)
				m.ListSubnet(subnets.ListOpts{Name: "test-subnet2"}).Return([]subnets.Subnet{
					{ID: subnetID2, NetworkID: networkID},
				}, nil)
				// Fetch the first subnet again, this time with network ID from the second subnet
				m.ListSubnet(subnets.ListOpts{NetworkID: networkID, Name: "test-subnet1"}).Return([]subnets.Subnet{
					{ID: subnetID1, NetworkID: networkID},
				}, nil)
			},
			want: []infrav1.ResolvedPortSpec{
				{
					NetworkID: networkID,
					FixedIPs: []infrav1.ResolvedFixedIP{
						{
							SubnetID: pointer.String(subnetID1),
						},
						{
							SubnetID: pointer.String(subnetID2),
						},
					},

					// Defaults
					Name:        "test-instance-0",
					Description: defaultDescription,
					Tags:        []string{"test-tag"},
				},
			},
		},
		{
			name: "machine spec security groups added to defaults",
			spec: infrav1.OpenStackMachineSpec{
				SecurityGroups: []infrav1.SecurityGroupFilter{
					{Name: "test-security-group"},
				},
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListSecGroup(groups.ListOpts{Name: "test-security-group"}).Return([]groups.SecGroup{
					{ID: securityGroupID1},
				}, nil)
			},
			want: []infrav1.ResolvedPortSpec{
				{
					Name:      "test-instance-0",
					NetworkID: defaultNetworkID,
					FixedIPs: []infrav1.ResolvedFixedIP{
						{SubnetID: pointer.String(defaultSubnetID)},
					},
					Description:    defaultDescription,
					Tags:           []string{"test-tag"},
					SecurityGroups: []string{securityGroupID1},
				},
			},
		},
		{
			name: "port security groups override machine spec security groups",
			spec: infrav1.OpenStackMachineSpec{
				SecurityGroups: []infrav1.SecurityGroupFilter{
					{Name: "machine-security-group"},
				},
				Ports: []infrav1.PortOpts{
					{SecurityGroups: []infrav1.SecurityGroupFilter{{Name: "port-security-group"}}},
				},
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListSecGroup(groups.ListOpts{Name: "machine-security-group"}).Return([]groups.SecGroup{
					{ID: securityGroupID1},
				}, nil)
				m.ListSecGroup(groups.ListOpts{Name: "port-security-group"}).Return([]groups.SecGroup{
					{ID: securityGroupID2},
				}, nil)
			},
			want: []infrav1.ResolvedPortSpec{
				{
					Name:      "test-instance-0",
					NetworkID: defaultNetworkID,
					FixedIPs: []infrav1.ResolvedFixedIP{
						{SubnetID: pointer.String(defaultSubnetID)},
					},
					Description:    defaultDescription,
					Tags:           []string{"test-tag"},
					SecurityGroups: []string{securityGroupID2},
				},
			},
		},
		{
			name:                 "managed security group added to port",
			spec:                 infrav1.OpenStackMachineSpec{},
			managedSecurityGroup: pointer.String(securityGroupID1),
			want: []infrav1.ResolvedPortSpec{
				{
					Name:      "test-instance-0",
					NetworkID: defaultNetworkID,
					FixedIPs: []infrav1.ResolvedFixedIP{
						{SubnetID: pointer.String(defaultSubnetID)},
					},
					Description:    defaultDescription,
					Tags:           []string{"test-tag"},
					SecurityGroups: []string{securityGroupID1},
				},
			},
		},
		{
			name: "managed security group and machine security groups added to port",
			spec: infrav1.OpenStackMachineSpec{
				SecurityGroups: []infrav1.SecurityGroupFilter{{Name: "machine-security-group"}},
			},
			managedSecurityGroup: pointer.String(securityGroupID1),
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListSecGroup(groups.ListOpts{Name: "machine-security-group"}).Return([]groups.SecGroup{
					{ID: securityGroupID2},
				}, nil)
			},
			want: []infrav1.ResolvedPortSpec{
				{
					Name:      "test-instance-0",
					NetworkID: defaultNetworkID,
					FixedIPs: []infrav1.ResolvedFixedIP{
						{SubnetID: pointer.String(defaultSubnetID)},
					},
					Description:    defaultDescription,
					Tags:           []string{"test-tag"},
					SecurityGroups: []string{securityGroupID2, securityGroupID1},
				},
			},
		},
	}
	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			g := NewWithT(t)
			log := testr.New(t)

			mockClient := mock.NewMockNetworkClient(mockCtrl)
			if tt.expectNetwork != nil {
				tt.expectNetwork(mockClient.EXPECT())
			}
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			s := Service{
				client: mockClient,
				scope:  scope.NewWithLogger(mockScopeFactory, log),
			}

			defaultNetwork := &infrav1.NetworkStatusWithSubnets{
				NetworkStatus: infrav1.NetworkStatus{
					ID: defaultNetworkID,
				},
				Subnets: []infrav1.Subnet{
					{ID: defaultSubnetID},
				},
			}

			clusterName := "test-cluster"
			baseName := "test-instance"
			baseTags := []string{"test-tag"}
			got, err := s.ConstructPorts(&tt.spec, clusterName, baseName, defaultNetwork, tt.managedSecurityGroup, baseTags)
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
	tests := []struct {
		name         string
		instanceName string
		spec         *infrav1.PortOpts
		netIndex     int
		want         string
	}{
		{
			name:         "with nil PortOpts",
			instanceName: "test-1-instance",
			netIndex:     2,
			want:         "test-1-instance-2",
		},
		{
			name:         "with PortOpts name suffix",
			instanceName: "test-1-instance",
			spec: &infrav1.PortOpts{
				NameSuffix: pointer.String("foo"),
			},
			netIndex: 4,
			want:     "test-1-instance-foo",
		},
		{
			name:         "without PortOpts name suffix",
			instanceName: "test-1-instance",
			spec:         &infrav1.PortOpts{},
			netIndex:     4,
			want:         "test-1-instance-4",
		},
		{
			name:         "with PortOpts name suffix",
			instanceName: "test-1-instance",
			spec: &infrav1.PortOpts{
				NameSuffix: pointer.String("foo2"),
				Network:    &infrav1.NetworkFilter{ID: "bar"},
				ResolvedPortSpecFields: infrav1.ResolvedPortSpecFields{
					DisablePortSecurity: pointer.Bool(true),
				},
			},
			netIndex: 4,
			want:     "test-1-instance-foo2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPortName(tt.instanceName, tt.spec, tt.netIndex); got != tt.want {
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
		desiredPorts       []infrav1.ResolvedPortSpec
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
			desiredPorts: []infrav1.ResolvedPortSpec{
				{NetworkID: networkID1},
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
			desiredPorts: []infrav1.ResolvedPortSpec{
				{Name: "test-machine-0", NetworkID: networkID1},
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
			desiredPorts: []infrav1.ResolvedPortSpec{
				{Name: "test-machine-0", NetworkID: networkID1},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListPort(ports.ListOpts{Name: "test-machine-0", NetworkID: networkID1}).
					Return(nil, nil)
			},
			want: infrav1.DependentMachineResources{},
		},
		{
			testName: "2 desired ports, first in status, second exists: adopt second",
			desiredPorts: []infrav1.ResolvedPortSpec{
				{Name: "test-machine-0", NetworkID: networkID1},
				{Name: "test-machine-1", NetworkID: networkID2},
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
			desiredPorts: []infrav1.ResolvedPortSpec{
				{Name: "test-machine-0", NetworkID: networkID1},
				{Name: "test-machine-1", NetworkID: networkID2},
				{Name: "test-machine-2", NetworkID: networkID3},
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
		{
			testName: "3 desired ports with arbitrary names, first in status, second does not exist: ignore, do no look for third",
			desiredPorts: []infrav1.ResolvedPortSpec{
				{Name: "test-machine-foo", NetworkID: networkID1},
				{Name: "test-machine-bar", NetworkID: networkID2},
				{Name: "test-machine-baz", NetworkID: networkID3},
			},
			dependentResources: infrav1.DependentMachineResources{
				Ports: []infrav1.PortStatus{
					{
						ID: portID1,
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListPort(ports.ListOpts{Name: "test-machine-bar", NetworkID: networkID2}).
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
				tt.desiredPorts, &tt.dependentResources)
			if tt.wantErr {
				g.Expect(err).Error()
				return
			}

			g.Expect(tt.dependentResources).To(Equal(tt.want), cmp.Diff(&tt.dependentResources, tt.want))
		})
	}
}
