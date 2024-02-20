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

package v1alpha7

import (
	apiconversion "k8s.io/apimachinery/pkg/conversion"
	"k8s.io/utils/pointer"
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/conversion"
)

var _ ctrlconversion.Convertible = &OpenStackCluster{}

func restorev1alpha7Bastion(previous **Bastion, dst **Bastion) {
	if *previous != nil && *dst != nil {
		restorev1alpha7MachineSpec(&(*previous).Instance, &(*dst).Instance)
	}
}

var v1alpha7OpenStackClusterRestorer = conversion.RestorerFor[*OpenStackCluster]{
	"bastion": conversion.HashedFieldRestorer(
		func(c *OpenStackCluster) **Bastion {
			return &c.Spec.Bastion
		},
		restorev1alpha7Bastion,
	),
	"spec": conversion.HashedFieldRestorer(
		func(c *OpenStackCluster) *OpenStackClusterSpec {
			return &c.Spec
		},
		restorev1alpha7ClusterSpec,

		// Filter out Bastion, which is restored separately
		conversion.HashedFilterField[*OpenStackCluster, OpenStackClusterSpec](
			func(s *OpenStackClusterSpec) *OpenStackClusterSpec {
				if s.Bastion != nil {
					f := *s
					f.Bastion = nil
					return &f
				}
				return s
			},
		),
	),
	"status": conversion.HashedFieldRestorer(
		func(c *OpenStackCluster) *OpenStackClusterStatus {
			return &c.Status
		},
		restorev1alpha7ClusterStatus,
	),
}

func restorev1alpha7SecurityGroup(previous *SecurityGroup, dst *SecurityGroup) {
	if previous == nil || dst == nil {
		return
	}

	for i, rule := range previous.Rules {
		dst.Rules[i].SecurityGroupID = rule.SecurityGroupID
	}
}

func restorev1alpha7ClusterStatus(previous *OpenStackClusterStatus, dst *OpenStackClusterStatus) {
	restorev1alpha7SecurityGroup(previous.ControlPlaneSecurityGroup, dst.ControlPlaneSecurityGroup)
	restorev1alpha7SecurityGroup(previous.WorkerSecurityGroup, dst.WorkerSecurityGroup)
	restorev1alpha7SecurityGroup(previous.BastionSecurityGroup, dst.BastionSecurityGroup)
}

func restorev1beta1SecurityGroupStatus(previous *infrav1.SecurityGroupStatus, dst *infrav1.SecurityGroupStatus) {
	if previous == nil || dst == nil {
		return
	}

	for i := range dst.Rules {
		dstRule := &dst.Rules[i]

		// Conversion from scalar to *scalar is lossy for zero values. We need to restore only nil values.
		if dstRule.Description != nil && *dstRule.Description == "" {
			dstRule.Description = previous.Rules[i].Description
		}
		if dstRule.EtherType != nil && *dstRule.EtherType == "" {
			dstRule.EtherType = previous.Rules[i].EtherType
		}
		if dstRule.PortRangeMin != nil && *dstRule.PortRangeMin == 0 {
			dstRule.PortRangeMin = previous.Rules[i].PortRangeMin
		}
		if dstRule.PortRangeMax != nil && *dstRule.PortRangeMax == 0 {
			dstRule.PortRangeMax = previous.Rules[i].PortRangeMax
		}
		if dstRule.Protocol != nil && *dstRule.Protocol == "" {
			dstRule.Protocol = previous.Rules[i].Protocol
		}
		if dstRule.RemoteGroupID != nil && *dstRule.RemoteGroupID == "" {
			dstRule.RemoteGroupID = previous.Rules[i].RemoteGroupID
		}
		if dstRule.RemoteIPPrefix != nil && *dstRule.RemoteIPPrefix == "" {
			dstRule.RemoteIPPrefix = previous.Rules[i].RemoteIPPrefix
		}
	}
}

func restorev1beta1ClusterStatus(previous *infrav1.OpenStackClusterStatus, dst *infrav1.OpenStackClusterStatus) {
	restorev1beta1SecurityGroupStatus(previous.ControlPlaneSecurityGroup, dst.ControlPlaneSecurityGroup)
	restorev1beta1SecurityGroupStatus(previous.WorkerSecurityGroup, dst.WorkerSecurityGroup)
	restorev1beta1SecurityGroupStatus(previous.BastionSecurityGroup, dst.BastionSecurityGroup)

	// ReferencedResources have no equivalent in v1alpha7
	if previous.Bastion != nil {
		dst.Bastion.ReferencedResources = previous.Bastion.ReferencedResources
	}

	if previous.Bastion != nil && previous.Bastion.DependentResources.PortsStatus != nil {
		dst.Bastion.DependentResources.PortsStatus = previous.Bastion.DependentResources.PortsStatus
	}
}

var v1beta1OpenStackClusterRestorer = conversion.RestorerFor[*infrav1.OpenStackCluster]{
	"bastion": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackCluster) **infrav1.Bastion {
			return &c.Spec.Bastion
		},
		restorev1beta1Bastion,
	),
	"spec": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackCluster) *infrav1.OpenStackClusterSpec {
			return &c.Spec
		},
		restorev1beta1ClusterSpec,

		// Filter out Bastion, which is restored separately
		conversion.HashedFilterField[*infrav1.OpenStackCluster, infrav1.OpenStackClusterSpec](
			func(s *infrav1.OpenStackClusterSpec) *infrav1.OpenStackClusterSpec {
				if s.Bastion != nil {
					f := *s
					f.Bastion = nil
					return &f
				}
				return s
			},
		),
	),

	"status": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackCluster) *infrav1.OpenStackClusterStatus {
			return &c.Status
		},
		restorev1beta1ClusterStatus,
	),
}

func restorev1alpha7SubnetFilter(previous *SubnetFilter, dst *SubnetFilter) {
	// The edge cases with multiple commas are too tricky in this direction,
	// so we just restore the whole thing.
	dst.Tags = previous.Tags
	dst.TagsAny = previous.TagsAny
	dst.NotTags = previous.NotTags
	dst.NotTagsAny = previous.NotTagsAny
}

func restorev1alpha7SecurityGroupFilter(previous *SecurityGroupFilter, dst *SecurityGroupFilter) {
	// The edge cases with multiple commas are too tricky in this direction,
	// so we just restore the whole thing.
	dst.Tags = previous.Tags
	dst.TagsAny = previous.TagsAny
	dst.NotTags = previous.NotTags
	dst.NotTagsAny = previous.NotTagsAny
}

func restorev1alpha7NetworkFilter(previous *NetworkFilter, dst *NetworkFilter) {
	// The edge cases with multiple commas are too tricky in this direction,
	// so we just restore the whole thing.
	dst.Tags = previous.Tags
	dst.TagsAny = previous.TagsAny
	dst.NotTags = previous.NotTags
	dst.NotTagsAny = previous.NotTagsAny
}

func restorev1alpha7RouterFilter(previous *RouterFilter, dst *RouterFilter) {
	// The edge cases with multiple commas are too tricky in this direction,
	// so we just restore the whole thing.
	dst.Tags = previous.Tags
	dst.TagsAny = previous.TagsAny
	dst.NotTags = previous.NotTags
	dst.NotTagsAny = previous.NotTagsAny
}

func restorev1alpha7Port(previous *PortOpts, dst *PortOpts) {
	if len(dst.SecurityGroupFilters) == len(previous.SecurityGroupFilters) {
		for i := range dst.SecurityGroupFilters {
			restorev1alpha7SecurityGroupFilter(&previous.SecurityGroupFilters[i], &dst.SecurityGroupFilters[i])
		}
	}

	if dst.Network != nil && previous.Network != nil {
		restorev1alpha7NetworkFilter(previous.Network, dst.Network)
	}

	if len(dst.FixedIPs) == len(previous.FixedIPs) {
		for i := range dst.FixedIPs {
			prevFixedIP := &previous.FixedIPs[i]
			dstFixedIP := &dst.FixedIPs[i]

			if dstFixedIP.Subnet != nil && prevFixedIP.Subnet != nil {
				restorev1alpha7SubnetFilter(prevFixedIP.Subnet, dstFixedIP.Subnet)
			}
		}
	}
}

func restorev1alpha7MachineSpec(previous *OpenStackMachineSpec, dst *OpenStackMachineSpec) {
	dst.FloatingIP = previous.FloatingIP

	// Conversion to v1beta1 truncates keys and values to 255 characters
	for k, v := range previous.ServerMetadata {
		kd := k
		if len(k) > 255 {
			kd = k[:255]
		}

		vd := v
		if len(v) > 255 {
			vd = v[:255]
		}

		if kd != k || vd != v {
			delete(dst.ServerMetadata, kd)
			dst.ServerMetadata[k] = v
		}
	}

	// Conversion to v1beta1 removes the Kind field
	dst.IdentityRef = previous.IdentityRef

	if len(dst.Ports) == len(previous.Ports) {
		for i := range dst.Ports {
			restorev1alpha7Port(&previous.Ports[i], &dst.Ports[i])
		}
	}

	if len(dst.SecurityGroups) == len(previous.SecurityGroups) {
		for i := range dst.SecurityGroups {
			restorev1alpha7SecurityGroupFilter(&previous.SecurityGroups[i], &dst.SecurityGroups[i])
		}
	}
}

func restorev1beta1MachineSpec(previous *infrav1.OpenStackMachineSpec, dst *infrav1.OpenStackMachineSpec) {
	dst.ServerGroup = previous.ServerGroup
	dst.Image = previous.Image

	if len(dst.Ports) == len(previous.Ports) {
		for i := range dst.Ports {
			restorev1beta1Port(&previous.Ports[i], &dst.Ports[i])
		}
	}
}

func restorev1beta1Port(previous *infrav1.PortOpts, dst *infrav1.PortOpts) {
	if dst.NameSuffix == nil || *dst.NameSuffix == "" {
		dst.NameSuffix = previous.NameSuffix
	}

	if dst.Description == nil || *dst.Description == "" {
		dst.Description = previous.Description
	}

	if dst.MACAddress == nil || *dst.MACAddress == "" {
		dst.MACAddress = previous.MACAddress
	}

	if len(dst.FixedIPs) == len(previous.FixedIPs) {
		for j := range dst.FixedIPs {
			prevFixedIP := &previous.FixedIPs[j]
			dstFixedIP := &dst.FixedIPs[j]

			if dstFixedIP.IPAddress == nil || *dstFixedIP.IPAddress == "" {
				dstFixedIP.IPAddress = prevFixedIP.IPAddress
			}
		}
	}

	if len(dst.AllowedAddressPairs) == len(previous.AllowedAddressPairs) {
		for j := range dst.AllowedAddressPairs {
			prevAAP := &previous.AllowedAddressPairs[j]
			dstAAP := &dst.AllowedAddressPairs[j]

			if dstAAP.MACAddress == nil || *dstAAP.MACAddress == "" {
				dstAAP.MACAddress = prevAAP.MACAddress
			}
		}
	}

	if dst.HostID == nil || *dst.HostID == "" {
		dst.HostID = previous.HostID
	}

	if dst.VNICType == nil || *dst.VNICType == "" {
		dst.VNICType = previous.VNICType
	}

	if dst.Profile == nil && previous.Profile != nil {
		dst.Profile = &infrav1.BindingProfile{}
	}

	if dst.Profile != nil && previous.Profile != nil {
		dstProfile := dst.Profile
		prevProfile := previous.Profile

		if dstProfile.OVSHWOffload == nil || !*dstProfile.OVSHWOffload {
			dstProfile.OVSHWOffload = prevProfile.OVSHWOffload
		}

		if dstProfile.TrustedVF == nil || !*dstProfile.TrustedVF {
			dstProfile.TrustedVF = prevProfile.TrustedVF
		}
	}
}

func restorev1beta1Bastion(previous **infrav1.Bastion, dst **infrav1.Bastion) {
	if *previous == nil && *dst == nil {
		return
	}

	restorev1beta1MachineSpec(&(*previous).Instance, &(*dst).Instance)
	if (*dst).AvailabilityZone == nil || *(*dst).AvailabilityZone == "" {
		(*dst).AvailabilityZone = (*previous).AvailabilityZone
	}
	if (*dst).FloatingIP == nil || *(*dst).FloatingIP == "" {
		(*dst).FloatingIP = (*previous).FloatingIP
	}
}

func restorev1alpha7ClusterSpec(previous *OpenStackClusterSpec, dst *OpenStackClusterSpec) {
	prevBastion := previous.Bastion
	dstBastion := dst.Bastion
	if prevBastion != nil && dstBastion != nil {
		restorev1alpha7MachineSpec(&prevBastion.Instance, &dstBastion.Instance)
	}

	// We only restore DNSNameservers when these were lossly converted when NodeCIDR is empty.
	if len(previous.DNSNameservers) > 0 && dst.NodeCIDR == "" {
		dst.DNSNameservers = previous.DNSNameservers
	}

	// To avoid lossy conversion, we need to restore AllowAllInClusterTraffic
	// even if ManagedSecurityGroups is set to false
	if previous.AllowAllInClusterTraffic && !previous.ManagedSecurityGroups {
		dst.AllowAllInClusterTraffic = true
	}

	// Conversion to v1beta1 removes the Kind field
	dst.IdentityRef = previous.IdentityRef

	if len(dst.ExternalRouterIPs) == len(previous.ExternalRouterIPs) {
		for i := range dst.ExternalRouterIPs {
			restorev1alpha7SubnetFilter(&previous.ExternalRouterIPs[i].Subnet, &dst.ExternalRouterIPs[i].Subnet)
		}
	}

	restorev1alpha7SubnetFilter(&previous.Subnet, &dst.Subnet)

	if dst.Router != nil && previous.Router != nil {
		restorev1alpha7RouterFilter(previous.Router, dst.Router)
	}

	restorev1alpha7NetworkFilter(&previous.Network, &dst.Network)
}

func restorev1beta1ClusterSpec(previous *infrav1.OpenStackClusterSpec, dst *infrav1.OpenStackClusterSpec) {
	prevBastion := previous.Bastion
	dstBastion := dst.Bastion
	if prevBastion != nil && dstBastion != nil {
		restorev1beta1MachineSpec(&prevBastion.Instance, &dstBastion.Instance)
	}

	// Restore all fields except ID, which should have been copied over in conversion
	if previous.ExternalNetwork != nil {
		if dst.ExternalNetwork == nil {
			dst.ExternalNetwork = &infrav1.NetworkFilter{}
		}

		dst.ExternalNetwork.Name = previous.ExternalNetwork.Name
		dst.ExternalNetwork.Description = previous.ExternalNetwork.Description
		dst.ExternalNetwork.ProjectID = previous.ExternalNetwork.ProjectID
		dst.ExternalNetwork.Tags = previous.ExternalNetwork.Tags
		dst.ExternalNetwork.TagsAny = previous.ExternalNetwork.TagsAny
		dst.ExternalNetwork.NotTags = previous.ExternalNetwork.NotTags
		dst.ExternalNetwork.NotTagsAny = previous.ExternalNetwork.NotTagsAny
	}

	dst.DisableExternalNetwork = previous.DisableExternalNetwork

	if len(previous.Subnets) > 1 {
		dst.Subnets = append(dst.Subnets, previous.Subnets[1:]...)
	}

	dst.ManagedSubnets = previous.ManagedSubnets

	if previous.ManagedSecurityGroups != nil {
		dst.ManagedSecurityGroups.AllNodesSecurityGroupRules = previous.ManagedSecurityGroups.AllNodesSecurityGroupRules
	}

	if previous.APIServerLoadBalancer == nil || previous.APIServerLoadBalancer.IsZero() {
		dst.APIServerLoadBalancer = previous.APIServerLoadBalancer
	}

	if dst.APIServerFloatingIP == nil || *dst.APIServerFloatingIP == "" {
		dst.APIServerFloatingIP = previous.APIServerFloatingIP
	}
	if dst.APIServerFixedIP == nil || *dst.APIServerFixedIP == "" {
		dst.APIServerFixedIP = previous.APIServerFixedIP
	}

	if previous.APIServerPort != nil && *previous.APIServerPort == 0 {
		dst.APIServerPort = pointer.Int(0)
	}
}

func (r *OpenStackCluster) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackCluster)

	return conversion.ConvertAndRestore(
		r, dst,
		Convert_v1alpha7_OpenStackCluster_To_v1beta1_OpenStackCluster, Convert_v1beta1_OpenStackCluster_To_v1alpha7_OpenStackCluster,
		v1alpha7OpenStackClusterRestorer, v1beta1OpenStackClusterRestorer,
	)
}

func (r *OpenStackCluster) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackCluster)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1beta1_OpenStackCluster_To_v1alpha7_OpenStackCluster, Convert_v1alpha7_OpenStackCluster_To_v1beta1_OpenStackCluster,
		v1beta1OpenStackClusterRestorer, v1alpha7OpenStackClusterRestorer,
	)
}

var _ ctrlconversion.Convertible = &OpenStackClusterList{}

func (r *OpenStackClusterList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1alpha7_OpenStackClusterList_To_v1beta1_OpenStackClusterList(r, dst, nil)
}

func (r *OpenStackClusterList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1beta1_OpenStackClusterList_To_v1alpha7_OpenStackClusterList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackClusterTemplate{}

func restorev1alpha7ClusterTemplateSpec(previous *OpenStackClusterTemplateSpec, dst *OpenStackClusterTemplateSpec) {
	restorev1alpha7ClusterSpec(&previous.Template.Spec, &dst.Template.Spec)
	restorev1alpha7Bastion(&previous.Template.Spec.Bastion, &dst.Template.Spec.Bastion)
}

func restorev1beta1ClusterTemplateSpec(previous *infrav1.OpenStackClusterTemplateSpec, dst *infrav1.OpenStackClusterTemplateSpec) {
	restorev1beta1Bastion(&previous.Template.Spec.Bastion, &dst.Template.Spec.Bastion)
	restorev1beta1ClusterSpec(&previous.Template.Spec, &dst.Template.Spec)
}

var v1alpha7OpenStackClusterTemplateRestorer = conversion.RestorerFor[*OpenStackClusterTemplate]{
	"spec": conversion.HashedFieldRestorer(
		func(c *OpenStackClusterTemplate) *OpenStackClusterTemplateSpec {
			return &c.Spec
		},
		restorev1alpha7ClusterTemplateSpec,
	),
}

var v1beta1OpenStackClusterTemplateRestorer = conversion.RestorerFor[*infrav1.OpenStackClusterTemplate]{
	"spec": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackClusterTemplate) *infrav1.OpenStackClusterTemplateSpec {
			return &c.Spec
		},
		restorev1beta1ClusterTemplateSpec,
	),
}

func (r *OpenStackClusterTemplate) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterTemplate)

	return conversion.ConvertAndRestore(
		r, dst,
		Convert_v1alpha7_OpenStackClusterTemplate_To_v1beta1_OpenStackClusterTemplate, Convert_v1beta1_OpenStackClusterTemplate_To_v1alpha7_OpenStackClusterTemplate,
		v1alpha7OpenStackClusterTemplateRestorer, v1beta1OpenStackClusterTemplateRestorer,
	)
}

func (r *OpenStackClusterTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterTemplate)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1beta1_OpenStackClusterTemplate_To_v1alpha7_OpenStackClusterTemplate, Convert_v1alpha7_OpenStackClusterTemplate_To_v1beta1_OpenStackClusterTemplate,
		v1beta1OpenStackClusterTemplateRestorer, v1alpha7OpenStackClusterTemplateRestorer,
	)
}

var _ ctrlconversion.Convertible = &OpenStackMachine{}

var v1alpha7OpenStackMachineRestorer = conversion.RestorerFor[*OpenStackMachine]{
	"spec": conversion.HashedFieldRestorer(
		func(c *OpenStackMachine) *OpenStackMachineSpec {
			return &c.Spec
		},
		restorev1alpha7MachineSpec,
	),
}

var v1beta1OpenStackMachineRestorer = conversion.RestorerFor[*infrav1.OpenStackMachine]{
	"spec": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackMachine) *infrav1.OpenStackMachineSpec {
			return &c.Spec
		},
		restorev1beta1MachineSpec,
	),
	"depresources": conversion.UnconditionalFieldRestorer(
		func(c *infrav1.OpenStackMachine) *infrav1.DependentMachineResources {
			return &c.Status.DependentResources
		},
	),

	// No equivalent in v1alpha7
	"refresources": conversion.UnconditionalFieldRestorer(
		func(c *infrav1.OpenStackMachine) *infrav1.ReferencedMachineResources {
			return &c.Status.ReferencedResources
		},
	),
}

func (r *OpenStackMachine) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachine)

	return conversion.ConvertAndRestore(
		r, dst,
		Convert_v1alpha7_OpenStackMachine_To_v1beta1_OpenStackMachine, Convert_v1beta1_OpenStackMachine_To_v1alpha7_OpenStackMachine,
		v1alpha7OpenStackMachineRestorer, v1beta1OpenStackMachineRestorer,
	)
}

func (r *OpenStackMachine) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachine)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1beta1_OpenStackMachine_To_v1alpha7_OpenStackMachine, Convert_v1alpha7_OpenStackMachine_To_v1beta1_OpenStackMachine,
		v1beta1OpenStackMachineRestorer, v1alpha7OpenStackMachineRestorer,
	)
}

var _ ctrlconversion.Convertible = &OpenStackMachineList{}

func (r *OpenStackMachineList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineList)
	return Convert_v1alpha7_OpenStackMachineList_To_v1beta1_OpenStackMachineList(r, dst, nil)
}

func (r *OpenStackMachineList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineList)
	return Convert_v1beta1_OpenStackMachineList_To_v1alpha7_OpenStackMachineList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplate{}

func restorev1alpha7MachineTemplateSpec(previous *OpenStackMachineTemplateSpec, dst *OpenStackMachineTemplateSpec) {
	restorev1alpha7MachineSpec(&previous.Template.Spec, &dst.Template.Spec)
}

var v1alpha7OpenStackMachineTemplateRestorer = conversion.RestorerFor[*OpenStackMachineTemplate]{
	"spec": conversion.HashedFieldRestorer(
		func(c *OpenStackMachineTemplate) *OpenStackMachineTemplateSpec {
			return &c.Spec
		},
		restorev1alpha7MachineTemplateSpec,
	),
}

var v1beta1OpenStackMachineTemplateRestorer = conversion.RestorerFor[*infrav1.OpenStackMachineTemplate]{
	"spec": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackMachineTemplate) *infrav1.OpenStackMachineSpec {
			return &c.Spec.Template.Spec
		},
		restorev1beta1MachineSpec,
	),
}

func (r *OpenStackMachineTemplate) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineTemplate)

	return conversion.ConvertAndRestore(
		r, dst,
		Convert_v1alpha7_OpenStackMachineTemplate_To_v1beta1_OpenStackMachineTemplate, Convert_v1beta1_OpenStackMachineTemplate_To_v1alpha7_OpenStackMachineTemplate,
		v1alpha7OpenStackMachineTemplateRestorer, v1beta1OpenStackMachineTemplateRestorer,
	)
}

func (r *OpenStackMachineTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplate)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1beta1_OpenStackMachineTemplate_To_v1alpha7_OpenStackMachineTemplate, Convert_v1alpha7_OpenStackMachineTemplate_To_v1beta1_OpenStackMachineTemplate,
		v1beta1OpenStackMachineTemplateRestorer, v1alpha7OpenStackMachineTemplateRestorer,
	)
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplateList{}

func (r *OpenStackMachineTemplateList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineTemplateList)
	return Convert_v1alpha7_OpenStackMachineTemplateList_To_v1beta1_OpenStackMachineTemplateList(r, dst, nil)
}

func (r *OpenStackMachineTemplateList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplateList)
	return Convert_v1beta1_OpenStackMachineTemplateList_To_v1alpha7_OpenStackMachineTemplateList(src, r, nil)
}

func Convert_v1beta1_OpenStackMachineSpec_To_v1alpha7_OpenStackMachineSpec(in *infrav1.OpenStackMachineSpec, out *OpenStackMachineSpec, s apiconversion.Scope) error {
	err := autoConvert_v1beta1_OpenStackMachineSpec_To_v1alpha7_OpenStackMachineSpec(in, out, s)
	if err != nil {
		return err
	}

	if in.ServerGroup != nil {
		out.ServerGroupID = in.ServerGroup.ID
	}

	if in.Image.Name != "" {
		out.Image = in.Image.Name
	}

	if in.Image.ID != "" {
		out.ImageUUID = in.Image.ID
	}

	if len(in.ServerMetadata) > 0 {
		serverMetadata := make(map[string]string, len(in.ServerMetadata))
		for i := range in.ServerMetadata {
			key := in.ServerMetadata[i].Key
			value := in.ServerMetadata[i].Value
			serverMetadata[key] = value
		}
		out.ServerMetadata = serverMetadata
	}

	if in.IdentityRef != nil {
		out.CloudName = in.IdentityRef.CloudName
	}

	return nil
}

func Convert_v1alpha7_OpenStackMachineSpec_To_v1beta1_OpenStackMachineSpec(in *OpenStackMachineSpec, out *infrav1.OpenStackMachineSpec, s apiconversion.Scope) error {
	err := autoConvert_v1alpha7_OpenStackMachineSpec_To_v1beta1_OpenStackMachineSpec(in, out, s)
	if err != nil {
		return err
	}

	if in.ServerGroupID != "" {
		out.ServerGroup = &infrav1.ServerGroupFilter{ID: in.ServerGroupID}
	} else {
		out.ServerGroup = nil
	}

	imageFilter := infrav1.ImageFilter{}
	if in.Image != "" {
		imageFilter.Name = in.Image
	}
	if in.ImageUUID != "" {
		imageFilter.ID = in.ImageUUID
	}
	out.Image = imageFilter

	if len(in.ServerMetadata) > 0 {
		serverMetadata := make([]infrav1.ServerMetadata, 0, len(in.ServerMetadata))
		for k, v := range in.ServerMetadata {
			// Truncate key and value to 255 characters if required, as this
			// was not validated prior to v1beta1
			if len(k) > 255 {
				k = k[:255]
			}
			if len(v) > 255 {
				v = v[:255]
			}

			serverMetadata = append(serverMetadata, infrav1.ServerMetadata{Key: k, Value: v})
		}
		out.ServerMetadata = serverMetadata
	}

	if in.CloudName != "" {
		if out.IdentityRef == nil {
			out.IdentityRef = &infrav1.OpenStackIdentityReference{}
		}
		out.IdentityRef.CloudName = in.CloudName
	}

	return nil
}

func Convert_v1beta1_OpenStackMachineStatus_To_v1alpha7_OpenStackMachineStatus(in *infrav1.OpenStackMachineStatus, out *OpenStackMachineStatus, s apiconversion.Scope) error {
	// ReferencedResources have no equivalent in v1alpha7
	return autoConvert_v1beta1_OpenStackMachineStatus_To_v1alpha7_OpenStackMachineStatus(in, out, s)
}

func Convert_v1beta1_BastionStatus_To_v1alpha7_BastionStatus(in *infrav1.BastionStatus, out *BastionStatus, s apiconversion.Scope) error {
	// ReferencedResources have no equivalent in v1alpha7
	return autoConvert_v1beta1_BastionStatus_To_v1alpha7_BastionStatus(in, out, s)
}

func Convert_v1alpha7_Bastion_To_v1beta1_Bastion(in *Bastion, out *infrav1.Bastion, s apiconversion.Scope) error {
	err := autoConvert_v1alpha7_Bastion_To_v1beta1_Bastion(in, out, s)
	if err != nil {
		return err
	}

	if in.Instance.ServerGroupID != "" {
		out.Instance.ServerGroup = &infrav1.ServerGroupFilter{ID: in.Instance.ServerGroupID}
	} else {
		out.Instance.ServerGroup = nil
	}

	if in.Instance.FloatingIP != "" {
		out.FloatingIP = pointer.String(in.Instance.FloatingIP)
	}
	return nil
}

func Convert_v1beta1_Bastion_To_v1alpha7_Bastion(in *infrav1.Bastion, out *Bastion, s apiconversion.Scope) error {
	err := autoConvert_v1beta1_Bastion_To_v1alpha7_Bastion(in, out, s)
	if err != nil {
		return err
	}

	if in.Instance.ServerGroup != nil && in.Instance.ServerGroup.ID != "" {
		out.Instance.ServerGroupID = in.Instance.ServerGroup.ID
	}

	if in.FloatingIP != nil {
		out.Instance.FloatingIP = *in.FloatingIP
	}
	return nil
}

func Convert_v1alpha7_OpenStackClusterSpec_To_v1beta1_OpenStackClusterSpec(in *OpenStackClusterSpec, out *infrav1.OpenStackClusterSpec, s apiconversion.Scope) error {
	err := autoConvert_v1alpha7_OpenStackClusterSpec_To_v1beta1_OpenStackClusterSpec(in, out, s)
	if err != nil {
		return err
	}

	if in.Network != (NetworkFilter{}) {
		out.Network = &infrav1.NetworkFilter{}
		if err := Convert_v1alpha7_NetworkFilter_To_v1beta1_NetworkFilter(&in.Network, out.Network, s); err != nil {
			return err
		}
	}

	if in.ExternalNetworkID != "" {
		out.ExternalNetwork = &infrav1.NetworkFilter{
			ID: in.ExternalNetworkID,
		}
	}

	emptySubnet := SubnetFilter{}
	if in.Subnet != emptySubnet {
		subnet := infrav1.SubnetFilter{}
		if err := Convert_v1alpha7_SubnetFilter_To_v1beta1_SubnetFilter(&in.Subnet, &subnet, s); err != nil {
			return err
		}
		out.Subnets = []infrav1.SubnetFilter{subnet}
	}

	// DNSNameservers without NodeCIDR doesn't make sense, so we drop that.
	if len(in.NodeCIDR) > 0 {
		out.ManagedSubnets = []infrav1.SubnetSpec{
			{
				CIDR:           in.NodeCIDR,
				DNSNameservers: in.DNSNameservers,
			},
		}
	}

	if in.ManagedSecurityGroups {
		out.ManagedSecurityGroups = &infrav1.ManagedSecurityGroups{}
		if !in.AllowAllInClusterTraffic {
			out.ManagedSecurityGroups.AllNodesSecurityGroupRules = infrav1.LegacyCalicoSecurityGroupRules()
		} else {
			out.ManagedSecurityGroups.AllowAllInClusterTraffic = true
		}
	}

	apiServerLoadBalancer := &infrav1.APIServerLoadBalancer{}
	if err := Convert_v1alpha7_APIServerLoadBalancer_To_v1beta1_APIServerLoadBalancer(&in.APIServerLoadBalancer, apiServerLoadBalancer, s); err != nil {
		return err
	}
	if !apiServerLoadBalancer.IsZero() {
		out.APIServerLoadBalancer = apiServerLoadBalancer
	}

	out.IdentityRef.CloudName = in.CloudName
	if in.IdentityRef != nil {
		out.IdentityRef.Name = in.IdentityRef.Name
	}

	// The generated conversion function converts "" to &"" which is not what we want
	if in.APIServerFloatingIP == "" {
		out.APIServerFloatingIP = nil
	}
	if in.APIServerFixedIP == "" {
		out.APIServerFixedIP = nil
	}

	if in.APIServerPort != 0 {
		out.APIServerPort = pointer.Int(in.APIServerPort)
	}

	return nil
}

func Convert_v1beta1_OpenStackClusterSpec_To_v1alpha7_OpenStackClusterSpec(in *infrav1.OpenStackClusterSpec, out *OpenStackClusterSpec, s apiconversion.Scope) error {
	err := autoConvert_v1beta1_OpenStackClusterSpec_To_v1alpha7_OpenStackClusterSpec(in, out, s)
	if err != nil {
		return err
	}

	if in.Network != nil {
		if err := Convert_v1beta1_NetworkFilter_To_v1alpha7_NetworkFilter(in.Network, &out.Network, s); err != nil {
			return err
		}
	}

	if in.ExternalNetwork != nil && in.ExternalNetwork.ID != "" {
		out.ExternalNetworkID = in.ExternalNetwork.ID
	}

	if len(in.Subnets) >= 1 {
		if err := Convert_v1beta1_SubnetFilter_To_v1alpha7_SubnetFilter(&in.Subnets[0], &out.Subnet, s); err != nil {
			return err
		}
	}

	if len(in.ManagedSubnets) > 0 {
		out.NodeCIDR = in.ManagedSubnets[0].CIDR
		out.DNSNameservers = in.ManagedSubnets[0].DNSNameservers
	}

	if in.ManagedSecurityGroups != nil {
		out.ManagedSecurityGroups = true
		out.AllowAllInClusterTraffic = in.ManagedSecurityGroups.AllowAllInClusterTraffic
	}

	if in.APIServerLoadBalancer != nil {
		if err := Convert_v1beta1_APIServerLoadBalancer_To_v1alpha7_APIServerLoadBalancer(in.APIServerLoadBalancer, &out.APIServerLoadBalancer, s); err != nil {
			return err
		}
	}

	out.CloudName = in.IdentityRef.CloudName
	out.IdentityRef = &OpenStackIdentityReference{Name: in.IdentityRef.Name}

	if in.APIServerPort != nil {
		out.APIServerPort = *in.APIServerPort
	}

	return nil
}

func Convert_v1beta1_SecurityGroupStatus_To_v1alpha7_SecurityGroup(in *infrav1.SecurityGroupStatus, out *SecurityGroup, s apiconversion.Scope) error { //nolint:revive
	out.ID = in.ID
	out.Name = in.Name
	out.Rules = make([]SecurityGroupRule, len(in.Rules))
	for i, rule := range in.Rules {
		out.Rules[i] = SecurityGroupRule{
			ID:        rule.ID,
			Direction: rule.Direction,
		}
		if rule.Description != nil {
			out.Rules[i].Description = *rule.Description
		}
		if rule.EtherType != nil {
			out.Rules[i].EtherType = *rule.EtherType
		}
		if rule.PortRangeMin != nil {
			out.Rules[i].PortRangeMin = *rule.PortRangeMin
		}
		if rule.PortRangeMax != nil {
			out.Rules[i].PortRangeMax = *rule.PortRangeMax
		}
		if rule.Protocol != nil {
			out.Rules[i].Protocol = *rule.Protocol
		}
		if rule.RemoteGroupID != nil {
			out.Rules[i].RemoteGroupID = *rule.RemoteGroupID
		}
		if rule.RemoteIPPrefix != nil {
			out.Rules[i].RemoteIPPrefix = *rule.RemoteIPPrefix
		}
	}
	return nil
}

func Convert_v1alpha7_SecurityGroup_To_v1beta1_SecurityGroupStatus(in *SecurityGroup, out *infrav1.SecurityGroupStatus, s apiconversion.Scope) error { //nolint:revive
	out.ID = in.ID
	out.Name = in.Name
	out.Rules = make([]infrav1.SecurityGroupRuleStatus, len(in.Rules))
	for i, rule := range in.Rules {
		out.Rules[i] = infrav1.SecurityGroupRuleStatus{
			ID:             rule.ID,
			Description:    pointer.String(rule.Description),
			Direction:      rule.Direction,
			EtherType:      pointer.String(rule.EtherType),
			PortRangeMin:   pointer.Int(rule.PortRangeMin),
			PortRangeMax:   pointer.Int(rule.PortRangeMax),
			Protocol:       pointer.String(rule.Protocol),
			RemoteGroupID:  pointer.String(rule.RemoteGroupID),
			RemoteIPPrefix: pointer.String(rule.RemoteIPPrefix),
		}
	}

	return nil
}

func Convert_v1alpha7_OpenStackIdentityReference_To_v1beta1_OpenStackIdentityReference(in *OpenStackIdentityReference, out *infrav1.OpenStackIdentityReference, s apiconversion.Scope) error {
	return autoConvert_v1alpha7_OpenStackIdentityReference_To_v1beta1_OpenStackIdentityReference(in, out, s)
}

func Convert_v1beta1_OpenStackClusterStatus_To_v1alpha7_OpenStackClusterStatus(in *infrav1.OpenStackClusterStatus, out *OpenStackClusterStatus, s apiconversion.Scope) error {
	return autoConvert_v1beta1_OpenStackClusterStatus_To_v1alpha7_OpenStackClusterStatus(in, out, s)
}

func Convert_v1alpha7_PortOpts_To_v1beta1_PortOpts(in *PortOpts, out *infrav1.PortOpts, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha7_PortOpts_To_v1beta1_PortOpts(in, out, s); err != nil {
		return err
	}

	if len(in.SecurityGroupFilters) > 0 {
		out.SecurityGroups = make([]infrav1.SecurityGroupFilter, len(in.SecurityGroupFilters))
		for i := range in.SecurityGroupFilters {
			if err := Convert_v1alpha7_SecurityGroupFilter_To_v1beta1_SecurityGroupFilter(&in.SecurityGroupFilters[i], &out.SecurityGroups[i], s); err != nil {
				return err
			}
		}
	}

	if in.Profile != (BindingProfile{}) {
		out.Profile = &infrav1.BindingProfile{}
		if err := Convert_v1alpha7_BindingProfile_To_v1beta1_BindingProfile(&in.Profile, out.Profile, s); err != nil {
			return err
		}
	}

	return nil
}

func Convert_v1beta1_PortOpts_To_v1alpha7_PortOpts(in *infrav1.PortOpts, out *PortOpts, s apiconversion.Scope) error {
	if err := autoConvert_v1beta1_PortOpts_To_v1alpha7_PortOpts(in, out, s); err != nil {
		return err
	}

	if len(in.SecurityGroups) > 0 {
		out.SecurityGroupFilters = make([]SecurityGroupFilter, len(in.SecurityGroups))
		for i := range in.SecurityGroups {
			if err := Convert_v1beta1_SecurityGroupFilter_To_v1alpha7_SecurityGroupFilter(&in.SecurityGroups[i], &out.SecurityGroupFilters[i], s); err != nil {
				return err
			}
		}
	}

	if in.Profile != nil {
		if err := Convert_v1beta1_BindingProfile_To_v1alpha7_BindingProfile(in.Profile, &out.Profile, s); err != nil {
			return err
		}
	}

	return nil
}

func Convert_v1beta1_OpenStackIdentityReference_To_v1alpha7_OpenStackIdentityReference(in *infrav1.OpenStackIdentityReference, out *OpenStackIdentityReference, _ apiconversion.Scope) error {
	out.Name = in.Name
	return nil
}

func Convert_v1alpha7_SubnetFilter_To_v1beta1_SubnetFilter(in *SubnetFilter, out *infrav1.SubnetFilter, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha7_SubnetFilter_To_v1beta1_SubnetFilter(in, out, s); err != nil {
		return err
	}
	infrav1.ConvertAllTagsTo(in.Tags, in.TagsAny, in.NotTags, in.NotTagsAny, &out.FilterByNeutronTags)
	return nil
}

func Convert_v1beta1_SubnetFilter_To_v1alpha7_SubnetFilter(in *infrav1.SubnetFilter, out *SubnetFilter, s apiconversion.Scope) error {
	if err := autoConvert_v1beta1_SubnetFilter_To_v1alpha7_SubnetFilter(in, out, s); err != nil {
		return err
	}
	infrav1.ConvertAllTagsFrom(&in.FilterByNeutronTags, &out.Tags, &out.TagsAny, &out.NotTags, &out.NotTagsAny)
	return nil
}

func Convert_v1alpha7_SecurityGroupFilter_To_v1beta1_SecurityGroupFilter(in *SecurityGroupFilter, out *infrav1.SecurityGroupFilter, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha7_SecurityGroupFilter_To_v1beta1_SecurityGroupFilter(in, out, s); err != nil {
		return err
	}
	infrav1.ConvertAllTagsTo(in.Tags, in.TagsAny, in.NotTags, in.NotTagsAny, &out.FilterByNeutronTags)
	return nil
}

func Convert_v1beta1_SecurityGroupFilter_To_v1alpha7_SecurityGroupFilter(in *infrav1.SecurityGroupFilter, out *SecurityGroupFilter, s apiconversion.Scope) error {
	if err := autoConvert_v1beta1_SecurityGroupFilter_To_v1alpha7_SecurityGroupFilter(in, out, s); err != nil {
		return err
	}
	infrav1.ConvertAllTagsFrom(&in.FilterByNeutronTags, &out.Tags, &out.TagsAny, &out.NotTags, &out.NotTagsAny)
	return nil
}

func Convert_v1alpha7_RouterFilter_To_v1beta1_RouterFilter(in *RouterFilter, out *infrav1.RouterFilter, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha7_RouterFilter_To_v1beta1_RouterFilter(in, out, s); err != nil {
		return err
	}
	infrav1.ConvertAllTagsTo(in.Tags, in.TagsAny, in.NotTags, in.NotTagsAny, &out.FilterByNeutronTags)
	return nil
}

func Convert_v1beta1_RouterFilter_To_v1alpha7_RouterFilter(in *infrav1.RouterFilter, out *RouterFilter, s apiconversion.Scope) error {
	if err := autoConvert_v1beta1_RouterFilter_To_v1alpha7_RouterFilter(in, out, s); err != nil {
		return err
	}
	infrav1.ConvertAllTagsFrom(&in.FilterByNeutronTags, &out.Tags, &out.TagsAny, &out.NotTags, &out.NotTagsAny)
	return nil
}

func Convert_v1alpha7_NetworkFilter_To_v1beta1_NetworkFilter(in *NetworkFilter, out *infrav1.NetworkFilter, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha7_NetworkFilter_To_v1beta1_NetworkFilter(in, out, s); err != nil {
		return err
	}
	infrav1.ConvertAllTagsTo(in.Tags, in.TagsAny, in.NotTags, in.NotTagsAny, &out.FilterByNeutronTags)
	return nil
}

func Convert_v1beta1_NetworkFilter_To_v1alpha7_NetworkFilter(in *infrav1.NetworkFilter, out *NetworkFilter, s apiconversion.Scope) error {
	if err := autoConvert_v1beta1_NetworkFilter_To_v1alpha7_NetworkFilter(in, out, s); err != nil {
		return err
	}
	infrav1.ConvertAllTagsFrom(&in.FilterByNeutronTags, &out.Tags, &out.TagsAny, &out.NotTags, &out.NotTagsAny)
	return nil
}
