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

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	. "github.com/onsi/gomega" //nolint:revive
	"go.uber.org/mock/gomock"

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
