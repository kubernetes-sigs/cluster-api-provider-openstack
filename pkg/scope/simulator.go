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
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha6"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/simulator"
)

type SimulatorScopeFactory struct {
	simulator *simulator.OpenStackSimulator
	projectID string
	logger    logr.Logger
}

func NewSimulatorScopeFactory(simulator *simulator.OpenStackSimulator, projectID string, logger logr.Logger) *SimulatorScopeFactory {
	return &SimulatorScopeFactory{
		simulator: simulator,
		projectID: projectID,
		logger:    logger,
	}
}

func (f *SimulatorScopeFactory) ProjectID() string {
	return f.projectID
}

func (f *SimulatorScopeFactory) Logger() logr.Logger {
	return f.logger
}

func (f *SimulatorScopeFactory) NewClientScopeFromMachine(ctx context.Context, ctrlClient client.Client, openStackMachine *infrav1.OpenStackMachine, logger logr.Logger) (Scope, error) {
	return f, nil
}

func (f *SimulatorScopeFactory) NewClientScopeFromCluster(ctx context.Context, ctrlClient client.Client, openStackCluster *infrav1.OpenStackCluster, logger logr.Logger) (Scope, error) {
	return f, nil
}

func (f *SimulatorScopeFactory) NewComputeClient() (clients.ComputeClient, error) {
	return f.simulator.NewComputeClient()
}

func (f *SimulatorScopeFactory) NewVolumeClient() (clients.VolumeClient, error) {
	return f.simulator.NewVolumeClient()
}

func (f *SimulatorScopeFactory) NewImageClient() (clients.ImageClient, error) {
	return f.simulator.NewImageClient()
}

func (f *SimulatorScopeFactory) NewNetworkClient() (clients.NetworkClient, error) {
	return f.simulator.NewNetworkClient()
}

func (f *SimulatorScopeFactory) NewLbClient() (clients.LbClient, error) {
	return f.simulator.NewLbClient()
}
