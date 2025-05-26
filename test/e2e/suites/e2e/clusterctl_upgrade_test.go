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

	. "github.com/onsi/ginkgo/v2"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"

	"sigs.k8s.io/cluster-api-provider-openstack/test/e2e/shared"
)

const OldCAPIVersion = "v1.6.0"

var _ = Describe("When testing clusterctl upgrades (v0.10=>current) [clusterctl-upgrade]", func() {
	BeforeEach(func(ctx context.Context) {
		shared.ApplyCoreImagesPlus(ctx, e2eCtx, upgradeImage)

		// The upgrade job can't use image references in its machine templates,
		// so we must wait for the images to be available before running the
		// test.
		coreImages := shared.CoreImages(e2eCtx)
		allImages := append(coreImages, upgradeImage)
		shared.WaitForGlanceImagesAvailable(ctx, e2eCtx, allImages)

		setDownloadE2EImageEnvVar()
	})

	capi_e2e.ClusterctlUpgradeSpec(context.TODO(), func() capi_e2e.ClusterctlUpgradeSpecInput {
		return capi_e2e.ClusterctlUpgradeSpecInput{
			E2EConfig:                       e2eCtx.E2EConfig,
			ClusterctlConfigPath:            e2eCtx.Environment.ClusterctlConfigPath,
			BootstrapClusterProxy:           e2eCtx.Environment.BootstrapClusterProxy,
			ArtifactFolder:                  e2eCtx.Settings.ArtifactFolder,
			SkipCleanup:                     false,
			InitWithBinary:                  "https://github.com/kubernetes-sigs/cluster-api/releases/download/" + OldCAPIVersion + "/clusterctl-{OS}-{ARCH}",
			InitWithProvidersContract:       "v1beta1",
			InitWithInfrastructureProviders: []string{"openstack:v0.10.8"},
			InitWithCoreProvider:            "cluster-api:" + OldCAPIVersion,
			InitWithBootstrapProviders:      []string{"kubeadm:" + OldCAPIVersion},
			InitWithControlPlaneProviders:   []string{"kubeadm:" + OldCAPIVersion},
			MgmtFlavor:                      shared.FlavorDefault,
			WorkloadFlavor:                  shared.FlavorWithoutORC,
			InitWithKubernetesVersion:       e2eCtx.E2EConfig.GetVariable(shared.KubernetesVersion),
		}
	})
})
