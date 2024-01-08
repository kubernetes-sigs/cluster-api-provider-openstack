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
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha8"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/conversion"
)

var _ ctrlconversion.Convertible = &OpenStackCluster{}

var v1alpha7OpenStackClusterRestorer = conversion.RestorerFor[*OpenStackCluster]{}

var v1alpha8OpenStackClusterRestorer = conversion.RestorerFor[*infrav1.OpenStackCluster]{
	"bastion": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackCluster) **infrav1.Bastion {
			return &c.Spec.Bastion
		},
		restorev1alpha8Bastion,
	),
	"spec": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackCluster) *infrav1.OpenStackClusterSpec {
			return &c.Spec
		},
		restorev1alpha8ClusterSpec,
	),

	// No equivalent in v1alpha7
	"bastionrefresources": conversion.UnconditionalFieldRestorer(
		func(c *infrav1.OpenStackCluster) *infrav1.ReferencedMachineResources {
			if c.Status.Bastion == nil {
				return nil
			}
			return &c.Status.Bastion.ReferencedResources
		},
	),
}

func restorev1alpha8MachineSpec(previous *infrav1.OpenStackMachineSpec, dst *infrav1.OpenStackMachineSpec) {
	dst.ServerGroup = previous.ServerGroup
}

func restorev1alpha8Bastion(previous **infrav1.Bastion, dst **infrav1.Bastion) {
	if *previous != nil && *dst != nil {
		restorev1alpha8MachineSpec(&(*previous).Instance, &(*dst).Instance)
	}
}

func restorev1alpha8ClusterSpec(previous *infrav1.OpenStackClusterSpec, dst *infrav1.OpenStackClusterSpec) {
	prevBastion := previous.Bastion
	dstBastion := dst.Bastion
	if prevBastion != nil && dstBastion != nil {
		restorev1alpha8MachineSpec(&prevBastion.Instance, &dstBastion.Instance)
	}
}

func (r *OpenStackCluster) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackCluster)

	return conversion.ConvertAndRestore(
		r, dst,
		Convert_v1alpha7_OpenStackCluster_To_v1alpha8_OpenStackCluster, Convert_v1alpha8_OpenStackCluster_To_v1alpha7_OpenStackCluster,
		v1alpha7OpenStackClusterRestorer, v1alpha8OpenStackClusterRestorer,
	)
}

func (r *OpenStackCluster) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackCluster)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1alpha8_OpenStackCluster_To_v1alpha7_OpenStackCluster, Convert_v1alpha7_OpenStackCluster_To_v1alpha8_OpenStackCluster,
		v1alpha8OpenStackClusterRestorer, v1alpha7OpenStackClusterRestorer,
	)
}

var _ ctrlconversion.Convertible = &OpenStackClusterList{}

func (r *OpenStackClusterList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1alpha7_OpenStackClusterList_To_v1alpha8_OpenStackClusterList(r, dst, nil)
}

func (r *OpenStackClusterList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterList)

	return Convert_v1alpha8_OpenStackClusterList_To_v1alpha7_OpenStackClusterList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackClusterTemplate{}

var v1alpha7OpenStackClusterTemplateRestorer = conversion.RestorerFor[*OpenStackClusterTemplate]{}

var v1alpha8OpenStackClusterTemplateRestorer = conversion.RestorerFor[*infrav1.OpenStackClusterTemplate]{
	"bastion": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackClusterTemplate) **infrav1.Bastion {
			return &c.Spec.Template.Spec.Bastion
		},
		restorev1alpha8Bastion,
	),
}

func (r *OpenStackClusterTemplate) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackClusterTemplate)

	return conversion.ConvertAndRestore(
		r, dst,
		Convert_v1alpha7_OpenStackClusterTemplate_To_v1alpha8_OpenStackClusterTemplate, Convert_v1alpha8_OpenStackClusterTemplate_To_v1alpha7_OpenStackClusterTemplate,
		v1alpha7OpenStackClusterTemplateRestorer, v1alpha8OpenStackClusterTemplateRestorer,
	)
}

func (r *OpenStackClusterTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackClusterTemplate)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1alpha8_OpenStackClusterTemplate_To_v1alpha7_OpenStackClusterTemplate, Convert_v1alpha7_OpenStackClusterTemplate_To_v1alpha8_OpenStackClusterTemplate,
		v1alpha8OpenStackClusterTemplateRestorer, v1alpha7OpenStackClusterTemplateRestorer,
	)
}

var _ ctrlconversion.Convertible = &OpenStackMachine{}

var v1alpha7OpenStackMachineRestorer = conversion.RestorerFor[*OpenStackMachine]{}

var v1alpha8OpenStackMachineRestorer = conversion.RestorerFor[*infrav1.OpenStackMachine]{
	"spec": conversion.HashedFieldRestorer(
		func(c *infrav1.OpenStackMachine) *infrav1.OpenStackMachineSpec {
			return &c.Spec
		},
		restorev1alpha8MachineSpec,
	),

	// No equivalent in v1alpha7
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
		Convert_v1alpha7_OpenStackMachine_To_v1alpha8_OpenStackMachine, Convert_v1alpha8_OpenStackMachine_To_v1alpha7_OpenStackMachine,
		v1alpha7OpenStackMachineRestorer, v1alpha8OpenStackMachineRestorer,
	)
}

func (r *OpenStackMachine) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachine)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1alpha8_OpenStackMachine_To_v1alpha7_OpenStackMachine, Convert_v1alpha7_OpenStackMachine_To_v1alpha8_OpenStackMachine,
		v1alpha8OpenStackMachineRestorer, v1alpha7OpenStackMachineRestorer,
	)
}

var _ ctrlconversion.Convertible = &OpenStackMachineList{}

func (r *OpenStackMachineList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineList)
	return Convert_v1alpha7_OpenStackMachineList_To_v1alpha8_OpenStackMachineList(r, dst, nil)
}

func (r *OpenStackMachineList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineList)
	return Convert_v1alpha8_OpenStackMachineList_To_v1alpha7_OpenStackMachineList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplate{}

var v1alpha7OpenStackMachineTemplateRestorer = conversion.RestorerFor[*OpenStackMachineTemplate]{}

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
		Convert_v1alpha7_OpenStackMachineTemplate_To_v1alpha8_OpenStackMachineTemplate, Convert_v1alpha8_OpenStackMachineTemplate_To_v1alpha7_OpenStackMachineTemplate,
		v1alpha7OpenStackMachineTemplateRestorer, v1alpha8OpenStackMachineTemplateRestorer,
	)
}

func (r *OpenStackMachineTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplate)

	return conversion.ConvertAndRestore(
		src, r,
		Convert_v1alpha8_OpenStackMachineTemplate_To_v1alpha7_OpenStackMachineTemplate, Convert_v1alpha7_OpenStackMachineTemplate_To_v1alpha8_OpenStackMachineTemplate,
		v1alpha8OpenStackMachineTemplateRestorer, v1alpha7OpenStackMachineTemplateRestorer,
	)
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplateList{}

func (r *OpenStackMachineTemplateList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachineTemplateList)
	return Convert_v1alpha7_OpenStackMachineTemplateList_To_v1alpha8_OpenStackMachineTemplateList(r, dst, nil)
}

func (r *OpenStackMachineTemplateList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*infrav1.OpenStackMachineTemplateList)
	return Convert_v1alpha8_OpenStackMachineTemplateList_To_v1alpha7_OpenStackMachineTemplateList(src, r, nil)
}

func Convert_v1alpha8_OpenStackMachineSpec_To_v1alpha7_OpenStackMachineSpec(in *infrav1.OpenStackMachineSpec, out *OpenStackMachineSpec, s apiconversion.Scope) error {
	err := autoConvert_v1alpha8_OpenStackMachineSpec_To_v1alpha7_OpenStackMachineSpec(in, out, s)
	if err != nil {
		return err
	}

	if in.ServerGroup != nil {
		out.ServerGroupID = in.ServerGroup.ID
	}

	return nil
}

func Convert_v1alpha7_OpenStackMachineSpec_To_v1alpha8_OpenStackMachineSpec(in *OpenStackMachineSpec, out *infrav1.OpenStackMachineSpec, s apiconversion.Scope) error {
	err := autoConvert_v1alpha7_OpenStackMachineSpec_To_v1alpha8_OpenStackMachineSpec(in, out, s)
	if err != nil {
		return err
	}

	if in.ServerGroupID != "" {
		out.ServerGroup = &infrav1.ServerGroupFilter{ID: in.ServerGroupID}
	} else {
		out.ServerGroup = nil
	}

	return nil
}

func Convert_v1alpha8_OpenStackMachineStatus_To_v1alpha7_OpenStackMachineStatus(in *infrav1.OpenStackMachineStatus, out *OpenStackMachineStatus, s apiconversion.Scope) error {
	// ReferencedResources have no equivalent in v1alpha7
	return autoConvert_v1alpha8_OpenStackMachineStatus_To_v1alpha7_OpenStackMachineStatus(in, out, s)
}

func Convert_v1alpha8_BastionStatus_To_v1alpha7_BastionStatus(in *infrav1.BastionStatus, out *BastionStatus, s apiconversion.Scope) error {
	// ReferencedResources have no equivalent in v1alpha7
	return autoConvert_v1alpha8_BastionStatus_To_v1alpha7_BastionStatus(in, out, s)
}

func Convert_v1alpha7_Bastion_To_v1alpha8_Bastion(in *Bastion, out *infrav1.Bastion, s apiconversion.Scope) error {
	err := autoConvert_v1alpha7_Bastion_To_v1alpha8_Bastion(in, out, s)
	if err != nil {
		return err
	}

	if in.Instance.ServerGroupID != "" {
		out.Instance.ServerGroup = &infrav1.ServerGroupFilter{ID: in.Instance.ServerGroupID}
	} else {
		out.Instance.ServerGroup = nil
	}

	return nil
}

func Convert_v1alpha8_Bastion_To_v1alpha7_Bastion(in *infrav1.Bastion, out *Bastion, s apiconversion.Scope) error {
	err := autoConvert_v1alpha8_Bastion_To_v1alpha7_Bastion(in, out, s)
	if err != nil {
		return err
	}

	if in.Instance.ServerGroup != nil && in.Instance.ServerGroup.ID != "" {
		out.Instance.ServerGroupID = in.Instance.ServerGroup.ID
	}

	return nil
}
