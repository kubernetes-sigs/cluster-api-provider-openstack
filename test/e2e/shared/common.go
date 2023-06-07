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

package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	expv1 "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha7"
)

func SetupSpecNamespace(ctx context.Context, specName string, e2eCtx *E2EContext) *corev1.Namespace {
	Logf("Creating a namespace for hosting the %q test spec", specName)
	namespace, cancelWatches := framework.CreateNamespaceAndWatchEvents(ctx, framework.CreateNamespaceAndWatchEventsInput{
		Creator:   e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
		ClientSet: e2eCtx.Environment.BootstrapClusterProxy.GetClientSet(),
		Name:      fmt.Sprintf("%s-%s", specName, util.RandomString(6)),
		LogFolder: filepath.Join(e2eCtx.Settings.ArtifactFolder, "clusters", e2eCtx.Environment.BootstrapClusterProxy.GetName()),
	})

	e2eCtx.Environment.Namespaces[namespace] = cancelWatches

	return namespace
}

func DumpSpecResourcesAndCleanup(ctx context.Context, specName string, namespace *corev1.Namespace, e2eCtx *E2EContext) {
	Logf("Running DumpSpecResourcesAndCleanup for namespace %q", namespace.Name)
	// Dump all Cluster API related resources to artifacts before deleting them.
	cancelWatches := e2eCtx.Environment.Namespaces[namespace]

	dumpAllResources := func(directory ...string) {
		dumpSpecResources(ctx, e2eCtx, namespace, directory...)
		dumpOpenStack(ctx, e2eCtx, e2eCtx.Environment.BootstrapClusterProxy.GetName(), directory...)

		Logf("Dumping all OpenStack server instances in the %q namespace", namespace.Name)
		dumpMachines(ctx, e2eCtx, namespace, directory...)
	}

	dumpAllResources()

	if !e2eCtx.Settings.SkipCleanup {
		func() {
			defer func() {
				r := recover()
				if r == nil {
					return
				}

				// If we fail to delete the cluster, dump all resources again to a different directory before propagating the failure
				dumpAllResources("deletion-failure")
				panic(r)
			}()
			framework.DeleteAllClustersAndWait(ctx, framework.DeleteAllClustersAndWaitInput{
				Client:    e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
				Namespace: namespace.Name,
			}, e2eCtx.E2EConfig.GetIntervals(specName, "wait-delete-cluster")...)
		}()

		Logf("Deleting namespace used for hosting the %q test spec", specName)
		framework.DeleteNamespace(ctx, framework.DeleteNamespaceInput{
			Deleter: e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
			Name:    namespace.Name,
		})
	}
	cancelWatches()
	delete(e2eCtx.Environment.Namespaces, namespace)
}

func dumpMachines(ctx context.Context, e2eCtx *E2EContext, namespace *corev1.Namespace, directory ...string) {
	cluster, err := ClusterForSpec(ctx, e2eCtx, namespace)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "cannot dump machines, couldn't get cluster in namespace %s: %v\n", namespace.Name, err)
		return
	}
	if cluster.Status.Bastion == nil || cluster.Status.Bastion.FloatingIP == "" {
		_, _ = fmt.Fprintln(GinkgoWriter, "cannot dump machines, cluster doesn't has a bastion host (yet) with a floating ip")
		return
	}
	machines, err := machinesForSpec(ctx, e2eCtx.Environment.BootstrapClusterProxy, namespace)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "cannot dump machines, could not get machines: %v\n", err)
		return
	}

	machineNames := sets.New[string]()
	for _, machine := range machines.Items {
		machineNames.Insert(machine.Name)
	}
	srvs, err := GetOpenStackServers(e2eCtx, cluster, machineNames)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "cannot dump machines, could not get servers from OpenStack: %v\n", err)
		return
	}
	for _, m := range machines.Items {
		srv, ok := srvs[m.Name]
		if !ok {
			continue
		}
		dumpMachine(ctx, e2eCtx, m, srv, cluster.Status.Bastion.FloatingIP, directory...)
	}
}

func ClusterForSpec(ctx context.Context, e2eCtx *E2EContext, namespace *corev1.Namespace) (*infrav1.OpenStackCluster, error) {
	lister := e2eCtx.Environment.BootstrapClusterProxy.GetClient()
	list := new(infrav1.OpenStackClusterList)
	if err := lister.List(ctx, list, client.InNamespace(namespace.GetName())); err != nil {
		return nil, fmt.Errorf("error listing cluster: %v", err)
	}
	if len(list.Items) != 1 {
		return nil, fmt.Errorf("error expected 1 cluster but got %d: %v", len(list.Items), list.Items)
	}
	return &list.Items[0], nil
}

func machinesForSpec(ctx context.Context, clusterProxy framework.ClusterProxy, namespace *corev1.Namespace) (*infrav1.OpenStackMachineList, error) {
	list := new(infrav1.OpenStackMachineList)
	if err := clusterProxy.GetClient().List(ctx, list, client.InNamespace(namespace.GetName())); err != nil {
		return nil, fmt.Errorf("error listing machines: %v", err)
	}
	return list, nil
}

func dumpMachine(ctx context.Context, e2eCtx *E2EContext, machine infrav1.OpenStackMachine, srv ServerExtWithIP, bastionIP string, directory ...string) {
	paths := append([]string{e2eCtx.Settings.ArtifactFolder, "clusters", e2eCtx.Environment.BootstrapClusterProxy.GetName()}, directory...)
	logPath := filepath.Join(paths...)
	machineLogBase := path.Join(logPath, "instances", machine.Namespace, machine.Name)
	metaLog := path.Join(machineLogBase, "instance.log")

	if err := os.MkdirAll(filepath.Dir(metaLog), 0o750); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "couldn't create directory %q for file: %s\n", metaLog, err)
	}

	f, err := os.OpenFile(metaLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "couldn't open file %q: %s\n", metaLog, err)
		return
	}
	defer f.Close()

	serverJSON, err := json.MarshalIndent(srv, "", "    ")
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "error marshalling server %v: %s", srv, err)
	}
	if err := os.WriteFile(path.Join(machineLogBase, "server.txt"), serverJSON, 0o600); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "error writing server JSON %s: %s", serverJSON, err)
	}

	srvUser := e2eCtx.E2EConfig.GetVariable(SSHUserMachine)

	_, _ = fmt.Fprintf(f, "instance found: %q\n", srv.ID)
	executeCommands(
		ctx,
		e2eCtx.Settings.ArtifactFolder,
		e2eCtx.Settings.Debug,
		filepath.Dir(f.Name()),
		srv.ip,
		bastionIP,
		srvUser,
		[]command{
			// don't do this for now, it just takes to long
			// {
			//	title: "systemd",
			//	cmd:   "journalctl --no-pager --output=short-precise | grep -v  'audit:\\|audit\\['",
			// },
			{
				title: "kern",
				cmd:   "journalctl --no-pager --output=short-precise -k",
			},
			{
				title: "containerd-info",
				cmd:   "crictl --runtime-endpoint unix:///run/containerd/containerd.sock info",
			},
			{
				title: "containerd-containers",
				cmd:   "crictl --runtime-endpoint unix:///run/containerd/containerd.sock ps",
			},
			{
				title: "containerd-pods",
				cmd:   "crictl --runtime-endpoint unix:///run/containerd/containerd.sock pods",
			},
			{
				title: "cloud-final",
				cmd:   "journalctl --no-pager -u cloud-final",
			},
			{
				title: "kubelet",
				cmd:   "journalctl --no-pager -u kubelet.service",
			},
			{
				title: "containerd",
				cmd:   "journalctl --no-pager -u containerd.service",
			},
		},
	)
}

func dumpSpecResources(ctx context.Context, e2eCtx *E2EContext, namespace *corev1.Namespace, directory ...string) {
	paths := append([]string{e2eCtx.Settings.ArtifactFolder, "clusters", e2eCtx.Environment.BootstrapClusterProxy.GetName(), "resources"}, directory...)
	framework.DumpAllResources(ctx, framework.DumpAllResourcesInput{
		Lister:    e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
		Namespace: namespace.Name,
		LogPath:   filepath.Join(paths...),
	})
}

func Logf(format string, a ...interface{}) {
	fmt.Fprintf(GinkgoWriter, "["+time.Now().Format(time.RFC3339)+"] "+format+"\n", a...)
}

func Debugf(debug bool, format string, a ...interface{}) {
	if debug {
		fmt.Fprintf(GinkgoWriter, "[DEBUG] ["+time.Now().Format(time.RFC3339)+"] "+format+"\n", a...)
	}
}

// LoadE2EConfig loads the e2econfig from the specified path.
func LoadE2EConfig(configPath string) *clusterctl.E2EConfig {
	config := clusterctl.LoadE2EConfig(context.TODO(), clusterctl.LoadE2EConfigInput{ConfigPath: configPath})
	Expect(config).ToNot(BeNil(), "Failed to load E2E config from %s", configPath)
	return config
}

// SetEnvVar sets an environment variable in the process. If marked private,
// the value is not printed.
func SetEnvVar(key, value string, private bool) {
	printableValue := "*******"
	if !private {
		printableValue = value
	}

	Logf("Setting environment variable: key=%s, value=%s", key, printableValue)
	_ = os.Setenv(key, value)
}

// getOpenStackClusterFromMachine gets the OpenStackCluster that is related to the given machine.
func getOpenStackClusterFromMachine(ctx context.Context, client client.Client, machine *clusterv1.Machine) (*infrav1.OpenStackCluster, error) {
	key := types.NamespacedName{
		Namespace: machine.Namespace,
		Name:      machine.Spec.ClusterName,
	}
	cluster := &clusterv1.Cluster{}
	err := client.Get(ctx, key, cluster)
	if err != nil {
		return nil, err
	}

	key = types.NamespacedName{
		Namespace: cluster.Spec.InfrastructureRef.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	openStackCluster := &infrav1.OpenStackCluster{}
	err = client.Get(ctx, key, openStackCluster)
	return openStackCluster, err
}

type OpenStackLogCollector struct {
	E2EContext E2EContext
}

// CollectMachineLog gets logs for the OpenStack resources related to the given machine.
func (o OpenStackLogCollector) CollectMachineLog(ctx context.Context, managementClusterClient client.Client, m *clusterv1.Machine, outputPath string) error {
	machineLogBase := path.Join(outputPath, "instances", m.Namespace, m.Name)
	metaLog := path.Join(machineLogBase, "instance.log")

	if err := os.MkdirAll(filepath.Dir(metaLog), 0o750); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "couldn't create directory %q for file: %s\n", metaLog, err)
	}

	f, err := os.OpenFile(metaLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "couldn't open file %q: %s\n", metaLog, err)
		return nil
	}
	defer f.Close()

	openStackCluster, err := getOpenStackClusterFromMachine(ctx, managementClusterClient, m)
	if err != nil {
		return fmt.Errorf("error getting OpenStackCluster for Machine: %s", err)
	}

	if len(m.Status.Addresses) < 1 {
		return fmt.Errorf("unable to get logs for machine since it has no address")
	}
	ip := m.Status.Addresses[0].Address

	srvs, err := GetOpenStackServers(&o.E2EContext, openStackCluster, sets.New(m.Spec.InfrastructureRef.Name))
	if err != nil {
		return fmt.Errorf("cannot dump machines, could not get servers from OpenStack: %v", err)
	}
	if len(srvs) != 1 {
		return fmt.Errorf("expected exactly 1 server but got %d", len(srvs))
	}
	srv := srvs[m.Spec.InfrastructureRef.Name]

	serverJSON, err := json.MarshalIndent(srv, "", "    ")
	if err != nil {
		return fmt.Errorf("error marshalling server %v: %s", srv, err)
	}
	if err := os.WriteFile(path.Join(machineLogBase, "server.txt"), serverJSON, 0o600); err != nil {
		return fmt.Errorf("error writing server JSON %s: %s", serverJSON, err)
	}
	_, _ = fmt.Fprintf(f, "instance found: %q\n", srv.ID)

	srvUser := o.E2EContext.E2EConfig.GetVariable(SSHUserMachine)
	executeCommands(
		ctx,
		o.E2EContext.Settings.ArtifactFolder,
		o.E2EContext.Settings.Debug,
		filepath.Dir(f.Name()),
		ip,
		openStackCluster.Status.Bastion.FloatingIP,
		srvUser,
		[]command{
			// don't do this for now, it just takes to long
			// {
			//	title: "systemd",
			//	cmd:   "journalctl --no-pager --output=short-precise | grep -v  'audit:\\|audit\\['",
			// },
			{
				title: "kern",
				cmd:   "journalctl --no-pager --output=short-precise -k",
			},
			{
				title: "containerd-info",
				cmd:   "crictl --runtime-endpoint unix:///run/containerd/containerd.sock info",
			},
			{
				title: "containerd-containers",
				cmd:   "crictl --runtime-endpoint unix:///run/containerd/containerd.sock ps",
			},
			{
				title: "containerd-pods",
				cmd:   "crictl --runtime-endpoint unix:///run/containerd/containerd.sock pods",
			},
			{
				title: "cloud-final",
				cmd:   "journalctl --no-pager -u cloud-final",
			},
			{
				title: "kubelet",
				cmd:   "journalctl --no-pager -u kubelet.service",
			},
			{
				title: "containerd",
				cmd:   "journalctl --no-pager -u containerd.service",
			},
		},
	)
	return nil
}

// CollectMachinePoolLog is not yet implemented for the OpenStack provider.
func (o OpenStackLogCollector) CollectMachinePoolLog(_ context.Context, _ client.Client, _ *expv1.MachinePool, _ string) error {
	return fmt.Errorf("not implemented")
}
