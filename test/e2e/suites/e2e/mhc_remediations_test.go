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
	"k8s.io/utils/pointer"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"

	"sigs.k8s.io/cluster-api-provider-openstack/test/e2e/shared"
)

var _ = Describe("When testing unhealthy machines remediation [mhc-remediations]", func() {
	ctx := context.TODO()
	shared.SetEnvVar("USE_CI_ARTIFACTS", "true", false)
	capi_e2e.MachineRemediationSpec(ctx, func() capi_e2e.MachineRemediationSpecInput {
		return capi_e2e.MachineRemediationSpecInput{
			E2EConfig:             e2eCtx.E2EConfig,
			ClusterctlConfigPath:  e2eCtx.Environment.ClusterctlConfigPath,
			BootstrapClusterProxy: e2eCtx.Environment.BootstrapClusterProxy,
			ArtifactFolder:        e2eCtx.Settings.ArtifactFolder,
			SkipCleanup:           false,
			KCPFlavor:             pointer.String(shared.FlavorKCPRemediation),
			MDFlavor:              pointer.String(shared.FlavorMDRemediation),
		}
	})
})
