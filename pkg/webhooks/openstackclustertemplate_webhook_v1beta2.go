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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	infrav1beta2 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
)

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1beta2-openstackclustertemplate,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=openstackclustertemplates,versions=v1beta2,name=validation.openstackclustertemplate.v1beta2.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1

func SetupOpenStackClusterTemplateWebhookV1Beta2(mgr manager.Manager) error {
	return builder.WebhookManagedBy(mgr).
		For(&infrav1beta2.OpenStackClusterTemplate{}).
		WithValidator(&openStackClusterTemplateWebhookV1Beta2{}).
		Complete()
}

type openStackClusterTemplateWebhookV1Beta2 struct{}

// Compile-time assertion that openStackClusterTemplateWebhookV1Beta2 implements webhook.CustomValidator.
var _ webhook.CustomValidator = &openStackClusterTemplateWebhookV1Beta2{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type.
func (*openStackClusterTemplateWebhookV1Beta2) ValidateCreate(_ context.Context, objRaw runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList
	newObj, err := castToOpenStackClusterTemplateV1Beta2(objRaw)
	if err != nil {
		return nil, err
	}

	return aggregateObjErrors(newObj.GroupVersionKind().GroupKind(), newObj.Name, allErrs)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type.
func (*openStackClusterTemplateWebhookV1Beta2) ValidateUpdate(_ context.Context, oldObjRaw, newObjRaw runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList
	oldObj, err := castToOpenStackClusterTemplateV1Beta2(oldObjRaw)
	if err != nil {
		return nil, err
	}
	newObj, err := castToOpenStackClusterTemplateV1Beta2(newObjRaw)
	if err != nil {
		return nil, err
	}

	if !reflect.DeepEqual(newObj.Spec.Template.Spec, oldObj.Spec.Template.Spec) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("OpenStackClusterTemplate", "spec", "template", "spec"), newObj, "OpenStackClusterTemplate spec.template.spec field is immutable. Please create new resource instead."),
		)
	}

	return aggregateObjErrors(newObj.GroupVersionKind().GroupKind(), newObj.Name, allErrs)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type.
func (*openStackClusterTemplateWebhookV1Beta2) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func castToOpenStackClusterTemplateV1Beta2(obj runtime.Object) (*infrav1beta2.OpenStackClusterTemplate, error) {
	cast, ok := obj.(*infrav1beta2.OpenStackClusterTemplate)
	if !ok {
		return nil, fmt.Errorf("expected an OpenStackClusterTemplate but got a %T", obj)
	}
	return cast, nil
}
