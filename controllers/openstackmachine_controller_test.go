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

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/compute"
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
			FailureDomain: pointer.String(failureDomain),
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
			Image:      infrav1.ImageFilter{ID: pointer.String(imageUUID)},
			SSHKeyName: sshKeyName,
			Tags:       []string{"test-tag"},
			ServerMetadata: []infrav1.ServerMetadata{
				{Key: "test-metadata", Value: "test-value"},
			},
			ConfigDrive:    pointer.Bool(true),
			SecurityGroups: []infrav1.SecurityGroupFilter{},
			ServerGroup:    &infrav1.ServerGroupFilter{ID: serverGroupUUID},
		},
		Status: infrav1.OpenStackMachineStatus{
			Resolved: infrav1.ResolvedMachineSpec{
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
		ConfigDrive:   *pointer.Bool(true),
		FailureDomain: *pointer.String(failureDomain),
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
			got := machineToInstanceSpec(tt.openStackCluster(), tt.machine(), tt.openStackMachine(), "user-data")
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
