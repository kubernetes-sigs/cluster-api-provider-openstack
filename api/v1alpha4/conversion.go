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

package v1alpha4

import (
	conversion "k8s.io/apimachinery/pkg/conversion"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha5"
)

var _ ctrlconversion.Convertible = &OpenStackCluster{}

func (r *OpenStackCluster) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackCluster)

	if err := Convert_v1alpha4_OpenStackCluster_To_v1alpha5_OpenStackCluster(r, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &infrav1.OpenStackCluster{}
	if ok, err := utilconversion.UnmarshalData(r, restored); err != nil || !ok {
		return err
	}

	if restored.Spec.APIServerLoadBalancer.AllowedCIDRs != nil {
		dst.Spec.APIServerLoadBalancer.AllowedCIDRs = restored.Spec.APIServerLoadBalancer.AllowedCIDRs
	}

	return nil
}

func (r *OpenStackCluster) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackCluster)

	if err := Convert_v1alpha5_OpenStackCluster_To_v1alpha4_OpenStackCluster(src, r, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata
	if err := utilconversion.MarshalData(src, r); err != nil {
		return err
	}

	return nil
}

var _ ctrlconversion.Convertible = &OpenStackClusterList{}

func (r *OpenStackClusterList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1alpha4_OpenStackClusterList_To_v1alpha5_OpenStackClusterList(r, dst, nil)
}

func (r *OpenStackClusterList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1alpha5_OpenStackClusterList_To_v1alpha4_OpenStackClusterList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackClusterTemplate{}

func (r *OpenStackClusterTemplate) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterTemplate)

	return Convert_v1alpha4_OpenStackClusterTemplate_To_v1alpha5_OpenStackClusterTemplate(r, dst, nil)
}

func (r *OpenStackClusterTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterTemplate)

	return Convert_v1alpha5_OpenStackClusterTemplate_To_v1alpha4_OpenStackClusterTemplate(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackClusterTemplateList{}

func (r *OpenStackClusterTemplateList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterTemplateList)

	return Convert_v1alpha4_OpenStackClusterTemplateList_To_v1alpha5_OpenStackClusterTemplateList(r, dst, nil)
}

func (r *OpenStackClusterTemplateList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterTemplateList)

	return Convert_v1alpha5_OpenStackClusterTemplateList_To_v1alpha4_OpenStackClusterTemplateList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackMachine{}

func (r *OpenStackMachine) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachine)

	return Convert_v1alpha4_OpenStackMachine_To_v1alpha5_OpenStackMachine(r, dst, nil)
}

func (r *OpenStackMachine) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachine)

	if err := Convert_v1alpha5_OpenStackMachine_To_v1alpha4_OpenStackMachine(src, r, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata
	if err := utilconversion.MarshalData(src, r); err != nil {
		return err
	}

	return nil
}

var _ ctrlconversion.Convertible = &OpenStackMachineList{}

func (r *OpenStackMachineList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineList)

	return Convert_v1alpha4_OpenStackMachineList_To_v1alpha5_OpenStackMachineList(r, dst, nil)
}

func (r *OpenStackMachineList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineList)

	return Convert_v1alpha5_OpenStackMachineList_To_v1alpha4_OpenStackMachineList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplate{}

func (r *OpenStackMachineTemplate) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineTemplate)

	return Convert_v1alpha4_OpenStackMachineTemplate_To_v1alpha5_OpenStackMachineTemplate(r, dst, nil)
}

func (r *OpenStackMachineTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplate)

	if err := Convert_v1alpha5_OpenStackMachineTemplate_To_v1alpha4_OpenStackMachineTemplate(src, r, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata
	if err := utilconversion.MarshalData(src, r); err != nil {
		return err
	}

	return nil
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplateList{}

func (r *OpenStackMachineTemplateList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineTemplateList)

	return Convert_v1alpha4_OpenStackMachineTemplateList_To_v1alpha5_OpenStackMachineTemplateList(r, dst, nil)
}

func (r *OpenStackMachineTemplateList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplateList)

	return Convert_v1alpha5_OpenStackMachineTemplateList_To_v1alpha4_OpenStackMachineTemplateList(src, r, nil)
}

func Convert_v1alpha4_SubnetFilter_To_v1alpha5_SubnetFilter(in *SubnetFilter, out *infrav1.SubnetFilter, s conversion.Scope) error {
	out.Name = in.Name
	out.Description = in.Description
	if in.ProjectID != "" {
		out.ProjectID = in.ProjectID
	} else {
		out.ProjectID = in.TenantID
	}
	out.IPVersion = in.IPVersion
	out.GatewayIP = in.GatewayIP
	out.CIDR = in.CIDR
	out.IPv6AddressMode = in.IPv6AddressMode
	out.IPv6RAMode = in.IPv6RAMode
	out.ID = in.ID
	out.Tags = in.Tags
	out.TagsAny = in.TagsAny
	out.NotTags = in.NotTags
	out.NotTagsAny = in.NotTagsAny
	return nil
}

func Convert_v1alpha5_SubnetFilter_To_v1alpha4_SubnetFilter(in *infrav1.SubnetFilter, out *SubnetFilter, s conversion.Scope) error {
	out.TenantID = in.ProjectID
	return autoConvert_v1alpha5_SubnetFilter_To_v1alpha4_SubnetFilter(in, out, s)
}

func Convert_v1alpha4_Filter_To_v1alpha5_NetworkFilter(in *Filter, out *infrav1.NetworkFilter, s conversion.Scope) error {
	out.Name = in.Name
	out.Description = in.Description
	if in.ProjectID != "" {
		out.ProjectID = in.ProjectID
	} else {
		out.ProjectID = in.TenantID
	}
	out.ID = in.ID
	out.Tags = in.Tags
	out.TagsAny = in.TagsAny
	out.NotTags = in.NotTags
	out.NotTagsAny = in.NotTagsAny
	return nil
}

func Convert_v1alpha5_NetworkFilter_To_v1alpha4_Filter(in *infrav1.NetworkFilter, out *Filter, s conversion.Scope) error {
	out.Name = in.Name
	out.Description = in.Description
	out.ProjectID = in.ProjectID
	out.TenantID = in.ProjectID
	out.ID = in.ID
	out.Tags = in.Tags
	out.TagsAny = in.TagsAny
	out.NotTags = in.NotTags
	out.NotTagsAny = in.NotTagsAny
	return nil
}

func Convert_v1alpha4_PortOpts_To_v1alpha5_PortOpts(in *PortOpts, out *infrav1.PortOpts, s conversion.Scope) error {
	err := autoConvert_v1alpha4_PortOpts_To_v1alpha5_PortOpts(in, out, s)
	if err != nil {
		return err
	}
	if in.NetworkID != "" {
		out.Network = &infrav1.NetworkFilter{ID: in.NetworkID}
	}
	return nil
}

func Convert_v1alpha5_PortOpts_To_v1alpha4_PortOpts(in *infrav1.PortOpts, out *PortOpts, s conversion.Scope) error {
	err := autoConvert_v1alpha5_PortOpts_To_v1alpha4_PortOpts(in, out, s)
	if err != nil {
		return err
	}
	if in.Network != nil {
		out.NetworkID = in.Network.ID
	}
	return nil
}

func Convert_Slice_v1alpha4_Network_To_Slice_v1alpha5_Network(in *[]Network, out *[]infrav1.Network, s conversion.Scope) error {
	*out = make([]infrav1.Network, len(*in))
	for i := range *in {
		if err := Convert_v1alpha4_Network_To_v1alpha5_Network(&(*in)[i], &(*out)[i], s); err != nil {
			return err
		}
	}

	return nil
}

func Convert_Slice_v1alpha5_Network_To_Slice_v1alpha4_Network(in *[]infrav1.Network, out *[]Network, s conversion.Scope) error {
	*out = make([]Network, len(*in))
	for i := range *in {
		if err := Convert_v1alpha5_Network_To_v1alpha4_Network(&(*in)[i], &(*out)[i], s); err != nil {
			return err
		}
	}
	return nil
}

func Convert_v1alpha4_FixedIP_To_v1alpha5_FixedIP(in *FixedIP, out *infrav1.FixedIP, s conversion.Scope) error {
	err := autoConvert_v1alpha4_FixedIP_To_v1alpha5_FixedIP(in, out, s)
	if err != nil {
		return err
	}
	if in.SubnetID != "" {
		out.Subnet = &infrav1.SubnetFilter{ID: in.SubnetID}
	}
	return nil
}

func Convert_v1alpha5_FixedIP_To_v1alpha4_FixedIP(in *infrav1.FixedIP, out *FixedIP, s conversion.Scope) error {
	err := autoConvert_v1alpha5_FixedIP_To_v1alpha4_FixedIP(in, out, s)
	if err != nil {
		return err
	}
	if in.Subnet != nil && in.Subnet.ID != "" {
		out.SubnetID = in.Subnet.ID
	}
	return nil
}

/*
 * RootVolume changes:
 * - DeviceType is removed in v1alpha5, hard-coded to disk for prior versions
 * - SourceType is removed in v1alpha5, hard-coded to image for prior versions
 * - SourceUUID is removed in v1alpha5, comes from the parent context
 */

func Convert_v1alpha5_RootVolume_To_v1alpha4_RootVolume(in *infrav1.RootVolume, out *RootVolume, s conversion.Scope) error {
	out.DeviceType = "disk"
	out.SourceType = "image"
	// SourceUUID needs to come from the parent context
	return autoConvert_v1alpha5_RootVolume_To_v1alpha4_RootVolume(in, out, s)
}

func Convert_v1alpha4_RootVolume_To_v1alpha5_RootVolume(in *RootVolume, out *infrav1.RootVolume, s conversion.Scope) error {
	return autoConvert_v1alpha4_RootVolume_To_v1alpha5_RootVolume(in, out, s)
}

func Convert_v1alpha4_Instance_To_v1alpha5_Instance(in *Instance, out *infrav1.Instance, s conversion.Scope) error {
	if err := autoConvert_v1alpha4_Instance_To_v1alpha5_Instance(in, out, s); err != nil {
		return err
	}
	if in.RootVolume != nil && in.RootVolume.Size > 0 {
		out.ImageUUID = in.RootVolume.SourceUUID
	}
	return nil
}

func Convert_v1alpha5_Instance_To_v1alpha4_Instance(in *infrav1.Instance, out *Instance, s conversion.Scope) error {
	if err := autoConvert_v1alpha5_Instance_To_v1alpha4_Instance(in, out, s); err != nil {
		return err
	}
	if in.RootVolume != nil && in.RootVolume.Size > 0 {
		out.RootVolume.SourceUUID = in.ImageUUID
		out.Image = ""
	}
	return nil
}

func Convert_v1alpha4_OpenStackMachineSpec_To_v1alpha5_OpenStackMachineSpec(in *OpenStackMachineSpec, out *infrav1.OpenStackMachineSpec, s conversion.Scope) error {
	if err := autoConvert_v1alpha4_OpenStackMachineSpec_To_v1alpha5_OpenStackMachineSpec(in, out, s); err != nil {
		return err
	}
	if in.RootVolume != nil && in.RootVolume.Size > 0 {
		out.ImageUUID = in.RootVolume.SourceUUID
	}
	return nil
}

func Convert_v1alpha5_OpenStackMachineSpec_To_v1alpha4_OpenStackMachineSpec(in *infrav1.OpenStackMachineSpec, out *OpenStackMachineSpec, s conversion.Scope) error {
	if err := autoConvert_v1alpha5_OpenStackMachineSpec_To_v1alpha4_OpenStackMachineSpec(in, out, s); err != nil {
		return err
	}
	if in.RootVolume != nil && in.RootVolume.Size > 0 {
		out.RootVolume.SourceUUID = in.ImageUUID
		out.Image = ""
	}
	return nil
}

func Convert_v1alpha5_Router_To_v1alpha4_Router(in *infrav1.Router, out *Router, s conversion.Scope) error {
	return autoConvert_v1alpha5_Router_To_v1alpha4_Router(in, out, s)
}

func Convert_v1alpha5_LoadBalancer_To_v1alpha4_LoadBalancer(in *infrav1.LoadBalancer, out *LoadBalancer, s conversion.Scope) error {
	return autoConvert_v1alpha5_LoadBalancer_To_v1alpha4_LoadBalancer(in, out, s)
}
