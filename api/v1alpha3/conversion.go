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
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha4"
)

var _ conversion.Convertible = &OpenStackCluster{}

func (r *OpenStackCluster) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha4.OpenStackCluster)

	return Convert_v1alpha3_OpenStackCluster_To_v1alpha4_OpenStackCluster(r, dst, nil)
}

func (r *OpenStackCluster) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha4.OpenStackCluster)

	return Convert_v1alpha4_OpenStackCluster_To_v1alpha3_OpenStackCluster(src, r, nil)
}

var _ conversion.Convertible = &OpenStackClusterList{}

func (r *OpenStackClusterList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha4.OpenStackClusterList)

	return Convert_v1alpha3_OpenStackClusterList_To_v1alpha4_OpenStackClusterList(r, dst, nil)
}

func (r *OpenStackClusterList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha4.OpenStackClusterList)

	return Convert_v1alpha4_OpenStackClusterList_To_v1alpha3_OpenStackClusterList(src, r, nil)
}

var _ conversion.Convertible = &OpenStackMachine{}

func (r *OpenStackMachine) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha4.OpenStackMachine)

	return Convert_v1alpha3_OpenStackMachine_To_v1alpha4_OpenStackMachine(r, dst, nil)
}

func (r *OpenStackMachine) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha4.OpenStackMachine)

	return Convert_v1alpha4_OpenStackMachine_To_v1alpha3_OpenStackMachine(src, r, nil)
}

var _ conversion.Convertible = &OpenStackMachineList{}

func (r *OpenStackMachineList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha4.OpenStackMachineList)

	return Convert_v1alpha3_OpenStackMachineList_To_v1alpha4_OpenStackMachineList(r, dst, nil)
}

func (r *OpenStackMachineList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha4.OpenStackMachineList)

	return Convert_v1alpha4_OpenStackMachineList_To_v1alpha3_OpenStackMachineList(src, r, nil)
}

var _ conversion.Convertible = &OpenStackMachineTemplate{}

func (r *OpenStackMachineTemplate) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha4.OpenStackMachineTemplate)

	return Convert_v1alpha3_OpenStackMachineTemplate_To_v1alpha4_OpenStackMachineTemplate(r, dst, nil)
}

func (r *OpenStackMachineTemplate) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha4.OpenStackMachineTemplate)

	return Convert_v1alpha4_OpenStackMachineTemplate_To_v1alpha3_OpenStackMachineTemplate(src, r, nil)
}

var _ conversion.Convertible = &OpenStackMachineTemplateList{}

func (r *OpenStackMachineTemplateList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha4.OpenStackMachineTemplateList)

	return Convert_v1alpha3_OpenStackMachineTemplateList_To_v1alpha4_OpenStackMachineTemplateList(r, dst, nil)
}

func (r *OpenStackMachineTemplateList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha4.OpenStackMachineTemplateList)

	return Convert_v1alpha4_OpenStackMachineTemplateList_To_v1alpha3_OpenStackMachineTemplateList(src, r, nil)
}
