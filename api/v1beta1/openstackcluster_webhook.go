/*
Copyright 2021 The Kubernetes Authors.

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
	"reflect"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
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
	if r.Spec.IdentityRef != nil && r.Spec.IdentityRef.Kind == "" {
		r.Spec.IdentityRef.Kind = defaultIdentityRefKind
	}
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *OpenStackCluster) ValidateCreate() error {
	var allErrs field.ErrorList

	if r.Spec.IdentityRef != nil && r.Spec.IdentityRef.Kind != defaultIdentityRefKind {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "identityRef", "kind"), "must be a Secret"))
	}

	return aggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, allErrs)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *OpenStackCluster) ValidateUpdate(old runtime.Object) error {
	var allErrs field.ErrorList

	newOpenStackCluster, err := runtime.DefaultUnstructuredConverter.ToUnstructured(r)
	if err != nil {
		return apierrors.NewInvalid(GroupVersion.WithKind("OpenStackCluster").GroupKind(), r.Name, field.ErrorList{
			field.InternalError(nil, errors.Wrap(err, "failed to convert new OpenStackCluster to unstructured object")),
		})
	}
	oldOpenStackCluster, err := runtime.DefaultUnstructuredConverter.ToUnstructured(old)
	if err != nil {
		return apierrors.NewInvalid(GroupVersion.WithKind("OpenStackCluster").GroupKind(), r.Name, field.ErrorList{
			field.InternalError(nil, errors.Wrap(err, "failed to convert old OpenStackCluster to unstructured object")),
		})
	}

	if r.Spec.IdentityRef != nil && r.Spec.IdentityRef.Kind != defaultIdentityRefKind {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "identityRef", "kind"), "must be a Secret"))
	}

	newOpenStackClusterSpec := newOpenStackCluster["spec"].(map[string]interface{})
	oldOpenStackClusterSpec := oldOpenStackCluster["spec"].(map[string]interface{})

	// get controlPlaneEndpoint, something like {"host":"", "port":""}
	cpe := oldOpenStackClusterSpec["controlPlaneEndpoint"].(map[string]interface{})

	// allow change only for the first time
	host, ok := cpe["host"].(string)
	if ok && len(host) == 0 {
		delete(oldOpenStackClusterSpec, "controlPlaneEndpoint")
		delete(newOpenStackClusterSpec, "controlPlaneEndpoint")
	}

	if !reflect.DeepEqual(oldOpenStackClusterSpec, newOpenStackClusterSpec) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec"), "cannot be modified"))
	}

	return aggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *OpenStackCluster) ValidateDelete() error {
	return nil
}
