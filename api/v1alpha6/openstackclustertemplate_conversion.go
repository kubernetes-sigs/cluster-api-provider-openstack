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
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/conversion"
)

var _ ctrlconversion.Convertible = &OpenStackClusterTemplate{}

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

var _ ctrlconversion.Convertible = &OpenStackClusterTemplateList{}

func (r *OpenStackClusterTemplateList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterTemplateList)
	return Convert_v1alpha6_OpenStackClusterTemplateList_To_v1beta1_OpenStackClusterTemplateList(r, dst, nil)
}

func (r *OpenStackClusterTemplateList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterTemplateList)
	return Convert_v1beta1_OpenStackClusterTemplateList_To_v1alpha6_OpenStackClusterTemplateList(src, r, nil)
}

/* Restorers */

var v1alpha6OpenStackClusterTemplateRestorer = conversion.RestorerFor[*OpenStackClusterTemplate]{
	"spec": conversion.HashedFieldRestorer(
		func(c *OpenStackClusterTemplate) *OpenStackClusterSpec {
			return &c.Spec.Template.Spec
		},
		restorev1alpha6ClusterSpec,
	),
}

var v1beta1OpenStackClusterTemplateRestorer = conversion.RestorerFor[*infrav1.OpenStackClusterTemplate]{
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
