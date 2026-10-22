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
	"encoding/json"
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"

	infrav1beta1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
)

// forgeConversionData sets the conversion-data annotation on hub to the
// serialized spoke, as a principal with update permission on the object (but
// not on its status subresource) could.
func forgeConversionData(g Gomega, spoke, hub metav1.Object) {
	g.Expect(utilconversion.MarshalData(spoke, hub)).To(Succeed())
}

// TestOpenStackClusterConvertFromIgnoresConversionDataStatus simulates a v1beta1
// update of an OpenStackCluster whose stored v1beta2 object carries a forged
// conversion-data annotation. The apiserver copies status from the old object
// as read in v1beta1 onto the new one, so whatever ConvertFrom returns as status
// is what gets persisted.
//
//nolint:staticcheck // Verify forged conversion data cannot restore deprecated v1beta1 status fields.
func TestOpenStackClusterConvertFromIgnoresConversionDataStatus(t *testing.T) {
	g := NewWithT(t)

	stored := &infrav1.OpenStackCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "default"},
		Spec: infrav1.OpenStackClusterSpec{
			ControlPlaneEndpoint: &clusterv1.APIEndpoint{Host: "10.0.0.10", Port: 6443},
		},
		Status: infrav1.OpenStackClusterStatus{
			Network: &infrav1.NetworkStatusWithSubnets{NetworkStatus: infrav1.NetworkStatus{ID: "own-network"}},
			Router:  &infrav1.Router{ID: "own-router"},
			Bastion: &infrav1.BastionStatus{ID: "own-bastion", FloatingIP: "192.0.2.10"},
		},
	}
	forgeConversionData(g, &infrav1beta1.OpenStackCluster{
		Spec: infrav1beta1.OpenStackClusterSpec{
			ControlPlaneEndpoint: &clusterv1beta1.APIEndpoint{Host: "198.51.100.1", Port: 443},
		},
		Status: infrav1beta1.OpenStackClusterStatus{
			Ready:          true,
			FailureMessage: ptr.To("forged"),
			Network:        &infrav1beta1.NetworkStatusWithSubnets{NetworkStatus: infrav1beta1.NetworkStatus{ID: "victim-network"}},
			Router:         &infrav1beta1.Router{ID: "victim-router"},
			Bastion:        &infrav1beta1.BastionStatus{ID: "victim-bastion", FloatingIP: "203.0.113.50"},
		},
	}, stored)

	old := &infrav1beta1.OpenStackCluster{}
	g.Expect(ConvertOpenStackClusterHubToV1Beta1(t.Context(), stored.DeepCopy(), old)).To(Succeed())
	g.Expect(old.Spec.ControlPlaneEndpoint).To(Equal(&clusterv1beta1.APIEndpoint{Host: "10.0.0.10", Port: 6443}))
	g.Expect(old.Status.Ready).To(BeFalse())
	g.Expect(old.Status.FailureMessage).To(BeNil())

	// A label-only update in v1beta1: status comes from the old object.
	updated := old.DeepCopy()
	updated.Labels = map[string]string{"touched": "yes"}

	hub := &infrav1.OpenStackCluster{}
	g.Expect(ConvertOpenStackClusterV1Beta1ToHub(t.Context(), updated, hub)).To(Succeed())
	g.Expect(hub.Spec.ControlPlaneEndpoint).To(Equal(stored.Spec.ControlPlaneEndpoint))
	g.Expect(hub.Status).To(Equal(stored.Status))
}

//nolint:staticcheck // Verify forged conversion data cannot restore the deprecated v1beta1 Ready field.
func TestOpenStackMachineConvertFromIgnoresConversionDataStatus(t *testing.T) {
	g := NewWithT(t)

	stored := &infrav1.OpenStackMachine{
		ObjectMeta: metav1.ObjectMeta{Name: "test-machine", Namespace: "default"},
		Spec: infrav1.OpenStackMachineSpec{
			ProviderID: ptr.To("openstack:///own-instance"),
			Flavor:     infrav1.FlavorParam{ID: ptr.To("own-flavor-id")},
		},
		Status: infrav1.OpenStackMachineStatus{
			InstanceID: ptr.To("own-instance"),
		},
	}
	forgeConversionData(g, &infrav1beta1.OpenStackMachine{
		Spec: infrav1beta1.OpenStackMachineSpec{
			ProviderID: ptr.To("openstack:///victim-instance"),
			FlavorID:   ptr.To("own-flavor-id"),
			Flavor:     ptr.To("m1.small"),
		},
		Status: infrav1beta1.OpenStackMachineStatus{
			Ready:      true,
			InstanceID: ptr.To("victim-instance"),
		},
	}, stored)

	old := &infrav1beta1.OpenStackMachine{}
	g.Expect(ConvertOpenStackMachineHubToV1Beta1(t.Context(), stored.DeepCopy(), old)).To(Succeed())
	g.Expect(old.Spec.ProviderID).To(Equal(ptr.To("openstack:///own-instance")))
	g.Expect(old.Status.Ready).To(BeFalse())
	g.Expect(old.Status.InstanceID).To(HaveValue(Equal("own-instance")))
	// The spoke-only flavor name is still restored alongside the same ID.
	g.Expect(old.Spec.Flavor).To(Equal(ptr.To("m1.small")))

	hub := &infrav1.OpenStackMachine{}
	g.Expect(ConvertOpenStackMachineV1Beta1ToHub(t.Context(), old, hub)).To(Succeed())
	g.Expect(hub.Spec.Flavor).To(Equal(stored.Spec.Flavor))
	g.Expect(hub.Status).To(Equal(stored.Status))
}

func TestOpenStackMachineTemplateConvertFromIgnoresConversionDataStatus(t *testing.T) {
	g := NewWithT(t)

	stored := &infrav1.OpenStackMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "test-machine-template", Namespace: "default"},
		Spec: infrav1.OpenStackMachineTemplateSpec{
			Template: infrav1.OpenStackMachineTemplateResource{
				Spec: infrav1.OpenStackMachineSpec{Flavor: infrav1.FlavorParam{ID: ptr.To("own-flavor-id")}},
			},
		},
		Status: infrav1.OpenStackMachineTemplateStatus{
			NodeInfo: infrav1.NodeInfo{OperatingSystem: "linux"},
		},
	}
	forgeConversionData(g, &infrav1beta1.OpenStackMachineTemplate{
		Status: infrav1beta1.OpenStackMachineTemplateStatus{
			NodeInfo: infrav1beta1.NodeInfo{OperatingSystem: "windows"},
		},
	}, stored)

	old := &infrav1beta1.OpenStackMachineTemplate{}
	g.Expect(ConvertOpenStackMachineTemplateHubToV1Beta1(t.Context(), stored.DeepCopy(), old)).To(Succeed())
	g.Expect(old.Spec.Template.Spec.FlavorID).To(Equal(ptr.To("own-flavor-id")))

	hub := &infrav1.OpenStackMachineTemplate{}
	g.Expect(ConvertOpenStackMachineTemplateV1Beta1ToHub(t.Context(), old, hub)).To(Succeed())
	g.Expect(hub.Status).To(Equal(stored.Status))
}

// TestConvertFromRestoresFlavorOnlyForSameFlavorID verifies that a stale or
// forged flavor name is not restored once the hub flavor ID no longer matches.
func TestConvertFromRestoresFlavorOnlyForSameFlavorID(t *testing.T) {
	g := NewWithT(t)

	hub := &infrav1.OpenStackCluster{
		Spec: infrav1.OpenStackClusterSpec{
			Bastion: &infrav1.Bastion{
				Spec: &infrav1.OpenStackMachineSpec{Flavor: infrav1.FlavorParam{ID: ptr.To("new-flavor-id")}},
			},
		},
	}
	forgeConversionData(g, &infrav1beta1.OpenStackCluster{
		Spec: infrav1beta1.OpenStackClusterSpec{
			Bastion: &infrav1beta1.Bastion{
				Spec: &infrav1beta1.OpenStackMachineSpec{FlavorID: ptr.To("old-flavor-id"), Flavor: ptr.To("m1.small")},
			},
		},
	}, hub)

	spoke := &infrav1beta1.OpenStackCluster{}
	g.Expect(ConvertOpenStackClusterHubToV1Beta1(t.Context(), hub, spoke)).To(Succeed())
	g.Expect(spoke.Spec.Bastion.Spec.FlavorID).To(Equal(ptr.To("new-flavor-id")))
	g.Expect(spoke.Spec.Bastion.Spec.Flavor).To(BeNil())
}

// TestConvertFromDoesNotRestoreEmptyFlavor verifies that an empty flavor name,
// which v1beta1 validation rejects, is not restored.
func TestConvertFromDoesNotRestoreEmptyFlavor(t *testing.T) {
	g := NewWithT(t)

	hub := &infrav1.OpenStackMachine{
		Spec: infrav1.OpenStackMachineSpec{Flavor: infrav1.FlavorParam{ID: ptr.To("flavor-id")}},
	}
	forgeConversionData(g, &infrav1beta1.OpenStackMachine{
		Spec: infrav1beta1.OpenStackMachineSpec{FlavorID: ptr.To("flavor-id"), Flavor: ptr.To("")},
	}, hub)

	spoke := &infrav1beta1.OpenStackMachine{}
	g.Expect(ConvertOpenStackMachineHubToV1Beta1(t.Context(), hub, spoke)).To(Succeed())
	g.Expect(spoke.Spec.FlavorID).To(Equal(ptr.To("flavor-id")))
	g.Expect(spoke.Spec.Flavor).To(BeNil())
}

// TestOpenStackClusterTemplateConvertFromRestoresOnlyFlavor verifies that the
// conversion data annotation can't override the template spec, while the
// bastion flavor name still round-trips.
func TestOpenStackClusterTemplateConvertFromRestoresOnlyFlavor(t *testing.T) {
	g := NewWithT(t)

	hub := &infrav1.OpenStackClusterTemplate{
		Spec: infrav1.OpenStackClusterTemplateSpec{
			Template: infrav1.OpenStackClusterTemplateResource{
				Spec: infrav1.OpenStackClusterSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{Name: "own-credentials", CloudName: "openstack"},
					Bastion: &infrav1.Bastion{
						Spec: &infrav1.OpenStackMachineSpec{Flavor: infrav1.FlavorParam{ID: ptr.To("flavor-id")}},
					},
				},
			},
		},
	}
	forgeConversionData(g, &infrav1beta1.OpenStackClusterTemplate{
		Spec: infrav1beta1.OpenStackClusterTemplateSpec{
			Template: infrav1beta1.OpenStackClusterTemplateResource{
				Spec: infrav1beta1.OpenStackClusterSpec{
					IdentityRef: infrav1beta1.OpenStackIdentityReference{Name: "other-credentials", CloudName: "openstack"},
					Bastion: &infrav1beta1.Bastion{
						Spec: &infrav1beta1.OpenStackMachineSpec{FlavorID: ptr.To("flavor-id"), Flavor: ptr.To("m1.small")},
					},
				},
			},
		},
	}, hub)

	spoke := &infrav1beta1.OpenStackClusterTemplate{}
	g.Expect(ConvertOpenStackClusterTemplateHubToV1Beta1(t.Context(), hub, spoke)).To(Succeed())
	g.Expect(spoke.Spec.Template.Spec.IdentityRef.Name).To(Equal("own-credentials"))
	g.Expect(spoke.Spec.Template.Spec.Bastion.Spec.Flavor).To(Equal(ptr.To("m1.small")))
}

// TestConvertToDoesNotStashStatus verifies that the conversion-data annotation
// written by ConvertTo carries no status.
//
//nolint:staticcheck // Verify conversion data omits status, including the deprecated v1beta1 Ready field.
func TestConvertToDoesNotStashStatus(t *testing.T) {
	t.Run("OpenStackCluster", func(t *testing.T) {
		g := NewWithT(t)
		spoke := &infrav1beta1.OpenStackCluster{Status: infrav1beta1.OpenStackClusterStatus{Ready: true, Router: &infrav1beta1.Router{ID: "router"}}}
		hub := &infrav1.OpenStackCluster{}
		g.Expect(ConvertOpenStackClusterV1Beta1ToHub(t.Context(), spoke, hub)).To(Succeed())
		data := map[string]any{}
		g.Expect(json.Unmarshal([]byte(hub.Annotations[utilconversion.DataAnnotation]), &data)).To(Succeed())
		g.Expect(data).To(HaveKey("spec"))
		g.Expect(data).NotTo(HaveKey("status"))
	})
	t.Run("OpenStackMachine", func(t *testing.T) {
		g := NewWithT(t)
		spoke := &infrav1beta1.OpenStackMachine{Status: infrav1beta1.OpenStackMachineStatus{Ready: true, InstanceID: ptr.To("instance")}}
		hub := &infrav1.OpenStackMachine{}
		g.Expect(ConvertOpenStackMachineV1Beta1ToHub(t.Context(), spoke, hub)).To(Succeed())
		data := map[string]any{}
		g.Expect(json.Unmarshal([]byte(hub.Annotations[utilconversion.DataAnnotation]), &data)).To(Succeed())
		g.Expect(data).To(HaveKey("spec"))
		g.Expect(data).NotTo(HaveKey("status"))
	})
	t.Run("OpenStackMachineTemplate", func(t *testing.T) {
		g := NewWithT(t)
		spoke := &infrav1beta1.OpenStackMachineTemplate{Status: infrav1beta1.OpenStackMachineTemplateStatus{NodeInfo: infrav1beta1.NodeInfo{OperatingSystem: "linux"}}}
		hub := &infrav1.OpenStackMachineTemplate{}
		g.Expect(ConvertOpenStackMachineTemplateV1Beta1ToHub(t.Context(), spoke, hub)).To(Succeed())
		data := map[string]any{}
		g.Expect(json.Unmarshal([]byte(hub.Annotations[utilconversion.DataAnnotation]), &data)).To(Succeed())
		g.Expect(data).To(HaveKey("spec"))
		g.Expect(data).NotTo(HaveKey("status"))
	})
}
