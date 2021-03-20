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

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"

	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/provider"
)

const (
	networkPrefix string = "k8s-clusterapi"
)

// Service interfaces with the OpenStack Networking API.
// It will create a network related infrastructure for the cluster, like network, subnet, router, security groups.
type Service struct {
	projectID string
	client    *gophercloud.ServiceClient
	logger    logr.Logger
}

// NewService returns an instance of the networking service.
func NewService(client *gophercloud.ProviderClient, clientOpts *clientconfig.ClientOpts, logger logr.Logger) (*Service, error) {
	serviceClient, err := openstack.NewNetworkV2(client, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create networking service client: %v", err)
	}

	if clientOpts.AuthInfo == nil {
		return nil, fmt.Errorf("failed to get project id: authInfo must be set: %v", err)
	}

	projectID := clientOpts.AuthInfo.ProjectID
	if projectID == "" && clientOpts.AuthInfo.ProjectName != "" {
		projectID, err = provider.GetProjectID(client, clientOpts.AuthInfo.ProjectName)
		if err != nil {
			return nil, fmt.Errorf("error retrieveing project id: %v", err)
		}
	}
	if projectID == "" {
		return nil, fmt.Errorf("failed to get project id")
	}

	return &Service{
		projectID: projectID,
		client:    serviceClient,
		logger:    logger,
	}, nil
}
