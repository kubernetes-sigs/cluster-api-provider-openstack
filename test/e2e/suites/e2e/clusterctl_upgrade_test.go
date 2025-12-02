//go:build e2e

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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"

	"sigs.k8s.io/cluster-api-provider-openstack/test/e2e/shared"
)

var (
	capoRelease011 string
	capoRelease012 string
	capiRelease110 string
)

var _ = Describe("When testing clusterctl upgrades for CAPO (v0.11=>current) and ORC (v1.0.2=>current) [clusterctl-upgrade]", func() {
	BeforeEach(func(ctx context.Context) {
		setDownloadE2EImageEnvVar()
		// Note: This gives the version without the 'v' prefix, so we need to add it below.
		capoRelease011, err = clusterctl.ResolveRelease(ctx, "go://github.com/kubernetes-sigs/cluster-api-provider-openstack@v0.11")
		Expect(err).ToNot(HaveOccurred(), "failed to get stable release of CAPO")
		capoRelease011 = "v" + capoRelease011
		// Note: This gives the version without the 'v' prefix, so we need to add it below.
		capiRelease110, err = capi_e2e.GetStableReleaseOfMinor(ctx, "1.10")
		Expect(err).ToNot(HaveOccurred(), "failed to get stable release of CAPI")
		capiRelease110 = "v" + capiRelease110
	})

	capi_e2e.ClusterctlUpgradeSpec(context.TODO(), func() capi_e2e.ClusterctlUpgradeSpecInput {
		return capi_e2e.ClusterctlUpgradeSpecInput{
			E2EConfig:                         e2eCtx.E2EConfig,
			ClusterctlConfigPath:              e2eCtx.Environment.ClusterctlConfigPath,
			BootstrapClusterProxy:             e2eCtx.Environment.BootstrapClusterProxy,
			ArtifactFolder:                    e2eCtx.Settings.ArtifactFolder,
			SkipCleanup:                       false,
			InitWithBinary:                    "https://github.com/kubernetes-sigs/cluster-api/releases/download/" + capiRelease110 + "/clusterctl-{OS}-{ARCH}",
			InitWithProvidersContract:         "v1beta1",
			InitWithInfrastructureProviders:   []string{"openstack:" + capoRelease011},
			InitWithCoreProvider:              "cluster-api:" + capiRelease110,
			InitWithBootstrapProviders:        []string{"kubeadm:" + capiRelease110},
			InitWithControlPlaneProviders:     []string{"kubeadm:" + capiRelease110},
			MgmtFlavor:                        shared.FlavorDefault,
			WorkloadFlavor:                    shared.FlavorDefault,
			InitWithKubernetesVersion:         e2eCtx.E2EConfig.MustGetVariable(shared.KubernetesVersion),
			InitWithRuntimeExtensionProviders: []string{"openstack-resource-controller:v1.0.2"},
		}
	})
})

var _ = Describe("When testing clusterctl upgrades for CAPO (v0.12=>current) and ORC (v1.0.2=>current)[clusterctl-upgrade]", func() {
	BeforeEach(func(ctx context.Context) {
		setDownloadE2EImageEnvVar()
		// Note: This gives the version without the 'v' prefix, so we need to add it below.
		capoRelease012, err = clusterctl.ResolveRelease(ctx, "go://github.com/kubernetes-sigs/cluster-api-provider-openstack@v0.12")
		Expect(err).ToNot(HaveOccurred(), "failed to get stable release of CAPO")
		capoRelease012 = "v" + capoRelease012
		// Note: This gives the version without the 'v' prefix, so we need to add it below.
		capiRelease110, err = capi_e2e.GetStableReleaseOfMinor(ctx, "1.10")
		Expect(err).ToNot(HaveOccurred(), "failed to get stable release of CAPI")
		capiRelease110 = "v" + capiRelease110
	})

	capi_e2e.ClusterctlUpgradeSpec(context.TODO(), func() capi_e2e.ClusterctlUpgradeSpecInput {
		return capi_e2e.ClusterctlUpgradeSpecInput{
			E2EConfig:                         e2eCtx.E2EConfig,
			ClusterctlConfigPath:              e2eCtx.Environment.ClusterctlConfigPath,
			BootstrapClusterProxy:             e2eCtx.Environment.BootstrapClusterProxy,
			ArtifactFolder:                    e2eCtx.Settings.ArtifactFolder,
			SkipCleanup:                       false,
			InitWithBinary:                    "https://github.com/kubernetes-sigs/cluster-api/releases/download/" + capiRelease110 + "/clusterctl-{OS}-{ARCH}",
			InitWithProvidersContract:         "v1beta1",
			InitWithInfrastructureProviders:   []string{"openstack:" + capoRelease012},
			InitWithCoreProvider:              "cluster-api:" + capiRelease110,
			InitWithBootstrapProviders:        []string{"kubeadm:" + capiRelease110},
			InitWithControlPlaneProviders:     []string{"kubeadm:" + capiRelease110},
			MgmtFlavor:                        shared.FlavorDefault,
			WorkloadFlavor:                    shared.FlavorDefault,
			InitWithKubernetesVersion:         e2eCtx.E2EConfig.MustGetVariable(shared.KubernetesVersion),
			InitWithRuntimeExtensionProviders: []string{"openstack-resource-controller:v1.0.2"},
		}
	})
})
