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

package helpers

import (
	"runtime/debug"
	"testing"

	"github.com/onsi/gomega/format"
	"k8s.io/client-go/kubernetes/scheme"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
)

// FuzzRestorer fuzzes the inputs to a restore function and ensures that the function does not panic.
func FuzzRestorer[T any](t *testing.T, name string, f func(*T, *T)) {
	t.Helper()
	fuzz := utilconversion.GetFuzzer(scheme.Scheme)

	t.Run(name, func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			previous := new(T)
			dst := new(T)
			fuzz.Fill(previous)
			fuzz.Fill(dst)

			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("PANIC with arguments\nPrevious: %s\nDest: %s\nStack: %s",
							format.Object(previous, 1),
							format.Object(dst, 1),
							debug.Stack())
						t.FailNow()
					}
				}()
				f(previous, dst)
			}()
		}
	})
}
