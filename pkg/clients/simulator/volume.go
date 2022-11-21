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
	"fmt"

	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
)

type VolumeSimulator struct{}

func (c *VolumeSimulator) ListVolumes(opts volumes.ListOptsBuilder) ([]volumes.Volume, error) {
	panic(fmt.Errorf("ListVolumes not implemented"))
}

func (c *VolumeSimulator) CreateVolume(opts volumes.CreateOptsBuilder) (*volumes.Volume, error) {
	panic(fmt.Errorf("CreateVolume not implemented"))
}

func (c *VolumeSimulator) DeleteVolume(volumeID string, opts volumes.DeleteOptsBuilder) error {
	panic(fmt.Errorf("DeleteVolume not implemented"))
}

func (c *VolumeSimulator) GetVolume(volumeID string) (*volumes.Volume, error) {
	panic(fmt.Errorf("GetVolume not implemented"))
}
