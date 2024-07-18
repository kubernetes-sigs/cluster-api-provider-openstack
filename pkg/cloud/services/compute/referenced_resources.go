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
	"fmt"
	"slices"

	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	infrav1alpha1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

// ResolveServerSpec is responsible for populating a ResolvedServerSpec from
// an OpenStackMachineSpec and any external dependencies. The result contains no
// external dependencies, and does not require any complex logic on creation.
// Note that we only set the fields in ResolvedServerSpec that are not set yet. This is ok because
// OpenStackServer is immutable, so we can't change the spec after the machine is created.
func ResolveServerSpec(scope *scope.WithLogger, openStackServer *infrav1alpha1.OpenStackServer) (changed bool, err error) {
	changed = false

	spec := &openStackServer.Spec
	resolved := openStackServer.Status.Resolved
	if resolved == nil {
		resolved = &infrav1alpha1.ResolvedServerSpec{}
		openStackServer.Status.Resolved = resolved
	}

	// If the server is bound to a cluster, we use the cluster name to generate the port description.
	var clusterName string
	if openStackServer.ObjectMeta.Labels[clusterv1.ClusterNameLabel] != "" {
		clusterName = openStackServer.ObjectMeta.Labels[clusterv1.ClusterNameLabel]
	}

	computeService, err := NewService(scope)
	if err != nil {
		return changed, err
	}

	networkingService, err := networking.NewService(scope)
	if err != nil {
		return changed, err
	}

	// ServerGroup is optional, so we only need to resolve it if it's set in the spec
	if spec.ServerGroup != nil && resolved.ServerGroupID == "" {
		serverGroupID, err := computeService.GetServerGroupID(spec.ServerGroup)
		if err != nil {
			return changed, err
		}
		resolved.ServerGroupID = serverGroupID
		changed = true
	}

	// Image is required, so we need to resolve it if it's not set
	if resolved.ImageID == "" {
		imageID, err := computeService.GetImageID(spec.Image)
		if err != nil {
			return changed, err
		}
		resolved.ImageID = imageID
		changed = true
	}

	specTrunk := ptr.Deref(spec.Trunk, false)

	// Network resources are required in order to get ports options.
	// Notes:
	// - clusterResourceName is not used in this context, so we pass an empty string. In the future,
	// we may want to remove that (it's only used for the port description) or allow a user to pass
	// a custom description.
	// - managedSecurityGroup is not used in this context, so we pass nil. The security groups are
	//   passed in the spec.SecurityGroups and spec.Ports.
	// - We run a safety check to ensure that the resolved.Ports has the same length as the spec.Ports.
	//   This is to ensure that we don't accidentally add ports to the resolved.Ports that are not in the spec.
	if len(resolved.Ports) == 0 {
		portsOpts, err := networkingService.ConstructPorts(spec.Ports, spec.SecurityGroups, specTrunk, clusterName, openStackServer.Name, nil, nil, spec.Tags)
		if err != nil {
			return changed, err
		}
		if portsOpts != nil && len(portsOpts) != len(spec.Ports) {
			return changed, fmt.Errorf("resolved.Ports has a different length than spec.Ports")
		}
		resolved.Ports = portsOpts
		changed = true
	}

	return changed, nil
}

// InstanceTags returns the tags that should be applied to an instance.
// The tags are a deduplicated combination of the tags specified in the
// OpenStackMachineSpec and the ones specified on the OpenStackCluster.
func InstanceTags(spec *infrav1.OpenStackMachineSpec, openStackCluster *infrav1.OpenStackCluster) []string {
	machineTags := slices.Concat(spec.Tags, openStackCluster.Spec.Tags)

	seen := make(map[string]struct{}, len(machineTags))
	unique := make([]string, 0, len(machineTags))
	for _, tag := range machineTags {
		if _, ok := seen[tag]; !ok {
			seen[tag] = struct{}{}
			unique = append(unique, tag)
		}
	}
	return slices.Clip(unique)
}
