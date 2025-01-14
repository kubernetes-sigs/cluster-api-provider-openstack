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
	"path/filepath"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	capie2e "sigs.k8s.io/cluster-api/test/e2e"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/bootstrap"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
)

// createClusterctlLocalRepository generates a clusterctl repository.
// Must always be run after kubetest.NewConfiguration.
func createClusterctlLocalRepository(config *clusterctl.E2EConfig, repositoryFolder string) string {
	createRepositoryInput := clusterctl.CreateRepositoryInput{
		E2EConfig:        config,
		RepositoryFolder: repositoryFolder,
	}

	// Ensuring a CNI file is defined in the config and register a FileTransformation to inject the referenced file as in place of the CNI_RESOURCES envSubst variable.
	Expect(config.Variables).To(HaveKey(capie2e.CNIPath), "Missing %s variable in the config", capie2e.CNIPath)
	cniPath := config.GetVariable(capie2e.CNIPath)
	Expect(cniPath).To(BeAnExistingFile(), "The %s variable should resolve to an existing file", capie2e.CNIPath)
	createRepositoryInput.RegisterClusterResourceSetConfigMapTransformation(cniPath, capie2e.CNIResources)

	// Ensuring a CCM file is defined in the config and register a FileTransformation to inject the referenced file as in place of the CCM_RESOURCES envSubst variable.
	Expect(config.Variables).To(HaveKey(CCMPath), "Missing %s variable in the config", CCMPath)
	ccmPath := config.GetVariable(CCMPath)
	Expect(ccmPath).To(BeAnExistingFile(), "The %s variable should resolve to an existing file", CCMPath)
	createRepositoryInput.RegisterClusterResourceSetConfigMapTransformation(ccmPath, CCMResources)

	clusterctlConfig := clusterctl.CreateRepository(context.TODO(), createRepositoryInput)
	Expect(clusterctlConfig).To(BeAnExistingFile(), "The clusterctlConfig file does not exists in the local repository %s", repositoryFolder)
	return clusterctlConfig
}

// setupBootstrapCluster installs Cluster API components via clusterctl.
func setupBootstrapCluster(config *clusterctl.E2EConfig, scheme *runtime.Scheme, useExistingCluster bool) (bootstrap.ClusterProvider, framework.ClusterProxy) {
	Logf("Running setupBootstrapCluster (useExistingCluster: %t)", useExistingCluster)

	// We only want to set clusterProvider if we create a new bootstrap cluster in this test.
	// If we re-use an existing one, we don't want to delete it afterwards, so we don't set it.
	var clusterProvider bootstrap.ClusterProvider
	var kubeconfigPath string

	// try to use an existing cluster
	if useExistingCluster {
		// If the kubeContext is locked: try to use the default kubeconfig with the current context
		kubeContext := config.GetVariable(KubeContext)
		if kubeContext != "" {
			testKubeconfigPath := clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
			kubecfg, err := clientcmd.LoadFromFile(testKubeconfigPath)
			Expect(err).NotTo(HaveOccurred())

			// Only use the kubeconfigPath if the current context is the configured kubeContext
			// Otherwise we might deploy to the wrong cluster.
			// TODO(sbuerin): this logic could be a lot nicer if we could hand over a kubeContext to NewClusterProxy
			Logf("Found currentContext %q in %q (configured kubeContext is %q)", kubecfg.CurrentContext, testKubeconfigPath, kubeContext)
			if kubecfg.CurrentContext == kubeContext {
				kubeconfigPath = testKubeconfigPath
			}
		}
	}

	// If useExistingCluster was false or we couldn't find an existing cluster in the default kubeconfig with the configured kubeContext, let's create a new one
	if kubeconfigPath == "" {
		clusterProvider = bootstrap.CreateKindBootstrapClusterAndLoadImages(context.TODO(), bootstrap.CreateKindBootstrapClusterAndLoadImagesInput{
			Name:               config.ManagementClusterName,
			RequiresDockerSock: config.HasDockerProvider(),
			Images:             config.Images,
		})
		Expect(clusterProvider).ToNot(BeNil(), "Failed to create a bootstrap cluster")

		kubeconfigPath = clusterProvider.GetKubeconfigPath()
		Expect(kubeconfigPath).To(BeAnExistingFile(), "Failed to get the kubeconfig file for the bootstrap cluster")
	}

	clusterProxy := framework.NewClusterProxy("bootstrap", kubeconfigPath, scheme)
	Expect(clusterProxy).ToNot(BeNil(), "Failed to get a bootstrap cluster proxy")

	return clusterProvider, clusterProxy
}

// initBootstrapCluster uses kind to create a cluster.
func initBootstrapCluster(e2eCtx *E2EContext) {
	clusterctl.InitManagementClusterAndWatchControllerLogs(context.TODO(), clusterctl.InitManagementClusterAndWatchControllerLogsInput{
		ClusterProxy:              e2eCtx.Environment.BootstrapClusterProxy,
		ClusterctlConfigPath:      e2eCtx.Environment.ClusterctlConfigPath,
		InfrastructureProviders:   e2eCtx.E2EConfig.InfrastructureProviders(),
		RuntimeExtensionProviders: e2eCtx.E2EConfig.RuntimeExtensionProviders(),
		LogFolder:                 filepath.Join(e2eCtx.Settings.ArtifactFolder, "clusters", e2eCtx.Environment.BootstrapClusterProxy.GetName()),
	}, e2eCtx.E2EConfig.GetIntervals(e2eCtx.Environment.BootstrapClusterProxy.GetName(), "wait-controllers")...)
}

// tearDown the bootstrap kind cluster.
func tearDown(bootstrapClusterProvider bootstrap.ClusterProvider, bootstrapClusterProxy framework.ClusterProxy) {
	if bootstrapClusterProxy != nil {
		bootstrapClusterProxy.Dispose(context.TODO())
	}
	if bootstrapClusterProvider != nil {
		bootstrapClusterProvider.Dispose(context.TODO())
	}
}
