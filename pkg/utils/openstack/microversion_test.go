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

package openstack

import "testing"

type microversionSupported struct {
	Version    string
	MinVersion string
	MaxVersion string
	Supported  bool
	Error      bool
}

func TestMicroversionSupported(t *testing.T) {
	tests := []microversionSupported{
		{
			// Checking min version
			Version:    "2.1",
			MinVersion: "2.1",
			MaxVersion: "2.90",
			Supported:  true,
			Error:      false,
		},
		{
			// Checking max version
			Version:    "2.90",
			MinVersion: "2.1",
			MaxVersion: "2.90",
			Supported:  true,
			Error:      false,
		},
		{
			// Checking too high version
			Version:    "2.95",
			MinVersion: "2.1",
			MaxVersion: "2.90",
			Supported:  false,
			Error:      false,
		},
		{
			// Checking too low version
			Version:    "2.1",
			MinVersion: "2.53",
			MaxVersion: "2.90",
			Supported:  false,
			Error:      false,
		},
		{
			// Invalid version
			Version:    "2.1.53",
			MinVersion: "2.53",
			MaxVersion: "2.90",
			Supported:  false,
			Error:      true,
		},
		{
			// No microversions supported
			Version:    "2.1",
			MinVersion: "",
			MaxVersion: "",
			Supported:  false,
			Error:      true,
		},
	}

	for _, test := range tests {
		supported, err := MicroversionSupported(test.Version, test.MinVersion, test.MaxVersion)
		if test.Error {
			if err == nil {
				t.Error("Expected error but got none!")
			}
		} else {
			if err != nil {
				t.Errorf("Expected no error but got %s", err.Error())
			}
		}
		if test.Supported != supported {
			t.Errorf("Expected supported=%t to be %t, when version=%s, min=%s and max=%s",
				supported, test.Supported, test.Version, test.MinVersion, test.MaxVersion)
		}
	}
}
