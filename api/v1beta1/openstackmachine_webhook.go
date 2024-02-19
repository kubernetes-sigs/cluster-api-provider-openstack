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
	"sigs.k8s.io/controller-runtime/pkg/builder"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var _ = logf.Log.WithName("openstackmachine-resource")

func (r *OpenStackMachine) SetupWebhookWithManager(mgr manager.Manager) error {
	return builder.WebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-openstackmachine,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=openstackmachines,versions=v1beta1,name=validation.openstackmachine.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1
// +kubebuilder:webhook:verbs=create;update,path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-openstackmachine,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=openstackmachines,versions=v1beta1,name=default.openstackmachine.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1

var (
	_ webhook.Defaulter = &OpenStackMachine{}
	_ webhook.Validator = &OpenStackMachine{}
)

// Default satisfies the defaulting webhook interface.
func (r *OpenStackMachine) Default() {
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *OpenStackMachine) ValidateCreate() (admission.Warnings, error) {
	var allErrs field.ErrorList

	if r.Spec.RootVolume != nil && r.Spec.AdditionalBlockDevices != nil {
		for _, device := range r.Spec.AdditionalBlockDevices {
			if device.Name == "root" {
				allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "additionalBlockDevices"), "cannot contain a device named \"root\" when rootVolume is set"))
			}
		}
	}

	return aggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, allErrs)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *OpenStackMachine) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	newOpenStackMachine, err := runtime.DefaultUnstructuredConverter.ToUnstructured(r)
	if err != nil {
		return nil, apierrors.NewInvalid(GroupVersion.WithKind("OpenStackMachine").GroupKind(), r.Name, field.ErrorList{
			field.InternalError(nil, fmt.Errorf("failed to convert new OpenStackMachine to unstructured object: %w", err)),
		})
	}
	oldOpenStackMachine, err := runtime.DefaultUnstructuredConverter.ToUnstructured(old)
	if err != nil {
		return nil, apierrors.NewInvalid(GroupVersion.WithKind("OpenStackMachine").GroupKind(), r.Name, field.ErrorList{
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

	return aggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *OpenStackMachine) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}
