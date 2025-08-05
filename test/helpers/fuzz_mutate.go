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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

type FuzzMutateTestFuncInput struct {
	utilconversion.FuzzTestFuncInput
	MutateFuzzerFuncs []fuzzer.FuzzerFuncs
}

func FuzzMutateTestFunc(input FuzzMutateTestFuncInput) func(*testing.T) {
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
				spokeBefore := input.Spoke.DeepCopyObject().(conversion.Convertible)
				fuzzer.Fill(spokeBefore)

				// First convert spoke to hub
				hubBefore := input.Hub.DeepCopyObject().(conversion.Hub)
				g.Expect(spokeBefore.ConvertTo(hubBefore)).To(gomega.Succeed())

				// Fuzz the converted hub with permitted mutations
				mutateFuzzer.Fill(hubBefore)

				// Convert hub back to spoke
				spokeAfter := input.Spoke.DeepCopyObject().(conversion.Convertible)
				g.Expect(spokeAfter.ConvertFrom(hubBefore)).To(gomega.Succeed())

				// Convert spoke back to hub and check if the resulting hub is equal to the hub before the round trip
				hubAfter := input.Hub.DeepCopyObject().(conversion.Hub)
				g.Expect(spokeAfter.ConvertTo(hubAfter)).To(gomega.Succeed())

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
				hubBefore := input.Hub.DeepCopyObject().(conversion.Hub)
				fuzzer.Fill(hubBefore)

				// First convert hub to spoke
				spokeBefore := input.Spoke.DeepCopyObject().(conversion.Convertible)
				g.Expect(spokeBefore.ConvertFrom(hubBefore)).To(gomega.Succeed())

				// Fuzz the converted spoke with permitted mutations
				mutateFuzzer.Fill(spokeBefore)

				// Convert spoke back to hub
				hubAfter := input.Hub.DeepCopyObject().(conversion.Hub)
				g.Expect(spokeBefore.ConvertTo(hubAfter)).To(gomega.Succeed())

				// Convert hub back to spoke and check if the resulting spoke is equal to the spoke before the round trip
				spokeAfter := input.Spoke.DeepCopyObject().(conversion.Convertible)
				g.Expect(spokeAfter.ConvertFrom(hubAfter)).To(gomega.Succeed())

				// Remove data annotation eventually added by ConvertFrom for avoiding data loss in hub-spoke-hub round trips
				// NOTE: There are use case when we want to skip this operation, e.g. if the spoke object does not have ObjectMeta (e.g. kubeadm types).
				if !input.SkipSpokeAnnotationCleanup {
					metaAfter := spokeAfter.(metav1.Object)
					delete(metaAfter.GetAnnotations(), utilconversion.DataAnnotation)
				}

				if input.SpokeAfterMutation != nil {
					input.SpokeAfterMutation(spokeAfter)
				}

				g.Expect(apiequality.Semantic.DeepEqual(spokeBefore, spokeAfter)).To(gomega.BeTrue(), cmp.Diff(spokeBefore, spokeAfter))
			}
		})
	}
}
