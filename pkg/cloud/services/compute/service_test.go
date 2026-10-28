/*
Copyright 2026 The Kubernetes Authors.

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
	"testing"

	"github.com/go-logr/logr"
	"go.uber.org/mock/gomock"

	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

// countingScope wraps a scope and counts how many times each client is created.
type countingScope struct {
	scope.Scope

	computeCreated int
	volumeCreated  int
	imageCreated   int
}

func (c *countingScope) NewComputeClient() (clients.ComputeClient, error) {
	c.computeCreated++
	return c.Scope.NewComputeClient()
}

func (c *countingScope) NewVolumeClient() (clients.VolumeClient, error) {
	c.volumeCreated++
	return c.Scope.NewVolumeClient()
}

func (c *countingScope) NewImageClient() (clients.ImageClient, error) {
	c.imageCreated++
	return c.Scope.NewImageClient()
}

func TestService_ClientGettersCacheClient(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	counting := &countingScope{Scope: scope.NewMockScopeFactory(mockCtrl, "project-id")}
	s := &Service{scope: scope.NewWithLogger(counting, logr.Discard())}

	for range 3 {
		_ = s.getComputeClient()
		_ = s.getVolumeClient()
		_ = s.getImageClient()
	}

	if counting.computeCreated != 1 {
		t.Errorf("compute client created %d times, want 1", counting.computeCreated)
	}
	if counting.volumeCreated != 1 {
		t.Errorf("volume client created %d times, want 1", counting.volumeCreated)
	}
	if counting.imageCreated != 1 {
		t.Errorf("image client created %d times, want 1", counting.imageCreated)
	}
}
