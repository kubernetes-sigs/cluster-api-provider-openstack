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

package v1beta2

import (
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
)

func TestConvertConditionsToV1Beta2(t *testing.T) {
	g := NewWithT(t)

	// Nil input returns nil
	g.Expect(ConvertConditionsToV1Beta2(nil, 5)).To(BeNil())

	// Empty slice returns empty (not nil)
	result := ConvertConditionsToV1Beta2(clusterv1beta1.Conditions{}, 5)
	g.Expect(result).NotTo(BeNil())
	g.Expect(result).To(HaveLen(0))

	// Standard conversion with ObservedGeneration
	now := metav1.Now()
	src := clusterv1beta1.Conditions{
		{
			Type:               clusterv1beta1.ReadyCondition,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: now,
			Reason:             "AllReady",
			Message:            "All components ready",
			Severity:           clusterv1beta1.ConditionSeverityInfo,
		},
		{
			Type:               "NetworkReady",
			Status:             corev1.ConditionFalse,
			LastTransitionTime: now,
			Reason:             "NetworkFailed",
			Message:            "Network creation failed",
			Severity:           clusterv1beta1.ConditionSeverityError,
		},
		{
			Type:               "SecurityGroupsReady",
			Status:             corev1.ConditionUnknown,
			LastTransitionTime: now,
			Reason:             "Pending",
			Message:            "",
			Severity:           clusterv1beta1.ConditionSeverityWarning,
		},
	}

	dst := ConvertConditionsToV1Beta2(src, 42)
	g.Expect(dst).To(HaveLen(3))

	// Verify all fields mapped correctly
	g.Expect(dst[0].Type).To(Equal("Ready"))
	g.Expect(dst[0].Status).To(Equal(metav1.ConditionTrue))
	g.Expect(dst[0].ObservedGeneration).To(Equal(int64(42)))
	g.Expect(dst[0].LastTransitionTime).To(Equal(now))
	g.Expect(dst[0].Reason).To(Equal("AllReady"))
	g.Expect(dst[0].Message).To(Equal("All components ready"))

	g.Expect(dst[1].Type).To(Equal("NetworkReady"))
	g.Expect(dst[1].Status).To(Equal(metav1.ConditionFalse))

	// Unknown status maps correctly
	g.Expect(dst[2].Status).To(Equal(metav1.ConditionStatus(corev1.ConditionUnknown)))
	g.Expect(dst[2].Message).To(Equal(""))

	// ObservedGeneration 0 is valid
	zeroGen := ConvertConditionsToV1Beta2(src[:1], 0)
	g.Expect(zeroGen[0].ObservedGeneration).To(Equal(int64(0)))
}

func TestConvertConditionsFromV1Beta2(t *testing.T) {
	g := NewWithT(t)

	// Nil input returns nil
	g.Expect(ConvertConditionsFromV1Beta2(nil)).To(BeNil())

	// Empty slice returns empty (not nil)
	result := ConvertConditionsFromV1Beta2([]metav1.Condition{})
	g.Expect(result).NotTo(BeNil())
	g.Expect(result).To(HaveLen(0))

	// Standard conversion - Severity is always None (lost in v1beta2)
	now := metav1.Now()
	src := []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			ObservedGeneration: 10,
			LastTransitionTime: now,
			Reason:             "AllReady",
			Message:            "Ready",
		},
	}

	dst := ConvertConditionsFromV1Beta2(src)
	g.Expect(dst).To(HaveLen(1))
	g.Expect(dst[0].Type).To(Equal(clusterv1beta1.ConditionType("Ready")))
	g.Expect(dst[0].Status).To(Equal(corev1.ConditionTrue))
	g.Expect(dst[0].Severity).To(Equal(clusterv1beta1.ConditionSeverityNone))
	g.Expect(dst[0].LastTransitionTime).To(Equal(now))
	g.Expect(dst[0].Reason).To(Equal("AllReady"))
	g.Expect(dst[0].Message).To(Equal("Ready"))
}

func TestConvertConditionsRoundTrip(t *testing.T) {
	g := NewWithT(t)

	// v1beta1 → v1beta2 → v1beta1 preserves all fields except Severity
	now := metav1.Now()
	original := clusterv1beta1.Conditions{
		{
			Type:               "Ready",
			Status:             corev1.ConditionTrue,
			LastTransitionTime: now,
			Reason:             "AllReady",
			Message:            "All components ready",
			Severity:           clusterv1beta1.ConditionSeverityInfo,
		},
		{
			Type:               "NetworkReady",
			Status:             corev1.ConditionFalse,
			LastTransitionTime: now,
			Reason:             "NetworkFailed",
			Message:            "Network error",
			Severity:           clusterv1beta1.ConditionSeverityError,
		},
	}

	v1beta2 := ConvertConditionsToV1Beta2(original, 5)
	restored := ConvertConditionsFromV1Beta2(v1beta2)

	g.Expect(restored).To(HaveLen(2))

	// Type, Status, LastTransitionTime, Reason, Message are preserved
	for i := range original {
		g.Expect(restored[i].Type).To(Equal(original[i].Type))
		g.Expect(restored[i].Status).To(Equal(original[i].Status))
		g.Expect(restored[i].LastTransitionTime).To(Equal(original[i].LastTransitionTime))
		g.Expect(restored[i].Reason).To(Equal(original[i].Reason))
		g.Expect(restored[i].Message).To(Equal(original[i].Message))
	}

	// Severity is always lost
	g.Expect(restored[0].Severity).To(Equal(clusterv1beta1.ConditionSeverityNone))
	g.Expect(restored[1].Severity).To(Equal(clusterv1beta1.ConditionSeverityNone))
}

func TestIsReady(t *testing.T) {
	g := NewWithT(t)

	// Ready condition True
	g.Expect(IsReady([]metav1.Condition{
		{Type: "Ready", Status: metav1.ConditionTrue},
	})).To(BeTrue())

	// Ready condition False
	g.Expect(IsReady([]metav1.Condition{
		{Type: "Ready", Status: metav1.ConditionFalse},
	})).To(BeFalse())

	// Ready condition Unknown
	g.Expect(IsReady([]metav1.Condition{
		{Type: "Ready", Status: metav1.ConditionStatus(corev1.ConditionUnknown)},
	})).To(BeFalse())

	// No Ready condition among others
	g.Expect(IsReady([]metav1.Condition{
		{Type: "NetworkReady", Status: metav1.ConditionTrue},
		{Type: "SecurityGroupsReady", Status: metav1.ConditionTrue},
	})).To(BeFalse())

	// Ready is True among multiple conditions
	g.Expect(IsReady([]metav1.Condition{
		{Type: "NetworkReady", Status: metav1.ConditionTrue},
		{Type: "Ready", Status: metav1.ConditionTrue},
		{Type: "SecurityGroupsReady", Status: metav1.ConditionFalse},
	})).To(BeTrue())

	// Nil conditions
	g.Expect(IsReady(nil)).To(BeFalse())

	// Empty conditions
	g.Expect(IsReady([]metav1.Condition{})).To(BeFalse())
}
