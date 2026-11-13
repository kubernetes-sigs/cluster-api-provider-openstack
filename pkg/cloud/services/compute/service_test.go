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

	"github.com/go-logr/logr/testr"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

// countingScope records how many times each client constructor is called.
type countingScope struct {
	scope.Scope

	compute, volume, image, network int
}

func (c *countingScope) NewComputeClient() (clients.ComputeClient, error) {
	c.compute++
	return c.Scope.NewComputeClient()
}

func (c *countingScope) NewVolumeClient() (clients.VolumeClient, error) {
	c.volume++
	return c.Scope.NewVolumeClient()
}

func (c *countingScope) NewImageClient() (clients.ImageClient, error) {
	c.image++
	return c.Scope.NewImageClient()
}

func (c *countingScope) NewNetworkClient() (clients.NetworkClient, error) {
	c.network++
	return c.Scope.NewNetworkClient()
}

func TestService_ClientGettersReuseClient(t *testing.T) {
	g := NewWithT(t)
	mockCtrl := gomock.NewController(t)

	counter := &countingScope{Scope: scope.NewMockScopeFactory(mockCtrl, "")}
	s, err := NewService(scope.NewWithLogger(counter, testr.New(t)))
	g.Expect(err).NotTo(HaveOccurred())

	for range 2 {
		s.getComputeClient()
		s.getVolumeClient()
		s.getImageClient()
		_, err := s.getNetworkingService()
		g.Expect(err).NotTo(HaveOccurred())
	}

	g.Expect(counter.compute).To(Equal(1), "compute client constructions")
	g.Expect(counter.volume).To(Equal(1), "volume client constructions")
	g.Expect(counter.image).To(Equal(1), "image client constructions")
	g.Expect(counter.network).To(Equal(1), "network client constructions")
}
