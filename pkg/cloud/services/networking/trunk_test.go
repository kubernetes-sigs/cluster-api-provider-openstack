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
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	. "github.com/onsi/gomega" //nolint:revive
	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"

	"sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
)

func Test_GetOrCreateTrunk(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	const portID = "021e5dbe-a27b-4824-839e-239d5a027c7f"

	tests := []struct {
		name   string
		port   *ports.Port
		expect func(m *mock.MockNetworkClientMockRecorder)
		// Note the 'wanted' port isn't so important, since it will be whatever we tell ListPort or CreatePort to return.
		// Mostly in this test suite, we're checking that ListPort/CreatePort is called with the expected port opts.
		want    *trunks.Trunk
		wantErr bool
	}{
		{
			name: "return trunk if found",
			port: &ports.Port{
				ID:   portID,
				Name: "trunk-1",
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListTrunk(trunks.ListOpts{
						Name:   "trunk-1",
						PortID: portID,
					}).Return([]trunks.Trunk{{
					Name: "trunk-1",
					ID:   portID,
				}}, nil)
			},
			want: &trunks.Trunk{
				Name: "trunk-1",
				ID:   portID,
			},
			wantErr: false,
		},
		{
			name: "creates trunk if not found",
			port: &ports.Port{
				ID:          portID,
				Name:        "trunk-1",
				Description: "Created by cluster-api-provider-openstack cluster test-cluster",
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				// No ports found
				m.
					ListTrunk(trunks.ListOpts{
						Name:   "trunk-1",
						PortID: portID,
					}).Return([]trunks.Trunk{}, nil)
				m.
					CreateTrunk(trunks.CreateOpts{
						Name:        "trunk-1",
						PortID:      portID,
						Description: "Created by cluster-api-provider-openstack cluster test-cluster",
					},
					).Return(&trunks.Trunk{Name: "trunk-1", ID: "port-1"}, nil)
			},
			want:    &trunks.Trunk{Name: "trunk-1", ID: "port-1"},
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
			got, err := s.getOrCreateTrunkForPort(eventObject, tt.port)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
			g.Expect(got).To(Equal(tt.want))
		})
	}
}

func Test_EnsureTrunkSubPorts(t *testing.T) {
	// Arbitrary values used in the tests
	const (
		netID       = "7fd24ceb-788a-441f-ad0a-d8e2f5d31a1d"
		portID      = "50214c48-c09e-4a54-914f-97b40fd22802"
		trunkPortID = "54fe7034-8e72-4e8d-92af-a9f02aed7f6f"
		trunkID     = "2efdd5e5-c85b-419f-8c59-678b5fccbdba"
		subportID   = "d002cdd7-7343-4376-997b-7f15cf97b89c"

		segmentationID   = 300
		segmentationType = "vlan"
	)

	tests := []struct {
		name            string
		ports           []infrav1.ResolvedPortSpec
		serverResources *v1alpha1.ServerResources
		expect          func(m *mock.MockNetworkClientMockRecorder, g Gomega)
		// Note the 'wanted' port isn't so important, since it will be whatever we tell ListPort or EnsurePort to return.
		// Mostly in this test suite, we're checking that EnsurePort is called with the expected port opts.
		wantResources *v1alpha1.ServerResources
		wantErr       bool
	}{
		{
			name: "creates subports and assign to trunk with tag",
			ports: []infrav1.ResolvedPortSpec{
				{
					Trunk: ptr.To(true),
					Subports: []infrav1.ResolvedSubportSpec{
						{
							SegmentationID:   segmentationID,
							SegmentationType: segmentationType,
							CommonResolvedPortSpec: infrav1.CommonResolvedPortSpec{
								Name:        "foo-port-1",
								Description: "Created by cluster-api-provider-openstack cluster test-cluster",
								NetworkID:   netID,
							},
						},
					},
				},
			},
			serverResources: &v1alpha1.ServerResources{
				Ports: []infrav1.PortStatus{
					{ID: trunkPortID},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder, g Gomega) {
				var expectedCreateOpts ports.CreateOptsBuilder
				expectedCreateOpts = ports.CreateOpts{
					Name:        "foo-port-1",
					Description: "Created by cluster-api-provider-openstack cluster test-cluster",
					NetworkID:   netID,
				}
				expectedCreateOpts = portsbinding.CreateOptsExt{
					CreateOptsBuilder: expectedCreateOpts,
				}

				m.ListTrunk(trunks.ListOpts{PortID: trunkPortID}).Return([]trunks.Trunk{{ID: trunkID}}, nil)
				m.ListTrunkSubports(trunkID).Return(nil, nil)
				m.ListPort(ports.ListOpts{
					Name:      "foo-port-1",
					NetworkID: netID,
				}).Return(nil, nil)
				// The following allows us to use gomega to
				// compare the argument instead of gomock.
				// Gomock's output in the case of a mismatch is
				// not usable for this struct.
				m.CreatePort(gomock.Any()).DoAndReturn(func(builder ports.CreateOptsBuilder) (*ports.Port, error) {
					gotCreateOpts := builder.(portsbinding.CreateOptsExt)
					g.Expect(gotCreateOpts).To(Equal(expectedCreateOpts), cmp.Diff(gotCreateOpts, expectedCreateOpts))
					return &ports.Port{ID: subportID}, nil
				})
				m.AddSubports(
					trunkID,
					trunks.AddSubportsOpts{
						Subports: []trunks.Subport{
							{
								SegmentationID:   segmentationID,
								SegmentationType: segmentationType,
								PortID:           subportID,
							},
						},
					},
				)
			},
			wantResources: &v1alpha1.ServerResources{
				Ports: []infrav1.PortStatus{
					{
						ID:       trunkPortID,
						Subports: []infrav1.SubPortStatus{{ID: subportID}},
					},
				},
			},
		},
		{
			name: "error if resources.Ports is not defined",
			ports: []infrav1.ResolvedPortSpec{
				{
					Trunk: ptr.To(true),
					Subports: []infrav1.ResolvedSubportSpec{
						{
							SegmentationID:   segmentationID,
							SegmentationType: segmentationType,
							CommonResolvedPortSpec: infrav1.CommonResolvedPortSpec{
								Name:        "foo-port-1",
								Description: "Created by cluster-api-provider-openstack cluster test-cluster",
								NetworkID:   netID,
							},
						},
					},
				},
			},
			serverResources: &v1alpha1.ServerResources{
				Ports: nil,
			},
			expect: func(m *mock.MockNetworkClientMockRecorder, g Gomega) {
				// No calls expected
			},
			wantResources: &v1alpha1.ServerResources{
				Ports: nil,
			},
			wantErr: true,
		},
		{
			name: "error if ListTrunk returns error",
			ports: []infrav1.ResolvedPortSpec{
				{
					Trunk: ptr.To(true),
					Subports: []infrav1.ResolvedSubportSpec{
						{
							SegmentationID:   segmentationID,
							SegmentationType: segmentationType,
							CommonResolvedPortSpec: infrav1.CommonResolvedPortSpec{
								Name:        "foo-port-1",
								Description: "Created by cluster-api-provider-openstack cluster test-cluster",
								NetworkID:   netID,
							},
						},
					},
				},
			},
			serverResources: &v1alpha1.ServerResources{
				Ports: []infrav1.PortStatus{
					{ID: trunkPortID},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder, g Gomega) {
				m.ListTrunk(trunks.ListOpts{PortID: trunkPortID}).Return(nil, fmt.Errorf("list trunk error"))
			},
			wantResources: &v1alpha1.ServerResources{
				Ports: []infrav1.PortStatus{
					{ID: trunkPortID},
				},
			},
			wantErr: true,
		},
		{
			name: "error if multiple trunks found",
			ports: []infrav1.ResolvedPortSpec{
				{
					Trunk: ptr.To(true),
					Subports: []infrav1.ResolvedSubportSpec{
						{
							SegmentationID:   segmentationID,
							SegmentationType: segmentationType,
							CommonResolvedPortSpec: infrav1.CommonResolvedPortSpec{
								Name:        "foo-port-1",
								Description: "Created by cluster-api-provider-openstack cluster test-cluster",
								NetworkID:   netID,
							},
						},
					},
				},
			},
			serverResources: &v1alpha1.ServerResources{
				Ports: []infrav1.PortStatus{
					{ID: trunkPortID},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder, g Gomega) {
				m.ListTrunk(trunks.ListOpts{PortID: trunkPortID}).Return([]trunks.Trunk{{ID: trunkID}, {ID: "another-trunk"}}, nil)
			},
			wantResources: &v1alpha1.ServerResources{
				Ports: []infrav1.PortStatus{
					{ID: trunkPortID},
				},
			},
			wantErr: true,
		},
		{
			name: "error if no trunks found",
			ports: []infrav1.ResolvedPortSpec{
				{
					Trunk: ptr.To(true),
					Subports: []infrav1.ResolvedSubportSpec{
						{
							SegmentationID:   segmentationID,
							SegmentationType: segmentationType,
							CommonResolvedPortSpec: infrav1.CommonResolvedPortSpec{
								Name:        "foo-port-1",
								Description: "Created by cluster-api-provider-openstack cluster test-cluster",
								NetworkID:   netID,
							},
						},
					},
				},
			},
			serverResources: &v1alpha1.ServerResources{
				Ports: []infrav1.PortStatus{
					{ID: trunkPortID},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder, g Gomega) {
				m.ListTrunk(trunks.ListOpts{PortID: trunkPortID}).Return([]trunks.Trunk{}, nil)
			},
			wantResources: &v1alpha1.ServerResources{
				Ports: []infrav1.PortStatus{
					{ID: trunkPortID},
				},
			},
			wantErr: true,
		},
		{
			name: "error from ListTrunkSubports",
			ports: []infrav1.ResolvedPortSpec{
				{
					Trunk: ptr.To(true),
					Subports: []infrav1.ResolvedSubportSpec{
						{
							SegmentationID:   segmentationID,
							SegmentationType: segmentationType,
							CommonResolvedPortSpec: infrav1.CommonResolvedPortSpec{
								Name:        "foo-port-1",
								Description: "Created by cluster-api-provider-openstack cluster test-cluster",
								NetworkID:   netID,
							},
						},
					},
				},
			},
			serverResources: &v1alpha1.ServerResources{
				Ports: []infrav1.PortStatus{
					{ID: trunkPortID},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder, g Gomega) {
				m.ListTrunk(trunks.ListOpts{PortID: trunkPortID}).Return([]trunks.Trunk{{ID: trunkID}}, nil)
				m.ListTrunkSubports(trunkID).Return(nil, fmt.Errorf("subports error"))
			},
			wantResources: &v1alpha1.ServerResources{
				Ports: []infrav1.PortStatus{
					{ID: trunkPortID},
				},
			},
			wantErr: true,
		},
		{
			name: "error from AddSubports",
			ports: []infrav1.ResolvedPortSpec{
				{
					Trunk: ptr.To(true),
					Subports: []infrav1.ResolvedSubportSpec{
						{
							SegmentationID:   segmentationID,
							SegmentationType: segmentationType,
							CommonResolvedPortSpec: infrav1.CommonResolvedPortSpec{
								Name:        "foo-port-1",
								Description: "Created by cluster-api-provider-openstack cluster test-cluster",
								NetworkID:   netID,
							},
						},
					},
				},
			},
			serverResources: &v1alpha1.ServerResources{
				Ports: []infrav1.PortStatus{
					{ID: trunkPortID},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder, g Gomega) {
				var expectedCreateOpts ports.CreateOptsBuilder
				expectedCreateOpts = ports.CreateOpts{
					Name:        "foo-port-1",
					Description: "Created by cluster-api-provider-openstack cluster test-cluster",
					NetworkID:   netID,
				}
				expectedCreateOpts = portsbinding.CreateOptsExt{
					CreateOptsBuilder: expectedCreateOpts,
				}

				m.ListTrunk(trunks.ListOpts{PortID: trunkPortID}).Return([]trunks.Trunk{{ID: trunkID}}, nil)
				m.ListTrunkSubports(trunkID).Return(nil, nil)
				m.ListPort(ports.ListOpts{
					Name:      "foo-port-1",
					NetworkID: netID,
				}).Return(nil, nil)
				m.CreatePort(gomock.Any()).DoAndReturn(func(builder ports.CreateOptsBuilder) (*ports.Port, error) {
					gotCreateOpts := builder.(portsbinding.CreateOptsExt)
					g.Expect(gotCreateOpts).To(Equal(expectedCreateOpts), cmp.Diff(gotCreateOpts, expectedCreateOpts))
					return &ports.Port{ID: subportID}, nil
				})
				m.AddSubports(
					trunkID,
					trunks.AddSubportsOpts{
						Subports: []trunks.Subport{
							{
								SegmentationID:   segmentationID,
								SegmentationType: segmentationType,
								PortID:           subportID,
							},
						},
					},
				).Return(nil, fmt.Errorf("add subports error"))
			},
			wantResources: &v1alpha1.ServerResources{
				Ports: []infrav1.PortStatus{
					{ID: trunkPortID, Subports: []infrav1.SubPortStatus{{ID: subportID}}},
				},
			},
			wantErr: true,
		},
		{
			name: "error creating port for subport",
			ports: []infrav1.ResolvedPortSpec{
				{
					Trunk: ptr.To(true),
					Subports: []infrav1.ResolvedSubportSpec{
						{
							SegmentationID:   segmentationID,
							SegmentationType: segmentationType,
							CommonResolvedPortSpec: infrav1.CommonResolvedPortSpec{
								Name:        "foo-port-1",
								Description: "Created by cluster-api-provider-openstack cluster test-cluster",
								NetworkID:   netID,
							},
						},
					},
				},
			},
			serverResources: &v1alpha1.ServerResources{
				Ports: []infrav1.PortStatus{
					{ID: trunkPortID},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder, g Gomega) {
				var expectedCreateOpts ports.CreateOptsBuilder
				expectedCreateOpts = ports.CreateOpts{
					Name:        "foo-port-1",
					Description: "Created by cluster-api-provider-openstack cluster test-cluster",
					NetworkID:   netID,
				}
				expectedCreateOpts = portsbinding.CreateOptsExt{
					CreateOptsBuilder: expectedCreateOpts,
				}

				m.ListTrunk(trunks.ListOpts{PortID: trunkPortID}).Return([]trunks.Trunk{{ID: trunkID}}, nil)
				m.ListTrunkSubports(trunkID).Return(nil, nil)
				m.ListPort(ports.ListOpts{
					Name:      "foo-port-1",
					NetworkID: netID,
				}).Return(nil, nil)
				// The following allows us to use gomega to
				// compare the argument instead of gomock.
				// Gomock's output in the case of a mismatch is
				// not usable for this struct.
				m.CreatePort(gomock.Any()).DoAndReturn(func(builder ports.CreateOptsBuilder) (*ports.Port, error) {
					gotCreateOpts := builder.(portsbinding.CreateOptsExt)
					g.Expect(gotCreateOpts).To(Equal(expectedCreateOpts), cmp.Diff(gotCreateOpts, expectedCreateOpts))
					return nil, fmt.Errorf("ensure port error")
				})
			},
			wantResources: &v1alpha1.ServerResources{
				Ports: []infrav1.PortStatus{
					{
						ID: trunkPortID,
					},
				},
			},
			wantErr: true,
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
			err := s.EnsureTrunkSubPorts(
				eventObject,
				tt.ports,
				tt.serverResources,
			)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
			g.Expect(tt.serverResources).To(Equal(tt.wantResources))
		})
	}
}
