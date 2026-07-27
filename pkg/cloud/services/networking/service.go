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
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

const (
	networkPrefix string = "k8s-clusterapi"
)

// Service interfaces with the OpenStack Networking API.
// It will create a network related infrastructure for the cluster, like network, subnet, router, security groups.
type Service struct {
	scope  *scope.WithLogger
	client clients.NetworkClient
}

// NewService returns an instance of the networking service.
func NewService(scope *scope.WithLogger) (*Service, error) {
	networkClient, err := scope.NewNetworkClient()
	if err != nil {
		return nil, err
	}

	return &Service{
		scope:  scope,
		client: networkClient,
	}, nil
}
