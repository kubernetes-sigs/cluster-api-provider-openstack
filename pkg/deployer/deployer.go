package deployer

import (
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/clientcmd"
	providerv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/certificates"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// Deployer satisfies the ProviderDeployer(https://github.com/kubernetes-sigs/cluster-api/blob/master/cmd/clusterctl/clusterdeployer/clusterdeployer.go) interface.
type Deployer struct {
}

// New returns a new Deployer.
func New() *Deployer {
	return &Deployer{}
}

// GetIP returns the controlPlaneEndpoint of the Kubernetes cluster, which makes sense in the current
// contexts where this method is called. Seems to be going away anyway with v1alpha2.
func (d *Deployer) GetIP(cluster *clusterv1.Cluster, machine *clusterv1.Machine) (string, error) {
	clusterProviderSpec, err := providerv1.ClusterSpecFromProviderSpec(cluster.Spec.ProviderSpec)
	if err != nil {
		return "", fmt.Errorf("could not get IP: %v", err)
	}
	return clusterProviderSpec.ClusterConfiguration.ControlPlaneEndpoint, nil
}

// GetKubeConfig returns the kubeConfig after the bootstrap process is complete.
func (d *Deployer) GetKubeConfig(cluster *clusterv1.Cluster, master *clusterv1.Machine) (string, error) {

	// Load provider config.
	clusterProviderSpec, err := providerv1.ClusterSpecFromProviderSpec(cluster.Spec.ProviderSpec)
	if err != nil {
		return "", errors.Errorf("failed to load cluster provider status: %v", err)
	}

	cert, err := certificates.DecodeCertPEM(clusterProviderSpec.CAKeyPair.Cert)
	if err != nil {
		return "", errors.Wrap(err, "failed to decode CA Cert")
	} else if cert == nil {
		return "", errors.New("certificate not found in clusterProviderSpec")
	}

	key, err := certificates.DecodePrivateKeyPEM(clusterProviderSpec.CAKeyPair.Key)
	if err != nil {
		return "", errors.Wrap(err, "failed to decode private key")
	} else if key == nil {
		return "", errors.New("key not found in clusterProviderSpec")
	}

	apiServerEndpoint, err := d.GetIP(cluster, master)
	if err != nil {
		return "", err
	}

	cfg, err := certificates.NewKubeconfig(cluster.Name, fmt.Sprintf("https://%s", apiServerEndpoint), cert, key)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate a kubeconfig")
	}

	yaml, err := clientcmd.Write(*cfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to serialize config to yaml")
	}

	return string(yaml), nil
}
