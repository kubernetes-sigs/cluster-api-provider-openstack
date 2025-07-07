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
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ctrl "sigs.k8s.io/controller-runtime"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/test/e2e/shared"
)

const specName = "hcp"

var e2eCtx *shared.E2EContext

func init() {
	e2eCtx = shared.NewE2EContext()
	shared.CreateDefaultFlags(e2eCtx)
}

func TestHCP(t *testing.T) {
	RegisterFailHandler(Fail)
	ctrl.SetLogger(GinkgoLogr)
	RunSpecs(t, "capo-hcp")
}

var _ = SynchronizedBeforeSuite(func(ctx context.Context) []byte {
	return shared.Node1BeforeSuite(ctx, e2eCtx)
}, func(data []byte) {
	shared.AllNodesBeforeSuite(e2eCtx, data)
})

var _ = SynchronizedAfterSuite(func() {
	shared.AllNodesAfterSuite(e2eCtx)
}, func(ctx context.Context) {
	shared.Node1AfterSuite(ctx, e2eCtx)
})

var _ = Describe("Hosted Control Plane tests", func() {
	var (
		namespace                *corev1.Namespace
		managementClusterName    string
		workloadClusterName      string
		clusterResources         *clusterctl.ApplyClusterTemplateAndWaitResult
		workloadClusterResources *clusterctl.ApplyClusterTemplateAndWaitResult
	)

	BeforeEach(func(ctx context.Context) {
		Expect(e2eCtx.Environment.BootstrapClusterProxy).ToNot(BeNil(), "Invalid argument. BootstrapClusterProxy can't be nil")
		// Setup a Namespace where to host objects for this spec and create a watcher for the namespace events.
		namespace = shared.SetupSpecNamespace(ctx, specName, e2eCtx)
		clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)
		workloadClusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)
		Expect(e2eCtx.E2EConfig).ToNot(BeNil(), "Invalid argument. e2eConfig can't be nil when calling %s spec", specName)
		Expect(e2eCtx.E2EConfig.Variables).To(HaveKey(shared.KubernetesVersion))

		managementClusterName = fmt.Sprintf("mgmt-%s", namespace.Name)
		workloadClusterName = fmt.Sprintf("hcp-%s", namespace.Name)
	})

	Describe("Management cluster setup and HCP workload cluster", func() {
		It("should create a management cluster, install Kamaji, and create HCP workload cluster", func(ctx context.Context) {
			By("Creating management cluster (normal CAPO cluster)")
			shared.Logf("Creating management cluster: %s", managementClusterName)

			configCluster := clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(e2eCtx.Settings.ArtifactFolder, "clusters", e2eCtx.Environment.BootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     e2eCtx.Environment.ClusterctlConfigPath,
				KubeconfigPath:           e2eCtx.Environment.BootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   "openstack",
				Flavor:                   shared.FlavorDefault,
				Namespace:                namespace.Name,
				ClusterName:              managementClusterName,
				KubernetesVersion:        e2eCtx.E2EConfig.Variables[shared.KubernetesVersion],
				ControlPlaneMachineCount: ptr.To(int64(1)),
				WorkerMachineCount:       ptr.To(int64(1)),
			}

			clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
				ClusterProxy:                 e2eCtx.Environment.BootstrapClusterProxy,
				ConfigCluster:                configCluster,
				WaitForClusterIntervals:      e2eCtx.E2EConfig.GetIntervals(specName, "wait-cluster"),
				WaitForControlPlaneIntervals: e2eCtx.E2EConfig.GetIntervals(specName, "wait-control-plane"),
				WaitForMachineDeployments:    e2eCtx.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
			}, clusterResources)

			By("Getting management cluster kubeconfig")
			workloadCluster := e2eCtx.Environment.BootstrapClusterProxy.GetWorkloadCluster(ctx, namespace.Name, managementClusterName)
			managementKubeconfig := workloadCluster.GetKubeconfigPath()

			By("Installing Kamaji v1.0.0 on management cluster using shell script")
			shared.Logf("Installing Kamaji via hack/install-kamaji.sh on management cluster: %s", managementClusterName)

			installCmd := exec.Command("../../../../hack/install-kamaji.sh", managementKubeconfig)
			output, err := installCmd.CombinedOutput()
			shared.Logf("Kamaji installation script output:\n%s", string(output))
			Expect(err).ToNot(HaveOccurred(), "Failed to install Kamaji using script")

			By("Waiting for Kamaji to be ready")
			// The script waits for pods to be ready, but a small extra delay can prevent race conditions
			// where the webhook is not yet fully available for the TenantControlPlane.
			time.Sleep(10 * time.Second)

			By("Creating workload cluster with Kamaji control plane")
			shared.Logf("Creating HCP workload cluster: %s", workloadClusterName)

			// Create workload cluster using the HCP template
			workloadConfigCluster := clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(e2eCtx.Settings.ArtifactFolder, "clusters", e2eCtx.Environment.BootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     e2eCtx.Environment.ClusterctlConfigPath,
				KubeconfigPath:           managementKubeconfig, // Use management cluster kubeconfig
				InfrastructureProvider:   "openstack",
				Flavor:                   shared.FlavorHCP, // Use HCP flavor
				Namespace:                namespace.Name,
				ClusterName:              workloadClusterName,
				KubernetesVersion:        e2eCtx.E2EConfig.Variables[shared.KubernetesVersion],
				ControlPlaneMachineCount: ptr.To(int64(1)), // Kamaji manages the control plane
				WorkerMachineCount:       ptr.To(int64(2)),
			}

			// Apply the HCP cluster template to the management cluster
			clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
				ClusterProxy:                 workloadCluster, // Apply to management cluster
				ConfigCluster:                workloadConfigCluster,
				WaitForClusterIntervals:      e2eCtx.E2EConfig.GetIntervals(specName, "wait-cluster"),
				WaitForControlPlaneIntervals: e2eCtx.E2EConfig.GetIntervals(specName, "wait-control-plane"),
				WaitForMachineDeployments:    e2eCtx.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
			}, workloadClusterResources)

			By("Validating network and security group configuration in HCP context")
			shared.Logf("Validating HCP cluster functionality")

			// Get the workload cluster kubeconfig from the management cluster
			workloadKubeconfig := workloadCluster.GetWorkloadCluster(ctx, namespace.Name, workloadClusterName).GetKubeconfigPath()

			// Test specific scenarios from hcp-2380 branch fixes
			By("Testing network configuration edge cases")
			workloadClusterProxy := workloadCluster.GetWorkloadCluster(ctx, namespace.Name, workloadClusterName)

			ValidateNetworkConfiguration(ctx, NetworkValidationInput{
				WorkloadClusterProxy: workloadClusterProxy,
				Namespace:            namespace.Name,
				ClusterName:          workloadClusterName,
				E2EContext:           e2eCtx,
			})

			By("Testing terminal error for missing network")
			// This test intentionally creates a machine that should fail to validate the terminal error handling
			func() {
				// Get the OpenStackCluster and patch its status to remove the network
				openStackCluster := &infrav1.OpenStackCluster{}
				err := workloadClusterProxy.GetClient().Get(ctx, types.NamespacedName{Name: workloadClusterName, Namespace: namespace.Name}, openStackCluster)
				Expect(err).ToNot(HaveOccurred())

				patch := client.MergeFrom(openStackCluster.DeepCopy())
				openStackCluster.Status.Network = nil
				Expect(workloadClusterProxy.GetClient().Status().Patch(ctx, openStackCluster, patch)).To(Succeed())

				// Create a machine without a network defined in its spec. This should fail because the cluster network is also gone.
				machineName := fmt.Sprintf("%s-terminal-test", workloadClusterName)
				openStackMachine := &infrav1.OpenStackMachine{
					ObjectMeta: metav1.ObjectMeta{Name: machineName, Namespace: namespace.Name},
					Spec: infrav1.OpenStackMachineSpec{
						Flavor:     e2eCtx.E2EConfig.Variables[shared.OpenstackNodeMachineFlavor],
						Image:      infrav1.ImageParam{Filter: &infrav1.ImageFilter{Name: &e2eCtx.E2EConfig.Variables[shared.OpenstackImageName]}},
						SSHKeyName: e2eCtx.E2EConfig.Variables[shared.OpenstackSSHKeyName],
					},
				}
				Expect(workloadClusterProxy.GetClient().Create(ctx, openStackMachine)).To(Succeed())

				// Assert that the machine gets the terminal error condition
				g := NewWithT(GinkgoT())
				g.Eventually(func() (bool, error) {
					err := workloadClusterProxy.GetClient().Get(ctx, client.ObjectKeyFromObject(openStackMachine), openStackMachine)
					if err != nil {
						return false, err
					}
					for _, condition := range openStackMachine.Status.Conditions {
						if condition.Type == clusterv1.ReadyCondition && condition.Status == corev1.ConditionFalse && condition.Severity == clusterv1.ConditionSeverityError && condition.Reason == infrav1.InvalidMachineSpecReason {
							shared.Logf("Found terminal condition with correct reason: %s", condition.Message)
							return true, nil
						}
					}
					return false, nil
				}, 10*time.Minute, 15*time.Second).Should(BeTrue(), "OpenStackMachine should have a terminal error condition for missing network")

				// Clean up the test machine
				Expect(workloadClusterProxy.GetClient().Delete(ctx, openStackMachine)).To(Succeed())
			}()

			By("Testing Konnectivity connectivity")
			ValidateKonectivityConnectivity(ctx, NetworkValidationInput{
				WorkloadClusterProxy: workloadClusterProxy,
				Namespace:            namespace.Name,
				ClusterName:          workloadClusterName,
			})

			shared.Logf("HCP test completed successfully!")
			shared.Logf("Management cluster: %s", managementClusterName)
			shared.Logf("Workload cluster: %s", workloadClusterName)
		})
	})

	AfterEach(func(ctx context.Context) {
		shared.Logf("Cleaning up HCP test resources")

		if workloadClusterResources.Cluster != nil {
			shared.Logf("Attempting to collect logs for workload cluster %q in namespace %q", workloadClusterResources.Cluster.Name, namespace.Name)
		}

		if clusterResources.Cluster != nil {
			shared.Logf("Attempting to collect logs for management cluster %q in namespace %q", clusterResources.Cluster.Name, namespace.Name)
			e2eCtx.Environment.BootstrapClusterProxy.CollectWorkloadClusterLogs(ctx, namespace.Name, clusterResources.Cluster.Name, filepath.Join(e2eCtx.Settings.ArtifactFolder, "clusters", e2eCtx.Environment.BootstrapClusterProxy.GetName(), namespace.Name))
		}

		// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
		shared.DumpSpecResourcesAndCleanup(ctx, specName, namespace, e2eCtx)
	})
})
