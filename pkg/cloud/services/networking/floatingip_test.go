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
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	. "github.com/onsi/gomega" //nolint:revive
	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

func Test_GetOrCreateFloatingIP(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name   string
		ip     string
		expect func(m *mock.MockNetworkClientMockRecorder)
		want   *floatingips.FloatingIP
	}{
		{
			name: "creates floating IP when one doesn't already exist",
			ip:   "192.168.111.0",
			expect: func(m *mock.MockNetworkClientMockRecorder) {
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
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListFloatingIP(floatingips.ListOpts{FloatingIP: "192.168.111.0"}).
					Return([]floatingips.FloatingIP{{FloatingIP: "192.168.111.0"}}, nil)
			},
			want: &floatingips.FloatingIP{FloatingIP: "192.168.111.0"},
		},
	}
	openStackCluster := &infrav1.OpenStackCluster{Status: infrav1.OpenStackClusterStatus{
		ExternalNetwork: &infrav1.NetworkStatus{
			ID: "",
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			log := testr.New(t)
			mockClient := mock.NewMockNetworkClient(mockCtrl)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			tt.expect(mockClient.EXPECT())

			scope := scope.NewWithLogger(mockScopeFactory, log)
			s := Service{
				scope:  scope,
				client: mockClient,
			}
			eventObject := infrav1.OpenStackMachine{}
			got, err := s.GetOrCreateFloatingIP(&eventObject, openStackCluster, "test-cluster", ptr.To(tt.ip))
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(got).To(Equal(tt.want))
		})
	}
}
