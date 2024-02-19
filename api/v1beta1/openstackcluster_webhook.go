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

package v1beta1

import (
	"fmt"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var _ = logf.Log.WithName("openstackcluster-resource")

func (r *OpenStackCluster) SetupWebhookWithManager(mgr manager.Manager) error {
	return builder.WebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-openstackcluster,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=openstackclusters,versions=v1beta1,name=validation.openstackcluster.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1
// +kubebuilder:webhook:verbs=create;update,path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-openstackcluster,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=openstackclusters,versions=v1beta1,name=default.openstackcluster.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1

var (
	_ webhook.Defaulter = &OpenStackCluster{}
	_ webhook.Validator = &OpenStackCluster{}
)

// Default satisfies the defaulting webhook interface.
func (r *OpenStackCluster) Default() {
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *OpenStackCluster) ValidateCreate() (admission.Warnings, error) {
	var allErrs field.ErrorList

	if r.Spec.ManagedSecurityGroups != nil {
		for _, rule := range r.Spec.ManagedSecurityGroups.AllNodesSecurityGroupRules {
			if rule.RemoteManagedGroups != nil && (rule.RemoteGroupID != nil || rule.RemoteIPPrefix != nil) {
				allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "managedSecurityGroups", "allNodesSecurityGroupRules"), "remoteManagedGroups cannot be used with remoteGroupID or remoteIPPrefix"))
			}
			if rule.RemoteGroupID != nil && (rule.RemoteManagedGroups != nil || rule.RemoteIPPrefix != nil) {
				allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "managedSecurityGroups", "allNodesSecurityGroupRules"), "remoteGroupID cannot be used with remoteManagedGroups or remoteIPPrefix"))
			}
			if rule.RemoteIPPrefix != nil && (rule.RemoteManagedGroups != nil || rule.RemoteGroupID != nil) {
				allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "managedSecurityGroups", "allNodesSecurityGroupRules"), "remoteIPPrefix cannot be used with remoteManagedGroups or remoteGroupID"))
			}
		}
	}

	return aggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, allErrs)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *OpenStackCluster) ValidateUpdate(oldRaw runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList
	old, ok := oldRaw.(*OpenStackCluster)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected an OpenStackCluster but got a %T", oldRaw))
	}

	// Allow changes to Spec.IdentityRef
	old.Spec.IdentityRef = OpenStackIdentityReference{}
	r.Spec.IdentityRef = OpenStackIdentityReference{}

	// Allow change only for the first time.
	if old.Spec.ControlPlaneEndpoint.Host == "" {
		old.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{}
		r.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{}
	}

	// Allow change only for the first time.
	if old.Spec.DisableAPIServerFloatingIP && old.Spec.APIServerFixedIP == "" {
		r.Spec.APIServerFixedIP = ""
	}

	// If API Server floating IP is disabled, allow the change of the API Server port only for the first time.
	if old.Spec.DisableAPIServerFloatingIP && old.Spec.APIServerPort == 0 && r.Spec.APIServerPort > 0 {
		r.Spec.APIServerPort = 0
	}

	// Allow to remove the bastion spec only if it was disabled before.
	if r.Spec.Bastion == nil {
		if old.Spec.Bastion != nil && old.Spec.Bastion.Enabled {
			allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "bastion"), "cannot be removed before disabling it"))
		}
	}

	// Allow changes to the bastion spec.
	old.Spec.Bastion = &Bastion{}
	r.Spec.Bastion = &Bastion{}

	// Allow changes to the managed allNodesSecurityGroupRules.
	if r.Spec.ManagedSecurityGroups != nil {
		old.Spec.ManagedSecurityGroups.AllNodesSecurityGroupRules = []SecurityGroupRuleSpec{}
		r.Spec.ManagedSecurityGroups.AllNodesSecurityGroupRules = []SecurityGroupRuleSpec{}

		// Allow change to the allowAllInClusterTraffic.
		old.Spec.ManagedSecurityGroups.AllowAllInClusterTraffic = false
		r.Spec.ManagedSecurityGroups.AllowAllInClusterTraffic = false
	}

	// Allow changes on AllowedCIDRs
	if r.Spec.APIServerLoadBalancer.Enabled {
		old.Spec.APIServerLoadBalancer.AllowedCIDRs = []string{}
		r.Spec.APIServerLoadBalancer.AllowedCIDRs = []string{}
	}

	// Allow changes to the availability zones.
	old.Spec.ControlPlaneAvailabilityZones = []string{}
	r.Spec.ControlPlaneAvailabilityZones = []string{}

	// Allow the scheduling to be changed from CAPI managed to Nova and
	// vice versa.
	old.Spec.ControlPlaneOmitAvailabilityZone = false
	r.Spec.ControlPlaneOmitAvailabilityZone = false

	// Allow change on the spec.APIServerFloatingIP only if it matches the current api server loadbalancer IP.
	if old.Status.APIServerLoadBalancer != nil && r.Spec.APIServerFloatingIP == old.Status.APIServerLoadBalancer.IP {
		r.Spec.APIServerFloatingIP = ""
		old.Spec.APIServerFloatingIP = ""
	}

	if !reflect.DeepEqual(old.Spec, r.Spec) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec"), "cannot be modified"))
	}

	return aggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *OpenStackCluster) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}
