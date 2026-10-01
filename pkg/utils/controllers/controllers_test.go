/*
Copyright 2023 The Kubernetes Authors.

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

package controllers

import (
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1alpha1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
)

func Test_validateSubnets(t *testing.T) {
	tests := []struct {
		name    string
		subnets []infrav1.Subnet
		wantErr bool
	}{
		{
			name: "valid IPv4 and IPv6 subnets",
			subnets: []infrav1.Subnet{
				{
					CIDR: "192.168.0.0/24",
				},
				{
					CIDR: "2001:db8:2222:5555::/64",
				},
			},
			wantErr: false,
		},
		{
			name: "valid IPv4 and IPv6 subnets",
			subnets: []infrav1.Subnet{
				{
					CIDR: "2001:db8:2222:5555::/64",
				},
				{
					CIDR: "192.168.0.0/24",
				},
			},
			wantErr: false,
		},
		{
			name: "multiple IPv4 subnets",
			subnets: []infrav1.Subnet{
				{
					CIDR: "192.168.0.0/24",
				},
				{
					CIDR: "10.0.0.0/24",
				},
			},
			wantErr: false,
		},
		{
			name: "multiple IPv6 subnets",
			subnets: []infrav1.Subnet{
				{
					CIDR: "2001:db8:2222:5555::/64",
				},
				{
					CIDR: "2001:db8:2222:6666::/64",
				},
			},
			wantErr: false,
		},
		{
			name: "three subnets mixed IP versions",
			subnets: []infrav1.Subnet{
				{
					CIDR: "192.168.0.0/24",
				},
				{
					CIDR: "10.0.0.0/24",
				},
				{
					CIDR: "2001:db8:2222:5555::/64",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid IP address",
			subnets: []infrav1.Subnet{
				{
					CIDR: "192.168.0.0/24",
				},
				{
					CIDR: "invalid",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSubnets(tt.subnets)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSubnets() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnsureConditionReasons(t *testing.T) {
	now := metav1.Now()

	tests := []struct {
		name        string
		conditions  []metav1.Condition
		want        []metav1.Condition
		wantChanged bool
	}{
		{
			name: "no conditions",
		},
		{
			name: "all reasons set is a no-op",
			conditions: []metav1.Condition{
				{Type: "InstanceReady", Status: metav1.ConditionTrue, Reason: "Ready", LastTransitionTime: now},
			},
			want: []metav1.Condition{
				{Type: "InstanceReady", Status: metav1.ConditionTrue, Reason: "Ready", LastTransitionTime: now},
			},
		},
		{
			name: "empty reasons are filled and everything else is preserved",
			conditions: []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionTrue, LastTransitionTime: now},
				{Type: "A", Status: metav1.ConditionFalse, LastTransitionTime: now, Message: "boom"},
				{Type: "B", Status: metav1.ConditionUnknown, LastTransitionTime: now},
				{Type: "InstanceReady", Status: metav1.ConditionTrue, Reason: "Custom", LastTransitionTime: now},
			},
			want: []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionTrue, Reason: "Ready", LastTransitionTime: now},
				{Type: "A", Status: metav1.ConditionFalse, Reason: "NotReady", LastTransitionTime: now, Message: "boom"},
				{Type: "B", Status: metav1.ConditionUnknown, Reason: "Unknown", LastTransitionTime: now},
				{Type: "InstanceReady", Status: metav1.ConditionTrue, Reason: "Custom", LastTransitionTime: now},
			},
			wantChanged: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			server := &infrav1alpha1.OpenStackServer{
				Status: infrav1alpha1.OpenStackServerStatus{Conditions: tt.conditions},
			}
			g.Expect(EnsureConditionReasons(server)).To(Equal(tt.wantChanged))
			g.Expect(server.Status.Conditions).To(Equal(tt.want))
		})
	}
}
