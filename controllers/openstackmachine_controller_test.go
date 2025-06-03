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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	infrav1alpha1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

const (
	networkUUID                   = "d412171b-9fd7-41c1-95a6-c24e5953974d"
	subnetUUID                    = "d2d8d98d-b234-477e-a547-868b7cb5d6a5"
	extraSecurityGroupUUID        = "514bb2d8-3390-4a3b-86a7-7864ba57b329"
	controlPlaneSecurityGroupUUID = "c9817a91-4821-42db-8367-2301002ab659"
	workerSecurityGroupUUID       = "9c6c0d28-03c9-436c-815d-58440ac2c1c8"
	serverGroupUUID               = "7b940d62-68ef-4e42-a76a-1a62e290509c"
	imageUUID                     = "ce96e584-7ebc-46d6-9e55-987d72e3806c"
	flavorUUID                    = "43b1c962-53ba-4690-b210-14e5a7651dbe"

	openStackMachineName = "test-openstack-machine"
	namespace            = "test-namespace"
	flavorName           = "test-flavor"
	sshKeyName           = "test-ssh-key"
	failureDomain        = "test-failure-domain"
)

func TestOpenStackMachineSpecToOpenStackServerSpec(t *testing.T) {
	identityRef := infrav1.OpenStackIdentityReference{
		Name:      "foo",
		CloudName: "my-cloud",
	}
	openStackCluster := &infrav1.OpenStackCluster{
		Spec: infrav1.OpenStackClusterSpec{
			ManagedSecurityGroups: &infrav1.ManagedSecurityGroups{},
		},
		Status: infrav1.OpenStackClusterStatus{
			WorkerSecurityGroup: &infrav1.SecurityGroupStatus{
				ID: workerSecurityGroupUUID,
			},
			Network: &infrav1.NetworkStatusWithSubnets{
				NetworkStatus: infrav1.NetworkStatus{
					ID: networkUUID,
				},
			},
		},
	}
	portOpts := []infrav1.PortOpts{
		{
			Network: &infrav1.NetworkParam{
				ID: ptr.To(openStackCluster.Status.Network.ID),
			},
			SecurityGroups: []infrav1.SecurityGroupParam{
				{
					ID: ptr.To(openStackCluster.Status.WorkerSecurityGroup.ID),
				},
			},
		},
	}
	portOptsWithAdditionalSecurityGroup := []infrav1.PortOpts{
		{
			Network: &infrav1.NetworkParam{
				ID: ptr.To(openStackCluster.Status.Network.ID),
			},
			SecurityGroups: []infrav1.SecurityGroupParam{
				{
					ID: ptr.To(openStackCluster.Status.WorkerSecurityGroup.ID),
				},
				{
					ID: ptr.To(extraSecurityGroupUUID),
				},
			},
		},
	}
	image := infrav1.ImageParam{Filter: &infrav1.ImageFilter{Name: ptr.To("my-image")}}
	tags := []string{"tag1", "tag2"}
	userData := &corev1.LocalObjectReference{Name: "server-data-secret"}
	tests := []struct {
		name string
		spec *infrav1.OpenStackMachineSpec
		want *infrav1alpha1.OpenStackServerSpec
	}{
		{
			name: "Test a minimum OpenStackMachineSpec to OpenStackServerSpec conversion",
			spec: &infrav1.OpenStackMachineSpec{
				Flavor:     ptr.To(flavorName),
				Image:      image,
				SSHKeyName: sshKeyName,
			},
			want: &infrav1alpha1.OpenStackServerSpec{
				Flavor:      ptr.To(flavorName),
				IdentityRef: identityRef,
				Image:       image,
				SSHKeyName:  sshKeyName,
				Ports:       portOpts,
				Tags:        tags,
				UserDataRef: userData,
			},
		},
		{
			name: "Test an OpenStackMachineSpec to OpenStackServerSpec conversion with an additional security group",
			spec: &infrav1.OpenStackMachineSpec{
				Flavor:     ptr.To(flavorName),
				Image:      image,
				SSHKeyName: sshKeyName,
				SecurityGroups: []infrav1.SecurityGroupParam{
					{
						ID: ptr.To(extraSecurityGroupUUID),
					},
				},
			},
			want: &infrav1alpha1.OpenStackServerSpec{
				Flavor:      ptr.To(flavorName),
				IdentityRef: identityRef,
				Image:       image,
				SSHKeyName:  sshKeyName,
				Ports:       portOptsWithAdditionalSecurityGroup,
				Tags:        tags,
				UserDataRef: userData,
			},
		},
		{
			name: "Test an OpenStackMachineSpec to OpenStackServerSpec conversion with flavor and flavorID specified",
			spec: &infrav1.OpenStackMachineSpec{
				Flavor:     ptr.To(flavorName),
				FlavorID:   ptr.To(flavorUUID),
				Image:      image,
				SSHKeyName: sshKeyName,
			},
			want: &infrav1alpha1.OpenStackServerSpec{
				Flavor:      ptr.To(flavorName),
				FlavorID:    ptr.To(flavorUUID),
				IdentityRef: identityRef,
				Image:       image,
				SSHKeyName:  sshKeyName,
				Ports:       portOpts,
				Tags:        tags,
				UserDataRef: userData,
			},
		},
		{
			name: "Test an OpenStackMachineSpec to OpenStackServerSpec conversion with flavorID specified but not flavor",
			spec: &infrav1.OpenStackMachineSpec{
				FlavorID:   ptr.To(flavorUUID),
				Image:      image,
				SSHKeyName: sshKeyName,
			},
			want: &infrav1alpha1.OpenStackServerSpec{
				FlavorID:    ptr.To(flavorUUID),
				IdentityRef: identityRef,
				Image:       image,
				SSHKeyName:  sshKeyName,
				Ports:       portOpts,
				Tags:        tags,
				UserDataRef: userData,
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			spec := openStackMachineSpecToOpenStackServerSpec(tt.spec, identityRef, tags, "", userData, &openStackCluster.Status.WorkerSecurityGroup.ID, openStackCluster.Status.Network.ID)
			if !reflect.DeepEqual(spec, tt.want) {
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
