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

	"k8s.io/utils/ptr"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1beta2 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
)

// ConvertTo converts this OpenStackCluster to the Hub version (v1beta2).
func (src *OpenStackCluster) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrav1beta2.OpenStackCluster)

	// Copy metadata
	dst.ObjectMeta = src.ObjectMeta

	// Copy Spec (using unsafe pointer since struct layouts are identical)
	dst.Spec = *(*infrav1beta2.OpenStackClusterSpec)(unsafe.Pointer(&src.Spec))

	// Convert Status - most fields are identical structures
	dst.Status.Initialization = (*infrav1beta2.ClusterInitialization)(unsafe.Pointer(src.Status.Initialization))
	dst.Status.Network = (*infrav1beta2.NetworkStatusWithSubnets)(unsafe.Pointer(src.Status.Network))
	dst.Status.ExternalNetwork = (*infrav1beta2.NetworkStatus)(unsafe.Pointer(src.Status.ExternalNetwork))
	dst.Status.Router = (*infrav1beta2.Router)(unsafe.Pointer(src.Status.Router))
	dst.Status.APIServerLoadBalancer = (*infrav1beta2.LoadBalancer)(unsafe.Pointer(src.Status.APIServerLoadBalancer))
	dst.Status.ControlPlaneSecurityGroup = (*infrav1beta2.SecurityGroupStatus)(unsafe.Pointer(src.Status.ControlPlaneSecurityGroup))
	dst.Status.WorkerSecurityGroup = (*infrav1beta2.SecurityGroupStatus)(unsafe.Pointer(src.Status.WorkerSecurityGroup))
	dst.Status.BastionSecurityGroup = (*infrav1beta2.SecurityGroupStatus)(unsafe.Pointer(src.Status.BastionSecurityGroup))
	dst.Status.Bastion = (*infrav1beta2.BastionStatus)(unsafe.Pointer(src.Status.Bastion))

	// Convert FailureDomains from map to slice
	// v1beta1: map[string]FailureDomainSpec where FailureDomainSpec has ControlPlane bool (not pointer)
	// v1beta2: []FailureDomain where FailureDomain has ControlPlane *bool and Name string
	if len(src.Status.FailureDomains) > 0 {
		dst.Status.FailureDomains = make([]clusterv1.FailureDomain, 0, len(src.Status.FailureDomains))
		for name, fd := range src.Status.FailureDomains {
			dst.Status.FailureDomains = append(dst.Status.FailureDomains, clusterv1.FailureDomain{
				Name:         name,
				ControlPlane: ptr.To(fd.ControlPlane),
				Attributes:   fd.Attributes,
			})
		}
	}

	// Convert conditions (v1beta1 format -> standard k8s format)
	dst.Status.Conditions = infrav1beta2.ConvertConditionsToV1Beta2(src.Status.Conditions, src.Generation)

	// Store original object for restoration
	return utilconversion.MarshalData(src, dst)
}

// ConvertFrom converts from the Hub version (v1beta2) to this version.
//
//nolint:revive // dst is the receiver here (converting FROM hub TO spoke)
func (dst *OpenStackCluster) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OpenStackCluster)

	// Copy metadata
	dst.ObjectMeta = src.ObjectMeta

	// Copy Spec (using unsafe pointer since struct layouts are identical)
	dst.Spec = *(*OpenStackClusterSpec)(unsafe.Pointer(&src.Spec))

	// Convert Status
	dst.Status.Initialization = (*ClusterInitialization)(unsafe.Pointer(src.Status.Initialization))
	dst.Status.Network = (*NetworkStatusWithSubnets)(unsafe.Pointer(src.Status.Network))
	dst.Status.ExternalNetwork = (*NetworkStatus)(unsafe.Pointer(src.Status.ExternalNetwork))
	dst.Status.Router = (*Router)(unsafe.Pointer(src.Status.Router))
	dst.Status.APIServerLoadBalancer = (*LoadBalancer)(unsafe.Pointer(src.Status.APIServerLoadBalancer))
	dst.Status.ControlPlaneSecurityGroup = (*SecurityGroupStatus)(unsafe.Pointer(src.Status.ControlPlaneSecurityGroup))
	dst.Status.WorkerSecurityGroup = (*SecurityGroupStatus)(unsafe.Pointer(src.Status.WorkerSecurityGroup))
	dst.Status.BastionSecurityGroup = (*SecurityGroupStatus)(unsafe.Pointer(src.Status.BastionSecurityGroup))
	dst.Status.Bastion = (*BastionStatus)(unsafe.Pointer(src.Status.Bastion))

	// Convert FailureDomains from slice to map
	// v1beta2: []FailureDomain where FailureDomain has ControlPlane *bool and Name string
	// v1beta1: map[string]FailureDomainSpec where FailureDomainSpec has ControlPlane bool (not pointer)
	if len(src.Status.FailureDomains) > 0 {
		dst.Status.FailureDomains = make(clusterv1beta1.FailureDomains, len(src.Status.FailureDomains))
		for _, fd := range src.Status.FailureDomains {
			controlPlane := false
			if fd.ControlPlane != nil {
				controlPlane = *fd.ControlPlane
			}
			dst.Status.FailureDomains[fd.Name] = clusterv1beta1.FailureDomainSpec{
				ControlPlane: controlPlane,
				Attributes:   fd.Attributes,
			}
		}
	}

	// Convert conditions (standard k8s format -> v1beta1 format)
	dst.Status.Conditions = infrav1beta2.ConvertConditionsFromV1Beta2(src.Status.Conditions)

	// Derive deprecated Ready field from Ready condition
	dst.Status.Ready = infrav1beta2.IsReady(src.Status.Conditions)
	// FailureReason and FailureMessage stay nil (deprecated)

	// Restore original data
	_, err := utilconversion.UnmarshalData(src, dst)
	return err
}

// ConvertTo converts this OpenStackMachine to the Hub version (v1beta2).
func (src *OpenStackMachine) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrav1beta2.OpenStackMachine)

	// Copy metadata and spec
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = *(*infrav1beta2.OpenStackMachineSpec)(unsafe.Pointer(&src.Spec))

	// Convert status
	dst.Status.Initialization = (*infrav1beta2.MachineInitialization)(unsafe.Pointer(src.Status.Initialization))
	dst.Status.InstanceID = src.Status.InstanceID
	dst.Status.InstanceState = (*infrav1beta2.InstanceState)(unsafe.Pointer(src.Status.InstanceState))
	dst.Status.Resolved = (*infrav1beta2.ResolvedMachineSpec)(unsafe.Pointer(src.Status.Resolved))
	dst.Status.Resources = (*infrav1beta2.MachineResources)(unsafe.Pointer(src.Status.Resources))
	dst.Status.Addresses = src.Status.Addresses

	// Convert conditions
	dst.Status.Conditions = infrav1beta2.ConvertConditionsToV1Beta2(src.Status.Conditions, src.Generation)

	return utilconversion.MarshalData(src, dst)
}

// ConvertFrom converts from the Hub version (v1beta2) to this version.
//
//nolint:revive // dst is the receiver here (converting FROM hub TO spoke)
func (dst *OpenStackMachine) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OpenStackMachine)

	// Copy metadata and spec
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = *(*OpenStackMachineSpec)(unsafe.Pointer(&src.Spec))

	// Convert status
	dst.Status.Initialization = (*MachineInitialization)(unsafe.Pointer(src.Status.Initialization))
	dst.Status.InstanceID = src.Status.InstanceID
	dst.Status.InstanceState = (*InstanceState)(unsafe.Pointer(src.Status.InstanceState))
	dst.Status.Resolved = (*ResolvedMachineSpec)(unsafe.Pointer(src.Status.Resolved))
	dst.Status.Resources = (*MachineResources)(unsafe.Pointer(src.Status.Resources))
	dst.Status.Addresses = src.Status.Addresses

	// Convert conditions
	dst.Status.Conditions = infrav1beta2.ConvertConditionsFromV1Beta2(src.Status.Conditions)

	// Derive deprecated fields
	dst.Status.Ready = infrav1beta2.IsReady(src.Status.Conditions)

	_, err := utilconversion.UnmarshalData(src, dst)
	return err
}

// ConvertTo converts this OpenStackClusterTemplate to the Hub version (v1beta2).
func (src *OpenStackClusterTemplate) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrav1beta2.OpenStackClusterTemplate)
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.Template.Spec = *(*infrav1beta2.OpenStackClusterSpec)(unsafe.Pointer(&src.Spec.Template.Spec))
	return nil
}

// ConvertFrom converts from the Hub version (v1beta2) to this version.
//
//nolint:revive // dst is the receiver here (converting FROM hub TO spoke)
func (dst *OpenStackClusterTemplate) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OpenStackClusterTemplate)
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.Template.Spec = *(*OpenStackClusterSpec)(unsafe.Pointer(&src.Spec.Template.Spec))
	return nil
}

// ConvertTo converts this OpenStackMachineTemplate to the Hub version (v1beta2).
func (src *OpenStackMachineTemplate) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrav1beta2.OpenStackMachineTemplate)
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.Template.Spec = *(*infrav1beta2.OpenStackMachineSpec)(unsafe.Pointer(&src.Spec.Template.Spec))
	return nil
}

// ConvertFrom converts from the Hub version (v1beta2) to this version.
//
//nolint:revive // dst is the receiver here (converting FROM hub TO spoke)
func (dst *OpenStackMachineTemplate) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OpenStackMachineTemplate)
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.Template.Spec = *(*OpenStackMachineSpec)(unsafe.Pointer(&src.Spec.Template.Spec))
	return nil
}

// ConvertTo converts this OpenStackClusterList to the Hub version (v1beta2).
func (src *OpenStackClusterList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrav1beta2.OpenStackClusterList)
	dst.ListMeta = src.ListMeta
	dst.Items = make([]infrav1beta2.OpenStackCluster, len(src.Items))
	for i := range src.Items {
		if err := src.Items[i].ConvertTo(&dst.Items[i]); err != nil {
			return err
		}
	}
	return nil
}

// ConvertFrom converts from the Hub version (v1beta2) to this version.
//
//nolint:revive // dst is the receiver here (converting FROM hub TO spoke)
func (dst *OpenStackClusterList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OpenStackClusterList)
	dst.ListMeta = src.ListMeta
	dst.Items = make([]OpenStackCluster, len(src.Items))
	for i := range src.Items {
		if err := dst.Items[i].ConvertFrom(&src.Items[i]); err != nil {
			return err
		}
	}
	return nil
}

// ConvertTo converts this OpenStackMachineList to the Hub version (v1beta2).
func (src *OpenStackMachineList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrav1beta2.OpenStackMachineList)
	dst.ListMeta = src.ListMeta
	dst.Items = make([]infrav1beta2.OpenStackMachine, len(src.Items))
	for i := range src.Items {
		if err := src.Items[i].ConvertTo(&dst.Items[i]); err != nil {
			return err
		}
	}
	return nil
}

// ConvertFrom converts from the Hub version (v1beta2) to this version.
//
//nolint:revive // dst is the receiver here (converting FROM hub TO spoke)
func (dst *OpenStackMachineList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OpenStackMachineList)
	dst.ListMeta = src.ListMeta
	dst.Items = make([]OpenStackMachine, len(src.Items))
	for i := range src.Items {
		if err := dst.Items[i].ConvertFrom(&src.Items[i]); err != nil {
			return err
		}
	}
	return nil
}

// ConvertTo converts this OpenStackClusterTemplateList to the Hub version (v1beta2).
func (src *OpenStackClusterTemplateList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrav1beta2.OpenStackClusterTemplateList)
	dst.ListMeta = src.ListMeta
	dst.Items = make([]infrav1beta2.OpenStackClusterTemplate, len(src.Items))
	for i := range src.Items {
		if err := src.Items[i].ConvertTo(&dst.Items[i]); err != nil {
			return err
		}
	}
	return nil
}

// ConvertFrom converts from the Hub version (v1beta2) to this version.
//
//nolint:revive // dst is the receiver here (converting FROM hub TO spoke)
func (dst *OpenStackClusterTemplateList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OpenStackClusterTemplateList)
	dst.ListMeta = src.ListMeta
	dst.Items = make([]OpenStackClusterTemplate, len(src.Items))
	for i := range src.Items {
		if err := dst.Items[i].ConvertFrom(&src.Items[i]); err != nil {
			return err
		}
	}
	return nil
}

// ConvertTo converts this OpenStackMachineTemplateList to the Hub version (v1beta2).
func (src *OpenStackMachineTemplateList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrav1beta2.OpenStackMachineTemplateList)
	dst.ListMeta = src.ListMeta
	dst.Items = make([]infrav1beta2.OpenStackMachineTemplate, len(src.Items))
	for i := range src.Items {
		if err := src.Items[i].ConvertTo(&dst.Items[i]); err != nil {
			return err
		}
	}
	return nil
}

// ConvertFrom converts from the Hub version (v1beta2) to this version.
//
//nolint:revive // dst is the receiver here (converting FROM hub TO spoke)
func (dst *OpenStackMachineTemplateList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OpenStackMachineTemplateList)
	dst.ListMeta = src.ListMeta
	dst.Items = make([]OpenStackMachineTemplate, len(src.Items))
	for i := range src.Items {
		if err := dst.Items[i].ConvertFrom(&src.Items[i]); err != nil {
			return err
		}
	}
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
