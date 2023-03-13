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

import "k8s.io/apimachinery/pkg/conversion"

// Hub marks OpenStackCluster as a conversion hub.
func (*OpenStackCluster) Hub() {}

// Hub marks OpenStackClusterList as a conversion hub.
func (*OpenStackClusterList) Hub() {}

// Hub marks OpenStackClusterTemplate as a conversion hub.
func (*OpenStackClusterTemplate) Hub() {}

// Hub marks OpenStackClusterTemplateList as a conversion hub.
func (*OpenStackClusterTemplateList) Hub() {}

// Hub marks OpenStackMachine as a conversion hub.
func (*OpenStackMachine) Hub() {}

// Hub marks OpenStackMachineList as a conversion hub.
func (*OpenStackMachineList) Hub() {}

// Hub marks OpenStackMachineTemplate as a conversion hub.
func (*OpenStackMachineTemplate) Hub() {}

// Hub marks OpenStackMachineTemplateList as a conversion hub.
func (*OpenStackMachineTemplateList) Hub() {}

func Convert_Slice_string_To_Slice_v1alpha7_SecurityGroupParam(in *[]string, out *[]SecurityGroupParam, s conversion.Scope) error {
	if in != nil {
		for _, securityGroupUID := range *in {
			*out = append(*out, SecurityGroupParam{UUID: securityGroupUID})
		}
	}
	return nil
}

func Convert_Slice_v1alpha7_SecurityGroupParam_To_Slice_string(in *[]SecurityGroupParam, out *[]string, s conversion.Scope) error {
	if in != nil {
		for _, v1alpha7SecurityGroups := range *in {
			// If a UUID is specified, use it. We cannot do anything about other fields so they are dropped.
			if v1alpha7SecurityGroups.UUID != "" {
				*out = append(*out, v1alpha7SecurityGroups.UUID)
			}
		}
	}
	return nil
}
