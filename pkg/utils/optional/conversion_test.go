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

package optional

import (
	"testing"

	"k8s.io/utils/ptr"
)

func TestConvert_bool_To_optional_Bool(t *testing.T) {
	tests := []struct {
		name     string
		in       bool
		wantNil  bool
		wantBool bool
	}{
		{
			name:     "true is preserved",
			in:       true,
			wantNil:  false,
			wantBool: true,
		},
		{
			name:     "false is preserved as explicit false, not nil",
			in:       false,
			wantNil:  false,
			wantBool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			var out Bool
			if err := Convert_bool_To_optional_Bool(&in, &out, nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if out != nil {
					t.Errorf("expected nil, got %v", *out)
				}
			} else {
				if out == nil {
					t.Fatalf("expected non-nil, got nil")
				}
				if *out != tt.wantBool {
					t.Errorf("expected %v, got %v", tt.wantBool, *out)
				}
			}
		})
	}
}

func TestConvert_optional_Bool_To_bool(t *testing.T) {
	tests := []struct {
		name string
		in   Bool
		want bool
	}{
		{
			name: "nil converts to false",
			in:   nil,
			want: false,
		},
		{
			name: "true converts to true",
			in:   ptr.To(true),
			want: true,
		},
		{
			name: "false converts to false",
			in:   ptr.To(false),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			var out bool
			if err := Convert_optional_Bool_To_bool(&in, &out, nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != tt.want {
				t.Errorf("expected %v, got %v", tt.want, out)
			}
		})
	}
}

func TestRestoreBool(t *testing.T) {
	tests := []struct {
		name     string
		previous Bool
		dst      Bool
		wantNil  bool
		wantBool bool
	}{
		{
			name:     "nil dst is restored from previous true",
			previous: ptr.To(true),
			dst:      nil,
			wantNil:  false,
			wantBool: true,
		},
		{
			name:     "nil dst is restored from previous false",
			previous: ptr.To(false),
			dst:      nil,
			wantNil:  false,
			wantBool: false,
		},
		{
			name:    "nil dst with nil previous stays nil",
			previous: nil,
			dst:      nil,
			wantNil:  true,
		},
		{
			name:     "explicit false dst is NOT overwritten",
			previous: ptr.To(true),
			dst:      ptr.To(false),
			wantNil:  false,
			wantBool: false,
		},
		{
			name:     "explicit true dst is NOT overwritten",
			previous: ptr.To(false),
			dst:      ptr.To(true),
			wantNil:  false,
			wantBool: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previous := tt.previous
			dst := tt.dst
			RestoreBool(&previous, &dst)
			if tt.wantNil {
				if dst != nil {
					t.Errorf("expected nil, got %v", *dst)
				}
			} else {
				if dst == nil {
					t.Fatalf("expected non-nil, got nil")
				}
				if *dst != tt.wantBool {
					t.Errorf("expected %v, got %v", tt.wantBool, *dst)
				}
			}
		})
	}
}
