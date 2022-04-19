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
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	. "github.com/onsi/gomega"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha5"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking/mock_networking"
)

func Test_GetOrCreateFloatingIP(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name   string
		ip     string
		expect func(m *mock_networking.MockNetworkClientMockRecorder)
		want   *floatingips.FloatingIP
	}{
		{
			name: "creates floating IP when one doesn't already exist",
			ip:   "192.168.111.0",
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
				m.
					ListFloatingIP(floatingips.ListOpts{FloatingIP: "192.168.111.0"}).
					Return([]floatingips.FloatingIP{}, nil)
				m.
					CreateFloatingIP(floatingips.CreateOpts{
						FloatingIP:  "192.168.111.0",
						Description: "Created by cluster-api-provider-openstack cluster test-cluster",
					}).
					Return(&floatingips.FloatingIP{FloatingIP: "192.168.111.0"}, nil)
			},
			want: &floatingips.FloatingIP{FloatingIP: "192.168.111.0"},
		},
		{
			name: "finds existing floating IP where one exists",
			ip:   "192.168.111.0",
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
				m.
					ListFloatingIP(floatingips.ListOpts{FloatingIP: "192.168.111.0"}).
					Return([]floatingips.FloatingIP{{FloatingIP: "192.168.111.0"}}, nil)
			},
			want: &floatingips.FloatingIP{FloatingIP: "192.168.111.0"},
		},
	}
	openStackCluster := &infrav1.OpenStackCluster{Status: infrav1.OpenStackClusterStatus{
		ExternalNetwork: &infrav1.Network{
			ID: "",
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			mockClient := mock_networking.NewMockNetworkClient(mockCtrl)
			tt.expect(mockClient.EXPECT())
			s := Service{
				client: mockClient,
			}
			eventObject := infrav1.OpenStackMachine{}
			got, err := s.GetOrCreateFloatingIP(&eventObject, openStackCluster, "test-cluster", tt.ip)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(got).To(Equal(tt.want))
		})
	}
}
