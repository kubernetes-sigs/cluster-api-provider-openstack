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

package conversion

import (
	"context"

	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/webhook/conversion"

	infrav1beta1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
)

// OpenStackMachineTemplate is the hub/spoke converter for OpenStackMachineTemplate.
var OpenStackMachineTemplate = conversion.NewHubSpokeConverter(&infrav1.OpenStackMachineTemplate{},
	conversion.NewSpokeConverter(&infrav1beta1.OpenStackMachineTemplate{}, ConvertOpenStackMachineTemplateHubToV1Beta1, ConvertOpenStackMachineTemplateV1Beta1ToHub),
)

// ConvertOpenStackMachineTemplateV1Beta1ToHub converts a v1beta1 OpenStackMachineTemplate to the hub version.
func ConvertOpenStackMachineTemplateV1Beta1ToHub(_ context.Context, src *infrav1beta1.OpenStackMachineTemplate, dst *infrav1.OpenStackMachineTemplate) error {
	if err := infrav1beta1.Convert_v1beta1_OpenStackMachineTemplate_To_v1beta2_OpenStackMachineTemplate(src, dst, nil); err != nil {
		return err
	}

	restored := &infrav1.OpenStackMachineTemplate{}
	if _, err := utilconversion.UnmarshalData(src, restored); err != nil {
		return err
	}

	return utilconversion.MarshalData(src, dst)
}

// ConvertOpenStackMachineTemplateHubToV1Beta1 converts a hub OpenStackMachineTemplate to v1beta1.
func ConvertOpenStackMachineTemplateHubToV1Beta1(_ context.Context, src *infrav1.OpenStackMachineTemplate, dst *infrav1beta1.OpenStackMachineTemplate) error {
	if err := infrav1beta1.Convert_v1beta2_OpenStackMachineTemplate_To_v1beta1_OpenStackMachineTemplate(src, dst, nil); err != nil {
		return err
	}
	if _, err := utilconversion.UnmarshalData(src, dst); err != nil {
		return err
	}
	return utilconversion.MarshalData(src, dst)
}
