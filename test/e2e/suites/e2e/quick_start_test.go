//go:build e2e

/*
Copyright 2024 The Kubernetes Authors.

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
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
	"sigs.k8s.io/cluster-api/test/framework"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
	shared "sigs.k8s.io/cluster-api-provider-openstack/test/e2e/shared"
)

var _ = Describe("When following the Cluster API quick-start with ClusterClass [PR-Blocking] [ClusterClass]", func() {
	capi_e2e.QuickStartSpec(context.TODO(), func() capi_e2e.QuickStartSpecInput {
		return capi_e2e.QuickStartSpecInput{
			E2EConfig:             e2eCtx.E2EConfig,
			ClusterctlConfigPath:  e2eCtx.Environment.ClusterctlConfigPath,
			BootstrapClusterProxy: e2eCtx.Environment.BootstrapClusterProxy,
			ArtifactFolder:        e2eCtx.Settings.ArtifactFolder,
			SkipCleanup:           false,
			Flavor:                ptr.To(shared.FlavorTopology),
			PostMachinesProvisioned: func(proxy framework.ClusterProxy, namespace, clusterName string) {
				shared.Logf("Verifying OpenStackClusterTemplate metadata propagation to OpenStackCluster")
				ctx := context.TODO()
				k8sClient := proxy.GetClient()

				cluster := &clusterv1.Cluster{}
				Expect(k8sClient.Get(ctx, crclient.ObjectKey{Namespace: namespace, Name: clusterName}, cluster)).To(Succeed())

				openStackCluster := &infrav1.OpenStackCluster{}
				Expect(k8sClient.Get(ctx, crclient.ObjectKey{
					Namespace: namespace,
					Name:      cluster.Spec.InfrastructureRef.Name,
				}, openStackCluster)).To(Succeed())

				Expect(openStackCluster.Labels).To(HaveKeyWithValue("capo.e2e.test/template-metadata", "propagated"),
					"labels from OpenStackClusterTemplate spec.template.metadata should be applied to the generated OpenStackCluster")
				Expect(openStackCluster.Annotations).To(HaveKeyWithValue("capo.e2e.test/template-metadata", "propagated"),
					"annotations from OpenStackClusterTemplate spec.template.metadata should be applied to the generated OpenStackCluster")
			},
		}
	})
})
