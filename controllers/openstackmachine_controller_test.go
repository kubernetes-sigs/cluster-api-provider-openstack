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

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha8"
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
			CloudName:  "test-cloud",
			Flavor:     flavorName,
			Image:      infrav1.ImageFilter{ID: imageUUID},
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
			ReferencedResources: infrav1.ReferencedMachineResources{
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
		ConfigDrive:    *pointer.Bool(true),
		FailureDomain:  *pointer.String(failureDomain),
		ServerGroupID:  serverGroupUUID,
		SecurityGroups: []infrav1.SecurityGroupFilter{},
		Tags:           []string{"test-tag"},
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
			name: "Control plane security group",
			openStackCluster: func() *infrav1.OpenStackCluster {
				c := getDefaultOpenStackCluster()
				c.Spec.ManagedSecurityGroups = &infrav1.ManagedSecurityGroups{}
				return c
			},
			machine: func() *clusterv1.Machine {
				m := getDefaultMachine()
				m.Labels = map[string]string{
					clusterv1.MachineControlPlaneLabel: "true",
				}
				return m
			},
			openStackMachine: getDefaultOpenStackMachine,
			wantInstanceSpec: func() *compute.InstanceSpec {
				i := getDefaultInstanceSpec()
				i.SecurityGroups = []infrav1.SecurityGroupFilter{{ID: controlPlaneSecurityGroupUUID}}
				return i
			},
		},
		{
			name: "Worker security group",
			openStackCluster: func() *infrav1.OpenStackCluster {
				c := getDefaultOpenStackCluster()
				c.Spec.ManagedSecurityGroups = &infrav1.ManagedSecurityGroups{}
				return c
			},
			machine:          getDefaultMachine,
			openStackMachine: getDefaultOpenStackMachine,
			wantInstanceSpec: func() *compute.InstanceSpec {
				i := getDefaultInstanceSpec()
				i.SecurityGroups = []infrav1.SecurityGroupFilter{{ID: workerSecurityGroupUUID}}
				return i
			},
		},
		{
			name: "Control plane security group not applied to worker",
			openStackCluster: func() *infrav1.OpenStackCluster {
				c := getDefaultOpenStackCluster()
				c.Spec.ManagedSecurityGroups = &infrav1.ManagedSecurityGroups{}
				c.Status.WorkerSecurityGroup = nil
				return c
			},
			machine:          getDefaultMachine,
			openStackMachine: getDefaultOpenStackMachine,
			wantInstanceSpec: func() *compute.InstanceSpec {
				i := getDefaultInstanceSpec()
				i.SecurityGroups = []infrav1.SecurityGroupFilter{}
				return i
			},
		},
		{
			name: "Worker security group not applied to control plane",
			openStackCluster: func() *infrav1.OpenStackCluster {
				c := getDefaultOpenStackCluster()
				c.Spec.ManagedSecurityGroups = &infrav1.ManagedSecurityGroups{}
				c.Status.ControlPlaneSecurityGroup = nil
				return c
			},
			machine: func() *clusterv1.Machine {
				m := getDefaultMachine()
				m.Labels = map[string]string{
					clusterv1.MachineControlPlaneLabel: "true",
				}
				return m
			},
			openStackMachine: getDefaultOpenStackMachine,
			wantInstanceSpec: func() *compute.InstanceSpec {
				i := getDefaultInstanceSpec()
				i.SecurityGroups = []infrav1.SecurityGroupFilter{}
				return i
			},
		},
		{
			name: "Extra security group",
			openStackCluster: func() *infrav1.OpenStackCluster {
				c := getDefaultOpenStackCluster()
				c.Spec.ManagedSecurityGroups = &infrav1.ManagedSecurityGroups{}
				return c
			},
			machine: getDefaultMachine,
			openStackMachine: func() *infrav1.OpenStackMachine {
				m := getDefaultOpenStackMachine()
				m.Spec.SecurityGroups = []infrav1.SecurityGroupFilter{{ID: extraSecurityGroupUUID}}
				return m
			},
			wantInstanceSpec: func() *compute.InstanceSpec {
				i := getDefaultInstanceSpec()
				i.SecurityGroups = []infrav1.SecurityGroupFilter{
					{ID: extraSecurityGroupUUID},
					{ID: workerSecurityGroupUUID},
				}
				return i
			},
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
			got := machineToInstanceSpec(tt.openStackCluster(), tt.machine(), tt.openStackMachine(), "user-data")
			Expect(got).To(Equal(tt.wantInstanceSpec()))
		})
	}
}

func Test_getInstanceTags(t *testing.T) {
	tests := []struct {
		name             string
		openStackMachine func() *infrav1.OpenStackMachine
		openStackCluster func() *infrav1.OpenStackCluster
		wantMachineTags  []string
	}{
		{
			name: "No tags",
			openStackMachine: func() *infrav1.OpenStackMachine {
				return &infrav1.OpenStackMachine{
					Spec: infrav1.OpenStackMachineSpec{},
				}
			},
			openStackCluster: func() *infrav1.OpenStackCluster {
				return &infrav1.OpenStackCluster{
					Spec: infrav1.OpenStackClusterSpec{},
				}
			},
			wantMachineTags: []string{},
		},
		{
			name: "Machine tags only",
			openStackMachine: func() *infrav1.OpenStackMachine {
				return &infrav1.OpenStackMachine{
					Spec: infrav1.OpenStackMachineSpec{
						Tags: []string{"machine-tag1", "machine-tag2"},
					},
				}
			},
			openStackCluster: func() *infrav1.OpenStackCluster {
				return &infrav1.OpenStackCluster{
					Spec: infrav1.OpenStackClusterSpec{},
				}
			},
			wantMachineTags: []string{"machine-tag1", "machine-tag2"},
		},
		{
			name: "Cluster tags only",
			openStackMachine: func() *infrav1.OpenStackMachine {
				return &infrav1.OpenStackMachine{
					Spec: infrav1.OpenStackMachineSpec{},
				}
			},
			openStackCluster: func() *infrav1.OpenStackCluster {
				return &infrav1.OpenStackCluster{
					Spec: infrav1.OpenStackClusterSpec{
						Tags: []string{"cluster-tag1", "cluster-tag2"},
					},
				}
			},
			wantMachineTags: []string{"cluster-tag1", "cluster-tag2"},
		},
		{
			name: "Machine and cluster tags",
			openStackMachine: func() *infrav1.OpenStackMachine {
				return &infrav1.OpenStackMachine{
					Spec: infrav1.OpenStackMachineSpec{
						Tags: []string{"machine-tag1", "machine-tag2"},
					},
				}
			},
			openStackCluster: func() *infrav1.OpenStackCluster {
				return &infrav1.OpenStackCluster{
					Spec: infrav1.OpenStackClusterSpec{
						Tags: []string{"cluster-tag1", "cluster-tag2"},
					},
				}
			},
			wantMachineTags: []string{"machine-tag1", "machine-tag2", "cluster-tag1", "cluster-tag2"},
		},
		{
			name: "Duplicate tags",
			openStackMachine: func() *infrav1.OpenStackMachine {
				return &infrav1.OpenStackMachine{
					Spec: infrav1.OpenStackMachineSpec{
						Tags: []string{"tag1", "tag2", "tag1"},
					},
				}
			},
			openStackCluster: func() *infrav1.OpenStackCluster {
				return &infrav1.OpenStackCluster{
					Spec: infrav1.OpenStackClusterSpec{
						Tags: []string{"tag2", "tag3", "tag3"},
					},
				}
			},
			wantMachineTags: []string{"tag1", "tag2", "tag3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMachineTags := getInstanceTags(tt.openStackMachine(), tt.openStackCluster())
			if !reflect.DeepEqual(gotMachineTags, tt.wantMachineTags) {
				t.Errorf("getInstanceTags() = %v, want %v", gotMachineTags, tt.wantMachineTags)
			}
		})
	}
}

func Test_getManagedSecurityGroups(t *testing.T) {
	tests := []struct {
		name               string
		openStackCluster   func() *infrav1.OpenStackCluster
		machine            func() *clusterv1.Machine
		openStackMachine   func() *infrav1.OpenStackMachine
		wantSecurityGroups []infrav1.SecurityGroupFilter
	}{
		{
			name:               "Defaults",
			openStackCluster:   getDefaultOpenStackCluster,
			machine:            getDefaultMachine,
			openStackMachine:   getDefaultOpenStackMachine,
			wantSecurityGroups: []infrav1.SecurityGroupFilter{},
		},
		{
			name: "Control plane machine with control plane security group",
			openStackCluster: func() *infrav1.OpenStackCluster {
				c := getDefaultOpenStackCluster()
				c.Spec.ManagedSecurityGroups = &infrav1.ManagedSecurityGroups{}
				c.Status.ControlPlaneSecurityGroup = &infrav1.SecurityGroupStatus{ID: controlPlaneSecurityGroupUUID}
				return c
			},
			machine: func() *clusterv1.Machine {
				m := getDefaultMachine()
				m.Labels = map[string]string{
					clusterv1.MachineControlPlaneLabel: "true",
				}
				return m
			},
			openStackMachine: getDefaultOpenStackMachine,
			wantSecurityGroups: []infrav1.SecurityGroupFilter{
				{ID: controlPlaneSecurityGroupUUID},
			},
		},
		{
			name: "Worker machine with worker security group",
			openStackCluster: func() *infrav1.OpenStackCluster {
				c := getDefaultOpenStackCluster()
				c.Spec.ManagedSecurityGroups = &infrav1.ManagedSecurityGroups{}
				c.Status.WorkerSecurityGroup = &infrav1.SecurityGroupStatus{ID: workerSecurityGroupUUID}
				return c
			},
			machine:          getDefaultMachine,
			openStackMachine: getDefaultOpenStackMachine,
			wantSecurityGroups: []infrav1.SecurityGroupFilter{
				{ID: workerSecurityGroupUUID},
			},
		},
		{
			name: "Control plane machine without control plane security group",
			openStackCluster: func() *infrav1.OpenStackCluster {
				c := getDefaultOpenStackCluster()
				c.Spec.ManagedSecurityGroups = &infrav1.ManagedSecurityGroups{}
				c.Status.ControlPlaneSecurityGroup = nil
				return c
			},
			machine: func() *clusterv1.Machine {
				m := getDefaultMachine()
				m.Labels = map[string]string{
					clusterv1.MachineControlPlaneLabel: "true",
				}
				return m
			},
			openStackMachine:   getDefaultOpenStackMachine,
			wantSecurityGroups: []infrav1.SecurityGroupFilter{},
		},
		{
			name: "Worker machine without worker security group",
			openStackCluster: func() *infrav1.OpenStackCluster {
				c := getDefaultOpenStackCluster()
				c.Spec.ManagedSecurityGroups = &infrav1.ManagedSecurityGroups{}
				c.Status.WorkerSecurityGroup = nil
				return c
			},
			machine:            getDefaultMachine,
			openStackMachine:   getDefaultOpenStackMachine,
			wantSecurityGroups: []infrav1.SecurityGroupFilter{},
		},
		{
			name: "Machine with additional security groups",
			openStackCluster: func() *infrav1.OpenStackCluster {
				c := getDefaultOpenStackCluster()
				c.Spec.ManagedSecurityGroups = &infrav1.ManagedSecurityGroups{}
				c.Status.ControlPlaneSecurityGroup = &infrav1.SecurityGroupStatus{ID: controlPlaneSecurityGroupUUID}
				c.Status.WorkerSecurityGroup = &infrav1.SecurityGroupStatus{ID: workerSecurityGroupUUID}
				return c
			},
			machine: getDefaultMachine,
			openStackMachine: func() *infrav1.OpenStackMachine {
				m := getDefaultOpenStackMachine()
				m.Spec.SecurityGroups = []infrav1.SecurityGroupFilter{{ID: extraSecurityGroupUUID}}
				return m
			},
			wantSecurityGroups: []infrav1.SecurityGroupFilter{
				{ID: extraSecurityGroupUUID},
				{ID: workerSecurityGroupUUID},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMachineSecurity := getManagedSecurityGroups(tt.openStackCluster(), tt.machine(), tt.openStackMachine())
			if !reflect.DeepEqual(gotMachineSecurity, tt.wantSecurityGroups) {
				t.Errorf("getManagedSecurityGroups() = %v, want %v", gotMachineSecurity, tt.wantSecurityGroups)
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
