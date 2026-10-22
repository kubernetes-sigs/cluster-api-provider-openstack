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

package conversion

import (
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"

	infrav1beta1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/optional"
)

//nolint:staticcheck // Verify conversion of the deprecated v1beta1 Ready field.
func TestOpenStackClusterConversion(t *testing.T) {
	g := NewWithT(t)

	src := &infrav1beta1.OpenStackCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-cluster",
			Namespace:  "default",
			Generation: 3,
		},
		Spec: infrav1beta1.OpenStackClusterSpec{
			IdentityRef: infrav1beta1.OpenStackIdentityReference{
				Name:      "cloud-config",
				CloudName: "openstack",
			},
			ManagedSubnets: []infrav1beta1.SubnetSpec{
				{
					CIDR: "192.168.0.0/24",
				},
			},
			ManagedSecurityGroups: &infrav1beta1.ManagedSecurityGroups{},
			Bastion: &infrav1beta1.Bastion{
				Enabled: ptr.To(true),
				Spec: &infrav1beta1.OpenStackMachineSpec{
					Flavor: ptr.To("m1.small"),
				},
			},
		},
		Status: infrav1beta1.OpenStackClusterStatus{
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
	g.Expect(ConvertOpenStackClusterV1Beta1ToHub(t.Context(), src, dst)).To(Succeed())

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
	restored := &infrav1beta1.OpenStackCluster{}
	g.Expect(ConvertOpenStackClusterHubToV1Beta1(t.Context(), dst, restored)).To(Succeed())

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

	src := &infrav1beta1.OpenStackMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-machine",
		},
		Spec: infrav1beta1.OpenStackMachineSpec{
			// Both set — FlavorID should win on upgrade.
			Flavor:   ptr.To("m1.small"),
			FlavorID: ptr.To("uuid-456"),
			Image: infrav1beta1.ImageParam{
				Filter: &infrav1beta1.ImageFilter{
					Name: ptr.To("ubuntu-22.04"),
				},
			},
		},
	}

	dst := &infrav1.OpenStackMachine{}
	g.Expect(ConvertOpenStackMachineV1Beta1ToHub(t.Context(), src, dst)).To(Succeed())

	// FlavorID takes precedence: ID must be set, Filter must be nil.
	g.Expect(dst.Spec.Flavor.ID).NotTo(BeNil())
	g.Expect(*dst.Spec.Flavor.ID).To(Equal("uuid-456"))
	g.Expect(dst.Spec.Flavor.Filter).To(BeNil())

	// Round-trip back: FlavorID is restored from the hub value.
	// The restore annotation also brings back the original Flavor name, so
	// both fields will be non-nil — this is expected CAPI behaviour.
	restored := &infrav1beta1.OpenStackMachine{}
	g.Expect(ConvertOpenStackMachineHubToV1Beta1(t.Context(), dst, restored)).To(Succeed())

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

	src := &infrav1beta1.OpenStackMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-machine",
		},
		Spec: infrav1beta1.OpenStackMachineSpec{
			Image: infrav1beta1.ImageParam{
				Filter: &infrav1beta1.ImageFilter{
					Name: ptr.To("ubuntu-22.04"),
				},
			},
		},
	}

	dst := &infrav1.OpenStackMachine{}
	g.Expect(ConvertOpenStackMachineV1Beta1ToHub(t.Context(), src, dst)).To(Succeed())

	// Neither Flavor nor FlavorID is set: the resulting FlavorParam is
	// zero-valued. API validation (not conversion) is responsible for
	// rejecting objects without a flavor.
	g.Expect(dst.Spec.Flavor).To(Equal(infrav1.FlavorParam{}))
}

//nolint:staticcheck // Verify conversion of the deprecated v1beta1 Ready field.
func TestOpenStackMachineConversion_FlavorName(t *testing.T) {
	g := NewWithT(t)

	src := &infrav1beta1.OpenStackMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-machine",
			Namespace:  "default",
			Generation: 5,
		},
		Spec: infrav1beta1.OpenStackMachineSpec{
			Flavor:     ptr.To("m1.small"),
			SSHKeyName: "test-key",
			Image: infrav1beta1.ImageParam{
				Filter: &infrav1beta1.ImageFilter{
					Name: ptr.To("ubuntu-22.04"),
				},
			},
		},
		Status: infrav1beta1.OpenStackMachineStatus{
			Ready: true,
			Initialization: &infrav1beta1.MachineInitialization{
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
	g.Expect(ConvertOpenStackMachineV1Beta1ToHub(t.Context(), src, dst)).To(Succeed())

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
	restored := &infrav1beta1.OpenStackMachine{}
	g.Expect(ConvertOpenStackMachineHubToV1Beta1(t.Context(), dst, restored)).To(Succeed())

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

	src := &infrav1beta1.OpenStackMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-machine",
		},
		Spec: infrav1beta1.OpenStackMachineSpec{
			FlavorID:   ptr.To("uuid-123"),
			SSHKeyName: "test-key",
			Image: infrav1beta1.ImageParam{
				Filter: &infrav1beta1.ImageFilter{
					Name: ptr.To("ubuntu-22.04"),
				},
			},
		},
	}

	dst := &infrav1.OpenStackMachine{}
	g.Expect(ConvertOpenStackMachineV1Beta1ToHub(t.Context(), src, dst)).To(Succeed())

	// Expect ID chosen, Filter nil
	g.Expect(dst.Spec.Flavor.ID).NotTo(BeNil())
	g.Expect(*dst.Spec.Flavor.ID).To(Equal("uuid-123"))
	g.Expect(dst.Spec.Flavor.Filter).To(BeNil())

	// Round-trip back: expect FlavorID set, Flavor nil
	restored := &infrav1beta1.OpenStackMachine{}
	g.Expect(ConvertOpenStackMachineHubToV1Beta1(t.Context(), dst, restored)).To(Succeed())

	g.Expect(restored.Spec.FlavorID).To(Equal(src.Spec.FlavorID))
	g.Expect(restored.Spec.Flavor).To(BeNil())
}

func TestOpenStackClusterTemplateConversion(t *testing.T) {
	g := NewWithT(t)

	src := &infrav1beta1.OpenStackClusterTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-template",
			Namespace: "default",
		},
		Spec: infrav1beta1.OpenStackClusterTemplateSpec{
			Template: infrav1beta1.OpenStackClusterTemplateResource{
				Spec: infrav1beta1.OpenStackClusterSpec{
					IdentityRef: infrav1beta1.OpenStackIdentityReference{
						Name:      "cloud-config",
						CloudName: "openstack",
					},
					ManagedSubnets: []infrav1beta1.SubnetSpec{
						{
							CIDR: "10.0.0.0/16",
						},
					},
					Bastion: &infrav1beta1.Bastion{
						Enabled: ptr.To(true),
						Spec: &infrav1beta1.OpenStackMachineSpec{
							Flavor: ptr.To("m1.small"),
						},
					},
				},
			},
		},
	}

	// Convert to v1beta2
	dst := &infrav1.OpenStackClusterTemplate{}
	g.Expect(ConvertOpenStackClusterTemplateV1Beta1ToHub(t.Context(), src, dst)).To(Succeed())

	g.Expect(dst.Spec.Template.ObjectMeta).To(BeZero())

	g.Expect(dst.Name).To(Equal("test-template"))
	g.Expect(dst.Spec.Template.Spec.IdentityRef.Name).To(Equal("cloud-config"))
	g.Expect(dst.Spec.Template.Spec.ManagedSubnets).To(HaveLen(1))

	// Verify flavor mapping (name -> FlavorParam.Filter.Name)
	g.Expect(dst.Spec.Template.Spec.Bastion.Spec.Flavor.ID).To(BeNil())
	g.Expect(dst.Spec.Template.Spec.Bastion.Spec.Flavor.Filter).NotTo(BeNil())
	g.Expect(dst.Spec.Template.Spec.Bastion.Spec.Flavor.Filter.Name).NotTo(BeNil())
	g.Expect(*dst.Spec.Template.Spec.Bastion.Spec.Flavor.Filter.Name).To(Equal("m1.small"))

	// Conversion to the hub stashes the v1beta1 object so spoke-only fields
	// (e.g. flavor name) survive a later hub -> spoke conversion.
	g.Expect(dst.GetAnnotations()).To(HaveKey(utilconversion.DataAnnotation))

	// Convert back
	restored := &infrav1beta1.OpenStackClusterTemplate{}
	g.Expect(ConvertOpenStackClusterTemplateHubToV1Beta1(t.Context(), dst, restored)).To(Succeed())

	// Verify round-trip
	g.Expect(restored.Name).To(Equal(src.Name))
	g.Expect(restored.Spec.Template.Spec.IdentityRef).To(Equal(src.Spec.Template.Spec.IdentityRef))

	// The bastion flavor name is a spoke-only field preserved via the
	// conversion-data annotation; it must survive the round trip.
	g.Expect(restored.Spec.Template.Spec.Bastion).NotTo(BeNil())
	g.Expect(restored.Spec.Template.Spec.Bastion.Spec).NotTo(BeNil())
	g.Expect(restored.Spec.Template.Spec.Bastion.Spec.Flavor).To(Equal(ptr.To("m1.small")))
}

func TestOpenStackClusterTemplateObjectMetaConversion(t *testing.T) {
	g := NewWithT(t)

	hub := &infrav1.OpenStackClusterTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-template",
			Namespace: "default",
		},
		Spec: infrav1.OpenStackClusterTemplateSpec{
			Template: infrav1.OpenStackClusterTemplateResource{
				ObjectMeta: clusterv1.ObjectMeta{
					Labels: map[string]string{
						"cluster.x-k8s.io/cluster-name": "my-cluster",
					},
					Annotations: map[string]string{
						"example.com/annotation": "value",
					},
				},
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "cloud-config",
						CloudName: "openstack",
					},
				},
			},
		},
	}

	spoke := &infrav1beta1.OpenStackClusterTemplate{}
	g.Expect(ConvertOpenStackClusterTemplateHubToV1Beta1(t.Context(), hub, spoke)).To(Succeed())

	g.Expect(spoke.GetAnnotations()).To(HaveKey(utilconversion.DataAnnotation))

	g.Expect(spoke.Name).To(Equal("test-template"))
	g.Expect(spoke.Spec.Template.Spec.IdentityRef.Name).To(Equal("cloud-config"))

	restored := &infrav1.OpenStackClusterTemplate{}
	g.Expect(ConvertOpenStackClusterTemplateV1Beta1ToHub(t.Context(), spoke, restored)).To(Succeed())
	g.Expect(restored.Spec.Template.ObjectMeta).To(Equal(hub.Spec.Template.ObjectMeta))

	g.Expect(restored.GetAnnotations()).To(HaveKey(utilconversion.DataAnnotation))
}

func TestOpenStackClusterTemplateV1Beta1ToHubWithoutAnnotation(t *testing.T) {
	g := NewWithT(t)

	spoke := &infrav1beta1.OpenStackClusterTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fresh-template",
			Namespace: "default",
		},
		Spec: infrav1beta1.OpenStackClusterTemplateSpec{
			Template: infrav1beta1.OpenStackClusterTemplateResource{
				Spec: infrav1beta1.OpenStackClusterSpec{
					IdentityRef: infrav1beta1.OpenStackIdentityReference{
						Name:      "cloud-config",
						CloudName: "openstack",
					},
				},
			},
		},
	}

	hub := &infrav1.OpenStackClusterTemplate{}
	g.Expect(ConvertOpenStackClusterTemplateV1Beta1ToHub(t.Context(), spoke, hub)).To(Succeed())

	// No annotation means nothing to restore; template metadata stays zero.
	g.Expect(hub.Spec.Template.ObjectMeta).To(BeZero())
	g.Expect(hub.Spec.Template.ObjectMeta.Labels).To(BeEmpty())
	g.Expect(hub.Spec.Template.ObjectMeta.Annotations).To(BeEmpty())

	// The rest of the spec converts normally.
	g.Expect(hub.Name).To(Equal("fresh-template"))
	g.Expect(hub.Spec.Template.Spec.IdentityRef.Name).To(Equal("cloud-config"))
}

func TestOpenStackMachineTemplateConversion(t *testing.T) {
	g := NewWithT(t)

	src := &infrav1beta1.OpenStackMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine-template",
			Namespace: "default",
		},
		Spec: infrav1beta1.OpenStackMachineTemplateSpec{
			Template: infrav1beta1.OpenStackMachineTemplateResource{
				Spec: infrav1beta1.OpenStackMachineSpec{
					Flavor: ptr.To("m1.small"),
					Image: infrav1beta1.ImageParam{
						Filter: &infrav1beta1.ImageFilter{
							Name: ptr.To("ubuntu-22.04"),
						},
					},
				},
			},
		},
	}

	// Convert to v1beta2
	dst := &infrav1.OpenStackMachineTemplate{}
	g.Expect(ConvertOpenStackMachineTemplateV1Beta1ToHub(t.Context(), src, dst)).To(Succeed())

	// Verify template spec
	g.Expect(dst.Name).To(Equal("test-machine-template"))

	// Verify flavor mapping (name -> FlavorParam.Filter.Name)
	g.Expect(dst.Spec.Template.Spec.Flavor.ID).To(BeNil())
	g.Expect(dst.Spec.Template.Spec.Flavor.Filter).NotTo(BeNil())
	g.Expect(dst.Spec.Template.Spec.Flavor.Filter.Name).NotTo(BeNil())
	g.Expect(*dst.Spec.Template.Spec.Flavor.Filter.Name).To(Equal("m1.small"))

	// Convert back
	restored := &infrav1beta1.OpenStackMachineTemplate{}
	g.Expect(ConvertOpenStackMachineTemplateHubToV1Beta1(t.Context(), dst, restored)).To(Succeed())

	// Verify round-trip
	g.Expect(restored.Name).To(Equal(src.Name))
	g.Expect(restored.Spec.Template.Spec.Flavor).To(Equal(src.Spec.Template.Spec.Flavor))
}

func TestFailureDomainsConversionEdgeCases(t *testing.T) {
	g := NewWithT(t)

	// Empty FailureDomains map should result in nil slice
	src := &infrav1beta1.OpenStackCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: infrav1beta1.OpenStackClusterSpec{
			IdentityRef: infrav1beta1.OpenStackIdentityReference{
				Name:      "cloud-config",
				CloudName: "openstack",
			},
		},
		Status: infrav1beta1.OpenStackClusterStatus{
			FailureDomains: clusterv1beta1.FailureDomains{},
		},
	}

	dst := &infrav1.OpenStackCluster{}
	g.Expect(ConvertOpenStackClusterV1Beta1ToHub(t.Context(), src, dst)).To(Succeed())
	g.Expect(dst.Status.FailureDomains).To(BeEmpty())

	// Nil FailureDomains map
	src.Status.FailureDomains = nil
	dst = &infrav1.OpenStackCluster{}
	g.Expect(ConvertOpenStackClusterV1Beta1ToHub(t.Context(), src, dst)).To(Succeed())
	g.Expect(dst.Status.FailureDomains).To(BeNil())

	// Single FailureDomain
	src.Status.FailureDomains = clusterv1beta1.FailureDomains{
		"az-only": clusterv1beta1.FailureDomainSpec{
			ControlPlane: true,
		},
	}
	dst = &infrav1.OpenStackCluster{}
	g.Expect(ConvertOpenStackClusterV1Beta1ToHub(t.Context(), src, dst)).To(Succeed())
	g.Expect(dst.Status.FailureDomains).To(HaveLen(1))
	g.Expect(dst.Status.FailureDomains[0].Name).To(Equal("az-only"))
	g.Expect(ptr.Deref(dst.Status.FailureDomains[0].ControlPlane, false)).To(BeTrue())

	// Round-trip single domain back
	restored := &infrav1beta1.OpenStackCluster{}
	g.Expect(ConvertOpenStackClusterHubToV1Beta1(t.Context(), dst, restored)).To(Succeed())
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
	g.Expect(ConvertOpenStackClusterV1Beta1ToHub(t.Context(), src, dst)).To(Succeed())
	g.Expect(dst.Status.FailureDomains[0].Attributes).To(BeNil())

	// Verify sorting: multiple domains come out sorted by name
	src.Status.FailureDomains = clusterv1beta1.FailureDomains{
		"zone-c": clusterv1beta1.FailureDomainSpec{ControlPlane: false},
		"zone-a": clusterv1beta1.FailureDomainSpec{ControlPlane: true},
		"zone-b": clusterv1beta1.FailureDomainSpec{ControlPlane: false},
	}
	dst = &infrav1.OpenStackCluster{}
	g.Expect(ConvertOpenStackClusterV1Beta1ToHub(t.Context(), src, dst)).To(Succeed())
	g.Expect(dst.Status.FailureDomains).To(HaveLen(3))
	g.Expect(dst.Status.FailureDomains[0].Name).To(Equal("zone-a"))
	g.Expect(dst.Status.FailureDomains[1].Name).To(Equal("zone-b"))
	g.Expect(dst.Status.FailureDomains[2].Name).To(Equal("zone-c"))
}

//nolint:staticcheck // Verify conversion of the deprecated v1beta1 Ready field.
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

	restored := &infrav1beta1.OpenStackCluster{}
	g.Expect(ConvertOpenStackClusterHubToV1Beta1(t.Context(), v1beta2Cluster, restored)).To(Succeed())
	g.Expect(restored.Status.Ready).To(BeTrue())

	// Not ready
	v1beta2Cluster.Status.Conditions[0].Status = metav1.ConditionFalse
	restored = &infrav1beta1.OpenStackCluster{}
	g.Expect(ConvertOpenStackClusterHubToV1Beta1(t.Context(), v1beta2Cluster, restored)).To(Succeed())
	g.Expect(restored.Status.Ready).To(BeFalse())

	// No conditions at all
	v1beta2Cluster.Status.Conditions = nil
	restored = &infrav1beta1.OpenStackCluster{}
	g.Expect(ConvertOpenStackClusterHubToV1Beta1(t.Context(), v1beta2Cluster, restored)).To(Succeed())
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

	restoredMachine := &infrav1beta1.OpenStackMachine{}
	g.Expect(ConvertOpenStackMachineHubToV1Beta1(t.Context(), v1beta2Machine, restoredMachine)).To(Succeed())
	g.Expect(restoredMachine.Status.Ready).To(BeTrue())
}

func TestOpenStackCluster_RoundTrip_ManagedNetwork(t *testing.T) {
	mtu := optional.Int(ptr.To(1500))
	disablePS := optional.Bool(ptr.To(true))

	tests := []struct {
		name             string
		in               infrav1beta1.OpenStackCluster
		expectedEnablePS optional.Bool
	}{
		{
			name: "both fields set",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{
					NetworkMTU:          mtu,
					DisablePortSecurity: disablePS,
				},
			},
			expectedEnablePS: optional.Bool(ptr.To(false)), // disablePS=true → enablePS=false
		},
		{
			name: "only MTU set",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{NetworkMTU: mtu},
			},
			expectedEnablePS: nil, // unset stays unset
		},
		{
			name: "only DisablePortSecurity set to true",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{DisablePortSecurity: disablePS},
			},
			expectedEnablePS: optional.Bool(ptr.To(false)), // disablePS=true → enablePS=false
		},
		{
			name: "DisablePortSecurity explicitly false",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{DisablePortSecurity: optional.Bool(ptr.To(false))},
			},
			expectedEnablePS: optional.Bool(ptr.To(true)), // disablePS=false → enablePS=true
		},
		{
			name:             "neither set — ManagedNetwork stays nil",
			in:               infrav1beta1.OpenStackCluster{},
			expectedEnablePS: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			hub := &infrav1.OpenStackCluster{}
			g.Expect(ConvertOpenStackClusterV1Beta1ToHub(t.Context(), &tt.in, hub)).To(Succeed())

			// Verify intermediate v1beta2 state
			if tt.in.Spec.NetworkMTU == nil && tt.in.Spec.DisablePortSecurity == nil {
				g.Expect(hub.Spec.ManagedNetwork).To(BeNil())
			} else {
				g.Expect(hub.Spec.ManagedNetwork).NotTo(BeNil())
				// hub.Spec.ManagedNetwork.MTU is *int32 while NetworkMTU is *int; compare values
				if tt.in.Spec.NetworkMTU != nil {
					g.Expect(hub.Spec.ManagedNetwork.MTU).NotTo(BeNil())
					g.Expect(int(*hub.Spec.ManagedNetwork.MTU)).To(Equal(*tt.in.Spec.NetworkMTU))
				} else {
					g.Expect(hub.Spec.ManagedNetwork.MTU).To(BeNil())
				}
				g.Expect(hub.Spec.ManagedNetwork.EnablePortSecurity).To(Equal(tt.expectedEnablePS))
			}

			restored := &infrav1beta1.OpenStackCluster{}
			g.Expect(ConvertOpenStackClusterHubToV1Beta1(t.Context(), hub, restored)).To(Succeed())

			// Verify final v1beta1 state round-trips correctly
			g.Expect(restored.Spec.NetworkMTU).To(Equal(tt.in.Spec.NetworkMTU))
			g.Expect(restored.Spec.DisablePortSecurity).To(Equal(tt.in.Spec.DisablePortSecurity))
		})
	}
}

func TestOpenStackCluster_RoundTrip_ManagedRouter(t *testing.T) {
	tests := []struct {
		name string
		in   infrav1beta1.OpenStackCluster
	}{
		{
			name: "single IP with subnet filter name",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{
					ExternalRouterIPs: []infrav1beta1.ExternalRouterIPParam{
						{
							FixedIP: "10.0.0.1",
							Subnet: infrav1beta1.SubnetParam{
								Filter: &infrav1beta1.SubnetFilter{Name: "my-subnet"},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple IPs",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{
					ExternalRouterIPs: []infrav1beta1.ExternalRouterIPParam{
						{
							FixedIP: "10.0.0.1",
							Subnet:  infrav1beta1.SubnetParam{Filter: &infrav1beta1.SubnetFilter{Name: "subnet-a"}},
						},
						{
							FixedIP: "10.0.0.2",
							Subnet:  infrav1beta1.SubnetParam{Filter: &infrav1beta1.SubnetFilter{Name: "subnet-b"}},
						},
					},
				},
			},
		},
		{
			name: "subnet by ID",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{
					ExternalRouterIPs: []infrav1beta1.ExternalRouterIPParam{
						{
							FixedIP: "10.0.0.1",
							Subnet:  infrav1beta1.SubnetParam{ID: optional.String(ptr.To("some-uuid"))},
						},
					},
				},
			},
		},
		{
			name: "no ExternalRouterIPs — ManagedRouter stays nil",
			in:   infrav1beta1.OpenStackCluster{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			hub := &infrav1.OpenStackCluster{}
			g.Expect(ConvertOpenStackClusterV1Beta1ToHub(t.Context(), &tt.in, hub)).To(Succeed())

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

			restored := &infrav1beta1.OpenStackCluster{}
			g.Expect(ConvertOpenStackClusterHubToV1Beta1(t.Context(), hub, restored)).To(Succeed())

			// Verify full round-trip
			g.Expect(restored.Spec.ExternalRouterIPs).To(Equal(tt.in.Spec.ExternalRouterIPs))
		})
	}
}

func TestOpenStackCluster_RoundTrip_ManagedSecurityGroups_ClusterNodesRules(t *testing.T) {
	tests := []struct {
		name string
		in   infrav1beta1.OpenStackCluster
	}{
		{
			name: "single rule",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{
					ManagedSecurityGroups: &infrav1beta1.ManagedSecurityGroups{
						AllNodesSecurityGroupRules: []infrav1beta1.SecurityGroupRuleSpec{
							{
								Name:      "allow-http",
								Direction: "ingress",
							},
						},
					},
				},
			},
		},
		{
			name: "multiple rules",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{
					ManagedSecurityGroups: &infrav1beta1.ManagedSecurityGroups{
						AllNodesSecurityGroupRules: []infrav1beta1.SecurityGroupRuleSpec{
							{
								Name:      "allow-http",
								Direction: "ingress",
							},
							{
								Name:      "allow-https",
								Direction: "ingress",
							},
						},
					},
				},
			},
		},
		{
			name: "mixed with other node rules",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{
					ManagedSecurityGroups: &infrav1beta1.ManagedSecurityGroups{
						AllNodesSecurityGroupRules: []infrav1beta1.SecurityGroupRuleSpec{
							{Name: "all-nodes-rule", Direction: "ingress"},
						},
						ControlPlaneNodesSecurityGroupRules: []infrav1beta1.SecurityGroupRuleSpec{
							{Name: "cp-rule", Direction: "egress"},
						},
						WorkerNodesSecurityGroupRules: []infrav1beta1.SecurityGroupRuleSpec{
							{Name: "worker-rule", Direction: "ingress"},
						},
					},
				},
			},
		},
		{
			name: "no AllNodesSecurityGroupRules — field stays nil",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{
					ManagedSecurityGroups: &infrav1beta1.ManagedSecurityGroups{
						AllowAllInClusterTraffic: true,
					},
				},
			},
		},
		{
			name: "no ManagedSecurityGroups at all",
			in:   infrav1beta1.OpenStackCluster{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			hub := &infrav1.OpenStackCluster{}
			g.Expect(ConvertOpenStackClusterV1Beta1ToHub(t.Context(), &tt.in, hub)).To(Succeed())

			// Verify intermediate v1beta2 state
			if tt.in.Spec.ManagedSecurityGroups == nil {
				g.Expect(hub.Spec.ManagedSecurityGroups).To(BeNil())
			} else {
				g.Expect(hub.Spec.ManagedSecurityGroups).NotTo(BeNil())
				g.Expect(hub.Spec.ManagedSecurityGroups.ClusterNodesSecurityGroupRules).To(HaveLen(len(tt.in.Spec.ManagedSecurityGroups.AllNodesSecurityGroupRules)))
				for i, rule := range tt.in.Spec.ManagedSecurityGroups.AllNodesSecurityGroupRules {
					g.Expect(hub.Spec.ManagedSecurityGroups.ClusterNodesSecurityGroupRules[i].Name).To(Equal(rule.Name))
					g.Expect(hub.Spec.ManagedSecurityGroups.ClusterNodesSecurityGroupRules[i].Direction).To(Equal(rule.Direction))
				}
			}

			restored := &infrav1beta1.OpenStackCluster{}
			g.Expect(ConvertOpenStackClusterHubToV1Beta1(t.Context(), hub, restored)).To(Succeed())

			// Verify full round-trip
			g.Expect(restored.Spec.ManagedSecurityGroups).To(Equal(tt.in.Spec.ManagedSecurityGroups))
		})
	}
}

func TestOpenStackCluster_RoundTrip_APIServer(t *testing.T) {
	floatingIP := optional.String(ptr.To("203.0.113.10"))
	fixedIP := optional.String(ptr.To("10.0.0.5"))
	port := optional.UInt16(ptr.To(uint16(6443)))
	disable := optional.Bool(ptr.To(true))

	tests := []struct {
		name                     string
		in                       infrav1beta1.OpenStackCluster
		expectedEnableFloatingIP optional.Bool
	}{
		{
			name: "all fields set",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{
					APIServerFloatingIP:        floatingIP,
					APIServerFixedIP:           fixedIP,
					APIServerPort:              port,
					DisableAPIServerFloatingIP: disable,
					APIServerLoadBalancer: &infrav1beta1.APIServerLoadBalancer{
						Enabled: ptr.To(true),
					},
				},
			},
			expectedEnableFloatingIP: optional.Bool(ptr.To(false)), // disable=true → enable=false
		},
		{
			name: "only floatingIP set",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{
					APIServerFloatingIP: floatingIP,
				},
			},
			expectedEnableFloatingIP: nil,
		},
		{
			name: "only fixedIP set",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{
					APIServerFixedIP: fixedIP,
				},
			},
			expectedEnableFloatingIP: nil,
		},
		{
			name: "only port set",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{
					APIServerPort: port,
				},
			},
			expectedEnableFloatingIP: nil,
		},
		{
			name: "DisableAPIServerFloatingIP explicitly true",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{
					DisableAPIServerFloatingIP: disable,
				},
			},
			expectedEnableFloatingIP: optional.Bool(ptr.To(false)), // disable=true → enable=false
		},
		{
			name: "DisableAPIServerFloatingIP explicitly false",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{
					DisableAPIServerFloatingIP: optional.Bool(ptr.To(false)),
				},
			},
			expectedEnableFloatingIP: optional.Bool(ptr.To(true)), // disable=false → enable=true
		},
		{
			name: "only loadBalancer set",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{
					APIServerLoadBalancer: &infrav1beta1.APIServerLoadBalancer{
						Enabled: ptr.To(true),
					},
				},
			},
			expectedEnableFloatingIP: nil,
		},
		{
			name: "loadBalancer disabled explicitly",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{
					APIServerLoadBalancer: &infrav1beta1.APIServerLoadBalancer{
						Enabled: ptr.To(false),
					},
				},
			},
			expectedEnableFloatingIP: nil,
		},
		{
			name: "disableFloatingIP with fixedIP (no-LB VIP case)",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{
					DisableAPIServerFloatingIP: disable,
					APIServerFixedIP:           fixedIP,
				},
			},
			expectedEnableFloatingIP: optional.Bool(ptr.To(false)), // disable=true → enable=false
		},
		{
			name:                     "no APIServer fields set — APIServer stays nil",
			in:                       infrav1beta1.OpenStackCluster{},
			expectedEnableFloatingIP: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			hub := &infrav1.OpenStackCluster{}
			g.Expect(ConvertOpenStackClusterV1Beta1ToHub(t.Context(), &tt.in, hub)).To(Succeed())

			src := tt.in.Spec

			// --- Verify intermediate v1beta2 state ---
			allNil := src.APIServerFloatingIP == nil &&
				src.APIServerFixedIP == nil &&
				src.APIServerPort == nil &&
				src.DisableAPIServerFloatingIP == nil &&
				src.APIServerLoadBalancer == nil

			if allNil {
				g.Expect(hub.Spec.APIServer).To(BeNil(), "APIServer should be nil when no source fields are set")
			} else {
				g.Expect(hub.Spec.APIServer).NotTo(BeNil(), "APIServer should be non-nil when at least one source field is set")
				g.Expect(hub.Spec.APIServer.FloatingIP).To(Equal(src.APIServerFloatingIP))
				g.Expect(hub.Spec.APIServer.FixedIP).To(Equal(src.APIServerFixedIP))
				g.Expect(hub.Spec.APIServer.Port).To(Equal(src.APIServerPort))
				g.Expect(hub.Spec.APIServer.EnableFloatingIP).To(Equal(tt.expectedEnableFloatingIP))

				if src.APIServerLoadBalancer == nil {
					g.Expect(hub.Spec.APIServer.ManagedLoadBalancer).To(BeNil())
				} else {
					g.Expect(hub.Spec.APIServer.ManagedLoadBalancer).NotTo(BeNil())
					g.Expect(hub.Spec.APIServer.ManagedLoadBalancer.Enabled).To(Equal(src.APIServerLoadBalancer.Enabled))
				}
			}

			// --- Verify full round-trip back to v1beta1 ---
			restored := &infrav1beta1.OpenStackCluster{}
			g.Expect(ConvertOpenStackClusterHubToV1Beta1(t.Context(), hub, restored)).To(Succeed())

			g.Expect(restored.Spec.APIServerFloatingIP).To(Equal(src.APIServerFloatingIP))
			g.Expect(restored.Spec.APIServerFixedIP).To(Equal(src.APIServerFixedIP))
			g.Expect(restored.Spec.APIServerPort).To(Equal(src.APIServerPort))
			g.Expect(restored.Spec.DisableAPIServerFloatingIP).To(Equal(src.DisableAPIServerFloatingIP))
			g.Expect(restored.Spec.APIServerLoadBalancer).To(Equal(src.APIServerLoadBalancer))
		})
	}
}

func TestOpenStackClusterStatusAPIServerLoadBalancerConversion(t *testing.T) {
	g := NewWithT(t)

	src := &infrav1beta1.OpenStackCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: infrav1beta1.OpenStackClusterSpec{
			IdentityRef: infrav1beta1.OpenStackIdentityReference{
				Name:      "cloud-config",
				CloudName: "openstack",
			},
		},
		Status: infrav1beta1.OpenStackClusterStatus{
			APIServerLoadBalancer: &infrav1beta1.LoadBalancer{
				Name:       "test-lb",
				ID:         "lb-id-123",
				IP:         "192.168.1.100",
				InternalIP: "10.0.0.10",
			},
		},
	}

	// Convert to v1beta2
	dst := &infrav1.OpenStackCluster{}
	g.Expect(ConvertOpenStackClusterV1Beta1ToHub(t.Context(), src, dst)).To(Succeed())

	// Verify field was mapped to the renamed destination field
	g.Expect(dst.Status.APIServerManagedLoadBalancer).NotTo(BeNil())
	g.Expect(dst.Status.APIServerManagedLoadBalancer.Name).To(Equal("test-lb"))
	g.Expect(dst.Status.APIServerManagedLoadBalancer.ID).To(Equal("lb-id-123"))
	g.Expect(dst.Status.APIServerManagedLoadBalancer.IP).To(Equal("192.168.1.100"))
	g.Expect(dst.Status.APIServerManagedLoadBalancer.InternalIP).To(Equal("10.0.0.10"))

	// Convert back
	restored := &infrav1beta1.OpenStackCluster{}
	g.Expect(ConvertOpenStackClusterHubToV1Beta1(t.Context(), dst, restored)).To(Succeed())

	// Verify round-trip
	g.Expect(restored.Status.APIServerLoadBalancer).NotTo(BeNil())
	g.Expect(restored.Status.APIServerLoadBalancer).To(Equal(src.Status.APIServerLoadBalancer))
}

func TestOpenStackClusterStatusAPIServerLoadBalancerNilConversion(t *testing.T) {
	g := NewWithT(t)

	src := &infrav1beta1.OpenStackCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: infrav1beta1.OpenStackClusterSpec{
			IdentityRef: infrav1beta1.OpenStackIdentityReference{
				Name:      "cloud-config",
				CloudName: "openstack",
			},
		},
		Status: infrav1beta1.OpenStackClusterStatus{
			APIServerLoadBalancer: nil,
		},
	}

	// Convert to v1beta2
	dst := &infrav1.OpenStackCluster{}
	g.Expect(ConvertOpenStackClusterV1Beta1ToHub(t.Context(), src, dst)).To(Succeed())

	// Verify nil is preserved and does not bleed into renamed field
	g.Expect(dst.Status.APIServerManagedLoadBalancer).To(BeNil())

	// Convert back
	restored := &infrav1beta1.OpenStackCluster{}
	g.Expect(ConvertOpenStackClusterHubToV1Beta1(t.Context(), dst, restored)).To(Succeed())

	// Verify round-trip preserves nil
	g.Expect(restored.Status.APIServerLoadBalancer).To(BeNil())
}

func TestOpenStackCluster_RoundTrip_ExternalNetwork(t *testing.T) {
	tests := []struct {
		name             string
		in               infrav1beta1.OpenStackCluster
		expectedEnableEN optional.Bool
	}{
		{
			name: "DisableExternalNetwork explicitly true",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{
					DisableExternalNetwork: optional.Bool(ptr.To(true)),
				},
			},
			expectedEnableEN: optional.Bool(ptr.To(false)), // disable=true → enable=false
		},
		{
			name: "DisableExternalNetwork explicitly false",
			in: infrav1beta1.OpenStackCluster{
				Spec: infrav1beta1.OpenStackClusterSpec{
					DisableExternalNetwork: optional.Bool(ptr.To(false)),
				},
			},
			expectedEnableEN: optional.Bool(ptr.To(true)), // disable=false → enable=true
		},
		{
			name:             "DisableExternalNetwork unset — EnableExternalNetwork stays nil",
			in:               infrav1beta1.OpenStackCluster{},
			expectedEnableEN: nil, // unset stays unset, controller applies default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			hub := &infrav1.OpenStackCluster{}
			g.Expect(ConvertOpenStackClusterV1Beta1ToHub(t.Context(), &tt.in, hub)).To(Succeed())

			// --- Verify intermediate v1beta2 state ---
			g.Expect(hub.Spec.EnableExternalNetwork).To(Equal(tt.expectedEnableEN))

			// --- Verify full round-trip back to v1beta1 ---
			restored := &infrav1beta1.OpenStackCluster{}
			g.Expect(ConvertOpenStackClusterHubToV1Beta1(t.Context(), hub, restored)).To(Succeed())

			g.Expect(restored.Spec.DisableExternalNetwork).To(Equal(tt.in.Spec.DisableExternalNetwork))
		})
	}
}
