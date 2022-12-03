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

	. "github.com/onsi/ginkgo"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"

	"sigs.k8s.io/cluster-api-provider-openstack/test/e2e/shared"
)

// Note: There isn't really anything fixing this to the v0.6 release. The test will
// simply pick the latest release with the correct contract from the E2EConfig, as seen here
// https://github.com/kubernetes-sigs/cluster-api/blob/3abb9089485f51d46054096c17ccb8b59f0f7331/test/e2e/clusterctl_upgrade.go#L265
// When new minor releases are added (with the same contract) we will need to work on this
// if we want to continue testing v0.6.
var _ = Describe("When testing clusterctl upgrades (v0.6=>current) [clusterctl-upgrade]", func() {
	ctx := context.TODO()
	shared.SetEnvVar("USE_CI_ARTIFACTS", "true", false)
	shared.SetEnvVar("DOWNLOAD_E2E_IMAGE", "true", false)
	shared.SetEnvVar("E2E_IMAGE_URL", "http://10.0.3.15/capo-e2e-image.tar", false)

	capi_e2e.ClusterctlUpgradeSpec(ctx, func() capi_e2e.ClusterctlUpgradeSpecInput {
		return capi_e2e.ClusterctlUpgradeSpecInput{
			E2EConfig:                 e2eCtx.E2EConfig,
			ClusterctlConfigPath:      e2eCtx.Environment.ClusterctlConfigPath,
			BootstrapClusterProxy:     e2eCtx.Environment.BootstrapClusterProxy,
			ArtifactFolder:            e2eCtx.Settings.ArtifactFolder,
			SkipCleanup:               false,
			InitWithBinary:            "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.3.0/clusterctl-{OS}-{ARCH}",
			InitWithProvidersContract: "v1beta1",
			MgmtFlavor:                shared.FlavorDefault,
			WorkloadFlavor:            shared.FlavorV1alpha5,
		}
	})
})
