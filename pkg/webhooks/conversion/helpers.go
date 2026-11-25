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
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"

	infrav1beta1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

// marshalSpokeData stashes src on dst like utilconversion.MarshalData, but
// without status. ConvertFrom never restores status from the annotation, so
// persisting a copy of it would only keep stale or client-supplied data.
func marshalSpokeData(src, dst metav1.Object) error {
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(src)
	if err != nil {
		return err
	}
	delete(u, "metadata")
	delete(u, "status")

	data, err := json.Marshal(u)
	if err != nil {
		return err
	}
	annotations := dst.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[utilconversion.DataAnnotation] = string(data)
	dst.SetAnnotations(annotations)
	return nil
}

// restoreOpenStackClusterSpec restores the spoke-only fields of an
// infrav1beta1.OpenStackClusterSpec.
func restoreOpenStackClusterSpec(previous, dst *infrav1beta1.OpenStackClusterSpec) {
	if previous.Bastion != nil && dst.Bastion != nil {
		restoreMachineSpecFlavor(previous.Bastion.Spec, dst.Bastion.Spec)
	}
}

// restoreMachineSpecFlavor restores the v1beta1 flavor name when both Flavor
// and FlavorID were set. v1beta2 keeps only the ID in that case. The name is
// restored only alongside the same ID, where ConvertTo ignores it again, so
// it can't change the hub.
func restoreMachineSpecFlavor(previous, dst *infrav1beta1.OpenStackMachineSpec) {
	if previous == nil || dst == nil {
		return
	}
	if previous.Flavor != nil && *previous.Flavor != "" && dst.Flavor == nil &&
		previous.FlavorID != nil && dst.FlavorID != nil && *previous.FlavorID == *dst.FlavorID {
		dst.Flavor = ptr.To(*previous.Flavor)
	}
}
