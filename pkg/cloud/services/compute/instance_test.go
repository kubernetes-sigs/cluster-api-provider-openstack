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

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	common "github.com/gophercloud/gophercloud/openstack/common/extensions"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/attachinterfaces"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha7"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

type gomegaMockMatcher struct {
	matcher     gomegatypes.GomegaMatcher
	description string
}

func newGomegaMockMatcher(matcher gomegatypes.GomegaMatcher) *gomegaMockMatcher {
	return &gomegaMockMatcher{
		matcher:     matcher,
		description: "",
	}
}

func (m *gomegaMockMatcher) String() string {
	return m.description
}

func (m *gomegaMockMatcher) Matches(x interface{}) bool {
	success, err := m.matcher.Match(x)
	Expect(err).NotTo(HaveOccurred())
	if !success {
		m.description = m.matcher.FailureMessage(x)
	}
	return success
}

func Test_getPortName(t *testing.T) {
	type args struct {
		instanceName string
		opts         *infrav1.PortOpts
		netIndex     int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "with nil PortOpts",
			args: args{"test-1-instance", nil, 2},
			want: "test-1-instance-2",
		},
		{
			name: "with PortOpts name suffix",
			args: args{"test-1-instance", &infrav1.PortOpts{NameSuffix: "foo"}, 4},
			want: "test-1-instance-foo",
		},
		{
			name: "without PortOpts name suffix",
			args: args{"test-1-instance", &infrav1.PortOpts{}, 4},
			want: "test-1-instance-4",
		},
		{
			name: "with PortOpts name suffix",
			args: args{"test-1-instance", &infrav1.PortOpts{NameSuffix: "foo2", Network: &infrav1.NetworkFilter{ID: "bar"}, DisablePortSecurity: pointer.Bool(true)}, 4},
			want: "test-1-instance-foo2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := networking.GetPortName(tt.args.instanceName, tt.args.opts, tt.args.netIndex); got != tt.want {
				t.Errorf("getPortName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_getImageID(t *testing.T) {
	const imageIDA = "ce96e584-7ebc-46d6-9e55-987d72e3806c"
	const imageIDB = "8f536889-5198-42d7-8314-cb78f4f4755c"
	const imageIDC = "8f536889-5198-42d7-8314-cb78f4f4755d"

	tests := []struct {
		testName  string
		imageUUID string
		imageName string
		expect    func(m *mock.MockImageClientMockRecorder)
		want      string
		wantErr   bool
	}{
		{
			testName:  "Return image uuid if uuid given",
			imageUUID: imageIDC,
			want:      imageIDC,
			expect: func(m *mock.MockImageClientMockRecorder) {
			},
			wantErr: false,
		},
		{
			testName:  "Return through uuid if both uuid and name given",
			imageName: "dummy",
			imageUUID: imageIDC,
			expect: func(m *mock.MockImageClientMockRecorder) {
			},
			want:    imageIDC,
			wantErr: false,
		},
		{
			testName:  "Return image ID",
			imageName: "test-image",
			expect: func(m *mock.MockImageClientMockRecorder) {
				m.ListImages(images.ListOpts{Name: "test-image"}).Return(
					[]images.Image{{ID: imageIDA, Name: "test-image"}},
					nil)
			},
			want:    imageIDA,
			wantErr: false,
		},
		{
			testName:  "Return no results",
			imageName: "test-image",
			expect: func(m *mock.MockImageClientMockRecorder) {
				m.ListImages(images.ListOpts{Name: "test-image"}).Return(
					[]images.Image{},
					nil)
			},
			want:    "",
			wantErr: true,
		},
		{
			testName:  "Return multiple results",
			imageName: "test-image",
			expect: func(m *mock.MockImageClientMockRecorder) {
				m.ListImages(images.ListOpts{Name: "test-image"}).Return(
					[]images.Image{
						{ID: imageIDA, Name: "test-image"},
						{ID: imageIDB, Name: "test-image"},
					}, nil)
			},
			want:    "",
			wantErr: true,
		},
		{
			testName:  "OpenStack returns error",
			imageName: "test-image",
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
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "", logr.Discard())

			s, err := NewService(mockScopeFactory)
			if err != nil {
				t.Fatalf("Failed to create service: %v", err)
			}
			tt.expect(mockScopeFactory.ImageClient.EXPECT())

			got, err := s.getImageID(tt.imageUUID, tt.imageName)
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
			ControlPlaneSecurityGroup: &infrav1.SecurityGroup{ID: controlPlaneSecurityGroupUUID},
			WorkerSecurityGroup:       &infrav1.SecurityGroup{ID: workerSecurityGroupUUID},
		},
	}
}

func getDefaultInstanceSpec() *InstanceSpec {
	return &InstanceSpec{
		Name:       openStackMachineName,
		Image:      imageName,
		Flavor:     flavorName,
		SSHKeyName: sshKeyName,
		UserData:   "user-data",
		Metadata: map[string]string{
			"test-metadata": "test-value",
		},
		ConfigDrive:    *pointer.Bool(true),
		FailureDomain:  *pointer.String(failureDomain),
		ServerGroupID:  serverGroupUUID,
		Tags:           []string{"test-tag"},
		SecurityGroups: []infrav1.SecurityGroupFilter{{ID: workerSecurityGroupUUID}},
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

	// Expected calls to create a server with a single default port
	expectUseExistingDefaultPort := func(networkRecorder *mock.MockNetworkClientMockRecorder) {
		// Returning a pre-existing port requires fewer mocks
		networkRecorder.ListPort(ports.ListOpts{
			Name:      portName,
			NetworkID: networkUUID,
		}).Return([]ports.Port{
			{
				ID:        portUUID,
				NetworkID: networkUUID,
			},
		}, nil)
	}

	expectCreatePort := func(networkRecorder *mock.MockNetworkClientMockRecorder, name string, networkID string) {
		networkRecorder.ListPort(ports.ListOpts{
			Name:      name,
			NetworkID: networkID,
		}).Return([]ports.Port{}, nil)

		// gomock won't match a pointer to a nil slice in SecurityGroups, so we do this
		networkRecorder.CreatePort(gomock.Any()).DoAndReturn(func(createOpts ports.CreateOptsBuilder) (*ports.Port, error) {
			createOptsMap, err := createOpts.ToPortCreateMap()
			Expect(err).NotTo(HaveOccurred())

			// Match only the fields we're interested in
			portOpts := createOptsMap["port"].(map[string]interface{})
			Expect(portOpts).To(MatchKeys(IgnoreExtras, Keys{
				"network_id": Equal(networkUUID),
				"name":       Equal(portName),
			}))

			return &ports.Port{
				ID:          portUUID,
				NetworkID:   networkUUID,
				Name:        portName,
				Description: portOpts["description"].(string),
			}, nil
		})
		networkRecorder.ReplaceAllAttributesTags("ports", portUUID, attributestags.ReplaceAllOpts{Tags: []string{"test-tag"}}).Return(nil, nil)
	}

	// Expected calls if we delete the network port
	expectCleanupDefaultPort := func(networkRecorder *mock.MockNetworkClientMockRecorder) {
		networkRecorder.ListExtensions()
		networkRecorder.DeletePort(portUUID).Return(nil)
	}

	// Expected calls when using default image and flavor
	expectDefaultImageAndFlavor := func(computeRecorder *mock.MockComputeClientMockRecorder, imageRecorder *mock.MockImageClientMockRecorder) {
		imageRecorder.ListImages(images.ListOpts{Name: imageName}).Return([]images.Image{{ID: imageUUID}}, nil)
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

	// Expected calls when polling for server creation
	expectServerPoll := func(computeRecorder *mock.MockComputeClientMockRecorder, states []string) {
		for _, state := range states {
			computeRecorder.GetServer(instanceUUID).Return(returnedServer(state), nil)
		}
	}

	expectServerPollSuccess := func(computeRecorder *mock.MockComputeClientMockRecorder) {
		expectServerPoll(computeRecorder, []string{"ACTIVE"})
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
				expectUseExistingDefaultPort(r.network)
				expectDefaultImageAndFlavor(r.compute, r.image)

				expectCreateServer(r.compute, getDefaultServerMap(), false)
				expectServerPollSuccess(r.compute)
			},
			wantErr: false,
		},
		{
			name:            "Delete ports on server create error",
			getInstanceSpec: getDefaultInstanceSpec,
			expect: func(r *recorders) {
				expectUseExistingDefaultPort(r.network)
				expectDefaultImageAndFlavor(r.compute, r.image)

				expectCreateServer(r.compute, getDefaultServerMap(), true)

				// Make sure we delete ports
				expectCleanupDefaultPort(r.network)
			},
			wantErr: true,
		},
		{
			name: "Delete previously created ports on port creation error",
			getInstanceSpec: func() *InstanceSpec {
				s := getDefaultInstanceSpec()
				s.Ports = []infrav1.PortOpts{
					{Description: "Test port 0"},
					{Description: "Test port 1"},
				}
				return s
			},
			expect: func(r *recorders) {
				expectDefaultImageAndFlavor(r.compute, r.image)
				expectUseExistingDefaultPort(r.network)

				// Looking up the second port fails
				r.network.ListPort(ports.ListOpts{
					Name:      "test-openstack-machine-1",
					NetworkID: networkUUID,
				}).Return(nil, fmt.Errorf("test error"))

				// We should cleanup the first port
				expectCleanupDefaultPort(r.network)
			},
			wantErr: true,
		},
		{
			name:            "Poll until server is created",
			getInstanceSpec: getDefaultInstanceSpec,
			expect: func(r *recorders) {
				expectUseExistingDefaultPort(r.network)
				expectDefaultImageAndFlavor(r.compute, r.image)

				expectCreateServer(r.compute, getDefaultServerMap(), false)
				expectServerPoll(r.compute, []string{"BUILDING", "ACTIVE"})
			},
			wantErr: false,
		},
		{
			name:            "Server errors during creation",
			getInstanceSpec: getDefaultInstanceSpec,
			expect: func(r *recorders) {
				expectUseExistingDefaultPort(r.network)
				expectDefaultImageAndFlavor(r.compute, r.image)

				expectCreateServer(r.compute, getDefaultServerMap(), false)
				expectServerPoll(r.compute, []string{"BUILDING", "ERROR"})

				// Don't delete ports because the server is created: DeleteInstance will do it
			},
			wantErr: true,
		},
		{
			name: "Boot from volume success",
			getInstanceSpec: func() *InstanceSpec {
				s := getDefaultInstanceSpec()
				s.RootVolume = &infrav1.RootVolume{
					Size: 50,
				}
				return s
			},
			expect: func(r *recorders) {
				expectUseExistingDefaultPort(r.network)
				expectDefaultImageAndFlavor(r.compute, r.image)

				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-root", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:             50,
					AvailabilityZone: failureDomain,
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
				expectServerPollSuccess(r.compute)

				// Don't delete ports because the server is created: DeleteInstance will do it
			},
			wantErr: false,
		},
		{
			name: "Boot from volume with explicit AZ and volume type",
			getInstanceSpec: func() *InstanceSpec {
				s := getDefaultInstanceSpec()
				s.RootVolume = &infrav1.RootVolume{
					Size:             50,
					AvailabilityZone: "test-alternate-az",
					VolumeType:       "test-volume-type",
				}
				return s
			},
			expect: func(r *recorders) {
				expectUseExistingDefaultPort(r.network)
				expectDefaultImageAndFlavor(r.compute, r.image)

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
				expectServerPollSuccess(r.compute)

				// Don't delete ports because the server is created: DeleteInstance will do it
			},
			wantErr: false,
		},
		{
			name: "Boot from volume failure cleans up ports",
			getInstanceSpec: func() *InstanceSpec {
				s := getDefaultInstanceSpec()
				s.RootVolume = &infrav1.RootVolume{
					Size: 50,
				}
				return s
			},
			expect: func(r *recorders) {
				expectUseExistingDefaultPort(r.network)
				expectDefaultImageAndFlavor(r.compute, r.image)

				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-root", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:             50,
					AvailabilityZone: failureDomain,
					Description:      fmt.Sprintf("Root volume for %s", openStackMachineName),
					Name:             fmt.Sprintf("%s-root", openStackMachineName),
					ImageID:          imageUUID,
					Multiattach:      false,
				}).Return(&volumes.Volume{ID: rootVolumeUUID}, nil)
				expectVolumePoll(r.volume, rootVolumeUUID, []string{"creating", "error"})

				expectCleanupDefaultPort(r.network)
			},
			wantErr: true,
		},
		{
			name: "Root volume with additional block device success",
			getInstanceSpec: func() *InstanceSpec {
				s := getDefaultInstanceSpec()
				s.RootVolume = &infrav1.RootVolume{
					Size: 50,
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
				expectUseExistingDefaultPort(r.network)
				expectDefaultImageAndFlavor(r.compute, r.image)

				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-root", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:             50,
					AvailabilityZone: failureDomain,
					Description:      fmt.Sprintf("Root volume for %s", openStackMachineName),
					Name:             fmt.Sprintf("%s-root", openStackMachineName),
					ImageID:          imageUUID,
					Multiattach:      false,
				}).Return(&volumes.Volume{ID: rootVolumeUUID}, nil)
				expectVolumePollSuccess(r.volume, rootVolumeUUID)

				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-etcd", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:             50,
					AvailabilityZone: failureDomain,
					Description:      fmt.Sprintf("Additional block device for %s", openStackMachineName),
					Name:             fmt.Sprintf("%s-etcd", openStackMachineName),
					Multiattach:      false,
					VolumeType:       "test-volume-type",
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
				expectServerPollSuccess(r.compute)

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
				expectUseExistingDefaultPort(r.network)
				expectDefaultImageAndFlavor(r.compute, r.image)

				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-etcd", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:             50,
					AvailabilityZone: failureDomain,
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
				expectServerPollSuccess(r.compute)

				// Don't delete ports because the server is created: DeleteInstance will do it
			},
			wantErr: false,
		},
		{
			name: "Additional block device success with explicit AZ",
			getInstanceSpec: func() *InstanceSpec {
				s := getDefaultInstanceSpec()
				s.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
					{
						Name:    "etcd",
						SizeGiB: 50,
						Storage: infrav1.BlockDeviceStorage{
							Type: "Volume",
							Volume: &infrav1.BlockDeviceVolume{
								AvailabilityZone: "test-alternate-az",
								Type:             "test-volume-type",
							},
						},
					},
				}
				return s
			},
			expect: func(r *recorders) {
				expectUseExistingDefaultPort(r.network)
				expectDefaultImageAndFlavor(r.compute, r.image)

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
				expectServerPollSuccess(r.compute)

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
				expectUseExistingDefaultPort(r.network)
				expectDefaultImageAndFlavor(r.compute, r.image)

				// Make sure we delete ports
				expectCleanupDefaultPort(r.network)
			},
			wantErr: true,
		},
		{
			name: "Delete trunks on port creation error",
			getInstanceSpec: func() *InstanceSpec {
				s := getDefaultInstanceSpec()
				s.Ports = []infrav1.PortOpts{
					{Description: "Test port 0", Trunk: pointer.Bool(true)},
					{Description: "Test port 1"},
				}
				return s
			},
			expect: func(r *recorders) {
				expectDefaultImageAndFlavor(r.compute, r.image)
				extensions := []extensions.Extension{
					{Extension: common.Extension{Alias: "trunk"}},
				}
				r.network.ListExtensions().Return(extensions, nil)

				expectCreatePort(r.network, portName, networkUUID)

				// Check for existing trunk
				r.network.ListTrunk(newGomegaMockMatcher(
					MatchFields(IgnoreExtras, Fields{
						"Name":   Equal(portName),
						"PortID": Equal(portUUID),
					}),
				)).Return([]trunks.Trunk{}, nil)

				// Create new trunk
				r.network.CreateTrunk(newGomegaMockMatcher(MatchFields(IgnoreExtras, Fields{
					"Name":   Equal(portName),
					"PortID": Equal(portUUID),
				}))).Return(&trunks.Trunk{
					PortID: portUUID,
					ID:     trunkUUID,
				}, nil)
				r.network.ReplaceAllAttributesTags("trunks", trunkUUID, attributestags.ReplaceAllOpts{Tags: []string{"test-tag"}}).Return(nil, nil)

				// Looking up the second port fails
				r.network.ListPort(ports.ListOpts{
					Name:      "test-openstack-machine-1",
					NetworkID: networkUUID,
				}).Return(nil, fmt.Errorf("test error"))

				r.network.ListExtensions().Return(extensions, nil)

				r.network.ListTrunk(newGomegaMockMatcher(
					MatchFields(IgnoreExtras, Fields{
						"PortID": Equal(portUUID),
					}),
				)).Return([]trunks.Trunk{{ID: trunkUUID}}, nil)

				// We should cleanup the first port and its trunk
				r.network.DeleteTrunk(trunkUUID).Return(nil)
				r.network.ListTrunkSubports(trunkUUID).Return([]trunks.Subport{}, nil)
				r.network.DeletePort(portUUID).Return(nil)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "", logr.Discard())

			computeRecorder := mockScopeFactory.ComputeClient.EXPECT()
			imageRecorder := mockScopeFactory.ImageClient.EXPECT()
			networkRecorder := mockScopeFactory.NetworkClient.EXPECT()
			volumeRecorder := mockScopeFactory.VolumeClient.EXPECT()

			tt.expect(&recorders{computeRecorder, imageRecorder, networkRecorder, volumeRecorder})

			s, err := NewService(mockScopeFactory)
			if err != nil {
				t.Fatalf("Failed to create service: %v", err)
			}

			// Call CreateInstance with a reduced retry interval to speed up the test
			_, err = s.createInstanceImpl(&infrav1.OpenStackMachine{}, getDefaultOpenStackCluster(), tt.getInstanceSpec(), "cluster-name", time.Nanosecond)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.CreateInstance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestService_DeleteInstance(t *testing.T) {
	RegisterTestingT(t)

	getDefaultInstanceStatus := func() *InstanceStatus {
		return &InstanceStatus{
			server: &clients.ServerExt{
				Server: servers.Server{
					ID: instanceUUID,
				},
			},
		}
	}

	// *******************
	// START OF TEST CASES
	// *******************

	type recorders struct {
		compute *mock.MockComputeClientMockRecorder
		network *mock.MockNetworkClientMockRecorder
		volume  *mock.MockVolumeClientMockRecorder
	}

	tests := []struct {
		name           string
		eventObject    runtime.Object
		instanceStatus func() *InstanceStatus
		rootVolume     *infrav1.RootVolume
		expect         func(r *recorders)
		wantErr        bool
	}{
		{
			name:           "Defaults",
			eventObject:    &infrav1.OpenStackMachine{},
			instanceStatus: getDefaultInstanceStatus,
			expect: func(r *recorders) {
				r.compute.ListAttachedInterfaces(instanceUUID).Return([]attachinterfaces.Interface{
					{
						PortID: portUUID,
					},
				}, nil)
				r.network.ListExtensions().Return([]extensions.Extension{{
					Extension: common.Extension{
						Alias: "trunk",
					},
				}}, nil)
				r.compute.DeleteServer(instanceUUID).Return(nil)
				r.compute.GetServer(instanceUUID).Return(nil, gophercloud.ErrDefault404{})

				// FIXME: Why we are looking for a trunk when we know the port is not trunked?
				r.network.ListTrunk(trunks.ListOpts{PortID: portUUID}).Return([]trunks.Trunk{}, nil)
				r.network.DeletePort(portUUID).Return(nil)
			},
			wantErr: false,
		},
		{
			name:           "Dangling volume",
			eventObject:    &infrav1.OpenStackMachine{},
			instanceStatus: func() *InstanceStatus { return nil },
			rootVolume: &infrav1.RootVolume{
				Size: 50,
			},
			expect: func(r *recorders) {
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
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "", logr.Discard())

			computeRecorder := mockScopeFactory.ComputeClient.EXPECT()
			networkRecorder := mockScopeFactory.NetworkClient.EXPECT()
			volumeRecorder := mockScopeFactory.VolumeClient.EXPECT()

			tt.expect(&recorders{computeRecorder, networkRecorder, volumeRecorder})

			s, err := NewService(mockScopeFactory)
			if err != nil {
				t.Fatalf("Failed to create service: %v", err)
			}

			instanceSpec := &InstanceSpec{
				Name:       openStackMachineName,
				RootVolume: tt.rootVolume,
			}

			if err := s.DeleteInstance(&infrav1.OpenStackCluster{}, tt.eventObject, tt.instanceStatus(), instanceSpec); (err != nil) != tt.wantErr {
				t.Errorf("Service.DeleteInstance() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_normalizePorts(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	const (
		defaultNetworkID = "3c66f3ca-2d26-4d9d-ae3b-568f54129773"
		defaultSubnetID  = "d8dbba89-8c39-4192-a571-e702fca35bac"

		networkID = "afa54944-1443-4132-9ef5-ce37eb4d6ab6"
		subnetID  = "d786e715-c299-4a97-911d-640c10fc0392"
	)

	openStackCluster := &infrav1.OpenStackCluster{
		Status: infrav1.OpenStackClusterStatus{
			Network: &infrav1.NetworkStatusWithSubnets{
				NetworkStatus: infrav1.NetworkStatus{
					ID: defaultNetworkID,
				},
				Subnets: []infrav1.Subnet{
					{ID: defaultSubnetID},
				},
			},
		},
	}

	tests := []struct {
		name          string
		ports         []infrav1.PortOpts
		instanceTrunk bool
		expectNetwork func(m *mock.MockNetworkClientMockRecorder)
		want          []infrav1.PortOpts
		wantErr       bool
	}{
		{
			name:  "No ports: no ports",
			ports: []infrav1.PortOpts{},
			want:  []infrav1.PortOpts{},
		},
		{
			name: "Nil network, no fixed IPs: cluster defaults",
			ports: []infrav1.PortOpts{
				{
					Network:  nil,
					FixedIPs: nil,
				},
			},
			want: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID: defaultNetworkID,
					},
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								ID: defaultSubnetID,
							},
						},
					},
					Trunk: pointer.Bool(false),
				},
			},
		},
		{
			name: "Empty network, no fixed IPs: cluster defaults",
			ports: []infrav1.PortOpts{
				{
					Network:  &infrav1.NetworkFilter{},
					FixedIPs: nil,
				},
			},
			want: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID: defaultNetworkID,
					},
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								ID: defaultSubnetID,
							},
						},
					},
					Trunk: pointer.Bool(false),
				},
			},
		},
		{
			name: "Port inherits trunk from instance",
			ports: []infrav1.PortOpts{
				{
					Network:  &infrav1.NetworkFilter{},
					FixedIPs: nil,
				},
			},
			instanceTrunk: true,
			want: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID: defaultNetworkID,
					},
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								ID: defaultSubnetID,
							},
						},
					},
					Trunk: pointer.Bool(true),
				},
			},
		},
		{
			name: "Port overrides trunk from instance",
			ports: []infrav1.PortOpts{
				{
					Network:  &infrav1.NetworkFilter{},
					FixedIPs: nil,
					Trunk:    pointer.Bool(true),
				},
			},
			want: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID: defaultNetworkID,
					},
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								ID: defaultSubnetID,
							},
						},
					},
					Trunk: pointer.Bool(true),
				},
			},
		},
		{
			name: "Network defined by ID: unchanged",
			ports: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID: networkID,
					},
				},
			},
			want: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID: networkID,
					},
					Trunk: pointer.Bool(false),
				},
			},
		},
		{
			name: "Network defined by filter: add ID from network lookup",
			ports: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						Name: "test-network",
					},
				},
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListNetwork(networks.ListOpts{Name: "test-network"}).Return([]networks.Network{
					{ID: networkID},
				}, nil)
			},
			want: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID:   networkID,
						Name: "test-network",
					},
					Trunk: pointer.Bool(false),
				},
			},
		},
		{
			name: "No network, fixed IP has subnet by ID: add ID from subnet",
			ports: []infrav1.PortOpts{
				{
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								ID: subnetID,
							},
						},
					},
				},
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.GetSubnet(subnetID).Return(&subnets.Subnet{ID: subnetID, NetworkID: networkID}, nil)
			},
			want: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID: networkID,
					},
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								ID: subnetID,
							},
						},
					},
					Trunk: pointer.Bool(false),
				},
			},
		},
		{
			name: "No network, fixed IP has subnet by filter: add ID from subnet",
			ports: []infrav1.PortOpts{
				{
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								Name: "test-subnet",
							},
						},
					},
				},
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListSubnet(subnets.ListOpts{Name: "test-subnet"}).Return([]subnets.Subnet{
					{ID: subnetID, NetworkID: networkID},
				}, nil)
			},
			want: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID: networkID,
					},
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								ID:   subnetID,
								Name: "test-subnet",
							},
						},
					},
					Trunk: pointer.Bool(false),
				},
			},
		},
		{
			name: "No network, fixed IP subnet returns no matches: error",
			ports: []infrav1.PortOpts{
				{
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								Name: "test-subnet",
							},
						},
					},
				},
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListSubnet(subnets.ListOpts{Name: "test-subnet"}).Return([]subnets.Subnet{}, nil)
			},
			wantErr: true,
		},
		{
			name: "No network, only fixed IP subnet returns multiple matches: error",
			ports: []infrav1.PortOpts{
				{
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								Name: "test-subnet",
							},
						},
					},
				},
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListSubnet(subnets.ListOpts{Name: "test-subnet"}).Return([]subnets.Subnet{
					{ID: subnetID, NetworkID: networkID},
					{ID: "8008494c-301e-4e5c-951b-a8ab568447fd", NetworkID: "5d48bfda-db28-42ee-8374-50e13d1fe5ea"},
				}, nil)
			},
			wantErr: true,
		},
		{
			name: "No network, first fixed IP subnet returns multiple matches: used ID from second fixed IP",
			ports: []infrav1.PortOpts{
				{
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								Name: "test-subnet1",
							},
						},
						{
							Subnet: &infrav1.SubnetFilter{
								Name: "test-subnet2",
							},
						},
					},
				},
			},
			expectNetwork: func(m *mock.MockNetworkClientMockRecorder) {
				m.ListSubnet(subnets.ListOpts{Name: "test-subnet1"}).Return([]subnets.Subnet{
					{ID: subnetID, NetworkID: networkID},
					{ID: "8008494c-301e-4e5c-951b-a8ab568447fd", NetworkID: "5d48bfda-db28-42ee-8374-50e13d1fe5ea"},
				}, nil)
				m.ListSubnet(subnets.ListOpts{Name: "test-subnet2"}).Return([]subnets.Subnet{
					{ID: subnetID, NetworkID: networkID},
				}, nil)
			},
			want: []infrav1.PortOpts{
				{
					Network: &infrav1.NetworkFilter{
						ID: networkID,
					},
					FixedIPs: []infrav1.FixedIP{
						{
							Subnet: &infrav1.SubnetFilter{
								Name: "test-subnet1",
							},
						},
						{
							Subnet: &infrav1.SubnetFilter{
								ID:   subnetID,
								Name: "test-subnet2",
							},
						},
					},
					Trunk: pointer.Bool(false),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// MockScopeFactory also implements Scope so no need to create separate Scope from it.
			mockScope := scope.NewMockScopeFactory(mockCtrl, "", logr.Discard())
			if tt.expectNetwork != nil {
				tt.expectNetwork(mockScope.NetworkClient.EXPECT())
			}

			s := &Service{
				scope: mockScope,
			}
			instanceSpec := &InstanceSpec{
				Trunk: tt.instanceTrunk,
			}

			got, err := s.normalizePorts(tt.ports, openStackCluster, instanceSpec)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
				return
			}

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(got).To(Equal(tt.want), cmp.Diff(got, tt.want))
		})
	}
}
