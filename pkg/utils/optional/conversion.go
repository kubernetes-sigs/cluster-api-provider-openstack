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
	"k8s.io/apimachinery/pkg/conversion"
)

func Convert_string_To_optional_String(in *string, out *String, _ conversion.Scope) error {
	// NOTE: This function has the opposite defaulting behaviour to
	// Convert_string_to_Pointer_string defined in apimachinery: it converts
	// empty strings to nil instead of a pointer to an empty string.

	if *in == "" {
		*out = nil
	} else {
		*out = in
	}
	return nil
}

func Convert_optional_String_To_string(in *String, out *string, _ conversion.Scope) error {
	if *in == nil {
		*out = ""
	} else {
		*out = **in
	}
	return nil
}

func Convert_int_To_optional_Int(in *int, out *Int, _ conversion.Scope) error {
	if *in == 0 {
		*out = nil
	} else {
		*out = in
	}
	return nil
}

func Convert_optional_Int_To_int(in *Int, out *int, _ conversion.Scope) error {
	if *in == nil {
		*out = 0
	} else {
		*out = **in
	}
	return nil
}
