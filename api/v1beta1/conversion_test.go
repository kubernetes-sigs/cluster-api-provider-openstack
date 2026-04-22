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

func TestOpenStackClusterConversion(t *testing.T) {
	g := NewWithT(t)

	src := &OpenStackCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-cluster",
			Namespace:  "default",
			Generation: 3,
		},
		Spec: OpenStackClusterSpec{
			IdentityRef: OpenStackIdentityReference{
				Name:      "cloud-config",
				CloudName: "openstack",
			},
			ManagedSubnets: []SubnetSpec{
				{
					CIDR: "192.168.0.0/24",
				},
			},
			ManagedSecurityGroups: &ManagedSecurityGroups{},
			Bastion: &Bastion{
				Enabled: ptr.To(true),
				Spec: &OpenStackMachineSpec{
					Flavor: ptr.To("m1.small"),
				},
			},
		},
		Status: OpenStackClusterStatus{
			Ready: true,
			FailureDomains: clusterv1beta1.FailureDomains{
				"az-1": clusterv1beta1.FailureDomainSpec{
					ControlPlane: true,
					Attributes:   map[string]string{"region": "us-east-1"},
				},
				"az-2": clusterv1beta1.FailureDomainSpec{
					ControlPlane: false,
					Attributes:   map[string]string{"region": "us-west-1"},
				},
			},
			Conditions: clusterv1beta1.Conditions{
				{
					Type:               clusterv1beta1.ReadyCondition,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             "Ready",
					Message:            "Cluster is ready",
					Severity:           clusterv1beta1.ConditionSeverityInfo,
				},
				{
					Type:               "NetworkReady",
					Status:             corev1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             "NetworkReady",
					Message:            "Network is ready",
					Severity:           clusterv1beta1.ConditionSeverityInfo,
				},
			},
		},
	}

	// Convert to v1beta2
	dst := &infrav1.OpenStackCluster{}
	g.Expect(src.ConvertTo(dst)).To(Succeed())

	// Verify basic fields
	g.Expect(dst.Name).To(Equal("test-cluster"))
	g.Expect(dst.Namespace).To(Equal("default"))
	g.Expect(dst.Spec.IdentityRef.Name).To(Equal("cloud-config"))
	g.Expect(dst.Spec.ManagedSubnets).To(HaveLen(1))
	g.Expect(dst.Spec.ManagedNetwork).To(BeNil())

	// Verify flavor mapping (name -> FlavorParam.Filter.Name)
	g.Expect(dst.Spec.Bastion.Spec.Flavor.ID).To(BeNil())
	g.Expect(dst.Spec.Bastion.Spec.Flavor.Filter).NotTo(BeNil())
	g.Expect(dst.Spec.Bastion.Spec.Flavor.Filter.Name).NotTo(BeNil())
	g.Expect(*dst.Spec.Bastion.Spec.Flavor.Filter.Name).To(Equal("m1.small"))

	// Verify FailureDomains converted from map to slice
	g.Expect(dst.Status.FailureDomains).To(HaveLen(2))
	for _, fd := range dst.Status.FailureDomains {
		switch fd.Name {
		case "az-1":
			g.Expect(ptr.Deref(fd.ControlPlane, false)).To(BeTrue())
			g.Expect(fd.Attributes).To(HaveKeyWithValue("region", "us-east-1"))
		case "az-2":
			g.Expect(ptr.Deref(fd.ControlPlane, false)).To(BeFalse())
			g.Expect(fd.Attributes).To(HaveKeyWithValue("region", "us-west-1"))
		default:
			t.Errorf("unexpected failure domain: %s", fd.Name)
		}
	}

	// Verify conditions converted
	g.Expect(dst.Status.Conditions).To(HaveLen(2))
	g.Expect(dst.Status.Conditions[0].Type).To(Equal("Ready"))
	g.Expect(dst.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
	g.Expect(dst.Status.Conditions[0].ObservedGeneration).To(Equal(int64(3)))
	g.Expect(dst.Status.Conditions[1].Type).To(Equal("NetworkReady"))

	// Convert back to v1beta1
	restored := &OpenStackCluster{}
	g.Expect(restored.ConvertFrom(dst)).To(Succeed())

	// Verify round-trip
	g.Expect(restored.Name).To(Equal(src.Name))
	g.Expect(restored.Spec.IdentityRef).To(Equal(src.Spec.IdentityRef))
	g.Expect(restored.Status.Ready).To(BeTrue())
	g.Expect(restored.Status.Conditions).To(HaveLen(2))
	g.Expect(restored.Spec.NetworkMTU).To(BeNil())
	g.Expect(restored.Spec.DisablePortSecurity).To(BeNil())

	// Severity is lost during conversion, so it won't match exactly
	g.Expect(restored.Status.Conditions[0].Type).To(Equal(src.Status.Conditions[0].Type))
	g.Expect(restored.Status.Conditions[0].Status).To(Equal(src.Status.Conditions[0].Status))

	// Verify FailureDomains round-trip (slice back to map)
	g.Expect(restored.Status.FailureDomains).To(HaveLen(2))
	g.Expect(restored.Status.FailureDomains).To(HaveKey("az-1"))
	g.Expect(restored.Status.FailureDomains).To(HaveKey("az-2"))
	g.Expect(restored.Status.FailureDomains["az-1"].ControlPlane).To(BeTrue())
	g.Expect(restored.Status.FailureDomains["az-1"].Attributes).To(HaveKeyWithValue("region", "us-east-1"))
	g.Expect(restored.Status.FailureDomains["az-2"].ControlPlane).To(BeFalse())
	g.Expect(restored.Status.FailureDomains["az-2"].Attributes).To(HaveKeyWithValue("region", "us-west-1"))
}

// TestOpenStackMachineConversion_FlavorIDTakesPrecedence verifies that when
// both Flavor (name) and FlavorID are set on a v1beta1 object, FlavorID wins
// on upgrade to v1beta2.
//
// On the round-trip back to v1beta1, CAPI's restore annotation mechanism
// preserves the original Flavor name alongside the restored FlavorID, so both
// fields are non-nil after the round-trip.
func TestOpenStackMachineConversion_FlavorIDTakesPrecedence(t *testing.T) {
	g := NewWithT(t)

	src := &OpenStackMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-machine",
		},
		Spec: OpenStackMachineSpec{
			// Both set — FlavorID should win on upgrade.
			Flavor:   ptr.To("m1.small"),
			FlavorID: ptr.To("uuid-456"),
			Image: ImageParam{
				Filter: &ImageFilter{
					Name: ptr.To("ubuntu-22.04"),
				},
			},
		},
	}

	dst := &infrav1.OpenStackMachine{}
	g.Expect(src.ConvertTo(dst)).To(Succeed())

	// FlavorID takes precedence: ID must be set, Filter must be nil.
	g.Expect(dst.Spec.Flavor.ID).NotTo(BeNil())
	g.Expect(*dst.Spec.Flavor.ID).To(Equal("uuid-456"))
	g.Expect(dst.Spec.Flavor.Filter).To(BeNil())

	// Round-trip back: FlavorID is restored from the hub value.
	// The restore annotation also brings back the original Flavor name, so
	// both fields will be non-nil — this is expected CAPI behaviour.
	restored := &OpenStackMachine{}
	g.Expect(restored.ConvertFrom(dst)).To(Succeed())

	g.Expect(restored.Spec.FlavorID).To(Equal(ptr.To("uuid-456")))
	// Flavor (name) is restored via annotation — it is NOT lost.
	g.Expect(restored.Spec.Flavor).To(Equal(ptr.To("m1.small")))
}

// TestOpenStackMachineConversion_NeitherFlavorNorFlavorID verifies that
// a v1beta1 object with neither Flavor nor FlavorID set is handled
// gracefully during conversion. This can happen when the apiserver sends
// objects without a spec (e.g. in the context of managedField conversion).
func TestOpenStackMachineConversion_NeitherFlavorNorFlavorID(t *testing.T) {
	g := NewWithT(t)

	src := &OpenStackMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-machine",
		},
		Spec: OpenStackMachineSpec{
			Image: ImageParam{
				Filter: &ImageFilter{
					Name: ptr.To("ubuntu-22.04"),
				},
			},
		},
	}

	dst := &infrav1.OpenStackMachine{}
	g.Expect(src.ConvertTo(dst)).To(Succeed())

	// Neither Flavor nor FlavorID is set: the resulting FlavorParam is
	// zero-valued. API validation (not conversion) is responsible for
	// rejecting objects without a flavor.
	g.Expect(dst.Spec.Flavor).To(Equal(infrav1.FlavorParam{}))
}

func TestOpenStackMachineConversion_FlavorName(t *testing.T) {
	g := NewWithT(t)

	src := &OpenStackMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-machine",
			Namespace:  "default",
			Generation: 5,
		},
		Spec: OpenStackMachineSpec{
			Flavor:     ptr.To("m1.small"),
			SSHKeyName: "test-key",
			Image: ImageParam{
				Filter: &ImageFilter{
					Name: ptr.To("ubuntu-22.04"),
				},
			},
		},
		Status: OpenStackMachineStatus{
			Ready: true,
			Initialization: &MachineInitialization{
				Provisioned: true,
			},
			InstanceID: ptr.To("instance-12345"),
			Conditions: clusterv1beta1.Conditions{
				{
					Type:               clusterv1beta1.ReadyCondition,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             "Ready",
					Severity:           clusterv1beta1.ConditionSeverityInfo,
				},
			},
		},
	}

	// Convert to v1beta2
	dst := &infrav1.OpenStackMachine{}
	g.Expect(src.ConvertTo(dst)).To(Succeed())

	// Verify basic fields
	g.Expect(dst.Name).To(Equal("test-machine"))
	g.Expect(dst.Spec.SSHKeyName).To(Equal("test-key"))
	g.Expect(ptr.Deref((*string)(dst.Spec.Image.Filter.Name), "")).To(Equal("ubuntu-22.04"))

	// Verify flavor mapping (name -> FlavorParam.Filter.Name)
	g.Expect(dst.Spec.Flavor.ID).To(BeNil())
	g.Expect(dst.Spec.Flavor.Filter).NotTo(BeNil())
	g.Expect(dst.Spec.Flavor.Filter.Name).NotTo(BeNil())
	g.Expect(*dst.Spec.Flavor.Filter.Name).To(Equal("m1.small"))

	// Verify status fields including Initialization and InstanceID
	g.Expect(dst.Status.Initialization).NotTo(BeNil())
	g.Expect(dst.Status.Initialization.Provisioned).To(BeTrue())
	g.Expect(*dst.Status.InstanceID).To(Equal("instance-12345"))

	// Verify conditions
	g.Expect(dst.Status.Conditions).To(HaveLen(1))
	g.Expect(dst.Status.Conditions[0].Type).To(Equal("Ready"))
	g.Expect(dst.Status.Conditions[0].ObservedGeneration).To(Equal(int64(5)))

	// Convert back
	restored := &OpenStackMachine{}
	g.Expect(restored.ConvertFrom(dst)).To(Succeed())

	// Verify round-trip
	g.Expect(restored.Name).To(Equal(src.Name))
	g.Expect(restored.Spec.Flavor).To(Equal(src.Spec.Flavor))
	g.Expect(restored.Spec.FlavorID).To(BeNil())
	g.Expect(restored.Spec.SSHKeyName).To(Equal("test-key"))
	g.Expect(restored.Status.Ready).To(BeTrue())
	g.Expect(restored.Status.Initialization).NotTo(BeNil())
	g.Expect(restored.Status.Initialization.Provisioned).To(BeTrue())
	g.Expect(*restored.Status.InstanceID).To(Equal("instance-12345"))
}

func TestOpenStackMachineConversion_FlavorID(t *testing.T) {
	g := NewWithT(t)

	src := &OpenStackMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-machine",
		},
		Spec: OpenStackMachineSpec{
			FlavorID:   ptr.To("uuid-123"),
			SSHKeyName: "test-key",
			Image: ImageParam{
				Filter: &ImageFilter{
					Name: ptr.To("ubuntu-22.04"),
				},
			},
		},
	}

	dst := &infrav1.OpenStackMachine{}
	g.Expect(src.ConvertTo(dst)).To(Succeed())

	// Expect ID chosen, Filter nil
	g.Expect(dst.Spec.Flavor.ID).NotTo(BeNil())
	g.Expect(*dst.Spec.Flavor.ID).To(Equal("uuid-123"))
	g.Expect(dst.Spec.Flavor.Filter).To(BeNil())

	// Round-trip back: expect FlavorID set, Flavor nil
	restored := &OpenStackMachine{}
	g.Expect(restored.ConvertFrom(dst)).To(Succeed())

	g.Expect(restored.Spec.FlavorID).To(Equal(src.Spec.FlavorID))
	g.Expect(restored.Spec.Flavor).To(BeNil())
}

func TestOpenStackClusterTemplateConversion(t *testing.T) {
	g := NewWithT(t)

	src := &OpenStackClusterTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-template",
			Namespace: "default",
		},
		Spec: OpenStackClusterTemplateSpec{
			Template: OpenStackClusterTemplateResource{
				Spec: OpenStackClusterSpec{
					IdentityRef: OpenStackIdentityReference{
						Name:      "cloud-config",
						CloudName: "openstack",
					},
					ManagedSubnets: []SubnetSpec{
						{
							CIDR: "10.0.0.0/16",
						},
					},
					Bastion: &Bastion{
						Enabled: ptr.To(true),
						Spec: &OpenStackMachineSpec{
							Flavor: ptr.To("m1.small"),
						},
					},
				},
			},
		},
	}

	// Convert to v1beta2
	dst := &infrav1.OpenStackClusterTemplate{}
	g.Expect(src.ConvertTo(dst)).To(Succeed())

	// Verify template spec
	g.Expect(dst.Name).To(Equal("test-template"))
	g.Expect(dst.Spec.Template.Spec.IdentityRef.Name).To(Equal("cloud-config"))
	g.Expect(dst.Spec.Template.Spec.ManagedSubnets).To(HaveLen(1))

	// Verify flavor mapping (name -> FlavorParam.Filter.Name)
	g.Expect(dst.Spec.Template.Spec.Bastion.Spec.Flavor.ID).To(BeNil())
	g.Expect(dst.Spec.Template.Spec.Bastion.Spec.Flavor.Filter).NotTo(BeNil())
	g.Expect(dst.Spec.Template.Spec.Bastion.Spec.Flavor.Filter.Name).NotTo(BeNil())
	g.Expect(*dst.Spec.Template.Spec.Bastion.Spec.Flavor.Filter.Name).To(Equal("m1.small"))

	// Convert back
	restored := &OpenStackClusterTemplate{}
	g.Expect(restored.ConvertFrom(dst)).To(Succeed())

	// Verify round-trip
	g.Expect(restored.Name).To(Equal(src.Name))
	g.Expect(restored.Spec.Template.Spec.IdentityRef).To(Equal(src.Spec.Template.Spec.IdentityRef))
}

func TestOpenStackMachineTemplateConversion(t *testing.T) {
	g := NewWithT(t)

	src := &OpenStackMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine-template",
			Namespace: "default",
		},
		Spec: OpenStackMachineTemplateSpec{
			Template: OpenStackMachineTemplateResource{
				Spec: OpenStackMachineSpec{
					Flavor: ptr.To("m1.small"),
					Image: ImageParam{
						Filter: &ImageFilter{
							Name: ptr.To("ubuntu-22.04"),
						},
					},
				},
			},
		},
	}

	// Convert to v1beta2
	dst := &infrav1.OpenStackMachineTemplate{}
	g.Expect(src.ConvertTo(dst)).To(Succeed())

	// Verify template spec
	g.Expect(dst.Name).To(Equal("test-machine-template"))

	// Verify flavor mapping (name -> FlavorParam.Filter.Name)
	g.Expect(dst.Spec.Template.Spec.Flavor.ID).To(BeNil())
	g.Expect(dst.Spec.Template.Spec.Flavor.Filter).NotTo(BeNil())
	g.Expect(dst.Spec.Template.Spec.Flavor.Filter.Name).NotTo(BeNil())
	g.Expect(*dst.Spec.Template.Spec.Flavor.Filter.Name).To(Equal("m1.small"))

	// Convert back
	restored := &OpenStackMachineTemplate{}
	g.Expect(restored.ConvertFrom(dst)).To(Succeed())

	// Verify round-trip
	g.Expect(restored.Name).To(Equal(src.Name))
	g.Expect(restored.Spec.Template.Spec.Flavor).To(Equal(src.Spec.Template.Spec.Flavor))
}

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
	g.Expect(src.ConvertTo(dst)).To(Succeed())

	// Verify items converted
	g.Expect(dst.Items).To(HaveLen(2))
	g.Expect(dst.Items[0].Name).To(Equal("cluster-1"))
	g.Expect(dst.Items[1].Name).To(Equal("cluster-2"))

	// Convert back
	restored := &OpenStackClusterList{}
	g.Expect(restored.ConvertFrom(dst)).To(Succeed())

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
	g.Expect(src.ConvertTo(dst)).To(Succeed())

	g.Expect(dst.Items).To(HaveLen(2))
	g.Expect(dst.Items[0].Name).To(Equal("machine-1"))
	g.Expect(dst.Items[0].Spec.Flavor.Filter.Name).To(Equal(optional.String(ptr.To("m1.small"))))
	g.Expect(dst.Items[1].Name).To(Equal("machine-2"))
	g.Expect(dst.Items[1].Spec.Flavor.Filter.Name).To(Equal(optional.String(ptr.To("m1.large"))))

	// Convert back
	restored := &OpenStackMachineList{}
	g.Expect(restored.ConvertFrom(dst)).To(Succeed())

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
	g.Expect(src.ConvertTo(dst)).To(Succeed())

	g.Expect(dst.Items).To(HaveLen(1))
	g.Expect(dst.Items[0].Name).To(Equal("template-1"))
	g.Expect(dst.Items[0].Spec.Template.Spec.IdentityRef.Name).To(Equal("cloud-1"))

	// Convert back
	restored := &OpenStackClusterTemplateList{}
	g.Expect(restored.ConvertFrom(dst)).To(Succeed())

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
	g.Expect(src.ConvertTo(dst)).To(Succeed())

	g.Expect(dst.Items).To(HaveLen(1))
	g.Expect(dst.Items[0].Name).To(Equal("mt-1"))
	g.Expect(dst.Items[0].Spec.Template.Spec.Flavor.Filter.Name).To(Equal(optional.String(ptr.To("m1.xlarge"))))
	// Convert back
	restored := &OpenStackMachineTemplateList{}
	g.Expect(restored.ConvertFrom(dst)).To(Succeed())

	g.Expect(restored.Items).To(HaveLen(1))
	g.Expect(restored.Items[0].Name).To(Equal("mt-1"))
}

func TestFailureDomainsConversionEdgeCases(t *testing.T) {
	g := NewWithT(t)

	// Empty FailureDomains map should result in nil slice
	src := &OpenStackCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: OpenStackClusterSpec{
			IdentityRef: OpenStackIdentityReference{
				Name:      "cloud-config",
				CloudName: "openstack",
			},
		},
		Status: OpenStackClusterStatus{
			FailureDomains: clusterv1beta1.FailureDomains{},
		},
	}

	dst := &infrav1.OpenStackCluster{}
	g.Expect(src.ConvertTo(dst)).To(Succeed())
	g.Expect(dst.Status.FailureDomains).To(BeEmpty())

	// Nil FailureDomains map
	src.Status.FailureDomains = nil
	dst = &infrav1.OpenStackCluster{}
	g.Expect(src.ConvertTo(dst)).To(Succeed())
	g.Expect(dst.Status.FailureDomains).To(BeNil())

	// Single FailureDomain
	src.Status.FailureDomains = clusterv1beta1.FailureDomains{
		"az-only": clusterv1beta1.FailureDomainSpec{
			ControlPlane: true,
		},
	}
	dst = &infrav1.OpenStackCluster{}
	g.Expect(src.ConvertTo(dst)).To(Succeed())
	g.Expect(dst.Status.FailureDomains).To(HaveLen(1))
	g.Expect(dst.Status.FailureDomains[0].Name).To(Equal("az-only"))
	g.Expect(ptr.Deref(dst.Status.FailureDomains[0].ControlPlane, false)).To(BeTrue())

	// Round-trip single domain back
	restored := &OpenStackCluster{}
	g.Expect(restored.ConvertFrom(dst)).To(Succeed())
	g.Expect(restored.Status.FailureDomains).To(HaveLen(1))
	g.Expect(restored.Status.FailureDomains).To(HaveKey("az-only"))
	g.Expect(restored.Status.FailureDomains["az-only"].ControlPlane).To(BeTrue())

	// FailureDomain with nil Attributes
	src.Status.FailureDomains = clusterv1beta1.FailureDomains{
		"az-no-attrs": clusterv1beta1.FailureDomainSpec{
			ControlPlane: false,
			Attributes:   nil,
		},
	}
	dst = &infrav1.OpenStackCluster{}
	g.Expect(src.ConvertTo(dst)).To(Succeed())
	g.Expect(dst.Status.FailureDomains[0].Attributes).To(BeNil())

	// Verify sorting: multiple domains come out sorted by name
	src.Status.FailureDomains = clusterv1beta1.FailureDomains{
		"zone-c": clusterv1beta1.FailureDomainSpec{ControlPlane: false},
		"zone-a": clusterv1beta1.FailureDomainSpec{ControlPlane: true},
		"zone-b": clusterv1beta1.FailureDomainSpec{ControlPlane: false},
	}
	dst = &infrav1.OpenStackCluster{}
	g.Expect(src.ConvertTo(dst)).To(Succeed())
	g.Expect(dst.Status.FailureDomains).To(HaveLen(3))
	g.Expect(dst.Status.FailureDomains[0].Name).To(Equal("zone-a"))
	g.Expect(dst.Status.FailureDomains[1].Name).To(Equal("zone-b"))
	g.Expect(dst.Status.FailureDomains[2].Name).To(Equal("zone-c"))
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

func TestReadyFlagFromConditions(t *testing.T) {
	g := NewWithT(t)

	// v1beta2 → v1beta1: Ready flag should be derived from conditions
	v1beta2Cluster := &infrav1.OpenStackCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: infrav1.OpenStackClusterSpec{
			IdentityRef: infrav1.OpenStackIdentityReference{
				Name:      "cloud-config",
				CloudName: "openstack",
			},
		},
		Status: infrav1.OpenStackClusterStatus{
			Conditions: []metav1.Condition{
				{
					Type:   "Ready",
					Status: metav1.ConditionTrue,
					Reason: "AllReady",
				},
			},
		},
	}

	restored := &OpenStackCluster{}
	g.Expect(restored.ConvertFrom(v1beta2Cluster)).To(Succeed())
	g.Expect(restored.Status.Ready).To(BeTrue())

	// Not ready
	v1beta2Cluster.Status.Conditions[0].Status = metav1.ConditionFalse
	restored = &OpenStackCluster{}
	g.Expect(restored.ConvertFrom(v1beta2Cluster)).To(Succeed())
	g.Expect(restored.Status.Ready).To(BeFalse())

	// No conditions at all
	v1beta2Cluster.Status.Conditions = nil
	restored = &OpenStackCluster{}
	g.Expect(restored.ConvertFrom(v1beta2Cluster)).To(Succeed())
	g.Expect(restored.Status.Ready).To(BeFalse())

	// Same for OpenStackMachine
	v1beta2Machine := &infrav1.OpenStackMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine",
			Namespace: "default",
		},
		Spec: infrav1.OpenStackMachineSpec{
			Flavor: infrav1.FlavorParam{
				Filter: &infrav1.FlavorFilter{
					Name: ptr.To("m1.small"),
				},
			},
			Image: infrav1.ImageParam{
				Filter: &infrav1.ImageFilter{
					Name: ptr.To("ubuntu"),
				},
			},
		},
		Status: infrav1.OpenStackMachineStatus{
			Conditions: []metav1.Condition{
				{
					Type:   "Ready",
					Status: metav1.ConditionTrue,
					Reason: "Ready",
				},
			},
		},
	}

	restoredMachine := &OpenStackMachine{}
	g.Expect(restoredMachine.ConvertFrom(v1beta2Machine)).To(Succeed())
	g.Expect(restoredMachine.Status.Ready).To(BeTrue())
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

func TestOpenStackCluster_RoundTrip_ManagedNetwork(t *testing.T) {
	mtu := optional.Int(ptr.To(1500))
	disablePS := optional.Bool(ptr.To(true))

	tests := []struct {
		name string
		in   OpenStackCluster
	}{
		{
			name: "both fields set",
			in: OpenStackCluster{
				Spec: OpenStackClusterSpec{
					NetworkMTU:          mtu,
					DisablePortSecurity: disablePS,
				},
			},
		},
		{
			name: "only MTU set",
			in: OpenStackCluster{
				Spec: OpenStackClusterSpec{NetworkMTU: mtu},
			},
		},
		{
			name: "only DisablePortSecurity set",
			in: OpenStackCluster{
				Spec: OpenStackClusterSpec{DisablePortSecurity: disablePS},
			},
		},
		{
			name: "neither set — ManagedNetwork stays nil",
			in:   OpenStackCluster{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			hub := &infrav1.OpenStackCluster{}
			g.Expect(tt.in.ConvertTo(hub)).To(Succeed())

			// Verify intermediate v1beta2 state
			if tt.in.Spec.NetworkMTU == nil && tt.in.Spec.DisablePortSecurity == nil {
				g.Expect(hub.Spec.ManagedNetwork).To(BeNil())
			} else {
				g.Expect(hub.Spec.ManagedNetwork).NotTo(BeNil())
				g.Expect(hub.Spec.ManagedNetwork.MTU).To(Equal(tt.in.Spec.NetworkMTU))
				g.Expect(hub.Spec.ManagedNetwork.DisablePortSecurity).To(Equal(tt.in.Spec.DisablePortSecurity))
			}

			restored := &OpenStackCluster{}
			g.Expect(restored.ConvertFrom(hub)).To(Succeed())

			// Verify final v1beta1 state
			g.Expect(restored.Spec.NetworkMTU).To(Equal(tt.in.Spec.NetworkMTU))
			g.Expect(restored.Spec.DisablePortSecurity).To(Equal(tt.in.Spec.DisablePortSecurity))
		})
	}
}

func TestOpenStackCluster_RoundTrip_ManagedRouter(t *testing.T) {
	tests := []struct {
		name string
		in   OpenStackCluster
	}{
		{
			name: "single IP with subnet filter name",
			in: OpenStackCluster{
				Spec: OpenStackClusterSpec{
					ExternalRouterIPs: []ExternalRouterIPParam{
						{
							FixedIP: "10.0.0.1",
							Subnet: SubnetParam{
								Filter: &SubnetFilter{Name: "my-subnet"},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple IPs",
			in: OpenStackCluster{
				Spec: OpenStackClusterSpec{
					ExternalRouterIPs: []ExternalRouterIPParam{
						{
							FixedIP: "10.0.0.1",
							Subnet:  SubnetParam{Filter: &SubnetFilter{Name: "subnet-a"}},
						},
						{
							FixedIP: "10.0.0.2",
							Subnet:  SubnetParam{Filter: &SubnetFilter{Name: "subnet-b"}},
						},
					},
				},
			},
		},
		{
			name: "subnet by ID",
			in: OpenStackCluster{
				Spec: OpenStackClusterSpec{
					ExternalRouterIPs: []ExternalRouterIPParam{
						{
							FixedIP: "10.0.0.1",
							Subnet:  SubnetParam{ID: optional.String(ptr.To("some-uuid"))},
						},
					},
				},
			},
		},
		{
			name: "no ExternalRouterIPs — ManagedRouter stays nil",
			in:   OpenStackCluster{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			hub := &infrav1.OpenStackCluster{}
			g.Expect(tt.in.ConvertTo(hub)).To(Succeed())

			// Verify intermediate v1beta2 state
			if len(tt.in.Spec.ExternalRouterIPs) == 0 {
				g.Expect(hub.Spec.ManagedRouter).To(BeNil())
			} else {
				g.Expect(hub.Spec.ManagedRouter).NotTo(BeNil())
				g.Expect(hub.Spec.ManagedRouter.ExternalIPs).To(HaveLen(len(tt.in.Spec.ExternalRouterIPs)))
				for i, rip := range tt.in.Spec.ExternalRouterIPs {
					g.Expect(hub.Spec.ManagedRouter.ExternalIPs[i].FixedIP).To(Equal(rip.FixedIP))
				}
			}

			restored := &OpenStackCluster{}
			g.Expect(restored.ConvertFrom(hub)).To(Succeed())

			// Verify full round-trip
			g.Expect(restored.Spec.ExternalRouterIPs).To(Equal(tt.in.Spec.ExternalRouterIPs))
		})
	}
}
