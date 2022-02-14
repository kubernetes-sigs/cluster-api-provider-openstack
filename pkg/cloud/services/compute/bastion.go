/*
Copyright 2020 The Kubernetes Authors.

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

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

func (s *Service) CreateBastion(openStackCluster *infrav1.OpenStackCluster, clusterName string) (*InstanceStatus, error) {
	name := fmt.Sprintf("%s-bastion", clusterName)
	instanceSpec := &InstanceSpec{
		Name:          name,
		Flavor:        openStackCluster.Spec.Bastion.Instance.Flavor,
		SSHKeyName:    openStackCluster.Spec.Bastion.Instance.SSHKeyName,
		Image:         openStackCluster.Spec.Bastion.Instance.Image,
		ImageUUID:     openStackCluster.Spec.Bastion.Instance.ImageUUID,
		FailureDomain: openStackCluster.Spec.Bastion.AvailabilityZone,
		RootVolume:    openStackCluster.Spec.Bastion.Instance.RootVolume,
	}

	instanceSpec.SecurityGroups = openStackCluster.Spec.Bastion.Instance.SecurityGroups
	if openStackCluster.Spec.ManagedSecurityGroups {
		instanceSpec.SecurityGroups = append(instanceSpec.SecurityGroups, infrav1.SecurityGroupParam{
			UUID: openStackCluster.Status.BastionSecurityGroup.ID,
		})
	}

	instanceSpec.Networks = openStackCluster.Spec.Bastion.Instance.Networks
	instanceSpec.Ports = openStackCluster.Spec.Bastion.Instance.Ports

	return s.createInstance(openStackCluster, openStackCluster, clusterName, instanceSpec, retryIntervalInstanceStatus)
}
