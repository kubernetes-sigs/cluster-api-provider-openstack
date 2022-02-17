/*
Copyright 2022 The Kubernetes Authors.

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

package compute

import (
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	gomock "github.com/golang/mock/gomock"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	. "github.com/onsi/gomega"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking/mock_networking"
)

func TestService_reconcileRootVolume(t *testing.T) {
	RegisterTestingT(t)

	const (
		volumeUUID = "de6488ac-2f08-4715-8a53-1bcf053927ec"
		imageUUID  = "bc85a3da-b12c-476c-97b8-f49d12e4324f"
	)

	tests := []struct {
		name           string
		rootVolume     infrav1.RootVolume
		resources      infrav1.OpenStackMachineResources
		wantResources  infrav1.OpenStackMachineResources
		instanceStatus *InstanceStatus
		expect         func(computeRecorder *MockClientMockRecorder)
		wantReconciled bool
		wantErr        bool
	}{
		{
			name:          "No root volume",
			rootVolume:    infrav1.RootVolume{},
			resources:     infrav1.OpenStackMachineResources{},
			wantResources: infrav1.OpenStackMachineResources{},
			expect: func(computeRecorder *MockClientMockRecorder) {
			},
			wantReconciled: true,
		},
		{
			name: "Create root volume",
			rootVolume: infrav1.RootVolume{
				Size:             10,
				VolumeType:       "test-volume-type",
				AvailabilityZone: "test-az",
			},
			resources: infrav1.OpenStackMachineResources{},
			wantResources: infrav1.OpenStackMachineResources{
				RootVolume: infrav1.OpenStackResource{
					ID:    volumeUUID,
					Ready: false,
				},
			},
			expect: func(computeRecorder *MockClientMockRecorder) {
				computeRecorder.ListVolumes(volumes.ListOpts{
					Name: "test-instance-root",
				}).Return([]volumes.Volume{}, nil)

				computeRecorder.CreateVolume(volumes.CreateOpts{
					Size:             10,
					AvailabilityZone: "test-az",
					Description:      "Root volume for test-instance",
					Name:             "test-instance-root",
					ImageID:          imageUUID,
					VolumeType:       "test-volume-type",
					Multiattach:      false,
				}).Return(&volumes.Volume{
					ID:               volumeUUID,
					Status:           "creating",
					Size:             10,
					AvailabilityZone: "test-az",
					Name:             "test-instance-root",
					Description:      "Root volume for test-instance",
					VolumeType:       "test-volume-type",
					Bootable:         "true",
					Encrypted:        false,
					Multiattach:      false,
				}, nil)
			},
			wantReconciled: false,
		},
		{
			name: "Poll creating volume",
			rootVolume: infrav1.RootVolume{
				Size:             10,
				VolumeType:       "test-volume-type",
				AvailabilityZone: "test-az",
			},
			resources: infrav1.OpenStackMachineResources{
				RootVolume: infrav1.OpenStackResource{
					ID:    volumeUUID,
					Ready: false,
				},
			},
			wantResources: infrav1.OpenStackMachineResources{
				RootVolume: infrav1.OpenStackResource{
					ID:    volumeUUID,
					Ready: true,
				},
			},
			expect: func(computeRecorder *MockClientMockRecorder) {
				computeRecorder.GetVolume(volumeUUID).Return(&volumes.Volume{
					ID:               volumeUUID,
					Status:           "available",
					Size:             10,
					AvailabilityZone: "test-az",
					Attachments:      []volumes.Attachment{},
					Name:             "test-instance-root",
					Description:      "Root volume for test-instance",
					VolumeType:       "test-volume-type",
					Bootable:         "true",
					Multiattach:      false,
				}, nil)
			},
			wantReconciled: true,
		},
		// TODO:
		// - Error creating volume
		// - Error polling volume
		// - Polled volume in error state
		// - Adopt volume for existing instance
		// - Adopt volume with same name, size
		// - Reject volume with different size
		// - Reject volume attached to another instance
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			mockComputeClient := NewMockClient(mockCtrl)
			mockNetworkClient := mock_networking.NewMockNetworkClient(mockCtrl)

			tt.expect(mockComputeClient.EXPECT())

			s := Service{
				projectID:      "",
				computeService: mockComputeClient,
				networkingService: networking.NewTestService(
					"", mockNetworkClient, logr.Discard(),
				),
				logger: logr.Discard(),
			}

			instanceSpec := InstanceSpec{
				Name:       "test-instance",
				ImageUUID:  imageUUID,
				RootVolume: &tt.rootVolume,
			}

			reconciled, err := s.reconcileRootVolume(&infrav1.OpenStackMachine{}, &instanceSpec, &tt.resources, tt.instanceStatus)
			if tt.wantErr {
				Expect(err).To(HaveOccurred(), fmt.Sprintf("Service.reconcileRootVolume() error = %v, wantErr %v", err, tt.wantErr))
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(reconciled).To(Equal(tt.wantReconciled), fmt.Sprintf("Service.reconcileRootVolume() = %v, wantReconcile %v", reconciled, tt.wantReconciled))
		})
	}
}
