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

	shared "sigs.k8s.io/cluster-api-provider-openstack/test/e2e/shared"
)

var (
	capoRelease013 string
	capoRelease014 string
	capoRelease015 string
	capiRelease112 string
	capiRelease113 string
)

// CAPO v0.13 is built against CAPI v1beta2 and therefore cannot run against CAPI v1.10.
var _ = Describe("When testing clusterctl upgrades for CAPO (v0.13=>current) and ORC (v1.0.2=>current)[clusterctl-upgrade]", func() {
	BeforeEach(func(ctx context.Context) {
		// Note: This gives the version without the 'v' prefix, so we need to add it below.
		capoRelease013, err = clusterctl.ResolveRelease(ctx, "go://github.com/kubernetes-sigs/cluster-api-provider-openstack@v0.13")
		Expect(err).ToNot(HaveOccurred(), "failed to get stable release of CAPO")
		capoRelease013 = "v" + capoRelease013
		// Note: This gives the version without the 'v' prefix, so we need to add it below.
		capiRelease112, err = capi_e2e.GetStableReleaseOfMinor(ctx, "1.12")
		Expect(err).ToNot(HaveOccurred(), "failed to get stable release of CAPI")
		capiRelease112 = "v" + capiRelease112
	})

	capi_e2e.ClusterctlUpgradeSpec(context.TODO(), func() capi_e2e.ClusterctlUpgradeSpecInput {
		return capi_e2e.ClusterctlUpgradeSpecInput{
			E2EConfig:                         e2eCtx.E2EConfig,
			ClusterctlConfigPath:              e2eCtx.Environment.ClusterctlConfigPath,
			BootstrapClusterProxy:             e2eCtx.Environment.BootstrapClusterProxy,
			ArtifactFolder:                    e2eCtx.Settings.ArtifactFolder,
			SkipCleanup:                       false,
			InitWithBinary:                    "https://github.com/kubernetes-sigs/cluster-api/releases/download/" + capiRelease112 + "/clusterctl-{OS}-{ARCH}",
			InitWithInfrastructureProviders:   []string{"openstack:" + capoRelease013},
			InitWithCoreProvider:              "cluster-api:" + capiRelease112,
			InitWithBootstrapProviders:        []string{"kubeadm:" + capiRelease112},
			InitWithControlPlaneProviders:     []string{"kubeadm:" + capiRelease112},
			InitWithRuntimeExtensionProviders: []string{"openstack-resource-controller:v1.0.2"},
			MgmtFlavor:                        shared.FlavorDefault,
			WorkloadFlavor:                    shared.FlavorCapiV1Beta1,
			InitWithKubernetesVersion:         e2eCtx.E2EConfig.MustGetVariable(shared.KubernetesKindVersion),
			// The capi-v1beta1 flavor's OpenStackMachineTemplates are pinned to
			// the OPENSTACK_IMAGE_NAME_UPGRADE_FROM image (Kubernetes
			// KUBERNETES_VERSION_UPGRADE_FROM), so the workload cluster's
			// Kubernetes version must match to avoid an unexpected rollout.
			WorkloadKubernetesVersion:   e2eCtx.E2EConfig.MustGetVariable(shared.KubernetesVersionUpgradeFrom),
			UseKindForManagementCluster: true,
		}
	})
})

var _ = Describe("When testing clusterctl upgrades for CAPO (v0.14=>current) and ORC (v1.0.2=>current)[clusterctl-upgrade]", func() {
	BeforeEach(func(ctx context.Context) {
		// Note: This gives the version without the 'v' prefix, so we need to add it below.
		capoRelease014, err = clusterctl.ResolveRelease(ctx, "go://github.com/kubernetes-sigs/cluster-api-provider-openstack@v0.14")
		Expect(err).ToNot(HaveOccurred(), "failed to get stable release of CAPO")
		capoRelease014 = "v" + capoRelease014
		// Note: This gives the version without the 'v' prefix, so we need to add it below.
		capiRelease112, err = capi_e2e.GetStableReleaseOfMinor(ctx, "1.12")
		Expect(err).ToNot(HaveOccurred(), "failed to get stable release of CAPI")
		capiRelease112 = "v" + capiRelease112
	})

	capi_e2e.ClusterctlUpgradeSpec(context.TODO(), func() capi_e2e.ClusterctlUpgradeSpecInput {
		return capi_e2e.ClusterctlUpgradeSpecInput{
			E2EConfig:                         e2eCtx.E2EConfig,
			ClusterctlConfigPath:              e2eCtx.Environment.ClusterctlConfigPath,
			BootstrapClusterProxy:             e2eCtx.Environment.BootstrapClusterProxy,
			ArtifactFolder:                    e2eCtx.Settings.ArtifactFolder,
			SkipCleanup:                       false,
			InitWithBinary:                    "https://github.com/kubernetes-sigs/cluster-api/releases/download/" + capiRelease112 + "/clusterctl-{OS}-{ARCH}",
			InitWithInfrastructureProviders:   []string{"openstack:" + capoRelease014},
			InitWithCoreProvider:              "cluster-api:" + capiRelease112,
			InitWithBootstrapProviders:        []string{"kubeadm:" + capiRelease112},
			InitWithControlPlaneProviders:     []string{"kubeadm:" + capiRelease112},
			InitWithRuntimeExtensionProviders: []string{"openstack-resource-controller:v1.0.2"},
			MgmtFlavor:                        shared.FlavorDefault,
			WorkloadFlavor:                    shared.FlavorCapiV1Beta1,
			InitWithKubernetesVersion:         e2eCtx.E2EConfig.MustGetVariable(shared.KubernetesKindVersion),
			// The capi-v1beta1 flavor's OpenStackMachineTemplates are pinned to
			// the OPENSTACK_IMAGE_NAME_UPGRADE_FROM image (Kubernetes
			// KUBERNETES_VERSION_UPGRADE_FROM), so the workload cluster's
			// Kubernetes version must match to avoid an unexpected rollout.
			WorkloadKubernetesVersion:   e2eCtx.E2EConfig.MustGetVariable(shared.KubernetesVersionUpgradeFrom),
			UseKindForManagementCluster: true,
		}
	})
})

var _ = Describe("When testing clusterctl upgrades for CAPO (v0.15=>current) and ORC (v1.0.2=>current)[clusterctl-upgrade]", func() {
	BeforeEach(func(ctx context.Context) {
		// Note: This gives the version without the 'v' prefix, so we need to add it below.
		capoRelease015, err = clusterctl.ResolveRelease(ctx, "go://github.com/kubernetes-sigs/cluster-api-provider-openstack@latest-v0.15")
		Expect(err).ToNot(HaveOccurred(), "failed to get stable release of CAPO")
		capoRelease015 = "v" + capoRelease015
		// Note: This gives the version without the 'v' prefix, so we need to add it below.
		capiRelease113, err = capi_e2e.GetStableReleaseOfMinor(ctx, "1.13")
		Expect(err).ToNot(HaveOccurred(), "failed to get stable release of CAPI")
		capiRelease113 = "v" + capiRelease113
	})

	capi_e2e.ClusterctlUpgradeSpec(context.TODO(), func() capi_e2e.ClusterctlUpgradeSpecInput {
		return capi_e2e.ClusterctlUpgradeSpecInput{
			E2EConfig:                         e2eCtx.E2EConfig,
			ClusterctlConfigPath:              e2eCtx.Environment.ClusterctlConfigPath,
			BootstrapClusterProxy:             e2eCtx.Environment.BootstrapClusterProxy,
			ArtifactFolder:                    e2eCtx.Settings.ArtifactFolder,
			SkipCleanup:                       false,
			InitWithBinary:                    "https://github.com/kubernetes-sigs/cluster-api/releases/download/" + capiRelease113 + "/clusterctl-{OS}-{ARCH}",
			InitWithInfrastructureProviders:   []string{"openstack:" + capoRelease015},
			InitWithCoreProvider:              "cluster-api:" + capiRelease113,
			InitWithBootstrapProviders:        []string{"kubeadm:" + capiRelease113},
			InitWithControlPlaneProviders:     []string{"kubeadm:" + capiRelease113},
			InitWithRuntimeExtensionProviders: []string{"openstack-resource-controller:v1.0.2"},
			MgmtFlavor:                        shared.FlavorDefault,
			WorkloadFlavor:                    shared.FlavorCapiV1Beta1,
			InitWithKubernetesVersion:         e2eCtx.E2EConfig.MustGetVariable(shared.KubernetesKindVersion),
			// The capi-v1beta1 flavor's OpenStackMachineTemplates are pinned to
			// the OPENSTACK_IMAGE_NAME_UPGRADE_FROM image (Kubernetes
			// KUBERNETES_VERSION_UPGRADE_FROM), so the workload cluster's
			// Kubernetes version must match to avoid an unexpected rollout.
			WorkloadKubernetesVersion:   e2eCtx.E2EConfig.MustGetVariable(shared.KubernetesVersionUpgradeFrom),
			UseKindForManagementCluster: true,
		}
	})
})
