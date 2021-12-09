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
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

var _ ctrlconversion.Convertible = &OpenStackCluster{}

func (r *OpenStackCluster) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackCluster)

	return Convert_v1alpha4_OpenStackCluster_To_v1beta1_OpenStackCluster(r, dst, nil)
}

func (r *OpenStackCluster) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackCluster)

	return Convert_v1beta1_OpenStackCluster_To_v1alpha4_OpenStackCluster(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackClusterList{}

func (r *OpenStackClusterList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1alpha4_OpenStackClusterList_To_v1beta1_OpenStackClusterList(r, dst, nil)
}

func (r *OpenStackClusterList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1beta1_OpenStackClusterList_To_v1alpha4_OpenStackClusterList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackMachine{}

func (r *OpenStackMachine) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachine)

	return Convert_v1alpha4_OpenStackMachine_To_v1beta1_OpenStackMachine(r, dst, nil)
}

func (r *OpenStackMachine) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachine)

	return Convert_v1beta1_OpenStackMachine_To_v1alpha4_OpenStackMachine(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackMachineList{}

func (r *OpenStackMachineList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineList)

	return Convert_v1alpha4_OpenStackMachineList_To_v1beta1_OpenStackMachineList(r, dst, nil)
}

func (r *OpenStackMachineList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineList)

	return Convert_v1beta1_OpenStackMachineList_To_v1alpha4_OpenStackMachineList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplate{}

func (r *OpenStackMachineTemplate) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineTemplate)

	return Convert_v1alpha4_OpenStackMachineTemplate_To_v1beta1_OpenStackMachineTemplate(r, dst, nil)
}

func (r *OpenStackMachineTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplate)

	return Convert_v1beta1_OpenStackMachineTemplate_To_v1alpha4_OpenStackMachineTemplate(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplateList{}

func (r *OpenStackMachineTemplateList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineTemplateList)

	return Convert_v1alpha4_OpenStackMachineTemplateList_To_v1beta1_OpenStackMachineTemplateList(r, dst, nil)
}

func (r *OpenStackMachineTemplateList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplateList)

	return Convert_v1beta1_OpenStackMachineTemplateList_To_v1alpha4_OpenStackMachineTemplateList(src, r, nil)
}

func Convert_v1alpha4_SubnetFilter_To_v1beta1_SubnetFilter(in *SubnetFilter, out *infrav1.SubnetFilter, s conversion.Scope) error {
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

func Convert_v1beta1_SubnetFilter_To_v1alpha4_SubnetFilter(in *infrav1.SubnetFilter, out *SubnetFilter, s conversion.Scope) error {
	out.TenantID = in.ProjectID
	return autoConvert_v1beta1_SubnetFilter_To_v1alpha4_SubnetFilter(in, out, s)
}

func Convert_v1alpha4_Filter_To_v1beta1_NetworkFilter(in *Filter, out *infrav1.NetworkFilter, s conversion.Scope) error {
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

func Convert_v1beta1_NetworkFilter_To_v1alpha4_Filter(in *infrav1.NetworkFilter, out *Filter, s conversion.Scope) error {
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

func Convert_v1alpha4_PortOpts_To_v1beta1_PortOpts(in *PortOpts, out *infrav1.PortOpts, s conversion.Scope) error {
	err := autoConvert_v1alpha4_PortOpts_To_v1beta1_PortOpts(in, out, s)
	if err != nil {
		return err
	}
	if in.NetworkID != "" {
		out.Network = &infrav1.NetworkFilter{ID: in.NetworkID}
	}
	return nil
}

func Convert_v1beta1_PortOpts_To_v1alpha4_PortOpts(in *infrav1.PortOpts, out *PortOpts, s conversion.Scope) error {
	err := autoConvert_v1beta1_PortOpts_To_v1alpha4_PortOpts(in, out, s)
	if err != nil {
		return err
	}
	if in.Network != nil {
		out.NetworkID = in.Network.ID
	}
	return nil
}

func Convert_Slice_v1alpha4_Network_To_Slice_v1beta1_Network(in *[]Network, out *[]infrav1.Network, s conversion.Scope) error {
	*out = make([]infrav1.Network, len(*in))
	for i := range *in {
		if err := Convert_v1alpha4_Network_To_v1beta1_Network(&(*in)[i], &(*out)[i], s); err != nil {
			return err
		}
	}

	return nil
}

func Convert_Slice_v1beta1_Network_To_Slice_v1alpha4_Network(in *[]infrav1.Network, out *[]Network, s conversion.Scope) error {
	*out = make([]Network, len(*in))
	for i := range *in {
		if err := Convert_v1beta1_Network_To_v1alpha4_Network(&(*in)[i], &(*out)[i], s); err != nil {
			return err
		}
	}
	return nil
}

func Convert_v1alpha4_FixedIP_To_v1beta1_FixedIP(in *FixedIP, out *infrav1.FixedIP, s conversion.Scope) error {
	err := autoConvert_v1alpha4_FixedIP_To_v1beta1_FixedIP(in, out, s)
	if err != nil {
		return err
	}
	if in.SubnetID != "" {
		out.Subnet = &infrav1.SubnetFilter{ID: in.SubnetID}
	}
	return nil
}

func Convert_v1beta1_FixedIP_To_v1alpha4_FixedIP(in *infrav1.FixedIP, out *FixedIP, s conversion.Scope) error {
	err := autoConvert_v1beta1_FixedIP_To_v1alpha4_FixedIP(in, out, s)
	if err != nil {
		return err
	}
	if in.Subnet != nil && in.Subnet.ID != "" {
		out.SubnetID = in.Subnet.ID
	}
	return nil
}
