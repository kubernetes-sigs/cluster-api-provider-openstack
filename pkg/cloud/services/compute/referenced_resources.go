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
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha8"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

// ResolveReferencedMachineResources is responsible for populating ReferencedMachineResources with IDs of
// the resources referenced in the OpenStackMachineSpec by querying the OpenStack APIs. It'll return error
// if resources cannot be found or their filters are ambiguous.
func ResolveReferencedMachineResources(scope scope.Scope, spec *infrav1.OpenStackMachineSpec, resources *infrav1.ReferencedMachineResources) error {
	compute, err := NewService(scope)
	if err != nil {
		return err
	}

	if spec.ServerGroup != nil && resources.ServerGroupID == "" {
		serverGroupID, err := compute.GetServerGroupID(spec.ServerGroup)
		if err != nil {
			return err
		}
		resources.ServerGroupID = serverGroupID
	}

	return nil
}
