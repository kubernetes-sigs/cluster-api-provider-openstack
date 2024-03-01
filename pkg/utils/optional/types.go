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

// String is a string that can be unspecified. strings which are converted to
// optional.String during API conversion will be converted to nil if the value
// was previously the empty string.
type String *string

// Int is an int that can be unspecified. ints which are converted to
// optional.Int during API conversion will be converted to nil if the value
// was previously 0.
type Int *int
