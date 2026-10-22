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

// OpenStackClusterTemplate is the hub/spoke converter for OpenStackClusterTemplate.
var OpenStackClusterTemplate = conversion.NewHubSpokeConverter(&infrav1.OpenStackClusterTemplate{},
	conversion.NewSpokeConverter(&infrav1beta1.OpenStackClusterTemplate{}, ConvertOpenStackClusterTemplateHubToV1Beta1, ConvertOpenStackClusterTemplateV1Beta1ToHub),
)

// ConvertOpenStackClusterTemplateV1Beta1ToHub converts a v1beta1 OpenStackClusterTemplate to the hub version.
func ConvertOpenStackClusterTemplateV1Beta1ToHub(_ context.Context, src *infrav1beta1.OpenStackClusterTemplate, dst *infrav1.OpenStackClusterTemplate) error {
	if err := infrav1beta1.Convert_v1beta1_OpenStackClusterTemplate_To_v1beta2_OpenStackClusterTemplate(src, dst, nil); err != nil {
		return err
	}

	// Restore hub-only fields (spec.template.metadata) stashed on the
	// spoke by ConvertOpenStackClusterTemplateHubToV1Beta1. UnmarshalData also deletes the annotation,
	// so it can't leak into the stash below.
	restored := &infrav1.OpenStackClusterTemplate{}
	ok, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}
	if ok {
		dst.Spec.Template.ObjectMeta = restored.Spec.Template.ObjectMeta
	}

	// Stash the v1beta1 object on the hub to preserve spoke-only fields
	// (e.g. bastion flavor name) across a later hub->spoke conversion.
	return utilconversion.MarshalData(src, dst)
}

// ConvertOpenStackClusterTemplateHubToV1Beta1 converts a hub OpenStackClusterTemplate to v1beta1.
func ConvertOpenStackClusterTemplateHubToV1Beta1(_ context.Context, src *infrav1.OpenStackClusterTemplate, dst *infrav1beta1.OpenStackClusterTemplate) error {
	if err := infrav1beta1.Convert_v1beta2_OpenStackClusterTemplate_To_v1beta1_OpenStackClusterTemplate(src, dst, nil); err != nil {
		return err
	}

	// Restore spoke-only fields stashed on the hub by ConvertOpenStackClusterTemplateV1Beta1ToHub.
	if _, err := utilconversion.UnmarshalData(src, dst); err != nil {
		return err
	}

	// Stash the hub object on the spoke so hub-only fields
	// (spec.template.metadata) survive the round trip.
	return utilconversion.MarshalData(src, dst)
}
