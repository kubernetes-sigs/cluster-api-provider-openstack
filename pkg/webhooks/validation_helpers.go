/*
Copyright 2026 The Kubernetes Authors.

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

package webhooks

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	infrav1beta2 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
)

const rootVolumeName = "root"

// validateSecurityGroupRulesRemoteMutualExclusionV1Beta1 validates that remote* fields are mutually exclusive for v1beta1.
func validateSecurityGroupRulesRemoteMutualExclusionV1Beta1(rules []infrav1.SecurityGroupRuleSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	for i := range rules {
		rule := &rules[i]
		rulePath := fldPath.Index(i)
		count := 0
		if rule.RemoteManagedGroups != nil {
			count++
		}
		if rule.RemoteGroupID != nil {
			count++
		}
		if rule.RemoteIPPrefix != nil {
			count++
		}
		if count > 1 {
			allErrs = append(allErrs, field.Forbidden(rulePath, "only one of remoteManagedGroups, remoteGroupID, or remoteIPPrefix can be set"))
		}
	}
	return allErrs
}

// validateSecurityGroupRulesRemoteMutualExclusionV1Beta2 validates that remote* fields are mutually exclusive for v1beta2.
func validateSecurityGroupRulesRemoteMutualExclusionV1Beta2(rules []infrav1beta2.SecurityGroupRuleSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	for i := range rules {
		rule := &rules[i]
		rulePath := fldPath.Index(i)
		count := 0
		if rule.RemoteManagedGroups != nil {
			count++
		}
		if rule.RemoteGroupID != nil {
			count++
		}
		if rule.RemoteIPPrefix != nil {
			count++
		}
		if count > 1 {
			allErrs = append(allErrs, field.Forbidden(rulePath, "only one of remoteManagedGroups, remoteGroupID, or remoteIPPrefix can be set"))
		}
	}
	return allErrs
}
