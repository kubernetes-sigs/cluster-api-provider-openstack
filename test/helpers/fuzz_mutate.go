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

package helpers

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/client-go/kubernetes/scheme"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FuzzMutateTestFuncInput[hubObject, spokeObject client.Object] struct {
	utilconversion.SpokeConverterFuzzTestFuncInput[hubObject, spokeObject]
	MutateFuzzerFuncs []fuzzer.FuzzerFuncs
}

func newObject[T client.Object]() T {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if t.Kind() == reflect.Pointer {
		return reflect.New(t.Elem()).Interface().(T)
	}
	return reflect.New(t).Elem().Interface().(T)
}

func FuzzMutateTestFunc[hubObject, spokeObject client.Object](input FuzzMutateTestFuncInput[hubObject, spokeObject]) func(*testing.T) {
	if input.Scheme == nil {
		input.Scheme = scheme.Scheme
	}

	return func(t *testing.T) {
		t.Helper()
		fuzzer := utilconversion.GetFuzzer(input.Scheme, input.FuzzerFuncs...)
		mutateFuzzer := utilconversion.GetFuzzer(input.Scheme, input.MutateFuzzerFuncs...)

		t.Run("spoke-hub-mutate-spoke-hub", func(t *testing.T) {
			g := gomega.NewWithT(t)
			for i := 0; i < 10000; i++ {
				// Create the spoke and fuzz it
				spokeBefore := newObject[spokeObject]()
				fuzzer.Fill(spokeBefore)

				// First convert spoke to hub
				hubBefore := newObject[hubObject]()
				g.Expect(input.ConvertSpokeToHubFunc(t.Context(), spokeBefore, hubBefore)).To(gomega.Succeed())

				// Fuzz the converted hub with permitted mutations
				mutateFuzzer.Fill(hubBefore)

				// Convert hub back to spoke
				spokeAfter := newObject[spokeObject]()
				g.Expect(input.ConvertHubToSpokeFunc(t.Context(), hubBefore.DeepCopyObject().(hubObject), spokeAfter)).To(gomega.Succeed())

				// Convert spoke back to hub and check if the resulting hub is equal to the hub before the round trip
				hubAfter := newObject[hubObject]()
				g.Expect(input.ConvertSpokeToHubFunc(t.Context(), spokeAfter, hubAfter)).To(gomega.Succeed())

				if input.HubAfterMutation != nil {
					input.HubAfterMutation(hubAfter)
				}

				g.Expect(apiequality.Semantic.DeepEqual(hubBefore, hubAfter)).To(gomega.BeTrue(), cmp.Diff(hubBefore, hubAfter))
			}
		})

		t.Run("hub-spoke-mutate-hub-spoke", func(t *testing.T) {
			g := gomega.NewWithT(t)
			for i := 0; i < 10000; i++ {
				// Create the hub and fuzz it
				hubBefore := newObject[hubObject]()
				fuzzer.Fill(hubBefore)

				// First convert hub to spoke
				spokeBefore := newObject[spokeObject]()
				g.Expect(input.ConvertHubToSpokeFunc(t.Context(), hubBefore, spokeBefore)).To(gomega.Succeed())

				// Fuzz the converted spoke with permitted mutations
				mutateFuzzer.Fill(spokeBefore)

				// Convert spoke back to hub
				hubAfter := newObject[hubObject]()
				g.Expect(input.ConvertSpokeToHubFunc(t.Context(), spokeBefore, hubAfter)).To(gomega.Succeed())

				// Convert hub back to spoke and check if the resulting spoke is equal to the spoke before the round trip
				spokeAfter := newObject[spokeObject]()
				g.Expect(input.ConvertHubToSpokeFunc(t.Context(), hubAfter, spokeAfter)).To(gomega.Succeed())

				// Remove the data annotation added during hub-to-spoke conversion,
				// unless the caller needs it for comparison.
				if !input.SkipSpokeAnnotationCleanup {
					delete(spokeAfter.GetAnnotations(), utilconversion.DataAnnotation)
				}

				if input.SpokeAfterMutation != nil {
					input.SpokeAfterMutation(spokeAfter)
				}

				g.Expect(apiequality.Semantic.DeepEqual(spokeBefore, spokeAfter)).To(gomega.BeTrue(), cmp.Diff(spokeBefore, spokeAfter))
			}
		})
	}
}
