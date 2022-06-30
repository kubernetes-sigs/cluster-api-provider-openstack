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

package compute

import (
	"fmt"

	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

type Service struct {
	scope             *scope.Scope
	computeService    clients.ComputeClient
	networkingService *networking.Service
}

// NewService returns an instance of the compute service.
func NewService(scope *scope.Scope) (*Service, error) {
	computeService, err := clients.NewComputeClient(scope)
	if err != nil {
		return nil, err
	}

	if scope.ProviderClientOpts.AuthInfo == nil {
		return nil, fmt.Errorf("authInfo must be set")
	}

	networkingService, err := networking.NewService(scope)
	if err != nil {
		return nil, fmt.Errorf("failed to create networking service: %v", err)
	}

	return &Service{
		scope:             scope,
		computeService:    computeService,
		networkingService: networkingService,
	}, nil
}
