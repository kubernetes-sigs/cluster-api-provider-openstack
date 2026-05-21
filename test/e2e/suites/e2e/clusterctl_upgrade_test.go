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
	"fmt"
	"io"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
	capi_framework "sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"

	shared "sigs.k8s.io/cluster-api-provider-openstack/test/e2e/shared"
)

var (
	capoRelease012 string
	capoRelease013 string
	capoRelease014 string
	capiRelease110 string
	capiRelease112 string
)

// NOTE: clusterctl v1.10 cannot handle RuntimeExtensionProvider in the local
// filesystem repository created by the CAPI v1.13 test framework. And clusterctl
// v1.12 refuses to operate against v1beta1 management clusters. Therefore, we
// cannot install ORC as a RuntimeExtension and have to use hooks instead.
//
// CAPO v0.13 was compiled against CAPI v1.11, and CAPO v0.14 against CAPI v1.12.
// Both CAPI v1.11 and v1.12 use the v1beta2 contract and promote the IPAM API
// from exp/ipam (v1alpha1) to api/ipam (v1beta2). This means neither v0.13 nor
// v0.14 can run against CAPI v1.10 (the last v1beta1 release). Therefore:
//   - The v0.13 upgrade test uses CAPI v1.11 as the starting point.
//   - The v0.14 upgrade test uses CAPI v1.12 as the starting point.

// orcInitVersion is the ORC version installed alongside the old CAPO release in
// PreInit. v1.0.2 is the version the v0.12/v0.13/v0.14 CAPO releases were tested
// against (see the previous InitWithRuntimeExtensionProviders entry).
const orcInitVersion = "v1.0.2"

var _ = Describe("When testing clusterctl upgrades for CAPO (v0.12=>current) and ORC (v1.0.2=>current)[clusterctl-upgrade]", func() {
	BeforeEach(func(ctx context.Context) {
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
			E2EConfig:                       e2eCtx.E2EConfig,
			ClusterctlConfigPath:            e2eCtx.Environment.ClusterctlConfigPath,
			BootstrapClusterProxy:           e2eCtx.Environment.BootstrapClusterProxy,
			ArtifactFolder:                  e2eCtx.Settings.ArtifactFolder,
			SkipCleanup:                     false,
			InitWithBinary:                  "https://github.com/kubernetes-sigs/cluster-api/releases/download/" + capiRelease110 + "/clusterctl-{OS}-{ARCH}",
			InitWithProvidersContract:       "v1beta1",
			InitWithInfrastructureProviders: []string{"openstack:" + capoRelease012},
			InitWithCoreProvider:            "cluster-api:" + capiRelease110,
			InitWithBootstrapProviders:      []string{"kubeadm:" + capiRelease110},
			InitWithControlPlaneProviders:   []string{"kubeadm:" + capiRelease110},
			// Explicit empty slice: we install ORC manually via PreInit/PreUpgrade
			// hooks rather than via clusterctl.
			InitWithRuntimeExtensionProviders: []string{},
			MgmtFlavor:                        shared.FlavorDefault,
			WorkloadFlavor:                    shared.FlavorCapiV1Beta1,
			InitWithKubernetesVersion:         e2eCtx.E2EConfig.MustGetVariable(shared.KubernetesKindVersion),
			UseKindForManagementCluster:       true,
			// Install ORC v1.0.2 before clusterctl init
			PreInit: func(managementClusterProxy capi_framework.ClusterProxy) {
				installORC(context.Background(), managementClusterProxy, orcInitVersion)
			},
			// Upgrade ORC to the current version before clusterctl upgrade
			PreUpgrade: func(managementClusterProxy capi_framework.ClusterProxy) {
				installLatestORC(context.Background(), managementClusterProxy, e2eCtx.E2EConfig)
			},
		}
	})
})

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
			UseKindForManagementCluster:       true,
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
			UseKindForManagementCluster:       true,
		}
	})
})

// installLatestORC downloads and applies the install manifest for the current/latest version of
// the OpenStack Resource Controller (ORC) to the management cluster.
//
// The ORC version is derived from the e2e config (the latest version with contract v1beta2), so no
// version hardcoding is required here — updating the provider entry in the e2e config is sufficient.
func installLatestORC(ctx context.Context, proxy capi_framework.ClusterProxy, e2eConfig *clusterctl.E2EConfig) {
	// GetProviderLatestVersionsByContract returns strings in the format "provider-name:version",
	// e.g. "openstack-resource-controller:v2.5.0".
	orcVersionStrings := e2eConfig.GetProviderLatestVersionsByContract("v1beta2", "openstack-resource-controller")
	Expect(orcVersionStrings).ToNot(BeEmpty(),
		"No ORC version with v1beta2 contract found in e2e config; cannot install ORC for upgrade")

	parts := strings.SplitN(orcVersionStrings[0], ":", 2)
	Expect(parts).To(HaveLen(2),
		"Unexpected ORC provider version string format (expected 'name:version'): %q", orcVersionStrings[0])
	orcVersion := parts[1]

	installORC(ctx, proxy, orcVersion)
}

// installORC downloads and applies the upstream install manifest for the given version of the
// OpenStack Resource Controller (ORC) to the management cluster.
func installORC(ctx context.Context, proxy capi_framework.ClusterProxy, orcVersion string) {
	By(fmt.Sprintf("Installing ORC %s on the management cluster", orcVersion))

	orcInstallURL := fmt.Sprintf(
		"https://github.com/k-orc/openstack-resource-controller/releases/download/%s/install.yaml",
		orcVersion,
	)
	By(fmt.Sprintf("Downloading ORC %s install manifest from %s", orcVersion, orcInstallURL))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, orcInstallURL, http.NoBody)
	Expect(err).ToNot(HaveOccurred(), "Failed to create HTTP request for ORC install manifest")

	resp, err := http.DefaultClient.Do(req) //nolint:bodyclose // closed below via defer
	Expect(err).ToNot(HaveOccurred(),
		"Failed to download ORC %s install manifest from %s", orcVersion, orcInstallURL)
	defer resp.Body.Close()

	Expect(resp.StatusCode).To(Equal(http.StatusOK),
		"Unexpected HTTP status %d when downloading ORC install manifest from %s", resp.StatusCode, orcInstallURL)

	orcManifest, err := io.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred(), "Failed to read ORC install manifest response body")

	By(fmt.Sprintf("Applying ORC %s install manifest to the management cluster", orcVersion))
	Expect(proxy.CreateOrUpdate(ctx, orcManifest)).To(Succeed(),
		"Failed to apply ORC %s install manifest to management cluster", orcVersion)
}
