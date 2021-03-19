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
	"fmt"
	"os"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha4"
)

func SetupSpecNamespace(ctx context.Context, specName string, e2eCtx *E2EContext) *corev1.Namespace {
	Byf("Creating a namespace for hosting the %q test spec", specName)
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
	Byf("Dumping all the Cluster API resources in the %q namespace", namespace.Name)
	// Dump all Cluster API related resources to artifacts before deleting them.
	cancelWatches := e2eCtx.Environment.Namespaces[namespace]
	DumpSpecResources(ctx, e2eCtx, namespace)
	Byf("Dumping all OpenStack server instances in the %q namespace", namespace.Name)
	DumpMachines(ctx, e2eCtx, namespace)
	if !e2eCtx.Settings.SkipCleanup {
		framework.DeleteAllClustersAndWait(ctx, framework.DeleteAllClustersAndWaitInput{
			Client:    e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
			Namespace: namespace.Name,
		}, e2eCtx.E2EConfig.GetIntervals(specName, "wait-delete-cluster")...)

		Byf("Deleting namespace used for hosting the %q test spec", specName)
		framework.DeleteNamespace(ctx, framework.DeleteNamespaceInput{
			Deleter: e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
			Name:    namespace.Name,
		})
	}
	cancelWatches()
	delete(e2eCtx.Environment.Namespaces, namespace)
}

func DumpMachines(ctx context.Context, e2eCtx *E2EContext, namespace *corev1.Namespace) {
	By("Running DumpMachines")
	cluster := ClusterForSpec(ctx, e2eCtx.Environment.BootstrapClusterProxy, namespace)
	if cluster.Status.Bastion == nil || cluster.Status.Bastion.FloatingIP == "" {
		_, _ = fmt.Fprintln(GinkgoWriter, "cannot dump machines, cluster doesn't have a bastion host with a floating ip")
		return
	}
	machines := MachinesForSpec(ctx, e2eCtx.Environment.BootstrapClusterProxy, namespace)
	instances, err := allMachines(ctx, e2eCtx)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "cannot dump machines, could not get instances from OpenStack: %v\n", err)
		return
	}
	var machineInstance instance
	for _, m := range machines.Items {
		for _, i := range instances {
			if i.name == m.Name {
				machineInstance = i
				break
			}
		}
		if machineInstance.id == "" {
			return
		}
		DumpMachine(ctx, e2eCtx, m, machineInstance, cluster.Status.Bastion.FloatingIP)
	}
}

func ClusterForSpec(ctx context.Context, clusterProxy framework.ClusterProxy, namespace *corev1.Namespace) *infrav1.OpenStackCluster {
	lister := clusterProxy.GetClient()
	list := new(infrav1.OpenStackClusterList)
	if err := lister.List(ctx, list, client.InNamespace(namespace.GetName())); err != nil {
		_, _ = fmt.Fprintln(GinkgoWriter, "couldn't find cluster")
		return nil
	}
	Expect(list.Items).To(HaveLen(1), "Expected to find one cluster, found %d", len(list.Items))
	return &list.Items[0]
}

func MachinesForSpec(ctx context.Context, clusterProxy framework.ClusterProxy, namespace *corev1.Namespace) *infrav1.OpenStackMachineList {
	lister := clusterProxy.GetClient()
	list := new(infrav1.OpenStackMachineList)
	if err := lister.List(ctx, list, client.InNamespace(namespace.GetName())); err != nil {
		_, _ = fmt.Fprintln(GinkgoWriter, "couldn't find machines")
		return nil
	}
	return list
}

func DumpMachine(_ context.Context, e2eCtx *E2EContext, machine infrav1.OpenStackMachine, machineInstance instance, bastionIP string) {
	logPath := filepath.Join(e2eCtx.Settings.ArtifactFolder, "clusters", e2eCtx.Environment.BootstrapClusterProxy.GetName())
	machineLogBase := path.Join(logPath, "instances", machine.Namespace, machine.Name)
	metaLog := path.Join(machineLogBase, "instance.log")
	if err := os.MkdirAll(filepath.Dir(metaLog), 0750); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "couldn't create directory %q for file: %s", metaLog, err)
	}
	f, err := os.OpenFile(metaLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "couldn't open file %q: %s", metaLog, err)
		return
	}
	defer f.Close()

	_, _ = fmt.Fprintf(f, "instance found: %q\n", machineInstance.id)
	commandsForMachine(
		f,
		machineInstance.ip,
		bastionIP,
		[]command{
			// don't do this for now, it just takes to long
			//{
			//	title: "systemd",
			//	cmd:   "journalctl --no-pager --output=short-precise | grep -v  'audit:\\|audit\\['",
			//},
			{
				title: "kern",
				cmd:   "journalctl --no-pager --output=short-precise -k",
			},
			{
				title: "containerd-info",
				cmd:   "crictl info",
			},
			{
				title: "containerd-containers",
				cmd:   "crictl ps",
			},
			{
				title: "containerd-pods",
				cmd:   "crictl pods",
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

func DumpSpecResources(ctx context.Context, e2eCtx *E2EContext, namespace *corev1.Namespace) {
	framework.DumpAllResources(ctx, framework.DumpAllResourcesInput{
		Lister:    e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
		Namespace: namespace.Name,
		LogPath:   filepath.Join(e2eCtx.Settings.ArtifactFolder, "clusters", e2eCtx.Environment.BootstrapClusterProxy.GetName(), "resources"),
	})
}

func Byf(format string, a ...interface{}) {
	By(fmt.Sprintf(format, a...))
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

	Byf("Setting environment variable: key=%s, value=%s", key, printableValue)
	_ = os.Setenv(key, value)
}
