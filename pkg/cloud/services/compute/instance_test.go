/*
Copyright 2018 The Kubernetes Authors.

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
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr/testr"
	"github.com/golang/mock/gomock"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

func TestService_getImageID(t *testing.T) {
	imageID := "ce96e584-7ebc-46d6-9e55-987d72e3806c"
	imageName := "test-image"
	imageTags := []string{"test-tag"}

	tests := []struct {
		testName string
		image    infrav1.ImageParam
		expect   func(m *mock.MockImageClientMockRecorder)
		want     string
		wantErr  bool
	}{
		{
			testName: "Return image ID when ID given",
			image:    infrav1.ImageParam{ID: &imageID},
			want:     imageID,
			expect:   func(m *mock.MockImageClientMockRecorder) {},
			wantErr:  false,
		},
		{
			testName: "Return image ID when name given",
			image: infrav1.ImageParam{
				Filter: &infrav1.ImageFilter{
					Name: &imageName,
				},
			},
			want: imageID,
			expect: func(m *mock.MockImageClientMockRecorder) {
				m.ListImages(images.ListOpts{Name: imageName}).Return(
					[]images.Image{{ID: imageID, Name: imageName}},
					nil)
			},
			wantErr: false,
		},
		{
			testName: "Return image ID when tags given",
			image: infrav1.ImageParam{
				Filter: &infrav1.ImageFilter{
					Tags: imageTags,
				},
			},
			want: imageID,
			expect: func(m *mock.MockImageClientMockRecorder) {
				m.ListImages(images.ListOpts{Tags: imageTags}).Return(
					[]images.Image{{ID: imageID, Name: imageName, Tags: imageTags}},
					nil)
			},
			wantErr: false,
		},
		{
			testName: "Return no results",
			image: infrav1.ImageParam{
				Filter: &infrav1.ImageFilter{
					Name: &imageName,
				},
			},
			expect: func(m *mock.MockImageClientMockRecorder) {
				m.ListImages(images.ListOpts{Name: imageName}).Return(
					[]images.Image{},
					nil)
			},
			want:    "",
			wantErr: true,
		},
		{
			testName: "Return multiple results",
			image: infrav1.ImageParam{
				Filter: &infrav1.ImageFilter{
					Name: &imageName,
				},
			},
			expect: func(m *mock.MockImageClientMockRecorder) {
				m.ListImages(images.ListOpts{Name: "test-image"}).Return(
					[]images.Image{
						{ID: imageID, Name: "test-image"},
						{ID: "123", Name: "test-image"},
					}, nil)
			},
			want:    "",
			wantErr: true,
		},
		{
			testName: "OpenStack returns error",
			image: infrav1.ImageParam{
				Filter: &infrav1.ImageFilter{
					Name: &imageName,
				},
			},
			expect: func(m *mock.MockImageClientMockRecorder) {
				m.ListImages(images.ListOpts{Name: "test-image"}).Return(
					nil,
					fmt.Errorf("test error"))
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			log := testr.New(t)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")

			s, err := NewService(scope.NewWithLogger(mockScopeFactory, log))
			if err != nil {
				t.Fatalf("Failed to create service: %v", err)
			}
			tt.expect(mockScopeFactory.ImageClient.EXPECT())

			got, err := s.GetImageID(tt.image)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.getImageID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Service.getImageID() = %v, want %v", got, tt.want)
			}
		})
	}
}

var portUUIDs = []string{"e7b7f3d1-0a81-40b1-bfa6-a22a31b17816"}

const (
	networkUUID                     = "d412171b-9fd7-41c1-95a6-c24e5953974d"
	subnetUUID                      = "d2d8d98d-b234-477e-a547-868b7cb5d6a5"
	portUUID                        = "e7b7f3d1-0a81-40b1-bfa6-a22a31b17816"
	trunkUUID                       = "2cf74a9f-3e89-4546-9779-20f2503c4ae8"
	imageUUID                       = "652b5a05-27fa-41d4-ac82-3e63cf6f7ab7"
	flavorUUID                      = "6dc820db-f912-454e-a1e3-1081f3b8cc72"
	instanceUUID                    = "383a8ec1-b6ea-4493-99dd-fc790da04ba9"
	controlPlaneSecurityGroupUUID   = "c9817a91-4821-42db-8367-2301002ab659"
	workerSecurityGroupUUID         = "9c6c0d28-03c9-436c-815d-58440ac2c1c8"
	serverGroupUUID                 = "7b940d62-68ef-4e42-a76a-1a62e290509c"
	rootVolumeUUID                  = "d84fe775-e25d-4f80-9888-f701e996c689"
	additionalBlockDeviceVolumeUUID = "1d1d5a56-c433-41dd-8446-cba2077e96e9"

	openStackMachineName = "test-openstack-machine"
	portName             = "test-openstack-machine-0"
	imageName            = "test-image"
	flavorName           = "test-flavor"
	sshKeyName           = "test-ssh-key"
	failureDomain        = "test-failure-domain"
)

func getDefaultInstanceSpec() *InstanceSpec {
	return &InstanceSpec{
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

func TestService_ReconcileInstance(t *testing.T) {
	RegisterTestingT(t)

	getDefaultServerMap := func() map[string]interface{} {
		// Add base64 user data to the create options the same way gophercloud does
		userData := base64.StdEncoding.EncodeToString([]byte("user-data"))

		return map[string]interface{}{
			"server": map[string]interface{}{
				"name":              openStackMachineName,
				"imageRef":          imageUUID,
				"flavorRef":         flavorUUID,
				"availability_zone": failureDomain,
				"networks": []map[string]interface{}{
					{"port": portUUID},
				},
				"config_drive": true,
				"key_name":     sshKeyName,
				"tags":         []interface{}{"test-tag"},
				"metadata": map[string]interface{}{
					"test-metadata": "test-value",
				},
				"user_data": &userData,
				"block_device_mapping_v2": []map[string]interface{}{
					{
						"delete_on_termination": true,
						"destination_type":      "local",
						"source_type":           "image",
						"uuid":                  imageUUID,
						"boot_index":            float64(0),
					},
				},
			},
			"os:scheduler_hints": map[string]interface{}{
				"group": serverGroupUUID,
			},
		}
	}

	returnedServer := func(status string) *clients.ServerExt {
		return &clients.ServerExt{
			Server: servers.Server{
				ID:      instanceUUID,
				Name:    openStackMachineName,
				Status:  status,
				KeyName: sshKeyName,
			},
			ServerAvailabilityZoneExt: availabilityzones.ServerAvailabilityZoneExt{},
		}
	}

	// Expected calls when using default flavor
	expectDefaultFlavor := func(computeRecorder *mock.MockComputeClientMockRecorder) {
		f := flavors.Flavor{
			ID:    flavorUUID,
			VCPUs: 2,
		}
		computeRecorder.GetFlavorFromName(flavorName).Return(&f, nil)
	}

	// Expected calls and custom match function for creating a server
	expectCreateServer := func(computeRecorder *mock.MockComputeClientMockRecorder, expectedCreateOpts map[string]interface{}, wantError bool) {
		// This nonsense is because ConfigDrive is a bool pointer, so we
		// can't assert its exact contents with gomock.
		// Instead we call ToServerCreateMap() on it to obtain a
		// map[string]interface{} of the create options, and then use
		// gomega to assert the contents of the map, which is more flexible.

		computeRecorder.CreateServer(gomock.Any()).DoAndReturn(func(createOpts servers.CreateOptsBuilder) (*clients.ServerExt, error) {
			optsMap, err := createOpts.ToServerCreateMap()
			Expect(err).NotTo(HaveOccurred())

			Expect(optsMap).To(Equal(expectedCreateOpts))

			if wantError {
				return nil, fmt.Errorf("test error")
			}
			return returnedServer("BUILDING"), nil
		})
	}

	returnedVolume := func(uuid string, status string) *volumes.Volume {
		return &volumes.Volume{
			ID:     uuid,
			Status: status,
		}
	}

	// Expected calls when polling for server creation
	expectVolumePoll := func(volumeRecorder *mock.MockVolumeClientMockRecorder, uuid string, states []string) {
		for _, state := range states {
			volumeRecorder.GetVolume(uuid).Return(returnedVolume(uuid, state), nil)
		}
	}

	expectVolumePollSuccess := func(volumeRecorder *mock.MockVolumeClientMockRecorder, uuid string) {
		expectVolumePoll(volumeRecorder, uuid, []string{"available"})
	}

	// *******************
	// START OF TEST CASES
	// *******************

	type recorders struct {
		compute *mock.MockComputeClientMockRecorder
		image   *mock.MockImageClientMockRecorder
		network *mock.MockNetworkClientMockRecorder
		volume  *mock.MockVolumeClientMockRecorder
	}

	tests := []struct {
		name            string
		getInstanceSpec func() *InstanceSpec
		expect          func(r *recorders)
		wantErr         bool
	}{
		{
			name:            "Defaults",
			getInstanceSpec: getDefaultInstanceSpec,
			expect: func(r *recorders) {
				expectDefaultFlavor(r.compute)

				expectCreateServer(r.compute, getDefaultServerMap(), false)
			},
			wantErr: false,
		},
		{
			name: "Boot from volume success",
			getInstanceSpec: func() *InstanceSpec {
				s := getDefaultInstanceSpec()
				s.RootVolume = &infrav1.RootVolume{
					SizeGiB: 50,
				}
				return s
			},
			expect: func(r *recorders) {
				expectDefaultFlavor(r.compute)

				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-root", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:        50,
					Description: fmt.Sprintf("Root volume for %s", openStackMachineName),
					Name:        fmt.Sprintf("%s-root", openStackMachineName),
					ImageID:     imageUUID,
					Multiattach: false,
				}).Return(&volumes.Volume{ID: rootVolumeUUID}, nil)
				expectVolumePollSuccess(r.volume, rootVolumeUUID)

				createMap := getDefaultServerMap()
				serverMap := createMap["server"].(map[string]interface{})
				serverMap["imageRef"] = ""
				serverMap["block_device_mapping_v2"] = []map[string]interface{}{
					{
						"delete_on_termination": true,
						"destination_type":      "volume",
						"source_type":           "volume",
						"uuid":                  rootVolumeUUID,
						"boot_index":            float64(0),
					},
				}
				expectCreateServer(r.compute, createMap, false)

				// Don't delete ports because the server is created: DeleteInstance will do it
			},
			wantErr: false,
		},
		{
			name: "Boot from volume with explicit AZ and volume type",
			getInstanceSpec: func() *InstanceSpec {
				s := getDefaultInstanceSpec()
				s.RootVolume = &infrav1.RootVolume{
					SizeGiB: 50,
				}
				azName := infrav1.VolumeAZName("test-alternate-az")
				s.RootVolume.AvailabilityZone = &infrav1.VolumeAvailabilityZone{
					Name: &azName,
				}
				s.RootVolume.Type = "test-volume-type"
				return s
			},
			expect: func(r *recorders) {
				expectDefaultFlavor(r.compute)

				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-root", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:             50,
					AvailabilityZone: "test-alternate-az",
					VolumeType:       "test-volume-type",
					Description:      fmt.Sprintf("Root volume for %s", openStackMachineName),
					Name:             fmt.Sprintf("%s-root", openStackMachineName),
					ImageID:          imageUUID,
					Multiattach:      false,
				}).Return(&volumes.Volume{ID: rootVolumeUUID}, nil)
				expectVolumePollSuccess(r.volume, rootVolumeUUID)

				createMap := getDefaultServerMap()
				serverMap := createMap["server"].(map[string]interface{})
				serverMap["imageRef"] = ""
				serverMap["block_device_mapping_v2"] = []map[string]interface{}{
					{
						"delete_on_termination": true,
						"destination_type":      "volume",
						"source_type":           "volume",
						"uuid":                  rootVolumeUUID,
						"boot_index":            float64(0),
					},
				}
				expectCreateServer(r.compute, createMap, false)

				// Don't delete ports because the server is created: DeleteInstance will do it
			},
			wantErr: false,
		},
		{
			name: "Boot from volume with AZ from machine",
			getInstanceSpec: func() *InstanceSpec {
				s := getDefaultInstanceSpec()
				s.RootVolume = &infrav1.RootVolume{
					SizeGiB: 50,
				}
				s.RootVolume.AvailabilityZone = &infrav1.VolumeAvailabilityZone{
					From: infrav1.VolumeAZFromMachine,
				}
				s.RootVolume.Type = "test-volume-type"
				return s
			},
			expect: func(r *recorders) {
				expectDefaultFlavor(r.compute)

				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-root", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:             50,
					AvailabilityZone: failureDomain,
					VolumeType:       "test-volume-type",
					Description:      fmt.Sprintf("Root volume for %s", openStackMachineName),
					Name:             fmt.Sprintf("%s-root", openStackMachineName),
					ImageID:          imageUUID,
					Multiattach:      false,
				}).Return(&volumes.Volume{ID: rootVolumeUUID}, nil)
				expectVolumePollSuccess(r.volume, rootVolumeUUID)

				createMap := getDefaultServerMap()
				serverMap := createMap["server"].(map[string]interface{})
				serverMap["imageRef"] = ""
				serverMap["block_device_mapping_v2"] = []map[string]interface{}{
					{
						"delete_on_termination": true,
						"destination_type":      "volume",
						"source_type":           "volume",
						"uuid":                  rootVolumeUUID,
						"boot_index":            float64(0),
					},
				}
				expectCreateServer(r.compute, createMap, false)

				// Don't delete ports because the server is created: DeleteInstance will do it
			},
			wantErr: false,
		},
		{
			name: "Boot from volume failure cleans up ports",
			getInstanceSpec: func() *InstanceSpec {
				s := getDefaultInstanceSpec()
				s.RootVolume = &infrav1.RootVolume{
					SizeGiB: 50,
				}
				return s
			},
			expect: func(r *recorders) {
				expectDefaultFlavor(r.compute)

				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-root", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:        50,
					Description: fmt.Sprintf("Root volume for %s", openStackMachineName),
					Name:        fmt.Sprintf("%s-root", openStackMachineName),
					ImageID:     imageUUID,
					Multiattach: false,
				}).Return(&volumes.Volume{ID: rootVolumeUUID}, nil)
				expectVolumePoll(r.volume, rootVolumeUUID, []string{"creating", "error"})
			},
			wantErr: true,
		},
		{
			name: "Root volume with additional block device success",
			getInstanceSpec: func() *InstanceSpec {
				s := getDefaultInstanceSpec()
				s.RootVolume = &infrav1.RootVolume{
					SizeGiB: 50,
				}
				s.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
					{
						Name:    "etcd",
						SizeGiB: 50,
						Storage: infrav1.BlockDeviceStorage{
							Type: "Volume",
							Volume: &infrav1.BlockDeviceVolume{
								Type: "test-volume-type",
							},
						},
					},
					{
						Name:    "local-device",
						SizeGiB: 10,
						Storage: infrav1.BlockDeviceStorage{
							Type: "Local",
						},
					},
				}
				return s
			},
			expect: func(r *recorders) {
				expectDefaultFlavor(r.compute)

				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-root", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:        50,
					Description: fmt.Sprintf("Root volume for %s", openStackMachineName),
					Name:        fmt.Sprintf("%s-root", openStackMachineName),
					ImageID:     imageUUID,
					Multiattach: false,
				}).Return(&volumes.Volume{ID: rootVolumeUUID}, nil)
				expectVolumePollSuccess(r.volume, rootVolumeUUID)

				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-etcd", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:        50,
					Description: fmt.Sprintf("Additional block device for %s", openStackMachineName),
					Name:        fmt.Sprintf("%s-etcd", openStackMachineName),
					Multiattach: false,
					VolumeType:  "test-volume-type",
				}).Return(&volumes.Volume{ID: additionalBlockDeviceVolumeUUID}, nil)
				expectVolumePollSuccess(r.volume, additionalBlockDeviceVolumeUUID)

				createMap := getDefaultServerMap()
				serverMap := createMap["server"].(map[string]interface{})
				serverMap["imageRef"] = ""
				serverMap["block_device_mapping_v2"] = []map[string]interface{}{
					{
						"source_type":           "volume",
						"uuid":                  rootVolumeUUID,
						"boot_index":            float64(0),
						"delete_on_termination": true,
						"destination_type":      "volume",
					},
					{
						"source_type":           "volume",
						"uuid":                  additionalBlockDeviceVolumeUUID,
						"boot_index":            float64(-1),
						"delete_on_termination": true,
						"destination_type":      "volume",
						"tag":                   "etcd",
					},
					{
						"source_type":           "blank",
						"destination_type":      "local",
						"boot_index":            float64(-1),
						"delete_on_termination": true,
						"volume_size":           float64(10),
						"tag":                   "local-device",
					},
				}
				expectCreateServer(r.compute, createMap, false)

				// Don't delete ports because the server is created: DeleteInstance will do it
			},
			wantErr: false,
		},
		{
			name: "Additional block devices success",
			getInstanceSpec: func() *InstanceSpec {
				s := getDefaultInstanceSpec()
				s.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
					{
						Name:    "etcd",
						SizeGiB: 50,
						Storage: infrav1.BlockDeviceStorage{
							Type: "Volume",
							Volume: &infrav1.BlockDeviceVolume{
								Type: "test-volume-type",
							},
						},
					},
					{
						Name:    "data",
						SizeGiB: 10,
						Storage: infrav1.BlockDeviceStorage{
							Type: "Local",
						},
					},
				}
				return s
			},
			expect: func(r *recorders) {
				expectDefaultFlavor(r.compute)

				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-etcd", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:        50,
					Description: fmt.Sprintf("Additional block device for %s", openStackMachineName),
					Name:        fmt.Sprintf("%s-etcd", openStackMachineName),
					Multiattach: false,
					VolumeType:  "test-volume-type",
				}).Return(&volumes.Volume{ID: additionalBlockDeviceVolumeUUID}, nil)
				expectVolumePollSuccess(r.volume, additionalBlockDeviceVolumeUUID)

				createMap := getDefaultServerMap()
				serverMap := createMap["server"].(map[string]interface{})
				serverMap["block_device_mapping_v2"] = []map[string]interface{}{
					{
						"source_type":           "image",
						"uuid":                  imageUUID,
						"boot_index":            float64(0),
						"delete_on_termination": true,
						"destination_type":      "local",
					},
					{
						"source_type":           "volume",
						"uuid":                  additionalBlockDeviceVolumeUUID,
						"boot_index":            float64(-1),
						"delete_on_termination": true,
						"destination_type":      "volume",
						"tag":                   "etcd",
					},
					{
						"source_type":           "blank",
						"destination_type":      "local",
						"boot_index":            float64(-1),
						"delete_on_termination": true,
						"volume_size":           float64(10),
						"tag":                   "data",
					},
				}
				expectCreateServer(r.compute, createMap, false)

				// Don't delete ports because the server is created: DeleteInstance will do it
			},
			wantErr: false,
		},
		{
			name: "Additional block device success with explicit AZ",
			getInstanceSpec: func() *InstanceSpec {
				s := getDefaultInstanceSpec()
				azName := infrav1.VolumeAZName("test-alternate-az")
				s.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
					{
						Name:    "etcd",
						SizeGiB: 50,
						Storage: infrav1.BlockDeviceStorage{
							Type: "Volume",
							Volume: &infrav1.BlockDeviceVolume{
								Type: "test-volume-type",
								AvailabilityZone: &infrav1.VolumeAvailabilityZone{
									Name: &azName,
								},
							},
						},
					},
				}
				return s
			},
			expect: func(r *recorders) {
				expectDefaultFlavor(r.compute)

				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-etcd", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:             50,
					AvailabilityZone: "test-alternate-az",
					Description:      fmt.Sprintf("Additional block device for %s", openStackMachineName),
					Name:             fmt.Sprintf("%s-etcd", openStackMachineName),
					Multiattach:      false,
					VolumeType:       "test-volume-type",
				}).Return(&volumes.Volume{ID: additionalBlockDeviceVolumeUUID}, nil)
				expectVolumePollSuccess(r.volume, additionalBlockDeviceVolumeUUID)

				createMap := getDefaultServerMap()
				serverMap := createMap["server"].(map[string]interface{})
				serverMap["block_device_mapping_v2"] = []map[string]interface{}{
					{
						"source_type":           "image",
						"uuid":                  imageUUID,
						"boot_index":            float64(0),
						"delete_on_termination": true,
						"destination_type":      "local",
					},
					{
						"source_type":           "volume",
						"uuid":                  additionalBlockDeviceVolumeUUID,
						"boot_index":            float64(-1),
						"delete_on_termination": true,
						"destination_type":      "volume",
						"tag":                   "etcd",
					},
				}
				expectCreateServer(r.compute, createMap, false)

				// Don't delete ports because the server is created: DeleteInstance will do it
			},
			wantErr: false,
		},
		{
			name: "Additional block device error when using wrong type",
			getInstanceSpec: func() *InstanceSpec {
				s := getDefaultInstanceSpec()
				s.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
					{
						Name:    "oops",
						SizeGiB: 1,
						Storage: infrav1.BlockDeviceStorage{
							Type: "doesnt-exist",
						},
					},
				}
				return s
			},
			expect: func(r *recorders) {
				expectDefaultFlavor(r.compute)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			log := testr.New(t)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")

			computeRecorder := mockScopeFactory.ComputeClient.EXPECT()
			imageRecorder := mockScopeFactory.ImageClient.EXPECT()
			networkRecorder := mockScopeFactory.NetworkClient.EXPECT()
			volumeRecorder := mockScopeFactory.VolumeClient.EXPECT()

			tt.expect(&recorders{computeRecorder, imageRecorder, networkRecorder, volumeRecorder})

			s, err := NewService(scope.NewWithLogger(mockScopeFactory, log))
			if err != nil {
				t.Fatalf("Failed to create service: %v", err)
			}

			// Call CreateInstance with a reduced retry interval to speed up the test
			_, err = s.createInstanceImpl(&infrav1.OpenStackMachine{}, tt.getInstanceSpec(), time.Nanosecond, portUUIDs)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.CreateInstance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
