//go:build e2e
// +build e2e

/*
Copyright 2021 The Kubernetes Authors.

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

package e2e

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumetypes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
	"sigs.k8s.io/cluster-api/controllers/noderefutil"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha5"
	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
	"sigs.k8s.io/cluster-api-provider-openstack/test/e2e/shared"
)

const specName = "e2e"

var _ = Describe("e2e tests", func() {
	var (
		namespace *corev1.Namespace
		ctx       context.Context

		// Cleanup functions which cannot run until after the cluster has been deleted
		postClusterCleanup []func()
	)

	BeforeEach(func() {
		Expect(e2eCtx.Environment.BootstrapClusterProxy).ToNot(BeNil(), "Invalid argument. BootstrapClusterProxy can't be nil")
		ctx = context.TODO()
		// Setup a Namespace where to host objects for this spec and create a watcher for the namespace events.
		namespace = shared.SetupSpecNamespace(ctx, specName, e2eCtx)
		Expect(e2eCtx.E2EConfig).ToNot(BeNil(), "Invalid argument. e2eConfig can't be nil when calling %s spec", specName)
		Expect(e2eCtx.E2EConfig.Variables).To(HaveKey(shared.KubernetesVersion))
		shared.SetEnvVar("USE_CI_ARTIFACTS", "true", false)
		postClusterCleanup = nil
	})

	Describe("Workload cluster (default)", func() {
		It("It should be creatable and deletable", func() {
			shared.Byf("Creating a cluster")
			clusterName := fmt.Sprintf("cluster-%s", namespace.Name)
			configCluster := defaultConfigCluster(clusterName, namespace.Name)
			configCluster.ControlPlaneMachineCount = pointer.Int64Ptr(3)
			configCluster.WorkerMachineCount = pointer.Int64Ptr(1)
			configCluster.Flavor = shared.FlavorDefault
			md := createCluster(ctx, configCluster)

			workerMachines := framework.GetMachinesByMachineDeployments(ctx, framework.GetMachinesByMachineDeploymentsInput{
				Lister:            e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
				ClusterName:       clusterName,
				Namespace:         namespace.Name,
				MachineDeployment: *md[0],
			})
			controlPlaneMachines := framework.GetControlPlaneMachinesByCluster(ctx, framework.GetControlPlaneMachinesByClusterInput{
				Lister:      e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
				ClusterName: clusterName,
				Namespace:   namespace.Name,
			})
			Expect(workerMachines).To(HaveLen(1))
			Expect(controlPlaneMachines).To(HaveLen(3))

			// Tag: clusterName is declared on OpenStackCluster and gets propagated to all machines
			// except the bastion host
			allServers, err := shared.DumpOpenStackServers(e2eCtx, servers.ListOpts{Tags: clusterName})
			Expect(err).NotTo(HaveOccurred())
			Expect(allServers).To(HaveLen(4))

			// When listing servers with multiple tags, nova api requires a single, comma-separated string
			// with all the tags
			controlPlaneTags := fmt.Sprintf("%s,%s", clusterName, "control-plane")
			controlPlaneServers, err := shared.DumpOpenStackServers(e2eCtx, servers.ListOpts{Tags: controlPlaneTags})
			Expect(err).NotTo(HaveOccurred())
			Expect(controlPlaneServers).To(HaveLen(3))

			machineTags := fmt.Sprintf("%s,%s", clusterName, "machine")
			machineServers, err := shared.DumpOpenStackServers(e2eCtx, servers.ListOpts{Tags: machineTags})
			Expect(err).NotTo(HaveOccurred())
			Expect(machineServers).To(HaveLen(1))

			networksList, err := shared.DumpOpenStackNetworks(e2eCtx, networks.ListOpts{Tags: clusterName})
			Expect(err).NotTo(HaveOccurred())
			Expect(networksList).To(HaveLen(1))

			subnetsList, err := shared.DumpOpenStackSubnets(e2eCtx, subnets.ListOpts{Tags: clusterName})
			Expect(err).NotTo(HaveOccurred())
			Expect(subnetsList).To(HaveLen(1))

			routersList, err := shared.DumpOpenStackRouters(e2eCtx, routers.ListOpts{Tags: clusterName})
			Expect(err).NotTo(HaveOccurred())
			Expect(routersList).To(HaveLen(1))

			securityGroupsList, err := shared.DumpOpenStackSecurityGroups(e2eCtx, groups.ListOpts{Tags: clusterName})
			Expect(err).NotTo(HaveOccurred())
			Expect(securityGroupsList).To(HaveLen(3))
		})
	})

	Describe("Workload cluster (external cloud provider)", func() {
		It("It should be creatable and deletable", func() {
			shared.Byf("Creating a cluster")
			clusterName := fmt.Sprintf("cluster-%s", namespace.Name)
			configCluster := defaultConfigCluster(clusterName, namespace.Name)
			configCluster.ControlPlaneMachineCount = pointer.Int64Ptr(1)
			configCluster.WorkerMachineCount = pointer.Int64Ptr(1)
			configCluster.Flavor = shared.FlavorExternalCloudProvider
			md := createCluster(ctx, configCluster)

			workerMachines := framework.GetMachinesByMachineDeployments(ctx, framework.GetMachinesByMachineDeploymentsInput{
				Lister:            e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
				ClusterName:       clusterName,
				Namespace:         namespace.Name,
				MachineDeployment: *md[0],
			})
			controlPlaneMachines := framework.GetControlPlaneMachinesByCluster(ctx, framework.GetControlPlaneMachinesByClusterInput{
				Lister:      e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
				ClusterName: clusterName,
				Namespace:   namespace.Name,
			})
			Expect(workerMachines).To(HaveLen(1))
			Expect(controlPlaneMachines).To(HaveLen(1))

			shared.Byf("Waiting for worker nodes to be in Running phase")
			statusChecks := []framework.MachineStatusCheck{framework.MachinePhaseCheck(string(clusterv1.MachinePhaseRunning))}
			machineStatusInput := framework.WaitForMachineStatusCheckInput{
				Getter:       e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
				Machine:      &workerMachines[0],
				StatusChecks: statusChecks,
			}
			framework.WaitForMachineStatusCheck(ctx, machineStatusInput, e2eCtx.E2EConfig.GetIntervals(specName, "wait-machine-status")...)

			workloadCluster := e2eCtx.Environment.BootstrapClusterProxy.GetWorkloadCluster(ctx, namespace.Name, clusterName)

			waitForDaemonSetRunning(ctx, workloadCluster.GetClient(), "kube-system", "openstack-cloud-controller-manager")

			waitForNodesReadyWithoutCCMTaint(ctx, workloadCluster.GetClient(), 2)
		})
	})

	Describe("Workload cluster (without lb)", func() {
		It("Should create port(s) with custom options", func() {
			shared.Byf("Creating a cluster")
			clusterName := fmt.Sprintf("cluster-%s", namespace.Name)
			configCluster := defaultConfigCluster(clusterName, namespace.Name)
			configCluster.ControlPlaneMachineCount = pointer.Int64Ptr(1)
			configCluster.WorkerMachineCount = pointer.Int64Ptr(1)
			configCluster.Flavor = shared.FlavorWithoutLB
			md := createCluster(ctx, configCluster)

			workerMachines := framework.GetMachinesByMachineDeployments(ctx, framework.GetMachinesByMachineDeploymentsInput{
				Lister:            e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
				ClusterName:       clusterName,
				Namespace:         namespace.Name,
				MachineDeployment: *md[0],
			})
			controlPlaneMachines := framework.GetControlPlaneMachinesByCluster(ctx, framework.GetControlPlaneMachinesByClusterInput{
				Lister:      e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
				ClusterName: clusterName,
				Namespace:   namespace.Name,
			})
			Expect(workerMachines).To(HaveLen(1))
			Expect(controlPlaneMachines).To(HaveLen(1))

			shared.Byf("Creating MachineDeployment with custom port options")
			md3Name := clusterName + "-md-3"

			customPortOptions := &[]infrav1.PortOpts{
				{
					Description: "primary",
				},
				{
					Description: "trunked",
					Trunk:       pointer.Bool(true),
				},
			}

			testTag := utilrand.String(6)
			machineTags := []string{testTag}

			// Note that as the bootstrap config does not have cloud.conf, the node will not be added to the cluster.
			// We still expect the port for the machine to be created.
			framework.CreateMachineDeployment(ctx, framework.CreateMachineDeploymentInput{
				Creator:                 e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
				MachineDeployment:       makeMachineDeployment(namespace.Name, md3Name, clusterName, "", 1),
				BootstrapConfigTemplate: makeJoinBootstrapConfigTemplate(namespace.Name, md3Name),
				InfraMachineTemplate:    makeOpenStackMachineTemplateWithPortOptions(namespace.Name, clusterName, md3Name, customPortOptions, machineTags),
			})

			shared.Byf("Waiting for custom port to be created")
			var plist []ports.Port
			var err error
			Eventually(func() int {
				plist, err = shared.DumpOpenStackPorts(e2eCtx, ports.ListOpts{Description: "primary", Tags: testTag})
				Expect(err).To(BeNil())
				return len(plist)
			}, e2eCtx.E2EConfig.GetIntervals(specName, "wait-worker-nodes")...).Should(Equal(1))

			port := plist[0]
			Expect(port.Description).To(Equal("primary"))
			Expect(port.Tags).To(ContainElement(testTag))

			// assert trunked port is created.
			Eventually(func() int {
				plist, err = shared.DumpOpenStackPorts(e2eCtx, ports.ListOpts{Description: "trunked", Tags: testTag})
				Expect(err).To(BeNil())
				return len(plist)
			}, e2eCtx.E2EConfig.GetIntervals(specName, "wait-worker-nodes")...).Should(Equal(1))
			port = plist[0]
			Expect(port.Description).To(Equal("trunked"))
			Expect(port.Tags).To(ContainElement(testTag))

			// assert trunk data.
			var trunk *trunks.Trunk
			Eventually(func() int {
				trunk, err = shared.DumpOpenStackTrunks(e2eCtx, port.ID)
				Expect(err).To(BeNil())
				Expect(trunk).NotTo(BeNil())
				return 1
			}, e2eCtx.E2EConfig.GetIntervals(specName, "wait-worker-nodes")...).Should(Equal(1))
			Expect(trunk.PortID).To(Equal(port.ID))
		})
	})

	Describe("Workload cluster (multiple attached networks)", func() {
		var (
			clusterName   string
			configCluster clusterctl.ConfigClusterInput
			md            []*clusterv1.MachineDeployment

			extraNet1, extraNet2 *networks.Network
		)

		BeforeEach(func() {
			var err error

			// Create 2 additional networks to be attached to all cluster nodes
			// We can't clean up these networks in a corresponding AfterEach because they will still be in use by the cluster.
			// Instead we clean them up after the cluster has been deleted.

			shared.Byf("Creating additional networks")

			extraNet1, err = shared.CreateOpenStackNetwork(e2eCtx, fmt.Sprintf("%s-extraNet1", namespace.Name), "10.14.0.0/24")
			Expect(err).NotTo(HaveOccurred())
			postClusterCleanup = append(postClusterCleanup, func() {
				shared.Byf("Deleting additional network %s", extraNet1.Name)
				err := shared.DeleteOpenStackNetwork(e2eCtx, extraNet1.ID)
				Expect(err).NotTo(HaveOccurred())
			})

			extraNet2, err = shared.CreateOpenStackNetwork(e2eCtx, fmt.Sprintf("%s-extraNet2", namespace.Name), "10.14.1.0/24")
			Expect(err).NotTo(HaveOccurred())
			postClusterCleanup = append(postClusterCleanup, func() {
				shared.Byf("Deleting additional network %s", extraNet2.Name)
				err := shared.DeleteOpenStackNetwork(e2eCtx, extraNet2.ID)
				Expect(err).NotTo(HaveOccurred())
			})

			os.Setenv("CLUSTER_EXTRA_NET_1", extraNet1.ID)
			os.Setenv("CLUSTER_EXTRA_NET_2", extraNet2.ID)

			shared.Byf("Creating a cluster")
			clusterName = fmt.Sprintf("cluster-%s", namespace.Name)
			configCluster = defaultConfigCluster(clusterName, namespace.Name)
			configCluster.ControlPlaneMachineCount = pointer.Int64Ptr(1)
			configCluster.WorkerMachineCount = pointer.Int64Ptr(1)
			configCluster.Flavor = shared.FlavorMultiNetwork
			md = createCluster(ctx, configCluster)
		})

		It("should attach all machines to multiple networks", func() {
			workerMachines := framework.GetMachinesByMachineDeployments(ctx, framework.GetMachinesByMachineDeploymentsInput{
				Lister:            e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
				ClusterName:       clusterName,
				Namespace:         namespace.Name,
				MachineDeployment: *md[0],
			})
			controlPlaneMachines := framework.GetControlPlaneMachinesByCluster(ctx, framework.GetControlPlaneMachinesByClusterInput{
				Lister:      e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
				ClusterName: clusterName,
				Namespace:   namespace.Name,
			})
			Expect(workerMachines).To(HaveLen(int(*configCluster.WorkerMachineCount)))
			Expect(controlPlaneMachines).To(HaveLen(int(*configCluster.ControlPlaneMachineCount)))

			openStackCluster, err := shared.ClusterForSpec(ctx, e2eCtx, namespace)
			Expect(err).NotTo(HaveOccurred())

			var allMachines []clusterv1.Machine
			allMachines = append(allMachines, controlPlaneMachines...)
			allMachines = append(allMachines, workerMachines...)

			// We expect each machine to have 3 ports in each of these 3 networks with the given description.
			expectedPorts := map[string]string{
				openStackCluster.Status.Network.ID: "primary",
				extraNet1.ID:                       "Extra Network 1",
				extraNet2.ID:                       "Extra Network 2",
			}

			for i := range allMachines {
				machine := &allMachines[i]
				shared.Byf("Checking ports for machine %s", machine.Name)
				instanceID := getInstanceIDForMachine(machine)

				shared.Byf("Fetching ports for instance %s", instanceID)
				ports, err := shared.DumpOpenStackPorts(e2eCtx, ports.ListOpts{
					DeviceID: instanceID,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(ports).To(HaveLen(len(expectedPorts)))

				var seenNetworks []string
				var seenAddresses clusterv1.MachineAddresses
				for j := range ports {
					port := &ports[j]

					// Check that the port has an expected network ID and description
					Expect(expectedPorts).To(HaveKeyWithValue(port.NetworkID, port.Description))

					// We don't expect to see another port with this network on this machine
					Expect(seenNetworks).ToNot(ContainElement(port.NetworkID))
					seenNetworks = append(seenNetworks, port.NetworkID)

					for k := range port.FixedIPs {
						seenAddresses = append(seenAddresses, clusterv1.MachineAddress{
							Type:    clusterv1.MachineInternalIP,
							Address: port.FixedIPs[k].IPAddress,
						})
					}
				}

				// All IP addresses on all ports should be reported in Addresses
				Expect(machine.Status.Addresses).To(ContainElements(seenAddresses))
			}
		})
	})

	Describe("MachineDeployment misconfigurations", func() {
		It("Should fail to create MachineDeployment with invalid subnet or invalid availability zone", func() {
			shared.Byf("Creating a cluster")
			clusterName := fmt.Sprintf("cluster-%s", namespace.Name)
			configCluster := defaultConfigCluster(clusterName, namespace.Name)
			configCluster.ControlPlaneMachineCount = pointer.Int64Ptr(1)
			configCluster.WorkerMachineCount = pointer.Int64Ptr(0)
			configCluster.Flavor = shared.FlavorDefault
			_ = createCluster(ctx, configCluster)

			shared.Byf("Creating Machine Deployment with invalid subnet id")
			md1Name := clusterName + "-md-1"
			framework.CreateMachineDeployment(ctx, framework.CreateMachineDeploymentInput{
				Creator:                 e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
				MachineDeployment:       makeMachineDeployment(namespace.Name, md1Name, clusterName, "", 1),
				BootstrapConfigTemplate: makeJoinBootstrapConfigTemplate(namespace.Name, md1Name),
				InfraMachineTemplate:    makeOpenStackMachineTemplate(namespace.Name, clusterName, md1Name, "invalid-subnet"),
			})

			shared.Byf("Looking for failure event to be reported")
			Eventually(func() bool {
				eventList := getEvents(namespace.Name)
				subnetError := "Failed to create server: no ports with fixed IPs found on Subnet \"invalid-subnet\""
				return isErrorEventExists(namespace.Name, md1Name, "FailedCreateServer", subnetError, eventList)
			}, e2eCtx.E2EConfig.GetIntervals(specName, "wait-worker-nodes")...).Should(BeTrue())

			shared.Byf("Creating Machine Deployment in an invalid Availability Zone")
			md2Name := clusterName + "-md-2"
			framework.CreateMachineDeployment(ctx, framework.CreateMachineDeploymentInput{
				Creator:                 e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
				MachineDeployment:       makeMachineDeployment(namespace.Name, md2Name, clusterName, "invalid-az", 1),
				BootstrapConfigTemplate: makeJoinBootstrapConfigTemplate(namespace.Name, md2Name),
				InfraMachineTemplate:    makeOpenStackMachineTemplate(namespace.Name, clusterName, md2Name, ""),
			})

			shared.Byf("Looking for failure event to be reported")
			Eventually(func() bool {
				eventList := getEvents(namespace.Name)
				azError := "The requested availability zone is not available"
				return isErrorEventExists(namespace.Name, md2Name, "FailedCreateServer", azError, eventList)
			}, e2eCtx.E2EConfig.GetIntervals(specName, "wait-worker-nodes")...).Should(BeTrue())
		})
	})

	Describe("Workload cluster (multi-AZ)", func() {
		var (
			clusterName                     string
			md                              []*clusterv1.MachineDeployment
			failureDomain, failureDomainAlt string
			volumeTypeAlt                   string
			cluster                         *infrav1.OpenStackCluster
		)

		BeforeEach(func() {
			failureDomain = e2eCtx.E2EConfig.GetVariable(shared.OpenStackFailureDomain)
			failureDomainAlt = e2eCtx.E2EConfig.GetVariable(shared.OpenStackFailureDomainAlt)
			volumeTypeAlt = e2eCtx.E2EConfig.GetVariable(shared.OpenStackVolumeTypeAlt)

			// We create the second compute host asynchronously, so
			// we need to ensure the alternate failure domain exists
			// before running these tests.
			//
			// For efficiency we run the multi-AZ tests late in the
			// test suite. In practise this should mean that the
			// second compute is already up by the time we get here,
			// and we don't have to wait.
			Eventually(func() []string {
				shared.Byf("Waiting for the alternate AZ '%s' to be created", failureDomainAlt)
				return shared.GetComputeAvailabilityZones(e2eCtx)
			}, e2eCtx.E2EConfig.GetIntervals(specName, "wait-alt-az")...).Should(ContainElement(failureDomainAlt))

			shared.Byf("Creating a cluster")
			clusterName = fmt.Sprintf("cluster-%s", namespace.Name)
			configCluster := defaultConfigCluster(clusterName, namespace.Name)
			configCluster.ControlPlaneMachineCount = pointer.Int64Ptr(3)
			configCluster.WorkerMachineCount = pointer.Int64Ptr(2)
			configCluster.Flavor = shared.FlavorMultiAZ
			md = createCluster(ctx, configCluster)

			var err error
			cluster, err = shared.ClusterForSpec(ctx, e2eCtx, namespace)
			Expect(err).NotTo(HaveOccurred())
		})

		It("It should be creatable and deletable", func() {
			workerMachines := framework.GetMachinesByMachineDeployments(ctx, framework.GetMachinesByMachineDeploymentsInput{
				Lister:            e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
				ClusterName:       clusterName,
				Namespace:         namespace.Name,
				MachineDeployment: *md[0],
			})
			controlPlaneMachines := framework.GetControlPlaneMachinesByCluster(ctx, framework.GetControlPlaneMachinesByClusterInput{
				Lister:      e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
				ClusterName: clusterName,
				Namespace:   namespace.Name,
			})
			Expect(controlPlaneMachines).To(
				HaveLen(3),
				fmt.Sprintf("Cluster %s does not have the expected number of control plane machines", cluster.Name))
			Expect(workerMachines).To(
				HaveLen(2),
				fmt.Sprintf("Cluster %s does not have the expected number of worker machines", cluster.Name))

			getAZsForMachines := func(machines []clusterv1.Machine) []string {
				azs := make(map[string]struct{})
				for _, machine := range machines {
					failureDomain := machine.Spec.FailureDomain
					if failureDomain == nil {
						continue
					}
					azs[*failureDomain] = struct{}{}
				}

				azSlice := make([]string, 0, len(azs))
				for az := range azs {
					azSlice = append(azSlice, az)
				}

				return azSlice
			}

			// The control plane should be spread across both AZs
			controlPlaneAZs := getAZsForMachines(controlPlaneMachines)
			Expect(controlPlaneAZs).To(
				ConsistOf(failureDomain, failureDomainAlt),
				fmt.Sprintf("Cluster %s: control plane machines were not scheduled in the expected AZs", cluster.Name))

			// All workers should be in the alt AZ
			workerAZs := getAZsForMachines(workerMachines)
			Expect(workerAZs).To(
				ConsistOf(failureDomainAlt),
				fmt.Sprintf("Cluster %s: worker machines were not scheduled in the expected AZ", cluster.Name))

			// Check that all machines were actually scheduled in the correct AZ
			var allMachines []clusterv1.Machine
			allMachines = append(allMachines, controlPlaneMachines...)
			allMachines = append(allMachines, workerMachines...)

			allServers, err := shared.GetOpenStackServers(e2eCtx, cluster)
			Expect(err).NotTo(HaveOccurred())

			allServerNames := make([]string, 0, len(allServers))
			for name := range allServers {
				allServerNames = append(allServerNames, name)
			}

			bootVolumes := make(map[string]*volumes.Volume)

			for _, machine := range allMachines {
				// The output of a HaveKey() failure against
				// allServers is too long and overflows
				openstackMachineName := machine.Spec.InfrastructureRef.Name
				Expect(allServerNames).To(
					ContainElement(openstackMachineName),
					fmt.Sprintf("Cluster %s: did not find a server for machine %s", cluster.Name, openstackMachineName))

				server := allServers[openstackMachineName]
				Expect(server.AvailabilityZone).To(
					Equal(*machine.Spec.FailureDomain),
					fmt.Sprintf("Server %s was not scheduled in the correct AZ", machine.Name))

				// Check that all machines have the expected boot volume
				volumes := server.AttachedVolumes
				Expect(volumes).To(HaveLen(1))

				bootVolume, err := shared.GetOpenStackVolume(e2eCtx, volumes[0].ID)
				Expect(err).NotTo(HaveOccurred(), "failed to get OpenStack volume %s for machine %s", volumes[0].ID, machine.Name)
				bootVolumes[machine.Name] = bootVolume

				Expect(*bootVolume).To(MatchFields(IgnoreExtras, Fields{
					"Name":     Equal(fmt.Sprintf("%s-root", server.Name)),
					"Size":     Equal(15),
					"Bootable": Equal("true"), // This is genuinely a string, not a bool
				}), "Boot volume %s for machine %s not as expected", bootVolume.ID, machine.Name)
			}

			// Expect all control plane machines to have a root volume in the same AZ as the machine, and the default volume type
			for _, machine := range controlPlaneMachines {
				bootVolume := bootVolumes[machine.Name]
				Expect(bootVolume.AvailabilityZone).To(Equal(*machine.Spec.FailureDomain))
				Expect(bootVolume.VolumeType).NotTo(Equal(volumeTypeAlt))
			}

			// Expect all worker machines to have a root volume in the primary AZ, and the test volume type
			for _, machine := range workerMachines {
				bootVolume := bootVolumes[machine.Name]
				Expect(bootVolume.AvailabilityZone).To(Equal(failureDomain))
				Expect(bootVolume.VolumeType).To(Equal(volumeTypeAlt))
			}
		})
	})

	AfterEach(func() {
		shared.SetEnvVar("USE_CI_ARTIFACTS", "false", false)
		// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
		shared.DumpSpecResourcesAndCleanup(ctx, specName, namespace, e2eCtx)

		// Cleanup resources which can't be cleaned up until the cluster has been deleted
		for _, cleanup := range postClusterCleanup {
			cleanup()
		}
	})
})

func createCluster(ctx context.Context, configCluster clusterctl.ConfigClusterInput) []*clusterv1.MachineDeployment {
	result := &clusterctl.ApplyClusterTemplateAndWaitResult{}
	clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
		ClusterProxy:                 e2eCtx.Environment.BootstrapClusterProxy,
		ConfigCluster:                configCluster,
		WaitForClusterIntervals:      e2eCtx.E2EConfig.GetIntervals(specName, "wait-cluster"),
		WaitForControlPlaneIntervals: e2eCtx.E2EConfig.GetIntervals(specName, "wait-control-plane"),
		WaitForMachineDeployments:    e2eCtx.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
	}, result)

	return result.MachineDeployments
}

func defaultConfigCluster(clusterName, namespace string) clusterctl.ConfigClusterInput {
	return clusterctl.ConfigClusterInput{
		LogFolder:              filepath.Join(e2eCtx.Settings.ArtifactFolder, "clusters", e2eCtx.Environment.BootstrapClusterProxy.GetName()),
		ClusterctlConfigPath:   e2eCtx.Environment.ClusterctlConfigPath,
		KubeconfigPath:         e2eCtx.Environment.BootstrapClusterProxy.GetKubeconfigPath(),
		InfrastructureProvider: clusterctl.DefaultInfrastructureProvider,
		Namespace:              namespace,
		ClusterName:            clusterName,
		KubernetesVersion:      e2eCtx.E2EConfig.GetVariable(shared.KubernetesVersion),
	}
}

func getEvents(namespace string) *corev1.EventList {
	eventsList := &corev1.EventList{}
	if err := e2eCtx.Environment.BootstrapClusterProxy.GetClient().List(context.TODO(), eventsList, crclient.InNamespace(namespace), crclient.MatchingLabels{}); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Got error while fetching events of namespace: %s, %s \n", namespace, err.Error())
	}

	return eventsList
}

func getInstanceIDForMachine(machine *clusterv1.Machine) string {
	providerID := machine.Spec.ProviderID
	Expect(providerID).NotTo(BeNil())

	providerIDSplit := strings.SplitN(*providerID, ":///", 2)
	Expect(providerIDSplit[0]).To(Equal("openstack"))
	return providerIDSplit[1]
}

func isErrorEventExists(namespace, machineDeploymentName, eventReason, errorMsg string, eList *corev1.EventList) bool {
	ctrlClient := e2eCtx.Environment.BootstrapClusterProxy.GetClient()
	machineDeployment := &clusterv1.MachineDeployment{}
	if err := ctrlClient.Get(context.TODO(), apimachinerytypes.NamespacedName{Namespace: namespace, Name: machineDeploymentName}, machineDeployment); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Got error while getting machinedeployment %s \n", machineDeploymentName)
		return false
	}

	selector, err := metav1.LabelSelectorAsMap(&machineDeployment.Spec.Selector)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Got error while reading lables of machinedeployment: %s, %s \n", machineDeploymentName, err.Error())
		return false
	}

	openStackMachineList := &infrav1.OpenStackMachineList{}
	if err := ctrlClient.List(context.TODO(), openStackMachineList, crclient.InNamespace(namespace), crclient.MatchingLabels(selector)); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Got error while getting openstackmachines of machinedeployment: %s, %s \n", machineDeploymentName, err.Error())
		return false
	}

	eventMachinesCnt := 0
	for _, openStackMachine := range openStackMachineList.Items {
		for _, event := range eList.Items {
			if strings.Contains(event.Name, openStackMachine.Name) && event.Reason == eventReason && strings.Contains(event.Message, errorMsg) {
				eventMachinesCnt++
				break
			}
		}
	}
	return len(openStackMachineList.Items) == eventMachinesCnt
}

func makeOpenStackMachineTemplate(namespace, clusterName, name string, subnetID string) *infrav1.OpenStackMachineTemplate {
	return &infrav1.OpenStackMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: infrav1.OpenStackMachineTemplateSpec{
			Template: infrav1.OpenStackMachineTemplateResource{
				Spec: infrav1.OpenStackMachineSpec{
					Flavor:     e2eCtx.E2EConfig.GetVariable(shared.OpenStackNodeMachineFlavor),
					Image:      e2eCtx.E2EConfig.GetVariable(shared.OpenStackImageName),
					SSHKeyName: shared.DefaultSSHKeyPairName,
					CloudName:  e2eCtx.E2EConfig.GetVariable(shared.OpenStackCloud),
					IdentityRef: &infrav1.OpenStackIdentityReference{
						Kind: "Secret",
						Name: fmt.Sprintf("%s-cloud-config", clusterName),
					},
					Subnet: subnetID,
				},
			},
		},
	}
}

func makeOpenStackMachineTemplateWithPortOptions(namespace, clusterName, name string, portOpts *[]infrav1.PortOpts, machineTags []string) *infrav1.OpenStackMachineTemplate {
	return &infrav1.OpenStackMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: infrav1.OpenStackMachineTemplateSpec{
			Template: infrav1.OpenStackMachineTemplateResource{
				Spec: infrav1.OpenStackMachineSpec{
					Flavor:     e2eCtx.E2EConfig.GetVariable(shared.OpenStackNodeMachineFlavor),
					Image:      e2eCtx.E2EConfig.GetVariable(shared.OpenStackImageName),
					SSHKeyName: shared.DefaultSSHKeyPairName,
					CloudName:  e2eCtx.E2EConfig.GetVariable(shared.OpenStackCloud),
					IdentityRef: &infrav1.OpenStackIdentityReference{
						Kind: "Secret",
						Name: fmt.Sprintf("%s-cloud-config", clusterName),
					},
					Ports: *portOpts,
					Tags:  machineTags,
				},
			},
		},
	}
}

// makeJoinBootstrapConfigTemplate returns a KubeadmConfigTemplate which can be used
// to test different error cases. As we're missing e.g. the cloud provider conf it cannot
// be used to successfully add nodes to a cluster.
func makeJoinBootstrapConfigTemplate(namespace, name string) *bootstrapv1.KubeadmConfigTemplate {
	return &bootstrapv1.KubeadmConfigTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: bootstrapv1.KubeadmConfigTemplateSpec{
			Template: bootstrapv1.KubeadmConfigTemplateResource{
				Spec: bootstrapv1.KubeadmConfigSpec{
					JoinConfiguration: &bootstrapv1.JoinConfiguration{
						NodeRegistration: bootstrapv1.NodeRegistrationOptions{
							Name: "{{ local_hostname }}",
							KubeletExtraArgs: map[string]string{
								"cloud-config":   "/etc/kubernetes/cloud.conf",
								"cloud-provider": "openstack",
							},
						},
					},
				},
			},
		},
	}
}

func makeMachineDeployment(namespace, mdName, clusterName string, failureDomain string, replicas int32) *clusterv1.MachineDeployment {
	if failureDomain == "" {
		failureDomain = e2eCtx.E2EConfig.GetVariable(shared.OpenStackFailureDomain)
	}
	return &clusterv1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mdName,
			Namespace: namespace,
			Labels: map[string]string{
				"cluster.x-k8s.io/cluster-name": clusterName,
				"nodepool":                      mdName,
			},
		},
		Spec: clusterv1.MachineDeploymentSpec{
			Replicas: &replicas,
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster.x-k8s.io/cluster-name": clusterName,
					"nodepool":                      mdName,
				},
			},
			ClusterName: clusterName,
			Template: clusterv1.MachineTemplateSpec{
				ObjectMeta: clusterv1.ObjectMeta{
					Labels: map[string]string{
						"cluster.x-k8s.io/cluster-name": clusterName,
						"nodepool":                      mdName,
					},
				},
				Spec: clusterv1.MachineSpec{
					ClusterName:   clusterName,
					FailureDomain: &failureDomain,
					Bootstrap: clusterv1.Bootstrap{
						ConfigRef: &corev1.ObjectReference{
							Kind:       "KubeadmConfigTemplate",
							APIVersion: bootstrapv1.GroupVersion.String(),
							Name:       mdName,
							Namespace:  namespace,
						},
					},
					InfrastructureRef: corev1.ObjectReference{
						Kind:       "OpenStackMachineTemplate",
						APIVersion: infrav1.GroupVersion.String(),
						Name:       mdName,
						Namespace:  namespace,
					},
					Version: pointer.StringPtr(e2eCtx.E2EConfig.GetVariable(shared.KubernetesVersion)),
				},
			},
		},
	}
}

func waitForDaemonSetRunning(ctx context.Context, ctrlClient crclient.Client, namespace, name string) {
	shared.Byf("Ensuring DaemonSet %s is running", name)
	daemonSet := &appsv1.DaemonSet{}
	Eventually(
		func() (bool, error) {
			if err := ctrlClient.Get(ctx, apimachinerytypes.NamespacedName{Namespace: namespace, Name: name}, daemonSet); err != nil {
				return false, err
			}
			return daemonSet.Status.CurrentNumberScheduled == daemonSet.Status.NumberReady, nil
		}, 10*time.Minute, 30*time.Second,
	).Should(BeTrue())
}

func waitForNodesReadyWithoutCCMTaint(ctx context.Context, ctrlClient crclient.Client, nodeCount int) {
	shared.Byf("Waiting for the workload nodes to be ready")
	Eventually(func() (int, error) {
		nodeList := &corev1.NodeList{}
		if err := ctrlClient.List(ctx, nodeList); err != nil {
			return 0, err
		}
		if len(nodeList.Items) == 0 {
			return 0, errors.New("no nodes were found")
		}

		count := 0
		for _, node := range nodeList.Items {
			n := node
			if noderefutil.IsNodeReady(&n) && isCloudProviderInitialized(node.Spec.Taints) {
				count++
			}
		}
		return count, nil
	}, "10m", "10s").Should(Equal(nodeCount))
}

func isCloudProviderInitialized(taints []corev1.Taint) bool {
	for _, taint := range taints {
		if taint.Key == "node.cloudprovider.kubernetes.io/uninitialized" {
			return false
		}
	}
	return true
}

func createTestVolumeType(e2eCtx *shared.E2EContext) {
	providerClient, clientOpts, _, err := shared.GetAdminProviderClient(e2eCtx)
	Expect(err).NotTo(HaveOccurred())

	volumeClient, err := openstack.NewBlockStorageV3(providerClient, gophercloud.EndpointOpts{Region: clientOpts.RegionName})
	Expect(err).NotTo(HaveOccurred())

	shared.Byf("Creating test volume type")
	_, err = volumetypes.Create(volumeClient, &volumetypes.CreateOpts{
		Name:        e2eCtx.E2EConfig.GetVariable(shared.OpenStackVolumeTypeAlt),
		Description: "Test volume type",
		IsPublic:    pointer.BoolPtr(true),
		ExtraSpecs:  map[string]string{},
	}).Extract()
	if capoerrors.IsConflict(err) {
		shared.Byf("Volume type already exists. This may happen in development environments, but it is not expected in CI.")
		return
	}
	Expect(err).NotTo(HaveOccurred())
}

func cleanupTestVolumeType(e2eCtx *shared.E2EContext) {
	// We don't delete the volume type because it's unnecessarily complex when we're going to delete the environment anyway.
	// See https://bugs.launchpad.net/cinder/+bug/1823880 for a discussion of this same issue in an OpenStack tempest test.
}
