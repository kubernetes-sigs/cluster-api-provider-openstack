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
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking/mock_networking"
)

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
			args: args{"test-1-instance", &infrav1.PortOpts{NameSuffix: "foo2", NetworkID: "bar", DisablePortSecurity: pointer.Bool(true)}, 4},
			want: "test-1-instance-foo2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPortName(tt.args.instanceName, tt.args.opts, tt.args.netIndex); got != tt.want {
				t.Errorf("getPortName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_getServerNetworks(t *testing.T) {
	const testClusterTag = "cluster=mycluster"

	// Network A:
	//  Network is tagged
	//  Has 3 subnets
	//  Subnets A1 and A2 are tagged
	//  Subnet A3 is not tagged
	// Network B:
	//  Network is tagged
	//  Has 1 subnet, B1, which is also tagged
	// Network C:
	//  Network is not tagged
	//  Has 1 subnet, C1, which is also not tagged

	networkAUUID := "7f0a7cc9-d7c8-41d2-87a2-2fc7f5ec544e"
	networkBUUID := "607559d9-a5a4-4a0b-a92d-75eba89e3343"
	networkCUUID := "9d7b0284-b22e-4bc7-b90e-28a652cac7cc"
	subnetA1UUID := "869f6790-17a9-44d5-83a1-89e180514515"
	subnetA2UUID := "bd926900-5277-47a5-bd71-c6f713165dbd"
	subnetA3UUID := "79dfde1b-07f1-48a0-97fd-07e2f6018c46"
	subnetB1UUID := "efc2cc7d-c6e0-45c6-8147-0e08b8530664"
	subnetC1UUID := "b33271f4-6bb1-430a-88bf-789394815aaf"

	testNetworkA := networks.Network{
		ID:      networkAUUID,
		Name:    "network-a",
		Subnets: []string{subnetA1UUID, subnetA2UUID},
		Tags:    []string{testClusterTag},
	}
	testNetworkB := networks.Network{
		ID:      networkBUUID,
		Name:    "network-b",
		Subnets: []string{subnetB1UUID},
		Tags:    []string{testClusterTag},
	}

	testSubnetA1 := subnets.Subnet{
		ID:        subnetA1UUID,
		Name:      "subnet-a1",
		NetworkID: networkAUUID,
		Tags:      []string{testClusterTag},
	}
	testSubnetA2 := subnets.Subnet{
		ID:        subnetA2UUID,
		Name:      "subnet-a2",
		NetworkID: networkAUUID,
		Tags:      []string{testClusterTag},
	}
	testSubnetB1 := subnets.Subnet{
		ID:        subnetB1UUID,
		Name:      "subnet-b1",
		NetworkID: networkBUUID,
		Tags:      []string{testClusterTag},
	}

	// Define arbitrary test network and subnet filters for use in multiple tests,
	// the gophercloud ListOpts they should translate to, and the arbitrary returned networks/subnets.
	testNetworkFilter := infrav1.NetworkFilter{Tags: testClusterTag}
	testNetworkListOpts := networks.ListOpts{Tags: testClusterTag}
	testSubnetFilter := infrav1.SubnetFilter{Tags: testClusterTag}
	testSubnetListOpts := subnets.ListOpts{Tags: testClusterTag}

	tests := []struct {
		name          string
		networkParams []infrav1.NetworkParam
		want          []infrav1.Network
		expect        func(m *mock_networking.MockNetworkClientMockRecorder)
		wantErr       bool
	}{
		{
			name: "Network UUID without subnet",
			networkParams: []infrav1.NetworkParam{
				{UUID: networkAUUID},
			},
			want: []infrav1.Network{
				{ID: networkAUUID, Subnet: &infrav1.Subnet{}},
			},
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
			},
			wantErr: false,
		},
		{
			name: "Network filter without subnet",
			networkParams: []infrav1.NetworkParam{
				{Filter: testNetworkFilter},
			},
			want: []infrav1.Network{
				{ID: networkAUUID, Subnet: &infrav1.Subnet{}},
				{ID: networkBUUID, Subnet: &infrav1.Subnet{}},
			},
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
				// List tagged networks (A & B)
				m.ListNetwork(&testNetworkListOpts).
					Return([]networks.Network{testNetworkA, testNetworkB}, nil)
			},
			wantErr: false,
		},
		{
			name: "Subnet by filter without network",
			networkParams: []infrav1.NetworkParam{
				{
					Subnets: []infrav1.SubnetParam{{Filter: testSubnetFilter}},
				},
			},
			want: []infrav1.Network{
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA1UUID}},
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA2UUID}},
				{ID: networkBUUID, Subnet: &infrav1.Subnet{ID: subnetB1UUID}},
			},
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
				// List all tagged subnets in any network (A1, A2, and B1)
				m.ListSubnet(&testSubnetListOpts).
					Return([]subnets.Subnet{testSubnetA1, testSubnetA2, testSubnetB1}, nil)
			},
			wantErr: false,
		},
		{
			name: "Network UUID and subnet filter",
			networkParams: []infrav1.NetworkParam{
				{
					UUID: networkAUUID,
					Subnets: []infrav1.SubnetParam{
						{Filter: testSubnetFilter},
					},
				},
			},
			want: []infrav1.Network{
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA1UUID}},
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA2UUID}},
			},
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
				// List tagged subnets in network A (A1 & A2)
				networkAFilter := testSubnetListOpts
				networkAFilter.NetworkID = networkAUUID
				m.ListSubnet(&networkAFilter).
					Return([]subnets.Subnet{testSubnetA1, testSubnetA2}, nil)
			},
			wantErr: false,
		},
		{
			name: "Network UUID and subnet UUID",
			networkParams: []infrav1.NetworkParam{
				{
					UUID: networkAUUID,
					Subnets: []infrav1.SubnetParam{
						{UUID: subnetA1UUID},
					},
				},
			},
			want: []infrav1.Network{
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA1UUID}},
			},
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
			},
			wantErr: false,
		},
		{
			name: "Network UUID and multiple subnet params",
			networkParams: []infrav1.NetworkParam{
				{
					UUID: networkAUUID,
					Subnets: []infrav1.SubnetParam{
						{UUID: subnetA3UUID},
						{Filter: testSubnetFilter},
					},
				},
			},
			want: []infrav1.Network{
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA3UUID}},
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA1UUID}},
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA2UUID}},
			},
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
				// List tagged subnets in network A
				networkAFilter := testSubnetListOpts
				networkAFilter.NetworkID = networkAUUID
				m.ListSubnet(&networkAFilter).
					Return([]subnets.Subnet{testSubnetA1, testSubnetA2}, nil)
			},
			wantErr: false,
		},
		{
			name: "Multiple network params",
			networkParams: []infrav1.NetworkParam{
				{
					UUID: networkCUUID,
					Subnets: []infrav1.SubnetParam{
						{UUID: subnetC1UUID},
					},
				},
				{
					Filter: testNetworkFilter,
					Subnets: []infrav1.SubnetParam{
						{Filter: testSubnetFilter},
					},
				},
			},
			want: []infrav1.Network{
				{ID: networkCUUID, Subnet: &infrav1.Subnet{ID: subnetC1UUID}},
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA1UUID}},
				{ID: networkAUUID, Subnet: &infrav1.Subnet{ID: subnetA2UUID}},
				{ID: networkBUUID, Subnet: &infrav1.Subnet{ID: subnetB1UUID}},
			},
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
				// List tagged networks (A & B)
				m.ListNetwork(&testNetworkListOpts).
					Return([]networks.Network{testNetworkA, testNetworkB}, nil)

				// List tagged subnets in network A (A1 & A2)
				networkAFilter := testSubnetListOpts
				networkAFilter.NetworkID = networkAUUID
				m.ListSubnet(&networkAFilter).
					Return([]subnets.Subnet{testSubnetA1, testSubnetA2}, nil)

				// List tagged subnets in network B (B1)
				networkBFilter := testSubnetListOpts
				networkBFilter.NetworkID = networkBUUID
				m.ListSubnet(&networkBFilter).
					Return([]subnets.Subnet{testSubnetB1}, nil)
			},
			wantErr: false,
		},
		{
			// Expect an error if a network filter doesn't match any networks
			name: "Network filter matches no networks",
			networkParams: []infrav1.NetworkParam{
				{Filter: testNetworkFilter},
			},
			want: nil,
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
				// List tagged networks (none for this test)
				m.ListNetwork(&testNetworkListOpts).Return([]networks.Network{}, nil)
			},
			wantErr: true,
		},
		{
			// Expect an error if a subnet filter doesn't match any subnets
			name: "Subnet filter matches no subnets",
			networkParams: []infrav1.NetworkParam{
				{
					UUID: networkAUUID,
					Subnets: []infrav1.SubnetParam{
						{Filter: testSubnetFilter},
					},
				},
			},
			want: nil,
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
				// List tagged subnets in network A
				networkAFilter := testSubnetListOpts
				networkAFilter.NetworkID = networkAUUID
				m.ListSubnet(&networkAFilter).Return([]subnets.Subnet{}, nil)
			},
			wantErr: true,
		},
		{
			name: "Subnet UUID without network",
			networkParams: []infrav1.NetworkParam{
				{Subnets: []infrav1.SubnetParam{
					{UUID: subnetA1UUID},
				}},
			},
			want: nil,
			expect: func(m *mock_networking.MockNetworkClientMockRecorder) {
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			mockNetworkClient := mock_networking.NewMockNetworkClient(mockCtrl)
			tt.expect(mockNetworkClient.EXPECT())

			networkingService := networking.NewTestService(
				"", mockNetworkClient, logr.Discard(),
			)
			s := &Service{
				networkingService: networkingService,
			}

			got, err := s.getServerNetworks(tt.networkParams)
			g := NewWithT(t)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
			g.Expect(got).To(Equal(tt.want))
		})
	}
}

func TestService_getImageID(t *testing.T) {
	const imageIDA = "ce96e584-7ebc-46d6-9e55-987d72e3806c"
	const imageIDB = "8f536889-5198-42d7-8314-cb78f4f4755c"

	tests := []struct {
		testName  string
		imageName string
		expect    func(m *MockClientMockRecorder)
		want      string
		wantErr   bool
	}{
		{
			testName:  "Return image ID",
			imageName: "test-image",
			expect: func(m *MockClientMockRecorder) {
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
			expect: func(m *MockClientMockRecorder) {
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
			expect: func(m *MockClientMockRecorder) {
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
			expect: func(m *MockClientMockRecorder) {
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
			mockComputeClient := NewMockClient(mockCtrl)
			tt.expect(mockComputeClient.EXPECT())

			s := Service{
				projectID:         "",
				computeService:    mockComputeClient,
				networkingService: &networking.Service{},
				logger:            logr.Discard(),
			}

			got, err := s.getImageID(tt.imageName)
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
	networkUUID                   = "d412171b-9fd7-41c1-95a6-c24e5953974d"
	subnetUUID                    = "d2d8d98d-b234-477e-a547-868b7cb5d6a5"
	portUUID                      = "e7b7f3d1-0a81-40b1-bfa6-a22a31b17816"
	imageUUID                     = "652b5a05-27fa-41d4-ac82-3e63cf6f7ab7"
	flavorUUID                    = "6dc820db-f912-454e-a1e3-1081f3b8cc72"
	instanceUUID                  = "383a8ec1-b6ea-4493-99dd-fc790da04ba9"
	extraSecurityGroupUUID        = "514bb2d8-3390-4a3b-86a7-7864ba57b329"
	controlPlaneSecurityGroupUUID = "c9817a91-4821-42db-8367-2301002ab659"
	workerSecurityGroupUUID       = "9c6c0d28-03c9-436c-815d-58440ac2c1c8"
	serverGroupUUID               = "7b940d62-68ef-4e42-a76a-1a62e290509c"

	openStackMachineName = "test-openstack-machine"
	portName             = "test-openstack-machine-0"
	namespace            = "test-namespace"
	imageName            = "test-image"
	flavorName           = "test-flavor"
	sshKeyName           = "test-ssh-key"
)

var failureDomain = "test-failure-domain"

func getDefaultOpenStackCluster() *infrav1.OpenStackCluster {
	return &infrav1.OpenStackCluster{
		Spec: infrav1.OpenStackClusterSpec{},
		Status: infrav1.OpenStackClusterStatus{
			Network: &infrav1.Network{
				ID: networkUUID,
				Subnet: &infrav1.Subnet{
					ID: subnetUUID,
				},
			},
			ControlPlaneSecurityGroup: &infrav1.SecurityGroup{ID: controlPlaneSecurityGroupUUID},
			WorkerSecurityGroup:       &infrav1.SecurityGroup{ID: workerSecurityGroupUUID},
		},
	}
}

func getDefaultMachine() *clusterv1.Machine {
	return &clusterv1.Machine{
		Spec: clusterv1.MachineSpec{
			FailureDomain: &failureDomain,
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
			CloudName:  "test-cloud",
			Flavor:     flavorName,
			Image:      imageName,
			SSHKeyName: sshKeyName,
			Tags:       []string{"test-tag"},
			ServerMetadata: map[string]string{
				"test-metadata": "test-value",
			},
			ConfigDrive:   pointer.BoolPtr(true),
			ServerGroupID: serverGroupUUID,
		},
	}
}

func TestService_CreateInstance(t *testing.T) {
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
			},
			"os:scheduler_hints": map[string]interface{}{
				"group": serverGroupUUID,
			},
		}
	}

	returnedServer := func(status string) *ServerExt {
		return &ServerExt{
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
	expectCreateDefaultPort := func(networkRecorder *mock_networking.MockNetworkClientMockRecorder) {
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

	// Expected calls if we delete the network port
	expectCleanupDefaultPort := func(networkRecorder *mock_networking.MockNetworkClientMockRecorder) {
		networkRecorder.GetPort(portUUID).Return(&ports.Port{ID: portUUID, Name: portName}, nil)
		networkRecorder.DeletePort(portUUID).Return(nil)
	}

	// Expected calls when using default image and flavor
	expectDefaultImageAndFlavor := func(computeRecorder *MockClientMockRecorder) {
		computeRecorder.ListImages(images.ListOpts{Name: imageName}).Return([]images.Image{{ID: imageUUID}}, nil)
		computeRecorder.GetFlavorIDFromName(flavorName).Return(flavorUUID, nil)
	}

	// Expected calls and custom match function for creating a server
	expectCreateServer := func(computeRecorder *MockClientMockRecorder, expectedCreateOpts map[string]interface{}, wantError bool) {
		// This nonsense is because ConfigDrive is a bool pointer, so we
		// can't assert its exact contents with gomock.
		// Instead we call ToServerCreateMap() on it to obtain a
		// map[string]interface{} of the create options, and then use
		// gomega to assert the contents of the map, which is more flexible.

		computeRecorder.CreateServer(gomock.Any()).DoAndReturn(func(createOpts servers.CreateOptsBuilder) (*ServerExt, error) {
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
	expectServerPoll := func(computeRecorder *MockClientMockRecorder, states []string) {
		for _, state := range states {
			computeRecorder.GetServer(instanceUUID).Return(returnedServer(state), nil)
		}
	}

	expectServerPollSuccess := func(computeRecorder *MockClientMockRecorder) {
		expectServerPoll(computeRecorder, []string{"ACTIVE"})
	}

	// *******************
	// START OF TEST CASES
	// *******************

	tests := []struct {
		name                string
		getMachine          func() *clusterv1.Machine
		getOpenStackCluster func() *infrav1.OpenStackCluster
		getOpenStackMachine func() *infrav1.OpenStackMachine
		expect              func(computeRecorder *MockClientMockRecorder, networkRecorder *mock_networking.MockNetworkClientMockRecorder)
		wantErr             bool
	}{
		{
			name:                "Defaults",
			getMachine:          getDefaultMachine,
			getOpenStackCluster: getDefaultOpenStackCluster,
			getOpenStackMachine: getDefaultOpenStackMachine,
			expect: func(computeRecorder *MockClientMockRecorder, networkRecorder *mock_networking.MockNetworkClientMockRecorder) {
				expectCreateDefaultPort(networkRecorder)
				expectDefaultImageAndFlavor(computeRecorder)

				expectCreateServer(computeRecorder, getDefaultServerMap(), false)
				expectServerPollSuccess(computeRecorder)
			},
			wantErr: false,
		},
		{
			name:                "Delete ports on image error",
			getMachine:          getDefaultMachine,
			getOpenStackCluster: getDefaultOpenStackCluster,
			getOpenStackMachine: getDefaultOpenStackMachine,
			expect: func(computeRecorder *MockClientMockRecorder, networkRecorder *mock_networking.MockNetworkClientMockRecorder) {
				expectCreateDefaultPort(networkRecorder)

				computeRecorder.ListImages(images.ListOpts{Name: imageName}).Return(nil, fmt.Errorf("test error"))

				expectCleanupDefaultPort(networkRecorder)
			},
			wantErr: true,
		},
		{
			name:                "Delete ports on flavor error",
			getMachine:          getDefaultMachine,
			getOpenStackCluster: getDefaultOpenStackCluster,
			getOpenStackMachine: getDefaultOpenStackMachine,
			expect: func(computeRecorder *MockClientMockRecorder, networkRecorder *mock_networking.MockNetworkClientMockRecorder) {
				expectCreateDefaultPort(networkRecorder)

				computeRecorder.ListImages(images.ListOpts{Name: imageName}).Return([]images.Image{{ID: imageUUID}}, nil)
				computeRecorder.GetFlavorIDFromName(flavorName).Return("", fmt.Errorf("test error"))

				expectCleanupDefaultPort(networkRecorder)
			},
			wantErr: true,
		},
		{
			name:                "Delete ports on server create error",
			getMachine:          getDefaultMachine,
			getOpenStackCluster: getDefaultOpenStackCluster,
			getOpenStackMachine: getDefaultOpenStackMachine,
			expect: func(computeRecorder *MockClientMockRecorder, networkRecorder *mock_networking.MockNetworkClientMockRecorder) {
				expectCreateDefaultPort(networkRecorder)
				expectDefaultImageAndFlavor(computeRecorder)

				expectCreateServer(computeRecorder, getDefaultServerMap(), true)

				// Make sure we delete ports
				expectCleanupDefaultPort(networkRecorder)
			},
			wantErr: true,
		},
		{
			name:                "Delete previously created ports on port creation error",
			getMachine:          getDefaultMachine,
			getOpenStackCluster: getDefaultOpenStackCluster,
			getOpenStackMachine: func() *infrav1.OpenStackMachine {
				m := getDefaultOpenStackMachine()
				m.Spec.Ports = []infrav1.PortOpts{
					{Description: "Test port 0"},
					{Description: "Test port 1"},
				}
				return m
			},
			expect: func(computeRecorder *MockClientMockRecorder, networkRecorder *mock_networking.MockNetworkClientMockRecorder) {
				expectCreateDefaultPort(networkRecorder)

				// Looking up the second port fails
				networkRecorder.ListPort(ports.ListOpts{
					Name:      "test-openstack-machine-1",
					NetworkID: networkUUID,
				}).Return(nil, fmt.Errorf("test error"))

				// We should cleanup the first port
				expectCleanupDefaultPort(networkRecorder)
			},
			wantErr: true,
		},
		{
			name:                "Poll until server is created",
			getMachine:          getDefaultMachine,
			getOpenStackCluster: getDefaultOpenStackCluster,
			getOpenStackMachine: getDefaultOpenStackMachine,
			expect: func(computeRecorder *MockClientMockRecorder, networkRecorder *mock_networking.MockNetworkClientMockRecorder) {
				expectCreateDefaultPort(networkRecorder)
				expectDefaultImageAndFlavor(computeRecorder)

				expectCreateServer(computeRecorder, getDefaultServerMap(), false)
				expectServerPoll(computeRecorder, []string{"BUILDING", "ACTIVE"})
			},
			wantErr: false,
		},
		{
			name:                "Server errors during creation",
			getMachine:          getDefaultMachine,
			getOpenStackCluster: getDefaultOpenStackCluster,
			getOpenStackMachine: getDefaultOpenStackMachine,
			expect: func(computeRecorder *MockClientMockRecorder, networkRecorder *mock_networking.MockNetworkClientMockRecorder) {
				expectCreateDefaultPort(networkRecorder)
				expectDefaultImageAndFlavor(computeRecorder)

				expectCreateServer(computeRecorder, getDefaultServerMap(), false)
				expectServerPoll(computeRecorder, []string{"BUILDING", "ERROR"})

				// Don't delete ports because the server is created: DeleteInstance will do it
			},
			wantErr: true,
		},
		{
			name: "Set control plane security group",
			getMachine: func() *clusterv1.Machine {
				machine := getDefaultMachine()
				machine.Labels = map[string]string{
					clusterv1.MachineControlPlaneLabelName: "true",
				}
				return machine
			},
			getOpenStackCluster: func() *infrav1.OpenStackCluster {
				osCluster := getDefaultOpenStackCluster()
				osCluster.Spec.ManagedSecurityGroups = true
				return osCluster
			},
			getOpenStackMachine: getDefaultOpenStackMachine,
			expect: func(computeRecorder *MockClientMockRecorder, networkRecorder *mock_networking.MockNetworkClientMockRecorder) {
				expectCreateDefaultPort(networkRecorder)
				expectDefaultImageAndFlavor(computeRecorder)

				createMap := getDefaultServerMap()
				serverMap := createMap["server"].(map[string]interface{})
				serverMap["security_groups"] = []map[string]interface{}{
					{"name": controlPlaneSecurityGroupUUID},
				}
				expectCreateServer(computeRecorder, createMap, false)
				expectServerPollSuccess(computeRecorder)
			},
			wantErr: false,
		},
		{
			name:       "Set worker security group",
			getMachine: getDefaultMachine,
			getOpenStackCluster: func() *infrav1.OpenStackCluster {
				osCluster := getDefaultOpenStackCluster()
				osCluster.Spec.ManagedSecurityGroups = true
				return osCluster
			},
			getOpenStackMachine: getDefaultOpenStackMachine,
			expect: func(computeRecorder *MockClientMockRecorder, networkRecorder *mock_networking.MockNetworkClientMockRecorder) {
				expectCreateDefaultPort(networkRecorder)
				expectDefaultImageAndFlavor(computeRecorder)

				createMap := getDefaultServerMap()
				serverMap := createMap["server"].(map[string]interface{})
				serverMap["security_groups"] = []map[string]interface{}{
					{"name": workerSecurityGroupUUID},
				}
				expectCreateServer(computeRecorder, createMap, false)
				expectServerPollSuccess(computeRecorder)
			},
			wantErr: false,
		},
		{
			name:       "Set extra security group",
			getMachine: getDefaultMachine,
			getOpenStackCluster: func() *infrav1.OpenStackCluster {
				osCluster := getDefaultOpenStackCluster()
				osCluster.Spec.ManagedSecurityGroups = true
				return osCluster
			},
			getOpenStackMachine: func() *infrav1.OpenStackMachine {
				osMachine := getDefaultOpenStackMachine()
				osMachine.Spec.SecurityGroups = []infrav1.SecurityGroupParam{{UUID: extraSecurityGroupUUID}}
				return osMachine
			},
			expect: func(computeRecorder *MockClientMockRecorder, networkRecorder *mock_networking.MockNetworkClientMockRecorder) {
				expectCreateDefaultPort(networkRecorder)
				expectDefaultImageAndFlavor(computeRecorder)

				// TODO: Shortcut this API call if security groups are passed by UUID
				networkRecorder.ListSecGroup(groups.ListOpts{ID: extraSecurityGroupUUID}).
					Return([]groups.SecGroup{{ID: extraSecurityGroupUUID}}, nil)

				createMap := getDefaultServerMap()
				serverMap := createMap["server"].(map[string]interface{})
				serverMap["security_groups"] = []map[string]interface{}{
					{"name": extraSecurityGroupUUID},
					{"name": workerSecurityGroupUUID},
				}
				expectCreateServer(computeRecorder, createMap, false)
				expectServerPollSuccess(computeRecorder)
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			mockComputeClient := NewMockClient(mockCtrl)
			mockNetworkClient := mock_networking.NewMockNetworkClient(mockCtrl)

			computeRecorder := mockComputeClient.EXPECT()
			networkRecorder := mockNetworkClient.EXPECT()

			tt.expect(computeRecorder, networkRecorder)

			s := Service{
				projectID:      "",
				computeService: mockComputeClient,
				networkingService: networking.NewTestService(
					"", mockNetworkClient, logr.Discard(),
				),
				logger: logr.Discard(),
			}
			// Call CreateInstance with a reduced retry interval to speed up the test
			_, err := s.createInstanceImpl(tt.getOpenStackCluster(), tt.getMachine(), tt.getOpenStackMachine(), "cluster-name", "user-data", time.Second)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.CreateInstance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
