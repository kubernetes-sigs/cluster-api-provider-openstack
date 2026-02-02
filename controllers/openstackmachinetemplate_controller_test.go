/*
Copyright 2025 The Kubernetes Authors.

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
	"fmt"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	. "github.com/onsi/gomega" //nolint:revive
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	capiannotations "sigs.k8s.io/cluster-api/util/annotations"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

var (
	flavorID = "661c21bc-be52-44e3-9d2e-8d1e11623b59"
	imageID  = "ce96e584-7ebc-46d6-9e55-987d72e3806c"
)

func TestOpenStackMachineTemplateReconciler_Reconcile_UnhappyPaths(t *testing.T) {
	ctx := context.Background()

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = clusterv1.AddToScheme(scheme)
	_ = infrav1.AddToScheme(scheme)

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-ns"}}

	tests := []struct {
		name    string
		reqName string
		objects []client.Object
		setup   func(r *OpenStackMachineTemplateReconciler)
		wantErr string
	}{
		{
			name:    "object does not exist",
			reqName: "does-not-exist",
			objects: []client.Object{ns},
		},
		{
			name:    "marked for deletion",
			reqName: "to-be-deleted",
			objects: func() []client.Object {
				tpl := newOSMT("to-be-deleted", "c1", false, false, true)
				now := metav1.Now()
				tpl.DeletionTimestamp = &now
				tpl.Finalizers = []string{"test.finalizer.cluster.x-k8s.io"}
				return []client.Object{ns, tpl}
			}(),
		},
		{
			name:    "missing cluster owner",
			reqName: "no-cluster",
			objects: []client.Object{
				ns,
				func() *infrav1.OpenStackMachineTemplate {
					tpl := newOSMT("no-cluster", "c1", false, false, false)
					delete(tpl.Labels, clusterv1.ClusterNameLabel)
					return tpl
				}(),
			},
		},
		{
			name:    "paused cluster",
			reqName: "paused-cluster",
			objects: []client.Object{
				ns,
				newOSMT("paused-cluster", "paused-cluster", false, false, true),
				newCluster("paused-cluster", "oscluster", true),
				newOSCluster("oscluster"),
			},
		},
		{
			name:    "paused tpl",
			reqName: "paused-tpl",
			objects: []client.Object{
				ns,
				newOSMT("paused-tpl", "cluster", true, false, true),
				newCluster("cluster", "oscluster", false),
				newOSCluster("oscluster"),
			},
		},
		{
			name:    "scope factory returns error",
			reqName: "scope-error",
			objects: []client.Object{
				ns,
				newOSMT("scope-error", "c1", false, false, true),
				newCluster("c1", "oscluster", false),
				newOSCluster("oscluster"),
			},
			setup: func(r *OpenStackMachineTemplateReconciler) {
				mockCtrl := gomock.NewController(t)
				mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "proj")
				mockScopeFactory.SetClientScopeCreateError(fmt.Errorf("boom"))
				r.ScopeFactory = mockScopeFactory
				t.Cleanup(mockCtrl.Finish)
			},
			wantErr: "boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			cl := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			r := &OpenStackMachineTemplateReconciler{
				Client:       cl,
				ScopeFactory: nil, // set in setup when needed
			}

			if tt.setup != nil {
				tt.setup(r)
			}

			req := ctrl.Request{
				NamespacedName: client.ObjectKey{
					Namespace: "test-ns",
					Name:      tt.reqName,
				},
			}

			_, err := r.Reconcile(ctx, req)
			if tt.wantErr == "" {
				g.Expect(err).ToNot(HaveOccurred())
			} else {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(ContainSubstring(tt.wantErr))
			}
		})
	}
}

func TestOpenStackMachineTemplateReconciler_reconcileNormal(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	// --- Scheme setup ---
	scheme := runtime.NewScheme()
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())
	g.Expect(infrav1.AddToScheme(scheme)).To(Succeed())

	type testCase struct {
		name    string
		tpl     *infrav1.OpenStackMachineTemplate
		expect  func(mf *scope.MockScopeFactory)
		wantErr string
		verify  func(g Gomega, tpl *infrav1.OpenStackMachineTemplate)
	}

	tests := []testCase{
		{
			name: "error getting flavor details",
			tpl:  newOSMT("test-osmt", "test-cluster", false, false, true),
			expect: func(mf *scope.MockScopeFactory) {
				mf.ComputeClient.
					EXPECT().
					GetFlavor(flavorID).
					Return(nil, fmt.Errorf("flavor-details-error"))
			},
			wantErr: "flavor-details-error",
		},
		{
			name: "error getting image details",
			tpl:  newOSMT("test-osmt", "test-cluster", false, false, true),
			expect: func(mf *scope.MockScopeFactory) {
				mf.ComputeClient.
					EXPECT().
					GetFlavor(flavorID).
					Return(&flavors.Flavor{
						VCPUs: 2, RAM: 1024, Disk: 5, Ephemeral: 1,
					}, nil)

				mf.ImageClient.
					EXPECT().
					GetImage(imageID).
					Return(nil, fmt.Errorf("image-details-error"))
			},
			wantErr: "image-details-error",
		},
		{
			name: "boot-from-image",
			tpl:  newOSMT("test-osmt", "test-cluster", false, false, true),
			expect: func(mf *scope.MockScopeFactory) {
				mf.ComputeClient.
					EXPECT().
					GetFlavor(flavorID).
					Return(&flavors.Flavor{
						VCPUs:     4,
						RAM:       8192,
						Disk:      50,
						Ephemeral: 10,
					}, nil)

				mf.ImageClient.
					EXPECT().
					GetImage(imageID).
					Return(&images.Image{
						ID: imageID,
						Properties: map[string]any{
							imagePropertyForOS: "linux",
						},
					}, nil)
			},
			wantErr: "",
			verify: func(g Gomega, tpl *infrav1.OpenStackMachineTemplate) {
				// CPU = 4 cores
				expCPU := *resource.NewQuantity(4, resource.DecimalSI)
				g.Expect(tpl.Status.Capacity[corev1.ResourceCPU]).To(Equal(expCPU))

				// Memory = 8192 MiB → bytes
				ramBytes := int64(8192) * 1024 * 1024
				expMem := *resource.NewQuantity(ramBytes, resource.BinarySI)
				g.Expect(tpl.Status.Capacity[corev1.ResourceMemory]).To(Equal(expMem))

				// Ephemeral = 10 GiB → bytes
				ephBytes := int64(10) * 1024 * 1024 * 1024
				expEph := *resource.NewQuantity(ephBytes, resource.BinarySI)
				g.Expect(tpl.Status.Capacity[corev1.ResourceEphemeralStorage]).To(Equal(expEph))

				// Storage = Disk = 50 GiB → bytes (because RootVolume is nil)
				storageBytes := int64(50) * 1024 * 1024 * 1024
				expStorage := *resource.NewQuantity(storageBytes, resource.BinarySI)
				g.Expect(tpl.Status.Capacity[corev1.ResourceStorage]).To(Equal(expStorage))

				// OS property
				g.Expect(tpl.Status.NodeInfo.OperatingSystem).To(Equal("linux"))
			},
		},
		{
			name: "boot-from-volume",
			tpl:  newOSMT("test-osmt", "test-cluster", false, true, true),
			expect: func(mf *scope.MockScopeFactory) {
				mf.ComputeClient.
					EXPECT().
					GetFlavor(flavorID).
					Return(&flavors.Flavor{
						VCPUs:     4,
						RAM:       8192,
						Disk:      50,
						Ephemeral: 10,
					}, nil)

				mf.ImageClient.
					EXPECT().
					GetImage(imageID).
					Return(&images.Image{
						ID: imageID,
						Properties: map[string]any{
							imagePropertyForOS: "linux",
						},
					}, nil)
			},
			wantErr: "",
			verify: func(g Gomega, tpl *infrav1.OpenStackMachineTemplate) {
				// CPU = 4 cores
				expCPU := *resource.NewQuantity(4, resource.DecimalSI)
				g.Expect(tpl.Status.Capacity[corev1.ResourceCPU]).To(Equal(expCPU))

				// Memory = 8192 MiB → bytes
				ramBytes := int64(8192) * 1024 * 1024
				expMem := *resource.NewQuantity(ramBytes, resource.BinarySI)
				g.Expect(tpl.Status.Capacity[corev1.ResourceMemory]).To(Equal(expMem))

				// Ephemeral = 10 GiB → bytes
				ephBytes := int64(10) * 1024 * 1024 * 1024
				expEph := *resource.NewQuantity(ephBytes, resource.BinarySI)
				g.Expect(tpl.Status.Capacity[corev1.ResourceEphemeralStorage]).To(Equal(expEph))

				// Storage = Disk = 100 GiB → bytes (because RootVolume is set)
				storageBytes := int64(100) * 1024 * 1024 * 1024
				expStorage := *resource.NewQuantity(storageBytes, resource.BinarySI)
				g.Expect(tpl.Status.Capacity[corev1.ResourceStorage]).To(Equal(expStorage))

				// OS property
				g.Expect(tpl.Status.NodeInfo.OperatingSystem).To(Equal("linux"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// fake k8s client (used by GetImageID)
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			// gomock controller
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			// CAPO's MockScopeFactory (this is the key)
			mf := scope.NewMockScopeFactory(mockCtrl, "proj")

			// The scope this function expects:
			log := ctrl.Log.WithName("test")
			withLogger := scope.NewWithLogger(mf, log)

			// reconciler
			r := &OpenStackMachineTemplateReconciler{
				Client:       k8sClient,
				ScopeFactory: mf,
			}

			tpl := tt.tpl.DeepCopy()

			tt.expect(mf)

			err := r.reconcileNormal(ctx, withLogger, tpl)

			if tt.wantErr == "" {
				g.Expect(err).ToNot(HaveOccurred())
				if tt.verify != nil {
					tt.verify(g, tpl)
				}
			} else {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(ContainSubstring(tt.wantErr))
			}
		})
	}
}

func newOSMT(name, clusterName string, paused bool, rootVolume bool, ownerRef bool) *infrav1.OpenStackMachineTemplate {
	osmt := &infrav1.OpenStackMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      name,
			Labels: map[string]string{
				clusterv1.ClusterNameLabel: clusterName,
			},
		},
		Spec: infrav1.OpenStackMachineTemplateSpec{
			Template: infrav1.OpenStackMachineTemplateResource{
				Spec: infrav1.OpenStackMachineSpec{
					FlavorID: ptr.To(flavorID),
					Image: infrav1.ImageParam{
						ID: &imageID,
					},
				},
			},
		},
	}

	if rootVolume {
		osmt.Spec.Template.Spec.RootVolume = &infrav1.RootVolume{
			SizeGiB: 100,
		}
	}

	if ownerRef {
		osmt.OwnerReferences = append(osmt.OwnerReferences, metav1.OwnerReference{
			APIVersion: clusterv1.GroupVersion.String(),
			Kind:       "Cluster",
			Name:       clusterName,
			UID:        types.UID("291655c1-d923-4b50-8a3b-d552c03e33a7"),
		})
	}

	if paused {
		capiannotations.AddAnnotations(osmt, map[string]string{
			clusterv1.PausedAnnotation: "true",
		})
	}

	return osmt
}

func newCluster(name, infraName string, paused bool) *clusterv1.Cluster {
	c := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      name,
		},
		Spec: clusterv1.ClusterSpec{
			InfrastructureRef: clusterv1.ContractVersionedObjectReference{
				APIGroup: infrav1.GroupName,
				Kind:     "OpenStackCluster",
				Name:     infraName,
			},
		},
	}
	if paused {
		c.Spec.Paused = &paused
	}
	return c
}

func newOSCluster(name string) *infrav1.OpenStackCluster {
	return &infrav1.OpenStackCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      name,
		},
	}
}
