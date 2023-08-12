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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	conversion "k8s.io/apimachinery/pkg/conversion"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha7"
)

const trueString = "true"

/*
 * HOW THIS WORKS
 *
 * The problem this solves is when functionality is added or removed from the
 * API in a manner which can't be losslessly converted. For example:
 *
 * v2 of our API adds a new field Foo. When converting from v1 to v2 we allow
 * Foo to be initialised to Foo's zero value. However, if we set the value of
 * Foo in v2 and then convert the object back to v1 it will be lost because
 * there is nowhere to store it.
 *
 * Note that this problem is symmetric on up-conversion and down-conversion. For
 * example, if instead v1 contains Foo but we removed it without replacement in
 * v2. In this case we would lose the value of Foo when up-converting to v2 and
 * converting back to v1.
 *
 * This scheme solves this problem by storing the original object before
 * conversion as an annotation on the converted object. This means that when we
 * convert the object back we can refer to the original object for values which
 * couldn't be converted.
 *
 * convertAndRestore() takes an input parameter src and 2 output parameters: dst
 * and previous. It always converts src to dst. If src had an annotation
 * containing a previously-converted object this is returned in previous. dst
 * always contains 'fresh' values which have directly converted from src.
 * previous should only be used to obtain values which could not be converted
 * but may have been set in the dst version of the object.
 *
 * Restoration of non-converted values is not automatic and must be done
 * explicitly after conversion.
 */

// Convert a source object of type S to dest type D using the provided conversion function.
// Store the original source object in the dest object's annotations.
// Also return any previous version of the object stored in the source object's annotations.
func convertAndRestore[S, D metav1.Object](src S, dst D, previous D, convert func(S, D, conversion.Scope) error) (bool, error) {
	// Restore data from previous conversion except for metadata.
	// We do this before conversion because after convert() src.Annotations
	// will be aliased as dst.Annotations and will therefore be modified by
	// MarshalData below.
	restored, err := utilconversion.UnmarshalData(src, previous)
	if err != nil {
		return false, err
	}

	if err := convert(src, dst, nil); err != nil {
		return false, err
	}

	// Store the original source object in the dest object's annotations.
	if err := utilconversion.MarshalData(src, dst); err != nil {
		return false, err
	}

	return restored, nil
}

func restorev1alpha6MachineSpec(previous *OpenStackMachineSpec, dst *OpenStackMachineSpec) {
	// Subnet is removed from v1alpha7 with no replacement, so can't be
	// losslessly converted. Restore the previously stored value on down-conversion.
	dst.Subnet = previous.Subnet

	// Strictly speaking this is lossy because we lose changes in
	// down-conversion which were made to the up-converted object. However
	// it isn't worth implementing this as the fields are immutable.
	dst.Networks = previous.Networks
	dst.Ports = previous.Ports
	dst.SecurityGroups = previous.SecurityGroups
}

func restorev1alpha6ClusterStatus(previous *OpenStackClusterStatus, dst *OpenStackClusterStatus) {
	// PortOpts.SecurityGroups have been removed in v1alpha7
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

func restorev1alpha7MachineSpec(previous *infrav1.OpenStackMachineSpec, dst *infrav1.OpenStackMachineSpec) {
	// PropagateUplinkStatus has been added in v1alpha7.
	// We restore the whole Ports since they are anyway immutable.
	dst.Ports = previous.Ports
}

func restorev1alpha7ClusterSpec(previous *infrav1.OpenStackClusterSpec, dst *infrav1.OpenStackClusterSpec) {
	dst.Router = previous.Router
	// PropagateUplinkStatus has been added in v1alpha7.
	// We restore the whole Ports since they are anyway immutable.
	if previous.Bastion != nil && previous.Bastion.Instance.Ports != nil {
		dst.Bastion.Instance.Ports = previous.Bastion.Instance.Ports
	}
}

func restorev1alpha7ClusterStatus(previous *infrav1.OpenStackClusterStatus, dst *infrav1.OpenStackClusterStatus) {
	// It's (theoretically) possible in v1alpha7 to have Network nil but
	// Router or APIServerLoadBalancer not nil. In hub-spoke-hub conversion this will
	// result in Network being a pointer to an empty object.
	if previous.Network == nil && dst.Network != nil && reflect.ValueOf(*dst.Network).IsZero() {
		dst.Network = nil
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
}

var _ ctrlconversion.Convertible = &OpenStackCluster{}

func (r *OpenStackCluster) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackCluster)
	var previous infrav1.OpenStackCluster
	restored, err := convertAndRestore(r, dst, &previous, Convert_v1alpha6_OpenStackCluster_To_v1alpha7_OpenStackCluster)
	if err != nil {
		return err
	}

	if restored {
		restorev1alpha7ClusterSpec(&previous.Spec, &dst.Spec)
		restorev1alpha7ClusterStatus(&previous.Status, &dst.Status)
	}

	return nil
}

func (r *OpenStackCluster) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackCluster)
	var previous OpenStackCluster
	restored, err := convertAndRestore(src, r, &previous, Convert_v1alpha7_OpenStackCluster_To_v1alpha6_OpenStackCluster)
	if err != nil {
		return err
	}

	if restored {
		prevBastion := previous.Spec.Bastion
		if prevBastion != nil {
			restorev1alpha6MachineSpec(&prevBastion.Instance, &r.Spec.Bastion.Instance)
		}
		restorev1alpha6ClusterSpec(&previous.Spec, &r.Spec)
		restorev1alpha6ClusterStatus(&previous.Status, &r.Status)
	}

	return nil
}

var _ ctrlconversion.Convertible = &OpenStackClusterList{}

func (r *OpenStackClusterList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1alpha6_OpenStackClusterList_To_v1alpha7_OpenStackClusterList(r, dst, nil)
}

func (r *OpenStackClusterList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1alpha7_OpenStackClusterList_To_v1alpha6_OpenStackClusterList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackClusterTemplate{}

func (r *OpenStackClusterTemplate) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterTemplate)
	var previous infrav1.OpenStackClusterTemplate
	restored, err := convertAndRestore(r, dst, &previous, Convert_v1alpha6_OpenStackClusterTemplate_To_v1alpha7_OpenStackClusterTemplate)
	if err != nil {
		return err
	}

	if restored {
		restorev1alpha7ClusterSpec(&previous.Spec.Template.Spec, &dst.Spec.Template.Spec)
	}

	return nil
}

func (r *OpenStackClusterTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterTemplate)
	var previous OpenStackClusterTemplate
	restored, err := convertAndRestore(src, r, &previous, Convert_v1alpha7_OpenStackClusterTemplate_To_v1alpha6_OpenStackClusterTemplate)
	if err != nil {
		return err
	}

	if restored {
		prevBastion := previous.Spec.Template.Spec.Bastion
		if prevBastion != nil {
			restorev1alpha6MachineSpec(&prevBastion.Instance, &r.Spec.Template.Spec.Bastion.Instance)
		}
		restorev1alpha6ClusterSpec(&previous.Spec.Template.Spec, &r.Spec.Template.Spec)
	}

	return nil
}

var _ ctrlconversion.Convertible = &OpenStackMachine{}

func (r *OpenStackMachine) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachine)
	var previous infrav1.OpenStackMachine
	restored, err := convertAndRestore(r, dst, &previous, Convert_v1alpha6_OpenStackMachine_To_v1alpha7_OpenStackMachine)

	if restored {
		restorev1alpha7MachineSpec(&previous.Spec, &dst.Spec)
	}

	return err
}

func (r *OpenStackMachine) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachine)
	var previous OpenStackMachine
	restored, err := convertAndRestore(src, r, &previous, Convert_v1alpha7_OpenStackMachine_To_v1alpha6_OpenStackMachine)
	if err != nil {
		return err
	}

	if restored {
		restorev1alpha6MachineSpec(&previous.Spec, &r.Spec)
	}

	return err
}

var _ ctrlconversion.Convertible = &OpenStackMachineList{}

func (r *OpenStackMachineList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineList)
	return Convert_v1alpha6_OpenStackMachineList_To_v1alpha7_OpenStackMachineList(r, dst, nil)
}

func (r *OpenStackMachineList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineList)
	return Convert_v1alpha7_OpenStackMachineList_To_v1alpha6_OpenStackMachineList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplate{}

func (r *OpenStackMachineTemplate) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineTemplate)
	var previous infrav1.OpenStackMachineTemplate
	restored, err := convertAndRestore(r, dst, &previous, Convert_v1alpha6_OpenStackMachineTemplate_To_v1alpha7_OpenStackMachineTemplate)

	if restored {
		restorev1alpha7MachineSpec(&previous.Spec.Template.Spec, &dst.Spec.Template.Spec)
	}

	return err
}

func (r *OpenStackMachineTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplate)
	var previous OpenStackMachineTemplate
	restored, err := convertAndRestore(src, r, &previous, Convert_v1alpha7_OpenStackMachineTemplate_To_v1alpha6_OpenStackMachineTemplate)
	if err != nil {
		return err
	}

	if restored {
		restorev1alpha6MachineSpec(&previous.Spec.Template.Spec, &r.Spec.Template.Spec)
	}

	return err
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplateList{}

func (r *OpenStackMachineTemplateList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineTemplateList)
	return Convert_v1alpha6_OpenStackMachineTemplateList_To_v1alpha7_OpenStackMachineTemplateList(r, dst, nil)
}

func (r *OpenStackMachineTemplateList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplateList)
	return Convert_v1alpha7_OpenStackMachineTemplateList_To_v1alpha6_OpenStackMachineTemplateList(src, r, nil)
}

func Convert_v1alpha6_OpenStackMachineSpec_To_v1alpha7_OpenStackMachineSpec(in *OpenStackMachineSpec, out *infrav1.OpenStackMachineSpec, s conversion.Scope) error {
	err := autoConvert_v1alpha6_OpenStackMachineSpec_To_v1alpha7_OpenStackMachineSpec(in, out, s)
	if err != nil {
		return err
	}

	if len(in.Networks) > 0 {
		ports := convertNetworksToPorts(in.Networks)
		// Networks were previously created first, so need to come before ports
		out.Ports = append(ports, out.Ports...)
	}
	return nil
}

func convertNetworksToPorts(networks []NetworkParam) []infrav1.PortOpts {
	var ports []infrav1.PortOpts

	for _, network := range networks {
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
			for _, subnet := range network.Subnets {
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

func Convert_v1alpha7_OpenStackClusterSpec_To_v1alpha6_OpenStackClusterSpec(in *infrav1.OpenStackClusterSpec, out *OpenStackClusterSpec, s conversion.Scope) error {
	return autoConvert_v1alpha7_OpenStackClusterSpec_To_v1alpha6_OpenStackClusterSpec(in, out, s)
}

func Convert_v1alpha6_PortOpts_To_v1alpha7_PortOpts(in *PortOpts, out *infrav1.PortOpts, s conversion.Scope) error {
	err := autoConvert_v1alpha6_PortOpts_To_v1alpha7_PortOpts(in, out, s)
	if err != nil {
		return err
	}
	// SecurityGroups are removed in v1alpha7 without replacement. SecurityGroupFilters can be used instead.
	for i := range in.SecurityGroups {
		out.SecurityGroupFilters = append(out.SecurityGroupFilters, infrav1.SecurityGroupFilter{ID: in.SecurityGroups[i]})
	}

	// Profile is now a struct in v1alpha7.
	if strings.Contains(in.Profile["capabilities"], "switchdev") {
		out.Profile.OVSHWOffload = true
	}
	if in.Profile["trusted"] == trueString {
		out.Profile.TrustedVF = true
	}
	return nil
}

func Convert_v1alpha7_PortOpts_To_v1alpha6_PortOpts(in *infrav1.PortOpts, out *PortOpts, s conversion.Scope) error {
	// value specs and propagate uplink status have been added in v1alpha7 but have no equivalent in v1alpha5
	err := autoConvert_v1alpha7_PortOpts_To_v1alpha6_PortOpts(in, out, s)
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

func Convert_v1alpha6_Instance_To_v1alpha7_BastionStatus(in *Instance, out *infrav1.BastionStatus, _ conversion.Scope) error {
	// BastionStatus is the same as Instance with unused fields removed
	out.ID = in.ID
	out.Name = in.Name
	out.SSHKeyName = in.SSHKeyName
	out.State = infrav1.InstanceState(in.State)
	out.IP = in.IP
	out.FloatingIP = in.FloatingIP
	return nil
}

func Convert_v1alpha7_BastionStatus_To_v1alpha6_Instance(in *infrav1.BastionStatus, out *Instance, _ conversion.Scope) error {
	// BastionStatus is the same as Instance with unused fields removed
	out.ID = in.ID
	out.Name = in.Name
	out.SSHKeyName = in.SSHKeyName
	out.State = InstanceState(in.State)
	out.IP = in.IP
	out.FloatingIP = in.FloatingIP
	return nil
}

func Convert_v1alpha6_Network_To_v1alpha7_NetworkStatusWithSubnets(in *Network, out *infrav1.NetworkStatusWithSubnets, s conversion.Scope) error {
	// PortOpts has been removed in v1alpha7
	err := Convert_v1alpha6_Network_To_v1alpha7_NetworkStatus(in, &out.NetworkStatus, s)
	if err != nil {
		return err
	}

	if in.Subnet != nil {
		out.Subnets = []infrav1.Subnet{infrav1.Subnet(*in.Subnet)}
	}
	return nil
}

func Convert_v1alpha7_NetworkStatusWithSubnets_To_v1alpha6_Network(in *infrav1.NetworkStatusWithSubnets, out *Network, s conversion.Scope) error {
	// PortOpts has been removed in v1alpha7
	err := Convert_v1alpha7_NetworkStatus_To_v1alpha6_Network(&in.NetworkStatus, out, s)
	if err != nil {
		return err
	}

	// Can only down-convert a single subnet
	if len(in.Subnets) > 0 {
		out.Subnet = (*Subnet)(&in.Subnets[0])
	}
	return nil
}

func Convert_v1alpha6_Network_To_v1alpha7_NetworkStatus(in *Network, out *infrav1.NetworkStatus, _ conversion.Scope) error {
	out.ID = in.ID
	out.Name = in.Name
	out.Tags = in.Tags

	return nil
}

func Convert_v1alpha7_NetworkStatus_To_v1alpha6_Network(in *infrav1.NetworkStatus, out *Network, _ conversion.Scope) error {
	out.ID = in.ID
	out.Name = in.Name
	out.Tags = in.Tags

	return nil
}

func Convert_v1alpha6_SecurityGroupFilter_To_v1alpha7_SecurityGroupFilter(in *SecurityGroupFilter, out *infrav1.SecurityGroupFilter, s conversion.Scope) error {
	err := autoConvert_v1alpha6_SecurityGroupFilter_To_v1alpha7_SecurityGroupFilter(in, out, s)
	if err != nil {
		return err
	}

	// TenantID has been removed in v1alpha7. Write it to ProjectID if ProjectID is not already set.
	if out.ProjectID == "" {
		out.ProjectID = in.TenantID
	}

	return nil
}

func Convert_v1alpha6_SecurityGroupParam_To_v1alpha7_SecurityGroupFilter(in *SecurityGroupParam, out *infrav1.SecurityGroupFilter, s conversion.Scope) error {
	// SecurityGroupParam is replaced by its contained SecurityGroupFilter in v1alpha7
	err := Convert_v1alpha6_SecurityGroupFilter_To_v1alpha7_SecurityGroupFilter(&in.Filter, out, s)
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

func Convert_v1alpha7_SecurityGroupFilter_To_v1alpha6_SecurityGroupParam(in *infrav1.SecurityGroupFilter, out *SecurityGroupParam, s conversion.Scope) error {
	// SecurityGroupParam is replaced by its contained SecurityGroupFilter in v1alpha7
	err := Convert_v1alpha7_SecurityGroupFilter_To_v1alpha6_SecurityGroupFilter(in, &out.Filter, s)
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

func Convert_v1alpha6_SubnetParam_To_v1alpha7_SubnetFilter(in *SubnetParam, out *infrav1.SubnetFilter, _ conversion.Scope) error {
	*out = infrav1.SubnetFilter(in.Filter)
	if in.UUID != "" {
		out.ID = in.UUID
	}
	return nil
}

func Convert_v1alpha7_SubnetFilter_To_v1alpha6_SubnetParam(in *infrav1.SubnetFilter, out *SubnetParam, _ conversion.Scope) error {
	out.Filter = SubnetFilter(*in)
	out.UUID = in.ID

	return nil
}

func Convert_Map_string_To_Interface_To_v1alpha7_BindingProfile(in map[string]string, out *infrav1.BindingProfile, _ conversion.Scope) error {
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

func Convert_v1alpha7_BindingProfile_To_Map_string_To_Interface(in *infrav1.BindingProfile, out map[string]string, _ conversion.Scope) error {
	if in.OVSHWOffload {
		(out)["capabilities"] = "[\"switchdev\"]"
	}
	if in.TrustedVF {
		(out)["trusted"] = trueString
	}
	return nil
}

func Convert_v1alpha7_OpenStackClusterStatus_To_v1alpha6_OpenStackClusterStatus(in *infrav1.OpenStackClusterStatus, out *OpenStackClusterStatus, s conversion.Scope) error {
	err := autoConvert_v1alpha7_OpenStackClusterStatus_To_v1alpha6_OpenStackClusterStatus(in, out, s)
	if err != nil {
		return err
	}

	// Router and APIServerLoadBalancer have been moved out of Network in v1alpha7
	if in.Router != nil || in.APIServerLoadBalancer != nil {
		if out.Network == nil {
			out.Network = &Network{}
		}

		out.Network.Router = (*Router)(in.Router)
		out.Network.APIServerLoadBalancer = (*LoadBalancer)(in.APIServerLoadBalancer)
	}

	return nil
}

func Convert_v1alpha6_OpenStackClusterStatus_To_v1alpha7_OpenStackClusterStatus(in *OpenStackClusterStatus, out *infrav1.OpenStackClusterStatus, s conversion.Scope) error {
	err := autoConvert_v1alpha6_OpenStackClusterStatus_To_v1alpha7_OpenStackClusterStatus(in, out, s)
	if err != nil {
		return err
	}

	// Router and APIServerLoadBalancer have been moved out of Network in v1alpha7
	if in.Network != nil {
		out.Router = (*infrav1.Router)(in.Network.Router)
		out.APIServerLoadBalancer = (*infrav1.LoadBalancer)(in.Network.APIServerLoadBalancer)
	}

	return nil
}
