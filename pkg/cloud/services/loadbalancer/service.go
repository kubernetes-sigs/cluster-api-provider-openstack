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

package loadbalancer

import (
	"fmt"

	"github.com/go-logr/logr"

	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"

	"github.com/gophercloud/gophercloud"
)

const (
	networkPrefix   string = "k8s-clusterapi"
	kubeapiLBSuffix string = "kubeapi"
)

// Service interfaces with the OpenStack Neutron LBaaS v2 API.
type Service struct {
	loadbalancerClient *gophercloud.ServiceClient
	networkingClient   *gophercloud.ServiceClient
	logger             logr.Logger
}

// NewService returns an instance of the loadbalancer service
func NewService(client *gophercloud.ProviderClient, clientOpts *clientconfig.ClientOpts, logger logr.Logger, useOctavia bool) (*Service, error) {
	var err error
	var loadbalancerClient *gophercloud.ServiceClient
	if useOctavia {
		loadbalancerClient, err = openstack.NewLoadBalancerV2(client, gophercloud.EndpointOpts{
			Region: clientOpts.RegionName,
		})
	} else {
		loadbalancerClient, err = openstack.NewNetworkV2(client, gophercloud.EndpointOpts{
			Region: clientOpts.RegionName,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create loadbalancer service client: %v", err)
	}
	networkingClient, err := openstack.NewNetworkV2(client, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create networking service client: %v", err)
	}

	return &Service{
		loadbalancerClient: loadbalancerClient,
		networkingClient:   networkingClient,
		logger:             logger,
	}, nil
}
