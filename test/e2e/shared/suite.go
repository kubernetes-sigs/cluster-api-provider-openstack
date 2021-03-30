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
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
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

// Node1BeforeSuite is the common setup down on the first ginkgo node before the test suite runs.
func Node1BeforeSuite(e2eCtx *E2EContext) []byte {
	Byf("Running Node1BeforeSuite")
	defer Byf("Finished Node1BeforeSuite")

	flag.Parse()
	Expect(e2eCtx.Settings.ConfigPath).To(BeAnExistingFile(), "Invalid test suite argument. configPath should be an existing file.")
	Expect(os.MkdirAll(e2eCtx.Settings.ArtifactFolder, 0o750)).To(Succeed(), "Invalid test suite argument. Can't create artifacts-folder %q", e2eCtx.Settings.ArtifactFolder)
	Byf("Loading the e2e test configuration from %q", e2eCtx.Settings.ConfigPath)
	e2eCtx.E2EConfig = LoadE2EConfig(e2eCtx.Settings.ConfigPath)

	Expect(e2eCtx.E2EConfig.GetVariable(OpenStackCloudYAMLFile)).To(BeAnExistingFile(), "Invalid test suite argument. Value of environment variable OPENSTACK_CLOUD_YAML_FILE should be an existing file: %s", e2eCtx.E2EConfig.GetVariable(OpenStackCloudYAMLFile))
	Byf("Loading the clouds.yaml from %q", e2eCtx.E2EConfig.GetVariable(OpenStackCloudYAMLFile))

	// TODO(sbuerin): we always need ci artifacts, because we don't have images for every Kubernetes version
	err := filepath.WalkDir(path.Join(e2eCtx.Settings.DataFolder, "infrastructure-openstack"), func(f string, d fs.DirEntry, _ error) error {
		filename := filepath.Base(f)
		fileExtension := filepath.Ext(filename)
		if d.IsDir() || !strings.HasPrefix(filename, "cluster-template") {
			return nil
		}

		sourceTemplate, err := ioutil.ReadFile(f)
		Expect(err).NotTo(HaveOccurred())

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

		targetName := fmt.Sprintf("%s-ci-artifacts.yaml", strings.TrimSuffix(filename, fileExtension))
		targetTemplate := path.Join(e2eCtx.Settings.ArtifactFolder, "templates", targetName)

		// We have to copy the file from ciTemplate to targetTemplate. Otherwise it would be overwritten because
		// ciTemplate is the same for all templates
		ciTemplateBytes, err := os.ReadFile(ciTemplate)
		Expect(err).NotTo(HaveOccurred())
		err = os.WriteFile(targetTemplate, ciTemplateBytes, 0o600)
		Expect(err).NotTo(HaveOccurred())

		clusterctlCITemplate := clusterctl.Files{
			SourcePath: targetTemplate,
			TargetName: targetName,
		}

		for i, prov := range e2eCtx.E2EConfig.Providers {
			if prov.Name != "openstack" {
				continue
			}
			e2eCtx.E2EConfig.Providers[i].Files = append(e2eCtx.E2EConfig.Providers[i].Files, clusterctlCITemplate)
		}
		return nil
	})
	Expect(err).NotTo(HaveOccurred())

	ensureSSHKeyPair(e2eCtx)

	Byf("Creating a clusterctl local repository into %q", e2eCtx.Settings.ArtifactFolder)
	e2eCtx.Environment.ClusterctlConfigPath = createClusterctlLocalRepository(e2eCtx.E2EConfig, filepath.Join(e2eCtx.Settings.ArtifactFolder, "repository"))

	Byf("Setting up the bootstrap cluster")
	e2eCtx.Environment.BootstrapClusterProvider, e2eCtx.Environment.BootstrapClusterProxy = setupBootstrapCluster(e2eCtx.E2EConfig, e2eCtx.Environment.Scheme, e2eCtx.Settings.UseExistingCluster)

	Byf("Initializing the bootstrap cluster")
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

// AllNodesBeforeSuite is the common setup down on each ginkgo parallel node before the test suite runs.
func AllNodesBeforeSuite(e2eCtx *E2EContext, data []byte) {
	Byf("Running AllNodesBeforeSuite")
	defer Byf("Finished AllNodesBeforeSuite")

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

	openStackCloudYAMLFile := e2eCtx.E2EConfig.GetVariable(OpenStackCloudYAMLFile)
	openStackCloud := e2eCtx.E2EConfig.GetVariable(OpenStackCloud)
	SetEnvVar("OPENSTACK_CLOUD_YAML_B64", getEncodedOpenStackCloudYAML(openStackCloudYAMLFile), true)
	SetEnvVar("OPENSTACK_CLOUD_PROVIDER_CONF_B64", getEncodedOpenStackCloudProviderConf(openStackCloudYAMLFile, openStackCloud), true)
	SetEnvVar("OPENSTACK_SSH_KEY_NAME", DefaultSSHKeyPairName, false)

	e2eCtx.Environment.ResourceTicker = time.NewTicker(time.Second * 10)
	e2eCtx.Environment.ResourceTickerDone = make(chan bool)
	// Get OpenStack server logs every 5 minutes
	e2eCtx.Environment.MachineTicker = time.NewTicker(time.Second * 60)
	e2eCtx.Environment.MachineTickerDone = make(chan bool)
	resourceCtx, resourceCancel := context.WithCancel(context.Background())
	machineCtx, machineCancel := context.WithCancel(context.Background())

	debug := e2eCtx.Settings.Debug
	// Dump resources every 5 seconds
	go func() {
		defer GinkgoRecover()
		for {
			select {
			case <-e2eCtx.Environment.ResourceTickerDone:
				Debugf(debug, "Running ResourceTickerDone")
				resourceCancel()
				Debugf(debug, "Finished ResourceTickerDone")
				return
			case <-e2eCtx.Environment.ResourceTicker.C:
				Debugf(debug, "Running ResourceTicker")
				for k := range e2eCtx.Environment.Namespaces {
					// ensure dumpSpecResources cannot get stuck indefinitely
					timeoutCtx, timeoutCancel := context.WithTimeout(resourceCtx, time.Second*5)
					dumpSpecResources(timeoutCtx, e2eCtx, k)
					timeoutCancel()
				}
				Debugf(debug, "Finished ResourceTicker")
			}
		}
	}()

	// Dump machine logs every 60 seconds
	go func() {
		defer GinkgoRecover()
		for {
			select {
			case <-e2eCtx.Environment.MachineTickerDone:
				Debugf(debug, "Running MachineTickerDone")
				machineCancel()
				Debugf(debug, "Finished MachineTickerDone")
				return
			case <-e2eCtx.Environment.MachineTicker.C:
				Debugf(debug, "Running MachineTicker")
				for k := range e2eCtx.Environment.Namespaces {
					// ensure dumpMachines cannot get stuck indefinitely
					timeoutCtx, timeoutCancel := context.WithTimeout(machineCtx, time.Second*30)
					dumpMachines(timeoutCtx, e2eCtx, k)
					timeoutCancel()
				}
				Debugf(debug, "Finished MachineTicker")
			}
		}
	}()
}

// AllNodesAfterSuite is cleanup that runs on all ginkgo parallel nodes after the test suite finishes.
func AllNodesAfterSuite(e2eCtx *E2EContext) {
	Byf("Running AllNodesAfterSuite")
	defer Byf("Finished AllNodesAfterSuite")

	Byf("Stopping ResourceTicker")
	if e2eCtx.Environment.ResourceTickerDone != nil {
		e2eCtx.Environment.ResourceTickerDone <- true
	}
	Byf("Stopped ResourceTicker")

	Byf("Stopping MachineTicker")
	if e2eCtx.Environment.MachineTickerDone != nil {
		e2eCtx.Environment.MachineTickerDone <- true
	}
	Byf("Stopped MachineTicker")

	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Minute)
	defer cancel()

	if e2eCtx.Environment.BootstrapClusterProxy == nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "bootstrap cluster proxy does not exist yet, cannot dump clusters and machines\n")
		return
	}

	for namespace := range e2eCtx.Environment.Namespaces {
		DumpSpecResourcesAndCleanup(ctx, "after-suite", namespace, e2eCtx)
	}
}

// Node1AfterSuite is cleanup that runs on the first ginkgo node after the test suite finishes.
func Node1AfterSuite(e2eCtx *E2EContext) {
	Byf("Running Node1AfterSuite")
	defer Byf("Finished Node1AfterSuite")

	Byf("Tearing down the management cluster")
	if !e2eCtx.Settings.SkipCleanup {
		tearDown(e2eCtx.Environment.BootstrapClusterProvider, e2eCtx.Environment.BootstrapClusterProxy)
	}
}
