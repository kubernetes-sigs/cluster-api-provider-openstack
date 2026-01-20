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
	"context"
	"fmt"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/cluster-api/util/topology"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	infrav1beta2 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
)

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1beta2-openstackmachinetemplate,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=openstackmachinetemplates,versions=v1beta2,name=validation.openstackmachinetemplate.v1beta2.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1

func SetupOpenStackMachineTemplateWebhookV1Beta2(mgr manager.Manager) error {
	return builder.WebhookManagedBy(mgr).
		For(&infrav1beta2.OpenStackMachineTemplate{}).
		WithValidator(&openStackMachineTemplateWebhookV1Beta2{}).
		Complete()
}

type openStackMachineTemplateWebhookV1Beta2 struct{}

// Compile-time assertion that openStackMachineTemplateWebhookV1Beta2 implements webhook.CustomValidator.
var _ webhook.CustomValidator = &openStackMachineTemplateWebhookV1Beta2{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type.
func (*openStackMachineTemplateWebhookV1Beta2) ValidateCreate(_ context.Context, objRaw runtime.Object) (admission.Warnings, error) {
	newObj, err := castToOpenStackMachineTemplateV1Beta2(objRaw)
	if err != nil {
		return nil, err
	}

	var allErrs field.ErrorList

	if newObj.Spec.Template.Spec.ProviderID != nil {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "template", "spec", "providerID"), "cannot be set in templates"))
	}

	return aggregateObjErrors(newObj.GroupVersionKind().GroupKind(), newObj.Name, allErrs)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type.
func (*openStackMachineTemplateWebhookV1Beta2) ValidateUpdate(ctx context.Context, oldObjRaw, newObjRaw runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList
	oldObj, err := castToOpenStackMachineTemplateV1Beta2(oldObjRaw)
	if err != nil {
		return nil, err
	}

	newObj, err := castToOpenStackMachineTemplateV1Beta2(newObjRaw)
	if err != nil {
		return nil, err
	}

	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a admission.Request inside context: %v", err))
	}

	if !topology.IsDryRunRequest(req, newObj) &&
		!reflect.DeepEqual(newObj.Spec.Template.Spec, oldObj.Spec.Template.Spec) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "template", "spec"), newObj.Spec.Template.Spec, "OpenStackMachineTemplate spec.template.spec field is immutable. Please create a new resource instead. Ref doc: https://cluster-api.sigs.k8s.io/tasks/change-machine-template.html"),
		)
	}

	return aggregateObjErrors(newObj.GroupVersionKind().GroupKind(), newObj.Name, allErrs)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type.
func (*openStackMachineTemplateWebhookV1Beta2) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func castToOpenStackMachineTemplateV1Beta2(obj runtime.Object) (*infrav1beta2.OpenStackMachineTemplate, error) {
	cast, ok := obj.(*infrav1beta2.OpenStackMachineTemplate)
	if !ok {
		return nil, fmt.Errorf("expected an OpenStackMachineTemplate but got a %T", obj)
	}
	return cast, nil
}
