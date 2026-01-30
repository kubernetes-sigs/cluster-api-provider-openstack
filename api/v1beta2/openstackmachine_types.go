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

package v1beta2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/optional"
)

const (
	// MachineFinalizer allows ReconcileOpenStackMachine to clean up OpenStack resources associated with OpenStackMachine before
	// removing it from the apiserver.
	MachineFinalizer        = "openstackmachine.infrastructure.cluster.x-k8s.io"
	IPClaimMachineFinalizer = "openstackmachine.infrastructure.cluster.x-k8s.io/ip-claim"
)

// SchedulerHintValueType is the type that represents allowed values for the Type field.
// +kubebuilder:validation:Enum=Bool;String;Number
type SchedulerHintValueType string

// Constants representing the allowed types for SchedulerHintAdditionalValue.
const (
	SchedulerHintTypeBool   SchedulerHintValueType = "Bool"
	SchedulerHintTypeString SchedulerHintValueType = "String"
	SchedulerHintTypeNumber SchedulerHintValueType = "Number"
)

// SchedulerHintAdditionalValue represents the value of a scheduler hint property.
// The value can be of various types: Bool, String, or Number.
// The Type field indicates the type of the value being used.
// +kubebuilder:validation:XValidation:rule="has(self.type) && self.type == 'Bool' ? has(self.bool) : !has(self.bool)",message="bool is required when type is Bool, and forbidden otherwise"
// +kubebuilder:validation:XValidation:rule="has(self.type) && self.type == 'Number' ? has(self.number) : !has(self.number)",message="number is required when type is Number, and forbidden otherwise"
// +kubebuilder:validation:XValidation:rule="has(self.type) && self.type == 'String' ? has(self.string) : !has(self.string)",message="string is required when type is String, and forbidden otherwise"
// +union.
type SchedulerHintAdditionalValue struct {
	// Type represents the type of the value.
	// Valid values are Bool, String, and Number.
	// +kubebuilder:validation:Required
	// +unionDiscriminator
	Type SchedulerHintValueType `json:"type"`

	// Bool is the boolean value of the scheduler hint, used when Type is "Bool".
	// This field is required if type is 'Bool', and must not be set otherwise.
	// +unionMember,optional
	Bool *bool `json:"bool,omitempty"`

	// Number is the integer value of the scheduler hint, used when Type is "Number".
	// This field is required if type is 'Number', and must not be set otherwise.
	// +unionMember,optional
	Number *int `json:"number,omitempty"`

	// String is the string value of the scheduler hint, used when Type is "String".
	// This field is required if type is 'String', and must not be set otherwise.
	// +unionMember,optional
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	String *string `json:"string,omitempty"`
}

// SchedulerHintAdditionalProperty represents a single additional property for a scheduler hint.
// It includes a Name to identify the property and a Value that can be of various types.
type SchedulerHintAdditionalProperty struct {
	// Name is the name of the scheduler hint property.
	// It is a unique identifier for the property.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Value is the value of the scheduler hint property, which can be of various types
	// (e.g., bool, string, int). The type is indicated by the Value.Type field.
	// +kubebuilder:validation:Required
	Value SchedulerHintAdditionalValue `json:"value"`
}

// OpenStackMachineSpec defines the desired state of OpenStackMachine.
// +kubebuilder:validation:XValidation:message="at least one of flavor or flavorID must be set",rule=(has(self.flavor) || has(self.flavorID))
type OpenStackMachineSpec struct {
	// ProviderID is the unique identifier as specified by the cloud provider.
	ProviderID *string `json:"providerID,omitempty"`

	// The flavor reference for the flavor for your server instance.
	// +kubebuilder:validation:MinLength=1
	Flavor *string `json:"flavor,omitempty"`

	// FlavorID allows flavors to be specified by ID.  This field takes precedence
	// over Flavor.
	// +kubebuilder:validation:MinLength=1
	FlavorID *string `json:"flavorID,omitempty"`

	// The image to use for your server instance.
	// If the rootVolume is specified, this will be used when creating the root volume.
	// +required
	Image ImageParam `json:"image"`

	// The ssh key to inject in the instance
	SSHKeyName string `json:"sshKeyName,omitempty"`

	// Ports to be attached to the server instance. They are created if a port with the given name does not already exist.
	// If not specified a default port will be added for the default cluster network.
	Ports []PortOpts `json:"ports,omitempty"`

	// The names of the security groups to assign to the instance
	SecurityGroups []SecurityGroupParam `json:"securityGroups,omitempty"`

	// Whether the server instance is created on a trunk port or not.
	Trunk bool `json:"trunk,omitempty"`

	// Tags which will be added to the machine and all dependent resources
	// which support them. These are in addition to Tags defined on the
	// cluster.
	// Requires Nova api 2.52 minimum!
	// +listType=set
	Tags []string `json:"tags,omitempty"`

	// Metadata mapping. Allows you to create a map of key value pairs to add to the server instance.
	// +listType=map
	// +listMapKey=key
	ServerMetadata []ServerMetadata `json:"serverMetadata,omitempty"`

	// Config Drive support
	ConfigDrive *bool `json:"configDrive,omitempty"`

	// The volume metadata to boot from
	RootVolume *RootVolume `json:"rootVolume,omitempty"`

	// AdditionalBlockDevices is a list of specifications for additional block devices to attach to the server instance
	// +listType=map
	// +listMapKey=name
	// +optional
	AdditionalBlockDevices []AdditionalBlockDevice `json:"additionalBlockDevices,omitempty"`

	// The server group to assign the machine to.
	// +optional
	ServerGroup *ServerGroupParam `json:"serverGroup,omitempty"`

	// IdentityRef is a reference to a secret holding OpenStack credentials
	// to be used when reconciling this machine. If not specified, the
	// credentials specified in the cluster will be used.
	// +optional
	IdentityRef *OpenStackIdentityReference `json:"identityRef,omitempty"`

	// floatingIPPoolRef is a reference to a IPPool that will be assigned
	// to an IPAddressClaim. Once the IPAddressClaim is fulfilled, the FloatingIP
	// will be assigned to the OpenStackMachine.
	// +optional
	FloatingIPPoolRef *corev1.TypedLocalObjectReference `json:"floatingIPPoolRef,omitempty"`

	// SchedulerHintAdditionalProperties are arbitrary key/value pairs that provide additional hints
	// to the OpenStack scheduler. These hints can influence how instances are placed on the infrastructure,
	// such as specifying certain host aggregates or availability zones.
	// +optional
	// +listType=map
	// +listMapKey=name
	SchedulerHintAdditionalProperties []SchedulerHintAdditionalProperty `json:"schedulerHintAdditionalProperties,omitempty"`
}

type ServerMetadata struct {
	// Key is the server metadata key
	// +kubebuilder:validation:MaxLength:=255
	// +kubebuilder:validation:Required
	Key string `json:"key"`

	// Value is the server metadata value
	// +kubebuilder:validation:MaxLength:=255
	// +kubebuilder:validation:Required
	Value string `json:"value"`
}

// MachineInitialization contains information about the initialization status of the machine.
type MachineInitialization struct {
	// Provisioned is set to true when the initial provisioning of the machine infrastructure is completed.
	// The value of this field is never updated after provisioning is completed.
	// +optional
	Provisioned bool `json:"provisioned,omitempty"`
}

// OpenStackMachineStatus defines the observed state of OpenStackMachine.
type OpenStackMachineStatus struct {
	// Initialization contains information about the initialization status of the machine.
	// +optional
	Initialization *MachineInitialization `json:"initialization,omitempty"`

	// InstanceID is the OpenStack instance ID for this machine.
	// +optional
	InstanceID optional.String `json:"instanceID,omitempty"`

	// Addresses contains the OpenStack instance associated addresses.
	Addresses []corev1.NodeAddress `json:"addresses,omitempty"`

	// InstanceState is the state of the OpenStack instance for this machine.
	// This field is not set anymore by the OpenStackMachine controller.
	// Instead, it's set by the OpenStackServer controller.
	// +optional
	InstanceState *InstanceState `json:"instanceState,omitempty"`

	// Resolved contains parts of the machine spec with all external
	// references fully resolved.
	// +optional
	Resolved *ResolvedMachineSpec `json:"resolved,omitempty"`

	// Resources contains references to OpenStack resources created for the machine.
	// +optional
	Resources *MachineResources `json:"resources,omitempty"`

	// Conditions defines current service state of the OpenStackMachine.
	// This field surfaces into Machine's status.conditions[InfrastructureReady] condition.
	// The Ready condition must surface issues during the entire lifecycle of the OpenStackMachine
	// (both during initial provisioning and after the initial provisioning is completed).
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=openstackmachines,scope=Namespaced,categories=cluster-api,shortName=osm
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this OpenStackMachine belongs"
// +kubebuilder:printcolumn:name="ProviderID",type="string",JSONPath=".spec.providerID",description="OpenStack instance ID"
// +kubebuilder:printcolumn:name="Machine",type="string",JSONPath=".metadata.ownerReferences[?(@.kind==\"Machine\")].name",description="Machine object which owns with this OpenStackMachine"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of OpenStackMachine"

// OpenStackMachine is the Schema for the openstackmachines API.
type OpenStackMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackMachineSpec   `json:"spec,omitempty"`
	Status OpenStackMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OpenStackMachineList contains a list of OpenStackMachine.
type OpenStackMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackMachine `json:"items"`
}

// GetConditions returns the observations of the operational state of the OpenStackMachine resource.
func (r *OpenStackMachine) GetConditions() []metav1.Condition {
	return r.Status.Conditions
}

// SetConditions sets the underlying service state of the OpenStackMachine to the predescribed clusterv1.Conditions.
func (r *OpenStackMachine) SetConditions(conditions []metav1.Condition) {
	r.Status.Conditions = conditions
}

var _ IdentityRefProvider = &OpenStackMachine{}

// GetIdentifyRef returns the object's namespace and IdentityRef if it has an IdentityRef, or nulls if it does not.
func (r *OpenStackMachine) GetIdentityRef() (*string, *OpenStackIdentityReference) {
	if r.Spec.IdentityRef != nil {
		return &r.Namespace, r.Spec.IdentityRef
	}
	return nil, nil
}

func init() {
	objectTypes = append(objectTypes, &OpenStackMachine{}, &OpenStackMachineList{})
}
