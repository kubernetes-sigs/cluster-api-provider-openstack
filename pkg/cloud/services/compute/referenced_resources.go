/*
Copyright 2023 The Kubernetes Authors.

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
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

// ResolveReferencedMachineResources is responsible for populating ReferencedMachineResources with IDs of
// the resources referenced in the OpenStackMachineSpec by querying the OpenStack APIs. It'll return error
// if resources cannot be found or their filters are ambiguous.
// Note that we only set the fields in ReferencedMachineResources that are not set yet. This is ok because:
// - OpenStackMachine is immutable, so we can't change the spec after the machine is created.
// - the bastion is mutable, but we delete the bastion when the spec changes, so the bastion status will be empty.
func ResolveReferencedMachineResources(scope *scope.WithLogger, openStackCluster *infrav1.OpenStackCluster, spec *infrav1.OpenStackMachineSpec, resources *infrav1.ReferencedMachineResources) (*infrav1.ReferencedMachineResources, error) {
	computeService, err := NewService(scope)
	if err != nil {
		return resources, err
	}

	networkingService, err := networking.NewService(scope)
	if err != nil {
		return resources, err
	}

	// ServerGroup is optional, so we only need to resolve it if it's set in the spec and not in ReferencedMachineResources yet.
	if spec.ServerGroup != nil && resources.ServerGroupID == "" {
		serverGroupID, err := computeService.GetServerGroupID(spec.ServerGroup)
		if err != nil {
			return resources, err
		}
		resources.ServerGroupID = serverGroupID
	}

	// Image is required, so we need to resolve it if it's not set in ReferencedMachineResources yet.
	if resources.ImageID == "" {
		imageID, err := computeService.GetImageID(spec.Image)
		if err != nil {
			return resources, err
		}
		resources.ImageID = imageID
	}

	// Network resources are required in order to get ports options.
	if len(resources.PortsOpts) == 0 && openStackCluster.Status.Network != nil {
		// For now we put this here but realistically an OpenStack administrator could enable/disable trunk
		// support at any time, so we should probably check this on every reconcile.
		trunkSupported, err := networkingService.IsTrunkExtSupported()
		if err != nil {
			return resources, err
		}
		portsOpts, err := networkingService.ConstructPorts(openStackCluster, spec.Ports, spec.Trunk, trunkSupported)
		if err != nil {
			return resources, err
		}
		resources.PortsOpts = portsOpts
	}

	return resources, nil
}
