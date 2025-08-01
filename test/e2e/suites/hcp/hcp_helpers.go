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
	"path/filepath"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/cluster-api-provider-openstack/test/e2e/shared"
)

// HCPTestContext holds context for HCP test isolation.
type HCPTestContext struct {
	ManagementCluster          framework.ClusterProxy
	ManagementClusterResources *clusterctl.ApplyClusterTemplateAndWaitResult
	ManagementNamespace        *corev1.Namespace
	KamajiInstalled            bool
	IsolationID                string
}

// createHCPTestContext creates an isolated test context for HCP tests.
func createHCPTestContext(ctx context.Context) *HCPTestContext {
	isolationID := fmt.Sprintf("hcp-%d", time.Now().Unix())
	shared.Logf("Creating HCP test context with isolation ID: %s", isolationID)

	return &HCPTestContext{
		IsolationID:     isolationID,
		KamajiInstalled: false,
	}
}

// setupSharedManagementCluster creates or reuses the management cluster for HCP tests.
func setupSharedManagementCluster(ctx context.Context, hcpCtx *HCPTestContext, e2eCtx *shared.E2EContext) {
	shared.Logf("Setting up shared management cluster for HCP tests")

	// Create dedicated namespace for management cluster
	hcpCtx.ManagementNamespace = shared.SetupSpecNamespace(ctx, "hcp-mgmt-"+hcpCtx.IsolationID, e2eCtx)
	hcpCtx.ManagementClusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)

	// Create management cluster using clusterctl
	managementClusterName := fmt.Sprintf("hcp-mgmt-%s", hcpCtx.ManagementNamespace.Name)
	configCluster := clusterctl.ConfigClusterInput{
		LogFolder:                filepath.Join(e2eCtx.Settings.ArtifactFolder, "clusters", e2eCtx.Environment.BootstrapClusterProxy.GetName()),
		ClusterctlConfigPath:     e2eCtx.Environment.ClusterctlConfigPath,
		KubeconfigPath:           e2eCtx.Environment.BootstrapClusterProxy.GetKubeconfigPath(),
		InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
		Flavor:                   shared.FlavorHCPManagement,
		Namespace:                hcpCtx.ManagementNamespace.Name,
		ClusterName:              managementClusterName,
		KubernetesVersion:        e2eCtx.E2EConfig.MustGetVariable(shared.KubernetesVersion),
		ControlPlaneMachineCount: ptr.To(int64(1)),
		WorkerMachineCount:       ptr.To(int64(2)),
	}

	shared.Logf("Creating management cluster %s", managementClusterName)
	clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
		ClusterProxy:                 e2eCtx.Environment.BootstrapClusterProxy,
		ConfigCluster:                configCluster,
		WaitForClusterIntervals:      e2eCtx.E2EConfig.GetIntervals("hcp", "wait-cluster"),
		WaitForControlPlaneIntervals: e2eCtx.E2EConfig.GetIntervals("hcp", "wait-control-plane"),
		WaitForMachineDeployments:    e2eCtx.E2EConfig.GetIntervals("hcp", "wait-worker-nodes"),
	}, hcpCtx.ManagementClusterResources)

	hcpCtx.ManagementCluster = e2eCtx.Environment.BootstrapClusterProxy.GetWorkloadCluster(ctx, hcpCtx.ManagementNamespace.Name, managementClusterName)
	shared.Logf("Management cluster %s created successfully", managementClusterName)
}

// installKamajiUsingClusterctl installs Kamaji using clusterctl.
func installKamajiUsingClusterctl(ctx context.Context, hcpCtx *HCPTestContext, e2eCtx *shared.E2EContext) {
	if hcpCtx.KamajiInstalled {
		shared.Logf("Kamaji already installed, skipping")
		return
	}

	shared.Logf("Installing Kamaji using clusterctl")

	// Initialize Kamaji provider on management cluster
	clusterctl.InitManagementClusterAndWatchControllerLogs(ctx, clusterctl.InitManagementClusterAndWatchControllerLogsInput{
		ClusterProxy:            hcpCtx.ManagementCluster,
		ClusterctlConfigPath:    e2eCtx.Environment.ClusterctlConfigPath,
		InfrastructureProviders: []string{"openstack"},
		BootstrapProviders:      []string{"kubeadm"},
		ControlPlaneProviders:   []string{"kubeadm"},
		CoreProvider:            "",
		LogFolder:               filepath.Join(e2eCtx.Settings.ArtifactFolder, "clusters", hcpCtx.ManagementCluster.GetName()),
	}, e2eCtx.E2EConfig.GetIntervals("hcp", "wait-controllers")...)

	// Create default datastore for Kamaji
	createDefaultDatastore(ctx, hcpCtx.ManagementCluster, e2eCtx)

	// Wait for Kamaji to be ready
	waitForKamajiReady(ctx, hcpCtx.ManagementCluster, e2eCtx)

	hcpCtx.KamajiInstalled = true
	shared.Logf("Kamaji installation completed successfully")
}

// createDefaultDatastore creates the default etcd datastore for Kamaji.
func createDefaultDatastore(ctx context.Context, managementCluster framework.ClusterProxy, e2eCtx *shared.E2EContext) {
	shared.Logf("Creating default datastore for Kamaji")

	datastore := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kamaji.clastix.io/v1alpha1",
			"kind":       "DataStore",
			"metadata": map[string]interface{}{
				"name": e2eCtx.E2EConfig.MustGetVariable("CLUSTER_DATASTORE"),
			},
			"spec": map[string]interface{}{
				"driver": "etcd",
				"endpoints": []interface{}{
					"etcd-cluster.kamaji-system.svc.cluster.local:2379",
				},
			},
		},
	}

	Eventually(func() error {
		return managementCluster.GetClient().Create(ctx, datastore)
	}, e2eCtx.E2EConfig.GetIntervals("hcp", "wait-controllers")...).Should(Succeed())

	shared.Logf("Default datastore created successfully")
}

// waitForKamajiReady waits for Kamaji components to be ready.
func waitForKamajiReady(ctx context.Context, managementCluster framework.ClusterProxy, e2eCtx *shared.E2EContext) {
	shared.Logf("Waiting for Kamaji components to be ready")

	Eventually(func() bool {
		podList := &corev1.PodList{}
		err := managementCluster.GetClient().List(ctx, podList, client.InNamespace(e2eCtx.E2EConfig.MustGetVariable("KAMAJI_NAMESPACE")))
		if err != nil {
			return false
		}

		if len(podList.Items) == 0 {
			return false
		}

		for _, pod := range podList.Items {
			if pod.Status.Phase != corev1.PodRunning {
				return false
			}
		}
		return true
	}, e2eCtx.E2EConfig.GetIntervals("hcp", "wait-controllers")...).Should(BeTrue())

	shared.Logf("Kamaji components are ready")
}

// waitForKamajiControlPlane waits for a KamajiControlPlane to be ready.
func waitForKamajiControlPlane(ctx context.Context, managementCluster framework.ClusterProxy, namespace, name string, e2eCtx *shared.E2EContext) {
	shared.Logf("Waiting for KamajiControlPlane %s/%s to be ready", namespace, name)

	kcpGVK := schema.GroupVersionKind{
		Group:   "controlplane.cluster.x-k8s.io",
		Version: "v1alpha1",
		Kind:    "KamajiControlPlane",
	}

	Eventually(func() bool {
		kcp := &unstructured.Unstructured{}
		kcp.SetGroupVersionKind(kcpGVK)

		err := managementCluster.GetClient().Get(ctx, client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}, kcp)
		if err != nil {
			return false
		}

		conditions, found, err := unstructured.NestedSlice(kcp.Object, "status", "conditions")
		if err != nil || !found {
			return false
		}

		for _, condition := range conditions {
			condMap, ok := condition.(map[string]interface{})
			if !ok {
				continue
			}

			if condMap["type"] == "Ready" && condMap["status"] == "True" {
				return true
			}
		}
		return false
	}, e2eCtx.E2EConfig.GetIntervals("hcp", "wait-control-plane")...).Should(BeTrue())

	shared.Logf("KamajiControlPlane %s/%s is ready", namespace, name)
}

// verifyHCPWorkloadCluster performs comprehensive validation of HCP workload cluster.
func verifyHCPWorkloadCluster(ctx context.Context, workloadCluster framework.ClusterProxy, clusterResources *clusterctl.ApplyClusterTemplateAndWaitResult, e2eCtx *shared.E2EContext) {
	shared.Logf("Verifying HCP workload cluster")

	// Verify cluster is accessible
	Eventually(func() error {
		_, err := workloadCluster.GetClientSet().Discovery().ServerVersion()
		return err
	}, e2eCtx.E2EConfig.GetIntervals("hcp", "wait-nodes-ready")...).Should(Succeed())

	// Verify worker nodes are ready
	Eventually(func() bool {
		nodeList := &corev1.NodeList{}
		err := workloadCluster.GetClient().List(ctx, nodeList)
		if err != nil {
			return false
		}

		readyNodes := 0
		for _, node := range nodeList.Items {
			for _, condition := range node.Status.Conditions {
				if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
					readyNodes++
					break
				}
			}
		}

		expectedNodes := int(*clusterResources.MachineDeployments[0].Spec.Replicas)
		return readyNodes == expectedNodes
	}, e2eCtx.E2EConfig.GetIntervals("hcp", "wait-nodes-ready")...).Should(BeTrue())

	// Verify no control plane nodes (HCP workload should have only workers)
	nodeList := &corev1.NodeList{}
	Expect(workloadCluster.GetClient().List(ctx, nodeList)).To(Succeed())

	for _, node := range nodeList.Items {
		Expect(node.Labels).NotTo(HaveKey("node-role.kubernetes.io/control-plane"))
		Expect(node.Labels).NotTo(HaveKey("node-role.kubernetes.io/master"))
	}

	shared.Logf("HCP workload cluster verification completed successfully")
}

// isolatedCleanup provides isolated cleanup for test resources.
func isolatedCleanup(ctx context.Context, hcpCtx *HCPTestContext, testNamespace *corev1.Namespace, clusterResources *clusterctl.ApplyClusterTemplateAndWaitResult, e2eCtx *shared.E2EContext) {
	shared.Logf("Starting isolated cleanup for test context %s", hcpCtx.IsolationID)

	// Clean up test-specific resources first
	if clusterResources != nil && clusterResources.Cluster != nil {
		shared.Logf("Cleaning up workload cluster %s", clusterResources.Cluster.Name)
		shared.DumpSpecResourcesAndCleanup(ctx, "hcp-workload", testNamespace, e2eCtx)
	}

	// Note: Management cluster cleanup is handled separately in suite cleanup
	shared.Logf("Isolated cleanup completed for test context %s", hcpCtx.IsolationID)
}

// cleanupSharedManagementCluster cleans up the shared management cluster (called once per suite).
func cleanupSharedManagementCluster(ctx context.Context, hcpCtx *HCPTestContext, e2eCtx *shared.E2EContext) {
	if hcpCtx.ManagementCluster == nil {
		return
	}

	shared.Logf("Cleaning up shared management cluster for context %s", hcpCtx.IsolationID)

	// Clean up management cluster resources
	if hcpCtx.ManagementClusterResources != nil && hcpCtx.ManagementClusterResources.Cluster != nil {
		shared.DumpSpecResourcesAndCleanup(ctx, "hcp-management", hcpCtx.ManagementNamespace, e2eCtx)
	}

	shared.Logf("Shared management cluster cleanup completed")
}

// waitForTerminalError waits for a cluster to reach a terminal error state (for broken HCP tests).
func waitForTerminalError(ctx context.Context, managementCluster framework.ClusterProxy, namespace, clusterName string, e2eCtx *shared.E2EContext) {
	shared.Logf("Waiting for cluster %s/%s to reach terminal error state", namespace, clusterName)

	Eventually(func() bool {
		cluster := &unstructured.Unstructured{}
		cluster.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "cluster.x-k8s.io",
			Version: "v1beta1",
			Kind:    "Cluster",
		})

		err := managementCluster.GetClient().Get(ctx, client.ObjectKey{
			Namespace: namespace,
			Name:      clusterName,
		}, cluster)
		if err != nil {
			return false
		}

		// Check for terminal error conditions
		conditions, found, err := unstructured.NestedSlice(cluster.Object, "status", "conditions")
		if err != nil || !found {
			return false
		}

		for _, condition := range conditions {
			condMap, ok := condition.(map[string]interface{})
			if !ok {
				continue
			}

			// Look for terminal error indicators
			if condMap["type"] == "InfrastructureReady" && condMap["status"] == "False" {
				if reason, ok := condMap["reason"].(string); ok &&
					(reason == "InvalidConfiguration" || reason == "ProvisioningFailed") {
					return true
				}
			}
		}
		return false
	}, e2eCtx.E2EConfig.GetIntervals("hcp", "wait-cluster")...).Should(BeTrue())

	shared.Logf("Cluster %s/%s reached terminal error state as expected", namespace, clusterName)
}
