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
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha8"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/conversion"
)

const trueString = "true"

func restorev1alpha6MachineSpec(previous *OpenStackMachineSpec, dst *OpenStackMachineSpec) {
	// Subnet is removed from v1alpha8 with no replacement, so can't be
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

func restorev1alpha6ClusterStatus(previous *OpenStackClusterStatus, dst *OpenStackClusterStatus) {
	// PortOpts.SecurityGroups have been removed in v1alpha8
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
}

func restorev1alpha8MachineSpec(previous *infrav1.OpenStackMachineSpec, dst *infrav1.OpenStackMachineSpec) {
	// PropagateUplinkStatus has been added in v1alpha8.
	// We restore the whole Ports since they are anyway immutable.
	dst.Ports = previous.Ports
	dst.AdditionalBlockDevices = previous.AdditionalBlockDevices
	dst.ServerGroup = previous.ServerGroup
	dst.Image = previous.Image
}

func restorev1alpha8Bastion(previous **infrav1.Bastion, dst **infrav1.Bastion) {
	if *previous != nil && *dst != nil {
		restorev1alpha8MachineSpec(&(*previous).Instance, &(*dst).Instance)
	}
}

func restorev1alpha8Subnets(previous *[]infrav1.SubnetFilter, dst *[]infrav1.SubnetFilter) {
	if len(*previous) > 1 {
		*dst = append(*dst, (*previous)[1:]...)
	}
}

func restorev1alpha8ClusterStatus(previous *infrav1.OpenStackClusterStatus, dst *infrav1.OpenStackClusterStatus) {
	// It's (theoretically) possible in v1alpha8 to have Network nil but
	// Router or APIServerLoadBalancer not nil. In hub-spoke-hub conversion this will
	// result in Network being a pointer to an empty object.
	if previous.Network == nil && dst.Network != nil && reflect.ValueOf(*dst.Network).IsZero() {
		dst.Network = nil
	}

	if previous.Bastion != nil {
		dst.Bastion.ReferencedResources = previous.Bastion.ReferencedResources
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

	prevBastion := previous.Bastion
	dstBastion := dst.Bastion
	if prevBastion != nil && dstBastion != nil {
		restorev1alpha6MachineSpec(&prevBastion.Instance, &dstBastion.Instance)
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

var v1alpha8OpenStackClusterRestorer = conversion.RestorerFor[*infrav1.OpenStackCluster]{
	"externalNetwork": conversion.UnconditionalFieldRestorer(
		func(c *infrav1.OpenStackCluster) *infrav1.NetworkFilter {
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
		func(c *infrav1.OpenStackCluster) *int {
			return &c.Spec.NetworkMTU
		},
	),
	"bastion": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackCluster) **infrav1.Bastion {
			return &c.Spec.Bastion
		},
		restorev1alpha8Bastion,
	),
	"subnets": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackCluster) *[]infrav1.SubnetFilter {
			return &c.Spec.Subnets
		},
		restorev1alpha8Subnets,
	),
	"status": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackCluster) *infrav1.OpenStackClusterStatus {
			return &c.Status
		},
		restorev1alpha8ClusterStatus,
	),
}

func (r *OpenStackCluster) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackCluster)

	return conversion.ConvertAndRestore(
		r, dst,
		Convert_v1alpha6_OpenStackCluster_To_v1alpha8_OpenStackCluster, Convert_v1alpha8_OpenStackCluster_To_v1alpha6_OpenStackCluster,
		v1alpha6OpenStackClusterRestorer, v1alpha8OpenStackClusterRestorer,
	)
}

func (r *OpenStackCluster) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackCluster)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1alpha8_OpenStackCluster_To_v1alpha6_OpenStackCluster, Convert_v1alpha6_OpenStackCluster_To_v1alpha8_OpenStackCluster,
		v1alpha8OpenStackClusterRestorer, v1alpha6OpenStackClusterRestorer,
	)
}

var _ ctrlconversion.Convertible = &OpenStackClusterList{}

func (r *OpenStackClusterList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1alpha6_OpenStackClusterList_To_v1alpha8_OpenStackClusterList(r, dst, nil)
}

func (r *OpenStackClusterList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1alpha8_OpenStackClusterList_To_v1alpha6_OpenStackClusterList(src, r, nil)
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

var v1alpha8OpenStackClusterTemplateRestorer = conversion.RestorerFor[*infrav1.OpenStackClusterTemplate]{
	"externalNetwork": conversion.UnconditionalFieldRestorer(
		func(c *infrav1.OpenStackClusterTemplate) *infrav1.NetworkFilter {
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
		func(c *infrav1.OpenStackClusterTemplate) *int {
			return &c.Spec.Template.Spec.NetworkMTU
		},
	),
	"bastion": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackClusterTemplate) **infrav1.Bastion {
			return &c.Spec.Template.Spec.Bastion
		},
		restorev1alpha8Bastion,
	),
	"subnets": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackClusterTemplate) *[]infrav1.SubnetFilter {
			return &c.Spec.Template.Spec.Subnets
		},
		restorev1alpha8Subnets,
	),
}

func (r *OpenStackClusterTemplate) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterTemplate)

	return conversion.ConvertAndRestore(
		r, dst,
		Convert_v1alpha6_OpenStackClusterTemplate_To_v1alpha8_OpenStackClusterTemplate, Convert_v1alpha8_OpenStackClusterTemplate_To_v1alpha6_OpenStackClusterTemplate,
		v1alpha6OpenStackClusterTemplateRestorer, v1alpha8OpenStackClusterTemplateRestorer,
	)
}

func (r *OpenStackClusterTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterTemplate)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1alpha8_OpenStackClusterTemplate_To_v1alpha6_OpenStackClusterTemplate, Convert_v1alpha6_OpenStackClusterTemplate_To_v1alpha8_OpenStackClusterTemplate,
		v1alpha8OpenStackClusterTemplateRestorer, v1alpha6OpenStackClusterTemplateRestorer,
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

var v1alpha8OpenStackMachineRestorer = conversion.RestorerFor[*infrav1.OpenStackMachine]{
	"spec": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackMachine) *infrav1.OpenStackMachineSpec {
			return &c.Spec
		},
		restorev1alpha8MachineSpec,
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
		Convert_v1alpha6_OpenStackMachine_To_v1alpha8_OpenStackMachine, Convert_v1alpha8_OpenStackMachine_To_v1alpha6_OpenStackMachine,
		v1alpha6OpenStackMachineRestorer, v1alpha8OpenStackMachineRestorer,
	)
}

func (r *OpenStackMachine) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachine)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1alpha8_OpenStackMachine_To_v1alpha6_OpenStackMachine, Convert_v1alpha6_OpenStackMachine_To_v1alpha8_OpenStackMachine,
		v1alpha8OpenStackMachineRestorer, v1alpha6OpenStackMachineRestorer,
	)
}

var _ ctrlconversion.Convertible = &OpenStackMachineList{}

func (r *OpenStackMachineList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineList)
	return Convert_v1alpha6_OpenStackMachineList_To_v1alpha8_OpenStackMachineList(r, dst, nil)
}

func (r *OpenStackMachineList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineList)
	return Convert_v1alpha8_OpenStackMachineList_To_v1alpha6_OpenStackMachineList(src, r, nil)
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
		Convert_v1alpha6_OpenStackMachineTemplate_To_v1alpha8_OpenStackMachineTemplate, Convert_v1alpha8_OpenStackMachineTemplate_To_v1alpha6_OpenStackMachineTemplate,
		v1alpha6OpenStackMachineTemplateRestorer, v1alpha8OpenStackMachineTemplateRestorer,
	)
}

func (r *OpenStackMachineTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplate)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1alpha8_OpenStackMachineTemplate_To_v1alpha6_OpenStackMachineTemplate, Convert_v1alpha6_OpenStackMachineTemplate_To_v1alpha8_OpenStackMachineTemplate,
		v1alpha8OpenStackMachineTemplateRestorer, v1alpha6OpenStackMachineTemplateRestorer,
	)
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplateList{}

func (r *OpenStackMachineTemplateList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineTemplateList)
	return Convert_v1alpha6_OpenStackMachineTemplateList_To_v1alpha8_OpenStackMachineTemplateList(r, dst, nil)
}

func (r *OpenStackMachineTemplateList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplateList)
	return Convert_v1alpha8_OpenStackMachineTemplateList_To_v1alpha6_OpenStackMachineTemplateList(src, r, nil)
}

func Convert_v1alpha6_OpenStackMachineSpec_To_v1alpha8_OpenStackMachineSpec(in *OpenStackMachineSpec, out *infrav1.OpenStackMachineSpec, s apiconversion.Scope) error {
	err := autoConvert_v1alpha6_OpenStackMachineSpec_To_v1alpha8_OpenStackMachineSpec(in, out, s)
	if err != nil {
		return err
	}

	if len(in.Networks) > 0 {
		ports := convertNetworksToPorts(in.Networks)
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

func convertNetworksToPorts(networks []NetworkParam) []infrav1.PortOpts {
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
			networkFilter = (*infrav1.NetworkFilter)(&network.Filter)
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
					ports = append(ports, infrav1.PortOpts{
						Network: networkFilter,
						FixedIPs: []infrav1.FixedIP{
							{Subnet: (*infrav1.SubnetFilter)(&subnet.Filter)},
						},
					})
				}
			}
		}
	}

	return ports
}

func Convert_v1alpha8_OpenStackClusterSpec_To_v1alpha6_OpenStackClusterSpec(in *infrav1.OpenStackClusterSpec, out *OpenStackClusterSpec, s apiconversion.Scope) error {
	err := autoConvert_v1alpha8_OpenStackClusterSpec_To_v1alpha6_OpenStackClusterSpec(in, out, s)
	if err != nil {
		return err
	}

	if in.ExternalNetwork.ID != "" {
		out.ExternalNetworkID = in.ExternalNetwork.ID
	}

	if len(in.Subnets) >= 1 {
		if err := Convert_v1alpha8_SubnetFilter_To_v1alpha6_SubnetFilter(&in.Subnets[0], &out.Subnet, s); err != nil {
			return err
		}
	}

	return nil
}

func Convert_v1alpha6_OpenStackClusterSpec_To_v1alpha8_OpenStackClusterSpec(in *OpenStackClusterSpec, out *infrav1.OpenStackClusterSpec, s apiconversion.Scope) error {
	err := autoConvert_v1alpha6_OpenStackClusterSpec_To_v1alpha8_OpenStackClusterSpec(in, out, s)
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
		if err := Convert_v1alpha6_SubnetFilter_To_v1alpha8_SubnetFilter(&in.Subnet, &subnet, s); err != nil {
			return err
		}
		out.Subnets = []infrav1.SubnetFilter{subnet}
	}

	return nil
}

func Convert_v1alpha6_PortOpts_To_v1alpha8_PortOpts(in *PortOpts, out *infrav1.PortOpts, s apiconversion.Scope) error {
	err := autoConvert_v1alpha6_PortOpts_To_v1alpha8_PortOpts(in, out, s)
	if err != nil {
		return err
	}
	// SecurityGroups are removed in v1alpha8 without replacement. SecurityGroupFilters can be used instead.
	for i := range in.SecurityGroups {
		out.SecurityGroupFilters = append(out.SecurityGroupFilters, infrav1.SecurityGroupFilter{ID: in.SecurityGroups[i]})
	}

	// Profile is now a struct in v1alpha8.
	if strings.Contains(in.Profile["capabilities"], "switchdev") {
		out.Profile.OVSHWOffload = true
	}
	if in.Profile["trusted"] == trueString {
		out.Profile.TrustedVF = true
	}
	return nil
}

func Convert_v1alpha8_PortOpts_To_v1alpha6_PortOpts(in *infrav1.PortOpts, out *PortOpts, s apiconversion.Scope) error {
	// value specs and propagate uplink status have been added in v1alpha8 but have no equivalent in v1alpha5
	err := autoConvert_v1alpha8_PortOpts_To_v1alpha6_PortOpts(in, out, s)
	if err != nil {
		return err
	}

	out.Profile = make(map[string]string)
	if in.Profile.OVSHWOffload {
		(out.Profile)["capabilities"] = "[\"switchdev\"]"
	}
	if in.Profile.TrustedVF {
		(out.Profile)["trusted"] = trueString
	}
	return nil
}

func Convert_v1alpha6_Instance_To_v1alpha8_BastionStatus(in *Instance, out *infrav1.BastionStatus, _ apiconversion.Scope) error {
	// BastionStatus is the same as Instance with unused fields removed
	out.ID = in.ID
	out.Name = in.Name
	out.SSHKeyName = in.SSHKeyName
	out.State = infrav1.InstanceState(in.State)
	out.IP = in.IP
	out.FloatingIP = in.FloatingIP
	return nil
}

func Convert_v1alpha8_BastionStatus_To_v1alpha6_Instance(in *infrav1.BastionStatus, out *Instance, _ apiconversion.Scope) error {
	// BastionStatus is the same as Instance with unused fields removed
	out.ID = in.ID
	out.Name = in.Name
	out.SSHKeyName = in.SSHKeyName
	out.State = InstanceState(in.State)
	out.IP = in.IP
	out.FloatingIP = in.FloatingIP
	return nil
}

func Convert_v1alpha6_Network_To_v1alpha8_NetworkStatusWithSubnets(in *Network, out *infrav1.NetworkStatusWithSubnets, s apiconversion.Scope) error {
	// PortOpts has been removed in v1alpha8
	err := Convert_v1alpha6_Network_To_v1alpha8_NetworkStatus(in, &out.NetworkStatus, s)
	if err != nil {
		return err
	}

	if in.Subnet != nil {
		out.Subnets = []infrav1.Subnet{infrav1.Subnet(*in.Subnet)}
	}
	return nil
}

func Convert_v1alpha8_NetworkStatusWithSubnets_To_v1alpha6_Network(in *infrav1.NetworkStatusWithSubnets, out *Network, s apiconversion.Scope) error {
	// PortOpts has been removed in v1alpha8
	err := Convert_v1alpha8_NetworkStatus_To_v1alpha6_Network(&in.NetworkStatus, out, s)
	if err != nil {
		return err
	}

	// Can only down-convert a single subnet
	if len(in.Subnets) > 0 {
		out.Subnet = (*Subnet)(&in.Subnets[0])
	}
	return nil
}

func Convert_v1alpha6_Network_To_v1alpha8_NetworkStatus(in *Network, out *infrav1.NetworkStatus, _ apiconversion.Scope) error {
	out.ID = in.ID
	out.Name = in.Name
	out.Tags = in.Tags

	return nil
}

func Convert_v1alpha8_NetworkStatus_To_v1alpha6_Network(in *infrav1.NetworkStatus, out *Network, _ apiconversion.Scope) error {
	out.ID = in.ID
	out.Name = in.Name
	out.Tags = in.Tags

	return nil
}

func Convert_v1alpha6_SecurityGroupFilter_To_v1alpha8_SecurityGroupFilter(in *SecurityGroupFilter, out *infrav1.SecurityGroupFilter, s apiconversion.Scope) error {
	err := autoConvert_v1alpha6_SecurityGroupFilter_To_v1alpha8_SecurityGroupFilter(in, out, s)
	if err != nil {
		return err
	}

	// TenantID has been removed in v1alpha8. Write it to ProjectID if ProjectID is not already set.
	if out.ProjectID == "" {
		out.ProjectID = in.TenantID
	}

	return nil
}

func Convert_v1alpha6_SecurityGroupParam_To_v1alpha8_SecurityGroupFilter(in *SecurityGroupParam, out *infrav1.SecurityGroupFilter, s apiconversion.Scope) error {
	// SecurityGroupParam is replaced by its contained SecurityGroupFilter in v1alpha8
	err := Convert_v1alpha6_SecurityGroupFilter_To_v1alpha8_SecurityGroupFilter(&in.Filter, out, s)
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

func Convert_v1alpha8_SecurityGroupFilter_To_v1alpha6_SecurityGroupParam(in *infrav1.SecurityGroupFilter, out *SecurityGroupParam, s apiconversion.Scope) error {
	// SecurityGroupParam is replaced by its contained SecurityGroupFilter in v1alpha8
	err := Convert_v1alpha8_SecurityGroupFilter_To_v1alpha6_SecurityGroupFilter(in, &out.Filter, s)
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

func Convert_v1alpha6_SubnetParam_To_v1alpha8_SubnetFilter(in *SubnetParam, out *infrav1.SubnetFilter, _ apiconversion.Scope) error {
	*out = infrav1.SubnetFilter(in.Filter)
	if in.UUID != "" {
		out.ID = in.UUID
	}
	return nil
}

func Convert_v1alpha8_SubnetFilter_To_v1alpha6_SubnetParam(in *infrav1.SubnetFilter, out *SubnetParam, _ apiconversion.Scope) error {
	out.Filter = SubnetFilter(*in)
	out.UUID = in.ID

	return nil
}

func Convert_Map_string_To_Interface_To_v1alpha8_BindingProfile(in map[string]string, out *infrav1.BindingProfile, _ apiconversion.Scope) error {
	for k, v := range in {
		if k == "capabilities" {
			if strings.Contains(v, "switchdev") {
				out.OVSHWOffload = true
			}
		}
		if k == "trusted" && v == trueString {
			out.TrustedVF = true
		}
	}
	return nil
}

func Convert_v1alpha8_BindingProfile_To_Map_string_To_Interface(in *infrav1.BindingProfile, out map[string]string, _ apiconversion.Scope) error {
	if in.OVSHWOffload {
		(out)["capabilities"] = "[\"switchdev\"]"
	}
	if in.TrustedVF {
		(out)["trusted"] = trueString
	}
	return nil
}

func Convert_v1alpha8_OpenStackClusterStatus_To_v1alpha6_OpenStackClusterStatus(in *infrav1.OpenStackClusterStatus, out *OpenStackClusterStatus, s apiconversion.Scope) error {
	err := autoConvert_v1alpha8_OpenStackClusterStatus_To_v1alpha6_OpenStackClusterStatus(in, out, s)
	if err != nil {
		return err
	}

	// Router and APIServerLoadBalancer have been moved out of Network in v1alpha8
	if in.Router != nil || in.APIServerLoadBalancer != nil {
		if out.Network == nil {
			out.Network = &Network{}
		}

		out.Network.Router = (*Router)(in.Router)
		out.Network.APIServerLoadBalancer = (*LoadBalancer)(in.APIServerLoadBalancer)
	}

	return nil
}

func Convert_v1alpha6_OpenStackClusterStatus_To_v1alpha8_OpenStackClusterStatus(in *OpenStackClusterStatus, out *infrav1.OpenStackClusterStatus, s apiconversion.Scope) error {
	err := autoConvert_v1alpha6_OpenStackClusterStatus_To_v1alpha8_OpenStackClusterStatus(in, out, s)
	if err != nil {
		return err
	}

	// Router and APIServerLoadBalancer have been moved out of Network in v1alpha8
	if in.Network != nil {
		out.Router = (*infrav1.Router)(in.Network.Router)
		out.APIServerLoadBalancer = (*infrav1.LoadBalancer)(in.Network.APIServerLoadBalancer)
	}

	return nil
}

func Convert_v1alpha8_OpenStackMachineSpec_To_v1alpha6_OpenStackMachineSpec(in *infrav1.OpenStackMachineSpec, out *OpenStackMachineSpec, s apiconversion.Scope) error {
	err := autoConvert_v1alpha8_OpenStackMachineSpec_To_v1alpha6_OpenStackMachineSpec(in, out, s)
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

func Convert_v1alpha8_OpenStackMachineStatus_To_v1alpha6_OpenStackMachineStatus(in *infrav1.OpenStackMachineStatus, out *OpenStackMachineStatus, s apiconversion.Scope) error {
	// ReferencedResources have no equivalent in v1alpha6
	return autoConvert_v1alpha8_OpenStackMachineStatus_To_v1alpha6_OpenStackMachineStatus(in, out, s)
}

func Convert_v1alpha6_Bastion_To_v1alpha8_Bastion(in *Bastion, out *infrav1.Bastion, s apiconversion.Scope) error {
	err := autoConvert_v1alpha6_Bastion_To_v1alpha8_Bastion(in, out, s)
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

func Convert_v1alpha8_Bastion_To_v1alpha6_Bastion(in *infrav1.Bastion, out *Bastion, s apiconversion.Scope) error {
	err := autoConvert_v1alpha8_Bastion_To_v1alpha6_Bastion(in, out, s)
	if err != nil {
		return err
	}

	if in.Instance.ServerGroup != nil && in.Instance.ServerGroup.ID != "" {
		out.Instance.ServerGroupID = in.Instance.ServerGroup.ID
	}

	out.Instance.FloatingIP = in.FloatingIP
	return nil
}
