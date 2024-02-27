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

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
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
			wantErr: true,
		},
		{
			name: "multiple IPv6 subnets",
			subnets: []infrav1.Subnet{
				{
					CIDR: "2001:db8:2222:5555::/64",
				},
				{
					CIDR: "2001:db8:2222:5555::/64",
				},
			},
			wantErr: true,
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
