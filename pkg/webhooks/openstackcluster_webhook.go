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

package webhooks

import (
	"context"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha7"
)

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1alpha7-openstackcluster,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=openstackclusters,versions=v1alpha7,name=validation.openstackcluster.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1

func SetupOpenStackClusterWebhook(mgr manager.Manager) error {
	return builder.WebhookManagedBy(mgr).
		For(&infrav1.OpenStackCluster{}).
		WithValidator(&openStackClusterWebhook{}).
		Complete()
}

type openStackClusterWebhook struct{}

// Compile-time assertion that openStackClusterWebhook implements webhook.CustomValidator.
var _ webhook.CustomValidator = &openStackClusterWebhook{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type.
func (*openStackClusterWebhook) ValidateCreate(_ context.Context, objRaw runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList

	newObj, err := castToOpenStackCluster(objRaw)
	if err != nil {
		return nil, err
	}

	if newObj.Spec.IdentityRef != nil && newObj.Spec.IdentityRef.Kind != defaultIdentityRefKind {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "identityRef", "kind"), "must be a Secret"))
	}

	return aggregateObjErrors(newObj.GroupVersionKind().GroupKind(), newObj.Name, allErrs)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type.
func (*openStackClusterWebhook) ValidateUpdate(_ context.Context, oldObjRaw, newObjRaw runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList
	oldObj, err := castToOpenStackCluster(oldObjRaw)
	if err != nil {
		return nil, err
	}
	newObj, err := castToOpenStackCluster(newObjRaw)
	if err != nil {
		return nil, err
	}

	if newObj.Spec.IdentityRef != nil && newObj.Spec.IdentityRef.Kind != defaultIdentityRefKind {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "identityRef", "kind"),
				newObj.Spec.IdentityRef, "must be a Secret"),
		)
	}

	// Allow changes to Spec.IdentityRef.Name.
	if oldObj.Spec.IdentityRef != nil && newObj.Spec.IdentityRef != nil {
		oldObj.Spec.IdentityRef.Name = ""
		newObj.Spec.IdentityRef.Name = ""
	}

	// Allow changes to Spec.IdentityRef if it was unset.
	if oldObj.Spec.IdentityRef == nil && newObj.Spec.IdentityRef != nil {
		oldObj.Spec.IdentityRef = &infrav1.OpenStackIdentityReference{}
		newObj.Spec.IdentityRef = &infrav1.OpenStackIdentityReference{}
	}

	if oldObj.Spec.IdentityRef != nil && newObj.Spec.IdentityRef == nil {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "identityRef"),
				newObj.Spec.IdentityRef, "field cannot be set to nil"),
		)
	}

	// Allow change only for the first time.
	if oldObj.Spec.ControlPlaneEndpoint.Host == "" {
		oldObj.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{}
		newObj.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{}
	}

	// Allow change only for the first time.
	if oldObj.Spec.DisableAPIServerFloatingIP && oldObj.Spec.APIServerFixedIP == "" {
		newObj.Spec.APIServerFixedIP = ""
	}

	// If API Server floating IP is disabled, allow the change of the API Server port only for the first time.
	if oldObj.Spec.DisableAPIServerFloatingIP && oldObj.Spec.APIServerPort == 0 && newObj.Spec.APIServerPort > 0 {
		newObj.Spec.APIServerPort = 0
	}

	// Allow changes to the bastion spec.
	oldObj.Spec.Bastion = &infrav1.Bastion{}
	newObj.Spec.Bastion = &infrav1.Bastion{}

	// Allow changes on AllowedCIDRs
	if newObj.Spec.APIServerLoadBalancer.Enabled {
		oldObj.Spec.APIServerLoadBalancer.AllowedCIDRs = []string{}
		newObj.Spec.APIServerLoadBalancer.AllowedCIDRs = []string{}
	}

	// Allow changes to the availability zones.
	oldObj.Spec.ControlPlaneAvailabilityZones = []string{}
	newObj.Spec.ControlPlaneAvailabilityZones = []string{}

	// Allow change to the allowAllInClusterTraffic.
	oldObj.Spec.AllowAllInClusterTraffic = false
	newObj.Spec.AllowAllInClusterTraffic = false

	// Allow change on the spec.APIServerFloatingIP only if it matches the current api server loadbalancer IP.
	if oldObj.Status.APIServerLoadBalancer != nil && newObj.Spec.APIServerFloatingIP == oldObj.Status.APIServerLoadBalancer.IP {
		newObj.Spec.APIServerFloatingIP = ""
		oldObj.Spec.APIServerFloatingIP = ""
	}

	if !reflect.DeepEqual(oldObj.Spec, newObj.Spec) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec"), "cannot be modified"))
	}

	return aggregateObjErrors(newObj.GroupVersionKind().GroupKind(), newObj.Name, allErrs)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type.
func (*openStackClusterWebhook) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func castToOpenStackCluster(obj runtime.Object) (*infrav1.OpenStackCluster, error) {
	cast, ok := obj.(*infrav1.OpenStackCluster)
	if !ok {
		return nil, fmt.Errorf("expected an OpenStackCluster but got a %T", obj)
	}
	return cast, nil
}
