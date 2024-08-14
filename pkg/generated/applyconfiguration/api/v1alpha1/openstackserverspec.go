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

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	v1beta1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/generated/applyconfiguration/api/v1beta1"
)

// OpenStackServerSpecApplyConfiguration represents an declarative configuration of the OpenStackServerSpec type for use
// with apply.
type OpenStackServerSpecApplyConfiguration struct {
	AdditionalBlockDevices []v1beta1.AdditionalBlockDeviceApplyConfiguration     `json:"additionalBlockDevices,omitempty"`
	AvailabilityZone       *string                                               `json:"availabilityZone,omitempty"`
	ConfigDrive            *bool                                                 `json:"configDrive,omitempty"`
	Flavor                 *string                                               `json:"flavor,omitempty"`
	FloatingIPPoolRef      *v1.TypedLocalObjectReference                         `json:"floatingIPPoolRef,omitempty"`
	IdentityRef            *v1beta1.OpenStackIdentityReferenceApplyConfiguration `json:"identityRef,omitempty"`
	Image                  *v1beta1.ImageParamApplyConfiguration                 `json:"image,omitempty"`
	Ports                  []v1beta1.PortOptsApplyConfiguration                  `json:"ports,omitempty"`
	RootVolume             *v1beta1.RootVolumeApplyConfiguration                 `json:"rootVolume,omitempty"`
	SSHKeyName             *string                                               `json:"sshKeyName,omitempty"`
	SecurityGroups         []v1beta1.SecurityGroupParamApplyConfiguration        `json:"securityGroups,omitempty"`
	ServerGroup            *v1beta1.ServerGroupParamApplyConfiguration           `json:"serverGroup,omitempty"`
	ServerMetadata         []v1beta1.ServerMetadataApplyConfiguration            `json:"serverMetadata,omitempty"`
	Tags                   []string                                              `json:"tags,omitempty"`
	Trunk                  *bool                                                 `json:"trunk,omitempty"`
	UserDataRef            *v1.LocalObjectReference                              `json:"userDataRef,omitempty"`
	Resolved               *ResolvedServerSpecApplyConfiguration                 `json:"resolved,omitempty"`
	Resources              *ServerResourcesApplyConfiguration                    `json:"resources,omitempty"`
}

// OpenStackServerSpecApplyConfiguration constructs an declarative configuration of the OpenStackServerSpec type for use with
// apply.
func OpenStackServerSpec() *OpenStackServerSpecApplyConfiguration {
	return &OpenStackServerSpecApplyConfiguration{}
}

// WithAdditionalBlockDevices adds the given value to the AdditionalBlockDevices field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the AdditionalBlockDevices field.
func (b *OpenStackServerSpecApplyConfiguration) WithAdditionalBlockDevices(values ...*v1beta1.AdditionalBlockDeviceApplyConfiguration) *OpenStackServerSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithAdditionalBlockDevices")
		}
		b.AdditionalBlockDevices = append(b.AdditionalBlockDevices, *values[i])
	}
	return b
}

// WithAvailabilityZone sets the AvailabilityZone field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the AvailabilityZone field is set to the value of the last call.
func (b *OpenStackServerSpecApplyConfiguration) WithAvailabilityZone(value string) *OpenStackServerSpecApplyConfiguration {
	b.AvailabilityZone = &value
	return b
}

// WithConfigDrive sets the ConfigDrive field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ConfigDrive field is set to the value of the last call.
func (b *OpenStackServerSpecApplyConfiguration) WithConfigDrive(value bool) *OpenStackServerSpecApplyConfiguration {
	b.ConfigDrive = &value
	return b
}

// WithFlavor sets the Flavor field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Flavor field is set to the value of the last call.
func (b *OpenStackServerSpecApplyConfiguration) WithFlavor(value string) *OpenStackServerSpecApplyConfiguration {
	b.Flavor = &value
	return b
}

// WithFloatingIPPoolRef sets the FloatingIPPoolRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the FloatingIPPoolRef field is set to the value of the last call.
func (b *OpenStackServerSpecApplyConfiguration) WithFloatingIPPoolRef(value v1.TypedLocalObjectReference) *OpenStackServerSpecApplyConfiguration {
	b.FloatingIPPoolRef = &value
	return b
}

// WithIdentityRef sets the IdentityRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the IdentityRef field is set to the value of the last call.
func (b *OpenStackServerSpecApplyConfiguration) WithIdentityRef(value *v1beta1.OpenStackIdentityReferenceApplyConfiguration) *OpenStackServerSpecApplyConfiguration {
	b.IdentityRef = value
	return b
}

// WithImage sets the Image field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Image field is set to the value of the last call.
func (b *OpenStackServerSpecApplyConfiguration) WithImage(value *v1beta1.ImageParamApplyConfiguration) *OpenStackServerSpecApplyConfiguration {
	b.Image = value
	return b
}

// WithPorts adds the given value to the Ports field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Ports field.
func (b *OpenStackServerSpecApplyConfiguration) WithPorts(values ...*v1beta1.PortOptsApplyConfiguration) *OpenStackServerSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithPorts")
		}
		b.Ports = append(b.Ports, *values[i])
	}
	return b
}

// WithRootVolume sets the RootVolume field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the RootVolume field is set to the value of the last call.
func (b *OpenStackServerSpecApplyConfiguration) WithRootVolume(value *v1beta1.RootVolumeApplyConfiguration) *OpenStackServerSpecApplyConfiguration {
	b.RootVolume = value
	return b
}

// WithSSHKeyName sets the SSHKeyName field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the SSHKeyName field is set to the value of the last call.
func (b *OpenStackServerSpecApplyConfiguration) WithSSHKeyName(value string) *OpenStackServerSpecApplyConfiguration {
	b.SSHKeyName = &value
	return b
}

// WithSecurityGroups adds the given value to the SecurityGroups field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the SecurityGroups field.
func (b *OpenStackServerSpecApplyConfiguration) WithSecurityGroups(values ...*v1beta1.SecurityGroupParamApplyConfiguration) *OpenStackServerSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithSecurityGroups")
		}
		b.SecurityGroups = append(b.SecurityGroups, *values[i])
	}
	return b
}

// WithServerGroup sets the ServerGroup field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ServerGroup field is set to the value of the last call.
func (b *OpenStackServerSpecApplyConfiguration) WithServerGroup(value *v1beta1.ServerGroupParamApplyConfiguration) *OpenStackServerSpecApplyConfiguration {
	b.ServerGroup = value
	return b
}

// WithServerMetadata adds the given value to the ServerMetadata field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the ServerMetadata field.
func (b *OpenStackServerSpecApplyConfiguration) WithServerMetadata(values ...*v1beta1.ServerMetadataApplyConfiguration) *OpenStackServerSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithServerMetadata")
		}
		b.ServerMetadata = append(b.ServerMetadata, *values[i])
	}
	return b
}

// WithTags adds the given value to the Tags field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Tags field.
func (b *OpenStackServerSpecApplyConfiguration) WithTags(values ...string) *OpenStackServerSpecApplyConfiguration {
	for i := range values {
		b.Tags = append(b.Tags, values[i])
	}
	return b
}

// WithTrunk sets the Trunk field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Trunk field is set to the value of the last call.
func (b *OpenStackServerSpecApplyConfiguration) WithTrunk(value bool) *OpenStackServerSpecApplyConfiguration {
	b.Trunk = &value
	return b
}

// WithUserDataRef sets the UserDataRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the UserDataRef field is set to the value of the last call.
func (b *OpenStackServerSpecApplyConfiguration) WithUserDataRef(value v1.LocalObjectReference) *OpenStackServerSpecApplyConfiguration {
	b.UserDataRef = &value
	return b
}

// WithResolved sets the Resolved field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Resolved field is set to the value of the last call.
func (b *OpenStackServerSpecApplyConfiguration) WithResolved(value *ResolvedServerSpecApplyConfiguration) *OpenStackServerSpecApplyConfiguration {
	b.Resolved = value
	return b
}

// WithResources sets the Resources field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Resources field is set to the value of the last call.
func (b *OpenStackServerSpecApplyConfiguration) WithResources(value *ServerResourcesApplyConfiguration) *OpenStackServerSpecApplyConfiguration {
	b.Resources = value
	return b
}
