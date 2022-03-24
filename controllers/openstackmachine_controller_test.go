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
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha5"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/compute"
)

const (
	networkUUID                   = "d412171b-9fd7-41c1-95a6-c24e5953974d"
	subnetUUID                    = "d2d8d98d-b234-477e-a547-868b7cb5d6a5"
	extraSecurityGroupUUID        = "514bb2d8-3390-4a3b-86a7-7864ba57b329"
	controlPlaneSecurityGroupUUID = "c9817a91-4821-42db-8367-2301002ab659"
	workerSecurityGroupUUID       = "9c6c0d28-03c9-436c-815d-58440ac2c1c8"
	serverGroupUUID               = "7b940d62-68ef-4e42-a76a-1a62e290509c"

	openStackMachineName = "test-openstack-machine"
	namespace            = "test-namespace"
	imageName            = "test-image"
	flavorName           = "test-flavor"
	sshKeyName           = "test-ssh-key"
	failureDomain        = "test-failure-domain"
)

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
			FailureDomain: pointer.StringPtr(failureDomain),
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

func getDefaultInstanceSpec() *compute.InstanceSpec {
	return &compute.InstanceSpec{
		Name:       openStackMachineName,
		Image:      imageName,
		Flavor:     flavorName,
		SSHKeyName: sshKeyName,
		UserData:   "user-data",
		Metadata: map[string]string{
			"test-metadata": "test-value",
		},
		ConfigDrive:   *pointer.BoolPtr(true),
		FailureDomain: *pointer.StringPtr(failureDomain),
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
		wantErr          bool
	}{
		{
			name:             "Defaults",
			openStackCluster: getDefaultOpenStackCluster,
			machine:          getDefaultMachine,
			openStackMachine: getDefaultOpenStackMachine,
			wantInstanceSpec: getDefaultInstanceSpec,
			wantErr:          false,
		},
		{
			name: "Control plane security group",
			openStackCluster: func() *infrav1.OpenStackCluster {
				c := getDefaultOpenStackCluster()
				c.Spec.ManagedSecurityGroups = true
				return c
			},
			machine: func() *clusterv1.Machine {
				m := getDefaultMachine()
				m.Labels = map[string]string{
					clusterv1.MachineControlPlaneLabelName: "true",
				}
				return m
			},
			openStackMachine: getDefaultOpenStackMachine,
			wantInstanceSpec: func() *compute.InstanceSpec {
				i := getDefaultInstanceSpec()
				i.SecurityGroups = []infrav1.SecurityGroupParam{{UUID: controlPlaneSecurityGroupUUID}}
				return i
			},
			wantErr: false,
		},
		{
			name: "Worker security group",
			openStackCluster: func() *infrav1.OpenStackCluster {
				c := getDefaultOpenStackCluster()
				c.Spec.ManagedSecurityGroups = true
				return c
			},
			machine:          getDefaultMachine,
			openStackMachine: getDefaultOpenStackMachine,
			wantInstanceSpec: func() *compute.InstanceSpec {
				i := getDefaultInstanceSpec()
				i.SecurityGroups = []infrav1.SecurityGroupParam{{UUID: workerSecurityGroupUUID}}
				return i
			},
			wantErr: false,
		},
		{
			name: "Extra security group",
			openStackCluster: func() *infrav1.OpenStackCluster {
				c := getDefaultOpenStackCluster()
				c.Spec.ManagedSecurityGroups = true
				return c
			},
			machine: getDefaultMachine,
			openStackMachine: func() *infrav1.OpenStackMachine {
				m := getDefaultOpenStackMachine()
				m.Spec.SecurityGroups = []infrav1.SecurityGroupParam{{UUID: extraSecurityGroupUUID}}
				return m
			},
			wantInstanceSpec: func() *compute.InstanceSpec {
				i := getDefaultInstanceSpec()
				i.SecurityGroups = []infrav1.SecurityGroupParam{
					{UUID: extraSecurityGroupUUID},
					{UUID: workerSecurityGroupUUID},
				}
				return i
			},
			wantErr: false,
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
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := machineToInstanceSpec(tt.openStackCluster(), tt.machine(), tt.openStackMachine(), "user-data")
			if tt.wantErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}

			Expect(got).To(Equal(tt.wantInstanceSpec()))
		})
	}
}
