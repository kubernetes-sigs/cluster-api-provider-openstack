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

	. "github.com/onsi/gomega" //nolint:revive // dot imports are fine in tests
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"

	infrav1beta2 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
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
	dst := &infrav1beta2.OpenStackCluster{}
	g.Expect(src.ConvertTo(dst)).To(Succeed())

	// Verify basic fields
	g.Expect(dst.Name).To(Equal("test-cluster"))
	g.Expect(dst.Namespace).To(Equal("default"))
	g.Expect(dst.Spec.IdentityRef.Name).To(Equal("cloud-config"))
	g.Expect(dst.Spec.ManagedSubnets).To(HaveLen(1))

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

func TestOpenStackMachineConversion(t *testing.T) {
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
	dst := &infrav1beta2.OpenStackMachine{}
	g.Expect(src.ConvertTo(dst)).To(Succeed())

	// Verify basic fields
	g.Expect(dst.Name).To(Equal("test-machine"))
	g.Expect(dst.Spec.Flavor).To(Equal(ptr.To("m1.small")))
	g.Expect(dst.Spec.SSHKeyName).To(Equal("test-key"))
	g.Expect(ptr.Deref((*string)(dst.Spec.Image.Filter.Name), "")).To(Equal("ubuntu-22.04"))

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
	g.Expect(restored.Spec.SSHKeyName).To(Equal("test-key"))
	g.Expect(restored.Status.Ready).To(BeTrue())
	g.Expect(restored.Status.Initialization).NotTo(BeNil())
	g.Expect(restored.Status.Initialization.Provisioned).To(BeTrue())
	g.Expect(*restored.Status.InstanceID).To(Equal("instance-12345"))
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
				},
			},
		},
	}

	// Convert to v1beta2
	dst := &infrav1beta2.OpenStackClusterTemplate{}
	g.Expect(src.ConvertTo(dst)).To(Succeed())

	// Verify template spec
	g.Expect(dst.Name).To(Equal("test-template"))
	g.Expect(dst.Spec.Template.Spec.IdentityRef.Name).To(Equal("cloud-config"))
	g.Expect(dst.Spec.Template.Spec.ManagedSubnets).To(HaveLen(1))

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
					Flavor: ptr.To("m1.large"),
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
	dst := &infrav1beta2.OpenStackMachineTemplate{}
	g.Expect(src.ConvertTo(dst)).To(Succeed())

	// Verify template spec
	g.Expect(dst.Name).To(Equal("test-machine-template"))
	g.Expect(dst.Spec.Template.Spec.Flavor).To(Equal(ptr.To("m1.large")))

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

	v1beta2Conditions := infrav1beta2.ConvertConditionsToV1Beta2(v1beta1Conditions, 10)

	g.Expect(v1beta2Conditions).To(HaveLen(2))
	g.Expect(v1beta2Conditions[0].Type).To(Equal("Ready"))
	g.Expect(v1beta2Conditions[0].Status).To(Equal(metav1.ConditionTrue))
	g.Expect(v1beta2Conditions[0].ObservedGeneration).To(Equal(int64(10)))
	g.Expect(v1beta2Conditions[1].Type).To(Equal("NetworkReady"))
	g.Expect(v1beta2Conditions[1].Status).To(Equal(metav1.ConditionFalse))

	// Test v1beta2 -> v1beta1 condition conversion
	restoredConditions := infrav1beta2.ConvertConditionsFromV1Beta2(v1beta2Conditions)

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
	dst := &infrav1beta2.OpenStackClusterList{}
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

func TestIsReadyHelper(t *testing.T) {
	g := NewWithT(t)

	// Test with Ready condition True
	conditions := []metav1.Condition{
		{
			Type:   "Ready",
			Status: metav1.ConditionTrue,
		},
	}
	g.Expect(infrav1beta2.IsReady(conditions)).To(BeTrue())

	// Test with Ready condition False
	conditions = []metav1.Condition{
		{
			Type:   "Ready",
			Status: metav1.ConditionFalse,
		},
	}
	g.Expect(infrav1beta2.IsReady(conditions)).To(BeFalse())

	// Test with no Ready condition
	conditions = []metav1.Condition{
		{
			Type:   "NetworkReady",
			Status: metav1.ConditionTrue,
		},
	}
	g.Expect(infrav1beta2.IsReady(conditions)).To(BeFalse())

	// Test with empty conditions
	g.Expect(infrav1beta2.IsReady(nil)).To(BeFalse())
	g.Expect(infrav1beta2.IsReady([]metav1.Condition{})).To(BeFalse())
}
