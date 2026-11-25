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
	"errors"
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/names"
)

func Test_GetOrCreateFloatingIP(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name    string
		ip      string
		tags    []string
		expect  func(m *mock.MockNetworkClientMockRecorder)
		want    *floatingips.FloatingIP
		wantErr bool
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
		{
			name: "creates floating IP with tags when standard-attr-tag is not supported",
			ip:   "192.168.111.0",
			tags: []string{"cluster-tag"},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				// standard-attr-tag support must be checked before the
				// floating IP is allocated, and ReplaceAllAttributesTags
				// must not be called since the extension is not advertised.
				m.ListExtensions().Return([]extensions.Extension{}, nil)

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
			name: "creates floating IP with tags when standard-attr-tag is supported",
			ip:   "192.168.111.0",
			tags: []string{"cluster-tag"},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				standardAttrTagExt := extensions.Extension{}
				standardAttrTagExt.Alias = "standard-attr-tag"
				m.ListExtensions().Return([]extensions.Extension{standardAttrTagExt}, nil)

				m.
					ListFloatingIP(floatingips.ListOpts{FloatingIP: "192.168.111.0"}).
					Return([]floatingips.FloatingIP{}, nil)
				m.
					CreateFloatingIP(floatingips.CreateOpts{
						FloatingIP:  "192.168.111.0",
						Description: "Created by cluster-api-provider-openstack cluster test-cluster",
					}).
					Return(&floatingips.FloatingIP{ID: "fip-1", FloatingIP: "192.168.111.0"}, nil)

				m.ReplaceAllAttributesTags("floatingips", "fip-1", attributestags.ReplaceAllOpts{
					Tags: []string{"cluster-tag"},
				}).Return([]string{"cluster-tag"}, nil)
			},
			want: &floatingips.FloatingIP{ID: "fip-1", FloatingIP: "192.168.111.0"},
		},
		{
			name: "tags requested, extension discovery fails before floating IP is allocated",
			ip:   "192.168.111.0",
			tags: []string{"cluster-tag"},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListFloatingIP(floatingips.ListOpts{FloatingIP: "192.168.111.0"}).
					Return([]floatingips.FloatingIP{}, nil)

				// ListExtensions fails before CreateFloatingIP is ever called.
				m.ListExtensions().Return(nil, errors.New("boom"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			log := testr.New(t)
			mockClient := mock.NewMockNetworkClient(mockCtrl)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			tt.expect(mockClient.EXPECT())

			openStackCluster := &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					Tags: tt.tags,
				},
				Status: infrav1.OpenStackClusterStatus{
					ExternalNetwork: &infrav1.NetworkStatus{
						ID: "",
					},
				},
			}

			scope := scope.NewWithLogger(mockScopeFactory, log)
			s := Service{
				scope:  scope,
				client: mockClient,
			}
			eventObject := infrav1.OpenStackMachine{}
			got, err := s.GetOrCreateFloatingIP(&eventObject, openStackCluster, "test-cluster", ptr.To(tt.ip))
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
				return
			}
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(got).To(Equal(tt.want))
		})
	}
}

func TestFloatingIPCreatedForClusterLifecycle(t *testing.T) {
	const (
		clusterResourceName = "test-cluster"
		floatingIP          = "192.168.111.0"
		floatingIPID        = "floating-ip-id"
		portID              = "port-id"
	)

	newService := func(t *testing.T) (*Service, *mock.MockNetworkClient) {
		t.Helper()
		mockCtrl := gomock.NewController(t)
		t.Cleanup(mockCtrl.Finish)
		mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
		return &Service{
			scope:  scope.NewWithLogger(mockScopeFactory, testr.New(t)),
			client: mockScopeFactory.NetworkClient,
		}, mockScopeFactory.NetworkClient
	}

	t.Run("deletes a CAPO-created floating IP", func(t *testing.T) {
		g := NewWithT(t)
		service, networkClient := newService(t)
		fip := floatingips.FloatingIP{
			ID:          floatingIPID,
			FloatingIP:  floatingIP,
			Description: names.GetDescription(clusterResourceName),
		}
		networkClient.EXPECT().ListFloatingIP(floatingips.ListOpts{FloatingIP: floatingIP}).Return([]floatingips.FloatingIP{fip}, nil)
		networkClient.EXPECT().DeleteFloatingIP(floatingIPID).Return(nil)

		handled, err := service.DeleteFloatingIPCreatedForCluster(&infrav1.OpenStackCluster{}, floatingIP, clusterResourceName)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(handled).To(BeTrue())
	})

	t.Run("does not delete a user-provided floating IP", func(t *testing.T) {
		g := NewWithT(t)
		service, networkClient := newService(t)
		fip := floatingips.FloatingIP{
			ID:          floatingIPID,
			FloatingIP:  floatingIP,
			Description: "created by a user",
		}
		networkClient.EXPECT().ListFloatingIP(floatingips.ListOpts{FloatingIP: floatingIP}).Return([]floatingips.FloatingIP{fip}, nil)

		handled, err := service.DeleteFloatingIPCreatedForCluster(&infrav1.OpenStackCluster{}, floatingIP, clusterResourceName)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(handled).To(BeFalse())
	})

	t.Run("disassociates a CAPO-created floating IP", func(t *testing.T) {
		g := NewWithT(t)
		service, networkClient := newService(t)
		fip := floatingips.FloatingIP{
			ID:          floatingIPID,
			FloatingIP:  floatingIP,
			Description: names.GetDescription(clusterResourceName),
			PortID:      portID,
		}
		networkClient.EXPECT().ListFloatingIP(floatingips.ListOpts{FloatingIP: floatingIP}).Return([]floatingips.FloatingIP{fip}, nil)
		networkClient.EXPECT().UpdateFloatingIP(floatingIPID, &floatingips.UpdateOpts{PortID: nil}).Return(nil, nil)
		networkClient.EXPECT().GetFloatingIP(floatingIPID).Return(&floatingips.FloatingIP{Status: "DOWN"}, nil)

		handled, err := service.DisassociateFloatingIPCreatedForCluster(&infrav1.OpenStackMachine{}, floatingIP, clusterResourceName)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(handled).To(BeTrue())
	})

	t.Run("does not disassociate a user-provided floating IP", func(t *testing.T) {
		g := NewWithT(t)
		service, networkClient := newService(t)
		fip := floatingips.FloatingIP{
			ID:          floatingIPID,
			FloatingIP:  floatingIP,
			Description: "created by a user",
			PortID:      portID,
		}
		networkClient.EXPECT().ListFloatingIP(floatingips.ListOpts{FloatingIP: floatingIP}).Return([]floatingips.FloatingIP{fip}, nil)

		handled, err := service.DisassociateFloatingIPCreatedForCluster(&infrav1.OpenStackMachine{}, floatingIP, clusterResourceName)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(handled).To(BeFalse())
	})
}
