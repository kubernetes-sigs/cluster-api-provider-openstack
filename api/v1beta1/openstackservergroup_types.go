/*
Copyright 2019 The Kubernetes Authors.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ServerGroupFinalizer allows ReconcileOpenStackServerGroup to clean up OpenStack resources associated with OpenStackServerGroup before
	// removing it from the apiserver.
	ServerGroupFinalizer = "openstackservergroup.infrastructure.cluster.x-k8s.io"
)

// OpenStackServerGroupSpec defines the desired state of OpenStackServerGroup.
type OpenStackServerGroupSpec struct {
	// Policy is a string with some valid values; affinity, anti-affinity, soft-affinity, soft-anti-affinity.
	Policy string `json:"policy"`

	// IdentityRef is a reference to a identity to be used when reconciling this resource
	// +optional
	IdentityRef *OpenStackIdentityReference `json:"identityRef,omitempty"`
}

// OpenStackServerGroupStatus defines the observed state of OpenStackServerGroup.
type OpenStackServerGroupStatus struct {
	// Ready is true when the resource is created.
	// +optional
	Ready bool `json:"ready"`

	// UUID of provisioned ServerGroup
	ID string `json:"uuid"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OpenStackServerGroup is the Schema for the openstackservergroups API.
type OpenStackServerGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackServerGroupSpec   `json:"spec,omitempty"`
	Status OpenStackServerGroupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenStackServerGroupList contains a list of OpenStackServerGroup.
type OpenStackServerGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackServerGroup `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &OpenStackServerGroup{}, &OpenStackServerGroupList{})
}
