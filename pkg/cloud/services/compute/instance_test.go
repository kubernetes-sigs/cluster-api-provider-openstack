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
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr/testr"
	"github.com/google/go-cmp/cmp"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	. "github.com/onsi/gomega" //nolint:revive
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
)

func TestService_getImageID(t *testing.T) {
	const (
		imageID   = "ce96e584-7ebc-46d6-9e55-987d72e3806c"
		imageName = "test-image"
		namespace = "test-namespace"
	)
	imageTags := []string{"test-tag"}

	scheme := runtime.NewScheme()
	if err := orcv1alpha1.AddToScheme(scheme); err != nil {
		panic(err)
	}

	tests := []struct {
		testName          string
		image             infrav1.ImageParam
		fakeObjects       []runtime.Object
		expect            func(m *mock.MockImageClientMockRecorder)
		want              *string
		wantErr           bool
		wantTerminalError bool
	}{
		{
			testName: "Return image ID when ID given",
			image:    infrav1.ImageParam{ID: ptr.To(imageID)},
			want:     ptr.To(imageID),
			expect:   func(*mock.MockImageClientMockRecorder) {},
			wantErr:  false,
		},
		{
			testName: "Return image ID when name given",
			image: infrav1.ImageParam{
				Filter: &infrav1.ImageFilter{
					Name: ptr.To(imageName),
				},
			},
			want: ptr.To(imageID),
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
			want: ptr.To(imageID),
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
					Name: ptr.To(imageName),
				},
			},
			expect: func(m *mock.MockImageClientMockRecorder) {
				m.ListImages(images.ListOpts{Name: imageName}).Return(
					[]images.Image{},
					nil)
			},
			want:    nil,
			wantErr: true,
		},
		{
			testName: "Return multiple results",
			image: infrav1.ImageParam{
				Filter: &infrav1.ImageFilter{
					Name: ptr.To(imageName),
				},
			},
			expect: func(m *mock.MockImageClientMockRecorder) {
				m.ListImages(images.ListOpts{Name: "test-image"}).Return(
					[]images.Image{
						{ID: imageID, Name: "test-image"},
						{ID: "123", Name: "test-image"},
					}, nil)
			},
			want:    nil,
			wantErr: true,
		},
		{
			testName: "OpenStack returns error",
			image: infrav1.ImageParam{
				Filter: &infrav1.ImageFilter{
					Name: ptr.To(imageName),
				},
			},
			expect: func(m *mock.MockImageClientMockRecorder) {
				m.ListImages(images.ListOpts{Name: "test-image"}).Return(
					nil,
					fmt.Errorf("test error"))
			},
			want:    nil,
			wantErr: true,
		},
		{
			testName: "Image by reference does not exist",
			image: infrav1.ImageParam{
				ImageRef: &infrav1.ResourceReference{
					Name: imageName,
				},
			},
			want: nil,
		},
		{
			testName: "Image by reference exists, is available",
			image: infrav1.ImageParam{
				ImageRef: &infrav1.ResourceReference{
					Name: imageName,
				},
			},
			fakeObjects: []runtime.Object{
				&orcv1alpha1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      imageName,
						Namespace: namespace,
					},
					Status: orcv1alpha1.ImageStatus{
						Conditions: []metav1.Condition{
							{
								Type:   orcv1alpha1.ConditionAvailable,
								Status: metav1.ConditionTrue,
							},
						},
						ID: ptr.To(imageID),
					},
				},
			},
			want: ptr.To(imageID),
		},
		{
			testName: "Image by reference exists, still reconciling",
			image: infrav1.ImageParam{
				ImageRef: &infrav1.ResourceReference{
					Name: imageName,
				},
			},
			fakeObjects: []runtime.Object{
				&orcv1alpha1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      imageName,
						Namespace: namespace,
					},
					Status: orcv1alpha1.ImageStatus{
						Conditions: []metav1.Condition{
							{
								Type:   orcv1alpha1.ConditionAvailable,
								Status: metav1.ConditionFalse,
							},
							{
								Type:   orcv1alpha1.ConditionProgressing,
								Status: metav1.ConditionTrue,
								Reason: orcv1alpha1.ConditionReasonProgressing,
							},
						},
						ID: ptr.To(imageID),
					},
				},
			},
			want: nil,
		},
		{
			testName: "Image by reference exists, terminal failure",
			image: infrav1.ImageParam{
				ImageRef: &infrav1.ResourceReference{
					Name: imageName,
				},
			},
			fakeObjects: []runtime.Object{
				&orcv1alpha1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      imageName,
						Namespace: namespace,
					},
					Status: orcv1alpha1.ImageStatus{
						Conditions: []metav1.Condition{
							{
								Type:   orcv1alpha1.ConditionAvailable,
								Status: metav1.ConditionFalse,
							},
							{
								Type:    orcv1alpha1.ConditionProgressing,
								Status:  metav1.ConditionFalse,
								Reason:  orcv1alpha1.ConditionReasonUnrecoverableError,
								Message: "test error",
							},
						},
						ID: ptr.To(imageID),
					},
				},
			},
			want:              nil,
			wantTerminalError: true,
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
			if tt.expect != nil {
				tt.expect(mockScopeFactory.ImageClient.EXPECT())
			}

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.fakeObjects...).Build()

			got, err := s.GetImageID(context.TODO(), fakeClient, namespace, tt.image)

			if tt.wantTerminalError {
				tt.wantErr = true
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("Service.getImageID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			var terminalError *capoerrors.TerminalError
			if errors.As(err, &terminalError) != tt.wantTerminalError {
				t.Errorf("Terminal error: wanted = %v, got = %v", tt.wantTerminalError, !tt.wantTerminalError)
			}

			// NOTE(mdbooth): there must be a simpler way to write this!
			if (tt.want == nil && got != nil) || (tt.want != nil && (got == nil || *tt.want != *got)) {
				t.Errorf("Service.getImageID() = '%v', want '%v'", ptr.Deref(got, ""), ptr.Deref(tt.want, ""))
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
		FlavorID:   flavorUUID,
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
	getDefaultServerCreateOpts := func() servers.CreateOpts {
		return servers.CreateOpts{
			Name:             openStackMachineName,
			ImageRef:         imageUUID,
			FlavorRef:        flavorUUID,
			UserData:         []byte(base64.StdEncoding.EncodeToString([]byte("user-data"))),
			AvailabilityZone: failureDomain,
			Networks:         []servers.Network{{Port: portUUID}},
			Metadata: map[string]string{
				"test-metadata": "test-value",
			},
			ConfigDrive: ptr.To(true),
			Tags:        []string{"test-tag"},
			BlockDevice: []servers.BlockDevice{
				{
					SourceType:          "image",
					UUID:                imageUUID,
					DeleteOnTermination: true,
					DestinationType:     "local",
				},
			},
		}
	}

	withSSHKey := func(builder servers.CreateOptsBuilder) servers.CreateOptsBuilder {
		return keypairs.CreateOptsExt{
			CreateOptsBuilder: builder,
			KeyName:           sshKeyName,
		}
	}

	getDefaultSchedulerHintOpts := func() servers.SchedulerHintOpts {
		return servers.SchedulerHintOpts{
			Group: serverGroupUUID,
		}
	}

	returnedServer := func(status string) *servers.Server {
		return &servers.Server{
			ID:      instanceUUID,
			Name:    openStackMachineName,
			Status:  status,
			KeyName: sshKeyName,
		}
	}

	// Expected calls and custom match function for creating a server
	expectCreateServer := func(g Gomega, computeRecorder *mock.MockComputeClientMockRecorder, expectedCreateOpts servers.CreateOptsBuilder, expectedSchedulerHintOpts servers.SchedulerHintOptsBuilder, computeClient clients.ComputeClient, wantError bool) {
		computeRecorder.WithMicroversion(clients.NovaTagging).Return(computeClient, nil)
		computeRecorder.CreateServer(gomock.Any(), gomock.Any()).DoAndReturn(func(createOpts servers.CreateOptsBuilder, schedulerHintOpts servers.SchedulerHintOptsBuilder) (*servers.Server, error) {
			createOptsMap, _ := createOpts.ToServerCreateMap()
			expectedCreateOptsMap, _ := expectedCreateOpts.ToServerCreateMap()
			g.Expect(createOptsMap).To(Equal(expectedCreateOptsMap), cmp.Diff(createOptsMap, expectedCreateOptsMap))

			schedulerHintOptsMap, _ := schedulerHintOpts.ToSchedulerHintsMap()
			expectedSchedulerHintOptsMap, _ := expectedSchedulerHintOpts.ToSchedulerHintsMap()
			g.Expect(schedulerHintOptsMap).To(Equal(expectedSchedulerHintOptsMap), cmp.Diff(schedulerHintOptsMap, expectedSchedulerHintOptsMap))

			if wantError {
				return nil, fmt.Errorf("test error")
			}
			return returnedServer("BUILDING"), nil
		})
	}

	expectUnsupportedMicroversion := func(computeRecorder *mock.MockComputeClientMockRecorder) {
		computeRecorder.WithMicroversion(clients.NovaTagging).Return(nil, errors.New("unsupported microversion"))
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

	expectVolumeRequiresMultiattachCheck := func(volumeRecorder *mock.MockVolumeClientMockRecorder, uuid string, requiresMultiattach bool) {
		vol := returnedVolume(uuid, "available")
		vol.Multiattach = requiresMultiattach
		volumeRecorder.GetVolume(uuid).Return(returnedVolume(uuid, "available"), nil)
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
		expect          func(g Gomega, r *recorders, factory *scope.MockScopeFactory)
		wantErr         bool
	}{
		{
			name:            "Defaults",
			getInstanceSpec: getDefaultInstanceSpec,
			expect: func(g Gomega, r *recorders, factory *scope.MockScopeFactory) {
				expectCreateServer(g, r.compute, withSSHKey(getDefaultServerCreateOpts()), getDefaultSchedulerHintOpts(), factory.ComputeClient, false)
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
			expect: func(g Gomega, r *recorders, factory *scope.MockScopeFactory) {
				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-root", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:        50,
					Description: fmt.Sprintf("Root volume for %s", openStackMachineName),
					Name:        fmt.Sprintf("%s-root", openStackMachineName),
					ImageID:     imageUUID,
				}).Return(&volumes.Volume{ID: rootVolumeUUID}, nil)
				expectVolumePollSuccess(r.volume, rootVolumeUUID)
				expectVolumeRequiresMultiattachCheck(r.volume, rootVolumeUUID, false)

				createOpts := getDefaultServerCreateOpts()
				createOpts.ImageRef = ""
				createOpts.BlockDevice = []servers.BlockDevice{
					{
						SourceType:          "volume",
						UUID:                rootVolumeUUID,
						BootIndex:           0,
						DeleteOnTermination: true,
						DestinationType:     "volume",
					},
				}
				expectCreateServer(g, r.compute, withSSHKey(createOpts), getDefaultSchedulerHintOpts(), factory.ComputeClient, false)

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
			expect: func(g Gomega, r *recorders, factory *scope.MockScopeFactory) {
				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-root", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:             50,
					AvailabilityZone: "test-alternate-az",
					VolumeType:       "test-volume-type",
					Description:      fmt.Sprintf("Root volume for %s", openStackMachineName),
					Name:             fmt.Sprintf("%s-root", openStackMachineName),
					ImageID:          imageUUID,
				}).Return(&volumes.Volume{ID: rootVolumeUUID}, nil)
				expectVolumePollSuccess(r.volume, rootVolumeUUID)
				expectVolumeRequiresMultiattachCheck(r.volume, rootVolumeUUID, false)

				createOpts := getDefaultServerCreateOpts()
				createOpts.ImageRef = ""
				createOpts.BlockDevice = []servers.BlockDevice{
					{
						SourceType:          "volume",
						UUID:                rootVolumeUUID,
						BootIndex:           0,
						DeleteOnTermination: true,
						DestinationType:     "volume",
					},
				}
				expectCreateServer(g, r.compute, withSSHKey(createOpts), getDefaultSchedulerHintOpts(), factory.ComputeClient, false)

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
			expect: func(g Gomega, r *recorders, factory *scope.MockScopeFactory) {
				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-root", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:             50,
					AvailabilityZone: failureDomain,
					VolumeType:       "test-volume-type",
					Description:      fmt.Sprintf("Root volume for %s", openStackMachineName),
					Name:             fmt.Sprintf("%s-root", openStackMachineName),
					ImageID:          imageUUID,
				}).Return(&volumes.Volume{ID: rootVolumeUUID}, nil)
				expectVolumePollSuccess(r.volume, rootVolumeUUID)
				expectVolumeRequiresMultiattachCheck(r.volume, rootVolumeUUID, false)

				createOpts := getDefaultServerCreateOpts()
				createOpts.ImageRef = ""
				createOpts.BlockDevice = []servers.BlockDevice{
					{
						SourceType:          "volume",
						UUID:                rootVolumeUUID,
						BootIndex:           0,
						DeleteOnTermination: true,
						DestinationType:     "volume",
					},
				}
				expectCreateServer(g, r.compute, withSSHKey(createOpts), getDefaultSchedulerHintOpts(), factory.ComputeClient, false)

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
			expect: func(_ Gomega, r *recorders, _ *scope.MockScopeFactory) {
				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-root", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:        50,
					Description: fmt.Sprintf("Root volume for %s", openStackMachineName),
					Name:        fmt.Sprintf("%s-root", openStackMachineName),
					ImageID:     imageUUID,
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
			expect: func(g Gomega, r *recorders, factory *scope.MockScopeFactory) {
				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-root", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:        50,
					Description: fmt.Sprintf("Root volume for %s", openStackMachineName),
					Name:        fmt.Sprintf("%s-root", openStackMachineName),
					ImageID:     imageUUID,
				}).Return(&volumes.Volume{ID: rootVolumeUUID}, nil)
				expectVolumePollSuccess(r.volume, rootVolumeUUID)

				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-etcd", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:        50,
					Description: fmt.Sprintf("Additional block device for %s", openStackMachineName),
					Name:        fmt.Sprintf("%s-etcd", openStackMachineName),
					VolumeType:  "test-volume-type",
				}).Return(&volumes.Volume{ID: additionalBlockDeviceVolumeUUID}, nil)
				expectVolumePollSuccess(r.volume, additionalBlockDeviceVolumeUUID)

				expectVolumeRequiresMultiattachCheck(r.volume, rootVolumeUUID, false)
				expectVolumeRequiresMultiattachCheck(r.volume, additionalBlockDeviceVolumeUUID, false)

				createOpts := getDefaultServerCreateOpts()
				createOpts.ImageRef = ""
				createOpts.BlockDevice = []servers.BlockDevice{
					{
						SourceType:          "volume",
						UUID:                rootVolumeUUID,
						BootIndex:           0,
						DeleteOnTermination: true,
						DestinationType:     "volume",
					},
					{
						SourceType:          "volume",
						UUID:                additionalBlockDeviceVolumeUUID,
						BootIndex:           -1,
						DeleteOnTermination: true,
						DestinationType:     "volume",
						Tag:                 "etcd",
					},
					{
						SourceType:          "blank",
						BootIndex:           -1,
						DeleteOnTermination: true,
						DestinationType:     "local",
						VolumeSize:          10,
						Tag:                 "local-device",
					},
				}
				expectCreateServer(g, r.compute, withSSHKey(createOpts), getDefaultSchedulerHintOpts(), factory.ComputeClient, false)

				// Don't delete ports because the server is created: DeleteInstance will do it
			},
			wantErr: false,
		},
		{
			name: "Volume that is multiattach",
			getInstanceSpec: func() *InstanceSpec {
				s := getDefaultInstanceSpec()
				s.RootVolume = &infrav1.RootVolume{
					SizeGiB: 50,
				}
				s.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
					{
						Name:    "data",
						SizeGiB: 50,
						Storage: infrav1.BlockDeviceStorage{
							Type: "Volume",
							Volume: &infrav1.BlockDeviceVolume{
								Type: "multiattach-volume-type",
							},
						},
					},
				}
				return s
			},
			expect: func(g Gomega, r *recorders, factory *scope.MockScopeFactory) {
				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-root", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:        50,
					Description: fmt.Sprintf("Root volume for %s", openStackMachineName),
					Name:        fmt.Sprintf("%s-root", openStackMachineName),
					ImageID:     imageUUID,
				}).Return(&volumes.Volume{ID: rootVolumeUUID}, nil)
				expectVolumePollSuccess(r.volume, rootVolumeUUID)

				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-data", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:        50,
					Description: fmt.Sprintf("Additional block device for %s", openStackMachineName),
					Name:        fmt.Sprintf("%s-data", openStackMachineName),
					VolumeType:  "multiattach-volume-type",
				}).Return(&volumes.Volume{ID: additionalBlockDeviceVolumeUUID, Multiattach: true}, nil)
				expectVolumePollSuccess(r.volume, additionalBlockDeviceVolumeUUID)

				expectVolumeRequiresMultiattachCheck(r.volume, rootVolumeUUID, false)
				expectVolumeRequiresMultiattachCheck(r.volume, additionalBlockDeviceVolumeUUID, true)

				createOpts := getDefaultServerCreateOpts()
				createOpts.ImageRef = ""
				createOpts.BlockDevice = []servers.BlockDevice{
					{
						SourceType:          "volume",
						UUID:                rootVolumeUUID,
						BootIndex:           0,
						DeleteOnTermination: true,
						DestinationType:     "volume",
					},
					{
						SourceType:          "volume",
						UUID:                additionalBlockDeviceVolumeUUID,
						BootIndex:           -1,
						DeleteOnTermination: true,
						DestinationType:     "volume",
						Tag:                 "data",
					},
				}
				expectCreateServer(g, r.compute, withSSHKey(createOpts), getDefaultSchedulerHintOpts(), factory.ComputeClient, false)

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
			expect: func(g Gomega, r *recorders, factory *scope.MockScopeFactory) {
				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-etcd", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:        50,
					Description: fmt.Sprintf("Additional block device for %s", openStackMachineName),
					Name:        fmt.Sprintf("%s-etcd", openStackMachineName),
					VolumeType:  "test-volume-type",
				}).Return(&volumes.Volume{ID: additionalBlockDeviceVolumeUUID}, nil)
				expectVolumePollSuccess(r.volume, additionalBlockDeviceVolumeUUID)
				expectVolumeRequiresMultiattachCheck(r.volume, additionalBlockDeviceVolumeUUID, false)

				createOpts := getDefaultServerCreateOpts()
				createOpts.BlockDevice = []servers.BlockDevice{
					{
						SourceType:          "image",
						UUID:                imageUUID,
						BootIndex:           0,
						DeleteOnTermination: true,
						DestinationType:     "local",
					},
					{
						SourceType:          "volume",
						UUID:                additionalBlockDeviceVolumeUUID,
						BootIndex:           -1,
						DeleteOnTermination: true,
						DestinationType:     "volume",
						Tag:                 "etcd",
					},
					{
						SourceType:          "blank",
						BootIndex:           -1,
						DeleteOnTermination: true,
						DestinationType:     "local",
						VolumeSize:          10,
						Tag:                 "data",
					},
				}
				expectCreateServer(g, r.compute, withSSHKey(createOpts), getDefaultSchedulerHintOpts(), factory.ComputeClient, false)

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
			expect: func(g Gomega, r *recorders, factory *scope.MockScopeFactory) {
				r.volume.ListVolumes(volumes.ListOpts{Name: fmt.Sprintf("%s-etcd", openStackMachineName)}).
					Return([]volumes.Volume{}, nil)
				r.volume.CreateVolume(volumes.CreateOpts{
					Size:             50,
					AvailabilityZone: "test-alternate-az",
					Description:      fmt.Sprintf("Additional block device for %s", openStackMachineName),
					Name:             fmt.Sprintf("%s-etcd", openStackMachineName),
					VolumeType:       "test-volume-type",
				}).Return(&volumes.Volume{ID: additionalBlockDeviceVolumeUUID}, nil)
				expectVolumePollSuccess(r.volume, additionalBlockDeviceVolumeUUID)
				expectVolumeRequiresMultiattachCheck(r.volume, additionalBlockDeviceVolumeUUID, false)

				createOpts := getDefaultServerCreateOpts()
				createOpts.BlockDevice = []servers.BlockDevice{
					{
						SourceType:          "image",
						UUID:                imageUUID,
						BootIndex:           0,
						DeleteOnTermination: true,
						DestinationType:     "local",
					},
					{
						SourceType:          "volume",
						UUID:                additionalBlockDeviceVolumeUUID,
						BootIndex:           -1,
						DeleteOnTermination: true,
						DestinationType:     "volume",
						Tag:                 "etcd",
					},
				}
				expectCreateServer(g, r.compute, withSSHKey(createOpts), getDefaultSchedulerHintOpts(), factory.ComputeClient, false)

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
			expect: func(_ Gomega, _ *recorders, _ *scope.MockScopeFactory) {
			},
			wantErr: true,
		},
		{
			name: "With custom scheduler hint bool",
			getInstanceSpec: func() *InstanceSpec {
				s := getDefaultInstanceSpec()
				s.SchedulerAdditionalProperties = []infrav1.SchedulerHintAdditionalProperty{
					{
						Name: "custom_hint",
						Value: infrav1.SchedulerHintAdditionalValue{
							Type: infrav1.SchedulerHintTypeBool,
							Bool: ptr.To(true),
						},
					},
				}
				return s
			},
			expect: func(g Gomega, r *recorders, factory *scope.MockScopeFactory) {
				createOpts := getDefaultServerCreateOpts()
				schedulerHintOpts := servers.SchedulerHintOpts{
					Group: serverGroupUUID,
					AdditionalProperties: map[string]any{
						"custom_hint": true,
					},
				}
				expectCreateServer(g, r.compute, withSSHKey(createOpts), schedulerHintOpts, factory.ComputeClient, false)
			},
			wantErr: false,
		},
		{
			name: "With custom scheduler hint number",
			getInstanceSpec: func() *InstanceSpec {
				s := getDefaultInstanceSpec()
				s.SchedulerAdditionalProperties = []infrav1.SchedulerHintAdditionalProperty{
					{
						Name: "custom_hint",
						Value: infrav1.SchedulerHintAdditionalValue{
							Type:   infrav1.SchedulerHintTypeNumber,
							Number: ptr.To(1),
						},
					},
				}
				return s
			},
			expect: func(g Gomega, r *recorders, factory *scope.MockScopeFactory) {
				createOpts := getDefaultServerCreateOpts()
				schedulerHintOpts := servers.SchedulerHintOpts{
					Group: serverGroupUUID,
					AdditionalProperties: map[string]any{
						"custom_hint": 1,
					},
				}
				expectCreateServer(g, r.compute, withSSHKey(createOpts), schedulerHintOpts, factory.ComputeClient, false)
			},
			wantErr: false,
		},
		{
			name: "With custom scheduler hint string",
			getInstanceSpec: func() *InstanceSpec {
				s := getDefaultInstanceSpec()
				s.SchedulerAdditionalProperties = []infrav1.SchedulerHintAdditionalProperty{
					{
						Name: "custom_hint",
						Value: infrav1.SchedulerHintAdditionalValue{
							Type:   infrav1.SchedulerHintTypeString,
							String: ptr.To("custom hint"),
						},
					},
				}
				return s
			},
			expect: func(g Gomega, r *recorders, factory *scope.MockScopeFactory) {
				createOpts := getDefaultServerCreateOpts()
				schedulerHintOpts := servers.SchedulerHintOpts{
					Group: serverGroupUUID,
					AdditionalProperties: map[string]any{
						"custom_hint": "custom hint",
					},
				}
				expectCreateServer(g, r.compute, withSSHKey(createOpts), schedulerHintOpts, factory.ComputeClient, false)
			},
			wantErr: false,
		},
		{
			name:            "Unsupported Nova microversion",
			getInstanceSpec: getDefaultInstanceSpec,
			expect: func(_ Gomega, r *recorders, _ *scope.MockScopeFactory) {
				expectUnsupportedMicroversion(r.compute)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			mockCtrl := gomock.NewController(t)
			log := testr.New(t)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")

			computeRecorder := mockScopeFactory.ComputeClient.EXPECT()
			imageRecorder := mockScopeFactory.ImageClient.EXPECT()
			networkRecorder := mockScopeFactory.NetworkClient.EXPECT()
			volumeRecorder := mockScopeFactory.VolumeClient.EXPECT()

			tt.expect(g, &recorders{computeRecorder, imageRecorder, networkRecorder, volumeRecorder}, mockScopeFactory)

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
