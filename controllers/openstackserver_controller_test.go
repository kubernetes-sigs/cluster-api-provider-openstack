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
	"fmt"
	"reflect"
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	. "github.com/onsi/gomega" //nolint:revive
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

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

var getDefaultFlavor = func(r *recorders) {
	f := flavors.Flavor{
		Name: defaultFlavor,
	}
	r.compute.GetFlavorFromName(defaultFlavor).Return(&f, nil)
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
					Flavor:           "large",
					Resolved: &infrav1alpha1.ResolvedServerSpec{
						ImageID:       "123",
						ServerGroupID: "456",
					},
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
				Flavor:        "large",
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
					Flavor: defaultFlavor,
					Image:  defaultImage,
					Ports:  defaultPortOpts,
					Resolved: &infrav1alpha1.ResolvedServerSpec{
						ImageID: imageUUID,
						Ports:   defaultResolvedPorts,
					},
					Resources: &infrav1alpha1.ServerResources{
						Ports: defaultPortsStatus,
					},
				},
				Status: infrav1alpha1.OpenStackServerStatus{
					InstanceID: ptr.To(instanceUUID),
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
					Resolved: &infrav1alpha1.ResolvedServerSpec{
						ImageID: imageUUID,
						Ports:   defaultResolvedPorts,
					},
					Resources: &infrav1alpha1.ServerResources{
						Ports: defaultPortsStatus,
					},
				},
				Status: infrav1alpha1.OpenStackServerStatus{
					InstanceID: ptr.To(instanceUUID),
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
		name     string
		osServer infrav1alpha1.OpenStackServer
		expect   func(r *recorders)
	}{
		{
			name: "Minimal server spec creating port and server",
			osServer: infrav1alpha1.OpenStackServer{
				Spec: infrav1alpha1.OpenStackServerSpec{
					Flavor: defaultFlavor,
					Image:  defaultImage,
					Ports:  defaultPortOpts,
					Resolved: &infrav1alpha1.ResolvedServerSpec{
						ImageID: imageUUID,
						Ports:   defaultResolvedPorts,
					},
				},
			},
			expect: func(r *recorders) {
				listDefaultPortsNotFound(r)
				createDefaultPort(r)
				getDefaultFlavor(r)
				listDefaultServerNotFound(r)
				createDefaultServer(r)
			},
		},
		{
			name: "Minimum server spec adopting port and server",
			osServer: infrav1alpha1.OpenStackServer{
				Spec: infrav1alpha1.OpenStackServerSpec{
					Flavor: defaultFlavor,
					Image:  defaultImage,
					Ports:  defaultPortOpts,
					Resolved: &infrav1alpha1.ResolvedServerSpec{
						ImageID: imageUUID,
						Ports:   defaultResolvedPorts,
					},
				},
			},
			expect: func(r *recorders) {
				listDefaultPorts(r)
				listDefaultServerFound(r)
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
			g.Expect(err).ToNot(HaveOccurred())
		})
	}
}
