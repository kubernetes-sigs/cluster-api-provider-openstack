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
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/yaml"
)

type synchronizedBeforeTestSuiteConfig struct {
	ArtifactFolder          string               `json:"artifactFolder,omitempty"`
	ConfigPath              string               `json:"configPath,omitempty"`
	ClusterctlConfigPath    string               `json:"clusterctlConfigPath,omitempty"`
	KubeconfigPath          string               `json:"kubeconfigPath,omitempty"`
	E2EConfig               clusterctl.E2EConfig `json:"e2eConfig,omitempty"`
	KubetestConfigFilePath  string               `json:"kubetestConfigFilePath,omitempty"`
	UseCIArtifacts          bool                 `json:"useCIArtifacts,omitempty"`
	GinkgoNodes             int                  `json:"ginkgoNodes,omitempty"`
	GinkgoSlowSpecThreshold int                  `json:"ginkgoSlowSpecThreshold,omitempty"`
}

// Node1BeforeSuite is the common setup down on the first ginkgo node before the test suite runs
func Node1BeforeSuite(e2eCtx *E2EContext) []byte {
	By("Running Node1BeforeSuite")

	flag.Parse()
	Expect(e2eCtx.Settings.ConfigPath).To(BeAnExistingFile(), "Invalid test suite argument. configPath should be an existing file.")
	Expect(os.MkdirAll(e2eCtx.Settings.ArtifactFolder, 0o750)).To(Succeed(), "Invalid test suite argument. Can't create artifacts-folder %q", e2eCtx.Settings.ArtifactFolder)
	Byf("Loading the e2e test configuration from %q", e2eCtx.Settings.ConfigPath)
	e2eCtx.E2EConfig = LoadE2EConfig(e2eCtx.Settings.ConfigPath)
	sourceTemplate, err := ioutil.ReadFile(filepath.Join(e2eCtx.Settings.DataFolder, e2eCtx.Settings.SourceTemplate))
	Expect(err).NotTo(HaveOccurred())

	var clusterctlCITemplate clusterctl.Files

	platformKustomization, err := ioutil.ReadFile(filepath.Join(e2eCtx.Settings.DataFolder, "ci-artifacts-platform-kustomization.yaml"))
	Expect(err).NotTo(HaveOccurred())

	// TODO(sbuerin): should be removed after: https://github.com/kubernetes-sigs/kustomize/issues/2825 is fixed
	//ciTemplate, err := kubernetesversions.GenerateCIArtifactsInjectedTemplateForDebian(
	//	kubernetesversions.GenerateCIArtifactsInjectedTemplateForDebianInput{
	//		ArtifactsDirectory:    e2eCtx.Settings.ArtifactFolder,
	//		SourceTemplate:        sourceTemplate,
	//		PlatformKustomization: platformKustomization,
	//	},
	//)
	ciTemplate, err := GenerateCIArtifactsInjectedTemplateForDebian(
		GenerateCIArtifactsInjectedTemplateForDebianInput{
			ArtifactsDirectory:    e2eCtx.Settings.ArtifactFolder,
			SourceTemplate:        sourceTemplate,
			PlatformKustomization: platformKustomization,
		},
	)
	Expect(err).NotTo(HaveOccurred())

	clusterctlCITemplate = clusterctl.Files{
		SourcePath: ciTemplate,
		TargetName: "cluster-template-conformance-ci-artifacts.yaml",
	}

	providers := e2eCtx.E2EConfig.Providers
	for i, prov := range providers {
		if prov.Name != "openstack" {
			continue
		}
		e2eCtx.E2EConfig.Providers[i].Files = append(e2eCtx.E2EConfig.Providers[i].Files, clusterctlCITemplate)
	}

	openStackCloudYAMLFile := e2eCtx.E2EConfig.GetVariable(OpenStackCloudYAMLFile)
	openStackCloud := e2eCtx.E2EConfig.GetVariable(OpenStackCloud)
	ensureSSHKeyPair(openStackCloudYAMLFile, openStackCloud, DefaultSSHKeyPairName)

	Byf("Creating a clusterctl local repository into %q", e2eCtx.Settings.ArtifactFolder)
	e2eCtx.Environment.ClusterctlConfigPath = createClusterctlLocalRepository(e2eCtx.E2EConfig, filepath.Join(e2eCtx.Settings.ArtifactFolder, "repository"))

	By("Setting up the bootstrap cluster")
	e2eCtx.Environment.BootstrapClusterProvider, e2eCtx.Environment.BootstrapClusterProxy = setupBootstrapCluster(e2eCtx.E2EConfig, e2eCtx.Environment.Scheme, e2eCtx.Settings.UseExistingCluster)

	SetEnvVar("OPENSTACK_CLOUD_YAML_B64", getEncodedOpenStackCloudYAML(openStackCloudYAMLFile), true)
	SetEnvVar("OPENSTACK_CLOUD_PROVIDER_CONF_B64", getEncodedOpenStackCloudProviderConf(openStackCloudYAMLFile, openStackCloud), true)

	By("Initializing the bootstrap cluster")
	initBootstrapCluster(e2eCtx)

	conf := synchronizedBeforeTestSuiteConfig{
		ArtifactFolder:          e2eCtx.Settings.ArtifactFolder,
		ConfigPath:              e2eCtx.Settings.ConfigPath,
		ClusterctlConfigPath:    e2eCtx.Environment.ClusterctlConfigPath,
		KubeconfigPath:          e2eCtx.Environment.BootstrapClusterProxy.GetKubeconfigPath(),
		E2EConfig:               *e2eCtx.E2EConfig,
		KubetestConfigFilePath:  e2eCtx.Settings.KubetestConfigFilePath,
		UseCIArtifacts:          e2eCtx.Settings.UseCIArtifacts,
		GinkgoNodes:             e2eCtx.Settings.GinkgoNodes,
		GinkgoSlowSpecThreshold: e2eCtx.Settings.GinkgoSlowSpecThreshold,
	}

	data, err := yaml.Marshal(conf)
	Expect(err).NotTo(HaveOccurred())
	return data
}

// AllNodesBeforeSuite is the common setup down on each ginkgo parallel node before the test suite runs
func AllNodesBeforeSuite(e2eCtx *E2EContext, data []byte) {
	By("Running AllNodesBeforeSuite")

	conf := &synchronizedBeforeTestSuiteConfig{}
	err := yaml.UnmarshalStrict(data, conf)
	Expect(err).NotTo(HaveOccurred())
	e2eCtx.Settings.ArtifactFolder = conf.ArtifactFolder
	e2eCtx.Settings.ConfigPath = conf.ConfigPath
	e2eCtx.Environment.ClusterctlConfigPath = conf.ClusterctlConfigPath
	e2eCtx.Environment.BootstrapClusterProxy = framework.NewClusterProxy("bootstrap", conf.KubeconfigPath, e2eCtx.Environment.Scheme)
	e2eCtx.E2EConfig = &conf.E2EConfig
	e2eCtx.Settings.KubetestConfigFilePath = conf.KubetestConfigFilePath
	e2eCtx.Settings.UseCIArtifacts = conf.UseCIArtifacts
	e2eCtx.Settings.GinkgoNodes = conf.GinkgoNodes
	e2eCtx.Settings.GinkgoSlowSpecThreshold = conf.GinkgoSlowSpecThreshold

	SetEnvVar("OPENSTACK_SSH_KEY_NAME", DefaultSSHKeyPairName, false)

	e2eCtx.Environment.ResourceTicker = time.NewTicker(time.Second * 5)
	e2eCtx.Environment.ResourceTickerDone = make(chan bool)
	// Get OpenStack server logs every 5 minutes
	e2eCtx.Environment.MachineTicker = time.NewTicker(time.Second * 300)
	e2eCtx.Environment.MachineTickerDone = make(chan bool)
	resourceCtx, resourceCancel := context.WithCancel(context.Background())
	machineCtx, machineCancel := context.WithCancel(context.Background())

	// Dump resources every 5 seconds
	go func() {
		defer GinkgoRecover()
		for {
			select {
			case <-e2eCtx.Environment.ResourceTickerDone:
				resourceCancel()
				return
			case <-e2eCtx.Environment.ResourceTicker.C:
				for k := range e2eCtx.Environment.Namespaces {
					DumpSpecResources(resourceCtx, e2eCtx, k)
				}
			}
		}
	}()

	// Dump machine logs every 60 seconds
	go func() {
		defer GinkgoRecover()
		for {
			select {
			case <-e2eCtx.Environment.MachineTickerDone:
				machineCancel()
				return
			case <-e2eCtx.Environment.MachineTicker.C:
				for k := range e2eCtx.Environment.Namespaces {
					DumpMachines(machineCtx, e2eCtx, k)
				}
			}
		}
	}()
}

// Node1AfterSuite is cleanup that runs on the first ginkgo node after the test suite finishes
func Node1AfterSuite(e2eCtx *E2EContext) {
	By("Running Node1AfterSuite")

	if e2eCtx.Environment.ResourceTickerDone != nil {
		e2eCtx.Environment.ResourceTickerDone <- true
	}
	if e2eCtx.Environment.MachineTickerDone != nil {
		e2eCtx.Environment.MachineTickerDone <- true
	}
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Minute)
	defer cancel()
	DumpOpenStackClusters(ctx, e2eCtx, e2eCtx.Environment.BootstrapClusterProxy.GetName())
	for k := range e2eCtx.Environment.Namespaces {
		DumpSpecResourcesAndCleanup(ctx, "", k, e2eCtx)
		DumpMachines(ctx, e2eCtx, k)
	}
}

// AllNodesAfterSuite is cleanup that runs on all ginkgo parallel nodes after the test suite finishes
func AllNodesAfterSuite(e2eCtx *E2EContext) {
	By("Running AllNodesAfterSuite")

	By("Tearing down the management cluster")
	if !e2eCtx.Settings.SkipCleanup {
		tearDown(e2eCtx.Environment.BootstrapClusterProvider, e2eCtx.Environment.BootstrapClusterProxy)
	}
}
