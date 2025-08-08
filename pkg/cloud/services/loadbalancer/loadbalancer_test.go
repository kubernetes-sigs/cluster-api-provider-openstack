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
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/apiversions"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/listeners"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/monitors"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/pools"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/providers"
	. "github.com/onsi/gomega" //nolint:revive
	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

const apiHostname = "api.test-cluster.test"

func Test_ReconcileLoadBalancer(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Shortcut wait timeout
	backoffDurationPrev := backoff.Duration
	backoff.Duration = 0
	defer func() {
		backoff.Duration = backoffDurationPrev
	}()

	// Stub the call to net.LookupHost
	lookupHost = func(host string) (addrs *string, err error) {
		if net.ParseIP(host) != nil {
			return &host, nil
		} else if host == apiHostname {
			ips := []string{"192.168.100.10"}
			return &ips[0], nil
		}
		return nil, errors.New("Unknown Host " + host)
	}

	openStackCluster := &infrav1.OpenStackCluster{
		Spec: infrav1.OpenStackClusterSpec{
			APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
				Enabled: ptr.To(true),
			},
			DisableAPIServerFloatingIP: ptr.To(true),
			ControlPlaneEndpoint: &clusterv1beta1.APIEndpoint{
				Host: apiHostname,
				Port: 6443,
			},
		},
		Status: infrav1.OpenStackClusterStatus{
			ExternalNetwork: &infrav1.NetworkStatus{
				ID: "aaaaaaaa-bbbb-cccc-dddd-111111111111",
			},
			Network: &infrav1.NetworkStatusWithSubnets{
				Subnets: []infrav1.Subnet{
					{ID: "aaaaaaaa-bbbb-cccc-dddd-222222222222"},
				},
			},
		},
	}
	lbtests := []struct {
		name               string
		clusterSpec        *infrav1.OpenStackCluster
		expectNetwork      func(m *mock.MockNetworkClientMockRecorder)
		expectLoadBalancer func(m *mock.MockLbClientMockRecorder)
		wantError          error
	}{
		{
			name:        "reconcile loadbalancer in non active state should wait for active state",
			clusterSpec: openStackCluster,
			expectNetwork: func(*mock.MockNetworkClientMockRecorder) {
				// add network api call results here
			},
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
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

				// create a monitor with values that match defaults to prevent update
				monitorList := []monitors.Monitor{
					{
						ID:             "aaaaaaaa-bbbb-cccc-dddd-666666666666",
						Name:           "k8s-clusterapi-cluster-AAAAA-kubeapi-0",
						Delay:          10,
						Timeout:        5,
						MaxRetries:     5,
						MaxRetriesDown: 3,
					},
				}
				m.ListMonitors(monitors.ListOpts{Name: monitorList[0].Name}).Return(monitorList, nil)
			},
			wantError: nil,
		},
		{
			name:        "reconcile loadbalancer in non active state should timeout",
			clusterSpec: openStackCluster,
			expectNetwork: func(*mock.MockNetworkClientMockRecorder) {
				// add network api call results here
			},
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
				pendingLB := loadbalancers.LoadBalancer{
					ID:                 "aaaaaaaa-bbbb-cccc-dddd-333333333333",
					Name:               "k8s-clusterapi-cluster-AAAAA-kubeapi",
					ProvisioningStatus: "PENDING_CREATE",
				}

				// return existing loadbalancer in non-active state
				lbList := []loadbalancers.LoadBalancer{pendingLB}
				m.ListLoadBalancers(loadbalancers.ListOpts{Name: pendingLB.Name}).Return(lbList, nil)

				// wait for loadbalancer until it times out
				m.GetLoadBalancer("aaaaaaaa-bbbb-cccc-dddd-333333333333").Return(&pendingLB, nil).Return(&pendingLB, nil).AnyTimes()
			},
			wantError: fmt.Errorf("load balancer \"k8s-clusterapi-cluster-AAAAA-kubeapi\" with id aaaaaaaa-bbbb-cccc-dddd-333333333333 is not active after timeout: timed out waiting for the condition"),
		},
		{
			name: "should update monitor when values are different than defaults",
			clusterSpec: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
						Enabled: ptr.To(true),
						Monitor: &infrav1.APIServerLoadBalancerMonitor{
							Delay:          15,
							Timeout:        8,
							MaxRetries:     6,
							MaxRetriesDown: 4,
						},
					},
					DisableAPIServerFloatingIP: ptr.To(true),
					ControlPlaneEndpoint: &clusterv1beta1.APIEndpoint{
						Host: apiHostname,
						Port: 6443,
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					ExternalNetwork: &infrav1.NetworkStatus{
						ID: "aaaaaaaa-bbbb-cccc-dddd-111111111111",
					},
					Network: &infrav1.NetworkStatusWithSubnets{
						Subnets: []infrav1.Subnet{
							{ID: "aaaaaaaa-bbbb-cccc-dddd-222222222222"},
						},
					},
				},
			},
			expectNetwork: func(*mock.MockNetworkClientMockRecorder) {
				// add network api call results here
			},
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
				activeLB := loadbalancers.LoadBalancer{
					ID:                 "aaaaaaaa-bbbb-cccc-dddd-333333333333",
					Name:               "k8s-clusterapi-cluster-AAAAA-kubeapi",
					ProvisioningStatus: "ACTIVE",
				}

				// return existing loadbalancer in active state
				lbList := []loadbalancers.LoadBalancer{activeLB}
				m.ListLoadBalancers(loadbalancers.ListOpts{Name: activeLB.Name}).Return(lbList, nil)

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

				// existing monitor has default values that need updating
				existingMonitor := monitors.Monitor{
					ID:             "aaaaaaaa-bbbb-cccc-dddd-666666666666",
					Name:           "k8s-clusterapi-cluster-AAAAA-kubeapi-0",
					Delay:          10,
					Timeout:        5,
					MaxRetries:     5,
					MaxRetriesDown: 3,
				}
				monitorList := []monitors.Monitor{existingMonitor}
				m.ListMonitors(monitors.ListOpts{Name: monitorList[0].Name}).Return(monitorList, nil)

				// Expect update call with the new values
				updateOpts := monitors.UpdateOpts{
					Delay:          15,
					Timeout:        8,
					MaxRetries:     6,
					MaxRetriesDown: 4,
				}

				updatedMonitor := existingMonitor
				updatedMonitor.Delay = 15
				updatedMonitor.Timeout = 8
				updatedMonitor.MaxRetries = 6
				updatedMonitor.MaxRetriesDown = 4

				m.UpdateMonitor(existingMonitor.ID, updateOpts).Return(&updatedMonitor, nil)

				// Expect wait for loadbalancer to be active after monitor update
				m.GetLoadBalancer(activeLB.ID).Return(&activeLB, nil)
			},
			wantError: nil,
		},
		{
			name: "should report error when monitor update fails",
			clusterSpec: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
						Enabled: ptr.To(true),
						Monitor: &infrav1.APIServerLoadBalancerMonitor{
							Delay:          15,
							Timeout:        8,
							MaxRetries:     6,
							MaxRetriesDown: 4,
						},
					},
					DisableAPIServerFloatingIP: ptr.To(true),
					ControlPlaneEndpoint: &clusterv1beta1.APIEndpoint{
						Host: apiHostname,
						Port: 6443,
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					ExternalNetwork: &infrav1.NetworkStatus{
						ID: "aaaaaaaa-bbbb-cccc-dddd-111111111111",
					},
					Network: &infrav1.NetworkStatusWithSubnets{
						Subnets: []infrav1.Subnet{
							{ID: "aaaaaaaa-bbbb-cccc-dddd-222222222222"},
						},
					},
				},
			},
			expectNetwork: func(*mock.MockNetworkClientMockRecorder) {
				// add network api call results here
			},
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
				activeLB := loadbalancers.LoadBalancer{
					ID:                 "aaaaaaaa-bbbb-cccc-dddd-333333333333",
					Name:               "k8s-clusterapi-cluster-AAAAA-kubeapi",
					ProvisioningStatus: "ACTIVE",
				}

				// return existing loadbalancer in active state
				lbList := []loadbalancers.LoadBalancer{activeLB}
				m.ListLoadBalancers(loadbalancers.ListOpts{Name: activeLB.Name}).Return(lbList, nil)

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

				// existing monitor has default values that need updating
				existingMonitor := monitors.Monitor{
					ID:             "aaaaaaaa-bbbb-cccc-dddd-666666666666",
					Name:           "k8s-clusterapi-cluster-AAAAA-kubeapi-0",
					Delay:          10,
					Timeout:        5,
					MaxRetries:     5,
					MaxRetriesDown: 3,
				}
				monitorList := []monitors.Monitor{existingMonitor}
				m.ListMonitors(monitors.ListOpts{Name: monitorList[0].Name}).Return(monitorList, nil)

				// Expect update call with the new values but return an error
				updateOpts := monitors.UpdateOpts{
					Delay:          15,
					Timeout:        8,
					MaxRetries:     6,
					MaxRetriesDown: 4,
				}

				updateError := fmt.Errorf("failed to update monitor")
				m.UpdateMonitor(existingMonitor.ID, updateOpts).Return(nil, updateError)
			},
			wantError: fmt.Errorf("failed to update monitor"),
		},
		{
			name: "should create monitor when it doesn't exist",
			clusterSpec: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
						Enabled: ptr.To(true),
						Monitor: &infrav1.APIServerLoadBalancerMonitor{
							Delay:          15,
							Timeout:        8,
							MaxRetries:     6,
							MaxRetriesDown: 4,
						},
					},
					DisableAPIServerFloatingIP: ptr.To(true),
					ControlPlaneEndpoint: &clusterv1beta1.APIEndpoint{
						Host: apiHostname,
						Port: 6443,
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					ExternalNetwork: &infrav1.NetworkStatus{
						ID: "aaaaaaaa-bbbb-cccc-dddd-111111111111",
					},
					Network: &infrav1.NetworkStatusWithSubnets{
						Subnets: []infrav1.Subnet{
							{ID: "aaaaaaaa-bbbb-cccc-dddd-222222222222"},
						},
					},
				},
			},
			expectNetwork: func(*mock.MockNetworkClientMockRecorder) {
				// add network api call results here
			},
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
				activeLB := loadbalancers.LoadBalancer{
					ID:                 "aaaaaaaa-bbbb-cccc-dddd-333333333333",
					Name:               "k8s-clusterapi-cluster-AAAAA-kubeapi",
					ProvisioningStatus: "ACTIVE",
				}

				// return existing loadbalancer in active state
				lbList := []loadbalancers.LoadBalancer{activeLB}
				m.ListLoadBalancers(loadbalancers.ListOpts{Name: activeLB.Name}).Return(lbList, nil)

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

				// No monitor exists yet
				var emptyMonitorList []monitors.Monitor
				m.ListMonitors(monitors.ListOpts{Name: "k8s-clusterapi-cluster-AAAAA-kubeapi-0"}).Return(emptyMonitorList, nil)

				// Expect create call with custom values
				createOpts := monitors.CreateOpts{
					Name:           "k8s-clusterapi-cluster-AAAAA-kubeapi-0",
					PoolID:         "aaaaaaaa-bbbb-cccc-dddd-555555555555",
					Type:           "TCP",
					Delay:          15,
					Timeout:        8,
					MaxRetries:     6,
					MaxRetriesDown: 4,
				}

				createdMonitor := monitors.Monitor{
					ID:             "aaaaaaaa-bbbb-cccc-dddd-666666666666",
					Name:           "k8s-clusterapi-cluster-AAAAA-kubeapi-0",
					Delay:          15,
					Timeout:        8,
					MaxRetries:     6,
					MaxRetriesDown: 4,
				}

				m.CreateMonitor(createOpts).Return(&createdMonitor, nil)

				// Expect wait for loadbalancer to be active after monitor creation
				m.GetLoadBalancer(activeLB.ID).Return(&activeLB, nil)
			},
			wantError: nil,
		},
	}
	for _, tt := range lbtests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			log := testr.New(t)

			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			lbs, err := NewService(scope.NewWithLogger(mockScopeFactory, log))
			g.Expect(err).NotTo(HaveOccurred())

			tt.expectNetwork(mockScopeFactory.NetworkClient.EXPECT())
			tt.expectLoadBalancer(mockScopeFactory.LbClient.EXPECT())
			_, err = lbs.ReconcileLoadBalancer(tt.clusterSpec, "AAAAA", 0)
			if tt.wantError != nil {
				g.Expect(err).To(MatchError(tt.wantError))
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
		})
	}
}

func Test_getAPIServerVIPAddress(t *testing.T) {
	// Stub the call to net.LookupHost
	lookupHost = func(host string) (addrs *string, err error) {
		if net.ParseIP(host) != nil {
			return &host, nil
		} else if host == apiHostname {
			ips := []string{"192.168.100.10"}
			return &ips[0], nil
		}
		return nil, errors.New("Unknown Host " + host)
	}
	tests := []struct {
		name             string
		openStackCluster *infrav1.OpenStackCluster
		want             *string
		wantError        bool
	}{
		{
			name:             "empty cluster returns empty VIP",
			openStackCluster: &infrav1.OpenStackCluster{},
			want:             nil,
			wantError:        false,
		},
		{
			name: "API server VIP is InternalIP",
			openStackCluster: &infrav1.OpenStackCluster{
				Status: infrav1.OpenStackClusterStatus{
					APIServerLoadBalancer: &infrav1.LoadBalancer{
						InternalIP: "1.2.3.4",
					},
				},
			},
			want:      ptr.To("1.2.3.4"),
			wantError: false,
		},
		{
			name: "API server VIP is API Server Fixed IP",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerFixedIP: ptr.To("1.2.3.4"),
				},
			},
			want:      ptr.To("1.2.3.4"),
			wantError: false,
		},
		{
			name: "API server VIP with valid control plane endpoint",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					DisableAPIServerFloatingIP: ptr.To(true),
					ControlPlaneEndpoint: &clusterv1beta1.APIEndpoint{
						Host: apiHostname,
						Port: 6443,
					},
				},
			},
			want:      ptr.To("192.168.100.10"),
			wantError: false,
		},
		{
			name: "API server VIP with invalid control plane endpoint",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					DisableAPIServerFloatingIP: ptr.To(true),
					ControlPlaneEndpoint: &clusterv1beta1.APIEndpoint{
						Host: "invalid-api.test-cluster.test",
						Port: 6443,
					},
				},
			},
			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			got, err := getAPIServerVIPAddress(tt.openStackCluster)
			if tt.wantError {
				g.Expect(err).To(HaveOccurred())
				g.Expect(got).To(BeNil())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(got).To(Equal(tt.want))
			}
		})
	}
}

func Test_getAPIServerFloatingIP(t *testing.T) {
	// Stub the call to net.LookupHost
	lookupHost = func(host string) (addrs *string, err error) {
		if net.ParseIP(host) != nil {
			return &host, nil
		} else if host == apiHostname {
			ips := []string{"192.168.100.10"}
			return &ips[0], nil
		}
		return nil, errors.New("Unknown Host " + host)
	}
	tests := []struct {
		name             string
		openStackCluster *infrav1.OpenStackCluster
		want             *string
		wantError        bool
	}{
		{
			name:             "empty cluster returns empty FIP",
			openStackCluster: &infrav1.OpenStackCluster{},
			want:             nil,
			wantError:        false,
		},
		{
			name: "API server FIP is API Server LB IP",
			openStackCluster: &infrav1.OpenStackCluster{
				Status: infrav1.OpenStackClusterStatus{
					APIServerLoadBalancer: &infrav1.LoadBalancer{
						IP: "1.2.3.4",
					},
				},
			},
			want:      ptr.To("1.2.3.4"),
			wantError: false,
		},
		{
			name: "API server FIP is API Server Floating IP",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerFloatingIP: ptr.To("1.2.3.4"),
				},
			},
			want:      ptr.To("1.2.3.4"),
			wantError: false,
		},
		{
			name: "API server FIP with valid control plane endpoint",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ControlPlaneEndpoint: &clusterv1beta1.APIEndpoint{
						Host: apiHostname,
						Port: 6443,
					},
				},
			},
			want:      ptr.To("192.168.100.10"),
			wantError: false,
		},
		{
			name: "API server FIP with invalid control plane endpoint",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ControlPlaneEndpoint: &clusterv1beta1.APIEndpoint{
						Host: "invalid-api.test-cluster.test",
						Port: 6443,
					},
				},
			},
			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			got, err := getAPIServerFloatingIP(tt.openStackCluster)
			if tt.wantError {
				g.Expect(err).To(HaveOccurred())
				g.Expect(got).To(BeNil())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(got).To(Equal(tt.want))
			}
		})
	}
}

func Test_getCanonicalAllowedCIDRs(t *testing.T) {
	tests := []struct {
		name             string
		openStackCluster *infrav1.OpenStackCluster
		want             []string
	}{
		{
			name:             "allowed CIDRs are empty",
			openStackCluster: &infrav1.OpenStackCluster{},
			want:             []string{},
		},
		{
			name: "allowed CIDRs are set",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
						AllowedCIDRs: []string{"1.2.3.4/32"},
					},
				},
			},
			want: []string{"1.2.3.4/32"},
		},
		{
			name: "allowed CIDRs are set with bastion",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
						AllowedCIDRs: []string{"1.2.3.4/32"},
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Bastion: &infrav1.BastionStatus{
						FloatingIP: "1.2.3.5",
						IP:         "192.168.0.1",
					},
				},
			},
			want: []string{"1.2.3.4/32", "1.2.3.5/32", "192.168.0.1/32"},
		},
		{
			name: "allowed CIDRs are set with network status",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
						AllowedCIDRs: []string{"1.2.3.4/32"},
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						Subnets: []infrav1.Subnet{
							{
								CIDR: "192.168.0.0/24",
							},
						},
					},
				},
			},
			want: []string{"1.2.3.4/32", "192.168.0.0/24"},
		},
		{
			name: "allowed CIDRs are set with network status and router IP",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
						AllowedCIDRs: []string{"1.2.3.4/32"},
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						Subnets: []infrav1.Subnet{
							{
								CIDR: "192.168.0.0/24",
							},
						},
					},
					Router: &infrav1.Router{
						IPs: []string{"1.2.3.5"},
					},
				},
			},
			want: []string{"1.2.3.4/32", "1.2.3.5/32", "192.168.0.0/24"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			got := getCanonicalAllowedCIDRs(tt.openStackCluster)
			g.Expect(got).To(Equal(tt.want))
		})
	}
}

func Test_getOrCreateAPILoadBalancer(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	octaviaProviders := []providers.Provider{
		{
			Name: "ovn",
		},
	}
	octaviaFlavors := []flavors.Flavor{
		{
			ID:      "aaaaaaaa-bbbb-cccc-dddd-111111111111",
			Name:    "flavorName",
			Enabled: true,
		},
	}
	lbtests := []struct {
		name               string
		openStackCluster   *infrav1.OpenStackCluster
		expectLoadBalancer func(m *mock.MockLbClientMockRecorder)
		want               *loadbalancers.LoadBalancer
		wantError          error
	}{
		{
			name:             "nothing exists",
			openStackCluster: &infrav1.OpenStackCluster{},
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
				m.ListLoadBalancers(gomock.Any()).Return([]loadbalancers.LoadBalancer{}, nil)
			},
			want:      &loadbalancers.LoadBalancer{},
			wantError: fmt.Errorf("network is not yet available in OpenStackCluster.Status"),
		},
		{
			name:             "loadbalancer already exists",
			openStackCluster: &infrav1.OpenStackCluster{},
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
				m.ListLoadBalancers(gomock.Any()).Return([]loadbalancers.LoadBalancer{{ID: "AAAAA"}}, nil)
			},
			want: &loadbalancers.LoadBalancer{
				ID: "AAAAA",
			},
		},
		{
			name: "loadbalancer created",
			openStackCluster: &infrav1.OpenStackCluster{
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						Subnets: []infrav1.Subnet{
							{ID: "aaaaaaaa-bbbb-cccc-dddd-222222222222"},
							{ID: "aaaaaaaa-bbbb-cccc-dddd-333333333333"},
						},
					},
					APIServerLoadBalancer: &infrav1.LoadBalancer{
						LoadBalancerNetwork: nil,
					},
				},
			},
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
				m.ListLoadBalancers(gomock.Any()).Return([]loadbalancers.LoadBalancer{}, nil)
				m.ListLoadBalancerProviders().Return(octaviaProviders, nil)
				m.CreateLoadBalancer(gomock.Any()).Return(&loadbalancers.LoadBalancer{
					ID:          "AAAAA",
					VipSubnetID: "aaaaaaaa-bbbb-cccc-dddd-222222222222",
				}, nil)
			},
			want: &loadbalancers.LoadBalancer{
				ID:          "AAAAA",
				VipSubnetID: "aaaaaaaa-bbbb-cccc-dddd-222222222222",
			},
		},
		{
			name: "loadbalancer on a specific network created",
			openStackCluster: &infrav1.OpenStackCluster{
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						Subnets: []infrav1.Subnet{
							{ID: "aaaaaaaa-bbbb-cccc-dddd-222222222222"},
						},
					},
					APIServerLoadBalancer: &infrav1.LoadBalancer{
						LoadBalancerNetwork: &infrav1.NetworkStatusWithSubnets{
							NetworkStatus: infrav1.NetworkStatus{
								Name: "VIPNET",
								ID:   "VIPNET",
							},
							Subnets: []infrav1.Subnet{
								{
									Name: "vip-subnet",
									CIDR: "10.0.0.0/24",
									ID:   "VIPSUBNET",
								},
							},
						},
					},
				},
			},
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
				m.ListLoadBalancers(gomock.Any()).Return([]loadbalancers.LoadBalancer{}, nil)
				m.ListLoadBalancerProviders().Return(octaviaProviders, nil)
				m.CreateLoadBalancer(gomock.Any()).Return(&loadbalancers.LoadBalancer{
					ID:           "AAAAA",
					VipSubnetID:  "VIPSUBNET",
					VipNetworkID: "VIPNET",
				}, nil)
			},
			want: &loadbalancers.LoadBalancer{
				ID:           "AAAAA",
				VipSubnetID:  "VIPSUBNET",
				VipNetworkID: "VIPNET",
			},
		},
		{
			name: "loadbalancer with specified flavor created",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
						Flavor: ptr.To("flavorName"),
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						Subnets: []infrav1.Subnet{
							{ID: "aaaaaaaa-bbbb-cccc-dddd-222222222222"},
							{ID: "aaaaaaaa-bbbb-cccc-dddd-333333333333"},
						},
					},
					APIServerLoadBalancer: &infrav1.LoadBalancer{
						LoadBalancerNetwork: nil,
					},
				},
			},
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
				m.ListLoadBalancers(gomock.Any()).Return([]loadbalancers.LoadBalancer{}, nil)
				m.ListLoadBalancerProviders().Return(octaviaProviders, nil)
				m.ListLoadBalancerFlavors().Return(octaviaFlavors, nil)
				m.CreateLoadBalancer(gomock.Any()).Return(&loadbalancers.LoadBalancer{
					ID:          "AAAAA",
					VipSubnetID: "aaaaaaaa-bbbb-cccc-dddd-222222222222",
				}, nil)
			},
			want: &loadbalancers.LoadBalancer{
				ID:          "AAAAA",
				VipSubnetID: "aaaaaaaa-bbbb-cccc-dddd-222222222222",
			},
		},
	}
	for _, tt := range lbtests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			log := testr.New(t)

			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			lbs, err := NewService(scope.NewWithLogger(mockScopeFactory, log))
			g.Expect(err).NotTo(HaveOccurred())

			tt.expectLoadBalancer(mockScopeFactory.LbClient.EXPECT())
			lb, err := lbs.getOrCreateAPILoadBalancer(tt.openStackCluster, "AAAAA")
			if tt.wantError != nil {
				g.Expect(err).To(MatchError(tt.wantError))
			} else {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(lb).To(Equal(tt.want))
			}
		})
	}
}
