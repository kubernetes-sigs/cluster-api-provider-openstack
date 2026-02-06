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
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	infrav1beta2 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
)

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1beta2-openstackmachine,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=openstackmachines,versions=v1beta2,name=validation.openstackmachine.v1beta2.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1

func SetupOpenStackMachineWebhookV1Beta2(mgr manager.Manager) error {
	return builder.WebhookManagedBy(mgr).
		For(&infrav1beta2.OpenStackMachine{}).
		WithValidator(&openStackMachineWebhookV1Beta2{}).
		Complete()
}

type openStackMachineWebhookV1Beta2 struct{}

// Compile-time assertion that openStackMachineWebhookV1Beta2 implements webhook.CustomValidator.
var _ webhook.CustomValidator = &openStackMachineWebhookV1Beta2{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type.
func (*openStackMachineWebhookV1Beta2) ValidateCreate(_ context.Context, objRaw runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList
	newObj, err := castToOpenStackMachineV1Beta2(objRaw)
	if err != nil {
		return nil, err
	}

	if newObj.Spec.RootVolume != nil && newObj.Spec.AdditionalBlockDevices != nil {
		for _, device := range newObj.Spec.AdditionalBlockDevices {
			if device.Name == rootVolumeName {
				allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "additionalBlockDevices"), "cannot contain a device named \"root\" when rootVolume is set"))
			}
		}
	}

	for _, port := range newObj.Spec.Ports {
		if ptr.Deref(port.DisablePortSecurity, false) && len(port.SecurityGroups) > 0 {
			allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "ports"), "cannot have security groups when DisablePortSecurity is set to true"))
		}
	}

	return aggregateObjErrors(newObj.GroupVersionKind().GroupKind(), newObj.Name, allErrs)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type.
func (*openStackMachineWebhookV1Beta2) ValidateUpdate(_ context.Context, oldObjRaw, newObjRaw runtime.Object) (admission.Warnings, error) {
	newObj, err := castToOpenStackMachineV1Beta2(newObjRaw)
	if err != nil {
		return nil, err
	}

	newOpenStackMachine, err := runtime.DefaultUnstructuredConverter.ToUnstructured(newObj)
	if err != nil {
		return nil, apierrors.NewInvalid(infrav1beta2.SchemeGroupVersion.WithKind("OpenStackMachine").GroupKind(), newObj.Name, field.ErrorList{
			field.InternalError(nil, fmt.Errorf("failed to convert new OpenStackMachine to unstructured object: %w", err)),
		})
	}
	oldOpenStackMachine, err := runtime.DefaultUnstructuredConverter.ToUnstructured(oldObjRaw)
	if err != nil {
		return nil, apierrors.NewInvalid(infrav1beta2.SchemeGroupVersion.WithKind("OpenStackMachine").GroupKind(), newObj.Name, field.ErrorList{
			field.InternalError(nil, fmt.Errorf("failed to convert old OpenStackMachine to unstructured object: %w", err)),
		})
	}

	var allErrs field.ErrorList

	newOpenStackMachineSpec := newOpenStackMachine["spec"].(map[string]interface{})
	oldOpenStackMachineSpec := oldOpenStackMachine["spec"].(map[string]interface{})

	// allow changes to providerID once
	if oldOpenStackMachineSpec["providerID"] == nil {
		delete(oldOpenStackMachineSpec, "providerID")
		delete(newOpenStackMachineSpec, "providerID")
	}

	// allow changes to instanceID once
	if oldOpenStackMachineSpec["instanceID"] == nil {
		delete(oldOpenStackMachineSpec, "instanceID")
		delete(newOpenStackMachineSpec, "instanceID")
	}

	// allow changes to identifyRef
	delete(oldOpenStackMachineSpec, "identityRef")
	delete(newOpenStackMachineSpec, "identityRef")

	if !reflect.DeepEqual(oldOpenStackMachineSpec, newOpenStackMachineSpec) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec"), "cannot be modified"))
	}

	return aggregateObjErrors(newObj.GroupVersionKind().GroupKind(), newObj.Name, allErrs)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type.
func (*openStackMachineWebhookV1Beta2) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func castToOpenStackMachineV1Beta2(obj runtime.Object) (*infrav1beta2.OpenStackMachine, error) {
	cast, ok := obj.(*infrav1beta2.OpenStackMachine)
	if !ok {
		return nil, fmt.Errorf("expected an OpenStackMachine but got a %T", obj)
	}
	return cast, nil
}
