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
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"

	"sigs.k8s.io/cluster-api-provider-openstack/test/e2e/shared"
)

const specName = "hcp"

var (
	sharedHCPContext *HCPTestContext
	setupOnce        sync.Once
)

var _ = Describe("HCP (Hosted Control Plane) tests", func() {
	Describe("Management cluster verification", func() {
		It("should create and manage HCP-capable cluster", func(ctx context.Context) {
			// Setup shared HCP infrastructure once
			setupOnce.Do(func() {
				shared.Logf("Setting up shared HCP test infrastructure")

				// Create isolated test context for this suite
				sharedHCPContext = createHCPTestContext(ctx)

				// Set up shared management cluster
				setupSharedManagementCluster(ctx, sharedHCPContext, e2eCtx)

				// Install Kamaji using clusterctl
				installKamajiUsingClusterctl(ctx, sharedHCPContext, e2eCtx)

				shared.Logf("Shared HCP infrastructure ready with isolation ID: %s", sharedHCPContext.IsolationID)
			})

			shared.Logf("Verifying management cluster capabilities")

			// Verify management cluster is operational
			Expect(sharedHCPContext.ManagementCluster).ToNot(BeNil())
			Expect(sharedHCPContext.KamajiInstalled).To(BeTrue())

			// Verify cluster can schedule workloads
			Eventually(func() error {
				_, err := sharedHCPContext.ManagementCluster.GetClientSet().Discovery().ServerVersion()
				return err
			}, e2eCtx.E2EConfig.GetIntervals(specName, "wait-nodes-ready")...).Should(Succeed())

			shared.Logf("Management cluster verification completed successfully")
		})
	})

	Describe("Workload cluster with hosted control plane", func() {
		var (
			// Per-test isolation
			testNamespace    *corev1.Namespace
			clusterResources *clusterctl.ApplyClusterTemplateAndWaitResult
		)

		BeforeEach(func(ctx context.Context) {
			// Ensure shared HCP infrastructure is ready
			setupOnce.Do(func() {
				shared.Logf("Setting up shared HCP test infrastructure")
				sharedHCPContext = createHCPTestContext(ctx)
				setupSharedManagementCluster(ctx, sharedHCPContext, e2eCtx)
				installKamajiUsingClusterctl(ctx, sharedHCPContext, e2eCtx)
				shared.Logf("Shared HCP infrastructure ready with isolation ID: %s", sharedHCPContext.IsolationID)
			})

			// Create isolated namespace for this specific test
			testNamespace = shared.SetupSpecNamespace(ctx, specName+"-workload", e2eCtx)
			clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)
		})

		AfterEach(func(ctx context.Context) {
			isolatedCleanup(ctx, sharedHCPContext, testNamespace, clusterResources, e2eCtx)
		})

		It("should create workload cluster with external control plane", func(ctx context.Context) {
			shared.Logf("Creating HCP workload cluster using shared management cluster")

			// Create HCP workload cluster configuration
			clusterName := fmt.Sprintf("hcp-workload-%s", testNamespace.Name)
			configCluster := clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(e2eCtx.Settings.ArtifactFolder, "clusters", e2eCtx.Environment.BootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     e2eCtx.Environment.ClusterctlConfigPath,
				KubeconfigPath:           e2eCtx.Environment.BootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
				Flavor:                   shared.FlavorHCPWorkload,
				Namespace:                testNamespace.Name,
				ClusterName:              clusterName,
				KubernetesVersion:        e2eCtx.E2EConfig.MustGetVariable(shared.KubernetesVersion),
				ControlPlaneMachineCount: ptr.To(int64(0)), // No control plane machines for HCP
				WorkerMachineCount:       ptr.To(int64(2)),
			}

			shared.Logf("Creating HCP workload cluster: %s", clusterName)
			clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
				ClusterProxy:                 e2eCtx.Environment.BootstrapClusterProxy,
				ConfigCluster:                configCluster,
				WaitForClusterIntervals:      e2eCtx.E2EConfig.GetIntervals(specName, "wait-cluster"),
				WaitForControlPlaneIntervals: e2eCtx.E2EConfig.GetIntervals(specName, "wait-control-plane"),
				WaitForMachineDeployments:    e2eCtx.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
			}, clusterResources)

			// Wait for KamajiControlPlane to be ready
			kcpName := fmt.Sprintf("%s-control-plane", clusterName)
			waitForKamajiControlPlane(ctx, sharedHCPContext.ManagementCluster, testNamespace.Name, kcpName, e2eCtx)

			// Get workload cluster proxy
			workloadCluster := e2eCtx.Environment.BootstrapClusterProxy.GetWorkloadCluster(ctx, testNamespace.Name, clusterName)

			// Verification of HCP workload cluster
			verifyHCPWorkloadCluster(ctx, workloadCluster, clusterResources, e2eCtx)

			shared.Logf("HCP workload cluster %s created and verified successfully", clusterName)
		})
	})

	Describe("Graceful failure handling", func() {
		var (
			testNamespace    *corev1.Namespace
			clusterResources *clusterctl.ApplyClusterTemplateAndWaitResult
		)

		BeforeEach(func(ctx context.Context) {
			// Ensure shared HCP infrastructure is ready
			setupOnce.Do(func() {
				shared.Logf("Setting up shared HCP test infrastructure")
				sharedHCPContext = createHCPTestContext(ctx)
				setupSharedManagementCluster(ctx, sharedHCPContext, e2eCtx)
				installKamajiUsingClusterctl(ctx, sharedHCPContext, e2eCtx)
				shared.Logf("Shared HCP infrastructure ready with isolation ID: %s", sharedHCPContext.IsolationID)
			})

			testNamespace = shared.SetupSpecNamespace(ctx, specName+"-broken", e2eCtx)
			clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)
		})

		AfterEach(func(ctx context.Context) {
			isolatedCleanup(ctx, sharedHCPContext, testNamespace, clusterResources, e2eCtx)
		})

		It("should handle broken HCP configuration gracefully", func(ctx context.Context) {
			shared.Logf("Testing graceful failure handling for broken HCP configuration")

			// Create broken HCP cluster configuration
			clusterName := fmt.Sprintf("hcp-broken-%s", testNamespace.Name)
			configCluster := clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(e2eCtx.Settings.ArtifactFolder, "clusters", e2eCtx.Environment.BootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     e2eCtx.Environment.ClusterctlConfigPath,
				KubeconfigPath:           e2eCtx.Environment.BootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
				Flavor:                   shared.FlavorBrokenHCP, // This template has broken networking
				Namespace:                testNamespace.Name,
				ClusterName:              clusterName,
				KubernetesVersion:        e2eCtx.E2EConfig.MustGetVariable(shared.KubernetesVersion),
				ControlPlaneMachineCount: ptr.To(int64(1)),
				WorkerMachineCount:       ptr.To(int64(1)),
			}

			shared.Logf("Creating broken HCP cluster for graceful failure testing: %s", clusterName)

			// Apply broken template - this should fail gracefully
			func() {
				defer func() {
					if r := recover(); r != nil {
						shared.Logf("Graceful recovery from panic during broken cluster creation: %v", r)
					}
				}()

				clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
					ClusterProxy:                 e2eCtx.Environment.BootstrapClusterProxy,
					ConfigCluster:                configCluster,
					WaitForClusterIntervals:      e2eCtx.E2EConfig.GetIntervals(specName, "wait-cluster"),
					WaitForControlPlaneIntervals: e2eCtx.E2EConfig.GetIntervals(specName, "wait-control-plane"),
					WaitForMachineDeployments:    e2eCtx.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
				}, clusterResources)
			}()

			// Verify cluster reaches terminal error state gracefully
			waitForTerminalError(ctx, e2eCtx.Environment.BootstrapClusterProxy, testNamespace.Name, clusterName, e2eCtx)

			shared.Logf("Graceful failure handling verified successfully")
		})
	})
})
