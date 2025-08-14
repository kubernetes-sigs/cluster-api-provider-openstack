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
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/listeners"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/monitors"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/pools"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/providers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	. "github.com/onsi/gomega" //nolint:revive
	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

const (
	apiHostname = "api.test-cluster.test"
	testAZ1     = "az1"
)

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
			_, err = lbs.ReconcileLoadBalancer(tt.clusterSpec, "AAAAA", azSubnet{az: ptr.To[string](""), subnet: infrav1.Subnet{ID: "aaaaaaaa-bbbb-cccc-dddd-222222222222"}}, 0)
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
		{
			name: "allowed CIDRs with multi-AZ network status",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
						AllowedCIDRs:      []string{"1.2.3.4/32"},
						AvailabilityZones: []string{"az1", "az2", "az3"},
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						Subnets: []infrav1.Subnet{
							{
								CIDR: "192.168.0.0/24",
							},
							{
								CIDR: "192.168.1.0/24",
							},
							{
								CIDR: "192.168.2.0/24",
							},
						},
					},
				},
			},
			want: []string{"1.2.3.4/32", "192.168.0.0/24", "192.168.1.0/24", "192.168.2.0/24"},
		},
		{
			name: "allowed CIDRs with multi-AZ including bastion and router IPs",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
						AllowedCIDRs:      []string{"1.2.3.4/32", "10.0.0.0/16"},
						AvailabilityZones: []string{"az1", "az2"},
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						Subnets: []infrav1.Subnet{
							{
								CIDR: "192.168.0.0/24",
							},
							{
								CIDR: "192.168.1.0/24",
							},
						},
					},
					Bastion: &infrav1.BastionStatus{
						FloatingIP: "1.2.3.10",
						IP:         "192.168.0.10",
					},
					Router: &infrav1.Router{
						IPs: []string{"1.2.3.11", "1.2.3.12"},
					},
				},
			},
			want: []string{"1.2.3.10/32", "1.2.3.11/32", "1.2.3.12/32", "1.2.3.4/32", "10.0.0.0/16", "192.168.0.0/24", "192.168.0.10/32", "192.168.1.0/24"},
		},
		{
			name: "single-AZ legacy format (no AvailabilityZones specified)",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
						AllowedCIDRs: []string{"1.2.3.4/32"},
						// No AvailabilityZones specified - legacy single-AZ format
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
			name: "legacy single-AZ with old AvailabilityZone field (for migration testing)",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
						AllowedCIDRs:     []string{"1.2.3.4/32"},
						AvailabilityZone: ptr.To("legacy-az"),
						// AvailabilityZones not specified - this should trigger migration to new format
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
					Bastion: &infrav1.BastionStatus{
						FloatingIP: "1.2.3.5",
						IP:         "192.168.0.1",
					},
				},
			},
			want: []string{"1.2.3.4/32", "1.2.3.5/32", "192.168.0.0/24", "192.168.0.1/32"},
		},
		{
			name: "complex multi-AZ with mixed subnets and overlapping IPs",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
						AllowedCIDRs:      []string{"1.2.3.4/32", "10.0.0.0/8"},
						AvailabilityZones: []string{"az1", "az2", "az3"},
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						Subnets: []infrav1.Subnet{
							{
								CIDR: "192.168.0.0/24",
							},
							{
								CIDR: "192.168.1.0/24",
							},
							{
								CIDR: "10.0.1.0/24", // This overlaps with allowed CIDR 10.0.0.0/8
							},
						},
					},
					Bastion: &infrav1.BastionStatus{
						FloatingIP: "1.2.3.4", // This duplicates an allowed CIDR (should be deduplicated)
						IP:         "192.168.0.100",
					},
					Router: &infrav1.Router{
						IPs: []string{"1.2.3.6", "1.2.3.6"}, // Duplicate IPs (should be deduplicated)
					},
				},
			},
			want: []string{"1.2.3.4/32", "1.2.3.6/32", "10.0.0.0/8", "10.0.1.0/24", "192.168.0.0/24", "192.168.0.100/32", "192.168.1.0/24"},
		},
		{
			name: "empty multi-AZ configuration",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
						AvailabilityZones: []string{"az1", "az2", "az3"},
						// No AllowedCIDRs specified - function only processes additional IPs when AllowedCIDRs is set
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						Subnets: []infrav1.Subnet{
							{
								CIDR: "192.168.0.0/24",
							},
							{
								CIDR: "192.168.1.0/24",
							},
							{
								CIDR: "192.168.2.0/24",
							},
						},
					},
				},
			},
			want: []string{}, // Empty because AllowedCIDRs is not specified
		},
		{
			name: "multi-AZ migration scenario - transitioning from single to multi-AZ",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
						AllowedCIDRs:      []string{"1.2.3.4/32"},
						AvailabilityZones: []string{"default", "az2"}, // "default" represents migrated single-AZ
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						Subnets: []infrav1.Subnet{
							{
								CIDR: "192.168.0.0/24", // Original single-AZ subnet
							},
							{
								CIDR: "192.168.1.0/24", // New AZ subnet
							},
						},
					},
					Bastion: &infrav1.BastionStatus{
						FloatingIP: "1.2.3.5",
						IP:         "192.168.0.1",
					},
					Router: &infrav1.Router{
						IPs: []string{"1.2.3.6"},
					},
				},
			},
			want: []string{"1.2.3.4/32", "1.2.3.5/32", "1.2.3.6/32", "192.168.0.0/24", "192.168.0.1/32", "192.168.1.0/24"},
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

func Test_ReconcileLoadBalancers(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	octaviaProviders := []providers.Provider{
		{
			Name: "ovn",
		},
	}

	baseCluster := &infrav1.OpenStackCluster{
		Spec: infrav1.OpenStackClusterSpec{
			APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
				Enabled: ptr.To(true),
			},
		},
		Status: infrav1.OpenStackClusterStatus{
			Network: &infrav1.NetworkStatusWithSubnets{
				NetworkStatus: infrav1.NetworkStatus{
					ID:   "cluster-network-id",
					Name: "cluster-network",
				},
				Subnets: []infrav1.Subnet{
					{
						ID:   "cluster-subnet-1",
						CIDR: "10.0.0.0/24",
					},
				},
			},
			ExternalNetwork: &infrav1.NetworkStatus{
				ID:   "external-network-id",
				Name: "external-network",
			},
			APIServerLoadBalancer: &infrav1.LoadBalancer{
				LoadBalancerNetwork: &infrav1.NetworkStatusWithSubnets{
					NetworkStatus: infrav1.NetworkStatus{
						ID:   "network-id",
						Name: "test-network",
					},
					Subnets: []infrav1.Subnet{
						{
							ID:   "subnet-1",
							CIDR: "192.168.1.0/24",
						},
						{
							ID:   "subnet-2",
							CIDR: "192.168.2.0/24",
						},
						{
							ID:   "subnet-3",
							CIDR: "192.168.3.0/24",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name               string
		openStackCluster   *infrav1.OpenStackCluster
		clusterResourceName string
		apiServerPort      int
		expectLoadBalancer func(m *mock.MockLbClientMockRecorder)
		want               bool
		wantError          error
	}{
		{
			name: "single-AZ default scenario (no AZ specified)",
			openStackCluster: func() *infrav1.OpenStackCluster {
				cluster := baseCluster.DeepCopy()
				// No AvailabilityZones specified - should create default AZ
				return cluster
			}(),
			clusterResourceName: "test-cluster",
			apiServerPort:      6443,
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
				// Expect migration calls first
				m.ListLoadBalancers(gomock.Any()).Return([]loadbalancers.LoadBalancer{}, nil).AnyTimes()
				m.ListListeners(gomock.Any()).Return([]listeners.Listener{}, nil).AnyTimes()
				m.ListPools(gomock.Any()).Return([]pools.Pool{}, nil).AnyTimes()
				m.ListMonitors(gomock.Any()).Return([]monitors.Monitor{}, nil).AnyTimes()
				
				// Expect load balancer creation for default AZ
				m.ListLoadBalancerProviders().Return(octaviaProviders, nil)
				m.CreateLoadBalancer(gomock.Any()).Return(&loadbalancers.LoadBalancer{
					ID:                   "lb-default",
					Name:                 "k8s-clusterapi-cluster-test-cluster-default-kubeapi",
					ProvisioningStatus:   "ACTIVE",
					VipSubnetID:         "subnet-1",
				}, nil)
				
				// Expect waiting for load balancer to be active
				m.GetLoadBalancer("lb-default").Return(&loadbalancers.LoadBalancer{
					ID:                   "lb-default",
					Name:                 "k8s-clusterapi-cluster-test-cluster-default-kubeapi",
					ProvisioningStatus:   "ACTIVE",
					VipSubnetID:         "subnet-1",
				}, nil).AnyTimes()
				
				// Expect Octavia version check for allowed CIDRs support
				m.ListOctaviaVersions().Return([]apiversions.APIVersion{
					{ID: "2.24"},
					{ID: "2.23"},
					{ID: "2.22"},
				}, nil)
				
				// Expect listener creation
				m.CreateListener(gomock.Any()).Return(&listeners.Listener{
					ID:              "listener-default",
					Name:            "k8s-clusterapi-cluster-test-cluster-default-kubeapi-6443",
					ProvisioningStatus: "ACTIVE",
				}, nil)
				
				// Expect waiting for listener to be ready
				m.GetListener("listener-default").Return(&listeners.Listener{
					ID:              "listener-default",
					Name:            "k8s-clusterapi-cluster-test-cluster-default-kubeapi-6443",
					ProvisioningStatus: "ACTIVE",
				}, nil).AnyTimes()
				
				// Expect pool creation
				m.CreatePool(gomock.Any()).Return(&pools.Pool{
					ID:              "pool-default",
					Name:            "k8s-clusterapi-cluster-test-cluster-default-kubeapi-6443",
					ProvisioningStatus: "ACTIVE",
				}, nil)
				
				// Expect monitor creation
				m.CreateMonitor(gomock.Any()).Return(&monitors.Monitor{
					ID:              "monitor-default",
					Name:            "k8s-clusterapi-cluster-test-cluster-default-kubeapi-6443",
					ProvisioningStatus: "ACTIVE",
				}, nil)
			},
			want:      false,
			wantError: nil,
		},
		{
			name: "multi-AZ scenario with 2 availability zones",
			openStackCluster: func() *infrav1.OpenStackCluster {
				cluster := baseCluster.DeepCopy()
				cluster.Spec.APIServerLoadBalancer.AvailabilityZones = []string{"az1", "az2"}
				return cluster
			}(),
			clusterResourceName: "test-cluster",
			apiServerPort:      6443,
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
				// Expect migration calls first
				m.ListLoadBalancers(gomock.Any()).Return([]loadbalancers.LoadBalancer{}, nil).AnyTimes()
				m.ListListeners(gomock.Any()).Return([]listeners.Listener{}, nil).AnyTimes()
				m.ListPools(gomock.Any()).Return([]pools.Pool{}, nil).AnyTimes()
				m.ListMonitors(gomock.Any()).Return([]monitors.Monitor{}, nil).AnyTimes()
				
				// Expect load balancer creation for az1
				m.ListLoadBalancerProviders().Return(octaviaProviders, nil)
				m.CreateLoadBalancer(gomock.Any()).Return(&loadbalancers.LoadBalancer{
					ID:                   "lb-az1",
					Name:                 "k8s-clusterapi-cluster-test-cluster-az1-kubeapi",
					ProvisioningStatus:   "ACTIVE",
					VipSubnetID:         "subnet-1",
				}, nil)
				
				// Expect waiting for load balancer to be active
				m.GetLoadBalancer("lb-az1").Return(&loadbalancers.LoadBalancer{
					ID:                   "lb-az1",
					Name:                 "k8s-clusterapi-cluster-test-cluster-az1-kubeapi",
					ProvisioningStatus:   "ACTIVE",
					VipSubnetID:         "subnet-1",
				}, nil).AnyTimes()
				
				// Expect Octavia version check for allowed CIDRs support
				m.ListOctaviaVersions().Return([]apiversions.APIVersion{
					{ID: "2.24"},
					{ID: "2.23"},
					{ID: "2.22"},
				}, nil).AnyTimes()
				
				// Expect listener, pool, monitor creation for az1
				m.CreateListener(gomock.Any()).Return(&listeners.Listener{
					ID:              "listener-az1",
					Name:            "k8s-clusterapi-cluster-test-cluster-az1-kubeapi-6443",
					ProvisioningStatus: "ACTIVE",
				}, nil)
				
				// Expect waiting for listener to be ready
				m.GetListener("listener-az1").Return(&listeners.Listener{
					ID:              "listener-az1",
					Name:            "k8s-clusterapi-cluster-test-cluster-az1-kubeapi-6443",
					ProvisioningStatus: "ACTIVE",
				}, nil).AnyTimes()
				
				m.CreatePool(gomock.Any()).Return(&pools.Pool{
					ID:              "pool-az1",
					Name:            "k8s-clusterapi-cluster-test-cluster-az1-kubeapi-6443",
					ProvisioningStatus: "ACTIVE",
				}, nil)
				m.CreateMonitor(gomock.Any()).Return(&monitors.Monitor{
					ID:              "monitor-az1",
					Name:            "k8s-clusterapi-cluster-test-cluster-az1-kubeapi-6443",
					ProvisioningStatus: "ACTIVE",
				}, nil)
				
				// Expect load balancer creation for az2  
				m.ListLoadBalancerProviders().Return(octaviaProviders, nil)
				m.CreateLoadBalancer(gomock.Any()).Return(&loadbalancers.LoadBalancer{
					ID:                   "lb-az2",
					Name:                 "k8s-clusterapi-cluster-test-cluster-az2-kubeapi",
					ProvisioningStatus:   "ACTIVE",
					VipSubnetID:         "subnet-2",
				}, nil)
				
				// Expect waiting for load balancer to be active
				m.GetLoadBalancer("lb-az2").Return(&loadbalancers.LoadBalancer{
					ID:                   "lb-az2",
					Name:                 "k8s-clusterapi-cluster-test-cluster-az2-kubeapi",
					ProvisioningStatus:   "ACTIVE",
					VipSubnetID:         "subnet-2",
				}, nil).AnyTimes()
				
				// Expect listener, pool, monitor creation for az2
				m.CreateListener(gomock.Any()).Return(&listeners.Listener{
					ID:              "listener-az2",
					Name:            "k8s-clusterapi-cluster-test-cluster-az2-kubeapi-6443",
					ProvisioningStatus: "ACTIVE",
				}, nil)
				
				// Expect waiting for listener to be ready
				m.GetListener("listener-az2").Return(&listeners.Listener{
					ID:              "listener-az2",
					Name:            "k8s-clusterapi-cluster-test-cluster-az2-kubeapi-6443",
					ProvisioningStatus: "ACTIVE",
				}, nil).AnyTimes()
				
				m.CreatePool(gomock.Any()).Return(&pools.Pool{
					ID:              "pool-az2",
					Name:            "k8s-clusterapi-cluster-test-cluster-az2-kubeapi-6443",
					ProvisioningStatus: "ACTIVE",
				}, nil)
				m.CreateMonitor(gomock.Any()).Return(&monitors.Monitor{
					ID:              "monitor-az2",
					Name:            "k8s-clusterapi-cluster-test-cluster-az2-kubeapi-6443",
					ProvisioningStatus: "ACTIVE",
				}, nil)
			},
			want:      false,
			wantError: nil,
		},
		{
			name: "multi-AZ scenario with 3 availability zones and additional ports",
			openStackCluster: func() *infrav1.OpenStackCluster {
				cluster := baseCluster.DeepCopy()
				cluster.Spec.APIServerLoadBalancer.AvailabilityZones = []string{"az1", "az2", "az3"}
				cluster.Spec.APIServerLoadBalancer.AdditionalPorts = []int{8080, 9090}
				return cluster
			}(),
			clusterResourceName: "test-cluster",
			apiServerPort:      6443,
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
				// Expect migration calls first
				m.ListLoadBalancers(gomock.Any()).Return([]loadbalancers.LoadBalancer{}, nil).AnyTimes()
				m.ListListeners(gomock.Any()).Return([]listeners.Listener{}, nil).AnyTimes()
				m.ListPools(gomock.Any()).Return([]pools.Pool{}, nil).AnyTimes()
				m.ListMonitors(gomock.Any()).Return([]monitors.Monitor{}, nil).AnyTimes()
				
				// Expect Octavia version check for allowed CIDRs support
				m.ListOctaviaVersions().Return([]apiversions.APIVersion{
					{ID: "2.24"},
					{ID: "2.23"},
					{ID: "2.22"},
				}, nil).AnyTimes()
				
				// For each AZ, expect load balancer creation
				for i, az := range []string{"az1", "az2", "az3"} {
					subnetID := fmt.Sprintf("subnet-%d", i+1)
					lbName := fmt.Sprintf("k8s-clusterapi-cluster-test-cluster-%s-kubeapi", az)
					lbID := fmt.Sprintf("lb-%s", az)
					
					m.ListLoadBalancerProviders().Return(octaviaProviders, nil)
					m.CreateLoadBalancer(gomock.Any()).Return(&loadbalancers.LoadBalancer{
						ID:                   lbID,
						Name:                 lbName,
						ProvisioningStatus:   "ACTIVE",
						VipSubnetID:         subnetID,
					}, nil)
					
					// Expect waiting for load balancer to be active
					m.GetLoadBalancer(lbID).Return(&loadbalancers.LoadBalancer{
						ID:                   lbID,
						Name:                 lbName,
						ProvisioningStatus:   "ACTIVE",
						VipSubnetID:         subnetID,
					}, nil).AnyTimes()
					
					// For each port (6443, 8080, 9090), expect listener/pool/monitor creation
					for _, port := range []int{6443, 8080, 9090} {
						portObjectName := fmt.Sprintf("%s-%d", lbName, port)
						listenerID := fmt.Sprintf("listener-%s-%d", az, port)
						m.CreateListener(gomock.Any()).Return(&listeners.Listener{
							ID:              listenerID,
							Name:            portObjectName,
							ProvisioningStatus: "ACTIVE",
						}, nil)
						
						// Expect waiting for listener to be ready
						m.GetListener(listenerID).Return(&listeners.Listener{
							ID:              listenerID,
							Name:            portObjectName,
							ProvisioningStatus: "ACTIVE",
						}, nil).AnyTimes()
						
						m.CreatePool(gomock.Any()).Return(&pools.Pool{
							ID:              fmt.Sprintf("pool-%s-%d", az, port),
							Name:            portObjectName,
							ProvisioningStatus: "ACTIVE",
						}, nil)
						m.CreateMonitor(gomock.Any()).Return(&monitors.Monitor{
							ID:              fmt.Sprintf("monitor-%s-%d", az, port),
							Name:            portObjectName,
							ProvisioningStatus: "ACTIVE",
						}, nil)
					}
				}
			},
			want:      false,
			wantError: nil,
		},
		{
			name: "migration scenario - legacy single AZ with old AvailabilityZone field",
			openStackCluster: func() *infrav1.OpenStackCluster {
				cluster := baseCluster.DeepCopy()
				cluster.Spec.APIServerLoadBalancer.AvailabilityZone = ptr.To("legacy-az")
				// AvailabilityZones will be automatically populated from AvailabilityZone
				return cluster
			}(),
			clusterResourceName: "test-cluster",
			apiServerPort:      6443,
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
				// Expect migration calls first
				m.ListLoadBalancers(gomock.Any()).Return([]loadbalancers.LoadBalancer{}, nil).AnyTimes()
				m.ListListeners(gomock.Any()).Return([]listeners.Listener{}, nil).AnyTimes()
				m.ListPools(gomock.Any()).Return([]pools.Pool{}, nil).AnyTimes()
				m.ListMonitors(gomock.Any()).Return([]monitors.Monitor{}, nil).AnyTimes()
				
				// Expect Octavia version check for allowed CIDRs support
				m.ListOctaviaVersions().Return([]apiversions.APIVersion{
					{ID: "2.24"},
					{ID: "2.23"},
					{ID: "2.22"},
				}, nil).AnyTimes()
				
				// Expect load balancer creation for migrated AZ
				m.ListLoadBalancerProviders().Return(octaviaProviders, nil)
				m.CreateLoadBalancer(gomock.Any()).Return(&loadbalancers.LoadBalancer{
					ID:                   "lb-legacy-az",
					Name:                 "k8s-clusterapi-cluster-test-cluster-legacy-az-kubeapi",
					ProvisioningStatus:   "ACTIVE",
					VipSubnetID:         "subnet-1",
				}, nil)
				
				// Expect waiting for load balancer to be active
				m.GetLoadBalancer("lb-legacy-az").Return(&loadbalancers.LoadBalancer{
					ID:                   "lb-legacy-az",
					Name:                 "k8s-clusterapi-cluster-test-cluster-legacy-az-kubeapi",
					ProvisioningStatus:   "ACTIVE",
					VipSubnetID:         "subnet-1",
				}, nil).AnyTimes()
				
				// Expect listener creation
				m.CreateListener(gomock.Any()).Return(&listeners.Listener{
					ID:              "listener-legacy-az",
					Name:            "k8s-clusterapi-cluster-test-cluster-legacy-az-kubeapi-6443",
					ProvisioningStatus: "ACTIVE",
				}, nil)
				
				// Expect waiting for listener to be ready
				m.GetListener("listener-legacy-az").Return(&listeners.Listener{
					ID:              "listener-legacy-az",
					Name:            "k8s-clusterapi-cluster-test-cluster-legacy-az-kubeapi-6443",
					ProvisioningStatus: "ACTIVE",
				}, nil).AnyTimes()
				
				// Expect pool creation
				m.CreatePool(gomock.Any()).Return(&pools.Pool{
					ID:              "pool-legacy-az",
					Name:            "k8s-clusterapi-cluster-test-cluster-legacy-az-kubeapi-6443",
					ProvisioningStatus: "ACTIVE",
				}, nil)
				
				// Expect monitor creation
				m.CreateMonitor(gomock.Any()).Return(&monitors.Monitor{
					ID:              "monitor-legacy-az",
					Name:            "k8s-clusterapi-cluster-test-cluster-legacy-az-kubeapi-6443",
					ProvisioningStatus: "ACTIVE",
				}, nil)
			},
			want:      false,
			wantError: nil,
		},
		{
			name: "error scenario - missing load balancer network information",
			openStackCluster: func() *infrav1.OpenStackCluster {
				cluster := baseCluster.DeepCopy()
				cluster.Status.APIServerLoadBalancer = nil // Missing load balancer network
				return cluster
			}(),
			clusterResourceName: "test-cluster",
			apiServerPort:      6443,
			expectLoadBalancer: func(_ *mock.MockLbClientMockRecorder) {
				// No expectations as it should fail early
			},
			want:      false,
			wantError: fmt.Errorf("load balancer network information not available"),
		},
		{
			name: "error scenario - mismatch between AZs and subnets",
			openStackCluster: func() *infrav1.OpenStackCluster {
				cluster := baseCluster.DeepCopy()
				cluster.Spec.APIServerLoadBalancer.AvailabilityZones = []string{"az1", "az2", "az3", "az4"} // 4 AZs but only 3 subnets
				return cluster
			}(),
			clusterResourceName: "test-cluster",
			apiServerPort:      6443,
			expectLoadBalancer: func(_ *mock.MockLbClientMockRecorder) {
				// No expectations as it should fail early
			},
			want:      false,
			wantError: fmt.Errorf("mismatch between availability zones and subnets: more AZs than subnets"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			log := testr.New(t)

			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			lbs, err := NewService(scope.NewWithLogger(mockScopeFactory, log))
			g.Expect(err).NotTo(HaveOccurred())

			// Set up load balancer mock expectations
			tt.expectLoadBalancer(mockScopeFactory.LbClient.EXPECT())
			
			// Set up network client mock expectations for CreateFloatingIP calls
			networkRecorder := mockScopeFactory.NetworkClient.EXPECT()
			networkRecorder.CreateFloatingIP(gomock.Any()).Return(&floatingips.FloatingIP{
				ID:         "floating-ip-id",
				FloatingIP: "192.168.1.100",
			}, nil).AnyTimes()
			
			// Add expectation for ListFloatingIP calls
			networkRecorder.ListFloatingIP(gomock.Any()).Return([]floatingips.FloatingIP{}, nil).AnyTimes()

			got, err := lbs.ReconcileLoadBalancers(tt.openStackCluster, tt.clusterResourceName, tt.apiServerPort)
			if tt.wantError != nil {
				g.Expect(err).To(MatchError(tt.wantError))
			} else {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(got).To(Equal(tt.want))
			}
		})
	}
}

func Test_migrateAPIServerLoadBalancer(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	baseCluster := &infrav1.OpenStackCluster{
		Spec: infrav1.OpenStackClusterSpec{
			APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
				Enabled: ptr.To(true),
				AdditionalPorts: []int{8080},
			},
		},
	}

	tests := []struct {
		name                string
		openStackCluster    *infrav1.OpenStackCluster
		clusterResourceName string
		azInfo              azSubnet
		apiServerPort       int
		expectLoadBalancer  func(m *mock.MockLbClientMockRecorder)
		wantError           error
	}{
		{
			name:             "migrate single-AZ to multi-AZ format with default AZ",
			openStackCluster: baseCluster.DeepCopy(),
			clusterResourceName: "test-cluster",
			azInfo: azSubnet{
				az: func() *string { s := "default"; return &s }(),
				subnet: infrav1.Subnet{ID: "subnet-1"},
			},
			apiServerPort: 6443,
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
				// Check if old load balancer exists for renaming
				m.ListLoadBalancers(loadbalancers.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi"}).Return([]loadbalancers.LoadBalancer{
					{
						ID:   "existing-lb",
						Name: "k8s-clusterapi-cluster-test-cluster-kubeapi",
					},
				}, nil)
				
				// Rename load balancer
				m.UpdateLoadBalancer("existing-lb", gomock.Any()).Return(&loadbalancers.LoadBalancer{
					ID:   "existing-lb",
					Name: "k8s-clusterapi-cluster-test-cluster-default-kubeapi",
				}, nil)
				
				// Check and rename listener for port 6443
				m.ListListeners(listeners.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-6443"}).Return([]listeners.Listener{
					{
						ID:   "existing-listener-6443",
						Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-6443",
					},
				}, nil)
				m.UpdateListener("existing-listener-6443", gomock.Any()).Return(&listeners.Listener{}, nil)
				
				// Check and rename pool for port 6443
				m.ListPools(pools.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-6443"}).Return([]pools.Pool{
					{
						ID:   "existing-pool-6443",
						Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-6443",
					},
				}, nil)
				m.UpdatePool("existing-pool-6443", gomock.Any()).Return(&pools.Pool{}, nil)
				
				// Check and rename monitor for port 6443
				m.ListMonitors(monitors.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-6443"}).Return([]monitors.Monitor{
					{
						ID:   "existing-monitor-6443",
						Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-6443",
					},
				}, nil)
				m.UpdateMonitor("existing-monitor-6443", gomock.Any()).Return(&monitors.Monitor{}, nil)
				
				// Check and rename listener for port 8080
				m.ListListeners(listeners.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-8080"}).Return([]listeners.Listener{
					{
						ID:   "existing-listener-8080",
						Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-8080",
					},
				}, nil)
				m.UpdateListener("existing-listener-8080", gomock.Any()).Return(&listeners.Listener{}, nil)
				
				// Check and rename pool for port 8080
				m.ListPools(pools.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-8080"}).Return([]pools.Pool{
					{
						ID:   "existing-pool-8080",
						Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-8080",
					},
				}, nil)
				m.UpdatePool("existing-pool-8080", gomock.Any()).Return(&pools.Pool{}, nil)
				
				// Check and rename monitor for port 8080
				m.ListMonitors(monitors.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-8080"}).Return([]monitors.Monitor{
					{
						ID:   "existing-monitor-8080",
						Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-8080",
					},
				}, nil)
				m.UpdateMonitor("existing-monitor-8080", gomock.Any()).Return(&monitors.Monitor{}, nil)
			},
			wantError: nil,
		},
		{
			name:             "migrate to named AZ format",
			openStackCluster: baseCluster.DeepCopy(),
			clusterResourceName: "test-cluster",
			azInfo: azSubnet{
				az: func() *string { s := "us-west-1a"; return &s }(),
				subnet: infrav1.Subnet{ID: "subnet-1"},
			},
			apiServerPort: 6443,
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
				// Check if old load balancer exists for renaming
				m.ListLoadBalancers(loadbalancers.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi"}).Return([]loadbalancers.LoadBalancer{
					{
						ID:   "existing-lb",
						Name: "k8s-clusterapi-cluster-test-cluster-kubeapi",
					},
				}, nil)
				
				// Rename load balancer
				m.UpdateLoadBalancer("existing-lb", gomock.Any()).Return(&loadbalancers.LoadBalancer{
					ID:   "existing-lb",
					Name: "k8s-clusterapi-cluster-test-cluster-us-west-1a-kubeapi",
				}, nil)
				
				// Check and rename listener for port 6443
				m.ListListeners(listeners.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-6443"}).Return([]listeners.Listener{
					{
						ID:   "existing-listener-6443",
						Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-6443",
					},
				}, nil)
				m.UpdateListener("existing-listener-6443", gomock.Any()).Return(&listeners.Listener{}, nil)
				
				// Check and rename pool for port 6443
				m.ListPools(pools.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-6443"}).Return([]pools.Pool{
					{
						ID:   "existing-pool-6443",
						Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-6443",
					},
				}, nil)
				m.UpdatePool("existing-pool-6443", gomock.Any()).Return(&pools.Pool{}, nil)
				
				// Check and rename monitor for port 6443
				m.ListMonitors(monitors.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-6443"}).Return([]monitors.Monitor{
					{
						ID:   "existing-monitor-6443",
						Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-6443",
					},
				}, nil)
				m.UpdateMonitor("existing-monitor-6443", gomock.Any()).Return(&monitors.Monitor{}, nil)
				
				// Check and rename listener for port 8080
				m.ListListeners(listeners.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-8080"}).Return([]listeners.Listener{
					{
						ID:   "existing-listener-8080",
						Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-8080",
					},
				}, nil)
				m.UpdateListener("existing-listener-8080", gomock.Any()).Return(&listeners.Listener{}, nil)
				
				// Check and rename pool for port 8080
				m.ListPools(pools.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-8080"}).Return([]pools.Pool{
					{
						ID:   "existing-pool-8080",
						Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-8080",
					},
				}, nil)
				m.UpdatePool("existing-pool-8080", gomock.Any()).Return(&pools.Pool{}, nil)
				
				// Check and rename monitor for port 8080
				m.ListMonitors(monitors.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-8080"}).Return([]monitors.Monitor{
					{
						ID:   "existing-monitor-8080",
						Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-8080",
					},
				}, nil)
				m.UpdateMonitor("existing-monitor-8080", gomock.Any()).Return(&monitors.Monitor{}, nil)
			},
			wantError: nil,
		},
		{
			name:             "no migration needed - no existing resources",
			openStackCluster: baseCluster.DeepCopy(),
			clusterResourceName: "new-cluster",
			azInfo: azSubnet{
				az: func() *string { s := testAZ1; return &s }(),
				subnet: infrav1.Subnet{ID: "subnet-1"},
			},
			apiServerPort: 6443,
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
				// Check for old load balancer - doesn't exist
				m.ListLoadBalancers(loadbalancers.ListOpts{Name: "k8s-clusterapi-cluster-new-cluster-kubeapi"}).Return([]loadbalancers.LoadBalancer{}, nil)
				
				// Check for old listeners - don't exist
				m.ListListeners(listeners.ListOpts{Name: "k8s-clusterapi-cluster-new-cluster-kubeapi-6443"}).Return([]listeners.Listener{}, nil)
				m.ListListeners(listeners.ListOpts{Name: "k8s-clusterapi-cluster-new-cluster-kubeapi-8080"}).Return([]listeners.Listener{}, nil)
				
				// Check for old pools - don't exist
				m.ListPools(pools.ListOpts{Name: "k8s-clusterapi-cluster-new-cluster-kubeapi-6443"}).Return([]pools.Pool{}, nil)
				m.ListPools(pools.ListOpts{Name: "k8s-clusterapi-cluster-new-cluster-kubeapi-8080"}).Return([]pools.Pool{}, nil)
				
				// Check for old monitors - don't exist
				m.ListMonitors(monitors.ListOpts{Name: "k8s-clusterapi-cluster-new-cluster-kubeapi-6443"}).Return([]monitors.Monitor{}, nil)
				m.ListMonitors(monitors.ListOpts{Name: "k8s-clusterapi-cluster-new-cluster-kubeapi-8080"}).Return([]monitors.Monitor{}, nil)
			},
			wantError: nil,
		},
		{
			name:             "migration skipped when load balancer disabled",
			openStackCluster: func() *infrav1.OpenStackCluster {
				cluster := baseCluster.DeepCopy()
				cluster.Spec.APIServerLoadBalancer.Enabled = ptr.To(false)
				return cluster
			}(),
			clusterResourceName: "test-cluster",
			azInfo: azSubnet{
				az: func() *string { s := "az1"; return &s }(),
				subnet: infrav1.Subnet{ID: "subnet-1"},
			},
			apiServerPort: 6443,
			expectLoadBalancer: func(_ *mock.MockLbClientMockRecorder) {
				// No expectations as migration should be skipped
			},
			wantError: nil,
		},
		{
			name:             "resources already have correct names - no updates needed",
			openStackCluster: baseCluster.DeepCopy(),
			clusterResourceName: "test-cluster",
			azInfo: azSubnet{
				az: func() *string { s := "az1"; return &s }(),
				subnet: infrav1.Subnet{ID: "subnet-1"},
			},
			apiServerPort: 6443,
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
				// Check if old load balancer exists - it doesn't
				m.ListLoadBalancers(loadbalancers.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi"}).Return([]loadbalancers.LoadBalancer{}, nil)
				
				// Check listeners - they don't exist with old names
				m.ListListeners(listeners.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-6443"}).Return([]listeners.Listener{}, nil)
				m.ListListeners(listeners.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-8080"}).Return([]listeners.Listener{}, nil)
				
				// Check pools - they don't exist with old names
				m.ListPools(pools.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-6443"}).Return([]pools.Pool{}, nil)
				m.ListPools(pools.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-8080"}).Return([]pools.Pool{}, nil)
				
				// Check monitors - they don't exist with old names
				m.ListMonitors(monitors.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-6443"}).Return([]monitors.Monitor{}, nil)
				m.ListMonitors(monitors.ListOpts{Name: "k8s-clusterapi-cluster-test-cluster-kubeapi-8080"}).Return([]monitors.Monitor{}, nil)
			},
			wantError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			log := testr.New(t)

			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			lbs, err := NewService(scope.NewWithLogger(mockScopeFactory, log))
			g.Expect(err).NotTo(HaveOccurred())

			tt.expectLoadBalancer(mockScopeFactory.LbClient.EXPECT())

			err = lbs.migrateAPIServerLoadBalancer(tt.openStackCluster, tt.clusterResourceName, tt.azInfo, tt.apiServerPort)
			if tt.wantError != nil {
				g.Expect(err).To(MatchError(tt.wantError))
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
		})
	}
}

func Test_APIServerLoadBalancer_AllowsCrossAZLoadBalancerMembers(t *testing.T) {
	tests := []struct {
		name         string
		allowCrossAZ *bool
		expected     bool
	}{
		{
			name:         "default (nil) should be false",
			allowCrossAZ: nil,
			expected:     false,
		},
		{
			name:         "explicit false",
			allowCrossAZ: ptr.To(false),
			expected:     false,
		},
		{
			name:         "explicit true",
			allowCrossAZ: ptr.To(true),
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			apiServerLB := &infrav1.APIServerLoadBalancer{
				AllowCrossAZLoadBalancerMembers: tt.allowCrossAZ,
			}

			result := apiServerLB.AllowsCrossAZLoadBalancerMembers()
			g.Expect(result).To(Equal(tt.expected))
		})
	}
}

func Test_ReconcileLoadBalancerMember_CrossAZLogic(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name                   string
		allowCrossAZ          *bool
		machineFailureDomain  string
		existingLoadBalancers []infrav1.LoadBalancer
		legacyLoadBalancer    *infrav1.LoadBalancer
		expectMemberCreations []string // Pool IDs where CreatePoolMember should be called
		wantError             bool
	}{
		{
			name:                 "default behavior (same-AZ only) - multi-AZ scenario",
			allowCrossAZ:         nil, // Default is false
			machineFailureDomain: "us-west-1a",
			existingLoadBalancers: []infrav1.LoadBalancer{
				{
					ID:               "lb-us-west-1a",
					Name:             "k8s-clusterapi-cluster-test-cluster-us-west-1a-kubeapi",
					AvailabilityZone: ptr.To("us-west-1a"),
				},
				{
					ID:               "lb-us-west-1b", 
					Name:             "k8s-clusterapi-cluster-test-cluster-us-west-1b-kubeapi",
					AvailabilityZone: ptr.To("us-west-1b"),
				},
			},
			expectMemberCreations: []string{"pool-lb-us-west-1a"}, // Only same AZ
		},
		{
			name:                 "cross-AZ enabled - multi-AZ scenario",
			allowCrossAZ:         ptr.To(true),
			machineFailureDomain: "us-west-1a",
			existingLoadBalancers: []infrav1.LoadBalancer{
				{
					ID:               "lb-us-west-1a",
					Name:             "k8s-clusterapi-cluster-test-cluster-us-west-1a-kubeapi", 
					AvailabilityZone: ptr.To("us-west-1a"),
				},
				{
					ID:               "lb-us-west-1b",
					Name:             "k8s-clusterapi-cluster-test-cluster-us-west-1b-kubeapi",
					AvailabilityZone: ptr.To("us-west-1b"),
				},
			},
			expectMemberCreations: []string{"pool-lb-us-west-1a", "pool-lb-us-west-1b"}, // All AZs when cross-AZ enabled
		},
		{
			name:                 "legacy single LB - always works regardless of setting",
			allowCrossAZ:         ptr.To(false), // Same-AZ only
			machineFailureDomain: "us-west-1a",
			legacyLoadBalancer: &infrav1.LoadBalancer{
				ID:   "legacy-lb-id",
				Name: "k8s-clusterapi-cluster-test-cluster-kubeapi",
			},
			existingLoadBalancers: []infrav1.LoadBalancer{},
			expectMemberCreations: []string{"pool-legacy-lb-id"}, // Legacy LB always works
		},
		{
			name:                 "same-AZ only with machine in different AZ",
			allowCrossAZ:         ptr.To(false),
			machineFailureDomain: "us-west-1c", // Different AZ
			existingLoadBalancers: []infrav1.LoadBalancer{
				{
					ID:               "lb-us-west-1a",
					Name:             "k8s-clusterapi-cluster-test-cluster-us-west-1a-kubeapi",
					AvailabilityZone: ptr.To("us-west-1a"),
				},
				{
					ID:               "lb-us-west-1b",
					Name:             "k8s-clusterapi-cluster-test-cluster-us-west-1b-kubeapi", 
					AvailabilityZone: ptr.To("us-west-1b"),
				},
			},
			expectMemberCreations: []string{}, // No registrations - machine AZ doesn't match any LB AZ
		},
		{
			name:                 "machine without AZ - same-AZ only mode",
			allowCrossAZ:         ptr.To(false),
			machineFailureDomain: "", // No failure domain
			existingLoadBalancers: []infrav1.LoadBalancer{
				{
					ID:               "lb-us-west-1a",
					Name:             "k8s-clusterapi-cluster-test-cluster-us-west-1a-kubeapi",
					AvailabilityZone: ptr.To("us-west-1a"),
				},
			},
			expectMemberCreations: []string{}, // No registration when machine has no AZ and same-AZ only
		},
		{
			name:                 "machine without AZ - cross-AZ enabled",
			allowCrossAZ:         ptr.To(true),
			machineFailureDomain: "", // No failure domain
			existingLoadBalancers: []infrav1.LoadBalancer{
				{
					ID:               "lb-us-west-1a",
					Name:             "k8s-clusterapi-cluster-test-cluster-us-west-1a-kubeapi",
					AvailabilityZone: ptr.To("us-west-1a"),
				},
				{
					ID:               "lb-us-west-1b",
					Name:             "k8s-clusterapi-cluster-test-cluster-us-west-1b-kubeapi",
					AvailabilityZone: ptr.To("us-west-1b"),
				},
			},
			expectMemberCreations: []string{"pool-lb-us-west-1a", "pool-lb-us-west-1b"}, // Register to all when cross-AZ enabled
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			log := testr.New(t)

			// Create mock scope factory
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			
			// Create service
			lbs, err := NewService(scope.NewWithLogger(mockScopeFactory, log))
			g.Expect(err).NotTo(HaveOccurred())

			// Create OpenStackCluster with test configuration
			openStackCluster := &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
						Enabled:                         ptr.To(true),
						AllowCrossAZLoadBalancerMembers: tt.allowCrossAZ,
					},
					ControlPlaneEndpoint: &clusterv1.APIEndpoint{
						Host: "192.168.1.100",
						Port: 6443,
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Ready: true,
					Network: &infrav1.NetworkStatusWithSubnets{
						NetworkStatus: infrav1.NetworkStatus{
							ID:   "network-id",
							Name: "test-network",
						},
						Subnets: []infrav1.Subnet{
							{
								ID:   "subnet-id", 
								Name: "test-subnet",
								CIDR: "10.0.0.0/24",
							},
						},
					},
					APIServerLoadBalancer:  tt.legacyLoadBalancer,
					APIServerLoadBalancers: tt.existingLoadBalancers,
				},
			}

			openStackMachine := &infrav1.OpenStackMachine{}

			// Track actual member creations
			actualMemberCreations := []string{}

			// Collect all load balancers to process (legacy + multi-AZ)
			allLoadBalancers := tt.existingLoadBalancers
			if tt.legacyLoadBalancer != nil {
				allLoadBalancers = append(allLoadBalancers, *tt.legacyLoadBalancer)
			}

			// Set up mock expectations for each load balancer
			// Note: The function will call these methods for ALL load balancers to check existence,
			// not just the ones that should get members created
			for _, lb := range allLoadBalancers {
				poolID := "pool-" + lb.ID
				poolName := lb.Name + "-6443" // Pool name includes the port
				
				// Mock checkIfPoolExists call - the function searches by pool name
				mockScopeFactory.LbClient.EXPECT().
					ListPools(pools.ListOpts{
						Name: poolName,
					}).
					Return([]pools.Pool{
						{
							ID:   poolID,
							Name: poolName,
						},
					}, nil).
					AnyTimes()
			}

			// Set up member-related expectations only for pools that should be processed
			// (based on our cross-AZ logic)
			for _, lb := range allLoadBalancers {
				poolID := "pool-" + lb.ID
				poolName := lb.Name + "-6443"
				memberName := poolName + "-" // The function appends a dash for member name
				
				// Check if this LB should be processed based on our cross-AZ logic
				shouldProcess := false
				allowCrossAZ := openStackCluster.Spec.APIServerLoadBalancer.AllowsCrossAZLoadBalancerMembers()
				
				if tt.legacyLoadBalancer != nil && lb.ID == tt.legacyLoadBalancer.ID {
					// Legacy LB always gets processed
					shouldProcess = true
				} else if allowCrossAZ {
					// Cross-AZ enabled: process all LBs
					shouldProcess = true
				} else if tt.machineFailureDomain != "" && lb.AvailabilityZone != nil {
					// Same-AZ only: process only if machine AZ matches LB AZ
					shouldProcess = tt.machineFailureDomain == *lb.AvailabilityZone
				}

				if shouldProcess {
					// Mock checkIfLbMemberExists call
					mockScopeFactory.LbClient.EXPECT().
						ListPoolMember(poolID, pools.ListMembersOpts{
							Name: memberName,
						}).
						Return([]pools.Member{}, nil) // Return empty to trigger creation

					// Set up CreatePoolMember expectation only for pools that should have members created
					shouldCreateMember := false
					for _, expectedPoolID := range tt.expectMemberCreations {
						if poolID == expectedPoolID {
							shouldCreateMember = true
							break
						}
					}
					
					if shouldCreateMember {
						mockScopeFactory.LbClient.EXPECT().
							CreatePoolMember(poolID, gomock.Any()).
							DoAndReturn(func(poolID string, _ pools.CreateMemberOptsBuilder) (*pools.Member, error) {
								actualMemberCreations = append(actualMemberCreations, poolID)
								return &pools.Member{
									ID:      "member-" + poolID,
									Address: "10.0.0.100",
								}, nil
							})

						// Mock the GetLoadBalancer call that happens during wait for active status
						mockScopeFactory.LbClient.EXPECT().
							GetLoadBalancer(lb.ID).
							Return(&loadbalancers.LoadBalancer{
								ID:                 lb.ID,
								Name:               lb.Name,
								ProvisioningStatus: "ACTIVE",
							}, nil).
							AnyTimes()
					}
				}
			}

			// Call the function under test
			err = lbs.ReconcileLoadBalancerMember(openStackCluster, openStackMachine, "test-cluster", "10.0.0.100", tt.machineFailureDomain)

			// Verify results
			if tt.wantError {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(actualMemberCreations).To(ConsistOf(tt.expectMemberCreations))
			}
		})
	}
}
func Test_CreateLoadBalancer_UsesVipSubnetIDOverride_Mapping(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	g := NewWithT(t)
	log := testr.New(t)
	mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")

	lbs, err := NewService(scope.NewWithLogger(mockScopeFactory, log))
	g.Expect(err).NotTo(HaveOccurred())

	// Cluster with LB enabled, explicit mapping present
	cluster := &infrav1.OpenStackCluster{
		Spec: infrav1.OpenStackClusterSpec{
			APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
				Enabled: ptr.To(true),
				// mapping takes precedence
				AvailabilityZoneSubnets: []infrav1.AZSubnetMapping{
					{AvailabilityZone: "az1", Subnet: infrav1.SubnetParam{ID: ptr.To("subnet-1")}},
					{AvailabilityZone: "az2", Subnet: infrav1.SubnetParam{ID: ptr.To("subnet-2")}},
				},
			},
		},
		Status: infrav1.OpenStackClusterStatus{
			APIServerLoadBalancer: &infrav1.LoadBalancer{
				LoadBalancerNetwork: &infrav1.NetworkStatusWithSubnets{
					NetworkStatus: infrav1.NetworkStatus{
						ID:   "lb-net",
						Name: "lb-network",
					},
					// controller resolved order matches mapping order
					Subnets: []infrav1.Subnet{
						{ID: "subnet-1", CIDR: "192.168.1.0/24"},
						{ID: "subnet-2", CIDR: "192.168.2.0/24"},
					},
				},
			},
			ExternalNetwork: &infrav1.NetworkStatus{ID: "ext-id"},
			Network: &infrav1.NetworkStatusWithSubnets{
				NetworkStatus: infrav1.NetworkStatus{ID: "cluster-net"},
				Subnets:       []infrav1.Subnet{{ID: "cluster-subnet-1", CIDR: "10.0.0.0/24"}},
			},
		},
	}

	// Migration lists - not relevant here; allow empty lists
	mockScopeFactory.LbClient.EXPECT().ListLoadBalancers(gomock.Any()).Return([]loadbalancers.LoadBalancer{}, nil).AnyTimes()
	mockScopeFactory.LbClient.EXPECT().ListListeners(gomock.Any()).Return([]listeners.Listener{}, nil).AnyTimes()
	mockScopeFactory.LbClient.EXPECT().ListPools(gomock.Any()).Return([]pools.Pool{}, nil).AnyTimes()
	mockScopeFactory.LbClient.EXPECT().ListMonitors(gomock.Any()).Return([]monitors.Monitor{}, nil).AnyTimes()

	// Octavia features
	mockScopeFactory.LbClient.EXPECT().ListLoadBalancerProviders().Return([]providers.Provider{{Name: "ovn"}}, nil).AnyTimes()
	mockScopeFactory.LbClient.EXPECT().ListOctaviaVersions().Return([]apiversions.APIVersion{{ID: "2.23"}, {ID: "2.24"}}, nil).AnyTimes()

	// Expect CreateLoadBalancer for az1 with VipSubnetID "subnet-1"
	mockScopeFactory.LbClient.EXPECT().
		CreateLoadBalancer(gomock.Any()).
		DoAndReturn(func(arg any) (*loadbalancers.LoadBalancer, error) {
			opts := arg.(loadbalancers.CreateOpts)
			g.Expect(opts.VipSubnetID).To(Equal("subnet-1"))
			return &loadbalancers.LoadBalancer{
				ID:                 "lb-az1",
				Name:               "k8s-clusterapi-cluster-test-az1-kubeapi",
				VipSubnetID:        opts.VipSubnetID,
				ProvisioningStatus: "ACTIVE",
			}, nil
		})

	// Get after creation
	mockScopeFactory.LbClient.EXPECT().GetLoadBalancer("lb-az1").Return(&loadbalancers.LoadBalancer{
		ID:                 "lb-az1",
		Name:               "k8s-clusterapi-cluster-test-az1-kubeapi",
		VipSubnetID:        "subnet-1",
		ProvisioningStatus: "ACTIVE",
	}, nil).AnyTimes()

	// Listener/pool/monitor for port 6443 (minimal path)
	mockScopeFactory.LbClient.EXPECT().CreateListener(gomock.Any()).DoAndReturn(func(arg any) (*listeners.Listener, error) {
		return &listeners.Listener{ID: "listener-az1-6443", Name: "k8s-clusterapi-cluster-test-az1-kubeapi-6443", ProvisioningStatus: "ACTIVE"}, nil
	})
	mockScopeFactory.LbClient.EXPECT().GetListener("listener-az1-6443").Return(&listeners.Listener{ID: "listener-az1-6443"}, nil).AnyTimes()
	mockScopeFactory.LbClient.EXPECT().CreatePool(gomock.Any()).Return(&pools.Pool{ID: "pool-az1-6443", Name: "k8s-clusterapi-cluster-test-az1-kubeapi-6443"}, nil)
	mockScopeFactory.LbClient.EXPECT().CreateMonitor(gomock.Any()).Return(&monitors.Monitor{ID: "monitor-az1-6443", Name: "k8s-clusterapi-cluster-test-az1-kubeapi-6443"}, nil)

	// Expect CreateLoadBalancer for az2 with VipSubnetID "subnet-2"
	mockScopeFactory.LbClient.EXPECT().
		CreateLoadBalancer(gomock.Any()).
		DoAndReturn(func(arg any) (*loadbalancers.LoadBalancer, error) {
			opts := arg.(loadbalancers.CreateOpts)
			g.Expect(opts.VipSubnetID).To(Equal("subnet-2"))
			return &loadbalancers.LoadBalancer{
				ID:                 "lb-az2",
				Name:               "k8s-clusterapi-cluster-test-az2-kubeapi",
				VipSubnetID:        opts.VipSubnetID,
				ProvisioningStatus: "ACTIVE",
			}, nil
		})

	mockScopeFactory.LbClient.EXPECT().GetLoadBalancer("lb-az2").Return(&loadbalancers.LoadBalancer{
		ID:                 "lb-az2",
		Name:               "k8s-clusterapi-cluster-test-az2-kubeapi",
		VipSubnetID:        "subnet-2",
		ProvisioningStatus: "ACTIVE",
	}, nil).AnyTimes()

	mockScopeFactory.LbClient.EXPECT().CreateListener(gomock.Any()).DoAndReturn(func(arg any) (*listeners.Listener, error) {
		return &listeners.Listener{ID: "listener-az2-6443", Name: "k8s-clusterapi-cluster-test-az2-kubeapi-6443", ProvisioningStatus: "ACTIVE"}, nil
	})
	mockScopeFactory.LbClient.EXPECT().GetListener("listener-az2-6443").Return(&listeners.Listener{ID: "listener-az2-6443"}, nil).AnyTimes()
	mockScopeFactory.LbClient.EXPECT().CreatePool(gomock.Any()).Return(&pools.Pool{ID: "pool-az2-6443", Name: "k8s-clusterapi-cluster-test-az2-kubeapi-6443"}, nil)
	mockScopeFactory.LbClient.EXPECT().CreateMonitor(gomock.Any()).Return(&monitors.Monitor{ID: "monitor-az2-6443", Name: "k8s-clusterapi-cluster-test-az2-kubeapi-6443"}, nil)

	// Networking FIP calls should be allowed but not required (DisableAPIServerFloatingIP defaults false).
	networkRecorder := mockScopeFactory.NetworkClient.EXPECT()
	networkRecorder.CreateFloatingIP(gomock.Any()).Return(&floatingips.FloatingIP{ID: "fip-id", FloatingIP: "198.51.100.10"}, nil).AnyTimes()
	networkRecorder.ListFloatingIP(gomock.Any()).Return([]floatingips.FloatingIP{}, nil).AnyTimes()

	_, err = lbs.ReconcileLoadBalancers(cluster, "test", 6443)
	g.Expect(err).NotTo(HaveOccurred())
}

func Test_CreateLoadBalancer_UsesVipSubnetIDOverride_Positional(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	g := NewWithT(t)
	log := testr.New(t)
	mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")

	lbs, err := NewService(scope.NewWithLogger(mockScopeFactory, log))
	g.Expect(err).NotTo(HaveOccurred())

	// Cluster with LB enabled, no mapping, positional fallback
	cluster := &infrav1.OpenStackCluster{
		Spec: infrav1.OpenStackClusterSpec{
			APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
				Enabled:           ptr.To(true),
				AvailabilityZones: []string{"az1", "az2"},
			},
		},
		Status: infrav1.OpenStackClusterStatus{
			APIServerLoadBalancer: &infrav1.LoadBalancer{
				LoadBalancerNetwork: &infrav1.NetworkStatusWithSubnets{
					NetworkStatus: infrav1.NetworkStatus{ID: "lb-net"},
					Subnets: []infrav1.Subnet{
						{ID: "subnet-1"},
						{ID: "subnet-2"},
					},
				},
			},
			ExternalNetwork: &infrav1.NetworkStatus{ID: "ext-id"},
			Network: &infrav1.NetworkStatusWithSubnets{
				NetworkStatus: infrav1.NetworkStatus{ID: "cluster-net"},
				Subnets:       []infrav1.Subnet{{ID: "cluster-subnet-1"}},
			},
		},
	}

	// Migration lists - allow empty
	mockScopeFactory.LbClient.EXPECT().ListLoadBalancers(gomock.Any()).Return([]loadbalancers.LoadBalancer{}, nil).AnyTimes()
	mockScopeFactory.LbClient.EXPECT().ListListeners(gomock.Any()).Return([]listeners.Listener{}, nil).AnyTimes()
	mockScopeFactory.LbClient.EXPECT().ListPools(gomock.Any()).Return([]pools.Pool{}, nil).AnyTimes()
	mockScopeFactory.LbClient.EXPECT().ListMonitors(gomock.Any()).Return([]monitors.Monitor{}, nil).AnyTimes()

	mockScopeFactory.LbClient.EXPECT().ListLoadBalancerProviders().Return([]providers.Provider{{Name: "ovn"}}, nil).AnyTimes()
	mockScopeFactory.LbClient.EXPECT().ListOctaviaVersions().Return([]apiversions.APIVersion{{ID: "2.24"}}, nil).AnyTimes()

	// az1 -> subnet-1
	mockScopeFactory.LbClient.EXPECT().
		CreateLoadBalancer(gomock.Any()).
		DoAndReturn(func(arg any) (*loadbalancers.LoadBalancer, error) {
			opts := arg.(loadbalancers.CreateOpts)
			g.Expect(opts.VipSubnetID).To(Equal("subnet-1"))
			return &loadbalancers.LoadBalancer{ID: "lb-az1", Name: "k8s-clusterapi-cluster-test-az1-kubeapi", VipSubnetID: opts.VipSubnetID, ProvisioningStatus: "ACTIVE"}, nil
		})
	mockScopeFactory.LbClient.EXPECT().GetLoadBalancer("lb-az1").Return(&loadbalancers.LoadBalancer{ID: "lb-az1", ProvisioningStatus: "ACTIVE"}, nil).AnyTimes()
	mockScopeFactory.LbClient.EXPECT().CreateListener(gomock.Any()).Return(&listeners.Listener{ID: "listener-az1-6443", ProvisioningStatus: "ACTIVE"}, nil)
	mockScopeFactory.LbClient.EXPECT().GetListener("listener-az1-6443").Return(&listeners.Listener{ID: "listener-az1-6443"}, nil).AnyTimes()
	mockScopeFactory.LbClient.EXPECT().CreatePool(gomock.Any()).Return(&pools.Pool{ID: "pool-az1-6443"}, nil)
	mockScopeFactory.LbClient.EXPECT().CreateMonitor(gomock.Any()).Return(&monitors.Monitor{ID: "monitor-az1-6443"}, nil)

	// az2 -> subnet-2
	mockScopeFactory.LbClient.EXPECT().
		CreateLoadBalancer(gomock.Any()).
		DoAndReturn(func(arg any) (*loadbalancers.LoadBalancer, error) {
			opts := arg.(loadbalancers.CreateOpts)
			g.Expect(opts.VipSubnetID).To(Equal("subnet-2"))
			return &loadbalancers.LoadBalancer{ID: "lb-az2", Name: "k8s-clusterapi-cluster-test-az2-kubeapi", VipSubnetID: opts.VipSubnetID, ProvisioningStatus: "ACTIVE"}, nil
		})
	mockScopeFactory.LbClient.EXPECT().GetLoadBalancer("lb-az2").Return(&loadbalancers.LoadBalancer{ID: "lb-az2", ProvisioningStatus: "ACTIVE"}, nil).AnyTimes()
	mockScopeFactory.LbClient.EXPECT().CreateListener(gomock.Any()).Return(&listeners.Listener{ID: "listener-az2-6443", ProvisioningStatus: "ACTIVE"}, nil)
	mockScopeFactory.LbClient.EXPECT().GetListener("listener-az2-6443").Return(&listeners.Listener{ID: "listener-az2-6443"}, nil).AnyTimes()
	mockScopeFactory.LbClient.EXPECT().CreatePool(gomock.Any()).Return(&pools.Pool{ID: "pool-az2-6443"}, nil)
	mockScopeFactory.LbClient.EXPECT().CreateMonitor(gomock.Any()).Return(&monitors.Monitor{ID: "monitor-az2-6443"}, nil)

	// Network FIP helpers
	networkRecorder := mockScopeFactory.NetworkClient.EXPECT()
	networkRecorder.CreateFloatingIP(gomock.Any()).Return(&floatingips.FloatingIP{ID: "fip-id", FloatingIP: "198.51.100.11"}, nil).AnyTimes()
	networkRecorder.ListFloatingIP(gomock.Any()).Return([]floatingips.FloatingIP{}, nil).AnyTimes()

	_, err = lbs.ReconcileLoadBalancers(cluster, "test", 6443)
	g.Expect(err).NotTo(HaveOccurred())
}

func Test_DeleteLoadBalancer_PendingDeleteRequeues(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	g := NewWithT(t)
	log := testr.New(t)
	mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")

	lbs, err := NewService(scope.NewWithLogger(mockScopeFactory, log))
	g.Expect(err).NotTo(HaveOccurred())

	cluster := &infrav1.OpenStackCluster{
		Spec: infrav1.OpenStackClusterSpec{
			APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
				Enabled:           ptr.To(true),
				AvailabilityZones: []string{"az1"},
			},
		},
	}

	// Legacy name (no AZ) - not existing
	mockScopeFactory.LbClient.EXPECT().
		ListLoadBalancers(loadbalancers.ListOpts{Name: "k8s-clusterapi-cluster-test-kubeapi"}).
		Return([]loadbalancers.LoadBalancer{}, nil)

	// AZ name exists but is PENDING_DELETE
	mockScopeFactory.LbClient.EXPECT().
		ListLoadBalancers(loadbalancers.ListOpts{Name: "k8s-clusterapi-cluster-test-az1-kubeapi"}).
		Return([]loadbalancers.LoadBalancer{
			{
				ID:                 "lb-az1",
				Name:               "k8s-clusterapi-cluster-test-az1-kubeapi",
				ProvisioningStatus: "PENDING_DELETE",
			},
		}, nil)

	// No DeleteLoadBalancer should be called for PENDING_DELETE

	result, err := lbs.DeleteLoadBalancer(cluster, "test")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).NotTo(BeNil()) // requeue to wait for deletion
}

func Test_DeleteLoadBalancer_CascadeDeletionRequeues(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	g := NewWithT(t)
	log := testr.New(t)
	mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")

	lbs, err := NewService(scope.NewWithLogger(mockScopeFactory, log))
	g.Expect(err).NotTo(HaveOccurred())

	cluster := &infrav1.OpenStackCluster{
		Spec: infrav1.OpenStackClusterSpec{
			APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
				Enabled:           ptr.To(true),
				AvailabilityZones: []string{"az1"},
			},
		},
	}

	// Legacy name - not existing
	mockScopeFactory.LbClient.EXPECT().
		ListLoadBalancers(loadbalancers.ListOpts{Name: "k8s-clusterapi-cluster-test-kubeapi"}).
		Return([]loadbalancers.LoadBalancer{}, nil)

	// AZ name exists and is ACTIVE -> should call DeleteLoadBalancer(cascade)
	mockScopeFactory.LbClient.EXPECT().
		ListLoadBalancers(loadbalancers.ListOpts{Name: "k8s-clusterapi-cluster-test-az1-kubeapi"}).
		Return([]loadbalancers.LoadBalancer{
			{
				ID:                 "lb-az1",
				Name:               "k8s-clusterapi-cluster-test-az1-kubeapi",
				ProvisioningStatus: "ACTIVE",
				VipPortID:          "", // simplify, no FIP work
			},
		}, nil)

	mockScopeFactory.LbClient.EXPECT().DeleteLoadBalancer("lb-az1", gomock.Any()).Return(nil)

	result, err := lbs.DeleteLoadBalancer(cluster, "test")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).NotTo(BeNil()) // requeue to ensure cleanup completion
}
