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
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/trunks"
	. "github.com/onsi/gomega"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha5"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking/mock_networking"
)

func Test_GetOrCreateTrunk(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name      string
		trunkName string
		portID    string
		expect    func(m *mock_networking.MockNetworkClientMockRecorder)
		// Note the 'wanted' port isn't so important, since it will be whatever we tell ListPort or CreatePort to return.
		// Mostly in this test suite, we're checking that ListPort/CreatePort is called with the expected port opts.
		want    *trunks.Trunk
		wantErr bool
	}{
		{
			"return trunk if found",
			"trunk-1",
			"port-1",
			func(m *mock_networking.MockNetworkClientMockRecorder) {
				m.
					ListTrunk(trunks.ListOpts{
						Name:   "trunk-1",
						PortID: "port-1",
					}).Return([]trunks.Trunk{{
					Name: "trunk-1",
					ID:   "port-1",
				}}, nil)
			},
			&trunks.Trunk{Name: "trunk-1", ID: "port-1"},
			false,
		},
		{
			"creates trunk if not found",
			"trunk-1",
			"port-1",
			func(m *mock_networking.MockNetworkClientMockRecorder) {
				// No ports found
				m.
					ListTrunk(trunks.ListOpts{
						Name:   "trunk-1",
						PortID: "port-1",
					}).Return([]trunks.Trunk{}, nil)
				m.
					CreateTrunk(trunks.CreateOpts{
						Name:        "trunk-1",
						PortID:      "port-1",
						Description: "Created by cluster-api-provider-openstack cluster test-cluster",
					},
					).Return(&trunks.Trunk{Name: "trunk-1", ID: "port-1"}, nil)
			},
			&trunks.Trunk{Name: "trunk-1", ID: "port-1"},
			false,
		},
	}

	eventObject := &infrav1.OpenStackMachine{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			mockClient := mock_networking.NewMockNetworkClient(mockCtrl)
			tt.expect(mockClient.EXPECT())
			s := Service{
				client: mockClient,
			}
			got, err := s.getOrCreateTrunk(
				eventObject,
				"test-cluster",
				tt.trunkName,
				tt.portID,
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
