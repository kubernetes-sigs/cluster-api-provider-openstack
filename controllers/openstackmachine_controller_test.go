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
	"reflect"
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	. "github.com/onsi/gomega" //nolint:revive
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	infrav1alpha1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
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

func TestOpenStackMachineSpecToOpenStackServerSpec(t *testing.T) {
	resolvedMachineSpec := &infrav1.ResolvedMachineSpec{
		ServerGroupID: serverGroupUUID,
		ImageID:       imageUUID,
		Ports: []infrav1.ResolvedPortSpec{
			{
				Name:      "port1",
				NetworkID: networkUUID,
			},
		},
	}
	identityRef := infrav1.OpenStackIdentityReference{
		Name:      "foo",
		CloudName: "my-cloud",
	}
	tags := []string{"tag1", "tag2"}
	failureDomain := "failure-domain"
	userData := &corev1.LocalObjectReference{Name: "server-data-secret"}
	tests := []struct {
		name string
		spec *infrav1.OpenStackMachineSpec
		want *infrav1alpha1.OpenStackServerSpec
	}{
		{
			name: "Test a minimum OpenStackMachineSpec to OpenStackServerSpec conversion",
			spec: &infrav1.OpenStackMachineSpec{
				Flavor:     flavorName,
				Image:      infrav1.ImageParam{Filter: &infrav1.ImageFilter{Name: ptr.To("my-image")}},
				SSHKeyName: sshKeyName,
			},
			want: &infrav1alpha1.OpenStackServerSpec{
				AvailabilityZone: ptr.To("failure-domain"),
				Flavor:           flavorName,
				IdentityRef:      identityRef,
				Image:            infrav1.ImageParam{ID: ptr.To(imageUUID)},
				SSHKeyName:       sshKeyName,
				Ports: []infrav1.PortOpts{
					{
						Network: &infrav1.NetworkParam{
							ID: ptr.To(networkUUID),
						},
					},
				},
				ServerGroup: &infrav1.ServerGroupParam{
					ID: ptr.To(serverGroupUUID),
				},
				Tags:        tags,
				UserDataRef: userData,
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			spec, err := openStackMachineSpecToOpenStackServerSpec(tt.spec, resolvedMachineSpec, identityRef, tags, failureDomain, userData)
			if err == nil && !reflect.DeepEqual(spec, tt.want) {
				t.Errorf("openStackMachineSpecToOpenStackServerSpec() got = %+v, want %+v", spec, tt.want)
			}
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

	tests := []struct {
		name                string
		osMachine           infrav1.OpenStackMachine
		expect              func(r *recorders)
		wantErr             bool
		wantRemoveFinalizer bool
		clusterNotReady     bool
	}{
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
