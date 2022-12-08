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
	"github.com/golang/mock/gomock"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha6"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
)

// MockScopeFactory implements both the ScopeFactory and ClientScope interfaces. It can be used in place of the default ProviderScopeFactory
// when we want to use mocked service clients which do not attempt to connect to a running OpenStack cloud.
type MockScopeFactory struct {
	ComputeClient *mock.MockComputeClient
	NetworkClient *mock.MockNetworkClient
	VolumeClient  *mock.MockVolumeClient
	ImageClient   *mock.MockImageClient
	LbClient      *mock.MockLbClient

	clientScopeCreateError error

	scope
}

func NewMockScopeFactory(mockCtrl *gomock.Controller) *MockScopeFactory {
	computeClient := mock.NewMockComputeClient(mockCtrl)
	volumeClient := mock.NewMockVolumeClient(mockCtrl)
	imageClient := mock.NewMockImageClient(mockCtrl)
	networkClient := mock.NewMockNetworkClient(mockCtrl)
	lbClient := mock.NewMockLbClient(mockCtrl)

	return &MockScopeFactory{
		ComputeClient: computeClient,
		VolumeClient:  volumeClient,
		ImageClient:   imageClient,
		NetworkClient: networkClient,
		LbClient:      lbClient,
		scope: scope{
			projectID: "",
			logger:    logr.Discard(),
		},
	}
}

func (f *MockScopeFactory) SetClientScopeCreateError(err error) {
	f.clientScopeCreateError = err
}

func (f *MockScopeFactory) NewClientScopeFromMachine(ctx context.Context, ctrlClient client.Client, openStackMachine *infrav1.OpenStackMachine, logger logr.Logger) (Scope, error) {
	if f.clientScopeCreateError != nil {
		return nil, f.clientScopeCreateError
	}
	return f, nil
}

func (f *MockScopeFactory) NewClientScopeFromCluster(ctx context.Context, ctrlClient client.Client, openStackCluster *infrav1.OpenStackCluster, logger logr.Logger) (Scope, error) {
	if f.clientScopeCreateError != nil {
		return nil, f.clientScopeCreateError
	}
	return f, nil
}

func (f *MockScopeFactory) NewComputeClient() (clients.ComputeClient, error) {
	return f.ComputeClient, nil
}

func (f *MockScopeFactory) NewVolumeClient() (clients.VolumeClient, error) {
	return f.VolumeClient, nil
}

func (f *MockScopeFactory) NewImageClient() (clients.ImageClient, error) {
	return f.ImageClient, nil
}

func (f *MockScopeFactory) NewNetworkClient() (clients.NetworkClient, error) {
	return f.NetworkClient, nil
}

func (f *MockScopeFactory) NewLbClient() (clients.LbClient, error) {
	return f.LbClient, nil
}
