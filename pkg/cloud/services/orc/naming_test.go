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

package orc

import (
	"testing"

	"k8s.io/utils/ptr"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
)

func TestNamingFunctions(t *testing.T) {
	const server = "my-server"

	tests := []struct {
		name string
		fn   func() string
		want string
	}{
		{"ImageName", func() string { return ImageName(server) }, "my-server-image"},
		{"FlavorName", func() string { return FlavorName(server) }, "my-server-flavor"},
		{"KeyPairName", func() string { return KeyPairName(server) }, "my-server-keypair"},
		{"ServerGroupORCName", func() string { return ServerGroupORCName(server) }, "my-server-servergroup"},
		{"ServerName", func() string { return ServerName(server) }, "my-server"},
		{"PortORCName 0", func() string { return PortORCName(server, 0) }, "my-server-port-0"},
		{"PortORCName 3", func() string { return PortORCName(server, 3) }, "my-server-port-3"},
		{"TrunkORCName 1", func() string { return TrunkORCName(server, 1) }, "my-server-trunk-1"},
		{"RootVolumeName", func() string { return RootVolumeName(server) }, "my-server-vol-root"},
		{"AdditionalVolumeName", func() string { return AdditionalVolumeName(server, "data") }, "my-server-vol-data"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHash6_Deterministic(t *testing.T) {
	a := hash6("same-input")
	b := hash6("same-input")
	if a != b {
		t.Errorf("hash6 not deterministic: %q != %q", a, b)
	}
	if len(a) != 6 {
		t.Errorf("hash6 length: got %d, want 6", len(a))
	}
}

func TestHash6_DifferentInputs(t *testing.T) {
	a := hash6("input-a")
	b := hash6("input-b")
	if a == b {
		t.Errorf("hash6 collision: %q == %q for different inputs", a, b)
	}
}

func TestHashedNames_Deterministic(t *testing.T) {
	// Hashed names must be deterministic across calls.
	const server = "srv"
	key := "id:abc-123"

	networkFirst, networkSecond := NetworkORCName(server, key), NetworkORCName(server, key)
	if networkFirst != networkSecond {
		t.Error("NetworkORCName not deterministic")
	}
	subnetFirst, subnetSecond := SubnetORCName(server, key), SubnetORCName(server, key)
	if subnetFirst != subnetSecond {
		t.Error("SubnetORCName not deterministic")
	}
	sgFirst, sgSecond := SecurityGroupORCName(server, key), SecurityGroupORCName(server, key)
	if sgFirst != sgSecond {
		t.Error("SecurityGroupORCName not deterministic")
	}
	volTypeFirst, volTypeSecond := VolumeTypeORCName(server, key), VolumeTypeORCName(server, key)
	if volTypeFirst != volTypeSecond {
		t.Error("VolumeTypeORCName not deterministic")
	}
}

func TestNetworkParamKey(t *testing.T) {
	tests := []struct {
		name  string
		param infrav1.NetworkParam
		want  string
	}{
		{
			name:  "by ID",
			param: infrav1.NetworkParam{ID: ptr.To("net-uuid")},
			want:  "id:net-uuid",
		},
		{
			name: "by filter name",
			param: infrav1.NetworkParam{
				Filter: &infrav1.NetworkFilter{Name: "my-net"},
			},
		},
		{
			name:  "empty",
			param: infrav1.NetworkParam{},
			want:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NetworkParamKey(tt.param)
			if tt.want != "" && got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
			if tt.want == "" && tt.param.Filter != nil {
				// Filter-based keys should be non-empty and start with "filter:"
				if got == "" || got[:7] != "filter:" {
					t.Errorf("filter key should start with 'filter:', got %q", got)
				}
			}
		})
	}
}

func TestSubnetParamKey(t *testing.T) {
	got := SubnetParamKey(infrav1.SubnetParam{ID: ptr.To("sub-uuid")})
	if got != "id:sub-uuid" {
		t.Errorf("got %q, want %q", got, "id:sub-uuid")
	}
}

func TestSecurityGroupParamKey(t *testing.T) {
	got := SecurityGroupParamKey(infrav1.SecurityGroupParam{ID: ptr.To("sg-uuid")})
	if got != "id:sg-uuid" {
		t.Errorf("got %q, want %q", got, "id:sg-uuid")
	}
}
