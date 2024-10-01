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

package objects

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetGVK(obj runtime.Object, scheme *runtime.Scheme) (*schema.GroupVersionKind, error) {
	// Return it if it's already set on the object
	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk.Kind != "" {
		return &gvk, nil
	}

	gvks, _, err := scheme.ObjectKinds(obj)
	if err != nil {
		return nil, err
	}
	if len(gvks) == 0 {
		// This is probably a programming error
		return nil, fmt.Errorf("scheme does not contain a gvk mapping for %T", obj)
	}

	return &gvks[0], nil
}
