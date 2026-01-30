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
	"fmt"
	"reflect"
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/util/conditions"
	v1beta1conditions "sigs.k8s.io/cluster-api/util/deprecated/v1beta1/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1alpha1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/compute"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

const (
	openStackServerName = "test-openstack-server"
	instanceUUID        = "8308882f-5e46-47e6-8e12-1fe869c43d1d"
	portUUID            = "55eac199-4836-4a98-b31c-9f65f382ad46"
	rootVolumeUUID      = "4724a66d-bd5e-47f3-bb57-a67fcb4168e0"
	trunkUUID           = "9d348baa-93b1-4e63-932f-dd0527fbd789"
	imageName           = "my-image"
	defaultFlavor       = "m1.small"
)

type recorders struct {
	compute *mock.MockComputeClientMockRecorder
	image   *mock.MockImageClientMockRecorder
	network *mock.MockNetworkClientMockRecorder
	volume  *mock.MockVolumeClientMockRecorder
}

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

var defaultResolvedPorts = []infrav1.ResolvedPortSpec{
	{
		Name:      openStackServerName + "-0",
		NetworkID: networkUUID,
	},
}

var defaultPortsStatus = []infrav1.PortStatus{
	{
		ID: portUUID,
	},
}

var createDefaultPort = func(r *recorders) {
	createOpts := ports.CreateOpts{
		Name:      openStackServerName + "-0",
		NetworkID: networkUUID,
	}
	portsBuilder := portsbinding.CreateOptsExt{
		CreateOptsBuilder: createOpts,
	}
	r.network.CreatePort(portsBuilder).Return(&ports.Port{
		ID: portUUID,
	}, nil)
}

var createDefaultPortFails = func(r *recorders) {
	createOpts := ports.CreateOpts{
		Name:      openStackServerName + "-0",
		NetworkID: networkUUID,
	}
	portsBuilder := portsbinding.CreateOptsExt{
		CreateOptsBuilder: createOpts,
	}
	r.network.CreatePort(portsBuilder).Return(nil, fmt.Errorf("Error creating port"))
}

var createDefaultServer = func(r *recorders) {
	// Mock any server creation
	r.compute.CreateServer(gomock.Any(), gomock.Any()).Return(&servers.Server{ID: instanceUUID}, nil)
}

var listDefaultPorts = func(r *recorders) {
	r.network.ListPort(ports.ListOpts{
		Name:      openStackServerName + "-0",
		NetworkID: networkUUID,
	}).Return([]ports.Port{
		{
			ID: portUUID,
		},
	}, nil)
}

var listDefaultPortsWithID = func(r *recorders) {
	r.network.ListPort(ports.ListOpts{
		ID: portUUID,
	}).Return([]ports.Port{
		{
			ID: portUUID,
		},
	}, nil)
}

var listDefaultPortsNotFound = func(r *recorders) {
	r.network.ListPort(ports.ListOpts{
		Name:      openStackServerName + "-0",
		NetworkID: networkUUID,
	}).Return(nil, nil)
}

var listDefaultServerNotFound = func(r *recorders) {
	r.compute.ListServers(servers.ListOpts{
		Name: "^" + openStackServerName + "$",
	}).Return([]servers.Server{}, nil)
}

var listDefaultServerFound = func(r *recorders) {
	r.compute.ListServers(servers.ListOpts{
		Name: "^" + openStackServerName + "$",
	}).Return([]servers.Server{{ID: instanceUUID}}, nil)
}

var deleteDefaultPorts = func(r *recorders) {
	trunkExtension := extensions.Extension{}
	trunkExtension.Alias = "trunk"
	r.network.ListExtensions().Return([]extensions.Extension{trunkExtension}, nil)
	r.network.ListTrunk(trunks.ListOpts{PortID: portUUID}).Return([]trunks.Trunk{{ID: trunkUUID}}, nil)
	r.network.ListTrunkSubports(trunkUUID).Return([]trunks.Subport{}, nil)
	r.network.DeleteTrunk(trunkUUID).Return(nil)
	r.network.DeletePort(portUUID).Return(nil)
}

var deleteServerByID = func(r *recorders) {
	r.compute.GetServer(instanceUUID).Return(&servers.Server{ID: instanceUUID, Name: openStackServerName}, nil)
	r.compute.DeleteServer(instanceUUID).Return(nil)
	r.compute.GetServer(instanceUUID).Return(nil, gophercloud.ErrUnexpectedResponseCode{Actual: 404})
}

var deleteServerByName = func(r *recorders) {
	r.compute.ListServers(servers.ListOpts{
		Name: "^" + openStackServerName + "$",
	}).Return([]servers.Server{{ID: instanceUUID, Name: openStackServerName}}, nil)
	r.compute.DeleteServer(instanceUUID).Return(nil)
	r.compute.GetServer(instanceUUID).Return(nil, gophercloud.ErrUnexpectedResponseCode{Actual: 404})
}

var deleteMissingServerByName = func(r *recorders) {
	// Lookup server by name because it is not in status.
	// Don't find it.
	r.compute.ListServers(servers.ListOpts{
		Name: "^" + openStackServerName + "$",
	}).Return(nil, nil)
}

var deleteRootVolume = func(r *recorders) {
	// Fetch volume by name
	volumeName := fmt.Sprintf("%s-root", openStackServerName)
	r.volume.ListVolumes(volumes.ListOpts{
		AllTenants: false,
		Name:       volumeName,
		TenantID:   "",
	}).Return([]volumes.Volume{{
		ID:   rootVolumeUUID,
		Name: volumeName,
	}}, nil)

	// Delete volume
	r.volume.DeleteVolume(rootVolumeUUID, volumes.DeleteOpts{}).Return(nil)
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
							clusterv1beta1.ClusterNameLabel: "test-cluster",
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
							clusterv1beta1.ClusterNameLabel: "test-cluster",
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
							clusterv1beta1.ClusterNameLabel: "test-cluster",
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
							clusterv1beta1.ClusterNameLabel: "test-cluster",
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
							clusterv1beta1.ClusterNameLabel: "other-cluster",
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

func TestOpenStackServer_serverToInstanceSpec(t *testing.T) {
	tests := []struct {
		name            string
		openStackServer *infrav1alpha1.OpenStackServer
		want            *compute.InstanceSpec
		wantErr         bool
	}{
		{
			name:            "Test serverToInstanceSpec without resolved resources",
			openStackServer: &infrav1alpha1.OpenStackServer{},
			wantErr:         true,
		},
		{
			name: "Test serverToInstanceSpec with resolved resources",
			openStackServer: &infrav1alpha1.OpenStackServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: infrav1alpha1.OpenStackServerSpec{
					AdditionalBlockDevices: []infrav1.AdditionalBlockDevice{
						{
							Name:    "block-device",
							SizeGiB: 10,
							Storage: infrav1.BlockDeviceStorage{
								Type: "ceph",
							},
						},
					},
					AvailabilityZone: ptr.To("failure-domain"),
					ConfigDrive:      ptr.To(true),
					RootVolume: &infrav1.RootVolume{
						SizeGiB: 10,
						BlockDeviceVolume: infrav1.BlockDeviceVolume{
							Type: "fast",
						},
					},
					ServerMetadata: []infrav1.ServerMetadata{{Key: "key", Value: "value"}},
					SSHKeyName:     "key",
					Tags:           []string{"tag1", "tag2"},
					Trunk:          ptr.To(true),
				},
				Status: infrav1alpha1.OpenStackServerStatus{
					Resolved: &infrav1alpha1.ResolvedServerSpec{
						FlavorID:      "xyz",
						ImageID:       "123",
						ServerGroupID: "456",
					},
				},
			},
			want: &compute.InstanceSpec{
				AdditionalBlockDevices: []infrav1.AdditionalBlockDevice{
					{
						Name:    "block-device",
						SizeGiB: 10,
						Storage: infrav1.BlockDeviceStorage{
							Type: "ceph",
						},
					},
				},
				ConfigDrive:   true,
				FailureDomain: "failure-domain",
				FlavorID:      "xyz",
				ImageID:       "123",
				Metadata: map[string]string{
					"key": "value",
				},
				Name: "test",
				RootVolume: &infrav1.RootVolume{
					SizeGiB: 10,
					BlockDeviceVolume: infrav1.BlockDeviceVolume{
						Type: "fast",
					},
				},
				ServerGroupID: "456",
				SSHKeyName:    "key",
				Tags:          []string{"tag1", "tag2"},
				Trunk:         true,
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			reconciler := OpenStackServerReconciler{}
			spec, err := reconciler.serverToInstanceSpec(ctx, tt.openStackServer)
			if (err != nil) != tt.wantErr {
				t.Fatalf("serverToInstanceSpec() error = %+v, wantErr %+v", err, tt.wantErr)
			}
			if err == nil && !reflect.DeepEqual(spec, tt.want) {
				t.Errorf("serverToInstanceSpec() got = %+v, want %+v", spec, tt.want)
			}
		})
	}
}

func Test_OpenStackServerReconcileDelete(t *testing.T) {
	tests := []struct {
		name                string
		osServer            infrav1alpha1.OpenStackServer
		expect              func(r *recorders)
		wantErr             bool
		wantRemoveFinalizer bool
	}{
		{
			name: "No volumes, resolved and resources populated",
			osServer: infrav1alpha1.OpenStackServer{
				Spec: infrav1alpha1.OpenStackServerSpec{
					Flavor: ptr.To(defaultFlavor),
					Image:  defaultImage,
					Ports:  defaultPortOpts,
				},
				Status: infrav1alpha1.OpenStackServerStatus{
					InstanceID: ptr.To(instanceUUID),
					Resolved: &infrav1alpha1.ResolvedServerSpec{
						ImageID: imageUUID,
						Ports:   defaultResolvedPorts,
					},
					Resources: &infrav1alpha1.ServerResources{
						Ports: defaultPortsStatus,
					},
				},
			},
			expect: func(r *recorders) {
				deleteServerByID(r)
				deleteDefaultPorts(r)
			},
			wantRemoveFinalizer: true,
		},
		{
			name: "Root volume, resolved and resources populated",
			osServer: infrav1alpha1.OpenStackServer{
				Spec: infrav1alpha1.OpenStackServerSpec{
					Image: defaultImage,
					RootVolume: &infrav1.RootVolume{
						SizeGiB: 50,
					},
					Ports: defaultPortOpts,
				},
				Status: infrav1alpha1.OpenStackServerStatus{
					InstanceID: ptr.To(instanceUUID),
					Resolved: &infrav1alpha1.ResolvedServerSpec{
						ImageID: imageUUID,
						Ports:   defaultResolvedPorts,
					},
					Resources: &infrav1alpha1.ServerResources{
						Ports: defaultPortsStatus,
					},
				},
			},
			expect: func(r *recorders) {
				// Server exists, so we don't delete root volume explicitly
				deleteServerByID(r)
				deleteDefaultPorts(r)
			},
			wantRemoveFinalizer: true,
		},
		{
			name: "Root volume, server not created, resolved and resources populated",
			osServer: infrav1alpha1.OpenStackServer{
				Spec: infrav1alpha1.OpenStackServerSpec{
					Image: defaultImage,
					RootVolume: &infrav1.RootVolume{
						SizeGiB: 50,
					},
					Ports: defaultPortOpts,
				},
				Status: infrav1alpha1.OpenStackServerStatus{
					Resolved: &infrav1alpha1.ResolvedServerSpec{
						ImageID: imageUUID,
						Ports:   defaultResolvedPorts,
					},
					Resources: &infrav1alpha1.ServerResources{
						Ports: defaultPortsStatus,
					},
				},
			},
			expect: func(r *recorders) {
				deleteMissingServerByName(r)
				deleteRootVolume(r)
				deleteDefaultPorts(r)
			},
			wantRemoveFinalizer: true,
		},
		{
			name: "No instance id, server and ports exist",
			osServer: infrav1alpha1.OpenStackServer{
				Spec: infrav1alpha1.OpenStackServerSpec{
					Image: defaultImage,
					Ports: defaultPortOpts,
				},
				Status: infrav1alpha1.OpenStackServerStatus{
					Resolved: &infrav1alpha1.ResolvedServerSpec{
						ImageID: imageUUID,
						Ports:   defaultResolvedPorts,
					},
					Resources: &infrav1alpha1.ServerResources{
						Ports: defaultPortsStatus,
					},
				},
			},
			expect: func(r *recorders) {
				deleteServerByName(r)
				deleteDefaultPorts(r)
			},
			wantRemoveFinalizer: true,
		},
		{
			name: "Adopt ports error should fail deletion and retry",
			osServer: infrav1alpha1.OpenStackServer{
				Spec: infrav1alpha1.OpenStackServerSpec{
					Image: defaultImage,
					Ports: defaultPortOpts,
				},
				Status: infrav1alpha1.OpenStackServerStatus{
					Resolved: &infrav1alpha1.ResolvedServerSpec{
						ImageID: imageUUID,
						Ports:   defaultResolvedPorts,
					},
				},
			},
			expect: func(r *recorders) {
				r.network.ListPort(ports.ListOpts{
					NetworkID: networkUUID,
					Name:      openStackServerName + "-0",
				}).Return(nil, fmt.Errorf("error adopting ports"))
			},
			wantErr:             true,
			wantRemoveFinalizer: false,
		},
	}
	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			log := testr.New(t)

			mockCtrl := gomock.NewController(t)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")

			reconciler := OpenStackServerReconciler{}

			computeRecorder := mockScopeFactory.ComputeClient.EXPECT()
			imageRecorder := mockScopeFactory.ImageClient.EXPECT()
			networkRecorder := mockScopeFactory.NetworkClient.EXPECT()
			volumeRecorder := mockScopeFactory.VolumeClient.EXPECT()

			if tt.expect != nil {
				tt.expect(&recorders{computeRecorder, imageRecorder, networkRecorder, volumeRecorder})
			}
			scopeWithLogger := scope.NewWithLogger(mockScopeFactory, log)

			osServer := &tt.osServer
			osServer.Name = openStackServerName
			osServer.Finalizers = []string{infrav1alpha1.OpenStackServerFinalizer}

			err := reconciler.reconcileDelete(scopeWithLogger, &tt.osServer)

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

func Test_OpenStackServerReconcileCreate(t *testing.T) {
	tests := []struct {
		name          string
		osServer      infrav1alpha1.OpenStackServer
		expect        func(r *recorders)
		wantErr       error
		wantCondition *clusterv1beta1.Condition
	}{
		{
			name: "Minimal server spec creating port and server",
			osServer: infrav1alpha1.OpenStackServer{
				Spec: infrav1alpha1.OpenStackServerSpec{
					Flavor: ptr.To(defaultFlavor),
					Image:  defaultImage,
					Ports:  defaultPortOpts,
				},
				Status: infrav1alpha1.OpenStackServerStatus{
					Resolved: &infrav1alpha1.ResolvedServerSpec{
						ImageID:  imageUUID,
						FlavorID: flavorUUID,
						Ports:    defaultResolvedPorts,
					},
				},
			},
			expect: func(r *recorders) {
				listDefaultPortsNotFound(r)
				createDefaultPort(r)
				listDefaultServerNotFound(r)
				listDefaultPortsNotFound(r)
				createDefaultServer(r)
			},
		},
		{
			name: "Minimum server spec adopting port and server",
			osServer: infrav1alpha1.OpenStackServer{
				Spec: infrav1alpha1.OpenStackServerSpec{
					Flavor: ptr.To(defaultFlavor),
					Image:  defaultImage,
					Ports:  defaultPortOpts,
				},
				Status: infrav1alpha1.OpenStackServerStatus{
					Resolved: &infrav1alpha1.ResolvedServerSpec{
						ImageID:  imageUUID,
						FlavorID: flavorUUID,
						Ports:    defaultResolvedPorts,
					},
				},
			},
			expect: func(r *recorders) {
				listDefaultPorts(r)
				listDefaultPortsWithID(r)
				listDefaultServerFound(r)
			},
		},
		{
			name: "Port created with error",
			osServer: infrav1alpha1.OpenStackServer{
				Spec: infrav1alpha1.OpenStackServerSpec{
					Flavor: ptr.To(defaultFlavor),
					Image:  defaultImage,
					Ports:  defaultPortOpts,
				},
				Status: infrav1alpha1.OpenStackServerStatus{
					Resolved: &infrav1alpha1.ResolvedServerSpec{
						ImageID:  imageUUID,
						FlavorID: flavorUUID,
						Ports:    defaultResolvedPorts,
					},
				},
			},
			expect: func(r *recorders) {
				listDefaultPortsNotFound(r)
				listDefaultPortsNotFound(r)
				createDefaultPortFails(r)
			},
			wantErr: fmt.Errorf("creating ports: %w", fmt.Errorf("Error creating port")),
			wantCondition: &clusterv1beta1.Condition{
				Type:     infrav1.InstanceReadyCondition,
				Status:   corev1.ConditionFalse,
				Severity: clusterv1beta1.ConditionSeverityError,
				Reason:   infrav1.PortCreateFailedReason,
				Message:  "Error creating port",
			},
		},
	}
	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			log := testr.New(t)

			mockCtrl := gomock.NewController(t)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")

			reconciler := OpenStackServerReconciler{}

			computeRecorder := mockScopeFactory.ComputeClient.EXPECT()
			imageRecorder := mockScopeFactory.ImageClient.EXPECT()
			networkRecorder := mockScopeFactory.NetworkClient.EXPECT()
			volumeRecorder := mockScopeFactory.VolumeClient.EXPECT()

			if tt.expect != nil {
				tt.expect(&recorders{computeRecorder, imageRecorder, networkRecorder, volumeRecorder})
			}
			scopeWithLogger := scope.NewWithLogger(mockScopeFactory, log)

			osServer := &tt.osServer
			osServer.Name = openStackServerName
			osServer.Finalizers = []string{infrav1alpha1.OpenStackServerFinalizer}

			_, err := reconciler.reconcileNormal(ctx, scopeWithLogger, &tt.osServer)

			// Check error result
			if tt.wantErr != nil {
				g.Expect(err).To(Equal(tt.wantErr))
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}

			// Check the condition is set correctly
			if tt.wantCondition != nil {
				// print openstackServer conditions
				for _, condition := range tt.osServer.Status.Conditions {
					t.Logf("Condition: %s, Status: %s, Reason: %s", condition.Type, condition.Status, condition.Reason)
				}
				unstructuredServer, err := tt.osServer.ToUnstructured()
				g.Expect(err).ToNot(HaveOccurred())
				conditionType, err := conditions.UnstructuredGet(unstructuredServer, string(tt.wantCondition.Type))
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(conditionType).ToNot(BeNil())
				g.Expect(string(conditionType.Status)).To(Equal(string(tt.wantCondition.Status)))
				g.Expect(conditionType.Reason).To(Equal(tt.wantCondition.Reason))
				g.Expect(conditionType.Message).To(Equal(tt.wantCondition.Message))
			}
		})
	}
}

func TestOpenStackServerReconciler_getOrCreateServer(t *testing.T) {
	tests := []struct {
		name            string
		openStackServer *infrav1alpha1.OpenStackServer
		setupMocks      func(r *recorders)
		wantServer      *servers.Server
		wantErr         bool
		wantCondition   *clusterv1beta1.Condition
	}{
		{
			name: "instanceID set in status but server not found",
			openStackServer: &infrav1alpha1.OpenStackServer{
				Status: infrav1alpha1.OpenStackServerStatus{
					InstanceID: ptr.To(instanceUUID),
				},
			},
			setupMocks: func(r *recorders) {
				r.compute.GetServer(instanceUUID).Return(nil, gophercloud.ErrUnexpectedResponseCode{Actual: 404})
			},
			wantErr: false,
			wantCondition: &clusterv1beta1.Condition{
				Type:    infrav1.InstanceReadyCondition,
				Status:  corev1.ConditionFalse,
				Reason:  infrav1.InstanceNotFoundReason,
				Message: infrav1.ServerUnexpectedDeletedMessage,
			},
		},
		{
			name: "instanceID set in status but server not found with error",
			openStackServer: &infrav1alpha1.OpenStackServer{
				Status: infrav1alpha1.OpenStackServerStatus{
					InstanceID: ptr.To(instanceUUID),
				},
			},
			setupMocks: func(r *recorders) {
				r.compute.GetServer(instanceUUID).Return(nil, fmt.Errorf("error"))
			},
			wantErr: true,
			wantCondition: &clusterv1beta1.Condition{
				Type:    infrav1.InstanceReadyCondition,
				Status:  corev1.ConditionFalse,
				Reason:  infrav1.OpenStackErrorReason,
				Message: "get server \"" + instanceUUID + "\" detail failed: error",
			},
		},
		{
			name: "instanceStatus is nil but server found with machine name",
			openStackServer: &infrav1alpha1.OpenStackServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: openStackServerName,
				},
				Status: infrav1alpha1.OpenStackServerStatus{},
			},
			setupMocks: func(r *recorders) {
				r.compute.ListServers(servers.ListOpts{
					Name: "^" + openStackServerName + "$",
				}).Return([]servers.Server{{ID: instanceUUID}}, nil)
			},
			wantErr: false,
			wantServer: &servers.Server{
				ID: instanceUUID,
			},
		},
		{
			name: "instanceStatus is nil and server not found and then created",
			openStackServer: &infrav1alpha1.OpenStackServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: openStackServerName,
				},
				Status: infrav1alpha1.OpenStackServerStatus{
					Resolved: &infrav1alpha1.ResolvedServerSpec{
						ImageID:  imageUUID,
						FlavorID: flavorUUID,
						Ports:    defaultResolvedPorts,
					},
				},
			},
			setupMocks: func(r *recorders) {
				r.compute.ListServers(servers.ListOpts{
					Name: "^" + openStackServerName + "$",
				}).Return([]servers.Server{}, nil)
				r.compute.CreateServer(gomock.Any(), gomock.Any()).Return(&servers.Server{ID: instanceUUID}, nil)
			},
			wantErr: false,
			wantServer: &servers.Server{
				ID: instanceUUID,
			},
			// It's off but no condition is set because the server creation was kicked off but we
			// don't know the result yet in this function.
		},
		{
			name: "instanceStatus is nil and server not found and then created with error",
			openStackServer: &infrav1alpha1.OpenStackServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: openStackServerName,
				},
				Status: infrav1alpha1.OpenStackServerStatus{
					Resolved: &infrav1alpha1.ResolvedServerSpec{
						ImageID:  imageUUID,
						FlavorID: flavorUUID,
						Ports:    defaultResolvedPorts,
					},
				},
			},
			setupMocks: func(r *recorders) {
				r.compute.ListServers(servers.ListOpts{
					Name: "^" + openStackServerName + "$",
				}).Return([]servers.Server{}, nil)
				r.compute.CreateServer(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))
			},
			wantErr: true,
			wantCondition: &clusterv1beta1.Condition{
				Type:    infrav1.InstanceReadyCondition,
				Status:  corev1.ConditionFalse,
				Reason:  infrav1.InstanceCreateFailedReason,
				Message: "error creating Openstack instance: " + "error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			log := testr.New(t)

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			scopeWithLogger := scope.NewWithLogger(mockScopeFactory, log)

			computeRecorder := mockScopeFactory.ComputeClient.EXPECT()
			imageRecorder := mockScopeFactory.ImageClient.EXPECT()
			networkRecorder := mockScopeFactory.NetworkClient.EXPECT()
			volumeRecorder := mockScopeFactory.VolumeClient.EXPECT()

			recorders := &recorders{
				compute: computeRecorder,
				image:   imageRecorder,
				network: networkRecorder,
				volume:  volumeRecorder,
			}

			if tt.setupMocks != nil {
				tt.setupMocks(recorders)
			}

			computeService, err := compute.NewService(scopeWithLogger)
			g.Expect(err).ToNot(HaveOccurred())

			reconciler := OpenStackServerReconciler{}
			status, err := reconciler.getOrCreateServer(ctx, log, tt.openStackServer, computeService, []string{portUUID})

			// Check error result
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).ToNot(HaveOccurred())
			}

			// Check instance status
			if tt.wantServer != nil {
				g.Expect(status.ID()).To(Equal(tt.wantServer.ID))
			}

			// Check the condition is set correctly
			if tt.wantCondition != nil {
				// print openstackServer conditions
				for _, condition := range tt.openStackServer.Status.Conditions {
					t.Logf("Condition: %s, Status: %s, Reason: %s", condition.Type, condition.Status, condition.Reason)
				}
				unstructuredServer, err := tt.openStackServer.ToUnstructured()
				g.Expect(err).ToNot(HaveOccurred())
				conditionType, err := conditions.UnstructuredGet(unstructuredServer, string(tt.wantCondition.Type))
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(conditionType).ToNot(BeNil())
				g.Expect(string(conditionType.Status)).To(Equal(string(tt.wantCondition.Status)))
				g.Expect(conditionType.Reason).To(Equal(tt.wantCondition.Reason))
				g.Expect(conditionType.Message).To(Equal(tt.wantCondition.Message))
			}
		})
	}
}

var _ = Describe("OpenStackServer controller", func() {
	var (
		testServer        *infrav1alpha1.OpenStackServer
		testNamespace     string
		serverReconciler  *OpenStackServerReconciler
		serverMockCtrl    *gomock.Controller
		serverMockFactory *scope.MockScopeFactory
		testNum           int
	)

	BeforeEach(func() {
		testNum++
		testNamespace = fmt.Sprintf("server-test-%d", testNum)

		testServer = &infrav1alpha1.OpenStackServer{
			TypeMeta: metav1.TypeMeta{
				APIVersion: infrav1alpha1.SchemeGroupVersion.Group + "/" + infrav1alpha1.SchemeGroupVersion.Version,
				Kind:       "OpenStackServer",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-server",
				Namespace: testNamespace,
			},
			Spec: infrav1alpha1.OpenStackServerSpec{
				IdentityRef: infrav1.OpenStackIdentityReference{
					Name:      "test-creds",
					CloudName: "openstack",
				},
				Flavor:     ptr.To(defaultFlavor),
				Image:      defaultImage,
				SSHKeyName: "test-ssh-key",
				Ports: []infrav1.PortOpts{
					{
						Network: &infrav1.NetworkParam{
							ID: ptr.To(networkUUID),
						},
					},
				},
			},
		}

		input := framework.CreateNamespaceInput{
			Creator: k8sClient,
			Name:    testNamespace,
		}
		framework.CreateNamespace(ctx, input)

		serverMockCtrl = gomock.NewController(GinkgoT())
		serverMockFactory = scope.NewMockScopeFactory(serverMockCtrl, "")
		serverReconciler = &OpenStackServerReconciler{
			Client:       k8sClient,
			ScopeFactory: serverMockFactory,
		}
	})

	AfterEach(func() {
		orphan := metav1.DeletePropagationOrphan
		deleteOptions := client.DeleteOptions{
			PropagationPolicy: &orphan,
		}

		// Remove finalizers and delete openstackserver
		patchHelper, err := patch.NewHelper(testServer, k8sClient)
		Expect(err).To(BeNil())
		testServer.SetFinalizers([]string{})
		err = patchHelper.Patch(ctx, testServer)
		Expect(err).To(BeNil())
		err = k8sClient.Delete(ctx, testServer, &deleteOptions)
		Expect(err).To(BeNil())
	})

	It("should set OpenStackAuthenticationSucceededCondition to False when credentials secret is missing", func() {
		testServer.SetName("missing-server-credentials")
		testServer.Spec.IdentityRef = infrav1.OpenStackIdentityReference{
			Type:      "Secret",
			Name:      "non-existent-secret",
			CloudName: "openstack",
		}

		err := k8sClient.Create(ctx, testServer)
		Expect(err).To(BeNil())

		credentialsErr := fmt.Errorf("secret not found: non-existent-secret")
		serverMockFactory.SetClientScopeCreateError(credentialsErr)

		req := reconcile.Request{
			NamespacedName: client.ObjectKey{
				Name:      testServer.Name,
				Namespace: testServer.Namespace,
			},
		}
		result, err := serverReconciler.Reconcile(ctx, req)

		Expect(err).To(MatchError(credentialsErr))
		Expect(result).To(Equal(reconcile.Result{}))

		// Fetch the updated OpenStackServer to verify the condition was set
		updatedServer := &infrav1alpha1.OpenStackServer{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: testServer.Name, Namespace: testServer.Namespace}, updatedServer)).To(Succeed())

		// Verify OpenStackAuthenticationSucceededCondition is set to False
		Expect(v1beta1conditions.IsFalse(updatedServer, infrav1.OpenStackAuthenticationSucceeded)).To(BeTrue())
		condition := v1beta1conditions.Get(updatedServer, infrav1.OpenStackAuthenticationSucceeded)
		Expect(condition).ToNot(BeNil())
		Expect(condition.Reason).To(Equal(infrav1.OpenStackAuthenticationFailedReason))
		Expect(condition.Severity).To(Equal(clusterv1beta1.ConditionSeverityError))
		Expect(condition.Message).To(ContainSubstring("Failed to create OpenStack client scope"))
	})

	It("should set OpenStackAuthenticationSucceededCondition to False when namespace is denied access to ClusterIdentity", func() {
		testServer.SetName("identity-access-denied-server")
		testServer.Spec.IdentityRef = infrav1.OpenStackIdentityReference{
			Type:      "ClusterIdentity",
			Name:      "test-cluster-identity",
			CloudName: "openstack",
		}

		err := k8sClient.Create(ctx, testServer)
		Expect(err).To(BeNil())

		identityAccessErr := &scope.IdentityAccessDeniedError{
			IdentityName:       "test-cluster-identity",
			RequesterNamespace: testNamespace,
		}
		serverMockFactory.SetClientScopeCreateError(identityAccessErr)

		req := reconcile.Request{
			NamespacedName: client.ObjectKey{
				Name:      testServer.Name,
				Namespace: testServer.Namespace,
			},
		}
		result, err := serverReconciler.Reconcile(ctx, req)

		Expect(err).To(MatchError(identityAccessErr))
		Expect(result).To(Equal(reconcile.Result{}))

		// Fetch the updated OpenStackServer to verify the condition was set
		updatedServer := &infrav1alpha1.OpenStackServer{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: testServer.Name, Namespace: testServer.Namespace}, updatedServer)).To(Succeed())

		// Verify OpenStackAuthenticationSucceededCondition is set to False
		Expect(v1beta1conditions.IsFalse(updatedServer, infrav1.OpenStackAuthenticationSucceeded)).To(BeTrue())
		condition := v1beta1conditions.Get(updatedServer, infrav1.OpenStackAuthenticationSucceeded)
		Expect(condition).ToNot(BeNil())
		Expect(condition.Reason).To(Equal(infrav1.OpenStackAuthenticationFailedReason))
		Expect(condition.Severity).To(Equal(clusterv1beta1.ConditionSeverityError))
		Expect(condition.Message).To(ContainSubstring("Failed to create OpenStack client scope"))
	})
})
