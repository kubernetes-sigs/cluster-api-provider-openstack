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
	// corev1 "k8s.io/api/core/v1"
	// conversion "k8s.io/apimachinery/pkg/conversion"
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	"sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

var _ ctrlconversion.Convertible = &OpenStackCluster{}

func (r *OpenStackCluster) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*v1beta1.OpenStackCluster)

	return Convert_v1alpha4_OpenStackCluster_To_v1beta1_OpenStackCluster(r, dst, nil)
}

func (r *OpenStackCluster) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*v1beta1.OpenStackCluster)

	return Convert_v1beta1_OpenStackCluster_To_v1alpha4_OpenStackCluster(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackClusterList{}

func (r *OpenStackClusterList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*v1beta1.OpenStackClusterList)

	return Convert_v1alpha4_OpenStackClusterList_To_v1beta1_OpenStackClusterList(r, dst, nil)
}

func (r *OpenStackClusterList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*v1beta1.OpenStackClusterList)

	return Convert_v1beta1_OpenStackClusterList_To_v1alpha4_OpenStackClusterList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackMachine{}

func (r *OpenStackMachine) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*v1beta1.OpenStackMachine)

	return Convert_v1alpha4_OpenStackMachine_To_v1beta1_OpenStackMachine(r, dst, nil)
}

func (r *OpenStackMachine) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*v1beta1.OpenStackMachine)

	return Convert_v1beta1_OpenStackMachine_To_v1alpha4_OpenStackMachine(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackMachineList{}

func (r *OpenStackMachineList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*v1beta1.OpenStackMachineList)

	return Convert_v1alpha4_OpenStackMachineList_To_v1beta1_OpenStackMachineList(r, dst, nil)
}

func (r *OpenStackMachineList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*v1beta1.OpenStackMachineList)

	return Convert_v1beta1_OpenStackMachineList_To_v1alpha4_OpenStackMachineList(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplate{}

func (r *OpenStackMachineTemplate) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*v1beta1.OpenStackMachineTemplate)

	return Convert_v1alpha4_OpenStackMachineTemplate_To_v1beta1_OpenStackMachineTemplate(r, dst, nil)
}

func (r *OpenStackMachineTemplate) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*v1beta1.OpenStackMachineTemplate)

	return Convert_v1beta1_OpenStackMachineTemplate_To_v1alpha4_OpenStackMachineTemplate(src, r, nil)
}

var _ ctrlconversion.Convertible = &OpenStackMachineTemplateList{}

func (r *OpenStackMachineTemplateList) ConvertTo(dstRaw ctrlconversion.Hub) error {
	dst := dstRaw.(*v1beta1.OpenStackMachineTemplateList)

	return Convert_v1alpha4_OpenStackMachineTemplateList_To_v1beta1_OpenStackMachineTemplateList(r, dst, nil)
}

func (r *OpenStackMachineTemplateList) ConvertFrom(srcRaw ctrlconversion.Hub) error {
	src := srcRaw.(*v1beta1.OpenStackMachineTemplateList)

	return Convert_v1beta1_OpenStackMachineTemplateList_To_v1alpha4_OpenStackMachineTemplateList(src, r, nil)
}
