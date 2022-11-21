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

package simulator

import (
	"time"

	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients"
)

type OpenStackSimulator struct {
	Compute *ComputeSimulator
	Image   *ImageSimulator
	Lb      *LbSimulator
	Network *NetworkSimulator
	Volume  VolumeSimulator

	DefaultDelay func()
}

func NewOpenStackSimulator() *OpenStackSimulator {
	s := OpenStackSimulator{}
	s.Compute = NewComputeSimulator(&s)
	s.Image = NewImageSimulator(&s)
	s.Lb = NewLbSimulator(&s)
	s.Network = NewNetworkSimulator(&s)

	s.DefaultDelay = func() {
		time.Sleep(100 * time.Millisecond)
	}

	return &s
}

func (s *OpenStackSimulator) NewComputeClient() (clients.ComputeClient, error) {
	return s.Compute, nil
}

func (s *OpenStackSimulator) NewImageClient() (clients.ImageClient, error) {
	return s.Image, nil
}

func (s *OpenStackSimulator) NewLbClient() (clients.LbClient, error) {
	return s.Lb, nil
}

func (s *OpenStackSimulator) NewNetworkClient() (clients.NetworkClient, error) {
	return s.Network, nil
}

func (s *OpenStackSimulator) NewVolumeClient() (clients.VolumeClient, error) {
	return &s.Volume, nil
}
