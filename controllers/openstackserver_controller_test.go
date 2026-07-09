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

package controllers

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"

	infrav1alpha1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
)

const (
	openStackServerName = "test-openstack-server"
	imageName           = "my-image"
	defaultFlavor       = "m1.small"
)

var defaultImage = infrav1.ImageParam{
	Filter: &infrav1.ImageFilter{
		Name: ptr.To(imageName),
	},
}

var defaultPortOpts = []infrav1.PortOpts{
	{
		Network: &infrav1.NetworkParam{
			ID: ptr.To(networkUUID),
		},
	},
}

// newORCTestScheme returns a scheme with CAPO and ORC types registered.
func newORCTestScheme(g Gomega) *runtime.Scheme {
	s := runtime.NewScheme()
	g.Expect(infrav1alpha1.AddToScheme(s)).To(Succeed())
	g.Expect(infrav1.AddToScheme(s)).To(Succeed())
	g.Expect(orcv1alpha1.AddToScheme(s)).To(Succeed())
	return s
}

func TestOpenStackServerReconciler_requeueOpenStackServersForCluster(t *testing.T) {
	tests := []struct {
		name            string
		cluster         *clusterv1.Cluster
		servers         []*infrav1alpha1.OpenStackServer
		clusterDeleting bool
		wantRequests    int
		wantServerNames []string
	}{
		{
			name: "returns reconcile requests for all servers in cluster",
			cluster: &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-ns",
				},
			},
			servers: []*infrav1alpha1.OpenStackServer{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "server-1",
						Namespace: "test-ns",
						Labels: map[string]string{
							clusterv1.ClusterNameLabel: "test-cluster",
						},
					},
					Spec: infrav1alpha1.OpenStackServerSpec{
						Flavor: ptr.To("m1.small"),
						Image: infrav1.ImageParam{
							Filter: &infrav1.ImageFilter{Name: ptr.To("test-image")},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "server-2",
						Namespace: "test-ns",
						Labels: map[string]string{
							clusterv1.ClusterNameLabel: "test-cluster",
						},
					},
					Spec: infrav1alpha1.OpenStackServerSpec{
						Flavor: ptr.To("m1.small"),
						Image: infrav1.ImageParam{
							Filter: &infrav1.ImageFilter{Name: ptr.To("test-image")},
						},
					},
				},
			},
			wantRequests:    2,
			wantServerNames: []string{"server-1", "server-2"},
		},
		{
			name: "returns empty for deleted cluster",
			cluster: &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-ns",
				},
			},
			servers: []*infrav1alpha1.OpenStackServer{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "server-1",
						Namespace: "test-ns",
						Labels: map[string]string{
							clusterv1.ClusterNameLabel: "test-cluster",
						},
					},
					Spec: infrav1alpha1.OpenStackServerSpec{
						Flavor: ptr.To("m1.small"),
						Image: infrav1.ImageParam{
							Filter: &infrav1.ImageFilter{Name: ptr.To("test-image")},
						},
					},
				},
			},
			clusterDeleting: true,
			wantRequests:    0,
		},
		{
			name: "returns empty when no servers exist",
			cluster: &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-ns",
				},
			},
			servers:      []*infrav1alpha1.OpenStackServer{},
			wantRequests: 0,
		},
		{
			name: "only returns servers from same cluster",
			cluster: &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-ns",
				},
			},
			servers: []*infrav1alpha1.OpenStackServer{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "server-1",
						Namespace: "test-ns",
						Labels: map[string]string{
							clusterv1.ClusterNameLabel: "test-cluster",
						},
					},
					Spec: infrav1alpha1.OpenStackServerSpec{
						Flavor: ptr.To("m1.small"),
						Image: infrav1.ImageParam{
							Filter: &infrav1.ImageFilter{Name: ptr.To("test-image")},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "server-2",
						Namespace: "test-ns",
						Labels: map[string]string{
							clusterv1.ClusterNameLabel: "other-cluster",
						},
					},
					Spec: infrav1alpha1.OpenStackServerSpec{
						Flavor: ptr.To("m1.small"),
						Image: infrav1.ImageParam{
							Filter: &infrav1.ImageFilter{Name: ptr.To("test-image")},
						},
					},
				},
			},
			wantRequests:    1,
			wantServerNames: []string{"server-1"},
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			ctx := context.TODO()

			// Set deletion timestamp and finalizers if needed
			if tt.clusterDeleting {
				now := metav1.Now()
				tt.cluster.DeletionTimestamp = &now
				tt.cluster.Finalizers = []string{"test-finalizer"}
			}

			// Create a fake client with the test data
			scheme := runtime.NewScheme()
			g.Expect(clusterv1.AddToScheme(scheme)).To(Succeed())
			g.Expect(infrav1alpha1.AddToScheme(scheme)).To(Succeed())

			objs := make([]client.Object, 0, 1+len(tt.servers))
			objs = append(objs, tt.cluster)
			for _, server := range tt.servers {
				objs = append(objs, server)
			}

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()

			// Create reconciler and call mapper function
			reconciler := &OpenStackServerReconciler{
				Client: fakeClient,
			}
			mapFunc := reconciler.requeueOpenStackServersForCluster(ctx)
			requests := mapFunc(ctx, tt.cluster)

			// Verify results
			if tt.wantRequests == 0 {
				g.Expect(requests).To(Or(BeNil(), BeEmpty()))
			} else {
				g.Expect(requests).To(HaveLen(tt.wantRequests))
				if len(tt.wantServerNames) > 0 {
					gotNames := make([]string, len(requests))
					for i, req := range requests {
						gotNames[i] = req.Name
					}
					g.Expect(gotNames).To(ConsistOf(tt.wantServerNames))
				}
			}
		})
	}
}

func Test_OpenStackServerReconcileDelete(t *testing.T) {
	tests := []struct {
		name                string
		existingObjects     []client.Object
		wantErr             bool
		wantRemoveFinalizer bool
	}{
		{
			name:                "No ORC Server - finalizer removed",
			existingObjects:     nil,
			wantRemoveFinalizer: true,
		},
		{
			name: "ORC Server exists - deletion initiated, still waiting",
			existingObjects: []client.Object{
				&orcv1alpha1.Server{
					ObjectMeta: metav1.ObjectMeta{
						Name:      openStackServerName,
						Namespace: "default",
					},
				},
			},
			wantRemoveFinalizer: false,
		},
	}
	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			ctx := context.TODO()

			scheme := newORCTestScheme(g)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.existingObjects...).
				Build()

			reconciler := &OpenStackServerReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			osServer := &infrav1alpha1.OpenStackServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:       openStackServerName,
					Namespace:  "default",
					Finalizers: []string{infrav1alpha1.OpenStackServerFinalizer},
				},
			}

			err := reconciler.reconcileDelete(ctx, osServer)

			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).ToNot(HaveOccurred())
			}

			if tt.wantRemoveFinalizer {
				g.Expect(osServer.Finalizers).To(BeEmpty())
			} else {
				g.Expect(osServer.Finalizers).To(ConsistOf(infrav1alpha1.OpenStackServerFinalizer))
			}
		})
	}
}

func Test_OpenStackServerReconcileNormal(t *testing.T) {
	tests := []struct {
		name            string
		osServer        infrav1alpha1.OpenStackServer
		existingObjects []client.Object
		wantErr         bool
		wantFinalizer   bool
		wantCondition   *metav1.Condition
		wantInstanceID  string
	}{
		{
			name: "Server in error state returns early",
			osServer: infrav1alpha1.OpenStackServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:       openStackServerName,
					Namespace:  "default",
					UID:        types.UID("test-uid"),
					Finalizers: []string{infrav1alpha1.OpenStackServerFinalizer},
				},
				Status: infrav1alpha1.OpenStackServerStatus{
					InstanceState: ptr.To(infrav1.InstanceStateError),
				},
			},
		},
		{
			name: "Finalizer not set - adds finalizer and returns",
			osServer: infrav1alpha1.OpenStackServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      openStackServerName,
					Namespace: "default",
					UID:       types.UID("test-uid"),
				},
			},
			wantFinalizer: true,
		},
		{
			name: "ORC reconcile creates resources and waits for availability",
			osServer: infrav1alpha1.OpenStackServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:       openStackServerName,
					Namespace:  "default",
					UID:        types.UID("test-uid"),
					Finalizers: []string{infrav1alpha1.OpenStackServerFinalizer},
				},
				Spec: infrav1alpha1.OpenStackServerSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "test-creds",
						CloudName: "openstack",
					},
					Flavor: ptr.To(defaultFlavor),
					Image:  defaultImage,
					Ports:  defaultPortOpts,
				},
			},
			wantCondition: &metav1.Condition{
				Type:   infrav1.InstanceReadyCondition,
				Status: metav1.ConditionFalse,
				Reason: infrav1.InstanceNotReadyReason,
			},
		},
		{
			name: "ORC Server available - status populated",
			osServer: infrav1alpha1.OpenStackServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:       openStackServerName,
					Namespace:  "default",
					UID:        types.UID("test-uid"),
					Finalizers: []string{infrav1alpha1.OpenStackServerFinalizer},
				},
				Spec: infrav1alpha1.OpenStackServerSpec{
					IdentityRef: infrav1.OpenStackIdentityReference{
						Name:      "test-creds",
						CloudName: "openstack",
					},
					Flavor: ptr.To(defaultFlavor),
					Image:  defaultImage,
					Ports:  defaultPortOpts,
				},
			},
			existingObjects: []client.Object{
				&orcv1alpha1.Server{
					ObjectMeta: metav1.ObjectMeta{
						Name:      openStackServerName,
						Namespace: "default",
					},
					Status: orcv1alpha1.ServerStatus{
						Conditions: []metav1.Condition{
							{
								Type:               orcv1alpha1.ConditionAvailable,
								Status:             metav1.ConditionTrue,
								Reason:             orcv1alpha1.ConditionReasonSuccess,
								LastTransitionTime: metav1.Now(),
							},
							{
								Type:               orcv1alpha1.ConditionProgressing,
								Status:             metav1.ConditionFalse,
								Reason:             orcv1alpha1.ConditionReasonSuccess,
								LastTransitionTime: metav1.Now(),
							},
						},
						ID: ptr.To("nova-server-uuid"),
						Resource: &orcv1alpha1.ServerResourceStatus{
							Status: "ACTIVE",
							Interfaces: []orcv1alpha1.ServerInterfaceStatus{
								{
									FixedIPs: []orcv1alpha1.ServerInterfaceFixedIP{
										{IPAddress: "10.0.0.5"},
									},
								},
							},
						},
					},
				},
			},
			wantInstanceID: "nova-server-uuid",
			wantCondition: &metav1.Condition{
				Type:   infrav1.InstanceReadyCondition,
				Status: metav1.ConditionTrue,
				Reason: infrav1.ReadyConditionReason,
			},
		},
	}
	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			ctx := context.TODO()

			scheme := newORCTestScheme(g)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.existingObjects...).
				Build()

			reconciler := &OpenStackServerReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			osServer := &tt.osServer
			_, err := reconciler.reconcileNormal(ctx, osServer)

			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).ToNot(HaveOccurred())
			}

			if tt.wantFinalizer {
				g.Expect(osServer.Finalizers).To(ConsistOf(infrav1alpha1.OpenStackServerFinalizer))
			}

			if tt.wantCondition != nil {
				cond := apimeta.FindStatusCondition(osServer.Status.Conditions, tt.wantCondition.Type)
				g.Expect(cond).ToNot(BeNil(), "expected condition %s to be set", tt.wantCondition.Type)
				g.Expect(cond.Status).To(Equal(tt.wantCondition.Status))
				g.Expect(cond.Reason).To(Equal(tt.wantCondition.Reason))
			}

			if tt.wantInstanceID != "" {
				g.Expect(osServer.Status.InstanceID).ToNot(BeNil())
				g.Expect(*osServer.Status.InstanceID).To(Equal(tt.wantInstanceID))
				g.Expect(osServer.Status.Addresses).ToNot(BeEmpty())
			}
		})
	}
}
