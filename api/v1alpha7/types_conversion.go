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

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

/* SecurityGroupFilter */

func restorev1alpha7SecurityGroupFilter(previous *SecurityGroupFilter, dst *SecurityGroupFilter) {
	// The edge cases with multiple commas are too tricky in this direction,
	// so we just restore the whole thing.
	dst.Tags = previous.Tags
	dst.TagsAny = previous.TagsAny
	dst.NotTags = previous.NotTags
	dst.NotTagsAny = previous.NotTagsAny
}

func restorev1alpha7SecurityGroup(previous *SecurityGroup, dst *SecurityGroup) {
	if previous == nil || dst == nil {
		return
	}

	for i, rule := range previous.Rules {
		dst.Rules[i].SecurityGroupID = rule.SecurityGroupID
	}
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

/* NetworkFilter */

func restorev1alpha7NetworkFilter(previous *NetworkFilter, dst *NetworkFilter) {
	// The edge cases with multiple commas are too tricky in this direction,
	// so we just restore the whole thing.
	dst.Tags = previous.Tags
	dst.TagsAny = previous.TagsAny
	dst.NotTags = previous.NotTags
	dst.NotTagsAny = previous.NotTagsAny
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

/* SubnetFilter */

func restorev1alpha7SubnetFilter(previous *SubnetFilter, dst *SubnetFilter) {
	// The edge cases with multiple commas are too tricky in this direction,
	// so we just restore the whole thing.
	dst.Tags = previous.Tags
	dst.TagsAny = previous.TagsAny
	dst.NotTags = previous.NotTags
	dst.NotTagsAny = previous.NotTagsAny
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

/* RouterFilter */

func restorev1alpha7RouterFilter(previous *RouterFilter, dst *RouterFilter) {
	// The edge cases with multiple commas are too tricky in this direction,
	// so we just restore the whole thing.
	dst.Tags = previous.Tags
	dst.TagsAny = previous.TagsAny
	dst.NotTags = previous.NotTags
	dst.NotTagsAny = previous.NotTagsAny
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

/* PortOpts */

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

/* SecurityGroup */

func Convert_v1alpha7_SecurityGroup_To_v1beta1_SecurityGroupStatus(in *SecurityGroup, out *infrav1.SecurityGroupStatus, _ apiconversion.Scope) error {
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

func Convert_v1beta1_SecurityGroupStatus_To_v1alpha7_SecurityGroup(in *infrav1.SecurityGroupStatus, out *SecurityGroup, _ apiconversion.Scope) error {
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

/* OpenStackIdentityReference */

func Convert_v1alpha7_OpenStackIdentityReference_To_v1beta1_OpenStackIdentityReference(in *OpenStackIdentityReference, out *infrav1.OpenStackIdentityReference, s apiconversion.Scope) error {
	return autoConvert_v1alpha7_OpenStackIdentityReference_To_v1beta1_OpenStackIdentityReference(in, out, s)
}

func Convert_v1beta1_OpenStackIdentityReference_To_v1alpha7_OpenStackIdentityReference(in *infrav1.OpenStackIdentityReference, out *OpenStackIdentityReference, _ apiconversion.Scope) error {
	out.Name = in.Name
	return nil
}
