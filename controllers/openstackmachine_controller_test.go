/*
Copyright 2020 The Kubernetes Authors.

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
	"github.com/google/go-cmp/cmp"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	. "github.com/onsi/gomega" //nolint:revive
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/compute"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

const (
	networkUUID                   = "d412171b-9fd7-41c1-95a6-c24e5953974d"
	subnetUUID                    = "d2d8d98d-b234-477e-a547-868b7cb5d6a5"
	extraSecurityGroupUUID        = "514bb2d8-3390-4a3b-86a7-7864ba57b329"
	controlPlaneSecurityGroupUUID = "c9817a91-4821-42db-8367-2301002ab659"
	workerSecurityGroupUUID       = "9c6c0d28-03c9-436c-815d-58440ac2c1c8"
	serverGroupUUID               = "7b940d62-68ef-4e42-a76a-1a62e290509c"
	imageUUID                     = "ce96e584-7ebc-46d6-9e55-987d72e3806c"

	openStackMachineName = "test-openstack-machine"
	namespace            = "test-namespace"
	flavorName           = "test-flavor"
	sshKeyName           = "test-ssh-key"
	failureDomain        = "test-failure-domain"
)

func getDefaultOpenStackCluster() *infrav1.OpenStackCluster {
	return &infrav1.OpenStackCluster{
		Spec: infrav1.OpenStackClusterSpec{},
		Status: infrav1.OpenStackClusterStatus{
			Network: &infrav1.NetworkStatusWithSubnets{
				NetworkStatus: infrav1.NetworkStatus{
					ID: networkUUID,
				},
				Subnets: []infrav1.Subnet{
					{ID: subnetUUID},
				},
			},
			ControlPlaneSecurityGroup: &infrav1.SecurityGroupStatus{ID: controlPlaneSecurityGroupUUID},
			WorkerSecurityGroup:       &infrav1.SecurityGroupStatus{ID: workerSecurityGroupUUID},
		},
	}
}

func getDefaultMachine() *clusterv1.Machine {
	return &clusterv1.Machine{
		Spec: clusterv1.MachineSpec{
			FailureDomain: ptr.To(failureDomain),
		},
	}
}

func getDefaultOpenStackMachine() *infrav1.OpenStackMachine {
	return &infrav1.OpenStackMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      openStackMachineName,
			Namespace: namespace,
		},
		Spec: infrav1.OpenStackMachineSpec{
			// ProviderID is set by the controller
			// InstanceID is set by the controller
			// FloatingIP is only used by the cluster controller for the Bastion
			// TODO: Test Networks, Ports, Subnet, and Trunk separately
			Flavor:     flavorName,
			Image:      infrav1.ImageParam{ID: ptr.To(imageUUID)},
			SSHKeyName: sshKeyName,
			Tags:       []string{"test-tag"},
			ServerMetadata: []infrav1.ServerMetadata{
				{Key: "test-metadata", Value: "test-value"},
			},
			ConfigDrive:    ptr.To(true),
			SecurityGroups: []infrav1.SecurityGroupParam{},
			ServerGroup:    &infrav1.ServerGroupParam{ID: ptr.To(serverGroupUUID)},
		},
		Status: infrav1.OpenStackMachineStatus{
			Resolved: &infrav1.ResolvedMachineSpec{
				ImageID:       imageUUID,
				ServerGroupID: serverGroupUUID,
			},
		},
	}
}

func getDefaultInstanceSpec() *compute.InstanceSpec {
	return &compute.InstanceSpec{
		Name:       openStackMachineName,
		ImageID:    imageUUID,
		Flavor:     flavorName,
		SSHKeyName: sshKeyName,
		UserData:   "user-data",
		Metadata: map[string]string{
			"test-metadata": "test-value",
		},
		ConfigDrive:   *ptr.To(true),
		FailureDomain: *ptr.To(failureDomain),
		ServerGroupID: serverGroupUUID,
		Tags:          []string{"test-tag"},
	}
}

func Test_machineToInstanceSpec(t *testing.T) {
	RegisterTestingT(t)

	tests := []struct {
		name             string
		openStackCluster func() *infrav1.OpenStackCluster
		machine          func() *clusterv1.Machine
		openStackMachine func() *infrav1.OpenStackMachine
		wantInstanceSpec func() *compute.InstanceSpec
	}{
		{
			name:             "Defaults",
			openStackCluster: getDefaultOpenStackCluster,
			machine:          getDefaultMachine,
			openStackMachine: getDefaultOpenStackMachine,
			wantInstanceSpec: getDefaultInstanceSpec,
		},
		{
			name: "Tags",
			openStackCluster: func() *infrav1.OpenStackCluster {
				c := getDefaultOpenStackCluster()
				c.Spec.Tags = []string{"cluster-tag", "duplicate-tag"}
				return c
			},
			machine: getDefaultMachine,
			openStackMachine: func() *infrav1.OpenStackMachine {
				m := getDefaultOpenStackMachine()
				m.Spec.Tags = []string{"machine-tag", "duplicate-tag"}
				return m
			},
			wantInstanceSpec: func() *compute.InstanceSpec {
				i := getDefaultInstanceSpec()
				i.Tags = []string{"machine-tag", "duplicate-tag", "cluster-tag"}
				return i
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			got, _ := machineToInstanceSpec(tt.openStackCluster(), tt.machine(), tt.openStackMachine(), "user-data")
			wanted := tt.wantInstanceSpec()

			g.Expect(got).To(Equal(wanted), cmp.Diff(got, wanted))
		})
	}
}

func TestGetPortIDs(t *testing.T) {
	tests := []struct {
		name  string
		ports []infrav1.PortStatus
		want  []string
	}{
		{
			name:  "Empty ports",
			ports: []infrav1.PortStatus{},
			want:  []string{},
		},
		{
			name: "Single port",
			ports: []infrav1.PortStatus{
				{ID: "port1"},
			},
			want: []string{"port1"},
		},
		{
			name: "Multiple ports",
			ports: []infrav1.PortStatus{
				{ID: "port1"},
				{ID: "port2"},
				{ID: "port3"},
			},
			want: []string{"port1", "port2", "port3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetPortIDs(tt.ports)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPortIDs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_reconcileDelete(t *testing.T) {
	const (
		instanceUUID   = "8308882f-5e46-47e6-8e12-1fe869c43d1d"
		portUUID       = "55eac199-4836-4a98-b31c-9f65f382ad46"
		rootVolumeUUID = "4724a66d-bd5e-47f3-bb57-a67fcb4168e0"
		trunkUUID      = "9d348baa-93b1-4e63-932f-dd0527fbd789"

		imageName = "my-image"
	)

	// *******************
	// START OF TEST CASES
	// *******************

	type recorders struct {
		compute *mock.MockComputeClientMockRecorder
		image   *mock.MockImageClientMockRecorder
		network *mock.MockNetworkClientMockRecorder
		volume  *mock.MockVolumeClientMockRecorder
	}

	defaultImage := infrav1.ImageParam{
		Filter: &infrav1.ImageFilter{
			Name: ptr.To(imageName),
		},
	}

	defaultResolvedPorts := []infrav1.ResolvedPortSpec{
		{
			Name:        openStackMachineName + "-0",
			Description: "my test port",
			NetworkID:   networkUUID,
		},
	}
	defaultPortsStatus := []infrav1.PortStatus{
		{
			ID: portUUID,
		},
	}

	deleteDefaultPorts := func(r *recorders) {
		trunkExtension := extensions.Extension{}
		trunkExtension.Alias = "trunk"
		r.network.ListExtensions().Return([]extensions.Extension{trunkExtension}, nil)
		r.network.ListTrunk(trunks.ListOpts{PortID: portUUID}).Return([]trunks.Trunk{{ID: trunkUUID}}, nil)
		r.network.ListTrunkSubports(trunkUUID).Return([]trunks.Subport{}, nil)
		r.network.DeleteTrunk(trunkUUID).Return(nil)
		r.network.DeletePort(portUUID).Return(nil)
	}

	deleteServerByID := func(r *recorders) {
		r.compute.GetServer(instanceUUID).Return(&clients.ServerExt{
			Server: servers.Server{
				ID:   instanceUUID,
				Name: openStackMachineName,
			},
		}, nil)
		r.compute.DeleteServer(instanceUUID).Return(nil)
		r.compute.GetServer(instanceUUID).Return(nil, gophercloud.ErrDefault404{})
	}
	deleteServerByName := func(r *recorders) {
		r.compute.ListServers(servers.ListOpts{
			Name: "^" + openStackMachineName + "$",
		}).Return([]clients.ServerExt{
			{Server: servers.Server{
				ID:   instanceUUID,
				Name: openStackMachineName,
			}},
		}, nil)
		r.compute.DeleteServer(instanceUUID).Return(nil)
		r.compute.GetServer(instanceUUID).Return(nil, gophercloud.ErrDefault404{})
	}

	deleteMissingServerByName := func(r *recorders) {
		// Lookup server by name because it is not in status.
		// Don't find it.
		r.compute.ListServers(servers.ListOpts{
			Name: "^" + openStackMachineName + "$",
		}).Return([]clients.ServerExt{}, nil)
	}

	deleteRootVolume := func(r *recorders) {
		// Fetch volume by name
		volumeName := fmt.Sprintf("%s-root", openStackMachineName)
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

	adoptExistingPorts := func(r *recorders) {
		r.network.ListPort(ports.ListOpts{
			NetworkID: networkUUID,
			Name:      openStackMachineName + "-0",
		}).Return([]ports.Port{{ID: portUUID}}, nil)
	}

	resolveImage := func(r *recorders) {
		r.image.ListImages(images.ListOpts{
			Name: imageName,
		}).Return([]images.Image{{ID: imageUUID}}, nil)
	}

	tests := []struct {
		name                string
		osMachine           infrav1.OpenStackMachine
		expect              func(r *recorders)
		wantErr             bool
		wantRemoveFinalizer bool
		clusterNotReady     bool
	}{
		{
			name: "No volumes, resolved and resources populated",
			osMachine: infrav1.OpenStackMachine{
				Spec: infrav1.OpenStackMachineSpec{
					Image: defaultImage,
				},
				Status: infrav1.OpenStackMachineStatus{
					InstanceID: ptr.To(instanceUUID),
					Resolved: &infrav1.ResolvedMachineSpec{
						ImageID: imageUUID,
						Ports:   defaultResolvedPorts,
					},
					Resources: &infrav1.MachineResources{
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
			osMachine: infrav1.OpenStackMachine{
				Spec: infrav1.OpenStackMachineSpec{
					Image: defaultImage,
					RootVolume: &infrav1.RootVolume{
						SizeGiB: 50,
					},
				},
				Status: infrav1.OpenStackMachineStatus{
					InstanceID: ptr.To(instanceUUID),
					Resolved: &infrav1.ResolvedMachineSpec{
						ImageID: imageUUID,
						Ports:   defaultResolvedPorts,
					},
					Resources: &infrav1.MachineResources{
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
			name: "Root volume, machine not created, resolved and resources populated",
			osMachine: infrav1.OpenStackMachine{
				Spec: infrav1.OpenStackMachineSpec{
					Image: defaultImage,
					RootVolume: &infrav1.RootVolume{
						SizeGiB: 50,
					},
				},
				Status: infrav1.OpenStackMachineStatus{
					Resolved: &infrav1.ResolvedMachineSpec{
						ImageID: imageUUID,
						Ports:   defaultResolvedPorts,
					},
					Resources: &infrav1.MachineResources{
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
			// N.B. The 'no resolved but resource exist' case can
			// only happen across an upgrade. At some point in the
			// future we should stop handling it.
			name: "No volumes, no resolved or resources, instance exists",
			osMachine: infrav1.OpenStackMachine{
				Spec: infrav1.OpenStackMachineSpec{
					Image: defaultImage,
				},
				Status: infrav1.OpenStackMachineStatus{
					// Unlike resolved and resources,
					// instanceID will have been converted
					// from the previous API version.
					InstanceID: ptr.To(instanceUUID),
				},
			},
			expect: func(r *recorders) {
				resolveImage(r)
				adoptExistingPorts(r)
				deleteServerByID(r)
				deleteDefaultPorts(r)
			},
			wantRemoveFinalizer: true,
		},
		{
			// This is an upgrade case because from v0.10 onwards
			// we don't add the finalizer until we add resolved, so
			// this can no longer occur. This will stop working when
			// we remove handling for empty resolved on delete.
			name: "Invalid image, no resolved or resources",
			osMachine: infrav1.OpenStackMachine{
				Spec: infrav1.OpenStackMachineSpec{
					Image: defaultImage,
				},
			},
			expect: func(r *recorders) {
				r.image.ListImages(images.ListOpts{Name: imageName}).Return([]images.Image{}, nil)
			},
			wantErr:             true,
			wantRemoveFinalizer: true,
		},
		{
			name: "No instance id, server and ports exist",
			osMachine: infrav1.OpenStackMachine{
				Spec: infrav1.OpenStackMachineSpec{
					Image: defaultImage,
				},
				Status: infrav1.OpenStackMachineStatus{
					Resolved: &infrav1.ResolvedMachineSpec{
						ImageID: imageUUID,
						Ports:   defaultResolvedPorts,
					},
					Resources: &infrav1.MachineResources{
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
			osMachine: infrav1.OpenStackMachine{
				Spec: infrav1.OpenStackMachineSpec{
					Image: defaultImage,
				},
				Status: infrav1.OpenStackMachineStatus{
					Resolved: &infrav1.ResolvedMachineSpec{
						ImageID: imageUUID,
						Ports:   defaultResolvedPorts,
					},
				},
			},
			expect: func(r *recorders) {
				r.network.ListPort(ports.ListOpts{
					NetworkID: networkUUID,
					Name:      openStackMachineName + "-0",
				}).Return(nil, fmt.Errorf("error adopting ports"))
			},
			wantErr:             true,
			wantRemoveFinalizer: false,
		},
		{
			// This is an upgrade case because from v0.10 onwards we
			// should not have added the finalizer until the cluster
			// is ready.
			name: "Cluster not ready should remove finalizer",
			osMachine: infrav1.OpenStackMachine{
				Spec: infrav1.OpenStackMachineSpec{
					Image: defaultImage,
				},
			},
			clusterNotReady:     true,
			wantRemoveFinalizer: true,
		},
	}
	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			log := testr.New(t)

			mockCtrl := gomock.NewController(t)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")

			reconciler := OpenStackMachineReconciler{}

			computeRecorder := mockScopeFactory.ComputeClient.EXPECT()
			imageRecorder := mockScopeFactory.ImageClient.EXPECT()
			networkRecorder := mockScopeFactory.NetworkClient.EXPECT()
			volumeRecorder := mockScopeFactory.VolumeClient.EXPECT()

			if tt.expect != nil {
				tt.expect(&recorders{computeRecorder, imageRecorder, networkRecorder, volumeRecorder})
			}
			scopeWithLogger := scope.NewWithLogger(mockScopeFactory, log)

			openStackCluster := infrav1.OpenStackCluster{}
			openStackCluster.Status.Ready = !tt.clusterNotReady
			openStackCluster.Status.Network = &infrav1.NetworkStatusWithSubnets{
				NetworkStatus: infrav1.NetworkStatus{
					Name: "my-network",
					ID:   networkUUID,
				},
				Subnets: []infrav1.Subnet{
					{
						Name: "my-subnet",
						ID:   subnetUUID,
						CIDR: "192.168.0.0/24",
					},
				},
			}

			machine := clusterv1.Machine{}

			osMachine := &tt.osMachine
			osMachine.Name = openStackMachineName
			osMachine.Finalizers = []string{infrav1.MachineFinalizer}

			_, err := reconciler.reconcileDelete(ctx, scopeWithLogger, openStackMachineName, &openStackCluster, &machine, &tt.osMachine)

			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).ToNot(HaveOccurred())
			}

			if tt.wantRemoveFinalizer {
				g.Expect(osMachine.Finalizers).To(BeEmpty())
			} else {
				g.Expect(osMachine.Finalizers).To(ConsistOf(infrav1.MachineFinalizer))
			}
		})
	}
}
