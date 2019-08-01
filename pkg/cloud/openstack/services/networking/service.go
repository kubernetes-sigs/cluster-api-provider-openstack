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
	"fmt"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"

	"github.com/gophercloud/gophercloud"
)

const (
	networkPrefix   string = "k8s"
	kubeapiLBSuffix string = "kubeapi"
)

// Service interfaces with the OpenStack Networking API.
// It will create a network related infrastructure for the cluster, like network, subnet, router, security groups.
type Service struct {
	client *gophercloud.ServiceClient
}

// NewService returns an instance of the networking service
func NewService(client *gophercloud.ProviderClient, clientOpts *clientconfig.ClientOpts) (*Service, error) {
	serviceClient, err := openstack.NewNetworkV2(client, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create networking service client: %v", err)
	}
	return &Service{
		client: serviceClient,
	}, nil
}
