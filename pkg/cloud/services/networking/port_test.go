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

	"github.com/golang/mock/gomock"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsecurity"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	. "github.com/onsi/gomega"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha7"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
)

func Test_GetOrCreatePort(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Arbitrary GUIDs used in the tests
	netID := "7fd24ceb-788a-441f-ad0a-d8e2f5d31a1d"
	subnetID1 := "d9c88a6d-0b8c-48ff-8f0e-8d85a078c194"
	subnetID2 := "d9c2346d-05gc-48er-9ut4-ig83ayt8c7h4"
	portID1 := "50214c48-c09e-4a54-914f-97b40fd22802"
	portID2 := "4c096384-f0a5-466d-9534-06a7ed281a79"
	hostID := "825c1b11-3dca-4bfe-a2d8-a3cc1964c8d5"
	trunkID := "eb7541fa-5e2a-4cca-b2c3-dfa409b917ce"
	portSecurityGroupID := "f51d1206-fc5a-4f7a-a5c0-2e03e44e4dc0"

	// Other arbitrary variables passed in to the tests
	portSecurityGroupFilters := []infrav1.SecurityGroupFilter{{ID: portSecurityGroupID, Name: "port-secgroup"}}
	valueSpecs := map[string]string{"key": "value"}

	pointerToTrue := pointerTo(true)
	pointerToFalse := pointerTo(false)

	tests := []struct {
		name     string
		portName string
		port     infrav1.PortOpts
		tags     []string
		expect   func(m *mock.MockNetworkClientMockRecorder)
		// Note the 'wanted' port isn't so important, since it will be whatever we tell ListPort or CreatePort to return.
		// Mostly in this test suite, we're checking that ListPort/CreatePort is called with the expected port opts.
		want    *ports.Port
		wantErr bool
	}{
		{
			"gets and returns existing port if name matches",
			"foo-port-1",
			infrav1.PortOpts{
				Network: &infrav1.NetworkFilter{
					ID: netID,
				},
			},
			[]string{},
			func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListPort(ports.ListOpts{
						Name:      "foo-port-1",
						NetworkID: netID,
					}).Return([]ports.Port{{
					ID: portID1,
				}}, nil)
			},
			&ports.Port{
				ID: portID1,
			},
			false,
		},
		{
			"errors if multiple matching ports are found",
			"foo-port-1",
			infrav1.PortOpts{
				Network: &infrav1.NetworkFilter{
					ID: netID,
				},
			},
			[]string{},
			func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListPort(ports.ListOpts{
						Name:      "foo-port-1",
						NetworkID: netID,
					}).Return([]ports.Port{
					{
						ID:        portID1,
						NetworkID: netID,
						Name:      "foo-port-1",
					},
					{
						ID:        portID2,
						NetworkID: netID,
						Name:      "foo-port-2",
					},
				}, nil)
			},
			nil,
			true,
		},
		{
			"creates port with defaults (description) if not specified in portOpts",
			"foo-port-1",
			infrav1.PortOpts{
				Network: &infrav1.NetworkFilter{
					ID: netID,
				},
			},
			[]string{},
			func(m *mock.MockNetworkClientMockRecorder) {
				// No ports found
				m.
					ListPort(ports.ListOpts{
						Name:      "foo-port-1",
						NetworkID: netID,
					}).Return([]ports.Port{}, nil)
				m.
					CreatePort(portsbinding.CreateOptsExt{
						CreateOptsBuilder: ports.CreateOpts{
							Name:                "foo-port-1",
							Description:         "Created by cluster-api-provider-openstack cluster test-cluster",
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
				NameSuffix:   "bar",
				Description:  "this is a test port",
				MACAddress:   "fe:fe:fe:fe:fe:fe",
				AdminStateUp: pointerToTrue,
				FixedIPs: []infrav1.FixedIP{{
					Subnet: &infrav1.SubnetFilter{
						Name: "subnetFoo",
					},
					IPAddress: "192.168.0.50",
				}, {IPAddress: "192.168.1.50"}},
				AllowedAddressPairs: []infrav1.AddressPair{{
					IPAddress:  "10.10.10.10",
					MACAddress: "f1:f1:f1:f1:f1:f1",
				}},
				HostID:   hostID,
				VNICType: "direct",
				Profile: infrav1.BindingProfile{
					OVSHWOffload: true,
					TrustedVF:    true,
				},
				DisablePortSecurity: pointerToFalse,
				Tags:                []string{"my-port-tag"},
			},
			nil,
			func(m *mock.MockNetworkClientMockRecorder) {
				portCreateOpts := ports.CreateOpts{
					NetworkID:    netID,
					Name:         "foo-port-bar",
					Description:  "this is a test port",
					AdminStateUp: pointerToTrue,
					MACAddress:   "fe:fe:fe:fe:fe:fe",
					FixedIPs: []ports.IP{
						{
							SubnetID:  subnetID1,
							IPAddress: "192.168.0.50",
						}, {
							IPAddress: "192.168.1.50",
						},
					},
					AllowedAddressPairs: []ports.AddressPair{{
						IPAddress:  "10.10.10.10",
						MACAddress: "f1:f1:f1:f1:f1:f1",
					}},
				}
				portsecurityCreateOptsExt := portsecurity.PortCreateOptsExt{
					CreateOptsBuilder:   portCreateOpts,
					PortSecurityEnabled: pointerToTrue,
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
					ListPort(ports.ListOpts{
						Name:      "foo-port-bar",
						NetworkID: netID,
					}).Return([]ports.Port{}, nil)
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
				NameSuffix:  "foo-port-bar",
				Description: "this is a test port",
				FixedIPs: []infrav1.FixedIP{{
					Subnet: &infrav1.SubnetFilter{
						Tags: "Foo",
					},
					IPAddress: "192.168.0.50",
				}},
			},
			nil,
			func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListPort(ports.ListOpts{
						Name:      "foo-port-bar",
						NetworkID: netID,
					}).Return([]ports.Port{}, nil)
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
			"fails to create port with specified portOpts if security group filters are specified",
			"foo-port-1",
			infrav1.PortOpts{
				Network: &infrav1.NetworkFilter{
					ID: netID,
				},
				SecurityGroupFilters: portSecurityGroupFilters,
			},
			[]string{},
			func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListPort(ports.ListOpts{
						Name:      "foo-port-1",
						NetworkID: netID,
					}).Return([]ports.Port{}, nil)
			},
			nil,
			true,
		},
		{
			"creates port with instance tags when port tags aren't specified",
			"foo-port-1",
			infrav1.PortOpts{
				Network: &infrav1.NetworkFilter{
					ID: netID,
				},
			},
			[]string{"my-instance-tag"},
			func(m *mock.MockNetworkClientMockRecorder) {
				// No ports found
				m.
					ListPort(ports.ListOpts{
						Name:      "foo-port-1",
						NetworkID: netID,
					}).Return([]ports.Port{}, nil)
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
			[]string{"my-instance-tag"},
			func(m *mock.MockNetworkClientMockRecorder) {
				// No ports found
				m.
					ListPort(ports.ListOpts{
						Name:      "foo-port-1",
						NetworkID: netID,
					}).Return([]ports.Port{}, nil)
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
				Trunk: pointerToTrue,
			},
			[]string{"my-tag"},
			func(m *mock.MockNetworkClientMockRecorder) {
				// No ports found
				m.
					ListPort(ports.ListOpts{
						Name:      "foo-port-1",
						NetworkID: netID,
					}).Return([]ports.Port{}, nil)
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
			func(m *mock.MockNetworkClientMockRecorder) {
				// No ports found
				m.
					ListPort(ports.ListOpts{
						Name:      "foo-port-1",
						NetworkID: netID,
					}).Return([]ports.Port{}, nil)
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
				PropagateUplinkStatus: pointerToTrue,
			},
			[]string{},
			func(m *mock.MockNetworkClientMockRecorder) {
				// No ports found
				m.
					ListPort(ports.ListOpts{
						Name:      "foo-port-1",
						NetworkID: netID,
					}).Return([]ports.Port{}, nil)
				m.
					CreatePort(portsbinding.CreateOptsExt{
						CreateOptsBuilder: ports.CreateOpts{
							Name:                  "foo-port-1",
							Description:           "Created by cluster-api-provider-openstack cluster test-cluster",
							NetworkID:             netID,
							AllowedAddressPairs:   []ports.AddressPair{},
							PropagateUplinkStatus: pointerToTrue,
						},
					}).Return(&ports.Port{ID: portID1, PropagateUplinkStatus: *pointerToTrue}, nil)
			},
			&ports.Port{ID: portID1, PropagateUplinkStatus: *pointerToTrue},
			false,
		},
	}

	eventObject := &infrav1.OpenStackMachine{}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			mockClient := mock.NewMockNetworkClient(mockCtrl)
			tt.expect(mockClient.EXPECT())
			s := Service{
				client: mockClient,
			}
			got, err := s.GetOrCreatePort(
				eventObject,
				"test-cluster",
				tt.portName,
				&tt.port,
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

func Test_GarbageCollectErrorInstancesPort(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	instanceName := "foo"
	portID1 := "dc6e0ae3-dad6-4240-a9cb-e541916f20d3"
	portID2 := "a38ab1cb-c2cc-4c1b-9d1d-696ec73356d2"
	portName1 := GetPortName(instanceName, nil, 0)
	portName2 := GetPortName(instanceName, nil, 1)

	tests := []struct {
		// man is the name of the test.
		name string
		// expect allows definition of any expected calls to the mock.
		expect func(m *mock.MockNetworkClientMockRecorder)
		// portOpts defines the instance ports as defined in the OSM spec.
		portOpts []infrav1.PortOpts
		// wantErr defines whether the test is supposed to fail.
		wantErr bool
	}{
		{
			name: "garbage collects all ports for an instance",
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				o1 := ports.ListOpts{
					Name: portName1,
				}
				p1 := []ports.Port{
					{
						ID:   portID1,
						Name: portName1,
					},
				}
				m.ListPort(o1).Return(p1, nil)
				m.DeletePort(portID1)
				o2 := ports.ListOpts{
					Name: portName2,
				}
				p2 := []ports.Port{
					{
						ID:   portID2,
						Name: portName2,
					},
				}

				m.ListPort(o2).Return(p2, nil)
				m.DeletePort(portID2)
			},
			portOpts: []infrav1.PortOpts{
				{},
				{},
			},
			wantErr: false,
		},
	}

	eventObject := &infrav1.OpenStackMachine{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			mockClient := mock.NewMockNetworkClient(mockCtrl)
			tt.expect(mockClient.EXPECT())
			s := Service{
				client: mockClient,
			}
			err := s.GarbageCollectErrorInstancesPort(
				eventObject,
				instanceName,
				tt.portOpts,
			)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
		})
	}
}

func pointerTo(b bool) *bool {
	return &b
}

func Test_GetPortIDsFromInstanceID(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	instanceID := "12345"
	expectedPortIDs := []string{"port1", "port2", "port3"}

	mockClient := mock.NewMockNetworkClient(mockCtrl)
	mockClient.EXPECT().ListPort(ports.ListOpts{
		DeviceID: instanceID,
	}).Return([]ports.Port{
		{ID: "port1"},
		{ID: "port2"},
		{ID: "port3"},
	}, nil)

	service := &Service{
		client: mockClient,
	}

	portIDs, err := service.GetPortIDsFromInstanceID(instanceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(portIDs) != len(expectedPortIDs) {
		t.Fatalf("unexpected number of port IDs, got: %d, want: %d", len(portIDs), len(expectedPortIDs))
	}

	for i, portID := range portIDs {
		if portID != expectedPortIDs[i] {
			t.Errorf("unexpected port ID at index %d, got: %s, want: %s", i, portID, expectedPortIDs[i])
		}
	}
}
