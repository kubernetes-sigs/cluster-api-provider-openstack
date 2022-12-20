//go:build e2e
// +build e2e

/*
Copyright 2022 The Kubernetes Authors.

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
	capie2e "sigs.k8s.io/cluster-api/test/e2e"

	"sigs.k8s.io/cluster-api-provider-openstack/test/e2e/shared"
)

var _ = Describe("When testing Cluster API provider Openstack working on [self-hosted] clusters", func() {
	ctx := context.TODO()
	shared.SetEnvVar("USE_CI_ARTIFACTS", "true", false)
	shared.SetEnvVar("DOWNLOAD_E2E_IMAGE", "true", false)
	// TODO(lentzi90): Since we currently rely on USE_CI_ARTIFACTS to get kubernetes set up,
	// we are stuck with one version only and cannot do the k8s upgrade that is part of the
	// SelfHostedSpec. The reason for this is that the script used to install kubernetes tools comes
	// with the KubeadmControlPlane and that does not change when upgrading the k8s version
	// (only the machineTemplate changes).
	// Until we have a way to work around this, the k8s upgrade test is disabled.
	capie2e.SelfHostedSpec(ctx, func() capie2e.SelfHostedSpecInput {
		return capie2e.SelfHostedSpecInput{
			E2EConfig:             e2eCtx.E2EConfig,
			ClusterctlConfigPath:  e2eCtx.Environment.ClusterctlConfigPath,
			BootstrapClusterProxy: e2eCtx.Environment.BootstrapClusterProxy,
			ArtifactFolder:        e2eCtx.Settings.ArtifactFolder,
			SkipUpgrade:           true,
			SkipCleanup:           false,
			Flavor:                shared.FlavorDefault,
		}
	})
})
