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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	conversion "k8s.io/apimachinery/pkg/conversion"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha7"
)

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
}

func restorev1alpha7ClusterSpec(previous *infrav1.OpenStackClusterSpec, dst *infrav1.OpenStackClusterSpec) {
	// APIServerLoadBalancer.Provider is new in v1alpha7
	dst.APIServerLoadBalancer.Provider = previous.APIServerLoadBalancer.Provider
	dst.Router = previous.Router
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
	}

	return nil
}

var _ ctrlconversion.Convertible = &OpenStackMachine{}

func (r *OpenStackMachine) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*infrav1.OpenStackMachine)
	var previous infrav1.OpenStackMachine
	_, err := convertAndRestore(r, dst, &previous, Convert_v1alpha6_OpenStackMachine_To_v1alpha7_OpenStackMachine)
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
	_, err := convertAndRestore(r, dst, &previous, Convert_v1alpha6_OpenStackMachineTemplate_To_v1alpha7_OpenStackMachineTemplate)
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
	return autoConvert_v1alpha6_OpenStackMachineSpec_To_v1alpha7_OpenStackMachineSpec(in, out, s)
}

func Convert_v1alpha7_APIServerLoadBalancer_To_v1alpha6_APIServerLoadBalancer(in *infrav1.APIServerLoadBalancer, out *APIServerLoadBalancer, s conversion.Scope) error {
	// Provider have been added in v1alpha7 but have no equivalent in v1alpha6
	return autoConvert_v1alpha7_APIServerLoadBalancer_To_v1alpha6_APIServerLoadBalancer(in, out, s)
}

func Convert_v1alpha7_OpenStackClusterSpec_To_v1alpha6_OpenStackClusterSpec(in *infrav1.OpenStackClusterSpec, out *OpenStackClusterSpec, s conversion.Scope) error {
	return autoConvert_v1alpha7_OpenStackClusterSpec_To_v1alpha6_OpenStackClusterSpec(in, out, s)
}
