/*
Copyright 2021 The Kubernetes Authors.

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

package v1alpha5

import (
	"strings"

	conversion "k8s.io/apimachinery/pkg/conversion"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha8"
)

var _ ctrlconversion.Convertible = &OpenStackCluster{}

const trueString = "true"

func (r *OpenStackCluster) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackCluster)

	if err := Convert_v1alpha5_OpenStackCluster_To_v1alpha8_OpenStackCluster(r, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &infrav1.OpenStackCluster{}
	if ok, err := utilconversion.UnmarshalData(r, restored); err != nil || !ok {
		return err
	}

	return nil
}

func (r *OpenStackCluster) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackCluster)

	if err := Convert_v1alpha8_OpenStackCluster_To_v1alpha5_OpenStackCluster(src, r, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata
	return utilconversion.MarshalData(src, r)
}

var _ ctrlconversion.Convertible = &OpenStackClusterList{}

func (r *OpenStackClusterList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1alpha5_OpenStackClusterList_To_v1alpha8_OpenStackClusterList(r, dst, nil)
}

func (r *OpenStackClusterList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1alpha8_OpenStackClusterList_To_v1alpha5_OpenStackClusterList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackClusterTemplate{}

func (r *OpenStackClusterTemplate) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterTemplate)

	if err := Convert_v1alpha5_OpenStackClusterTemplate_To_v1alpha8_OpenStackClusterTemplate(r, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &infrav1.OpenStackClusterTemplate{}
	if ok, err := utilconversion.UnmarshalData(r, restored); err != nil || !ok {
		return err
	}

	return nil
}

func (r *OpenStackClusterTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterTemplate)

	if err := Convert_v1alpha8_OpenStackClusterTemplate_To_v1alpha5_OpenStackClusterTemplate(src, r, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata
	return utilconversion.MarshalData(src, r)
}

var _ ctrlconversion.Convertible = &OpenStackMachine{}

func (r *OpenStackMachine) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachine)

	if err := Convert_v1alpha5_OpenStackMachine_To_v1alpha8_OpenStackMachine(r, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &infrav1.OpenStackMachine{}
	if ok, err := utilconversion.UnmarshalData(r, restored); err != nil || !ok {
		return err
	}

	return nil
}

func (r *OpenStackMachine) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachine)

	if err := Convert_v1alpha8_OpenStackMachine_To_v1alpha5_OpenStackMachine(src, r, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata
	return utilconversion.MarshalData(src, r)
}

var _ ctrlconversion.Convertible = &OpenStackMachineList{}

func (r *OpenStackMachineList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineList)

	return Convert_v1alpha5_OpenStackMachineList_To_v1alpha8_OpenStackMachineList(r, dst, nil)
}

func (r *OpenStackMachineList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineList)

	return Convert_v1alpha8_OpenStackMachineList_To_v1alpha5_OpenStackMachineList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplate{}

func (r *OpenStackMachineTemplate) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineTemplate)

	if err := Convert_v1alpha5_OpenStackMachineTemplate_To_v1alpha8_OpenStackMachineTemplate(r, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &infrav1.OpenStackMachineTemplate{}
	if ok, err := utilconversion.UnmarshalData(r, restored); err != nil || !ok {
		return err
	}

	return nil
}

func (r *OpenStackMachineTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplate)

	if err := Convert_v1alpha8_OpenStackMachineTemplate_To_v1alpha5_OpenStackMachineTemplate(src, r, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata
	return utilconversion.MarshalData(src, r)
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplateList{}

func (r *OpenStackMachineTemplateList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineTemplateList)

	return Convert_v1alpha5_OpenStackMachineTemplateList_To_v1alpha8_OpenStackMachineTemplateList(r, dst, nil)
}

func (r *OpenStackMachineTemplateList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplateList)

	return Convert_v1alpha8_OpenStackMachineTemplateList_To_v1alpha5_OpenStackMachineTemplateList(src, r, nil)
}

func Convert_v1alpha8_OpenStackClusterSpec_To_v1alpha5_OpenStackClusterSpec(in *infrav1.OpenStackClusterSpec, out *OpenStackClusterSpec, s conversion.Scope) error {
	err := autoConvert_v1alpha8_OpenStackClusterSpec_To_v1alpha5_OpenStackClusterSpec(in, out, s)
	if err != nil {
		return err
	}

	if in.ExternalNetwork.ID != "" {
		out.ExternalNetworkID = in.ExternalNetwork.ID
	}

	return nil
}

func Convert_v1alpha5_OpenStackClusterSpec_To_v1alpha8_OpenStackClusterSpec(in *OpenStackClusterSpec, out *infrav1.OpenStackClusterSpec, s conversion.Scope) error {
	err := autoConvert_v1alpha5_OpenStackClusterSpec_To_v1alpha8_OpenStackClusterSpec(in, out, s)
	if err != nil {
		return err
	}

	if in.ExternalNetworkID != "" {
		out.ExternalNetwork = infrav1.NetworkFilter{
			ID: in.ExternalNetworkID,
		}
	}

	return nil
}

func Convert_v1alpha8_LoadBalancer_To_v1alpha5_LoadBalancer(in *infrav1.LoadBalancer, out *LoadBalancer, s conversion.Scope) error {
	return autoConvert_v1alpha8_LoadBalancer_To_v1alpha5_LoadBalancer(in, out, s)
}

func Convert_v1alpha8_PortOpts_To_v1alpha5_PortOpts(in *infrav1.PortOpts, out *PortOpts, s conversion.Scope) error {
	// value specs and propagate uplink status have been added in v1alpha8 but have no equivalent in v1alpha5
	err := autoConvert_v1alpha8_PortOpts_To_v1alpha5_PortOpts(in, out, s)
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

func Convert_v1alpha5_OpenStackMachineSpec_To_v1alpha8_OpenStackMachineSpec(in *OpenStackMachineSpec, out *infrav1.OpenStackMachineSpec, s conversion.Scope) error {
	err := autoConvert_v1alpha5_OpenStackMachineSpec_To_v1alpha8_OpenStackMachineSpec(in, out, s)
	if err != nil {
		return err
	}

	if in.ServerGroupID != "" {
		out.ServerGroup = &infrav1.ServerGroupFilter{ID: in.ServerGroupID}
	} else {
		out.ServerGroup = &infrav1.ServerGroupFilter{}
	}

	return nil
}

func Convert_v1alpha8_APIServerLoadBalancer_To_v1alpha5_APIServerLoadBalancer(in *infrav1.APIServerLoadBalancer, out *APIServerLoadBalancer, s conversion.Scope) error {
	// Provider was originally added in v1alpha8, but was backported to v1alpha6, but has no equivalent in v1alpha5
	return autoConvert_v1alpha8_APIServerLoadBalancer_To_v1alpha5_APIServerLoadBalancer(in, out, s)
}

func Convert_v1alpha5_PortOpts_To_v1alpha8_PortOpts(in *PortOpts, out *infrav1.PortOpts, s conversion.Scope) error {
	// SecurityGroups have been removed in v1alpha8.
	err := autoConvert_v1alpha5_PortOpts_To_v1alpha8_PortOpts(in, out, s)
	if err != nil {
		return err
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

func Convert_v1alpha5_Instance_To_v1alpha8_BastionStatus(in *Instance, out *infrav1.BastionStatus, _ conversion.Scope) error {
	// BastionStatus is the same as Instance with unused fields removed
	out.ID = in.ID
	out.Name = in.Name
	out.SSHKeyName = in.SSHKeyName
	out.State = infrav1.InstanceState(in.State)
	out.IP = in.IP
	out.FloatingIP = in.FloatingIP
	out.ReferencedResources.ServerGroupID = in.ServerGroupID
	return nil
}

func Convert_v1alpha8_BastionStatus_To_v1alpha5_Instance(in *infrav1.BastionStatus, out *Instance, _ conversion.Scope) error {
	// BastionStatus is the same as Instance with unused fields removed
	out.ID = in.ID
	out.Name = in.Name
	out.SSHKeyName = in.SSHKeyName
	out.State = InstanceState(in.State)
	out.IP = in.IP
	out.FloatingIP = in.FloatingIP
	out.ServerGroupID = in.ReferencedResources.ServerGroupID
	return nil
}

func Convert_v1alpha5_Network_To_v1alpha8_NetworkStatusWithSubnets(in *Network, out *infrav1.NetworkStatusWithSubnets, s conversion.Scope) error {
	// PortOpts has been removed in v1alpha8
	err := Convert_v1alpha5_Network_To_v1alpha8_NetworkStatus(in, &out.NetworkStatus, s)
	if err != nil {
		return err
	}

	if in.Subnet != nil {
		out.Subnets = []infrav1.Subnet{infrav1.Subnet(*in.Subnet)}
	}
	return nil
}

func Convert_v1alpha8_NetworkStatusWithSubnets_To_v1alpha5_Network(in *infrav1.NetworkStatusWithSubnets, out *Network, s conversion.Scope) error {
	// PortOpts has been removed in v1alpha8
	err := Convert_v1alpha8_NetworkStatus_To_v1alpha5_Network(&in.NetworkStatus, out, s)
	if err != nil {
		return err
	}

	// Can only down-convert a single subnet
	if len(in.Subnets) > 0 {
		out.Subnet = (*Subnet)(&in.Subnets[0])
	}
	return nil
}

func Convert_v1alpha5_Network_To_v1alpha8_NetworkStatus(in *Network, out *infrav1.NetworkStatus, _ conversion.Scope) error {
	out.ID = in.ID
	out.Name = in.Name
	out.Tags = in.Tags

	return nil
}

func Convert_v1alpha8_NetworkStatus_To_v1alpha5_Network(in *infrav1.NetworkStatus, out *Network, _ conversion.Scope) error {
	out.ID = in.ID
	out.Name = in.Name
	out.Tags = in.Tags

	return nil
}

func Convert_v1alpha5_SecurityGroupFilter_To_v1alpha8_SecurityGroupFilter(in *SecurityGroupFilter, out *infrav1.SecurityGroupFilter, s conversion.Scope) error {
	err := autoConvert_v1alpha5_SecurityGroupFilter_To_v1alpha8_SecurityGroupFilter(in, out, s)
	if err != nil {
		return err
	}

	// TenantID has been removed in v1alpha8. Write it to ProjectID if ProjectID is not already set.
	if out.ProjectID == "" {
		out.ProjectID = in.TenantID
	}

	return nil
}

func Convert_v1alpha5_SecurityGroupParam_To_v1alpha8_SecurityGroupFilter(in *SecurityGroupParam, out *infrav1.SecurityGroupFilter, s conversion.Scope) error {
	// SecurityGroupParam is replaced by its contained SecurityGroupFilter in v1alpha8
	err := Convert_v1alpha5_SecurityGroupFilter_To_v1alpha8_SecurityGroupFilter(&in.Filter, out, s)
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

func Convert_v1alpha8_SecurityGroupFilter_To_v1alpha5_SecurityGroupParam(in *infrav1.SecurityGroupFilter, out *SecurityGroupParam, s conversion.Scope) error {
	// SecurityGroupParam is replaced by its contained SecurityGroupFilter in v1alpha8
	err := Convert_v1alpha8_SecurityGroupFilter_To_v1alpha5_SecurityGroupFilter(in, &out.Filter, s)
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

func Convert_v1alpha5_SubnetParam_To_v1alpha8_SubnetFilter(in *SubnetParam, out *infrav1.SubnetFilter, _ conversion.Scope) error {
	*out = infrav1.SubnetFilter(in.Filter)
	if in.UUID != "" {
		out.ID = in.UUID
	}
	return nil
}

func Convert_v1alpha8_SubnetFilter_To_v1alpha5_SubnetParam(in *infrav1.SubnetFilter, out *SubnetParam, _ conversion.Scope) error {
	out.Filter = SubnetFilter(*in)
	out.UUID = in.ID

	return nil
}

func Convert_Map_string_To_Interface_To_v1alpha8_BindingProfile(in map[string]string, out *infrav1.BindingProfile, _ conversion.Scope) error {
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

func Convert_v1alpha8_BindingProfile_To_Map_string_To_Interface(in *infrav1.BindingProfile, out map[string]string, _ conversion.Scope) error {
	if in.OVSHWOffload {
		(out)["capabilities"] = "[\"switchdev\"]"
	}
	if in.TrustedVF {
		(out)["trusted"] = trueString
	}
	return nil
}

func Convert_v1alpha8_OpenStackClusterStatus_To_v1alpha5_OpenStackClusterStatus(in *infrav1.OpenStackClusterStatus, out *OpenStackClusterStatus, s conversion.Scope) error {
	err := autoConvert_v1alpha8_OpenStackClusterStatus_To_v1alpha5_OpenStackClusterStatus(in, out, s)
	if err != nil {
		return err
	}

	// Router and APIServerLoadBalancer have been moved out of Network in v1alpha8
	if in.Router != nil || in.APIServerLoadBalancer != nil {
		if out.Network == nil {
			out.Network = &Network{}
		}

		out.Network.Router = (*Router)(in.Router)
		if in.APIServerLoadBalancer != nil {
			out.Network.APIServerLoadBalancer = &LoadBalancer{}
			err = Convert_v1alpha8_LoadBalancer_To_v1alpha5_LoadBalancer(in.APIServerLoadBalancer, out.Network.APIServerLoadBalancer, s)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func Convert_v1alpha5_OpenStackClusterStatus_To_v1alpha8_OpenStackClusterStatus(in *OpenStackClusterStatus, out *infrav1.OpenStackClusterStatus, s conversion.Scope) error {
	err := autoConvert_v1alpha5_OpenStackClusterStatus_To_v1alpha8_OpenStackClusterStatus(in, out, s)
	if err != nil {
		return err
	}

	// Router and APIServerLoadBalancer have been moved out of Network in v1alpha8
	if in.Network != nil {
		out.Router = (*infrav1.Router)(in.Network.Router)
		if in.Network.APIServerLoadBalancer != nil {
			out.APIServerLoadBalancer = &infrav1.LoadBalancer{}
			err = Convert_v1alpha5_LoadBalancer_To_v1alpha8_LoadBalancer(in.Network.APIServerLoadBalancer, out.APIServerLoadBalancer, s)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func Convert_v1alpha8_OpenStackMachineSpec_To_v1alpha5_OpenStackMachineSpec(in *infrav1.OpenStackMachineSpec, out *OpenStackMachineSpec, s conversion.Scope) error {
	err := autoConvert_v1alpha8_OpenStackMachineSpec_To_v1alpha5_OpenStackMachineSpec(in, out, s)
	if err != nil {
		return err
	}

	if in.ServerGroup != nil {
		out.ServerGroupID = in.ServerGroup.ID
	}

	return nil
}

func Convert_v1alpha8_OpenStackMachineStatus_To_v1alpha5_OpenStackMachineStatus(in *infrav1.OpenStackMachineStatus, out *OpenStackMachineStatus, s conversion.Scope) error {
	// ReferencedResources have no equivalent in v1alpha5
	return autoConvert_v1alpha8_OpenStackMachineStatus_To_v1alpha5_OpenStackMachineStatus(in, out, s)
}
