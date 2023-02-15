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

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha6"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/compute"
)

const (
	networkUUID                   = "d412171b-9fd7-41c1-95a6-c24e5953974d"
	subnetUUID                    = "d2d8d98d-b234-477e-a547-868b7cb5d6a5"
	network2UUID                  = "02f8eed8-8115-44c6-9954-ad7f17daffe0"
	subnet2UUID                   = "95bb6643-8c98-481e-adb6-6efc33334736"
	extraSecurityGroupUUID        = "514bb2d8-3390-4a3b-86a7-7864ba57b329"
	controlPlaneSecurityGroupUUID = "c9817a91-4821-42db-8367-2301002ab659"
	workerSecurityGroupUUID       = "9c6c0d28-03c9-436c-815d-58440ac2c1c8"
	serverGroupUUID               = "7b940d62-68ef-4e42-a76a-1a62e290509c"

	openStackMachineName = "test-openstack-machine"
	namespace            = "test-namespace"
	imageName            = "test-image"
	flavorName           = "test-flavor"
	sshKeyName           = "test-ssh-key"
	novaAZ               = "test-nova-az"
	cinderAZ             = "test-cinder-az"
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
			// This is for completeness: the test shouldn't actually read it
			FailureDomain: pointer.StringPtr(novaAZ),
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
		Status: infrav1.OpenStackMachineStatus{
			FailureDomain: &infrav1.FailureDomain{
				ComputeAvailabilityZone: novaAZ,
			},
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
		ConfigDrive:             *pointer.BoolPtr(true),
		ComputeAvailabilityZone: *pointer.StringPtr(novaAZ),
		ServerGroupID:           serverGroupUUID,
		Tags:                    []string{"test-tag"},
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
		{
			name:             "Empty failure domain",
			openStackCluster: getDefaultOpenStackCluster,
			machine:          getDefaultMachine,
			openStackMachine: func() *infrav1.OpenStackMachine {
				m := getDefaultOpenStackMachine()
				m.Status.FailureDomain = &infrav1.FailureDomain{}
				return m
			},
			wantInstanceSpec: func() *compute.InstanceSpec {
				i := getDefaultInstanceSpec()
				i.ComputeAvailabilityZone = ""
				return i
			},
			wantErr: false,
		},
		{
			name:             "Storage availability should set AZ of root volume",
			openStackCluster: getDefaultOpenStackCluster,
			machine:          getDefaultMachine,
			openStackMachine: func() *infrav1.OpenStackMachine {
				m := getDefaultOpenStackMachine()
				m.Status.FailureDomain = &infrav1.FailureDomain{
					ComputeAvailabilityZone: novaAZ,
					StorageAvailabilityZone: cinderAZ,
				}
				m.Spec.RootVolume = &infrav1.RootVolume{
					Size: 50,
				}
				return m
			},
			wantInstanceSpec: func() *compute.InstanceSpec {
				i := getDefaultInstanceSpec()
				i.RootVolume = &infrav1.RootVolume{
					Size:             50,
					AvailabilityZone: cinderAZ,
				}
				return i
			},
			wantErr: false,
		},
		{
			name:             "Setting storage AZ zone and root volume AZ is an error",
			openStackCluster: getDefaultOpenStackCluster,
			machine:          getDefaultMachine,
			openStackMachine: func() *infrav1.OpenStackMachine {
				m := getDefaultOpenStackMachine()
				m.Status.FailureDomain = &infrav1.FailureDomain{
					ComputeAvailabilityZone: novaAZ,
					StorageAvailabilityZone: cinderAZ,
				}
				m.Spec.RootVolume = &infrav1.RootVolume{
					Size:             50,
					AvailabilityZone: cinderAZ,
				}
				return m
			},
			wantErr: true,
		},
		{
			name:             "Setting failure domain ports only sets only failure domain ports",
			openStackCluster: getDefaultOpenStackCluster,
			machine:          getDefaultMachine,
			openStackMachine: func() *infrav1.OpenStackMachine {
				m := getDefaultOpenStackMachine()
				m.Status.FailureDomain = &infrav1.FailureDomain{
					ComputeAvailabilityZone: novaAZ,
					Ports: []infrav1.PortOpts{
						{
							Description: "fd-port-0",
						},
						{
							Description: "fd-port-1",
						},
					},
				}
				return m
			},
			wantInstanceSpec: func() *compute.InstanceSpec {
				i := getDefaultInstanceSpec()
				i.Ports = []infrav1.PortOpts{
					{
						Description: "fd-port-0",
					},
					{
						Description: "fd-port-1",
					},
				}
				return i
			},
			wantErr: false,
		},
		{
			name:             "Setting machine ports only sets only machine ports",
			openStackCluster: getDefaultOpenStackCluster,
			machine:          getDefaultMachine,
			openStackMachine: func() *infrav1.OpenStackMachine {
				m := getDefaultOpenStackMachine()
				m.Spec.Ports = []infrav1.PortOpts{
					{
						Description: "machine-port-0",
					},
					{
						Description: "machine-port-1",
					},
				}
				return m
			},
			wantInstanceSpec: func() *compute.InstanceSpec {
				i := getDefaultInstanceSpec()
				i.Ports = []infrav1.PortOpts{
					{
						Description: "machine-port-0",
					},
					{
						Description: "machine-port-1",
					},
				}
				return i
			},
			wantErr: false,
		},
		{
			name:             "Setting both failure domain and machine ports prepends failure domain ports to machine ports",
			openStackCluster: getDefaultOpenStackCluster,
			machine:          getDefaultMachine,
			openStackMachine: func() *infrav1.OpenStackMachine {
				m := getDefaultOpenStackMachine()
				m.Status.FailureDomain = &infrav1.FailureDomain{
					ComputeAvailabilityZone: novaAZ,
					Ports: []infrav1.PortOpts{
						{
							Description: "fd-port-0",
						},
						{
							Description: "fd-port-1",
						},
					},
				}
				m.Spec.Ports = []infrav1.PortOpts{
					{
						Description: "machine-port-0",
					},
					{
						Description: "machine-port-1",
					},
				}
				return m
			},
			wantInstanceSpec: func() *compute.InstanceSpec {
				i := getDefaultInstanceSpec()
				i.Ports = []infrav1.PortOpts{
					{
						Description: "fd-port-0",
					},
					{
						Description: "fd-port-1",
					},
					{
						Description: "machine-port-0",
					},
					{
						Description: "machine-port-1",
					},
				}
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
				Expect(got).To(BeNil())
			} else {
				Expect(got).To(Equal(tt.wantInstanceSpec()))
				Expect(err).ToNot(HaveOccurred())
			}
		})
	}
}

func Test_getFailureDomainForMachine(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name                string
		osMachineProviderID *string
		failureDomainName   *string
		clusterSpecFDs      []infrav1.FailureDomainDefinition
		clusterStatusFDs    clusterv1.FailureDomains
		want                *infrav1.FailureDomain
		wantErr             bool
	}{
		{
			name:              "Nil failure domain name should set empty failure domain",
			failureDomainName: nil,
			want:              &infrav1.FailureDomain{},
			wantErr:           false,
		},
		{
			name:              "Empty failure domain name should set empty failure domain",
			failureDomainName: pointer.StringPtr(""),
			want:              &infrav1.FailureDomain{},
			wantErr:           false,
		},
		{
			name:              "Nova AZ failure domain should set failure domain with ComputeAvailabilityZone",
			failureDomainName: pointer.StringPtr("nova-az-0"),
			clusterStatusFDs: map[string]clusterv1.FailureDomainSpec{
				"nova-az-0": {
					ControlPlane: true,
					Attributes: map[string]string{
						infrav1.FailureDomainType: infrav1.FailureDomainTypeAZ,
					},
				},
			},
			want: &infrav1.FailureDomain{
				ComputeAvailabilityZone: "nova-az-0",
			},
			wantErr: false,
		},
		{
			name:                "Machine with providerID should use failure doman name as AZ even when it's undefined in cluster status",
			osMachineProviderID: pointer.StringPtr("openstack:///7d9b2a67-8e48-448e-804a-572e2a751491"),
			failureDomainName:   pointer.StringPtr("nova-az-0"),
			want: &infrav1.FailureDomain{
				ComputeAvailabilityZone: "nova-az-0",
			},
			wantErr: false,
		},
		{
			name:              "Cluster failure domain should be copied to machine",
			failureDomainName: pointer.StringPtr("cluster-fd-0"),
			clusterSpecFDs: []infrav1.FailureDomainDefinition{
				{
					Name:             "cluster-fd-0",
					MachinePlacement: infrav1.FailureDomainMachinePlacementAll,
					FailureDomain: infrav1.FailureDomain{
						ComputeAvailabilityZone: "nova-az-0",
						StorageAvailabilityZone: "cinder-az-1",
						Ports: []infrav1.PortOpts{
							{
								Network: &infrav1.NetworkFilter{
									ID: "31b211b5-269d-44b7-b199-0a689de44346",
								},
								Description: "K8s control plane port",
								FixedIPs: []infrav1.FixedIP{
									{
										Subnet: &infrav1.SubnetFilter{
											ID: "1d39f59b-b164-458b-8201-5cd550443d42",
										},
									},
								},
								Tags: []string{
									"control-plane",
								},
							},
							{
								Network: &infrav1.NetworkFilter{
									ID: "c0dba1c8-9510-4d43-b7aa-a67edf5a60d3",
								},
								Description: "Storage network",
								FixedIPs: []infrav1.FixedIP{
									{
										Subnet: &infrav1.SubnetFilter{
											ID: "9f26add6-20b3-46e8-9c47-b2c6b7590502",
										},
									},
								},
								Tags: []string{
									"storage",
								},
							},
						},
					},
				},
			},
			clusterStatusFDs: map[string]clusterv1.FailureDomainSpec{
				"cluster-fd-0": {
					ControlPlane: true,
					Attributes: map[string]string{
						infrav1.FailureDomainType: infrav1.FailureDomainTypeCluster,
					},
				},
			},
			want: &infrav1.FailureDomain{
				ComputeAvailabilityZone: "nova-az-0",
				StorageAvailabilityZone: "cinder-az-1",
				Ports: []infrav1.PortOpts{
					{
						Network: &infrav1.NetworkFilter{
							ID: "31b211b5-269d-44b7-b199-0a689de44346",
						},
						Description: "K8s control plane port",
						FixedIPs: []infrav1.FixedIP{
							{
								Subnet: &infrav1.SubnetFilter{
									ID: "1d39f59b-b164-458b-8201-5cd550443d42",
								},
							},
						},
						Tags: []string{
							"control-plane",
						},
					},
					{
						Network: &infrav1.NetworkFilter{
							ID: "c0dba1c8-9510-4d43-b7aa-a67edf5a60d3",
						},
						Description: "Storage network",
						FixedIPs: []infrav1.FixedIP{
							{
								Subnet: &infrav1.SubnetFilter{
									ID: "9f26add6-20b3-46e8-9c47-b2c6b7590502",
								},
							},
						},
						Tags: []string{
							"storage",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:              "Return an error if the failure domain is not defined in the cluster spec",
			failureDomainName: pointer.StringPtr("nova-az-1"),
			clusterStatusFDs: map[string]clusterv1.FailureDomainSpec{
				"nova-az-0": {
					ControlPlane: true,
					Attributes: map[string]string{
						infrav1.FailureDomainType: infrav1.FailureDomainTypeAZ,
					},
				},
			},
			wantErr: true,
		},
		{
			name:              "Return an error if the failure domain has an unknown type",
			failureDomainName: pointer.StringPtr("nova-az-0"),
			clusterStatusFDs: map[string]clusterv1.FailureDomainSpec{
				"nova-az-0": {
					ControlPlane: true,
					Attributes: map[string]string{
						infrav1.FailureDomainType: "InvalidType",
					},
				},
			},
			wantErr: true,
		},
		{
			name:              "Return an error if the cluster failure domain is not defined in the cluster spec",
			failureDomainName: pointer.StringPtr("cluster-fd-0"),
			clusterSpecFDs:    []infrav1.FailureDomainDefinition{},
			clusterStatusFDs: map[string]clusterv1.FailureDomainSpec{
				"cluster-fd-0": {
					ControlPlane: true,
					Attributes: map[string]string{
						infrav1.FailureDomainType: infrav1.FailureDomainTypeCluster,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			openStackCluster := &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					FailureDomains: tt.clusterSpecFDs,
				},
				Status: infrav1.OpenStackClusterStatus{
					FailureDomains: tt.clusterStatusFDs,
				},
			}
			machine := &clusterv1.Machine{
				Spec: clusterv1.MachineSpec{
					FailureDomain: tt.failureDomainName,
				},
			}
			openStackMachine := &infrav1.OpenStackMachine{
				Spec: infrav1.OpenStackMachineSpec{
					ProviderID: tt.osMachineProviderID,
				},
			}
			got, err := getFailureDomainForMachine(machine, openStackMachine, openStackCluster)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
				g.Expect(got).To(BeNil())
				return
			}

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(got).To(Equal(tt.want))
		})
	}
}
