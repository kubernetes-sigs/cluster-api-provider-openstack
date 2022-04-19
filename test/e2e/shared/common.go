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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha5"
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
	Byf("Running DumpSpecResourcesAndCleanup for namespace %q", namespace.Name)
	// Dump all Cluster API related resources to artifacts before deleting them.
	cancelWatches := e2eCtx.Environment.Namespaces[namespace]
	dumpSpecResources(ctx, e2eCtx, namespace)

	dumpOpenStack(ctx, e2eCtx, e2eCtx.Environment.BootstrapClusterProxy.GetName())

	Byf("Dumping all OpenStack server instances in the %q namespace", namespace.Name)
	dumpMachines(ctx, e2eCtx, namespace)

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

func dumpMachines(ctx context.Context, e2eCtx *E2EContext, namespace *corev1.Namespace) {
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
	srvs, err := GetOpenStackServers(e2eCtx, cluster)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "cannot dump machines, could not get servers from OpenStack: %v\n", err)
		return
	}
	for _, m := range machines.Items {
		srv, ok := srvs[m.Name]
		if !ok {
			continue
		}
		dumpMachine(ctx, e2eCtx, m, srv, cluster.Status.Bastion.FloatingIP)
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

func dumpMachine(ctx context.Context, e2eCtx *E2EContext, machine infrav1.OpenStackMachine, srv ServerExtWithIP, bastionIP string) {
	logPath := filepath.Join(e2eCtx.Settings.ArtifactFolder, "clusters", e2eCtx.Environment.BootstrapClusterProxy.GetName())
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

	_, _ = fmt.Fprintf(f, "instance found: %q\n", srv.ID)
	executeCommands(
		ctx,
		e2eCtx.Settings.ArtifactFolder,
		e2eCtx.Settings.Debug,
		filepath.Dir(f.Name()),
		srv.ip,
		bastionIP,
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

func dumpSpecResources(ctx context.Context, e2eCtx *E2EContext, namespace *corev1.Namespace) {
	framework.DumpAllResources(ctx, framework.DumpAllResourcesInput{
		Lister:    e2eCtx.Environment.BootstrapClusterProxy.GetClient(),
		Namespace: namespace.Name,
		LogPath:   filepath.Join(e2eCtx.Settings.ArtifactFolder, "clusters", e2eCtx.Environment.BootstrapClusterProxy.GetName(), "resources"),
	})
}

func Byf(format string, a ...interface{}) {
	By("[" + time.Now().Format(time.RFC3339) + "] " + fmt.Sprintf(format, a...))
}

func Debugf(debug bool, format string, a ...interface{}) {
	if debug {
		By("[DEBUG] [" + time.Now().Format(time.RFC3339) + "] " + fmt.Sprintf(format, a...))
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

	Byf("Setting environment variable: key=%s, value=%s", key, printableValue)
	_ = os.Setenv(key, value)
}
