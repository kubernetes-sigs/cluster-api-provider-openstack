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

package v1beta1

import (
	"k8s.io/utils/pointer"
)

// Hub marks OpenStackCluster as a conversion hub.
func (*OpenStackCluster) Hub() {}

// Hub marks OpenStackClusterList as a conversion hub.
func (*OpenStackClusterList) Hub() {}

// Hub marks OpenStackClusterTemplate as a conversion hub.
func (*OpenStackClusterTemplate) Hub() {}

// Hub marks OpenStackClusterTemplateList as a conversion hub.
func (*OpenStackClusterTemplateList) Hub() {}

// Hub marks OpenStackMachine as a conversion hub.
func (*OpenStackMachine) Hub() {}

// Hub marks OpenStackMachineList as a conversion hub.
func (*OpenStackMachineList) Hub() {}

// Hub marks OpenStackMachineTemplate as a conversion hub.
func (*OpenStackMachineTemplate) Hub() {}

// Hub marks OpenStackMachineTemplateList as a conversion hub.
func (*OpenStackMachineTemplateList) Hub() {}

// LegacyCalicoSecurityGroupRules returns a list of security group rules for calico
// that need to be applied to the control plane and worker security groups when
// managed security groups are enabled and upgrading to v1beta1.
func LegacyCalicoSecurityGroupRules() []SecurityGroupRuleSpec {
	return []SecurityGroupRuleSpec{
		{
			Name:                "BGP (calico)",
			Description:         pointer.String("Created by cluster-api-provider-openstack API conversion - BGP (calico)"),
			Direction:           "ingress",
			EtherType:           pointer.String("IPv4"),
			PortRangeMin:        pointer.Int(179),
			PortRangeMax:        pointer.Int(179),
			Protocol:            pointer.String("tcp"),
			RemoteManagedGroups: []ManagedSecurityGroupName{"controlplane", "worker"},
		},
		{
			Name:                "IP-in-IP (calico)",
			Description:         pointer.String("Created by cluster-api-provider-openstack API conversion - IP-in-IP (calico)"),
			Direction:           "ingress",
			EtherType:           pointer.String("IPv4"),
			Protocol:            pointer.String("4"),
			RemoteManagedGroups: []ManagedSecurityGroupName{"controlplane", "worker"},
		},
	}
}
