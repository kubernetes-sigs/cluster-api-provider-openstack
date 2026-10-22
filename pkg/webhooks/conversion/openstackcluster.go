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

// OpenStackCluster is the hub/spoke converter for OpenStackCluster.
var OpenStackCluster = conversion.NewHubSpokeConverter(&infrav1.OpenStackCluster{},
	conversion.NewSpokeConverter(&infrav1beta1.OpenStackCluster{}, ConvertOpenStackClusterHubToV1Beta1, ConvertOpenStackClusterV1Beta1ToHub),
)

// ConvertOpenStackClusterV1Beta1ToHub converts a v1beta1 OpenStackCluster to the hub version.
func ConvertOpenStackClusterV1Beta1ToHub(_ context.Context, src *infrav1beta1.OpenStackCluster, dst *infrav1.OpenStackCluster) error {
	if err := infrav1beta1.Convert_v1beta1_OpenStackCluster_To_v1beta2_OpenStackCluster(src, dst, nil); err != nil {
		return err
	}
	for i := range dst.Status.Conditions {
		dst.Status.Conditions[i].ObservedGeneration = src.Generation
	}

	restored := &infrav1.OpenStackCluster{}
	if _, err := utilconversion.UnmarshalData(src, restored); err != nil {
		return err
	}

	return utilconversion.MarshalData(src, dst)
}

// ConvertOpenStackClusterHubToV1Beta1 converts a hub OpenStackCluster to v1beta1.
func ConvertOpenStackClusterHubToV1Beta1(_ context.Context, src *infrav1.OpenStackCluster, dst *infrav1beta1.OpenStackCluster) error {
	if err := infrav1beta1.Convert_v1beta2_OpenStackCluster_To_v1beta1_OpenStackCluster(src, dst, nil); err != nil {
		return err
	}
	if _, err := utilconversion.UnmarshalData(src, dst); err != nil {
		return err
	}
	return utilconversion.MarshalData(src, dst)
}
