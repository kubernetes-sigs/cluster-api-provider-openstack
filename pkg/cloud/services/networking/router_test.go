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

package networking

import (
	"errors"
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/names"
)

func TestService_DeleteRouter(t *testing.T) {
	const (
		clusterResourceName = "test-cluster"
		resourceName        = "k8s-clusterapi-cluster-test-cluster"

		routerID = "38052015-5cbc-4cb4-8e45-445d53260f60"
		subnetID = "283ee906-0072-4c81-92fb-9858e90c3c4e"
	)
	tests := []struct {
		name             string
		openStackCluster infrav1.OpenStackCluster
		expect           func(g Gomega, m *mock.MockNetworkClientMockRecorder)
		wantErr          bool
	}{
		{
			name: "Managed router in status exists",
			openStackCluster: infrav1.OpenStackCluster{
				Status: infrav1.OpenStackClusterStatus{
					Router: &infrav1.Router{
						ID: routerID,
					},
				},
			},
			expect: func(g Gomega, m *mock.MockNetworkClientMockRecorder) {
				// Get by ID in status
				// Remove subnet interfaces
				// Delete router
				m.GetRouter(routerID).Return(&routers.Router{ID: routerID}, nil)
				m.ListSubnet(gomock.Any()).DoAndReturn(func(opts subnets.ListOpts) ([]subnets.Subnet, error) {
					g.Expect(opts.Name).To(Equal(resourceName))
					return []subnets.Subnet{
						{ID: subnetID, Name: resourceName},
					}, nil
				})
				m.RemoveRouterInterface(routerID, routers.RemoveInterfaceOpts{SubnetID: subnetID}).Return(&routers.InterfaceInfo{}, nil)
				m.DeleteRouter(routerID).Return(nil)
			},
		},
		{
			name: "Managed router in status does not exist",
			openStackCluster: infrav1.OpenStackCluster{
				Status: infrav1.OpenStackClusterStatus{
					Router: &infrav1.Router{
						ID: routerID,
					},
				},
			},
			expect: func(_ Gomega, m *mock.MockNetworkClientMockRecorder) {
				// Get by ID in status returns 404
				// No further action
				m.GetRouter(routerID).Return(&routers.Router{ID: routerID}, gophercloud.ErrUnexpectedResponseCode{Actual: 404})
			},
		},
		{
			name:             "Managed router not in status exists",
			openStackCluster: infrav1.OpenStackCluster{},
			expect: func(g Gomega, m *mock.MockNetworkClientMockRecorder) {
				// Lookup by cluster resource name
				// Remove subnet interfaces
				// Delete router
				m.ListRouter(gomock.Any()).DoAndReturn(func(opts routers.ListOpts) ([]routers.Router, error) {
					g.Expect(opts.Name).To(Equal(resourceName))
					return []routers.Router{
						{ID: routerID, Name: resourceName},
					}, nil
				})
				m.ListSubnet(gomock.Any()).DoAndReturn(func(opts subnets.ListOpts) ([]subnets.Subnet, error) {
					g.Expect(opts.Name).To(Equal(resourceName))
					return []subnets.Subnet{
						{ID: subnetID, Name: resourceName},
					}, nil
				})
				m.RemoveRouterInterface(routerID, routers.RemoveInterfaceOpts{SubnetID: subnetID}).Return(&routers.InterfaceInfo{}, nil)
				m.DeleteRouter(routerID).Return(nil)
			},
		},
		{
			name:             "Managed router not in status does not exist",
			openStackCluster: infrav1.OpenStackCluster{},
			expect: func(g Gomega, m *mock.MockNetworkClientMockRecorder) {
				// Lookup by cluster resource name returns 404
				// No further action
				m.ListRouter(gomock.Any()).DoAndReturn(func(opts routers.ListOpts) ([]routers.Router, error) {
					g.Expect(opts.Name).To(Equal(resourceName))
					return []routers.Router{}, nil
				})
			},
		},
		{
			name: "Unmanaged router in status exists",
			openStackCluster: infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					Router: &infrav1.RouterParam{
						Filter: &infrav1.RouterFilter{
							Name: "my-router",
						},
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Router: &infrav1.Router{
						ID: routerID,
					},
				},
			},
			expect: func(g Gomega, m *mock.MockNetworkClientMockRecorder) {
				// Get by ID in status
				// Remove subnet interfaces
				// Don't delete unmanaged router
				m.GetRouter(routerID).Return(&routers.Router{ID: routerID}, nil)
				m.ListSubnet(gomock.Any()).DoAndReturn(func(opts subnets.ListOpts) ([]subnets.Subnet, error) {
					g.Expect(opts.Name).To(Equal(resourceName))
					return []subnets.Subnet{
						{ID: subnetID, Name: resourceName},
					}, nil
				})
				m.RemoveRouterInterface(routerID, routers.RemoveInterfaceOpts{SubnetID: subnetID}).Return(&routers.InterfaceInfo{}, nil)
			},
		},
		{
			name: "Unmanaged router in status does not exist",
			openStackCluster: infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					Router: &infrav1.RouterParam{
						Filter: &infrav1.RouterFilter{
							Name: "my-router",
						},
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Router: &infrav1.Router{
						ID: routerID,
					},
				},
			},
			expect: func(_ Gomega, m *mock.MockNetworkClientMockRecorder) {
				// Get by ID in status returns 404
				// Error
				m.GetRouter(routerID).Return(nil, gophercloud.ErrUnexpectedResponseCode{Actual: 404})
			},
			wantErr: true,
		},
		{
			name: "Unmanaged router not in status does not exist",
			openStackCluster: infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					Router: &infrav1.RouterParam{
						Filter: &infrav1.RouterFilter{
							Name: "my-router",
						},
					},
				},
			},
			expect: func(g Gomega, m *mock.MockNetworkClientMockRecorder) {
				// Lookup by name returns no results
				// Error
				m.ListRouter(gomock.Any()).DoAndReturn(func(opts routers.ListOpts) ([]routers.Router, error) {
					g.Expect(opts.Name).To(Equal("my-router"))
					return []routers.Router{}, nil
				})
			},
			wantErr: true,
		},
	}
	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			g := NewWithT(t)
			log := testr.New(t)

			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			s, err := NewService(scope.NewWithLogger(mockScopeFactory, log))
			g.Expect(err).NotTo(HaveOccurred())

			tt.expect(g, mockScopeFactory.NetworkClient.EXPECT())

			if err := s.DeleteRouter(&tt.openStackCluster, clusterResourceName); (err != nil) != tt.wantErr {
				t.Errorf("Service.DeleteRouter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_createRouter(t *testing.T) {
	const (
		clusterResourceName = "test-cluster"
		routerName          = "k8s-clusterapi-cluster-test-cluster"
		routerID            = "38052015-5cbc-4cb4-8e45-445d53260f60"
		externalNetworkID   = "6f2be7de-2f6b-4bdb-b6d3-8e1f1e6b7f01"
	)

	tests := []struct {
		name    string
		tags    []string
		expect  func(m *mock.MockNetworkClientMockRecorder)
		wantErr bool
	}{
		{
			name: "no tags requested, no extension lookup needed",
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.CreateRouter(routers.CreateOpts{
					Name:        routerName,
					Description: names.GetDescription(clusterResourceName),
					GatewayInfo: &routers.GatewayInfo{NetworkID: externalNetworkID},
				}).Return(&routers.Router{ID: routerID}, nil)
			},
		},
		{
			name: "tags requested, standard-attr-tag not supported",
			tags: []string{"cluster-tag"},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				// standard-attr-tag support must be checked before the router
				// is created, and ReplaceAllAttributesTags must not be called
				// since the extension is not advertised.
				m.ListExtensions().Return([]extensions.Extension{}, nil)

				m.CreateRouter(routers.CreateOpts{
					Name:        routerName,
					Description: names.GetDescription(clusterResourceName),
					GatewayInfo: &routers.GatewayInfo{NetworkID: externalNetworkID},
				}).Return(&routers.Router{ID: routerID}, nil)
			},
		},
		{
			name: "tags requested, standard-attr-tag supported",
			tags: []string{"cluster-tag"},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				standardAttrTagExt := extensions.Extension{}
				standardAttrTagExt.Alias = "standard-attr-tag"
				m.ListExtensions().Return([]extensions.Extension{standardAttrTagExt}, nil)

				m.CreateRouter(routers.CreateOpts{
					Name:        routerName,
					Description: names.GetDescription(clusterResourceName),
					GatewayInfo: &routers.GatewayInfo{NetworkID: externalNetworkID},
				}).Return(&routers.Router{ID: routerID}, nil)

				m.ReplaceAllAttributesTags("routers", routerID, attributestags.ReplaceAllOpts{
					Tags: []string{"cluster-tag"},
				}).Return([]string{"cluster-tag"}, nil)
			},
		},
		{
			name: "tags requested, extension discovery fails before router is created",
			tags: []string{"cluster-tag"},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				// ListExtensions fails before CreateRouter is ever called.
				m.ListExtensions().Return(nil, errors.New("boom"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			g := NewWithT(t)
			log := testr.New(t)

			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			s, err := NewService(scope.NewWithLogger(mockScopeFactory, log))
			g.Expect(err).NotTo(HaveOccurred())

			openStackCluster := &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					Tags: tt.tags,
				},
				Status: infrav1.OpenStackClusterStatus{
					ExternalNetwork: &infrav1.NetworkStatus{ID: externalNetworkID},
				},
			}

			tt.expect(mockScopeFactory.NetworkClient.EXPECT())

			router, err := s.createRouter(openStackCluster, clusterResourceName, routerName)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
				return
			}
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(router).To(Equal(&routers.Router{ID: routerID}))
		})
	}
}
