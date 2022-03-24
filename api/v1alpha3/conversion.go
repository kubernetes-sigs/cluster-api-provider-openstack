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

package v1alpha3

import (
	unsafe "unsafe"

	corev1 "k8s.io/api/core/v1"
	conversion "k8s.io/apimachinery/pkg/conversion"
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha5"
)

var _ ctrlconversion.Convertible = &OpenStackCluster{}

func (r *OpenStackCluster) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackCluster)

	return Convert_v1alpha3_OpenStackCluster_To_v1alpha5_OpenStackCluster(r, dst, nil)
}

func (r *OpenStackCluster) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackCluster)

	return Convert_v1alpha5_OpenStackCluster_To_v1alpha3_OpenStackCluster(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackClusterList{}

func (r *OpenStackClusterList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1alpha3_OpenStackClusterList_To_v1alpha5_OpenStackClusterList(r, dst, nil)
}

func (r *OpenStackClusterList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1alpha5_OpenStackClusterList_To_v1alpha3_OpenStackClusterList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackMachine{}

func (r *OpenStackMachine) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachine)

	return Convert_v1alpha3_OpenStackMachine_To_v1alpha5_OpenStackMachine(r, dst, nil)
}

func (r *OpenStackMachine) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachine)

	return Convert_v1alpha5_OpenStackMachine_To_v1alpha3_OpenStackMachine(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackMachineList{}

func (r *OpenStackMachineList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineList)

	return Convert_v1alpha3_OpenStackMachineList_To_v1alpha5_OpenStackMachineList(r, dst, nil)
}

func (r *OpenStackMachineList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineList)

	return Convert_v1alpha5_OpenStackMachineList_To_v1alpha3_OpenStackMachineList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplate{}

func (r *OpenStackMachineTemplate) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineTemplate)

	return Convert_v1alpha3_OpenStackMachineTemplate_To_v1alpha5_OpenStackMachineTemplate(r, dst, nil)
}

func (r *OpenStackMachineTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplate)

	return Convert_v1alpha5_OpenStackMachineTemplate_To_v1alpha3_OpenStackMachineTemplate(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplateList{}

func (r *OpenStackMachineTemplateList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineTemplateList)

	return Convert_v1alpha3_OpenStackMachineTemplateList_To_v1alpha5_OpenStackMachineTemplateList(r, dst, nil)
}

func (r *OpenStackMachineTemplateList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplateList)

	return Convert_v1alpha5_OpenStackMachineTemplateList_To_v1alpha3_OpenStackMachineTemplateList(src, r, nil)
}

// Convert_v1alpha3_OpenStackClusterSpec_To_v1alpha5_OpenStackClusterSpec has to be added by us because we dropped
// the useOctavia parameter. We don't have to migrate this parameter to v1alpha5 so there is nothing to do.
func Convert_v1alpha3_OpenStackClusterSpec_To_v1alpha5_OpenStackClusterSpec(in *OpenStackClusterSpec, out *infrav1.OpenStackClusterSpec, s conversion.Scope) error {
	if in.CloudsSecret != nil {
		out.IdentityRef = &infrav1.OpenStackIdentityReference{
			Kind: "Secret",
			Name: in.CloudsSecret.Name,
		}
	}
	out.APIServerLoadBalancer = infrav1.APIServerLoadBalancer{
		Enabled:         in.ManagedAPIServerLoadBalancer,
		AdditionalPorts: *(*[]int)(unsafe.Pointer(&in.APIServerLoadBalancerAdditionalPorts)),
	}
	return autoConvert_v1alpha3_OpenStackClusterSpec_To_v1alpha5_OpenStackClusterSpec(in, out, s)
}

// Convert_v1alpha5_OpenStackClusterSpec_To_v1alpha3_OpenStackClusterSpec has to be added by us because we have to
// convert the Type of CloudsSecret from SecretReference to string.
func Convert_v1alpha5_OpenStackClusterSpec_To_v1alpha3_OpenStackClusterSpec(in *infrav1.OpenStackClusterSpec, out *OpenStackClusterSpec, s conversion.Scope) error {
	if in.IdentityRef != nil {
		out.CloudsSecret = &corev1.SecretReference{
			Name: in.IdentityRef.Name,
		}
	}

	if in.Bastion != nil && in.Bastion.Instance.IdentityRef != nil {
		outBastion := out.Bastion
		if outBastion == nil {
			outBastion = &Bastion{}
		}

		outBastion.Instance.CloudsSecret = &corev1.SecretReference{
			Name: in.Bastion.Instance.IdentityRef.Name,
		}
	}

	out.ManagedAPIServerLoadBalancer = in.APIServerLoadBalancer.Enabled
	out.APIServerLoadBalancerAdditionalPorts = *(*[]int)(unsafe.Pointer(&in.APIServerLoadBalancer.AdditionalPorts))

	return autoConvert_v1alpha5_OpenStackClusterSpec_To_v1alpha3_OpenStackClusterSpec(in, out, s)
}

// Convert_v1alpha3_OpenStackMachineSpec_To_v1alpha5_OpenStackMachineSpec is an autogenerated conversion function.
// v1alpha5 drops the field .UserDataSecret which is why we reuqire to define the function here.
func Convert_v1alpha3_OpenStackMachineSpec_To_v1alpha5_OpenStackMachineSpec(in *OpenStackMachineSpec, out *infrav1.OpenStackMachineSpec, s conversion.Scope) error {
	if in.CloudsSecret != nil {
		out.IdentityRef = &infrav1.OpenStackIdentityReference{
			Name: in.CloudsSecret.Name,
			Kind: "Secret",
		}
	}
	if err := autoConvert_v1alpha3_OpenStackMachineSpec_To_v1alpha5_OpenStackMachineSpec(in, out, s); err != nil {
		return err
	}
	if in.RootVolume != nil && in.RootVolume.Size > 0 {
		out.Image = in.RootVolume.SourceUUID
	}
	return nil
}

// Convert_v1alpha5_Network_To_v1alpha3_Network has to be added by us for the new portOpts
// parameter in v1alpha5. There is no intention to support this parameter in v1alpha3, so the field is just dropped.
func Convert_v1alpha5_Network_To_v1alpha3_Network(in *infrav1.Network, out *Network, s conversion.Scope) error {
	return autoConvert_v1alpha5_Network_To_v1alpha3_Network(in, out, s)
}

// Convert_v1alpha5_OpenStackMachineSpec_To_v1alpha3_OpenStackMachineSpec has to be added by us for the new ports
// parameter in v1alpha5. There is no intention to support this parameter in v1alpha3, so the field is just dropped.
// Further, we want to convert the Type of CloudsSecret from SecretReference to string.
func Convert_v1alpha5_OpenStackMachineSpec_To_v1alpha3_OpenStackMachineSpec(in *infrav1.OpenStackMachineSpec, out *OpenStackMachineSpec, s conversion.Scope) error {
	if err := autoConvert_v1alpha5_OpenStackMachineSpec_To_v1alpha3_OpenStackMachineSpec(in, out, s); err != nil {
		return err
	}
	if in.IdentityRef != nil {
		out.CloudsSecret = &corev1.SecretReference{
			Name: in.IdentityRef.Name,
		}
	}
	if in.RootVolume != nil && in.RootVolume.Size > 0 {
		out.RootVolume.SourceUUID = in.Image
		out.Image = ""
	}
	return nil
}

// Convert_v1alpha5_OpenStackClusterStatus_To_v1alpha3_OpenStackClusterStatus has to be added
// in order to drop the FailureReason and FailureMessage fields that are not present in v1alpha3.
func Convert_v1alpha5_OpenStackClusterStatus_To_v1alpha3_OpenStackClusterStatus(in *infrav1.OpenStackClusterStatus, out *OpenStackClusterStatus, s conversion.Scope) error {
	return autoConvert_v1alpha5_OpenStackClusterStatus_To_v1alpha3_OpenStackClusterStatus(in, out, s)
}

func Convert_Slice_v1alpha5_Network_To_Slice_v1alpha3_Network(in *[]infrav1.Network, out *[]Network, s conversion.Scope) error {
	for i := range *in {
		inNet := &(*in)[i]
		outNet := new(Network)
		if err := autoConvert_v1alpha5_Network_To_v1alpha3_Network(inNet, outNet, s); err != nil {
			return err
		}
		*out = append(*out, *outNet)
	}
	return nil
}

func Convert_Slice_v1alpha3_Network_To_Slice_v1alpha5_Network(in *[]Network, out *[]infrav1.Network, s conversion.Scope) error {
	for i := range *in {
		inNet := &(*in)[i]
		outNet := new(infrav1.Network)
		if err := autoConvert_v1alpha3_Network_To_v1alpha5_Network(inNet, outNet, s); err != nil {
			return err
		}
		*out = append(*out, *outNet)
	}
	return nil
}

func Convert_v1alpha3_SubnetFilter_To_v1alpha5_SubnetFilter(in *SubnetFilter, out *infrav1.SubnetFilter, s conversion.Scope) error {
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

func Convert_v1alpha5_SubnetFilter_To_v1alpha3_SubnetFilter(in *infrav1.SubnetFilter, out *SubnetFilter, s conversion.Scope) error {
	out.TenantID = in.ProjectID
	return autoConvert_v1alpha5_SubnetFilter_To_v1alpha3_SubnetFilter(in, out, s)
}

func Convert_v1alpha3_Filter_To_v1alpha5_NetworkFilter(in *Filter, out *infrav1.NetworkFilter, s conversion.Scope) error {
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

func Convert_v1alpha5_NetworkFilter_To_v1alpha3_Filter(in *infrav1.NetworkFilter, out *Filter, s conversion.Scope) error {
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

/*
 * RootVolume changes:
 * - DeviceType is removed in v1alpha5, hard-coded to disk for prior versions
 * - SourceType is removed in v1alpha5, hard-coded to image for prior versions
 * - SourceUUID is removed in v1alpha5, comes from the parent context
 */

func Convert_v1alpha5_RootVolume_To_v1alpha3_RootVolume(in *infrav1.RootVolume, out *RootVolume, s conversion.Scope) error {
	out.DeviceType = "disk"
	out.SourceType = "image"
	// SourceUUID needs to come from the parent context
	return autoConvert_v1alpha5_RootVolume_To_v1alpha3_RootVolume(in, out, s)
}

func Convert_v1alpha3_RootVolume_To_v1alpha5_RootVolume(in *RootVolume, out *infrav1.RootVolume, s conversion.Scope) error {
	return autoConvert_v1alpha3_RootVolume_To_v1alpha5_RootVolume(in, out, s)
}

func Convert_v1alpha3_Instance_To_v1alpha5_Instance(in *Instance, out *infrav1.Instance, s conversion.Scope) error {
	if err := autoConvert_v1alpha3_Instance_To_v1alpha5_Instance(in, out, s); err != nil {
		return err
	}
	if in.RootVolume != nil && in.RootVolume.Size > 0 {
		out.Image = in.RootVolume.SourceUUID
	}
	return nil
}

func Convert_v1alpha5_Instance_To_v1alpha3_Instance(in *infrav1.Instance, out *Instance, s conversion.Scope) error {
	if err := autoConvert_v1alpha5_Instance_To_v1alpha3_Instance(in, out, s); err != nil {
		return err
	}
	if in.RootVolume != nil && in.RootVolume.Size > 0 {
		out.RootVolume.SourceUUID = in.Image
		out.Image = ""
	}
	return nil
}
