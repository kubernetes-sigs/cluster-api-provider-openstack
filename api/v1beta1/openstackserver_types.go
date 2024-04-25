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

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/optional"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/errors"
)

// OpenStackServerSpec defines the desired state of OpenStackServer.
type OpenStackServerSpec struct {
	// AdditionalBlockDevices is a list of specifications for additional block devices to attach to the server instance.
	// +listType=map
	// +listMapKey=name
	// +optional
	AdditionalBlockDevices []AdditionalBlockDevice `json:"additionalBlockDevices,omitempty"`

	// AvailabilityZone is the availability zone in which to create the server instance.
	//+optional
	AvailabilityZone optional.String `json:"availabilityZone,omitempty"`

	// ConfigDrive is a flag to enable config drive for the server instance.
	// +optional
	ConfigDrive optional.Bool `json:"configDrive,omitempty"`

	// The flavor reference for the flavor for the server instance.
	// +required
	Flavor string `json:"flavor"`

	// FloatingIPPoolRef is a reference to a FloatingIPPool to allocate a floating IP from.
	// +optional
	FloatingIPPoolRef *corev1.TypedLocalObjectReference `json:"floatingIPPoolRef,omitempty"`

	// The identity reference for the server instance.
	// +optional
	IdentityRef *OpenStackIdentityReference `json:"identityRef,omitempty"`

	// The image to use for the server instance.
	// +required
	Image ImageParam `json:"image"`

	// Ports to be attached to the server instance.
	// +optional
	Ports []PortOpts `json:"ports,omitempty"`

	// RootVolume is the specification for the root volume of the server instance.
	// +optional
	RootVolume *RootVolume `json:"rootVolume,omitempty"`

	// SSHKeyName is the name of the SSH key to inject in the instance.
	// +required
	SSHKeyName string `json:"sshKeyName"`

	// SecurityGroups is a list of security groups names to assign to the instance.
	// +optional
	SecurityGroups []SecurityGroupParam `json:"securityGroups,omitempty"`

	// ServerGroup is the server group to which the server instance belongs.
	// +optional
	ServerGroup *ServerGroupParam `json:"serverGroup,omitempty"`

	// ServerMetadata is a map of key value pairs to add to the server instance.
	// +listType=map
	// +listMapKey=key
	// +optional
	ServerMetadata []ServerMetadata `json:"serverMetadata,omitempty"`

	// Tags which will be added to the machine and all dependent resources
	// which support them. These are in addition to Tags defined on the
	// cluster.
	// Requires Nova api 2.52 minimum!
	// +listType=set
	Tags []string `json:"tags,omitempty"`

	// Trunk is a flag to indicate if the server instance is created on a trunk port or not.
	// +optional
	Trunk optional.Bool `json:"trunk,omitempty"`

	// UserDataRef is a reference to a UserData to be injected into the server instance.
	// +optional
	UserDataRef *corev1.TypedLocalObjectReference `json:"userDataRef,omitempty"`
}

// OpenStackServerStatus defines the observed state of OpenStackServer.
type OpenStackServerStatus struct {
	// Ready is true when the OpenStack server is ready.
	// +kubebuilder:default=false
	Ready bool `json:"ready"`

	// InstanceID is the ID of the server instance.
	// +optional
	InstanceID optional.String `json:"instanceID,omitempty"`

	// InstanceState is the state of the server instance.
	// +optional
	InstanceState *InstanceState `json:"instanceState,omitempty"`

	// Addresses is the list of addresses of the server instance.
	// +optional
	Addresses []corev1.NodeAddress `json:"addresses,omitempty"`

	// Resolved contains parts of the machine spec with all external
	// references fully resolved.
	// +optional
	Resolved *ResolvedServerSpec `json:"resolved,omitempty"`

	// Resources contains references to OpenStack resources created for the machine.
	// +optional
	Resources *ServerResources `json:"resources,omitempty"`

	// FailureReason will be set in the event that there is a terminal problem
	// reconciling the OpenStackServer and will be the name the state that the
	// server is in.
	// +optional
	FailureReason *errors.ServerStatusError `json:"failureReason,omitempty"`

	// FailureMessage will be set in the event that there is a terminal problem
	// reconciling the Server and will contain a more verbose string suitable
	// for logging and human consumption.
	//
	// This field should not be set for transitive errors that a controller
	// faces that are expected to be fixed automatically over
	// time (like service outages), but instead indicate that something is
	// fundamentally wrong with the OpenStackServer's spec or the configuration of
	// the controller, and that manual intervention is required. Examples
	// of terminal errors would be invalid combinations of settings in the
	// spec, values that are unsupported by the controller, or the
	// responsible controller itself being critically misconfigured.
	//
	// Any transient errors that occur during the reconciliation of OpenStackServers
	// can be added as events to the OpenStackServer object and/or logged in the
	// controller's output.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`

	// Conditions defines current service state of the OpenStackServer.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

// +genclient
// +genclient:Namespaced
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:path=openstackservers,scope=Namespaced,categories=cluster-api,shortName=osm
// +kubebuilder:subresource:status

// OpenStackServer is the Schema for the openstackservers API.
type OpenStackServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackServerSpec   `json:"spec,omitempty"`
	Status OpenStackServerStatus `json:"status,omitempty"`
}
