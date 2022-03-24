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
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/loadbalancers"
	. "github.com/onsi/gomega"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha5"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/loadbalancer/mock_loadbalancer"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking/mock_networking"
)

func Test_ReconcileLoadBalancer(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	openStackCluster := &infrav1.OpenStackCluster{Status: infrav1.OpenStackClusterStatus{
		ExternalNetwork: &infrav1.Network{
			ID: "aaaaaaaa-bbbb-cccc-dddd-111111111111",
		},
		Network: &infrav1.Network{
			Subnet: &infrav1.Subnet{
				ID: "aaaaaaaa-bbbb-cccc-dddd-222222222222",
			},
		},
	}}
	type serviceFields struct {
		projectID          string
		networkingClient   *mock_networking.MockNetworkClient
		loadbalancerClient *mock_loadbalancer.MockLbClient
	}
	lbtests := []struct {
		name               string
		fields             serviceFields
		prepareServiceMock func(sf *serviceFields)
		expectNetwork      func(m *mock_networking.MockNetworkClientMockRecorder)
		expectLoadBalancer func(m *mock_loadbalancer.MockLbClientMockRecorder)
		wantError          error
	}{
		{
			name: "reconcile loadbalancer in non active state should should cause error",
			prepareServiceMock: func(sf *serviceFields) {
				sf.networkingClient = mock_networking.NewMockNetworkClient(mockCtrl)
				sf.loadbalancerClient = mock_loadbalancer.NewMockLbClient(mockCtrl)
			},
			expectNetwork: func(m *mock_networking.MockNetworkClientMockRecorder) {
				// add network api call results here
			},
			expectLoadBalancer: func(m *mock_loadbalancer.MockLbClientMockRecorder) {
				// return existing loadbalancer in non-active state
				lbList := []loadbalancers.LoadBalancer{
					{
						ID:                 "aaaaaaaa-bbbb-cccc-dddd-333333333333",
						Name:               "k8s-clusterapi-cluster-AAAAA-kubeapi",
						ProvisioningStatus: "PENDING_CREATE",
					},
				}
				m.ListLoadBalancers(loadbalancers.ListOpts{Name: "k8s-clusterapi-cluster-AAAAA-kubeapi"}).Return(lbList, nil)
			},
			wantError: fmt.Errorf("load balancer %q is not in expected state %s, current state is %s", "aaaaaaaa-bbbb-cccc-dddd-333333333333", "ACTIVE", "PENDING_CREATE"),
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
			g.Expect(err).To(MatchError(tt.wantError))
		})
	}
}
