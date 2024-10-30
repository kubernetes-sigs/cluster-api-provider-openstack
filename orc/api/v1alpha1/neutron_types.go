/*
Copyright 2024 The ORC Authors.

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

// NeutronTag represents a tag on a Neutron resource.
// It may not be empty and may not contain commas.
// +kubebuilder:validation:Pattern:="^[^,]+$"
// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=512
type NeutronTag string

type FilterByNeutronTags struct {
	// Tags is a list of tags to filter by. If specified, the resource must
	// have all of the tags specified to be included in the result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=32
	Tags []NeutronTag `json:"tags,omitempty"`

	// TagsAny is a list of tags to filter by. If specified, the resource
	// must have at least one of the tags specified to be included in the
	// result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=32
	TagsAny []NeutronTag `json:"tagsAny,omitempty"`

	// NotTags is a list of tags to filter by. If specified, resources which
	// contain all of the given tags will be excluded from the result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=32
	NotTags []NeutronTag `json:"notTags,omitempty"`

	// NotTagsAny is a list of tags to filter by. If specified, resources
	// which contain any of the given tags will be excluded from the result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=32
	NotTagsAny []NeutronTag `json:"notTagsAny,omitempty"`
}

// +kubebuilder:validation:Enum:=4;6
type IPVersion int8

// +kubebuilder:validation:Format:=cidr
// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=49
type CIDR string

// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=45
type IPvAny string
