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
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/test/framework"
	v1beta1conditions "sigs.k8s.io/cluster-api/util/deprecated/v1beta1/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1alpha1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
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
	testInstanceID       = "test-instance-id-12345"
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
	openStackClusterWithSubnet := &infrav1.OpenStackCluster{
		Spec: infrav1.OpenStackClusterSpec{
			ManagedSecurityGroups: &infrav1.ManagedSecurityGroups{},
			Subnets: []infrav1.SubnetParam{
				{
					ID: ptr.To(subnetUUID),
				},
			},
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
	openStackClusterNoNetwork := &infrav1.OpenStackCluster{
		Spec: infrav1.OpenStackClusterSpec{
			ManagedSecurityGroups: &infrav1.ManagedSecurityGroups{},
			Subnets: []infrav1.SubnetParam{
				{
					ID: ptr.To(subnetUUID),
				},
			},
		},
		Status: infrav1.OpenStackClusterStatus{
			WorkerSecurityGroup: &infrav1.SecurityGroupStatus{
				ID: workerSecurityGroupUUID,
			},
		},
	}
	openStackClusterNetworkWithoutID := &infrav1.OpenStackCluster{
		Spec: infrav1.OpenStackClusterSpec{
			ManagedSecurityGroups: &infrav1.ManagedSecurityGroups{},
			Subnets: []infrav1.SubnetParam{
				{
					ID: ptr.To(subnetUUID),
				},
			},
		},
		Status: infrav1.OpenStackClusterStatus{
			WorkerSecurityGroup: &infrav1.SecurityGroupStatus{
				ID: workerSecurityGroupUUID,
			},
			Network: &infrav1.NetworkStatusWithSubnets{
				NetworkStatus: infrav1.NetworkStatus{
					ID: "",
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
	portOptsWithAdditionalSubnet := []infrav1.PortOpts{
		{
			Network: &infrav1.NetworkParam{
				ID: ptr.To(openStackCluster.Status.Network.ID),
			},
			SecurityGroups: []infrav1.SecurityGroupParam{
				{
					ID: ptr.To(openStackCluster.Status.WorkerSecurityGroup.ID),
				},
			},
			FixedIPs: []infrav1.FixedIP{
				{
					Subnet: &infrav1.SubnetParam{
						ID: ptr.To(subnetUUID),
					},
				},
			},
		},
	}
	image := infrav1.ImageParam{Filter: &infrav1.ImageFilter{Name: ptr.To("my-image")}}
	tags := []string{"tag1", "tag2"}
	userData := &corev1.LocalObjectReference{Name: "server-data-secret"}
	tests := []struct {
		name    string
		cluster *infrav1.OpenStackCluster
		spec    *infrav1.OpenStackMachineSpec
		want    *infrav1alpha1.OpenStackServerSpec
		wantErr bool
	}{
		{
			name:    "Test a minimum OpenStackMachineSpec to OpenStackServerSpec conversion",
			cluster: openStackCluster,
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
			name:    "Test an OpenStackMachineSpec to OpenStackServerSpec conversion with an additional security group",
			cluster: openStackCluster,
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
			name:    "Test a OpenStackMachineSpec to OpenStackServerSpec conversion with a specified subnet",
			cluster: openStackClusterWithSubnet,
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
				Ports:       portOptsWithAdditionalSubnet,
				Tags:        tags,
				UserDataRef: userData,
			},
		},
		{
			name:    "Test an OpenStackMachineSpec to OpenStackServerSpec conversion with flavor and flavorID specified",
			cluster: openStackCluster,
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
			name:    "Test an OpenStackMachineSpec to OpenStackServerSpec conversion with flavorID specified but not flavor",
			cluster: openStackCluster,
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
		{
			name: "Cluster network nil, machine defines port network and overrides SG",
			spec: &infrav1.OpenStackMachineSpec{
				Ports: []infrav1.PortOpts{{
					Network: &infrav1.NetworkParam{ID: ptr.To(networkUUID)},
				}},
				SecurityGroups: []infrav1.SecurityGroupParam{{ID: ptr.To(extraSecurityGroupUUID)}},
			},
			cluster: openStackClusterNoNetwork,
			want: &infrav1alpha1.OpenStackServerSpec{
				IdentityRef: identityRef,
				Ports: []infrav1.PortOpts{{
					Network: &infrav1.NetworkParam{ID: ptr.To(networkUUID)},
					SecurityGroups: []infrav1.SecurityGroupParam{
						{ID: ptr.To(workerSecurityGroupUUID)},
						{ID: ptr.To(extraSecurityGroupUUID)},
					},
				}},
				Tags:        tags,
				UserDataRef: userData,
			},
		},
		{
			name: "Cluster network nil, machine defines port network and falls back to cluster SG",
			spec: &infrav1.OpenStackMachineSpec{
				Ports: []infrav1.PortOpts{{
					Network: &infrav1.NetworkParam{ID: ptr.To(networkUUID)},
				}},
			},
			cluster: openStackClusterNoNetwork,
			want: &infrav1alpha1.OpenStackServerSpec{
				IdentityRef: identityRef,
				Ports: []infrav1.PortOpts{{
					Network:        &infrav1.NetworkParam{ID: ptr.To(networkUUID)},
					SecurityGroups: []infrav1.SecurityGroupParam{{ID: ptr.To(workerSecurityGroupUUID)}},
				}},
				Tags:        tags,
				UserDataRef: userData,
			},
		},
		{
			name: "Error case: no cluster network and no machine ports",
			spec: &infrav1.OpenStackMachineSpec{
				Flavor:     ptr.To(flavorName),
				Image:      image,
				SSHKeyName: sshKeyName,
				// No ports defined
			},
			cluster: openStackClusterNoNetwork,
			want:    nil,
			wantErr: true,
		},
		{
			name: "Empty cluster network ID, machine defines explicit ports",
			spec: &infrav1.OpenStackMachineSpec{
				Flavor: ptr.To(flavorName),
				Image:  image,
				Ports: []infrav1.PortOpts{{
					Network: &infrav1.NetworkParam{ID: ptr.To(networkUUID)},
				}},
			},
			cluster: openStackClusterNetworkWithoutID,
			want: &infrav1alpha1.OpenStackServerSpec{
				Flavor:      ptr.To(flavorName),
				IdentityRef: identityRef,
				Image:       image,
				Ports: []infrav1.PortOpts{{
					Network:        &infrav1.NetworkParam{ID: ptr.To(networkUUID)},
					SecurityGroups: []infrav1.SecurityGroupParam{{ID: ptr.To(workerSecurityGroupUUID)}},
				}},
				Tags:        tags,
				UserDataRef: userData,
			},
		},
		{
			name: "Explicit port with disablePortSecurity",
			spec: &infrav1.OpenStackMachineSpec{
				Flavor: ptr.To(flavorName),
				Image:  image,
				Ports: []infrav1.PortOpts{{
					Network: &infrav1.NetworkParam{ID: ptr.To(networkUUID)},
					ResolvedPortSpecFields: infrav1.ResolvedPortSpecFields{
						DisablePortSecurity: ptr.To(true),
					},
				}},
			},
			cluster: openStackClusterNetworkWithoutID,
			want: &infrav1alpha1.OpenStackServerSpec{
				Flavor:      ptr.To(flavorName),
				IdentityRef: identityRef,
				Image:       image,
				Ports: []infrav1.PortOpts{{
					Network:        &infrav1.NetworkParam{ID: ptr.To(networkUUID)},
					SecurityGroups: nil,
					ResolvedPortSpecFields: infrav1.ResolvedPortSpecFields{
						DisablePortSecurity: ptr.To(true),
					},
				}},
				Tags:        tags,
				UserDataRef: userData,
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			spec, err := openStackMachineSpecToOpenStackServerSpec(tt.spec, identityRef, tags, "", userData, &openStackCluster.Status.WorkerSecurityGroup.ID, tt.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("openStackMachineSpecToOpenStackServerSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(spec, tt.want) {
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

func TestReconcileMachineState(t *testing.T) {
	tests := []struct {
		name                            string
		instanceState                   infrav1.InstanceState
		machineHasNodeRef               bool
		expectRequeue                   bool
		expectedInstanceReadyCondition  *clusterv1beta1.Condition
		expectedReadyCondition          *clusterv1beta1.Condition
		expectInitializationProvisioned bool
		expectFailureSet                bool
	}{
		{
			name:          "Instance state ACTIVE sets conditions to True and initialization.provisioned",
			instanceState: infrav1.InstanceStateActive,
			expectRequeue: false,
			expectedInstanceReadyCondition: &clusterv1beta1.Condition{
				Type:   infrav1.InstanceReadyCondition,
				Status: corev1.ConditionTrue,
			},
			expectedReadyCondition: &clusterv1beta1.Condition{
				Type:   clusterv1beta1.ReadyCondition,
				Status: corev1.ConditionTrue,
			},
			expectInitializationProvisioned: true,
		},
		{
			name:              "Instance state ERROR sets conditions to False without NodeRef",
			instanceState:     infrav1.InstanceStateError,
			machineHasNodeRef: false,
			expectRequeue:     true,
			expectedInstanceReadyCondition: &clusterv1beta1.Condition{
				Type:     infrav1.InstanceReadyCondition,
				Status:   corev1.ConditionFalse,
				Severity: clusterv1beta1.ConditionSeverityError,
				Reason:   infrav1.InstanceStateErrorReason,
			},
			expectedReadyCondition: &clusterv1beta1.Condition{
				Type:     clusterv1beta1.ReadyCondition,
				Status:   corev1.ConditionFalse,
				Severity: clusterv1beta1.ConditionSeverityError,
				Reason:   infrav1.InstanceStateErrorReason,
			},
			expectFailureSet: true,
		},
		{
			name:              "Instance state ERROR with NodeRef does not set failure",
			instanceState:     infrav1.InstanceStateError,
			machineHasNodeRef: true,
			expectRequeue:     true,
			expectedInstanceReadyCondition: &clusterv1beta1.Condition{
				Type:     infrav1.InstanceReadyCondition,
				Status:   corev1.ConditionFalse,
				Severity: clusterv1beta1.ConditionSeverityError,
				Reason:   infrav1.InstanceStateErrorReason,
			},
			expectedReadyCondition: &clusterv1beta1.Condition{
				Type:     clusterv1beta1.ReadyCondition,
				Status:   corev1.ConditionFalse,
				Severity: clusterv1beta1.ConditionSeverityError,
				Reason:   infrav1.InstanceStateErrorReason,
			},
			expectFailureSet: false,
		},
		{
			name:          "Instance state DELETED sets conditions to False",
			instanceState: infrav1.InstanceStateDeleted,
			expectRequeue: true,
			expectedInstanceReadyCondition: &clusterv1beta1.Condition{
				Type:     infrav1.InstanceReadyCondition,
				Status:   corev1.ConditionFalse,
				Severity: clusterv1beta1.ConditionSeverityError,
				Reason:   infrav1.InstanceDeletedReason,
			},
			expectedReadyCondition: &clusterv1beta1.Condition{
				Type:     clusterv1beta1.ReadyCondition,
				Status:   corev1.ConditionFalse,
				Severity: clusterv1beta1.ConditionSeverityError,
				Reason:   infrav1.InstanceDeletedReason,
			},
		},
		{
			name:          "Instance state BUILD sets ReadyCondition to False",
			instanceState: infrav1.InstanceStateBuild,
			expectRequeue: true,
			expectedReadyCondition: &clusterv1beta1.Condition{
				Type:     clusterv1beta1.ReadyCondition,
				Status:   corev1.ConditionFalse,
				Severity: clusterv1beta1.ConditionSeverityInfo,
				Reason:   infrav1.InstanceNotReadyReason,
			},
		},
		{
			name:          "Instance state SHUTOFF sets conditions to Unknown",
			instanceState: infrav1.InstanceStateShutoff,
			expectRequeue: true,
			expectedInstanceReadyCondition: &clusterv1beta1.Condition{
				Type:   infrav1.InstanceReadyCondition,
				Status: corev1.ConditionUnknown,
				Reason: infrav1.InstanceNotReadyReason,
			},
			expectedReadyCondition: &clusterv1beta1.Condition{
				Type:   clusterv1beta1.ReadyCondition,
				Status: corev1.ConditionUnknown,
				Reason: infrav1.InstanceNotReadyReason,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			openStackMachine := &infrav1.OpenStackMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      openStackMachineName,
					Namespace: namespace,
				},
				Spec: infrav1.OpenStackMachineSpec{
					Flavor: ptr.To(flavorName),
					Image: infrav1.ImageParam{
						Filter: &infrav1.ImageFilter{
							Name: ptr.To("test-image"),
						},
					},
				},
			}

			machine := &clusterv1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-machine",
					Namespace: namespace,
				},
			}
			if tt.machineHasNodeRef {
				machine.Status.NodeRef = clusterv1.MachineNodeReference{
					Name: "test-node",
				}
			}

			openStackServer := &infrav1alpha1.OpenStackServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      openStackMachineName,
					Namespace: namespace,
				},
				Status: infrav1alpha1.OpenStackServerStatus{
					InstanceID:    ptr.To(testInstanceID),
					InstanceState: ptr.To(tt.instanceState),
				},
			}

			r := &OpenStackMachineReconciler{}
			result := r.reconcileMachineState(scope.NewWithLogger(nil, logr.Discard()), openStackMachine, machine, openStackServer)

			// Check requeue
			if tt.expectRequeue && result == nil {
				t.Errorf("expected requeue result, got nil")
			}
			if !tt.expectRequeue && result != nil {
				t.Errorf("expected no requeue, got %v", result)
			}

			// Check InstanceReadyCondition
			if tt.expectedInstanceReadyCondition != nil {
				condition := v1beta1conditions.Get(openStackMachine, tt.expectedInstanceReadyCondition.Type)
				if condition == nil {
					t.Errorf("expected %s condition to be set", tt.expectedInstanceReadyCondition.Type)
				} else {
					if condition.Status != tt.expectedInstanceReadyCondition.Status {
						t.Errorf("expected %s status %s, got %s", tt.expectedInstanceReadyCondition.Type, tt.expectedInstanceReadyCondition.Status, condition.Status)
					}
					if tt.expectedInstanceReadyCondition.Reason != "" && condition.Reason != tt.expectedInstanceReadyCondition.Reason {
						t.Errorf("expected %s reason %s, got %s", tt.expectedInstanceReadyCondition.Type, tt.expectedInstanceReadyCondition.Reason, condition.Reason)
					}
					if tt.expectedInstanceReadyCondition.Severity != "" && condition.Severity != tt.expectedInstanceReadyCondition.Severity {
						t.Errorf("expected %s severity %s, got %s", tt.expectedInstanceReadyCondition.Type, tt.expectedInstanceReadyCondition.Severity, condition.Severity)
					}
				}
			}

			// Check ReadyCondition
			if tt.expectedReadyCondition != nil {
				condition := v1beta1conditions.Get(openStackMachine, tt.expectedReadyCondition.Type)
				if condition == nil {
					t.Errorf("expected %s condition to be set", tt.expectedReadyCondition.Type)
				} else {
					if condition.Status != tt.expectedReadyCondition.Status {
						t.Errorf("expected %s status %s, got %s", tt.expectedReadyCondition.Type, tt.expectedReadyCondition.Status, condition.Status)
					}
					if tt.expectedReadyCondition.Reason != "" && condition.Reason != tt.expectedReadyCondition.Reason {
						t.Errorf("expected %s reason %s, got %s", tt.expectedReadyCondition.Type, tt.expectedReadyCondition.Reason, condition.Reason)
					}
					if tt.expectedReadyCondition.Severity != "" && condition.Severity != tt.expectedReadyCondition.Severity {
						t.Errorf("expected %s severity %s, got %s", tt.expectedReadyCondition.Type, tt.expectedReadyCondition.Severity, condition.Severity)
					}
				}
			}

			// Check initialization.provisioned
			if tt.expectInitializationProvisioned {
				if openStackMachine.Status.Initialization == nil || !openStackMachine.Status.Initialization.Provisioned {
					t.Errorf("expected Initialization.Provisioned to be true")
				}
			}

			// Check failure is set
			if tt.expectFailureSet {
				if openStackMachine.Status.FailureReason == nil || openStackMachine.Status.FailureMessage == nil {
					t.Errorf("expected FailureReason and FailureMessage to be set")
				}
			} else {
				if openStackMachine.Status.FailureReason != nil || openStackMachine.Status.FailureMessage != nil {
					t.Errorf("expected FailureReason and FailureMessage to not be set")
				}
			}
		})
	}
}

var _ = Describe("OpenStackMachine controller", func() {
	var (
		testMachine        *infrav1.OpenStackMachine
		capiMachine        *clusterv1.Machine
		capiCluster        *clusterv1.Cluster
		testCluster        *infrav1.OpenStackCluster
		testNamespace      string
		machineReconciler  *OpenStackMachineReconciler
		machineMockCtrl    *gomock.Controller
		machineMockFactory *scope.MockScopeFactory
		testNum            int
	)

	capiClusterName := "capi-cluster"
	testClusterName := "test-cluster"
	testMachineName := "test-machine"
	capiMachineName := "capi-machine"

	BeforeEach(func() {
		ctx = context.TODO()
		testNum++
		testNamespace = fmt.Sprintf("machine-test-%d", testNum)

		testCluster = &infrav1.OpenStackCluster{
			TypeMeta: metav1.TypeMeta{
				APIVersion: infrav1.SchemeGroupVersion.Group + "/" + infrav1.SchemeGroupVersion.Version,
				Kind:       "OpenStackCluster",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testClusterName,
				Namespace: testNamespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: clusterv1.GroupVersion.Group + "/" + clusterv1.GroupVersion.Version,
						Kind:       "Cluster",
						Name:       capiClusterName,
						UID:        types.UID("cluster-uid"),
					},
				},
			},
			Spec: infrav1.OpenStackClusterSpec{
				IdentityRef: infrav1.OpenStackIdentityReference{
					Name:      "test-creds",
					CloudName: "openstack",
				},
			},
			Status: infrav1.OpenStackClusterStatus{
				Ready: true,
			},
		}

		capiCluster = &clusterv1.Cluster{
			TypeMeta: metav1.TypeMeta{
				APIVersion: clusterv1.GroupVersion.Group + "/" + clusterv1.GroupVersion.Version,
				Kind:       "Cluster",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      capiClusterName,
				Namespace: testNamespace,
			},
			Spec: clusterv1.ClusterSpec{
				InfrastructureRef: clusterv1.ContractVersionedObjectReference{
					APIGroup: infrav1.GroupName,
					Kind:     "OpenStackCluster",
					Name:     testClusterName,
				},
			},
		}

		capiMachine = &clusterv1.Machine{
			TypeMeta: metav1.TypeMeta{
				APIVersion: clusterv1.GroupVersion.Group + "/" + clusterv1.GroupVersion.Version,
				Kind:       "Machine",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      capiMachineName,
				Namespace: testNamespace,
				Labels: map[string]string{
					clusterv1.ClusterNameLabel: capiClusterName,
				},
			},
		}

		testMachine = &infrav1.OpenStackMachine{
			TypeMeta: metav1.TypeMeta{
				APIVersion: infrav1.SchemeGroupVersion.Group + "/" + infrav1.SchemeGroupVersion.Version,
				Kind:       "OpenStackMachine",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testMachineName,
				Namespace: testNamespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: clusterv1.GroupVersion.Group + "/" + clusterv1.GroupVersion.Version,
						Kind:       "Machine",
						Name:       capiMachineName,
						UID:        types.UID("machine-uid"),
					},
				},
			},
			Spec: infrav1.OpenStackMachineSpec{
				Flavor: ptr.To(flavorName),
				Image: infrav1.ImageParam{
					Filter: &infrav1.ImageFilter{
						Name: ptr.To("test-image"),
					},
				},
			},
		}

		input := framework.CreateNamespaceInput{
			Creator: k8sClient,
			Name:    testNamespace,
		}
		framework.CreateNamespace(ctx, input)

		machineMockCtrl = gomock.NewController(GinkgoT())
		machineMockFactory = scope.NewMockScopeFactory(machineMockCtrl, "")
		machineReconciler = &OpenStackMachineReconciler{
			Client:       k8sClient,
			ScopeFactory: machineMockFactory,
		}
	})

	AfterEach(func() {
		orphan := metav1.DeletePropagationOrphan
		deleteOptions := client.DeleteOptions{
			PropagationPolicy: &orphan,
		}

		// Remove finalizers and delete openstackmachine
		patchHelper, err := patch.NewHelper(testMachine, k8sClient)
		Expect(err).To(BeNil())
		testMachine.SetFinalizers([]string{})
		err = patchHelper.Patch(ctx, testMachine)
		Expect(err).To(BeNil())
		err = k8sClient.Delete(ctx, testMachine, &deleteOptions)
		Expect(err).To(BeNil())

		// Remove finalizers and delete openstackcluster
		patchHelper, err = patch.NewHelper(testCluster, k8sClient)
		Expect(err).To(BeNil())
		testCluster.SetFinalizers([]string{})
		err = patchHelper.Patch(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Delete(ctx, testCluster, &deleteOptions)
		Expect(err).To(BeNil())

		// Remove finalizers and delete cluster
		patchHelper, err = patch.NewHelper(capiCluster, k8sClient)
		Expect(err).To(BeNil())
		capiCluster.SetFinalizers([]string{})
		err = patchHelper.Patch(ctx, capiCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Delete(ctx, capiCluster, &deleteOptions)
		Expect(err).To(BeNil())

		// Remove finalizers and delete machine
		patchHelper, err = patch.NewHelper(capiMachine, k8sClient)
		Expect(err).To(BeNil())
		capiMachine.SetFinalizers([]string{})
		err = patchHelper.Patch(ctx, capiMachine)
		Expect(err).To(BeNil())
		err = k8sClient.Delete(ctx, capiMachine, &deleteOptions)
		Expect(err).To(BeNil())
	})

	It("should set OpenStackAuthenticationSucceededCondition to False when credentials secret is missing", func() {
		testMachine.SetName("missing-machine-credentials")
		testMachine.Spec.IdentityRef = &infrav1.OpenStackIdentityReference{
			Type:      "Secret",
			Name:      "non-existent-secret",
			CloudName: "openstack",
		}

		err := k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiMachine)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, testMachine)
		Expect(err).To(BeNil())

		credentialsErr := fmt.Errorf("secret not found: non-existent-secret")
		machineMockFactory.SetClientScopeCreateError(credentialsErr)

		req := reconcile.Request{
			NamespacedName: client.ObjectKey{
				Name:      testMachine.Name,
				Namespace: testMachine.Namespace,
			},
		}
		result, err := machineReconciler.Reconcile(ctx, req)

		Expect(err).To(MatchError(credentialsErr))
		Expect(result).To(Equal(reconcile.Result{}))

		// Fetch the updated OpenStackMachine to verify the condition was set
		updatedMachine := &infrav1.OpenStackMachine{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: testMachine.Name, Namespace: testMachine.Namespace}, updatedMachine)).To(Succeed())

		// Verify OpenStackAuthenticationSucceededCondition is set to False
		Expect(v1beta1conditions.IsFalse(updatedMachine, infrav1.OpenStackAuthenticationSucceeded)).To(BeTrue())
		condition := v1beta1conditions.Get(updatedMachine, infrav1.OpenStackAuthenticationSucceeded)
		Expect(condition).ToNot(BeNil())
		Expect(condition.Reason).To(Equal(infrav1.OpenStackAuthenticationFailedReason))
		Expect(condition.Severity).To(Equal(clusterv1beta1.ConditionSeverityError))
		Expect(condition.Message).To(ContainSubstring("Failed to create OpenStack client scope"))
	})

	It("should set OpenStackAuthenticationSucceededCondition to False when namespace is denied access to ClusterIdentity", func() {
		testMachine.SetName("identity-access-denied-machine")
		testMachine.Spec.IdentityRef = &infrav1.OpenStackIdentityReference{
			Type:      "ClusterIdentity",
			Name:      "test-cluster-identity",
			CloudName: "openstack",
		}

		err := k8sClient.Create(ctx, capiCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, testCluster)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, capiMachine)
		Expect(err).To(BeNil())
		err = k8sClient.Create(ctx, testMachine)
		Expect(err).To(BeNil())

		identityAccessErr := &scope.IdentityAccessDeniedError{
			IdentityName:       "test-cluster-identity",
			RequesterNamespace: testNamespace,
		}
		machineMockFactory.SetClientScopeCreateError(identityAccessErr)

		req := reconcile.Request{
			NamespacedName: client.ObjectKey{
				Name:      testMachine.Name,
				Namespace: testMachine.Namespace,
			},
		}
		result, err := machineReconciler.Reconcile(ctx, req)

		Expect(err).To(MatchError(identityAccessErr))
		Expect(result).To(Equal(reconcile.Result{}))

		// Fetch the updated OpenStackMachine to verify the condition was set
		updatedMachine := &infrav1.OpenStackMachine{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: testMachine.Name, Namespace: testMachine.Namespace}, updatedMachine)).To(Succeed())

		// Verify OpenStackAuthenticationSucceededCondition is set to False
		Expect(v1beta1conditions.IsFalse(updatedMachine, infrav1.OpenStackAuthenticationSucceeded)).To(BeTrue())
		condition := v1beta1conditions.Get(updatedMachine, infrav1.OpenStackAuthenticationSucceeded)
		Expect(condition).ToNot(BeNil())
		Expect(condition.Reason).To(Equal(infrav1.OpenStackAuthenticationFailedReason))
		Expect(condition.Severity).To(Equal(clusterv1beta1.ConditionSeverityError))
		Expect(condition.Message).To(ContainSubstring("Failed to create OpenStack client scope"))
	})
})
