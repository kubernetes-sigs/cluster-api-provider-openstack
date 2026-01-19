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

package v1beta1

import (
	"strings"
	"unsafe"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiconversion "k8s.io/apimachinery/pkg/conversion"
	"k8s.io/utils/ptr"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1beta2 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
)

// ConvertTo converts this OpenStackCluster to the Hub version (v1beta2).
func (src *OpenStackCluster) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1beta2.OpenStackCluster)
	if err := Convert_v1beta1_OpenStackCluster_To_v1beta2_OpenStackCluster(src, dst, nil); err != nil {
		return err
	}
	for i := range dst.Status.Conditions {
		dst.Status.Conditions[i].ObservedGeneration = src.Generation
	}
	return utilconversion.MarshalData(src, dst)
}

// ConvertFrom converts from the Hub version (v1beta2) to this version.
//
//nolint:revive // dst is the receiver here (converting FROM hub TO spoke)
func (dst *OpenStackCluster) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OpenStackCluster)
	if err := Convert_v1beta2_OpenStackCluster_To_v1beta1_OpenStackCluster(src, dst, nil); err != nil {
		return err
	}
	_, err := utilconversion.UnmarshalData(src, dst)
	return err
}

// ConvertTo converts this OpenStackMachine to the Hub version (v1beta2).
func (src *OpenStackMachine) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1beta2.OpenStackMachine)
	if err := Convert_v1beta1_OpenStackMachine_To_v1beta2_OpenStackMachine(src, dst, nil); err != nil {
		return err
	}
	for i := range dst.Status.Conditions {
		dst.Status.Conditions[i].ObservedGeneration = src.Generation
	}
	return utilconversion.MarshalData(src, dst)
}

// ConvertFrom converts from the Hub version (v1beta2) to this version.
//
//nolint:revive // dst is the receiver here (converting FROM hub TO spoke)
func (dst *OpenStackMachine) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OpenStackMachine)
	if err := Convert_v1beta2_OpenStackMachine_To_v1beta1_OpenStackMachine(src, dst, nil); err != nil {
		return err
	}
	_, err := utilconversion.UnmarshalData(src, dst)
	return err
}

// ConvertTo converts this OpenStackClusterTemplate to the Hub version (v1beta2).
func (src *OpenStackClusterTemplate) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1beta2.OpenStackClusterTemplate)
	if err := Convert_v1beta1_OpenStackClusterTemplate_To_v1beta2_OpenStackClusterTemplate(src, dst, nil); err != nil {
		return err
	}
	return utilconversion.MarshalData(src, dst)
}

// ConvertFrom converts from the Hub version (v1beta2) to this version.
//
//nolint:revive // dst is the receiver here (converting FROM hub TO spoke)
func (dst *OpenStackClusterTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OpenStackClusterTemplate)
	if err := Convert_v1beta2_OpenStackClusterTemplate_To_v1beta1_OpenStackClusterTemplate(src, dst, nil); err != nil {
		return err
	}
	_, err := utilconversion.UnmarshalData(src, dst)
	return err
}

// ConvertTo converts this OpenStackMachineTemplate to the Hub version (v1beta2).
func (src *OpenStackMachineTemplate) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1beta2.OpenStackMachineTemplate)
	if err := Convert_v1beta1_OpenStackMachineTemplate_To_v1beta2_OpenStackMachineTemplate(src, dst, nil); err != nil {
		return err
	}
	return utilconversion.MarshalData(src, dst)
}

// ConvertFrom converts from the Hub version (v1beta2) to this version.
//
//nolint:revive // dst is the receiver here (converting FROM hub TO spoke)
func (dst *OpenStackMachineTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OpenStackMachineTemplate)
	if err := Convert_v1beta2_OpenStackMachineTemplate_To_v1beta1_OpenStackMachineTemplate(src, dst, nil); err != nil {
		return err
	}
	_, err := utilconversion.UnmarshalData(src, dst)
	return err
}

// ConvertTo converts this OpenStackClusterList to the Hub version (v1beta2).
func (src *OpenStackClusterList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1beta2.OpenStackClusterList)
	return Convert_v1beta1_OpenStackClusterList_To_v1beta2_OpenStackClusterList(src, dst, nil)
}

// ConvertFrom converts from the Hub version (v1beta2) to this version.
//
//nolint:revive // dst is the receiver here (converting FROM hub TO spoke)
func (dst *OpenStackClusterList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OpenStackClusterList)
	return Convert_v1beta2_OpenStackClusterList_To_v1beta1_OpenStackClusterList(src, dst, nil)
}

// ConvertTo converts this OpenStackMachineList to the Hub version (v1beta2).
func (src *OpenStackMachineList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1beta2.OpenStackMachineList)
	return Convert_v1beta1_OpenStackMachineList_To_v1beta2_OpenStackMachineList(src, dst, nil)
}

// ConvertFrom converts from the Hub version (v1beta2) to this version.
//
//nolint:revive // dst is the receiver here (converting FROM hub TO spoke)
func (dst *OpenStackMachineList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OpenStackMachineList)
	return Convert_v1beta2_OpenStackMachineList_To_v1beta1_OpenStackMachineList(src, dst, nil)
}

// ConvertTo converts this OpenStackClusterTemplateList to the Hub version (v1beta2).
func (src *OpenStackClusterTemplateList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1beta2.OpenStackClusterTemplateList)
	return Convert_v1beta1_OpenStackClusterTemplateList_To_v1beta2_OpenStackClusterTemplateList(src, dst, nil)
}

// ConvertFrom converts from the Hub version (v1beta2) to this version.
//
//nolint:revive // dst is the receiver here (converting FROM hub TO spoke)
func (dst *OpenStackClusterTemplateList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OpenStackClusterTemplateList)
	return Convert_v1beta2_OpenStackClusterTemplateList_To_v1beta1_OpenStackClusterTemplateList(src, dst, nil)
}

// ConvertTo converts this OpenStackMachineTemplateList to the Hub version (v1beta2).
func (src *OpenStackMachineTemplateList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1beta2.OpenStackMachineTemplateList)
	return Convert_v1beta1_OpenStackMachineTemplateList_To_v1beta2_OpenStackMachineTemplateList(src, dst, nil)
}

// ConvertFrom converts from the Hub version (v1beta2) to this version.
//
//nolint:revive // dst is the receiver here (converting FROM hub TO spoke)
func (dst *OpenStackMachineTemplateList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OpenStackMachineTemplateList)
	return Convert_v1beta2_OpenStackMachineTemplateList_To_v1beta1_OpenStackMachineTemplateList(src, dst, nil)
}

// Manual conversion functions for Status types that conversion-gen cannot
// auto-generate due to FailureDomains (map↔slice), Conditions (CAPI↔metav1),
// and deprecated fields (Ready, FailureReason, FailureMessage).

func Convert_v1beta1_OpenStackClusterStatus_To_v1beta2_OpenStackClusterStatus(in *OpenStackClusterStatus, out *infrav1beta2.OpenStackClusterStatus, _ apiconversion.Scope) error {
	out.Initialization = (*infrav1beta2.ClusterInitialization)(unsafe.Pointer(in.Initialization))
	out.Network = (*infrav1beta2.NetworkStatusWithSubnets)(unsafe.Pointer(in.Network))
	out.ExternalNetwork = (*infrav1beta2.NetworkStatus)(unsafe.Pointer(in.ExternalNetwork))
	out.Router = (*infrav1beta2.Router)(unsafe.Pointer(in.Router))
	out.APIServerLoadBalancer = (*infrav1beta2.LoadBalancer)(unsafe.Pointer(in.APIServerLoadBalancer))
	out.ControlPlaneSecurityGroup = (*infrav1beta2.SecurityGroupStatus)(unsafe.Pointer(in.ControlPlaneSecurityGroup))
	out.WorkerSecurityGroup = (*infrav1beta2.SecurityGroupStatus)(unsafe.Pointer(in.WorkerSecurityGroup))
	out.BastionSecurityGroup = (*infrav1beta2.SecurityGroupStatus)(unsafe.Pointer(in.BastionSecurityGroup))
	out.Bastion = (*infrav1beta2.BastionStatus)(unsafe.Pointer(in.Bastion))

	if len(in.FailureDomains) > 0 {
		out.FailureDomains = make([]clusterv1.FailureDomain, 0, len(in.FailureDomains))
		for name, fd := range in.FailureDomains {
			out.FailureDomains = append(out.FailureDomains, clusterv1.FailureDomain{
				Name:         name,
				ControlPlane: ptr.To(fd.ControlPlane),
				Attributes:   fd.Attributes,
			})
		}
	}

	out.Conditions = infrav1beta2.ConvertConditionsToV1Beta2(in.Conditions, 0)

	return nil
}

func Convert_v1beta2_OpenStackClusterStatus_To_v1beta1_OpenStackClusterStatus(in *infrav1beta2.OpenStackClusterStatus, out *OpenStackClusterStatus, _ apiconversion.Scope) error {
	out.Initialization = (*ClusterInitialization)(unsafe.Pointer(in.Initialization))
	out.Network = (*NetworkStatusWithSubnets)(unsafe.Pointer(in.Network))
	out.ExternalNetwork = (*NetworkStatus)(unsafe.Pointer(in.ExternalNetwork))
	out.Router = (*Router)(unsafe.Pointer(in.Router))
	out.APIServerLoadBalancer = (*LoadBalancer)(unsafe.Pointer(in.APIServerLoadBalancer))
	out.ControlPlaneSecurityGroup = (*SecurityGroupStatus)(unsafe.Pointer(in.ControlPlaneSecurityGroup))
	out.WorkerSecurityGroup = (*SecurityGroupStatus)(unsafe.Pointer(in.WorkerSecurityGroup))
	out.BastionSecurityGroup = (*SecurityGroupStatus)(unsafe.Pointer(in.BastionSecurityGroup))
	out.Bastion = (*BastionStatus)(unsafe.Pointer(in.Bastion))

	if len(in.FailureDomains) > 0 {
		out.FailureDomains = make(clusterv1beta1.FailureDomains, len(in.FailureDomains))
		for _, fd := range in.FailureDomains {
			out.FailureDomains[fd.Name] = clusterv1beta1.FailureDomainSpec{
				ControlPlane: ptr.Deref(fd.ControlPlane, false),
				Attributes:   fd.Attributes,
			}
		}
	}

	out.Conditions = infrav1beta2.ConvertConditionsFromV1Beta2(in.Conditions)
	out.Ready = infrav1beta2.IsReady(in.Conditions)

	return nil
}

func Convert_v1beta1_OpenStackMachineStatus_To_v1beta2_OpenStackMachineStatus(in *OpenStackMachineStatus, out *infrav1beta2.OpenStackMachineStatus, _ apiconversion.Scope) error {
	out.Initialization = (*infrav1beta2.MachineInitialization)(unsafe.Pointer(in.Initialization))
	out.InstanceID = in.InstanceID
	out.Addresses = in.Addresses
	out.InstanceState = (*infrav1beta2.InstanceState)(unsafe.Pointer(in.InstanceState))
	out.Resolved = (*infrav1beta2.ResolvedMachineSpec)(unsafe.Pointer(in.Resolved))
	out.Resources = (*infrav1beta2.MachineResources)(unsafe.Pointer(in.Resources))

	out.Conditions = infrav1beta2.ConvertConditionsToV1Beta2(in.Conditions, 0)

	return nil
}

func Convert_v1beta2_OpenStackMachineStatus_To_v1beta1_OpenStackMachineStatus(in *infrav1beta2.OpenStackMachineStatus, out *OpenStackMachineStatus, _ apiconversion.Scope) error {
	out.Initialization = (*MachineInitialization)(unsafe.Pointer(in.Initialization))
	out.InstanceID = in.InstanceID
	out.Addresses = in.Addresses
	out.InstanceState = (*InstanceState)(unsafe.Pointer(in.InstanceState))
	out.Resolved = (*ResolvedMachineSpec)(unsafe.Pointer(in.Resolved))
	out.Resources = (*MachineResources)(unsafe.Pointer(in.Resources))

	out.Conditions = infrav1beta2.ConvertConditionsFromV1Beta2(in.Conditions)
	out.Ready = infrav1beta2.IsReady(in.Conditions)

	return nil
}

// Element-level Condition conversion functions required by conversion-gen's
// autoConvert functions for Status types. The actual condition conversion is
// handled at the Status level by the manual Convert_*_Status_* functions above.

func Convert_v1beta1_Condition_To_v1_Condition(in *clusterv1beta1.Condition, out *metav1.Condition, _ apiconversion.Scope) error {
	out.Type = string(in.Type)
	out.Status = metav1.ConditionStatus(in.Status)
	out.LastTransitionTime = in.LastTransitionTime
	out.Reason = in.Reason
	out.Message = in.Message
	return nil
}

func Convert_v1_Condition_To_v1beta1_Condition(in *metav1.Condition, out *clusterv1beta1.Condition, _ apiconversion.Scope) error {
	out.Type = clusterv1beta1.ConditionType(in.Type)
	out.Status = corev1.ConditionStatus(in.Status)
	out.LastTransitionTime = in.LastTransitionTime
	out.Reason = in.Reason
	out.Message = in.Message
	return nil
}

// LegacyCalicoSecurityGroupRules returns a list of security group rules for calico
// that need to be applied to the control plane and worker security groups when
// managed security groups are enabled and upgrading to v1beta1.
func LegacyCalicoSecurityGroupRules() []SecurityGroupRuleSpec {
	return []SecurityGroupRuleSpec{
		{
			Name:                "BGP (calico)",
			Description:         ptr.To("Created by cluster-api-provider-openstack API conversion - BGP (calico)"),
			Direction:           "ingress",
			EtherType:           ptr.To("IPv4"),
			PortRangeMin:        ptr.To(179),
			PortRangeMax:        ptr.To(179),
			Protocol:            ptr.To("tcp"),
			RemoteManagedGroups: []ManagedSecurityGroupName{"controlplane", "worker"},
		},
		{
			Name:                "IP-in-IP (calico)",
			Description:         ptr.To("Created by cluster-api-provider-openstack API conversion - IP-in-IP (calico)"),
			Direction:           "ingress",
			EtherType:           ptr.To("IPv4"),
			Protocol:            ptr.To("4"),
			RemoteManagedGroups: []ManagedSecurityGroupName{"controlplane", "worker"},
		},
	}
}

// splitTags splits a comma separated list of tags into a slice of tags.
// If the input is an empty string, it returns nil representing no list rather
// than an empty list.
func splitTags(tags string) []NeutronTag {
	if tags == "" {
		return nil
	}

	var ret []NeutronTag
	for _, tag := range strings.Split(tags, ",") {
		if tag != "" {
			ret = append(ret, NeutronTag(tag))
		}
	}

	return ret
}

// JoinTags joins a slice of tags into a comma separated list of tags.
func JoinTags(tags []NeutronTag) string {
	var b strings.Builder
	for i := range tags {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(string(tags[i]))
	}
	return b.String()
}

func ConvertAllTagsTo(tags, tagsAny, notTags, notTagsAny string, neutronTags *FilterByNeutronTags) {
	neutronTags.Tags = splitTags(tags)
	neutronTags.TagsAny = splitTags(tagsAny)
	neutronTags.NotTags = splitTags(notTags)
	neutronTags.NotTagsAny = splitTags(notTagsAny)
}

func ConvertAllTagsFrom(neutronTags *FilterByNeutronTags, tags, tagsAny, notTags, notTagsAny *string) {
	*tags = JoinTags(neutronTags.Tags)
	*tagsAny = JoinTags(neutronTags.TagsAny)
	*notTags = JoinTags(neutronTags.NotTags)
	*notTagsAny = JoinTags(neutronTags.NotTagsAny)
}
