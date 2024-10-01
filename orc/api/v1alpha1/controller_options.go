/*
Copyright 2024 The Kubernetes Authors.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum:=AdoptOrCreate;Adopt
type ControllerOptionsOnCreate string

// +kubebuilder:validation:Enum:=Delete;Retain
type ControllerOptionsOnDelete string

const (
	// ControllerOptionsOnCreateAdoptOrCreate specifies that the controller will
	// attempt to adopt a resource with the expected name if one exists, or
	// create a new resource if it does not.
	ControllerOptionsOnCreateAdoptOrCreate ControllerOptionsOnCreate = "AdoptOrCreate"

	// ControllerOptionsOnCreateAdopt specifies that the controller will wait
	// for the resource to exist, and will not create it.
	ControllerOptionsOnCreateAdopt ControllerOptionsOnCreate = "Adopt"

	// ControllerOptionsOnDeleteDelete specifies that the controller will delete
	// the resource when the kubernetes object owning it is deleted.
	ControllerOptionsOnDeleteDelete ControllerOptionsOnDelete = "Delete"

	// ControllerOptionsOnDeleteRetain specifies that the controller will not
	// delete the resource when the kubernetes object owning it is deleted.
	ControllerOptionsOnDeleteRetain ControllerOptionsOnDelete = "Retain"
)

type ControllerOptions struct {
	// OnCreate defines the controller's behaviour when creating a resource.
	// If not specified, the default is AdoptOrCreate.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="onCreate is immutable"
	// +optional
	OnCreate *ControllerOptionsOnCreate `json:"onCreate,omitempty"`

	// OnDelete defines the controller's behaviour when deleting a resource. If
	// not specified, the default is Delete.
	// +optional
	OnDelete *ControllerOptionsOnDelete `json:"onDelete,omitempty"`
}

// ObjectWithControllerOptions is a metav1.Object which also has ControllerOptions
// +kubebuilder:object:generate:=false
type ObjectWithControllerOptions interface {
	metav1.Object
	GetControllerOptions() *ControllerOptions
}

// GetOnCreate returns the value of OnCreate from ControllerOptions, or
// the default, AdoptOrCreate, if it is not set.
func (o *ControllerOptions) GetOnCreate() ControllerOptionsOnCreate {
	const defaultOnCreate = ControllerOptionsOnCreateAdoptOrCreate

	if o == nil || o.OnCreate == nil {
		return defaultOnCreate
	}

	return *o.OnCreate
}

// GetOnDelete returns the value of OnDelete from ControllerOptions, or
// the default, Delete, if it is not set.
func (o *ControllerOptions) GetOnDelete() ControllerOptionsOnDelete {
	const defaultOnDelete = ControllerOptionsOnDeleteDelete

	if o == nil || o.OnDelete == nil {
		return defaultOnDelete
	}

	return *o.OnDelete
}
