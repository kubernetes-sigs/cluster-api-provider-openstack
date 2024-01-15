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

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha8"
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

func restorev1alpha8SecurityGroupStatus(previous *infrav1.SecurityGroupStatus, dst *infrav1.SecurityGroupStatus) {
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

func restorev1alpha8ClusterStatus(previous *infrav1.OpenStackClusterStatus, dst *infrav1.OpenStackClusterStatus) {
	restorev1alpha8SecurityGroupStatus(previous.ControlPlaneSecurityGroup, dst.ControlPlaneSecurityGroup)
	restorev1alpha8SecurityGroupStatus(previous.WorkerSecurityGroup, dst.WorkerSecurityGroup)
	restorev1alpha8SecurityGroupStatus(previous.BastionSecurityGroup, dst.BastionSecurityGroup)

	// ReferencedResources have no equivalent in v1alpha7
	if dst.Bastion != nil {
		dst.Bastion.ReferencedResources = previous.Bastion.ReferencedResources
	}
}

var v1alpha8OpenStackClusterRestorer = conversion.RestorerFor[*infrav1.OpenStackCluster]{
	"bastion": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackCluster) **infrav1.Bastion {
			return &c.Spec.Bastion
		},
		restorev1alpha8Bastion,
	),
	"spec": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackCluster) *infrav1.OpenStackClusterSpec {
			return &c.Spec
		},
		restorev1alpha8ClusterSpec,

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
		restorev1alpha8ClusterStatus,
	),
}

func restorev1alpha7MachineSpec(previous *OpenStackMachineSpec, dst *OpenStackMachineSpec) {
	dst.FloatingIP = previous.FloatingIP

	// Conversion to v1alpha8 truncates keys and values to 255 characters
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
}

func restorev1alpha8MachineSpec(previous *infrav1.OpenStackMachineSpec, dst *infrav1.OpenStackMachineSpec) {
	dst.ServerGroup = previous.ServerGroup
	dst.Image = previous.Image
}

func restorev1alpha8Bastion(previous **infrav1.Bastion, dst **infrav1.Bastion) {
	if *previous != nil && *dst != nil {
		restorev1alpha8MachineSpec(&(*previous).Instance, &(*dst).Instance)
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
}

func restorev1alpha8ClusterSpec(previous *infrav1.OpenStackClusterSpec, dst *infrav1.OpenStackClusterSpec) {
	prevBastion := previous.Bastion
	dstBastion := dst.Bastion
	if prevBastion != nil && dstBastion != nil {
		restorev1alpha8MachineSpec(&prevBastion.Instance, &dstBastion.Instance)
	}

	// Restore all fields except ID, which should have been copied over in conversion
	dst.ExternalNetwork.Name = previous.ExternalNetwork.Name
	dst.ExternalNetwork.Description = previous.ExternalNetwork.Description
	dst.ExternalNetwork.ProjectID = previous.ExternalNetwork.ProjectID
	dst.ExternalNetwork.Tags = previous.ExternalNetwork.Tags
	dst.ExternalNetwork.TagsAny = previous.ExternalNetwork.TagsAny
	dst.ExternalNetwork.NotTags = previous.ExternalNetwork.NotTags
	dst.ExternalNetwork.NotTagsAny = previous.ExternalNetwork.NotTagsAny

	dst.DisableExternalNetwork = previous.DisableExternalNetwork

	if len(previous.Subnets) > 1 {
		dst.Subnets = append(dst.Subnets, previous.Subnets[1:]...)
	}

	dst.ManagedSubnets = previous.ManagedSubnets

	if previous.ManagedSecurityGroups != nil {
		dst.ManagedSecurityGroups.AllNodesSecurityGroupRules = previous.ManagedSecurityGroups.AllNodesSecurityGroupRules
	}
}

func (r *OpenStackCluster) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackCluster)

	return conversion.ConvertAndRestore(
		r, dst,
		Convert_v1alpha7_OpenStackCluster_To_v1alpha8_OpenStackCluster, Convert_v1alpha8_OpenStackCluster_To_v1alpha7_OpenStackCluster,
		v1alpha7OpenStackClusterRestorer, v1alpha8OpenStackClusterRestorer,
	)
}

func (r *OpenStackCluster) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackCluster)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1alpha8_OpenStackCluster_To_v1alpha7_OpenStackCluster, Convert_v1alpha7_OpenStackCluster_To_v1alpha8_OpenStackCluster,
		v1alpha8OpenStackClusterRestorer, v1alpha7OpenStackClusterRestorer,
	)
}

var _ ctrlconversion.Convertible = &OpenStackClusterList{}

func (r *OpenStackClusterList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1alpha7_OpenStackClusterList_To_v1alpha8_OpenStackClusterList(r, dst, nil)
}

func (r *OpenStackClusterList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1alpha8_OpenStackClusterList_To_v1alpha7_OpenStackClusterList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackClusterTemplate{}

func restorev1alpha7ClusterTemplateSpec(previous *OpenStackClusterTemplateSpec, dst *OpenStackClusterTemplateSpec) {
	restorev1alpha7ClusterSpec(&previous.Template.Spec, &dst.Template.Spec)
	restorev1alpha7Bastion(&previous.Template.Spec.Bastion, &dst.Template.Spec.Bastion)
}

func restorev1alpha8ClusterTemplateSpec(previous *infrav1.OpenStackClusterTemplateSpec, dst *infrav1.OpenStackClusterTemplateSpec) {
	restorev1alpha8Bastion(&previous.Template.Spec.Bastion, &dst.Template.Spec.Bastion)
	restorev1alpha8ClusterSpec(&previous.Template.Spec, &dst.Template.Spec)
}

var v1alpha7OpenStackClusterTemplateRestorer = conversion.RestorerFor[*OpenStackClusterTemplate]{
	"spec": conversion.HashedFieldRestorer(
		func(c *OpenStackClusterTemplate) *OpenStackClusterTemplateSpec {
			return &c.Spec
		},
		restorev1alpha7ClusterTemplateSpec,
	),
}

var v1alpha8OpenStackClusterTemplateRestorer = conversion.RestorerFor[*infrav1.OpenStackClusterTemplate]{
	"spec": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackClusterTemplate) *infrav1.OpenStackClusterTemplateSpec {
			return &c.Spec
		},
		restorev1alpha8ClusterTemplateSpec,
	),
}

func (r *OpenStackClusterTemplate) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterTemplate)

	return conversion.ConvertAndRestore(
		r, dst,
		Convert_v1alpha7_OpenStackClusterTemplate_To_v1alpha8_OpenStackClusterTemplate, Convert_v1alpha8_OpenStackClusterTemplate_To_v1alpha7_OpenStackClusterTemplate,
		v1alpha7OpenStackClusterTemplateRestorer, v1alpha8OpenStackClusterTemplateRestorer,
	)
}

func (r *OpenStackClusterTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterTemplate)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1alpha8_OpenStackClusterTemplate_To_v1alpha7_OpenStackClusterTemplate, Convert_v1alpha7_OpenStackClusterTemplate_To_v1alpha8_OpenStackClusterTemplate,
		v1alpha8OpenStackClusterTemplateRestorer, v1alpha7OpenStackClusterTemplateRestorer,
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

var v1alpha8OpenStackMachineRestorer = conversion.RestorerFor[*infrav1.OpenStackMachine]{
	"spec": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackMachine) *infrav1.OpenStackMachineSpec {
			return &c.Spec
		},
		restorev1alpha8MachineSpec,
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
		Convert_v1alpha7_OpenStackMachine_To_v1alpha8_OpenStackMachine, Convert_v1alpha8_OpenStackMachine_To_v1alpha7_OpenStackMachine,
		v1alpha7OpenStackMachineRestorer, v1alpha8OpenStackMachineRestorer,
	)
}

func (r *OpenStackMachine) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachine)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1alpha8_OpenStackMachine_To_v1alpha7_OpenStackMachine, Convert_v1alpha7_OpenStackMachine_To_v1alpha8_OpenStackMachine,
		v1alpha8OpenStackMachineRestorer, v1alpha7OpenStackMachineRestorer,
	)
}

var _ ctrlconversion.Convertible = &OpenStackMachineList{}

func (r *OpenStackMachineList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineList)
	return Convert_v1alpha7_OpenStackMachineList_To_v1alpha8_OpenStackMachineList(r, dst, nil)
}

func (r *OpenStackMachineList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineList)
	return Convert_v1alpha8_OpenStackMachineList_To_v1alpha7_OpenStackMachineList(src, r, nil)
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

var v1alpha8OpenStackMachineTemplateRestorer = conversion.RestorerFor[*infrav1.OpenStackMachineTemplate]{
	"spec": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackMachineTemplate) *infrav1.OpenStackMachineSpec {
			return &c.Spec.Template.Spec
		},
		restorev1alpha8MachineSpec,
	),
}

func (r *OpenStackMachineTemplate) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineTemplate)

	return conversion.ConvertAndRestore(
		r, dst,
		Convert_v1alpha7_OpenStackMachineTemplate_To_v1alpha8_OpenStackMachineTemplate, Convert_v1alpha8_OpenStackMachineTemplate_To_v1alpha7_OpenStackMachineTemplate,
		v1alpha7OpenStackMachineTemplateRestorer, v1alpha8OpenStackMachineTemplateRestorer,
	)
}

func (r *OpenStackMachineTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplate)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1alpha8_OpenStackMachineTemplate_To_v1alpha7_OpenStackMachineTemplate, Convert_v1alpha7_OpenStackMachineTemplate_To_v1alpha8_OpenStackMachineTemplate,
		v1alpha8OpenStackMachineTemplateRestorer, v1alpha7OpenStackMachineTemplateRestorer,
	)
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplateList{}

func (r *OpenStackMachineTemplateList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineTemplateList)
	return Convert_v1alpha7_OpenStackMachineTemplateList_To_v1alpha8_OpenStackMachineTemplateList(r, dst, nil)
}

func (r *OpenStackMachineTemplateList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplateList)
	return Convert_v1alpha8_OpenStackMachineTemplateList_To_v1alpha7_OpenStackMachineTemplateList(src, r, nil)
}

func Convert_v1alpha8_OpenStackMachineSpec_To_v1alpha7_OpenStackMachineSpec(in *infrav1.OpenStackMachineSpec, out *OpenStackMachineSpec, s apiconversion.Scope) error {
	err := autoConvert_v1alpha8_OpenStackMachineSpec_To_v1alpha7_OpenStackMachineSpec(in, out, s)
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

	return nil
}

func Convert_v1alpha7_OpenStackMachineSpec_To_v1alpha8_OpenStackMachineSpec(in *OpenStackMachineSpec, out *infrav1.OpenStackMachineSpec, s apiconversion.Scope) error {
	err := autoConvert_v1alpha7_OpenStackMachineSpec_To_v1alpha8_OpenStackMachineSpec(in, out, s)
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
			// was not validated prior to v1alpha8
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

	return nil
}

func Convert_v1alpha8_OpenStackMachineStatus_To_v1alpha7_OpenStackMachineStatus(in *infrav1.OpenStackMachineStatus, out *OpenStackMachineStatus, s apiconversion.Scope) error {
	// ReferencedResources have no equivalent in v1alpha7
	return autoConvert_v1alpha8_OpenStackMachineStatus_To_v1alpha7_OpenStackMachineStatus(in, out, s)
}

func Convert_v1alpha8_BastionStatus_To_v1alpha7_BastionStatus(in *infrav1.BastionStatus, out *BastionStatus, s apiconversion.Scope) error {
	// ReferencedResources have no equivalent in v1alpha7
	return autoConvert_v1alpha8_BastionStatus_To_v1alpha7_BastionStatus(in, out, s)
}

func Convert_v1alpha7_Bastion_To_v1alpha8_Bastion(in *Bastion, out *infrav1.Bastion, s apiconversion.Scope) error {
	err := autoConvert_v1alpha7_Bastion_To_v1alpha8_Bastion(in, out, s)
	if err != nil {
		return err
	}

	if in.Instance.ServerGroupID != "" {
		out.Instance.ServerGroup = &infrav1.ServerGroupFilter{ID: in.Instance.ServerGroupID}
	} else {
		out.Instance.ServerGroup = nil
	}

	out.FloatingIP = in.Instance.FloatingIP
	return nil
}

func Convert_v1alpha8_Bastion_To_v1alpha7_Bastion(in *infrav1.Bastion, out *Bastion, s apiconversion.Scope) error {
	err := autoConvert_v1alpha8_Bastion_To_v1alpha7_Bastion(in, out, s)
	if err != nil {
		return err
	}

	if in.Instance.ServerGroup != nil && in.Instance.ServerGroup.ID != "" {
		out.Instance.ServerGroupID = in.Instance.ServerGroup.ID
	}

	out.Instance.FloatingIP = in.FloatingIP
	return nil
}

func Convert_v1alpha7_OpenStackClusterSpec_To_v1alpha8_OpenStackClusterSpec(in *OpenStackClusterSpec, out *infrav1.OpenStackClusterSpec, s apiconversion.Scope) error {
	err := autoConvert_v1alpha7_OpenStackClusterSpec_To_v1alpha8_OpenStackClusterSpec(in, out, s)
	if err != nil {
		return err
	}

	if in.ExternalNetworkID != "" {
		out.ExternalNetwork = infrav1.NetworkFilter{
			ID: in.ExternalNetworkID,
		}
	}

	emptySubnet := SubnetFilter{}
	if in.Subnet != emptySubnet {
		subnet := infrav1.SubnetFilter{}
		if err := Convert_v1alpha7_SubnetFilter_To_v1alpha8_SubnetFilter(&in.Subnet, &subnet, s); err != nil {
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
		}
	}

	return nil
}

func Convert_v1alpha8_OpenStackClusterSpec_To_v1alpha7_OpenStackClusterSpec(in *infrav1.OpenStackClusterSpec, out *OpenStackClusterSpec, s apiconversion.Scope) error {
	err := autoConvert_v1alpha8_OpenStackClusterSpec_To_v1alpha7_OpenStackClusterSpec(in, out, s)
	if err != nil {
		return err
	}

	if in.ExternalNetwork.ID != "" {
		out.ExternalNetworkID = in.ExternalNetwork.ID
	}

	if len(in.Subnets) >= 1 {
		if err := Convert_v1alpha8_SubnetFilter_To_v1alpha7_SubnetFilter(&in.Subnets[0], &out.Subnet, s); err != nil {
			return err
		}
	}

	if len(in.ManagedSubnets) > 0 {
		out.NodeCIDR = in.ManagedSubnets[0].CIDR
		out.DNSNameservers = in.ManagedSubnets[0].DNSNameservers
	}

	if in.ManagedSecurityGroups != nil {
		out.ManagedSecurityGroups = true
	}

	return nil
}

func Convert_v1alpha8_SecurityGroupStatus_To_v1alpha7_SecurityGroup(in *infrav1.SecurityGroupStatus, out *SecurityGroup, s apiconversion.Scope) error { //nolint:revive
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

func Convert_v1alpha7_SecurityGroup_To_v1alpha8_SecurityGroupStatus(in *SecurityGroup, out *infrav1.SecurityGroupStatus, s apiconversion.Scope) error { //nolint:revive
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
