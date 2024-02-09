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

package strings

import (
	"slices"
	"testing"
)

func TestCanonicalize(t *testing.T) {
	tests := []struct {
		name  string
		value []string
		want  []string
	}{
		{
			name:  "Empty list",
			value: []string{},
			want:  []string{},
		},
		{
			name:  "Identity",
			value: []string{"a", "b", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "Out of order",
			value: []string{"c", "b", "a"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "Duplicate elements",
			value: []string{"c", "b", "a", "c"},
			want:  []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Canonicalize(tt.value)
			if !slices.Equal(got, tt.want) {
				t.Errorf("CompareLists() = %v, want %v", got, tt.want)
			}
		})
	}
}
