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
	v1beta1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	apiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// OpenStackServerStatusApplyConfiguration represents an declarative configuration of the OpenStackServerStatus type for use
// with apply.
type OpenStackServerStatusApplyConfiguration struct {
	Ready         *bool                  `json:"ready,omitempty"`
	InstanceID    *string                `json:"instanceID,omitempty"`
	InstanceState *v1beta1.InstanceState `json:"instanceState,omitempty"`
	Addresses     []v1.NodeAddress       `json:"addresses,omitempty"`
	Conditions    *apiv1beta1.Conditions `json:"conditions,omitempty"`
}

// OpenStackServerStatusApplyConfiguration constructs an declarative configuration of the OpenStackServerStatus type for use with
// apply.
func OpenStackServerStatus() *OpenStackServerStatusApplyConfiguration {
	return &OpenStackServerStatusApplyConfiguration{}
}

// WithReady sets the Ready field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Ready field is set to the value of the last call.
func (b *OpenStackServerStatusApplyConfiguration) WithReady(value bool) *OpenStackServerStatusApplyConfiguration {
	b.Ready = &value
	return b
}

// WithInstanceID sets the InstanceID field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the InstanceID field is set to the value of the last call.
func (b *OpenStackServerStatusApplyConfiguration) WithInstanceID(value string) *OpenStackServerStatusApplyConfiguration {
	b.InstanceID = &value
	return b
}

// WithInstanceState sets the InstanceState field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the InstanceState field is set to the value of the last call.
func (b *OpenStackServerStatusApplyConfiguration) WithInstanceState(value v1beta1.InstanceState) *OpenStackServerStatusApplyConfiguration {
	b.InstanceState = &value
	return b
}

// WithAddresses adds the given value to the Addresses field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Addresses field.
func (b *OpenStackServerStatusApplyConfiguration) WithAddresses(values ...v1.NodeAddress) *OpenStackServerStatusApplyConfiguration {
	for i := range values {
		b.Addresses = append(b.Addresses, values[i])
	}
	return b
}

// WithConditions sets the Conditions field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Conditions field is set to the value of the last call.
func (b *OpenStackServerStatusApplyConfiguration) WithConditions(value apiv1beta1.Conditions) *OpenStackServerStatusApplyConfiguration {
	b.Conditions = &value
	return b
}
