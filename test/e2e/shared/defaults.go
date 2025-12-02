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

package shared

import (
	"errors"
	"flag"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"

	infrav1alpha1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

const (
	DefaultSSHKeyPairName      = "cluster-api-provider-openstack-sigs-k8s-io"
	KubeContext                = "KUBE_CONTEXT"
	KubernetesVersion          = "KUBERNETES_VERSION"
	CCMPath                    = "CCM"
	CCMResources               = "CCM_RESOURCES"
	OpenStackBastionFlavorAlt  = "OPENSTACK_BASTION_MACHINE_FLAVOR_ALT"
	OpenStackCloudYAMLFile     = "OPENSTACK_CLOUD_YAML_FILE"
	OpenStackCloud             = "OPENSTACK_CLOUD"
	OpenStackCloudCACertB64    = "OPENSTACK_CLOUD_CACERT_B64"
	OpenStackCloudAdmin        = "OPENSTACK_CLOUD_ADMIN"
	OpenStackFailureDomain     = "OPENSTACK_FAILURE_DOMAIN" //nolint:gosec // Linter thinks this could be credentials...
	OpenStackFailureDomainAlt  = "OPENSTACK_FAILURE_DOMAIN_ALT"
	OpenStackVolumeTypeAlt     = "OPENSTACK_VOLUME_TYPE_ALT"
	OpenStackImageName         = "OPENSTACK_IMAGE_NAME"
	OpenStackNodeMachineFlavor = "OPENSTACK_NODE_MACHINE_FLAVOR"
	SSHUserMachine             = "SSH_USER_MACHINE"
	FlavorDefault              = ""
	FlavorNoBastion            = "no-bastion"
	FlavorWithoutLB            = "without-lb"
	FlavorMultiNetwork         = "multi-network"
	FlavorMultiAZ              = "multi-az"
	FlavorMDRemediation        = "md-remediation"
	FlavorKCPRemediation       = "kcp-remediation"
	FlavorFlatcar              = "flatcar"
	FlavorKubernetesUpgrade    = "k8s-upgrade"
	FlavorFlatcarSysext        = "flatcar-sysext"
	FlavorHealthMonitor        = "health-monitor"
)

// DefaultScheme returns the default scheme to use for testing.
func DefaultScheme() *runtime.Scheme {
	sc := runtime.NewScheme()
	framework.TryAddDefaultSchemes(sc)

	err := errors.Join(
		clusterv1beta1.AddToScheme(sc),
		orcv1alpha1.AddToScheme(sc),
		infrav1alpha1.AddToScheme(sc),
		infrav1.AddToScheme(sc),
		clientgoscheme.AddToScheme(sc),
	)
	if err != nil {
		panic("error adding types to scheme: " + err.Error())
	}
	return sc
}

// CreateDefaultFlags will create the default flags used for the tests and binds them to the e2e context.
func CreateDefaultFlags(ctx *E2EContext) {
	flag.StringVar(&ctx.Settings.ConfigPath, "config-path", "", "path to the e2e config file")
	flag.StringVar(&ctx.Settings.ArtifactFolder, "artifacts-folder", "", "folder where e2e test artifact should be stored")
	flag.BoolVar(&ctx.Settings.UseCIArtifacts, "kubetest.use-ci-artifacts", false, "use the latest build from the main branch of the Kubernetes repository")
	flag.StringVar(&ctx.Settings.KubetestConfigFilePath, "kubetest.config-file", "", "path to the kubetest configuration file")
	flag.IntVar(&ctx.Settings.GinkgoNodes, "kubetest.ginkgo-nodes", 1, "number of ginkgo nodes to use")
	flag.IntVar(&ctx.Settings.GinkgoSlowSpecThreshold, "kubetest.ginkgo-slowSpecThreshold", 120, "time in s before spec is marked as slow")
	flag.BoolVar(&ctx.Settings.UseExistingCluster, "use-existing-cluster", false, "if true, the test will try to use an existing cluster and fallback to create a new one if it couldn't be found")
	flag.BoolVar(&ctx.Settings.SkipCleanup, "skip-cleanup", false, "if true, the resource cleanup after tests will be skipped")
	flag.StringVar(&ctx.Settings.DataFolder, "data-folder", "", "path to the data folder")
	flag.BoolVar(&ctx.Settings.Debug, "debug", false, "enables the debug log")
}
