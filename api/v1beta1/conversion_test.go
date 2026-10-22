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

package v1beta1

import (
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/optional"
)

func TestConditionConversion(t *testing.T) {
	g := NewWithT(t)

	// Test v1beta1 -> v1beta2 condition conversion
	v1beta1Conditions := clusterv1beta1.Conditions{
		{
			Type:               "Ready",
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "AllComponentsReady",
			Message:            "All components are ready",
			Severity:           clusterv1beta1.ConditionSeverityInfo,
		},
		{
			Type:               "NetworkReady",
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "NetworkCreateFailed",
			Message:            "Failed to create network",
			Severity:           clusterv1beta1.ConditionSeverityError,
		},
	}

	v1beta2Conditions := infrav1.ConvertConditionsToV1Beta2(v1beta1Conditions, 10)

	g.Expect(v1beta2Conditions).To(HaveLen(2))
	g.Expect(v1beta2Conditions[0].Type).To(Equal("Ready"))
	g.Expect(v1beta2Conditions[0].Status).To(Equal(metav1.ConditionTrue))
	g.Expect(v1beta2Conditions[0].ObservedGeneration).To(Equal(int64(10)))
	g.Expect(v1beta2Conditions[1].Type).To(Equal("NetworkReady"))
	g.Expect(v1beta2Conditions[1].Status).To(Equal(metav1.ConditionFalse))

	// Test v1beta2 -> v1beta1 condition conversion
	restoredConditions := infrav1.ConvertConditionsFromV1Beta2(v1beta2Conditions)

	g.Expect(restoredConditions).To(HaveLen(2))
	g.Expect(restoredConditions[0].Type).To(Equal(clusterv1beta1.ConditionType("Ready")))
	g.Expect(restoredConditions[0].Status).To(Equal(corev1.ConditionTrue))
	// Severity is lost during conversion
	g.Expect(restoredConditions[0].Severity).To(Equal(clusterv1beta1.ConditionSeverityNone))
}

func TestOpenStackClusterListConversion(t *testing.T) {
	g := NewWithT(t)

	src := &OpenStackClusterList{
		Items: []OpenStackCluster{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "cluster-1",
					Namespace:  "default",
					Generation: 1,
				},
				Spec: OpenStackClusterSpec{
					IdentityRef: OpenStackIdentityReference{
						Name:      "cloud-config",
						CloudName: "openstack",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "cluster-2",
					Namespace:  "default",
					Generation: 2,
				},
				Spec: OpenStackClusterSpec{
					IdentityRef: OpenStackIdentityReference{
						Name:      "cloud-config-2",
						CloudName: "openstack",
					},
				},
			},
		},
	}

	// Convert to v1beta2
	dst := &infrav1.OpenStackClusterList{}
	g.Expect(Convert_v1beta1_OpenStackClusterList_To_v1beta2_OpenStackClusterList(src, dst, nil)).To(Succeed())

	// Verify items converted
	g.Expect(dst.Items).To(HaveLen(2))
	g.Expect(dst.Items[0].Name).To(Equal("cluster-1"))
	g.Expect(dst.Items[1].Name).To(Equal("cluster-2"))

	// Convert back
	restored := &OpenStackClusterList{}
	g.Expect(Convert_v1beta2_OpenStackClusterList_To_v1beta1_OpenStackClusterList(dst, restored, nil)).To(Succeed())

	// Verify round-trip
	g.Expect(restored.Items).To(HaveLen(2))
	g.Expect(restored.Items[0].Name).To(Equal("cluster-1"))
	g.Expect(restored.Items[1].Spec.IdentityRef.Name).To(Equal("cloud-config-2"))
}

func TestSplitTags(t *testing.T) {
	g := NewWithT(t)

	// Single tag
	g.Expect(splitTags("foo")).To(Equal([]NeutronTag{"foo"}))

	// Multiple tags
	g.Expect(splitTags("foo,bar,baz")).To(Equal([]NeutronTag{"foo", "bar", "baz"}))

	// Empty string returns nil
	g.Expect(splitTags("")).To(BeNil())

	// Trailing comma (empty segments are skipped)
	g.Expect(splitTags("foo,")).To(Equal([]NeutronTag{"foo"}))

	// Leading comma
	g.Expect(splitTags(",foo")).To(Equal([]NeutronTag{"foo"}))

	// Multiple consecutive commas
	g.Expect(splitTags("foo,,bar")).To(Equal([]NeutronTag{"foo", "bar"}))

	// Only commas
	g.Expect(splitTags(",,,")).To(BeNil())
}

func TestJoinTags(t *testing.T) {
	g := NewWithT(t)

	// Single tag
	g.Expect(JoinTags([]NeutronTag{"foo"})).To(Equal("foo"))

	// Multiple tags
	g.Expect(JoinTags([]NeutronTag{"foo", "bar", "baz"})).To(Equal("foo,bar,baz"))

	// Nil slice returns empty string
	g.Expect(JoinTags(nil)).To(Equal(""))

	// Empty slice returns empty string
	g.Expect(JoinTags([]NeutronTag{})).To(Equal(""))
}

func TestConvertAllTagsRoundTrip(t *testing.T) {
	g := NewWithT(t)

	// Test ConvertAllTagsTo: strings → struct
	var neutronTags FilterByNeutronTags
	ConvertAllTagsTo("web,api", "prod", "deprecated", "staging,dev", &neutronTags)

	g.Expect(neutronTags.Tags).To(Equal([]NeutronTag{"web", "api"}))
	g.Expect(neutronTags.TagsAny).To(Equal([]NeutronTag{"prod"}))
	g.Expect(neutronTags.NotTags).To(Equal([]NeutronTag{"deprecated"}))
	g.Expect(neutronTags.NotTagsAny).To(Equal([]NeutronTag{"staging", "dev"}))

	// Test ConvertAllTagsFrom: struct → strings (round-trip)
	var tags, tagsAny, notTags, notTagsAny string
	ConvertAllTagsFrom(&neutronTags, &tags, &tagsAny, &notTags, &notTagsAny)

	g.Expect(tags).To(Equal("web,api"))
	g.Expect(tagsAny).To(Equal("prod"))
	g.Expect(notTags).To(Equal("deprecated"))
	g.Expect(notTagsAny).To(Equal("staging,dev"))

	// Test with all empty strings
	var emptyTags FilterByNeutronTags
	ConvertAllTagsTo("", "", "", "", &emptyTags)
	g.Expect(emptyTags.Tags).To(BeNil())
	g.Expect(emptyTags.TagsAny).To(BeNil())
	g.Expect(emptyTags.NotTags).To(BeNil())
	g.Expect(emptyTags.NotTagsAny).To(BeNil())

	// Round-trip the empty case
	ConvertAllTagsFrom(&emptyTags, &tags, &tagsAny, &notTags, &notTagsAny)
	g.Expect(tags).To(Equal(""))
	g.Expect(tagsAny).To(Equal(""))
	g.Expect(notTags).To(Equal(""))
	g.Expect(notTagsAny).To(Equal(""))

	// Test with mixed empty and non-empty
	var mixedTags FilterByNeutronTags
	ConvertAllTagsTo("web", "", "old", "", &mixedTags)
	g.Expect(mixedTags.Tags).To(Equal([]NeutronTag{"web"}))
	g.Expect(mixedTags.TagsAny).To(BeNil())
	g.Expect(mixedTags.NotTags).To(Equal([]NeutronTag{"old"}))
	g.Expect(mixedTags.NotTagsAny).To(BeNil())
}

func TestOpenStackMachineListConversion(t *testing.T) {
	g := NewWithT(t)

	src := &OpenStackMachineList{
		Items: []OpenStackMachine{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machine-1",
					Namespace: "default",
				},
				Spec: OpenStackMachineSpec{
					Flavor: ptr.To("m1.small"),
					Image: ImageParam{
						Filter: &ImageFilter{
							Name: ptr.To("ubuntu-22.04"),
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machine-2",
					Namespace: "default",
				},
				Spec: OpenStackMachineSpec{
					Flavor: ptr.To("m1.large"),
					Image: ImageParam{
						Filter: &ImageFilter{
							Name: ptr.To("ubuntu-24.04"),
						},
					},
				},
			},
		},
	}

	// Convert to v1beta2
	dst := &infrav1.OpenStackMachineList{}
	g.Expect(Convert_v1beta1_OpenStackMachineList_To_v1beta2_OpenStackMachineList(src, dst, nil)).To(Succeed())

	g.Expect(dst.Items).To(HaveLen(2))
	g.Expect(dst.Items[0].Name).To(Equal("machine-1"))
	g.Expect(dst.Items[0].Spec.Flavor.Filter.Name).To(Equal(optional.String(ptr.To("m1.small"))))
	g.Expect(dst.Items[1].Name).To(Equal("machine-2"))
	g.Expect(dst.Items[1].Spec.Flavor.Filter.Name).To(Equal(optional.String(ptr.To("m1.large"))))

	// Convert back
	restored := &OpenStackMachineList{}
	g.Expect(Convert_v1beta2_OpenStackMachineList_To_v1beta1_OpenStackMachineList(dst, restored, nil)).To(Succeed())

	g.Expect(restored.Items).To(HaveLen(2))
	g.Expect(restored.Items[0].Name).To(Equal("machine-1"))
	g.Expect(restored.Items[1].Name).To(Equal("machine-2"))
	g.Expect(restored.Items[1].Spec.Flavor).To(Equal(ptr.To("m1.large")))
}

func TestOpenStackClusterTemplateListConversion(t *testing.T) {
	g := NewWithT(t)

	src := &OpenStackClusterTemplateList{
		Items: []OpenStackClusterTemplate{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "template-1",
					Namespace: "default",
				},
				Spec: OpenStackClusterTemplateSpec{
					Template: OpenStackClusterTemplateResource{
						Spec: OpenStackClusterSpec{
							IdentityRef: OpenStackIdentityReference{
								Name:      "cloud-1",
								CloudName: "openstack",
							},
						},
					},
				},
			},
		},
	}

	// Convert to v1beta2
	dst := &infrav1.OpenStackClusterTemplateList{}
	g.Expect(Convert_v1beta1_OpenStackClusterTemplateList_To_v1beta2_OpenStackClusterTemplateList(src, dst, nil)).To(Succeed())

	g.Expect(dst.Items).To(HaveLen(1))
	g.Expect(dst.Items[0].Name).To(Equal("template-1"))
	g.Expect(dst.Items[0].Spec.Template.Spec.IdentityRef.Name).To(Equal("cloud-1"))

	// Convert back
	restored := &OpenStackClusterTemplateList{}
	g.Expect(Convert_v1beta2_OpenStackClusterTemplateList_To_v1beta1_OpenStackClusterTemplateList(dst, restored, nil)).To(Succeed())

	g.Expect(restored.Items).To(HaveLen(1))
	g.Expect(restored.Items[0].Name).To(Equal("template-1"))
}

func TestOpenStackMachineTemplateListConversion(t *testing.T) {
	g := NewWithT(t)

	src := &OpenStackMachineTemplateList{
		Items: []OpenStackMachineTemplate{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mt-1",
					Namespace: "default",
				},
				Spec: OpenStackMachineTemplateSpec{
					Template: OpenStackMachineTemplateResource{
						Spec: OpenStackMachineSpec{
							Flavor: ptr.To("m1.xlarge"),
							Image: ImageParam{
								Filter: &ImageFilter{
									Name: ptr.To("centos-9"),
								},
							},
						},
					},
				},
			},
		},
	}

	// Convert to v1beta2
	dst := &infrav1.OpenStackMachineTemplateList{}
	g.Expect(Convert_v1beta1_OpenStackMachineTemplateList_To_v1beta2_OpenStackMachineTemplateList(src, dst, nil)).To(Succeed())

	g.Expect(dst.Items).To(HaveLen(1))
	g.Expect(dst.Items[0].Name).To(Equal("mt-1"))
	g.Expect(dst.Items[0].Spec.Template.Spec.Flavor.Filter.Name).To(Equal(optional.String(ptr.To("m1.xlarge"))))
	// Convert back
	restored := &OpenStackMachineTemplateList{}
	g.Expect(Convert_v1beta2_OpenStackMachineTemplateList_To_v1beta1_OpenStackMachineTemplateList(dst, restored, nil)).To(Succeed())

	g.Expect(restored.Items).To(HaveLen(1))
	g.Expect(restored.Items[0].Name).To(Equal("mt-1"))
}

func TestConditionConversionEdgeCases(t *testing.T) {
	g := NewWithT(t)

	// Nil conditions
	g.Expect(infrav1.ConvertConditionsToV1Beta2(nil, 5)).To(BeNil())
	g.Expect(infrav1.ConvertConditionsFromV1Beta2(nil)).To(BeNil())

	// Empty slice (not nil)
	emptyV1beta1 := clusterv1beta1.Conditions{}
	result := infrav1.ConvertConditionsToV1Beta2(emptyV1beta1, 5)
	g.Expect(result).To(HaveLen(0))
	g.Expect(result).NotTo(BeNil())

	emptyV1beta2 := []metav1.Condition{}
	resultBack := infrav1.ConvertConditionsFromV1Beta2(emptyV1beta2)
	g.Expect(resultBack).To(HaveLen(0))
	g.Expect(resultBack).NotTo(BeNil())

	// Condition with empty Reason and Message
	sparse := clusterv1beta1.Conditions{
		{
			Type:   "SomeCondition",
			Status: corev1.ConditionUnknown,
		},
	}
	converted := infrav1.ConvertConditionsToV1Beta2(sparse, 0)
	g.Expect(converted).To(HaveLen(1))
	g.Expect(converted[0].Reason).To(Equal(""))
	g.Expect(converted[0].Message).To(Equal(""))
	g.Expect(converted[0].ObservedGeneration).To(Equal(int64(0)))
}

func TestIsReadyHelper(t *testing.T) {
	g := NewWithT(t)

	// Test with Ready condition True
	conditions := []metav1.Condition{
		{
			Type:   "Ready",
			Status: metav1.ConditionTrue,
		},
	}
	g.Expect(infrav1.IsReady(conditions)).To(BeTrue())

	// Test with Ready condition False
	conditions = []metav1.Condition{
		{
			Type:   "Ready",
			Status: metav1.ConditionFalse,
		},
	}
	g.Expect(infrav1.IsReady(conditions)).To(BeFalse())

	// Test with no Ready condition
	conditions = []metav1.Condition{
		{
			Type:   "NetworkReady",
			Status: metav1.ConditionTrue,
		},
	}
	g.Expect(infrav1.IsReady(conditions)).To(BeFalse())

	// Test with empty conditions
	g.Expect(infrav1.IsReady(nil)).To(BeFalse())
	g.Expect(infrav1.IsReady([]metav1.Condition{})).To(BeFalse())
}

func TestResolvedPortSpecFields_RoundTrip_PortSecurity(t *testing.T) {
	tests := []struct {
		name             string
		in               ResolvedPortSpecFields
		expectedEnablePS *bool
	}{
		{
			name: "DisablePortSecurity explicitly true",
			in: ResolvedPortSpecFields{
				DisablePortSecurity: ptr.To(true),
			},
			expectedEnablePS: ptr.To(false), // disable=true → enable=false
		},
		{
			name: "DisablePortSecurity explicitly false",
			in: ResolvedPortSpecFields{
				DisablePortSecurity: ptr.To(false),
			},
			expectedEnablePS: ptr.To(true), // disable=false → enable=true
		},
		{
			name:             "DisablePortSecurity unset — EnablePortSecurity stays nil",
			in:               ResolvedPortSpecFields{},
			expectedEnablePS: nil, // unset stays unset, inherits from network level
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// --- Convert to v1beta2 ---
			out := &infrav1.ResolvedPortSpecFields{}
			g.Expect(Convert_v1beta1_ResolvedPortSpecFields_To_v1beta2_ResolvedPortSpecFields(&tt.in, out, nil)).To(Succeed())

			// --- Verify intermediate v1beta2 state ---
			g.Expect(out.EnablePortSecurity).To(Equal(tt.expectedEnablePS))

			// --- Convert back to v1beta1 ---
			restored := &ResolvedPortSpecFields{}
			g.Expect(Convert_v1beta2_ResolvedPortSpecFields_To_v1beta1_ResolvedPortSpecFields(out, restored, nil)).To(Succeed())

			// --- Verify full round-trip ---
			g.Expect(restored.DisablePortSecurity).To(Equal(tt.in.DisablePortSecurity))
		})
	}
}
