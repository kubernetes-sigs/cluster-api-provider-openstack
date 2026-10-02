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

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	conditions "sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

// writeLegacyV1beta1Conditions writes `True` conditions without a reason to the
// status of an existing object, the same way CAPO < 0.15 did with MarkTrue.
//
// The envtest CRDs have no conversion webhook, so this goes through the
// (reason-optional) v1beta1 schema and is read back unchanged as v1beta2.
// Writing through v1beta2 is not possible because its schema requires a reason.
func writeLegacyV1beta1Conditions(ctx context.Context, c client.Client, kind string, key client.ObjectKey, conditionTypes ...string) error {
	legacy := make([]map[string]any, 0, len(conditionTypes))
	for _, t := range conditionTypes {
		legacy = append(legacy, map[string]any{
			"type":               t,
			"status":             string(metav1.ConditionTrue),
			"lastTransitionTime": "2026-09-17T12:53:31Z",
		})
	}
	body, err := json.Marshal(map[string]any{"status": map[string]any{"conditions": legacy}})
	if err != nil {
		return err
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: infrav1.SchemeGroupVersion.Group, Version: "v1beta1", Kind: kind})
	obj.SetNamespace(key.Namespace)
	obj.SetName(key.Name)
	return c.Status().Patch(ctx, obj, client.RawPatch(types.MergePatchType, body))
}

// TestOpenStackMachineReconcile_FillsMissingConditionReasons verifies that the
// OpenStackMachine reconciler can persist its status when the stored
// conditions have no reason, as written by CAPO < 0.15.
//
// This uses a fake client because the envtest CRDs cannot hold an
// OpenStackMachine with reason-less conditions: v1beta2 rejects them on write,
// and v1beta1 is incompatible with the v1beta2 spec without a conversion
// webhook. The envtest-based equivalent for OpenStackCluster lives in
// openstackcluster_controller_test.go.
func TestOpenStackMachineReconcile_FillsMissingConditionReasons(t *testing.T) {
	g := NewWithT(t)
	ctx := context.TODO()

	const (
		ns          = "legacy-conditions"
		clusterName = "cluster"
		machineName = "machine"
	)

	capiCluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: clusterName, Namespace: ns},
		Spec: clusterv1.ClusterSpec{
			InfrastructureRef: clusterv1.ContractVersionedObjectReference{
				APIGroup: infrav1.GroupName,
				Kind:     "OpenStackCluster",
				Name:     clusterName,
			},
		},
	}
	infraCluster := &infrav1.OpenStackCluster{
		ObjectMeta: metav1.ObjectMeta{Name: clusterName, Namespace: ns},
	}
	capiMachine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      machineName,
			Namespace: ns,
			Labels:    map[string]string{clusterv1.ClusterNameLabel: clusterName},
		},
	}

	legacyConditionTypes := []string{string(clusterv1.ReadyCondition), infrav1.InstanceReadyCondition}
	infraMachine := &infrav1.OpenStackMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      machineName,
			Namespace: ns,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: clusterv1.GroupVersion.String(),
				Kind:       "Machine",
				Name:       machineName,
			}},
		},
	}
	for _, conditionType := range legacyConditionTypes {
		infraMachine.Status.Conditions = append(infraMachine.Status.Conditions, metav1.Condition{
			Type:               conditionType,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
		})
	}

	scheme := runtime.NewScheme()
	g.Expect(clusterv1.AddToScheme(scheme)).To(Succeed())
	g.Expect(infrav1.AddToScheme(scheme)).To(Succeed())
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(capiCluster, infraCluster, capiMachine, infraMachine).
		WithStatusSubresource(&infrav1.OpenStackMachine{}).
		Build()

	mockCtrl := gomock.NewController(t)
	mockFactory := scope.NewMockScopeFactory(mockCtrl, "")
	credentialsErr := fmt.Errorf("secret not found: non-existent-secret")
	mockFactory.SetClientScopeCreateError(credentialsErr)

	reconciler := &OpenStackMachineReconciler{
		Client:       fakeClient,
		ScopeFactory: mockFactory,
	}

	result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(infraMachine)})
	g.Expect(err).To(MatchError(credentialsErr))
	g.Expect(result).To(Equal(reconcile.Result{}))

	updated := &infrav1.OpenStackMachine{}
	g.Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(infraMachine), updated)).To(Succeed())

	// The new condition was persisted and all legacy conditions got a reason.
	g.Expect(conditions.IsFalse(updated, infrav1.OpenStackAuthenticationSucceeded)).To(BeTrue())
	for _, conditionType := range legacyConditionTypes {
		condition := conditions.Get(updated, conditionType)
		g.Expect(condition).ToNot(BeNil(), conditionType)
		g.Expect(condition.Status).To(Equal(metav1.ConditionTrue), conditionType)
		g.Expect(condition.Reason).To(Equal(infrav1.ReadyConditionReason), conditionType)
	}
}
