/*
Copyright 2022 The Kubernetes Authors.

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

package loadbalancer

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/gophercloud/gophercloud/openstack/compute/apiversions"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/listeners"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/monitors"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/pools"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/providers"
	. "github.com/onsi/gomega"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha6"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
)

func Test_ReconcileLoadBalancer(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	openStackCluster := &infrav1.OpenStackCluster{
		Spec: infrav1.OpenStackClusterSpec{
			DisableAPIServerFloatingIP: true,
		},
		Status: infrav1.OpenStackClusterStatus{
			ExternalNetwork: &infrav1.Network{
				ID: "aaaaaaaa-bbbb-cccc-dddd-111111111111",
			},
			Network: &infrav1.Network{
				APIServerLoadBalancer: &infrav1.LoadBalancer{
					Monitor: &infrav1.Monitor{
						Delay:      33,
						Timeout:    11,
						MaxRetries: 22,
					},
				},
				Subnet: &infrav1.Subnet{
					ID: "aaaaaaaa-bbbb-cccc-dddd-222222222222",
				},
			},
		},
	}
	type serviceFields struct {
		projectID          string
		networkingClient   *mock.MockNetworkClient
		loadbalancerClient *mock.MockLbClient
	}
	lbtests := []struct {
		name               string
		fields             serviceFields
		prepareServiceMock func(sf *serviceFields)
		expectNetwork      func(m *mock.MockNetworkClientMockRecorder)
		expectLoadBalancer func(m *mock.MockLbClientMockRecorder)
		wantError          error
	}{
		{
			name: "reconcile loadbalancer in non active state should wait for active state",
			prepareServiceMock: func(sf *serviceFields) {
				sf.networkingClient = mock.NewMockNetworkClient(mockCtrl)
				sf.loadbalancerClient = mock.NewMockLbClient(mockCtrl)
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				// add network api call results here
			},
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
				// return loadbalancer providers
				providers := []providers.Provider{
					{Name: "amphora", Description: "The Octavia Amphora driver."},
					{Name: "octavia", Description: "Deprecated alias of the Octavia Amphora driver."},
				}
				m.ListLoadBalancerProviders().Return(providers, nil)

				pendingLB := loadbalancers.LoadBalancer{
					ID:                 "aaaaaaaa-bbbb-cccc-dddd-333333333333",
					Name:               "k8s-clusterapi-cluster-AAAAA-kubeapi",
					ProvisioningStatus: "PENDING_CREATE",
				}
				activeLB := pendingLB
				activeLB.ProvisioningStatus = "ACTIVE"

				// return existing loadbalancer in non-active state
				lbList := []loadbalancers.LoadBalancer{pendingLB}
				m.ListLoadBalancers(loadbalancers.ListOpts{Name: pendingLB.Name}).Return(lbList, nil)

				// wait for active loadbalancer by returning active loadbalancer on second call
				m.GetLoadBalancer("aaaaaaaa-bbbb-cccc-dddd-333333333333").Return(&pendingLB, nil).Return(&activeLB, nil)

				// return octavia versions
				versions := []apiversions.APIVersion{
					{ID: "2.24"},
					{ID: "2.23"},
					{ID: "2.22"},
				}
				m.ListOctaviaVersions().Return(versions, nil)

				listenerList := []listeners.Listener{
					{
						ID:   "aaaaaaaa-bbbb-cccc-dddd-444444444444",
						Name: "k8s-clusterapi-cluster-AAAAA-kubeapi-0",
					},
				}
				m.ListListeners(listeners.ListOpts{Name: listenerList[0].Name}).Return(listenerList, nil)

				poolList := []pools.Pool{
					{
						ID:   "aaaaaaaa-bbbb-cccc-dddd-555555555555",
						Name: "k8s-clusterapi-cluster-AAAAA-kubeapi-0",
					},
				}
				m.ListPools(pools.ListOpts{Name: poolList[0].Name}).Return(poolList, nil)

				monitorList := []monitors.Monitor{
					{
						ID:   "aaaaaaaa-bbbb-cccc-dddd-666666666666",
						Name: "k8s-clusterapi-cluster-AAAAA-kubeapi-0",
					},
				}
				m.ListMonitors(monitors.ListOpts{Name: monitorList[0].Name}).Return(monitorList, nil)
			},
			wantError: nil,
		},
	}
	for _, tt := range lbtests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepareServiceMock(&tt.fields)
			networkingService := networking.NewTestService(tt.fields.projectID, tt.fields.networkingClient, logr.Discard())
			lbs := NewLoadBalancerTestService(tt.fields.projectID, tt.fields.loadbalancerClient, networkingService, logr.Discard())
			g := NewWithT(t)
			tt.expectNetwork(tt.fields.networkingClient.EXPECT())
			tt.expectLoadBalancer(tt.fields.loadbalancerClient.EXPECT())
			err := lbs.ReconcileLoadBalancer(openStackCluster, "AAAAA", 0)
			if tt.wantError != nil {
				g.Expect(err).To(MatchError(tt.wantError))
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
		})
	}
}
