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

package scope

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha6"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients"
)

// Factory instantiates a new ClientScope using credentials from either a cluster or a machine.
type Factory interface {
	NewClientScopeFromMachine(ctx context.Context, ctrlClient client.Client, openStackMachine *infrav1.OpenStackMachine, defaultCACert []byte, logger logr.Logger) (Scope, error)
	NewClientScopeFromCluster(ctx context.Context, ctrlClient client.Client, openStackCluster *infrav1.OpenStackCluster, defaultCACert []byte, logger logr.Logger) (Scope, error)
}

// Scope contains arguments common to most operations.
type Scope interface {
	NewComputeClient() (clients.ComputeClient, error)
	NewVolumeClient() (clients.VolumeClient, error)
	NewImageClient() (clients.ImageClient, error)
	NewNetworkClient() (clients.NetworkClient, error)
	NewLbClient() (clients.LbClient, error)
	Logger() logr.Logger
	ProjectID() string
}

type scope struct {
	providerClient     *gophercloud.ProviderClient
	providerClientOpts *clientconfig.ClientOpts
	projectID          string
	logger             logr.Logger
}

// NewTestScope returns a Scope with no initialization. It should only be used by tests.
func NewTestScope(projectID string, logger logr.Logger) Scope {
	providerClient := new(gophercloud.ProviderClient)
	clientOpts := new(clientconfig.ClientOpts)

	return &scope{
		providerClient:     providerClient,
		providerClientOpts: clientOpts,
		projectID:          projectID,
		logger:             logger,
	}
}

func NewScope(cloud clientconfig.Cloud, caCert []byte, logger logr.Logger) (Scope, error) {
	providerClient, clientOpts, projectID, err := NewProviderClient(cloud, caCert)
	if err != nil {
		return nil, err
	}

	return &scope{
		providerClient:     providerClient,
		providerClientOpts: clientOpts,
		projectID:          projectID,
		logger:             logger,
	}, nil
}

func (s *scope) Logger() logr.Logger {
	return s.logger
}

func (s *scope) ProjectID() string {
	return s.projectID
}

func (s *scope) NewComputeClient() (clients.ComputeClient, error) {
	return clients.NewComputeClient(s.providerClient, s.providerClientOpts)
}

func (s *scope) NewNetworkClient() (clients.NetworkClient, error) {
	return clients.NewNetworkClient(s.providerClient, s.providerClientOpts)
}

func (s *scope) NewVolumeClient() (clients.VolumeClient, error) {
	return clients.NewVolumeClient(s.providerClient, s.providerClientOpts)
}

func (s *scope) NewImageClient() (clients.ImageClient, error) {
	return clients.NewImageClient(s.providerClient, s.providerClientOpts)
}

func (s *scope) NewLbClient() (clients.LbClient, error) {
	return clients.NewLbClient(s.providerClient, s.providerClientOpts)
}
