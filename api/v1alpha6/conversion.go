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

package v1alpha6

import (
	"reflect"
	"strings"

	apiconversion "k8s.io/apimachinery/pkg/conversion"
	"k8s.io/utils/pointer"
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/conversion"
	optional "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/optional"
)

const trueString = "true"

func restorev1alpha6SubnetFilter(previous *SubnetFilter, dst *SubnetFilter) {
	// The edge cases with multiple commas are too tricky in this direction,
	// so we just restore the whole thing.
	dst.Tags = previous.Tags
	dst.TagsAny = previous.TagsAny
	dst.NotTags = previous.NotTags
	dst.NotTagsAny = previous.NotTagsAny
}

func restorev1alpha6SecurityGroupFilter(previous *SecurityGroupFilter, dst *SecurityGroupFilter) {
	// The edge cases with multiple commas are too tricky in this direction,
	// so we just restore the whole thing.
	dst.Tags = previous.Tags
	dst.TagsAny = previous.TagsAny
	dst.NotTags = previous.NotTags
	dst.NotTagsAny = previous.NotTagsAny
}

func restorev1alpha6NetworkFilter(previous *NetworkFilter, dst *NetworkFilter) {
	// The edge cases with multiple commas are too tricky in this direction,
	// so we just restore the whole thing.
	dst.Tags = previous.Tags
	dst.TagsAny = previous.TagsAny
	dst.NotTags = previous.NotTags
	dst.NotTagsAny = previous.NotTagsAny
}

func restorev1alpha6SecurityGroup(previous *SecurityGroup, dst *SecurityGroup) {
	if previous == nil || dst == nil {
		return
	}

	for i, rule := range previous.Rules {
		dst.Rules[i].SecurityGroupID = rule.SecurityGroupID
	}
}

func restorev1alpha6Port(previous *PortOpts, dst *PortOpts) {
	if len(dst.SecurityGroupFilters) == len(previous.SecurityGroupFilters) {
		for i := range dst.SecurityGroupFilters {
			restorev1alpha6SecurityGroupFilter(&previous.SecurityGroupFilters[i].Filter, &dst.SecurityGroupFilters[i].Filter)
		}
	}

	if dst.Network != nil && previous.Network != nil {
		restorev1alpha6NetworkFilter(previous.Network, dst.Network)
	}

	if len(dst.FixedIPs) == len(previous.FixedIPs) {
		for i := range dst.FixedIPs {
			prevFixedIP := &previous.FixedIPs[i]
			dstFixedIP := &dst.FixedIPs[i]

			if dstFixedIP.Subnet != nil && prevFixedIP.Subnet != nil {
				restorev1alpha6SubnetFilter(prevFixedIP.Subnet, dstFixedIP.Subnet)
			}
		}
	}
}

func restorev1alpha6MachineSpec(previous *OpenStackMachineSpec, dst *OpenStackMachineSpec) {
	// Subnet is removed from v1beta1 with no replacement, so can't be
	// losslessly converted. Restore the previously stored value on down-conversion.
	dst.Subnet = previous.Subnet

	// Strictly speaking this is lossy because we lose changes in
	// down-conversion which were made to the up-converted object. However
	// it isn't worth implementing this as the fields are immutable.
	dst.Networks = previous.Networks
	dst.Ports = previous.Ports
	dst.SecurityGroups = previous.SecurityGroups

	// FloatingIP is removed from v1alpha7 with no replacement, so can't be
	// losslessly converted. Restore the previously stored value on down-conversion.
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

	// Conversion to v1beta1 removes the Kind fild
	dst.IdentityRef = previous.IdentityRef

	if len(dst.Ports) == len(previous.Ports) {
		for i := range dst.Ports {
			restorev1alpha6Port(&previous.Ports[i], &dst.Ports[i])
		}
	}

	if len(dst.SecurityGroups) == len(previous.SecurityGroups) {
		for i := range dst.SecurityGroups {
			restorev1alpha6SecurityGroupFilter(&previous.SecurityGroups[i].Filter, &dst.SecurityGroups[i].Filter)
		}
	}
}

func restorev1alpha6ClusterStatus(previous *OpenStackClusterStatus, dst *OpenStackClusterStatus) {
	// PortOpts.SecurityGroups have been removed in v1beta1
	// We restore the whole PortOpts/Networks since they are anyway immutable.
	if previous.ExternalNetwork != nil {
		dst.ExternalNetwork.PortOpts = previous.ExternalNetwork.PortOpts
	}
	if previous.Network != nil {
		dst.Network = previous.Network
	}
	if previous.Bastion != nil && previous.Bastion.Networks != nil {
		dst.Bastion.Networks = previous.Bastion.Networks
	}

	restorev1alpha6SecurityGroup(previous.ControlPlaneSecurityGroup, dst.ControlPlaneSecurityGroup)
	restorev1alpha6SecurityGroup(previous.WorkerSecurityGroup, dst.WorkerSecurityGroup)
	restorev1alpha6SecurityGroup(previous.BastionSecurityGroup, dst.BastionSecurityGroup)
}

func restorev1beta1MachineSpec(previous *infrav1.OpenStackMachineSpec, dst *infrav1.OpenStackMachineSpec) {
	// PropagateUplinkStatus has been added in v1beta1.
	// We restore the whole Ports since they are anyway immutable.
	dst.Ports = previous.Ports
	dst.AdditionalBlockDevices = previous.AdditionalBlockDevices
	dst.ServerGroup = previous.ServerGroup
	dst.Image = previous.Image
}

func restorev1beta1Bastion(previous **infrav1.Bastion, dst **infrav1.Bastion) {
	if *previous != nil && *dst != nil {
		restorev1beta1MachineSpec(&(*previous).Instance, &(*dst).Instance)
	}
}

func restorev1beta1Subnets(previous *[]infrav1.SubnetFilter, dst *[]infrav1.SubnetFilter) {
	if len(*previous) > 1 {
		*dst = append(*dst, (*previous)[1:]...)
	}
}

func restorev1beta1APIServerLoadBalancer(previous **infrav1.APIServerLoadBalancer, dst **infrav1.APIServerLoadBalancer) {
	// Ensure empty and zero values are restored identically
	if *previous == nil || (*previous).IsZero() {
		*dst = *previous
	}
}

func restorev1beta1ClusterStatus(previous *infrav1.OpenStackClusterStatus, dst *infrav1.OpenStackClusterStatus) {
	// It's (theoretically) possible in v1beta1 to have Network nil but
	// Router or APIServerLoadBalancer not nil. In hub-spoke-hub conversion this will
	// result in Network being a pointer to an empty object.
	if previous.Network == nil && dst.Network != nil && reflect.ValueOf(*dst.Network).IsZero() {
		dst.Network = nil
	}

	dst.ControlPlaneSecurityGroup = previous.ControlPlaneSecurityGroup
	dst.WorkerSecurityGroup = previous.WorkerSecurityGroup
	dst.BastionSecurityGroup = previous.BastionSecurityGroup

	if previous.Bastion != nil {
		dst.Bastion.ReferencedResources = previous.Bastion.ReferencedResources
	}
	if previous.Bastion != nil && previous.Bastion.DependentResources.PortsStatus != nil {
		dst.Bastion.DependentResources.PortsStatus = previous.Bastion.DependentResources.PortsStatus
	}
}

func restorev1alpha6ClusterSpec(previous *OpenStackClusterSpec, dst *OpenStackClusterSpec) {
	for i := range previous.ExternalRouterIPs {
		dstIP := &dst.ExternalRouterIPs[i]
		previousIP := &previous.ExternalRouterIPs[i]

		// Subnet.Filter.ID was overwritten in up-conversion by Subnet.UUID
		dstIP.Subnet.Filter.ID = previousIP.Subnet.Filter.ID

		// If Subnet.UUID was previously unset, we overwrote it with the value of Subnet.Filter.ID
		// Don't unset it again if it doesn't have the previous value of Subnet.Filter.ID, because that means it was genuinely changed
		if previousIP.Subnet.UUID == "" && dstIP.Subnet.UUID == previousIP.Subnet.Filter.ID {
			dstIP.Subnet.UUID = ""
		}
	}

	// We only restore DNSNameservers when these were lossly converted when NodeCIDR is empty.
	if len(previous.DNSNameservers) > 0 && dst.NodeCIDR == "" {
		dst.DNSNameservers = previous.DNSNameservers
	}

	prevBastion := previous.Bastion
	dstBastion := dst.Bastion
	if prevBastion != nil && dstBastion != nil {
		restorev1alpha6MachineSpec(&prevBastion.Instance, &dstBastion.Instance)
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
			restorev1alpha6SubnetFilter(&previous.ExternalRouterIPs[i].Subnet.Filter, &dst.ExternalRouterIPs[i].Subnet.Filter)
		}
	}

	restorev1alpha6SubnetFilter(&previous.Subnet, &dst.Subnet)

	restorev1alpha6NetworkFilter(&previous.Network, &dst.Network)
}

// Ensure nil and &0 are restored to whichever they were previously.
func restoreIntPointer(previous **int, dst **int) {
	if *previous == nil || **previous == 0 {
		*dst = *previous
	}
}

var _ ctrlconversion.Convertible = &OpenStackCluster{}

var v1alpha6OpenStackClusterRestorer = conversion.RestorerFor[*OpenStackCluster]{
	"spec": conversion.HashedFieldRestorer(
		func(c *OpenStackCluster) *OpenStackClusterSpec {
			return &c.Spec
		},
		restorev1alpha6ClusterSpec,
	),
	"status": conversion.HashedFieldRestorer(
		func(c *OpenStackCluster) *OpenStackClusterStatus {
			return &c.Status
		},
		restorev1alpha6ClusterStatus,
	),
}

var v1beta1OpenStackClusterRestorer = conversion.RestorerFor[*infrav1.OpenStackCluster]{
	"apiServerLoadBalancer": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackCluster) **infrav1.APIServerLoadBalancer {
			return &c.Spec.APIServerLoadBalancer
		},
		restorev1beta1APIServerLoadBalancer,
	),
	"apiServerFloatingIP": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackCluster) *optional.String {
			return &c.Spec.APIServerFloatingIP
		},
		optional.RestoreString,
	),
	"apiServerFixedIP": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackCluster) *optional.String {
			return &c.Spec.APIServerFixedIP
		},
		optional.RestoreString,
	),
	"apiServerPort": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackCluster) *optional.Int {
			return &c.Spec.APIServerPort
		},
		optional.RestoreInt,
	),
	"externalNetwork": conversion.UnconditionalFieldRestorer(
		func(c *infrav1.OpenStackCluster) **infrav1.NetworkFilter {
			return &c.Spec.ExternalNetwork
		},
	),
	"disableExternalNetwork": conversion.UnconditionalFieldRestorer(
		func(c *infrav1.OpenStackCluster) *bool {
			return &c.Spec.DisableExternalNetwork
		},
	),
	"router": conversion.UnconditionalFieldRestorer(
		func(c *infrav1.OpenStackCluster) **infrav1.RouterFilter {
			return &c.Spec.Router
		},
	),
	"networkMtu": conversion.UnconditionalFieldRestorer(
		func(c *infrav1.OpenStackCluster) *optional.Int {
			return &c.Spec.NetworkMTU
		},
	),
	"bastion": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackCluster) **infrav1.Bastion {
			return &c.Spec.Bastion
		},
		restorev1beta1Bastion,
	),
	"subnets": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackCluster) *[]infrav1.SubnetFilter {
			return &c.Spec.Subnets
		},
		restorev1beta1Subnets,
	),
	"allNodesSecurityGroupRules": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackCluster) *infrav1.ManagedSecurityGroups {
			return c.Spec.ManagedSecurityGroups
		},
		restorev1beta1ManagedSecurityGroups,
	),
	"status": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackCluster) *infrav1.OpenStackClusterStatus {
			return &c.Status
		},
		restorev1beta1ClusterStatus,
	),
	"managedSubnets": conversion.UnconditionalFieldRestorer(
		func(c *infrav1.OpenStackCluster) *[]infrav1.SubnetSpec {
			return &c.Spec.ManagedSubnets
		},
	),
}

func (r *OpenStackCluster) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackCluster)

	return conversion.ConvertAndRestore(
		r, dst,
		Convert_v1alpha6_OpenStackCluster_To_v1beta1_OpenStackCluster, Convert_v1beta1_OpenStackCluster_To_v1alpha6_OpenStackCluster,
		v1alpha6OpenStackClusterRestorer, v1beta1OpenStackClusterRestorer,
	)
}

func (r *OpenStackCluster) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackCluster)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1beta1_OpenStackCluster_To_v1alpha6_OpenStackCluster, Convert_v1alpha6_OpenStackCluster_To_v1beta1_OpenStackCluster,
		v1beta1OpenStackClusterRestorer, v1alpha6OpenStackClusterRestorer,
	)
}

var _ ctrlconversion.Convertible = &OpenStackClusterList{}

func (r *OpenStackClusterList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1alpha6_OpenStackClusterList_To_v1beta1_OpenStackClusterList(r, dst, nil)
}

func (r *OpenStackClusterList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1beta1_OpenStackClusterList_To_v1alpha6_OpenStackClusterList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackClusterTemplate{}

var v1alpha6OpenStackClusterTemplateRestorer = conversion.RestorerFor[*OpenStackClusterTemplate]{
	"spec": conversion.HashedFieldRestorer(
		func(c *OpenStackClusterTemplate) *OpenStackClusterSpec {
			return &c.Spec.Template.Spec
		},
		restorev1alpha6ClusterSpec,
	),
}

func restorev1beta1ManagedSecurityGroups(previous *infrav1.ManagedSecurityGroups, dst *infrav1.ManagedSecurityGroups) {
	dst.AllNodesSecurityGroupRules = previous.AllNodesSecurityGroupRules
}

var v1beta1OpenStackClusterTemplateRestorer = conversion.RestorerFor[*infrav1.OpenStackClusterTemplate]{
	"apiServerLoadBalancer": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackClusterTemplate) **infrav1.APIServerLoadBalancer {
			return &c.Spec.Template.Spec.APIServerLoadBalancer
		},
		restorev1beta1APIServerLoadBalancer,
	),
	"apiServerFloatingIP": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackClusterTemplate) *optional.String {
			return &c.Spec.Template.Spec.APIServerFloatingIP
		},
		optional.RestoreString,
	),
	"apiServerFixedIP": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackClusterTemplate) *optional.String {
			return &c.Spec.Template.Spec.APIServerFixedIP
		},
		optional.RestoreString,
	),
	"apiServerPort": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackClusterTemplate) *optional.Int {
			return &c.Spec.Template.Spec.APIServerPort
		},
		optional.RestoreInt,
	),
	"externalNetwork": conversion.UnconditionalFieldRestorer(
		func(c *infrav1.OpenStackClusterTemplate) **infrav1.NetworkFilter {
			return &c.Spec.Template.Spec.ExternalNetwork
		},
	),
	"disableExternalNetwork": conversion.UnconditionalFieldRestorer(
		func(c *infrav1.OpenStackClusterTemplate) *bool {
			return &c.Spec.Template.Spec.DisableExternalNetwork
		},
	),
	"router": conversion.UnconditionalFieldRestorer(
		func(c *infrav1.OpenStackClusterTemplate) **infrav1.RouterFilter {
			return &c.Spec.Template.Spec.Router
		},
	),
	"networkMtu": conversion.UnconditionalFieldRestorer(
		func(c *infrav1.OpenStackClusterTemplate) *optional.Int {
			return &c.Spec.Template.Spec.NetworkMTU
		},
	),
	"bastion": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackClusterTemplate) **infrav1.Bastion {
			return &c.Spec.Template.Spec.Bastion
		},
		restorev1beta1Bastion,
	),
	"subnets": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackClusterTemplate) *[]infrav1.SubnetFilter {
			return &c.Spec.Template.Spec.Subnets
		},
		restorev1beta1Subnets,
	),
	"allNodesSecurityGroupRules": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackClusterTemplate) *infrav1.ManagedSecurityGroups {
			return c.Spec.Template.Spec.ManagedSecurityGroups
		},
		restorev1beta1ManagedSecurityGroups,
	),
	"managedSubnets": conversion.UnconditionalFieldRestorer(
		func(c *infrav1.OpenStackClusterTemplate) *[]infrav1.SubnetSpec {
			return &c.Spec.Template.Spec.ManagedSubnets
		},
	),
}

func (r *OpenStackClusterTemplate) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterTemplate)

	return conversion.ConvertAndRestore(
		r, dst,
		Convert_v1alpha6_OpenStackClusterTemplate_To_v1beta1_OpenStackClusterTemplate, Convert_v1beta1_OpenStackClusterTemplate_To_v1alpha6_OpenStackClusterTemplate,
		v1alpha6OpenStackClusterTemplateRestorer, v1beta1OpenStackClusterTemplateRestorer,
	)
}

func (r *OpenStackClusterTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterTemplate)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1beta1_OpenStackClusterTemplate_To_v1alpha6_OpenStackClusterTemplate, Convert_v1alpha6_OpenStackClusterTemplate_To_v1beta1_OpenStackClusterTemplate,
		v1beta1OpenStackClusterTemplateRestorer, v1alpha6OpenStackClusterTemplateRestorer,
	)
}

var _ ctrlconversion.Convertible = &OpenStackMachine{}

var v1alpha6OpenStackMachineRestorer = conversion.RestorerFor[*OpenStackMachine]{
	"spec": conversion.HashedFieldRestorer(
		func(c *OpenStackMachine) *OpenStackMachineSpec {
			return &c.Spec
		},
		restorev1alpha6MachineSpec,
		conversion.HashedFilterField[*OpenStackMachine, OpenStackMachineSpec](func(s *OpenStackMachineSpec) *OpenStackMachineSpec {
			// Despite being spec fields, ProviderID and InstanceID
			// are both set by the machine controller. If these are
			// the only changes to the spec, we still want to
			// restore the rest of the spec to its original state.
			if s.ProviderID != nil || s.InstanceID != nil {
				f := *s
				f.ProviderID = nil
				f.InstanceID = nil
				return &f
			}
			return s
		}),
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
	// No equivalent in v1alpha6
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
		Convert_v1alpha6_OpenStackMachine_To_v1beta1_OpenStackMachine, Convert_v1beta1_OpenStackMachine_To_v1alpha6_OpenStackMachine,
		v1alpha6OpenStackMachineRestorer, v1beta1OpenStackMachineRestorer,
	)
}

func (r *OpenStackMachine) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachine)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1beta1_OpenStackMachine_To_v1alpha6_OpenStackMachine, Convert_v1alpha6_OpenStackMachine_To_v1beta1_OpenStackMachine,
		v1beta1OpenStackMachineRestorer, v1alpha6OpenStackMachineRestorer,
	)
}

var _ ctrlconversion.Convertible = &OpenStackMachineList{}

func (r *OpenStackMachineList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineList)
	return Convert_v1alpha6_OpenStackMachineList_To_v1beta1_OpenStackMachineList(r, dst, nil)
}

func (r *OpenStackMachineList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineList)
	return Convert_v1beta1_OpenStackMachineList_To_v1alpha6_OpenStackMachineList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplate{}

var v1alpha6OpenStackMachineTemplateRestorer = conversion.RestorerFor[*OpenStackMachineTemplate]{
	"spec": conversion.HashedFieldRestorer(
		func(c *OpenStackMachineTemplate) *OpenStackMachineSpec {
			return &c.Spec.Template.Spec
		},
		restorev1alpha6MachineSpec,
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
		Convert_v1alpha6_OpenStackMachineTemplate_To_v1beta1_OpenStackMachineTemplate, Convert_v1beta1_OpenStackMachineTemplate_To_v1alpha6_OpenStackMachineTemplate,
		v1alpha6OpenStackMachineTemplateRestorer, v1beta1OpenStackMachineTemplateRestorer,
	)
}

func (r *OpenStackMachineTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplate)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1beta1_OpenStackMachineTemplate_To_v1alpha6_OpenStackMachineTemplate, Convert_v1alpha6_OpenStackMachineTemplate_To_v1beta1_OpenStackMachineTemplate,
		v1beta1OpenStackMachineTemplateRestorer, v1alpha6OpenStackMachineTemplateRestorer,
	)
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplateList{}

func (r *OpenStackMachineTemplateList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineTemplateList)
	return Convert_v1alpha6_OpenStackMachineTemplateList_To_v1beta1_OpenStackMachineTemplateList(r, dst, nil)
}

func (r *OpenStackMachineTemplateList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplateList)
	return Convert_v1beta1_OpenStackMachineTemplateList_To_v1alpha6_OpenStackMachineTemplateList(src, r, nil)
}

func Convert_v1alpha6_OpenStackMachineSpec_To_v1beta1_OpenStackMachineSpec(in *OpenStackMachineSpec, out *infrav1.OpenStackMachineSpec, s apiconversion.Scope) error {
	err := autoConvert_v1alpha6_OpenStackMachineSpec_To_v1beta1_OpenStackMachineSpec(in, out, s)
	if err != nil {
		return err
	}

	if len(in.Networks) > 0 {
		ports, err := convertNetworksToPorts(in.Networks, s)
		if err != nil {
			return err
		}
		// Networks were previously created first, so need to come before ports
		out.Ports = append(ports, out.Ports...)
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

	if in.IdentityRef != nil {
		out.IdentityRef = &infrav1.OpenStackIdentityReference{Name: in.IdentityRef.Name}
	}
	if in.CloudName != "" {
		if out.IdentityRef == nil {
			out.IdentityRef = &infrav1.OpenStackIdentityReference{}
		}
		out.IdentityRef.CloudName = in.CloudName
	}

	return nil
}

func convertNetworksToPorts(networks []NetworkParam, s apiconversion.Scope) ([]infrav1.PortOpts, error) {
	var ports []infrav1.PortOpts

	for i := range networks {
		network := networks[i]

		// This will remain null if the network is not specified in NetworkParam
		var networkFilter *infrav1.NetworkFilter

		// In v1alpha6, if network.Filter resolved to multiple networks
		// then we would add multiple ports. It is not possible to
		// support this behaviour during k8s API conversion as it
		// requires an OpenStack API call. A network filter returning
		// multiple networks now becomes an error when we attempt to
		// create the port.
		switch {
		case network.UUID != "":
			networkFilter = &infrav1.NetworkFilter{
				ID: network.UUID,
			}
		case network.Filter != (NetworkFilter{}):
			networkFilter = &infrav1.NetworkFilter{}
			if err := Convert_v1alpha6_NetworkFilter_To_v1beta1_NetworkFilter(&network.Filter, networkFilter, s); err != nil {
				return nil, err
			}
		}

		// Note that network.FixedIP was unused in v1alpha6 so we also ignore it here.

		// In v1alpha6, specifying multiple subnets created multiple
		// ports. We maintain this behaviour in conversion by adding
		// multiple portOpts in this case.
		//
		// Also, similar to network.Filter above, if a subnet filter
		// resolved to multiple subnets then we would add a port for
		// each subnet. Again, it is not possible to support this
		// behaviour during k8s API conversion as it requires an
		// OpenStack API call. A subnet filter returning multiple
		// subnets now becomes an error when we attempt to create the
		// port.
		if len(network.Subnets) == 0 {
			// If the network has no explicit subnets then we create a single port with no subnets.
			ports = append(ports, infrav1.PortOpts{Network: networkFilter})
		} else {
			// If the network has explicit subnets then we create a separate port for each subnet.
			for i := range network.Subnets {
				subnet := network.Subnets[i]
				if subnet.UUID != "" {
					ports = append(ports, infrav1.PortOpts{
						Network: networkFilter,
						FixedIPs: []infrav1.FixedIP{
							{Subnet: &infrav1.SubnetFilter{ID: subnet.UUID}},
						},
					})
				} else {
					subnetFilter := &infrav1.SubnetFilter{}
					if err := Convert_v1alpha6_SubnetFilter_To_v1beta1_SubnetFilter(&subnet.Filter, subnetFilter, s); err != nil {
						return nil, err
					}
					ports = append(ports, infrav1.PortOpts{
						Network: networkFilter,
						FixedIPs: []infrav1.FixedIP{
							{Subnet: subnetFilter},
						},
					})
				}
			}
		}
	}

	return ports, nil
}

func Convert_v1beta1_OpenStackClusterSpec_To_v1alpha6_OpenStackClusterSpec(in *infrav1.OpenStackClusterSpec, out *OpenStackClusterSpec, s apiconversion.Scope) error {
	err := autoConvert_v1beta1_OpenStackClusterSpec_To_v1alpha6_OpenStackClusterSpec(in, out, s)
	if err != nil {
		return err
	}

	if in.Network != nil {
		if err := Convert_v1beta1_NetworkFilter_To_v1alpha6_NetworkFilter(in.Network, &out.Network, s); err != nil {
			return err
		}
	}

	if in.ExternalNetwork != nil && in.ExternalNetwork.ID != "" {
		out.ExternalNetworkID = in.ExternalNetwork.ID
	}

	if len(in.Subnets) >= 1 {
		if err := Convert_v1beta1_SubnetFilter_To_v1alpha6_SubnetFilter(&in.Subnets[0], &out.Subnet, s); err != nil {
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
		if err := Convert_v1beta1_APIServerLoadBalancer_To_v1alpha6_APIServerLoadBalancer(in.APIServerLoadBalancer, &out.APIServerLoadBalancer, s); err != nil {
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

func Convert_v1alpha6_OpenStackClusterSpec_To_v1beta1_OpenStackClusterSpec(in *OpenStackClusterSpec, out *infrav1.OpenStackClusterSpec, s apiconversion.Scope) error {
	err := autoConvert_v1alpha6_OpenStackClusterSpec_To_v1beta1_OpenStackClusterSpec(in, out, s)
	if err != nil {
		return err
	}

	if in.Network != (NetworkFilter{}) {
		out.Network = &infrav1.NetworkFilter{}
		if err := Convert_v1alpha6_NetworkFilter_To_v1beta1_NetworkFilter(&in.Network, out.Network, s); err != nil {
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
		if err := Convert_v1alpha6_SubnetFilter_To_v1beta1_SubnetFilter(&in.Subnet, &subnet, s); err != nil {
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

	out.IdentityRef.CloudName = in.CloudName
	if in.IdentityRef != nil {
		out.IdentityRef.Name = in.IdentityRef.Name
	}

	apiServerLoadBalancer := &infrav1.APIServerLoadBalancer{}
	if err := Convert_v1alpha6_APIServerLoadBalancer_To_v1beta1_APIServerLoadBalancer(&in.APIServerLoadBalancer, apiServerLoadBalancer, s); err != nil {
		return err
	}
	if !apiServerLoadBalancer.IsZero() {
		out.APIServerLoadBalancer = apiServerLoadBalancer
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

func Convert_v1alpha6_PortOpts_To_v1beta1_PortOpts(in *PortOpts, out *infrav1.PortOpts, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha6_PortOpts_To_v1beta1_PortOpts(in, out, s); err != nil {
		return err
	}

	if len(in.SecurityGroups) > 0 || len(in.SecurityGroupFilters) > 0 {
		out.SecurityGroups = make([]infrav1.SecurityGroupFilter, 0, len(in.SecurityGroups)+len(in.SecurityGroupFilters))
		for i := range in.SecurityGroupFilters {
			sgParam := &in.SecurityGroupFilters[i]
			switch {
			case sgParam.UUID != "":
				out.SecurityGroups = append(out.SecurityGroups, infrav1.SecurityGroupFilter{ID: sgParam.UUID})
			case sgParam.Name != "":
				out.SecurityGroups = append(out.SecurityGroups, infrav1.SecurityGroupFilter{Name: sgParam.Name})
			case sgParam.Filter != (SecurityGroupFilter{}):
				out.SecurityGroups = append(out.SecurityGroups, infrav1.SecurityGroupFilter{})
				outSG := &out.SecurityGroups[len(out.SecurityGroups)-1]
				if err := Convert_v1alpha6_SecurityGroupFilter_To_v1beta1_SecurityGroupFilter(&sgParam.Filter, outSG, s); err != nil {
					return err
				}
			}
		}
		for _, id := range in.SecurityGroups {
			out.SecurityGroups = append(out.SecurityGroups, infrav1.SecurityGroupFilter{ID: id})
		}
	}

	// Profile is now a struct in v1beta1.
	var ovsHWOffload, trustedVF bool
	if strings.Contains(in.Profile["capabilities"], "switchdev") {
		ovsHWOffload = true
	}
	if in.Profile["trusted"] == trueString {
		trustedVF = true
	}
	if ovsHWOffload || trustedVF {
		out.Profile = &infrav1.BindingProfile{}
		if ovsHWOffload {
			out.Profile.OVSHWOffload = &ovsHWOffload
		}
		if trustedVF {
			out.Profile.TrustedVF = &trustedVF
		}
	}

	return nil
}

func Convert_v1beta1_SecurityGroupFilter_To_string(in *infrav1.SecurityGroupFilter, out *string, _ apiconversion.Scope) error {
	if in.ID != "" {
		*out = in.ID
	}
	return nil
}

func Convert_v1beta1_PortOpts_To_v1alpha6_PortOpts(in *infrav1.PortOpts, out *PortOpts, s apiconversion.Scope) error {
	if err := autoConvert_v1beta1_PortOpts_To_v1alpha6_PortOpts(in, out, s); err != nil {
		return err
	}

	// The auto-generated function converts v1beta1 SecurityGroup to
	// v1alpha6 SecurityGroup, but v1alpha6 SecurityGroupFilter is more
	// appropriate. Unset them and convert to SecurityGroupFilter instead.
	out.SecurityGroups = nil
	if len(in.SecurityGroups) > 0 {
		out.SecurityGroupFilters = make([]SecurityGroupParam, len(in.SecurityGroups))
		for i := range in.SecurityGroups {
			securityGroupParam := &out.SecurityGroupFilters[i]
			if in.SecurityGroups[i].ID != "" {
				securityGroupParam.UUID = in.SecurityGroups[i].ID
			} else {
				if err := Convert_v1beta1_SecurityGroupFilter_To_v1alpha6_SecurityGroupFilter(&in.SecurityGroups[i], &securityGroupParam.Filter, s); err != nil {
					return err
				}
			}
		}
	}

	if in.Profile != nil {
		out.Profile = make(map[string]string)
		if pointer.BoolDeref(in.Profile.OVSHWOffload, false) {
			(out.Profile)["capabilities"] = "[\"switchdev\"]"
		}
		if pointer.BoolDeref(in.Profile.TrustedVF, false) {
			(out.Profile)["trusted"] = trueString
		}
	}

	return nil
}

func Convert_v1alpha6_Instance_To_v1beta1_BastionStatus(in *Instance, out *infrav1.BastionStatus, _ apiconversion.Scope) error {
	// BastionStatus is the same as Instance with unused fields removed
	out.ID = in.ID
	out.Name = in.Name
	out.SSHKeyName = in.SSHKeyName
	out.State = infrav1.InstanceState(in.State)
	out.IP = in.IP
	out.FloatingIP = in.FloatingIP
	return nil
}

func Convert_v1beta1_BastionStatus_To_v1alpha6_Instance(in *infrav1.BastionStatus, out *Instance, _ apiconversion.Scope) error {
	// BastionStatus is the same as Instance with unused fields removed
	out.ID = in.ID
	out.Name = in.Name
	out.SSHKeyName = in.SSHKeyName
	out.State = InstanceState(in.State)
	out.IP = in.IP
	out.FloatingIP = in.FloatingIP
	return nil
}

func Convert_v1alpha6_Network_To_v1beta1_NetworkStatusWithSubnets(in *Network, out *infrav1.NetworkStatusWithSubnets, s apiconversion.Scope) error {
	// PortOpts has been removed in v1beta1
	err := Convert_v1alpha6_Network_To_v1beta1_NetworkStatus(in, &out.NetworkStatus, s)
	if err != nil {
		return err
	}

	if in.Subnet != nil {
		out.Subnets = []infrav1.Subnet{infrav1.Subnet(*in.Subnet)}
	}
	return nil
}

func Convert_v1beta1_NetworkStatusWithSubnets_To_v1alpha6_Network(in *infrav1.NetworkStatusWithSubnets, out *Network, s apiconversion.Scope) error {
	// PortOpts has been removed in v1beta1
	err := Convert_v1beta1_NetworkStatus_To_v1alpha6_Network(&in.NetworkStatus, out, s)
	if err != nil {
		return err
	}

	// Can only down-convert a single subnet
	if len(in.Subnets) > 0 {
		out.Subnet = (*Subnet)(&in.Subnets[0])
	}
	return nil
}

func Convert_v1alpha6_Network_To_v1beta1_NetworkStatus(in *Network, out *infrav1.NetworkStatus, _ apiconversion.Scope) error {
	out.ID = in.ID
	out.Name = in.Name
	out.Tags = in.Tags

	return nil
}

func Convert_v1beta1_NetworkStatus_To_v1alpha6_Network(in *infrav1.NetworkStatus, out *Network, _ apiconversion.Scope) error {
	out.ID = in.ID
	out.Name = in.Name
	out.Tags = in.Tags

	return nil
}

func Convert_v1alpha6_SecurityGroupParam_To_v1beta1_SecurityGroupFilter(in *SecurityGroupParam, out *infrav1.SecurityGroupFilter, s apiconversion.Scope) error {
	// SecurityGroupParam is replaced by its contained SecurityGroupFilter in v1beta1
	err := Convert_v1alpha6_SecurityGroupFilter_To_v1beta1_SecurityGroupFilter(&in.Filter, out, s)
	if err != nil {
		return err
	}

	if in.UUID != "" {
		out.ID = in.UUID
	}
	if in.Name != "" {
		out.Name = in.Name
	}
	return nil
}

func Convert_v1beta1_SecurityGroupFilter_To_v1alpha6_SecurityGroupParam(in *infrav1.SecurityGroupFilter, out *SecurityGroupParam, s apiconversion.Scope) error {
	// SecurityGroupParam is replaced by its contained SecurityGroupFilter in v1beta1
	err := Convert_v1beta1_SecurityGroupFilter_To_v1alpha6_SecurityGroupFilter(in, &out.Filter, s)
	if err != nil {
		return err
	}

	if in.ID != "" {
		out.UUID = in.ID
	}
	if in.Name != "" {
		out.Name = in.Name
	}
	return nil
}

func Convert_v1alpha6_SubnetParam_To_v1beta1_SubnetFilter(in *SubnetParam, out *infrav1.SubnetFilter, s apiconversion.Scope) error {
	if err := Convert_v1alpha6_SubnetFilter_To_v1beta1_SubnetFilter(&in.Filter, out, s); err != nil {
		return err
	}
	if in.UUID != "" {
		out.ID = in.UUID
	}
	return nil
}

func Convert_v1beta1_SubnetFilter_To_v1alpha6_SubnetParam(in *infrav1.SubnetFilter, out *SubnetParam, s apiconversion.Scope) error {
	if err := Convert_v1beta1_SubnetFilter_To_v1alpha6_SubnetFilter(in, &out.Filter, s); err != nil {
		return err
	}
	out.UUID = in.ID

	return nil
}

func Convert_Map_string_To_Interface_To_v1beta1_BindingProfile(in map[string]string, out *infrav1.BindingProfile, _ apiconversion.Scope) error {
	for k, v := range in {
		if k == "capabilities" {
			if strings.Contains(v, "switchdev") {
				out.OVSHWOffload = pointer.Bool(true)
			}
		}
		if k == "trusted" && v == trueString {
			out.TrustedVF = pointer.Bool(true)
		}
	}
	return nil
}

func Convert_v1beta1_BindingProfile_To_Map_string_To_Interface(in *infrav1.BindingProfile, out map[string]string, _ apiconversion.Scope) error {
	if pointer.BoolDeref(in.OVSHWOffload, false) {
		(out)["capabilities"] = "[\"switchdev\"]"
	}
	if pointer.BoolDeref(in.TrustedVF, false) {
		(out)["trusted"] = trueString
	}
	return nil
}

func Convert_v1beta1_OpenStackClusterStatus_To_v1alpha6_OpenStackClusterStatus(in *infrav1.OpenStackClusterStatus, out *OpenStackClusterStatus, s apiconversion.Scope) error {
	err := autoConvert_v1beta1_OpenStackClusterStatus_To_v1alpha6_OpenStackClusterStatus(in, out, s)
	if err != nil {
		return err
	}

	// Router and APIServerLoadBalancer have been moved out of Network in v1beta1
	if in.Router != nil || in.APIServerLoadBalancer != nil {
		if out.Network == nil {
			out.Network = &Network{}
		}

		out.Network.Router = (*Router)(in.Router)
		out.Network.APIServerLoadBalancer = (*LoadBalancer)(in.APIServerLoadBalancer)
	}

	return nil
}

func Convert_v1alpha6_OpenStackClusterStatus_To_v1beta1_OpenStackClusterStatus(in *OpenStackClusterStatus, out *infrav1.OpenStackClusterStatus, s apiconversion.Scope) error {
	err := autoConvert_v1alpha6_OpenStackClusterStatus_To_v1beta1_OpenStackClusterStatus(in, out, s)
	if err != nil {
		return err
	}

	// Router and APIServerLoadBalancer have been moved out of Network in v1beta1
	if in.Network != nil {
		out.Router = (*infrav1.Router)(in.Network.Router)
		out.APIServerLoadBalancer = (*infrav1.LoadBalancer)(in.Network.APIServerLoadBalancer)
	}

	return nil
}

func Convert_v1beta1_OpenStackMachineSpec_To_v1alpha6_OpenStackMachineSpec(in *infrav1.OpenStackMachineSpec, out *OpenStackMachineSpec, s apiconversion.Scope) error {
	err := autoConvert_v1beta1_OpenStackMachineSpec_To_v1alpha6_OpenStackMachineSpec(in, out, s)
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
		out.IdentityRef = &OpenStackIdentityReference{Name: in.IdentityRef.Name}
		out.CloudName = in.IdentityRef.CloudName
	}

	return nil
}

func Convert_v1beta1_OpenStackMachineStatus_To_v1alpha6_OpenStackMachineStatus(in *infrav1.OpenStackMachineStatus, out *OpenStackMachineStatus, s apiconversion.Scope) error {
	// ReferencedResources have no equivalent in v1alpha6
	return autoConvert_v1beta1_OpenStackMachineStatus_To_v1alpha6_OpenStackMachineStatus(in, out, s)
}

func Convert_v1alpha6_Bastion_To_v1beta1_Bastion(in *Bastion, out *infrav1.Bastion, s apiconversion.Scope) error {
	err := autoConvert_v1alpha6_Bastion_To_v1beta1_Bastion(in, out, s)
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

func Convert_v1beta1_Bastion_To_v1alpha6_Bastion(in *infrav1.Bastion, out *Bastion, s apiconversion.Scope) error {
	err := autoConvert_v1beta1_Bastion_To_v1alpha6_Bastion(in, out, s)
	if err != nil {
		return err
	}

	if in.Instance.ServerGroup != nil && in.Instance.ServerGroup.ID != "" {
		out.Instance.ServerGroupID = in.Instance.ServerGroup.ID
	}

	out.Instance.FloatingIP = in.FloatingIP
	return nil
}

func Convert_v1beta1_SecurityGroupStatus_To_v1alpha6_SecurityGroup(in *infrav1.SecurityGroupStatus, out *SecurityGroup, s apiconversion.Scope) error { //nolint:revive
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

func Convert_v1alpha6_SecurityGroup_To_v1beta1_SecurityGroupStatus(in *SecurityGroup, out *infrav1.SecurityGroupStatus, s apiconversion.Scope) error { //nolint:revive
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

func Convert_v1alpha6_OpenStackIdentityReference_To_v1beta1_OpenStackIdentityReference(in *OpenStackIdentityReference, out *infrav1.OpenStackIdentityReference, s apiconversion.Scope) error {
	return autoConvert_v1alpha6_OpenStackIdentityReference_To_v1beta1_OpenStackIdentityReference(in, out, s)
}

func Convert_v1beta1_OpenStackIdentityReference_To_v1alpha6_OpenStackIdentityReference(in *infrav1.OpenStackIdentityReference, out *OpenStackIdentityReference, _ apiconversion.Scope) error {
	out.Name = in.Name
	return nil
}

func Convert_v1alpha6_SubnetFilter_To_v1beta1_SubnetFilter(in *SubnetFilter, out *infrav1.SubnetFilter, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha6_SubnetFilter_To_v1beta1_SubnetFilter(in, out, s); err != nil {
		return err
	}
	infrav1.ConvertAllTagsTo(in.Tags, in.TagsAny, in.NotTags, in.NotTagsAny, &out.FilterByNeutronTags)
	return nil
}

func Convert_v1beta1_SubnetFilter_To_v1alpha6_SubnetFilter(in *infrav1.SubnetFilter, out *SubnetFilter, s apiconversion.Scope) error {
	if err := autoConvert_v1beta1_SubnetFilter_To_v1alpha6_SubnetFilter(in, out, s); err != nil {
		return err
	}
	infrav1.ConvertAllTagsFrom(&in.FilterByNeutronTags, &out.Tags, &out.TagsAny, &out.NotTags, &out.NotTagsAny)
	return nil
}

func Convert_v1alpha6_SecurityGroupFilter_To_v1beta1_SecurityGroupFilter(in *SecurityGroupFilter, out *infrav1.SecurityGroupFilter, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha6_SecurityGroupFilter_To_v1beta1_SecurityGroupFilter(in, out, s); err != nil {
		return err
	}
	infrav1.ConvertAllTagsTo(in.Tags, in.TagsAny, in.NotTags, in.NotTagsAny, &out.FilterByNeutronTags)

	// TenantID has been removed in v1beta1. Write it to ProjectID if ProjectID is not already set.
	if out.ProjectID == "" {
		out.ProjectID = in.TenantID
	}
	return nil
}

func Convert_v1beta1_SecurityGroupFilter_To_v1alpha6_SecurityGroupFilter(in *infrav1.SecurityGroupFilter, out *SecurityGroupFilter, s apiconversion.Scope) error {
	if err := autoConvert_v1beta1_SecurityGroupFilter_To_v1alpha6_SecurityGroupFilter(in, out, s); err != nil {
		return err
	}
	infrav1.ConvertAllTagsFrom(&in.FilterByNeutronTags, &out.Tags, &out.TagsAny, &out.NotTags, &out.NotTagsAny)
	return nil
}

func Convert_v1alpha6_NetworkFilter_To_v1beta1_NetworkFilter(in *NetworkFilter, out *infrav1.NetworkFilter, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha6_NetworkFilter_To_v1beta1_NetworkFilter(in, out, s); err != nil {
		return err
	}
	infrav1.ConvertAllTagsTo(in.Tags, in.TagsAny, in.NotTags, in.NotTagsAny, &out.FilterByNeutronTags)
	return nil
}

func Convert_v1beta1_NetworkFilter_To_v1alpha6_NetworkFilter(in *infrav1.NetworkFilter, out *NetworkFilter, s apiconversion.Scope) error {
	if err := autoConvert_v1beta1_NetworkFilter_To_v1alpha6_NetworkFilter(in, out, s); err != nil {
		return err
	}
	infrav1.ConvertAllTagsFrom(&in.FilterByNeutronTags, &out.Tags, &out.TagsAny, &out.NotTags, &out.NotTagsAny)
	return nil
}
