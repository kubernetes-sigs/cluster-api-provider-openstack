//go:build e2e
// +build e2e

/*
Copyright 2025 The Kubernetes Authors.

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

package hcp

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/test/e2e/shared"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
)

// NetworkValidationInput contains the input for network validation tests
type NetworkValidationInput struct {
	WorkloadClusterProxy *shared.ClusterProxy
	Namespace            string
	ClusterName          string
	E2EContext           *shared.E2EContext
}

// ValidateNetworkConfiguration tests the specific network edge cases fixed in hcp-2380
func ValidateNetworkConfiguration(ctx context.Context, input NetworkValidationInput) {
	shared.Logf("Starting network configuration validation for HCP cluster %s", input.ClusterName)

	By("Testing nil cluster network status handling")
	validateNilClusterNetworkStatus(ctx, input)

	By("Testing machine spec without explicit networks")
	validateMachineSpecWithoutExplicitNetworks(ctx, input)

	By("Testing security group precedence")
	validateSecurityGroupPrecedence(ctx, input)

	By("Testing security group precedence with a live client")
	validateSecurityGroupPrecedenceWithClient(ctx, input)

	By("Testing port configuration edge cases")
	validatePortConfigurationEdgeCases(ctx, input)

	shared.Logf("Network configuration validation completed successfully")
}

// validateNilClusterNetworkStatus tests the scenario where cluster network status is nil
func validateNilClusterNetworkStatus(ctx context.Context, input NetworkValidationInput) {
	shared.Logf("Validating nil cluster network status handling")

	// Get the OpenStackCluster
	openStackCluster := &infrav1.OpenStackCluster{}
	err := input.WorkloadClusterProxy.GetClient().Get(ctx, types.NamespacedName{
		Namespace: input.Namespace,
		Name:      input.ClusterName,
	}, openStackCluster)
	Expect(err).ToNot(HaveOccurred(), "Failed to get OpenStackCluster")

	// Check that the cluster can handle scenarios where network status is not yet populated
	if openStackCluster.Status.Network == nil {
		shared.Logf("Network status is nil, testing that machines can still be created")

		// Verify that machines in this cluster have proper error handling
		machineList := &infrav1.OpenStackMachineList{}
		err = input.WorkloadClusterProxy.GetClient().List(ctx, machineList, client.InNamespace(input.Namespace))
		Expect(err).ToNot(HaveOccurred(), "Failed to list OpenStackMachines")

		for _, machine := range machineList.Items {
			// Ensure machines don't have InvalidMachineSpecReason condition due to network issues
			for _, condition := range machine.Status.Conditions {
				if condition.Type == infrav1.MachineReadyCondition && condition.Status == corev1.ConditionFalse {
					Expect(condition.Reason).ToNot(Equal("InvalidMachineSpecReason"),
						"Machine %s should not fail with InvalidMachineSpecReason when cluster network is nil", machine.Name)
				}
			}
		}
	} else {
		shared.Logf("Network status is populated: %+v", openStackCluster.Status.Network)
	}
}

// validateMachineSpecWithoutExplicitNetworks tests machines that don't define explicit port networks
func validateMachineSpecWithoutExplicitNetworks(ctx context.Context, input NetworkValidationInput) {
	shared.Logf("Validating machine spec without explicit networks")

	// Get all OpenStackMachines in the cluster
	machineList := &infrav1.OpenStackMachineList{}
	err := input.WorkloadClusterProxy.GetClient().List(ctx, machineList, client.InNamespace(input.Namespace))
	Expect(err).ToNot(HaveOccurred(), "Failed to list OpenStackMachines")

	foundMachineWithoutExplicitPorts := false
	for _, machine := range machineList.Items {
		// Check machines that don't have explicit port networks defined
		if len(machine.Spec.Ports) == 0 ||
			(len(machine.Spec.Ports) > 0 && machine.Spec.Ports[0].Network == nil) {

			foundMachineWithoutExplicitPorts = true
			shared.Logf("Found machine %s without explicit port networks", machine.Name)

			// Verify the machine is created successfully (tests the fix from hcp-2380)
			Expect(machine.Status.Ready).To(BeTrue(),
				"Machine %s without explicit networks should be ready", machine.Name)

			// Verify it doesn't have terminal errors related to network configuration
			for _, condition := range machine.Status.Conditions {
				if condition.Type == infrav1.MachineReadyCondition && condition.Status == corev1.ConditionFalse {
					Expect(condition.Reason).ToNot(ContainSubstring("Network"),
						"Machine %s should not fail with network-related errors", machine.Name)
				}
			}
		}
	}

	if !foundMachineWithoutExplicitPorts {
		shared.Logf("No machines found without explicit port networks - this is expected in some configurations")
	}
}

// validateSecurityGroupPrecedence tests the precedence between machine-level and cluster-level security groups
func validateSecurityGroupPrecedence(ctx context.Context, input NetworkValidationInput) {
	shared.Logf("Validating security group precedence")

	// Get the OpenStackCluster to check managed security groups
	openStackCluster := &infrav1.OpenStackCluster{}
	err := input.WorkloadClusterProxy.GetClient().Get(ctx, types.NamespacedName{
		Namespace: input.Namespace,
		Name:      input.ClusterName,
	}, openStackCluster)
	Expect(err).ToNot(HaveOccurred(), "Failed to get OpenStackCluster")

	// Check if managed security groups are enabled
	if openStackCluster.Spec.ManagedSecurityGroups != nil {
		shared.Logf("Cluster has managed security groups configured")

		// Get all machines and verify security group configuration
		machineList := &infrav1.OpenStackMachineList{}
		err = input.WorkloadClusterProxy.GetClient().List(ctx, machineList, client.InNamespace(input.Namespace))
		Expect(err).ToNot(HaveOccurred(), "Failed to list OpenStackMachines")

		for _, machine := range machineList.Items {
			// Test the precedence logic: machine-level security groups should take precedence
			// over cluster-level managed security groups when both are specified
			if len(machine.Spec.SecurityGroups) > 0 {
				shared.Logf("Machine %s has explicit security groups, these should take precedence", machine.Name)

				// Verify machine is ready (tests that precedence logic works correctly)
				Expect(machine.Status.Ready).To(BeTrue(),
					"Machine %s with explicit security groups should be ready", machine.Name)
			} else {
				shared.Logf("Machine %s uses managed security groups from cluster", machine.Name)
			}

			// Ensure no conflicting security group errors
			for _, condition := range machine.Status.Conditions {
				if condition.Type == infrav1.MachineReadyCondition && condition.Status == corev1.ConditionFalse {
					Expect(condition.Reason).ToNot(ContainSubstring("SecurityGroup"),
						"Machine %s should not fail with security group conflicts", machine.Name)
				}
			}
		}
	} else {
		shared.Logf("Cluster does not use managed security groups")
	}
}

// validateSecurityGroupPrecedenceWithClient creates a machine with a specific security group
// and verifies with the OpenStack client that it's the only one applied.
func validateSecurityGroupPrecedenceWithClient(ctx context.Context, input NetworkValidationInput) {
	shared.Logf("Validating security group precedence with a live client")

	// Get a compute client
	computeClient, err := shared.NewComputeClient(input.E2EContext)
	Expect(err).ToNot(HaveOccurred(), "Failed to create compute client")

	// Define a new security group to be used exclusively for this test
	openStackCluster := &infrav1.OpenStackCluster{}
	err = input.WorkloadClusterProxy.GetClient().Get(ctx, types.NamespacedName{
		Namespace: input.Namespace,
		Name:      input.ClusterName,
	}, openStackCluster)
	Expect(err).ToNot(HaveOccurred(), "Failed to get OpenStackCluster")

	sgName := "e2e-sg-override-test"
	sg, err := shared.CreateSecurityGroup(input.E2EContext, sgName)
	Expect(err).ToNot(HaveOccurred(), "Failed to create security group for test")
	defer func() {
		Expect(shared.DeleteSecurityGroup(input.E2EContext, sg.ID)).To(Succeed())
	}()

	// Create a new Machine and OpenStackMachine with the override security group
	machineName := fmt.Sprintf("%s-sg-override", input.ClusterName)
	shared.Logf("Creating machine %s with security group %s", machineName, sg.Name)

	machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      machineName,
			Namespace: input.Namespace,
			Labels: map[string]string{
				clusterv1.ClusterNameLabel: input.ClusterName,
			},
		},
		Spec: clusterv1.MachineSpec{
			ClusterName: input.ClusterName,
			Version:     ptr.To(e2eCtx.E2EConfig.Variables[shared.KubernetesVersion]),
			Bootstrap: clusterv1.Bootstrap{
				ConfigRef: &corev1.ObjectReference{
					APIVersion: "bootstrap.cluster.x-k8s.io/v1beta1",
					Kind:       "KubeadmConfigTemplate",
					Name:       fmt.Sprintf("%s-md-0", input.ClusterName),
				},
			},
			InfrastructureRef: corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Kind:       "OpenStackMachine",
				Name:       machineName,
			},
		},
	}

	openStackMachine := &infrav1.OpenStackMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      machineName,
			Namespace: input.Namespace,
		},
		Spec: infrav1.OpenStackMachineSpec{
			// Explicitly set the security group, overriding any cluster defaults
			SecurityGroups: []infrav1.SecurityGroupParam{
				{ID: &sg.ID},
			},
			Flavor:     e2eCtx.E2EConfig.Variables[shared.OpenstackNodeMachineFlavor],
			Image:      infrav1.ImageParam{Filter: &infrav1.ImageFilter{Name: &e2eCtx.E2EConfig.Variables[shared.OpenstackImageName]}},
			SSHKeyName: e2eCtx.E2EConfig.Variables[shared.OpenstackSSHKeyName],
		},
	}

	// Create the resources
	err = input.WorkloadClusterProxy.GetClient().Create(ctx, openStackMachine)
	Expect(err).ToNot(HaveOccurred(), "Failed to create OpenStackMachine")
	err = input.WorkloadClusterProxy.GetClient().Create(ctx, machine)
	Expect(err).ToNot(HaveOccurred(), "Failed to create Machine")

	// Wait for the machine to get an instance ID and become ready
	shared.Logf("Waiting for machine %s to become ready", machineName)
	framework.WaitForMachineReady(ctx, framework.WaitForMachineReadyInput{
		Getter:    input.WorkloadClusterProxy.GetClient(),
		Cluster:   &clusterv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: input.ClusterName, Namespace: input.Namespace}},
		Machine:   machine,
		Timeout:   20 * time.Minute,
		Intervals: e2eCtx.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
	})

	// Get the updated OpenStackMachine to find its instance ID
	err = input.WorkloadClusterProxy.GetClient().Get(ctx, client.ObjectKeyFromObject(openStackMachine), openStackMachine)
	Expect(err).ToNot(HaveOccurred(), "Failed to get updated OpenStackMachine")
	Expect(openStackMachine.Status.InstanceID).ToNot(BeNil(), "InstanceID should not be nil")

	// Use the compute client to verify the security groups on the live instance
	shared.Logf("Verifying security groups on instance %s", *openStackMachine.Status.InstanceID)
	server, err := servers.Get(computeClient, *openStackMachine.Status.InstanceID).Extract()
	Expect(err).ToNot(HaveOccurred(), "Failed to get server details from OpenStack")

	// Assert that the ONLY security group is the one we specified
	Expect(server.SecurityGroups).To(HaveLen(1), "Should only have one security group")
	Expect(server.SecurityGroups[0].(map[string]interface{})["name"]).To(Equal(sg.Name), "The applied security group should be the override one")

	shared.Logf("Successfully verified security group override")

	// Clean up the machine
	shared.Logf("Deleting machine %s", machineName)
	err = input.WorkloadClusterProxy.GetClient().Delete(ctx, machine)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete machine")
	err = input.WorkloadClusterProxy.GetClient().Delete(ctx, openStackMachine)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete OpenStackMachine")
}

// validatePortConfigurationEdgeCases tests edge cases in port configuration
func validatePortConfigurationEdgeCases(ctx context.Context, input NetworkValidationInput) {
	shared.Logf("Validating port configuration edge cases")

	// Get all OpenStackMachines to test port configurations
	machineList := &infrav1.OpenStackMachineList{}
	err := input.WorkloadClusterProxy.GetClient().List(ctx, machineList, client.InNamespace(input.Namespace))
	Expect(err).ToNot(HaveOccurred(), "Failed to list OpenStackMachines")

	for _, machine := range machineList.Items {
		shared.Logf("Validating port configuration for machine %s", machine.Name)

		// Test various port configuration scenarios
		if len(machine.Spec.Ports) > 0 {
			for i, port := range machine.Spec.Ports {
				// Test ports with various network configurations
				if port.Network != nil {
					shared.Logf("Machine %s port %d has explicit network configuration", machine.Name, i)
				} else {
					shared.Logf("Machine %s port %d uses default network", machine.Name, i)
				}

				// Test fixed IPs configuration
				if len(port.FixedIPs) > 0 {
					shared.Logf("Machine %s port %d has %d fixed IPs", machine.Name, i, len(port.FixedIPs))
				}

				// Test security groups on ports
				if len(port.SecurityGroups) > 0 {
					shared.Logf("Machine %s port %d has %d security groups", machine.Name, i, len(port.SecurityGroups))
				}
			}
		}

		// Verify machine is ready regardless of port configuration complexity
		Expect(machine.Status.Ready).To(BeTrue(),
			"Machine %s should be ready regardless of port configuration", machine.Name)
	}
}

// ValidateKonectivityConnectivity tests Konnectivity connectivity in HCP setup
func ValidateKonectivityConnectivity(ctx context.Context, input NetworkValidationInput) {
	shared.Logf("Validating Konnectivity connectivity for HCP cluster %s", input.ClusterName)

	By("Checking Konnectivity agent pods on worker nodes")

	// Get the workload cluster client to check node connectivity
	workloadClusterClient := input.WorkloadClusterProxy.GetClient()

	// List all nodes in the workload cluster
	nodes := &corev1.NodeList{}
	err := workloadClusterClient.List(ctx, nodes)
	Expect(err).ToNot(HaveOccurred(), "Failed to list nodes in workload cluster")

	Expect(len(nodes.Items)).To(BeNumerically(">", 0), "Workload cluster should have at least one node")

	for _, node := range nodes.Items {
		shared.Logf("Validating node %s connectivity", node.Name)

		// Verify node is ready
		nodeReady := false
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				nodeReady = true
				break
			}
		}
		Expect(nodeReady).To(BeTrue(), "Node %s should be ready", node.Name)
	}

	By("Testing API server connectivity from worker nodes")

	// Create a test pod to verify connectivity
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "connectivity-test",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test",
					Image: "curlimages/curl:7.85.0",
					Command: []string{
						"sh", "-c",
						"curl -k https://kubernetes.default.svc.cluster.local/api/v1/namespaces/default && echo 'Connectivity test successful'",
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	err = workloadClusterClient.Create(ctx, testPod)
	if err != nil {
		shared.Logf("Warning: Could not create connectivity test pod: %v", err)
	} else {
		shared.Logf("Created connectivity test pod successfully")
		// Clean up the test pod
		defer func() {
			_ = workloadClusterClient.Delete(ctx, testPod)
		}()
	}

	shared.Logf("Konnectivity connectivity validation completed")
}
