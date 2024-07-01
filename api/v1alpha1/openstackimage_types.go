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

// See https://docs.openstack.org/glance/latest/admin/useful-image-properties.html
// for a list of 'well known' image properties we might consider supporting explicitly.

// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=255
type ImageTag string

// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=10
type ImageContainerFormat string

// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=10
type ImageDiskFormat string

// kubebuilder:validation:Enum:=public;private;shared;community
type ImageVisibility string

// kubebuilder:validation:Enum:=md5;sha1;sha256;sha512
type ImageHashAlgorithm string

const (
	ImageHashAlgorithmMD5    ImageHashAlgorithm = "md5"
	ImageHashAlgorithmSHA1   ImageHashAlgorithm = "sha1"
	ImageHashAlgorithmSHA256 ImageHashAlgorithm = "sha256"
	ImageHashAlgorithmSHA512 ImageHashAlgorithm = "sha512"
)

// +kubebuilder:validation:XValidation:rule="!in(self, ['created_at', 'updated_at', 'status', 'checksum', 'size', 'virtual_size', 'direct_url', 'self', 'file', 'schema', 'id', 'os_hash_algo', 'os_hash_value', 'location', 'deleted', 'deleted_at', 'container_format', 'disk_format', 'min_disk', 'min_ram', 'name', 'tags', 'owner', 'visibility', 'protected', 'os_hidden'])"
type ImageAdditionalPropertyName string

type ImageAdditionalProperty struct {
	// Name is the name of the glance property. It is an error for it to conflict with any explicitly defined property.
	Name ImageAdditionalPropertyName `json:"name"`

	// Value is the value of the glance property
	// +required
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=1024
	Value string `json:"value"`
}

type ImageProperties struct {
	// MinDisk is the minimum amount of disk space in GB that is required to boot the image
	// +kubebuilder:validation:Minimum:=1
	MinDiskGB *int `json:"minDiskGB,omitempty"`

	// MinRAMMB is the minimum amount of RAM in MB that is required to boot the image.
	// +kubebuilder:validation:Minimum:=1
	MinRAMMB *int `json:"minRAMMB,omitempty"`
}

// +kubebuilder:validation:Enum:=xz;gz
type ImageCompression string

const (
	ImageCompressionXZ ImageCompression = "xz"
	ImageCompressionGZ ImageCompression = "gz"
)

type ImageContent struct {
	// Hash is a hash which can be used to verify the downloaded image data
	Hash *ImageHash `json:"hash,omitempty"`

	// GlanceHashAlgorithm is the algorithm of the hash glance will publish.
	// If not set it will default to sha512, which is Glance's default. It
	// MUST be set to the value configured in Glance.
	// +kubebuilder:default:=sha512
	// +required
	GlanceHashAlgorithm *ImageHashAlgorithm `json:"glanceHashAlgorithm,omitempty"`

	// Source specifies how to obtain the image data
	Source ImageSource `json:"source"`
}

type ImageSourceType string

const (
	ImageSourceTypeURL ImageSourceType = "url"
)

// ImageSource specifies the source of image data
// +kubebuilder:validation:XValidation:rule="has(self.type) && self.type == 'url' ?  has(self.url) : !has(self.url)",message="url is required when type is url, and forbidden otherwise"
// +union
type ImageSource struct {
	// Type is the type of the image source
	// +kubebuilder:validation:Required
	// +unionDiscriminator
	Type ImageSourceType `json:"type"`

	// URL describes how to obtain image data by downloading it from a URL. Must be set if Type is 'url'
	// +unionMember
	URL *ImageSourceURL `json:"url,omitempty"`
}

type ImageSourceURL struct {
	// URL containing image data
	// +kubebuilder:validation:Format=uri
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// Decompress specifies that the source data must be decompressed with the given compression algorithm before being stored
	// +optional
	Decompress *ImageCompression `json:"decompress,omitempty"`
}

type ImageHash struct {
	// Algorithm is the hash algorithm used to generate value.
	// +required
	Algorithm ImageHashAlgorithm `json:"algorithm"`

	// Value is the hash of the image data using Algorithm.
	// kubebuilder:validation:MinLength:=1
	// +required
	Value string `json:"value"`
}

// OpenStackImageSpec defines the desired state of OpenStackImage.
type OpenStackImageSpec struct {
	// ContainerFormat is the format of the image container.
	// qcow2 and raw images do not usually have a container, and this can be omitted.
	// +optional
	ContainerFormat *ImageContainerFormat `json:"containerFormat,omitempty"`

	// DiskFormat is the format of the disk image.
	// Normal values are "qcow2", or "raw". Glance may be configured to support others.
	// +optional
	DiskFormat *ImageDiskFormat `json:"diskFormat,omitempty"`

	// Protected specifies that the image is protected from deletion.
	// If not specified, the default is false.
	// +optional
	Protected *bool `json:"protected,omitempty"`

	// Tags is a list of tags which will be applied to the image. A tag has a maximum length of 255 characters.
	// +optional
	Tags []ImageTag `json:"tags,omitempty"`

	// Visibility of the image
	// +optional
	Visibility *ImageVisibility `json:"visibility,omitempty"`

	// AdditionalProperties allows arbitrary glance properties to be set. These will be merged
	// +listType=map
	// +listMapKey=name
	// +optional
	AdditionalProperties []ImageAdditionalProperty `json:"additionalProperties,omitempty"`
}

// OpenStackImageStatus defines the observed state of OpenStackImage.
type OpenStackImageStatus struct {
	// ID is the UUID of the glance image
	ID *string `json:"id,omitempty"`

	// Hash is the hash of the image data calculated by glance
	Hash *ImageHash `json:"hash,omitempty"`

	// SizeB is the size of the image data, in bytes
	SizeB *int64 `json:"sizeB,omitempty"`

	// VirtualSizeB is the size of the disk the image data represents, in bytes
	VirtualSizeB *int64 `json:"virtualSizeB,omitempty"`

	// Conditions represents the
	// Known .status.conditions.type are: "Available", "Progressing", and "Status"
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OpenStackImage is the Schema for the openstackimages API.
type OpenStackImage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackImageSpec   `json:"spec,omitempty"`
	Status OpenStackImageStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenStackImageList contains a list of OpenStackImage.
type OpenStackImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackImage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackImage{}, &OpenStackImageList{})
}
