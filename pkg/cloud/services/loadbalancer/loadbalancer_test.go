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
	"github.com/golang/mock/gomock"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/apiversions"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/listeners"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/monitors"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/pools"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/providers"
	. "github.com/onsi/gomega"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

const apiHostname = "api.test-cluster.test"

func Test_ReconcileLoadBalancer(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Stub the call to net.LookupHost
	lookupHost = func(host string) (addrs string, err error) {
		if net.ParseIP(host) != nil {
			return host, nil
		} else if host == apiHostname {
			ips := []string{"192.168.100.10"}
			return ips[0], nil
		}
		return "", errors.New("Unknown Host " + host)
	}

	openStackCluster := &infrav1.OpenStackCluster{
		Spec: infrav1.OpenStackClusterSpec{
			APIServerLoadBalancer: &infrav1.APIServerLoadBalancer{
				Enabled: true,
			},
			DisableAPIServerFloatingIP: true,
			ControlPlaneEndpoint: clusterv1.APIEndpoint{
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
		expectNetwork      func(m *mock.MockNetworkClientMockRecorder)
		expectLoadBalancer func(m *mock.MockLbClientMockRecorder)
		wantError          error
	}{
		{
			name: "reconcile loadbalancer in non active state should wait for active state",
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
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
			g := NewWithT(t)
			log := testr.New(t)

			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			lbs, err := NewService(scope.NewWithLogger(mockScopeFactory, log))
			g.Expect(err).NotTo(HaveOccurred())

			tt.expectNetwork(mockScopeFactory.NetworkClient.EXPECT())
			tt.expectLoadBalancer(mockScopeFactory.LbClient.EXPECT())
			_, err = lbs.ReconcileLoadBalancer(openStackCluster, "AAAAA", 0)
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
	lookupHost = func(host string) (addrs string, err error) {
		if net.ParseIP(host) != nil {
			return host, nil
		} else if host == apiHostname {
			ips := []string{"192.168.100.10"}
			return ips[0], nil
		}
		return "", errors.New("Unknown Host " + host)
	}
	tests := []struct {
		name             string
		openStackCluster *infrav1.OpenStackCluster
		want             string
		wantError        bool
	}{
		{
			name:             "empty cluster returns empty VIP",
			openStackCluster: &infrav1.OpenStackCluster{},
			want:             "",
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
			want:      "1.2.3.4",
			wantError: false,
		},
		{
			name: "API server VIP is API Server Fixed IP",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerFixedIP: "1.2.3.4",
				},
			},
			want:      "1.2.3.4",
			wantError: false,
		},
		{
			name: "API server VIP with valid control plane endpoint",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					DisableAPIServerFloatingIP: true,
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: apiHostname,
						Port: 6443,
					},
				},
			},
			want:      "192.168.100.10",
			wantError: false,
		},
		{
			name: "API server VIP with invalid control plane endpoint",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					DisableAPIServerFloatingIP: true,
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
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
			} else {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(got).To(Equal(tt.want))
			}
		})
	}
}

func Test_getAPIServerFloatingIP(t *testing.T) {
	// Stub the call to net.LookupHost
	lookupHost = func(host string) (addrs string, err error) {
		if net.ParseIP(host) != nil {
			return host, nil
		} else if host == apiHostname {
			ips := []string{"192.168.100.10"}
			return ips[0], nil
		}
		return "", errors.New("Unknown Host " + host)
	}
	tests := []struct {
		name             string
		openStackCluster *infrav1.OpenStackCluster
		want             string
		wantError        bool
	}{
		{
			name:             "empty cluster returns empty FIP",
			openStackCluster: &infrav1.OpenStackCluster{},
			want:             "",
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
			want:      "1.2.3.4",
			wantError: false,
		},
		{
			name: "API server FIP is API Server Floating IP",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					APIServerFloatingIP: "1.2.3.4",
				},
			},
			want:      "1.2.3.4",
			wantError: false,
		},
		{
			name: "API server FIP with valid control plane endpoint",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: apiHostname,
						Port: 6443,
					},
				},
			},
			want:      "192.168.100.10",
			wantError: false,
		},
		{
			name: "API server FIP with invalid control plane endpoint",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
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
				},
			},
			expectLoadBalancer: func(m *mock.MockLbClientMockRecorder) {
				m.ListLoadBalancers(gomock.Any()).Return([]loadbalancers.LoadBalancer{}, nil)
				m.ListLoadBalancerProviders().Return(octaviaProviders, nil)
				m.CreateLoadBalancer(gomock.Any()).Return(&loadbalancers.LoadBalancer{
					ID: "AAAAA",
				}, nil)
			},
			want: &loadbalancers.LoadBalancer{
				ID: "AAAAA",
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
