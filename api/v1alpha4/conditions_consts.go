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

package v1alpha4

import clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"

// OpenstackCluster Conditions and Reasons.
const (
	// NetworkInfrastructureReadyCondition reports of current status of cluster infrastructure.
	NetworkInfrastructureReadyCondition = "NetworkInfrastructureReady"
	// LoadBalancerProvisioningReason API Server endpoint for the loadbalancer.
	LoadBalancerProvisioningReason = "LoadBalancerProvisioning"
	// LoadBalancerProvisioningFailedReason used for failure during provisioning of loadbalancer.
	LoadBalancerProvisioningFailedReason = "LoadBalancerProvisioningFailed"
	// NamespaceNotAllowedByIdentity used to indicate cluster in a namespace not allowed by identity.
	NamespaceNotAllowedByIdentity = "NamespaceNotAllowedByIdentity"
)

// OpenStackMachine Conditions and Reasons.
const (
	// VMRunningCondition reports on current status of the Azure VM.
	VMRunningCondition clusterv1.ConditionType = "VMRunning"
	// VMNCreatingReason used when the vm creation is in progress.
	VMNCreatingReason = "VMCreating"
	// VMNUpdatingReason used when the vm updating is in progress.
	VMNUpdatingReason = "VMUpdating"
	// VMNotFoundReason used when the vm couldn't be retrieved.
	VMNotFoundReason = "VMNotFound"
	// VMDeletingReason used when the vm is in a deleting state.
	VMDDeletingReason = "VMDeleting"
	// VMStoppedReason vm is in a stopped state.
	VMStoppedReason = "VMStopped"
	// VMProvisionFailedReason used for failures during vm provisioning.
	VMProvisionFailedReason = "VMProvisionFailed"
	// WaitingForClusterInfrastructureReason used when machine is waiting for cluster infrastructure to be ready before proceeding.
	WaitingForClusterInfrastructureReason = "WaitingForClusterInfrastructure"
	// WaitingForBootstrapDataReason used when machine is waiting for bootstrap data to be ready before proceeding.
	WaitingForBootstrapDataReason = "WaitingForBootstrapData"
)
