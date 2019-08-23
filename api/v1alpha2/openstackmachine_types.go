/*

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

package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/errors"
)

const (
	// MachineFinalizer allows ReconcileOpenStackMachine to clean up OpenStack resources associated with OpenStackMachine before
	// removing it from the apiserver.
	MachineFinalizer = "openstackmachine.infrastructure.cluster.x-k8s.io"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OpenStackMachineSpec defines the desired state of OpenStackMachine
type OpenStackMachineSpec struct {

	// ProviderID is the unique identifier as specified by the cloud provider.
	ProviderID *string `json:"providerID,omitempty"`

	// The name of the secret containing the openstack credentials
	// +optional
	CloudsSecret *corev1.SecretReference `json:"cloudsSecret"`

	// The name of the cloud to use from the clouds secret
	// +optional
	CloudName string `json:"cloudName"`

	// The flavor reference for the flavor for your server instance.
	Flavor string `json:"flavor"`

	// The name of the image to use for your server instance.
	// If the RootVolume is specified, this will be ignored and use rootVolume directly.
	Image string `json:"image"`

	// The ssh key to inject in the instance
	KeyName string `json:"keyName,omitempty"`

	// A networks object. Required parameter when there are multiple networks defined for the tenant.
	// When you do not specify the networks parameter, the server attaches to the only network created for the current tenant.
	Networks []NetworkParam `json:"networks,omitempty"`
	// The floatingIP which will be associated to the machine, only used for master.
	// The floatingIP should have been created and haven't been associated.
	FloatingIP string `json:"floatingIP,omitempty"`

	// The availability zone from which to launch the server.
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// The names of the security groups to assign to the instance
	SecurityGroups []SecurityGroupParam `json:"securityGroups,omitempty"`

	// The name of the secret containing the user data (startup script in most cases)
	UserDataSecret *corev1.SecretReference `json:"userDataSecret,omitempty"`

	// Whether the server instance is created on a trunk port or not.
	Trunk bool `json:"trunk,omitempty"`

	// Machine tags
	// Requires Nova api 2.52 minimum!
	Tags []string `json:"tags,omitempty"`

	// Metadata mapping. Allows you to create a map of key value pairs to add to the server instance.
	ServerMetadata map[string]string `json:"serverMetadata,omitempty"`

	// Config Drive support
	ConfigDrive *bool `json:"configDrive,omitempty"`

	// The volume metadata to boot from
	RootVolume *RootVolume `json:"rootVolume,omitempty"`

	// KubeadmConfiguration holds the kubeadm configuration options
	// +optional
	KubeadmConfiguration KubeadmConfiguration `json:"kubeadmConfiguration,omitempty"`
}

// OpenStackMachineStatus defines the observed state of OpenStackMachine
type OpenStackMachineStatus struct {

	// Ready is true when the provider resource is ready.
	// +optional
	Ready bool `json:"ready"`

	// Addresses contains the OpenStack instance associated addresses.
	Addresses []corev1.NodeAddress `json:"addresses,omitempty"`

	// InstanceState is the state of the OpenStack instance for this machine.
	// +optional
	InstanceState *InstanceState `json:"instanceState,omitempty"`

	ErrorReason *errors.MachineStatusError `json:"errorReason,omitempty"`

	// ErrorMessage will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a more verbose string suitable
	// for logging and human consumption.
	//
	// This field should not be set for transitive errors that a controller
	// faces that are expected to be fixed automatically over
	// time (like service outages), but instead indicate that something is
	// fundamentally wrong with the Machine's spec or the configuration of
	// the controller, and that manual intervention is required. Examples
	// of terminal errors would be invalid combinations of settings in the
	// spec, values that are unsupported by the controller, or the
	// responsible controller itself being critically misconfigured.
	//
	// Any transient errors that occur during the reconciliation of Machines
	// can be added as events to the Machine object and/or logged in the
	// controller's output.
	// +optional
	ErrorMessage *string `json:"errorMessage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=openstackmachines,scope=Namespaced
// +kubebuilder:storageversion
// +kubebuilder:subresource:status

// OpenStackMachine is the Schema for the openstackmachines API
type OpenStackMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackMachineSpec   `json:"spec,omitempty"`
	Status OpenStackMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OpenStackMachineList contains a list of OpenStackMachine
type OpenStackMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackMachine{}, &OpenStackMachineList{})
}
