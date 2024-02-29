/*
Copyright 2024 The Kubernetes Authors.

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

func ResolveDependentMachineResources(scope *scope.WithLogger, openStackMachine *infrav1.OpenStackMachine) error {
	if openStackMachine.Status.ReferencedResources == nil {
		return nil
	}

	networkingService, err := networking.NewService(scope)
	if err != nil {
		return err
	}

	return networkingService.AdoptMachinePorts(scope, openStackMachine, openStackMachine.Status.ReferencedResources.PortsOpts)
}

func ResolveDependentBastionResources(scope *scope.WithLogger, openStackCluster *infrav1.OpenStackCluster, bastionName string) error {
	if openStackCluster.Status.Bastion == nil {
		return nil
	}

	if openStackCluster.Status.Bastion.DependentResources == nil {
		return nil
	}

	networkingService, err := networking.NewService(scope)
	if err != nil {
		return err
	}

	return networkingService.AdoptBastionPorts(scope, openStackCluster, bastionName)
}
