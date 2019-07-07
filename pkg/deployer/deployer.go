package deployer

import (
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	providerv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	constants "sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/contants"
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

// GetIP returns the IP of a machine, but this is going away.
func (d *Deployer) GetIP(cluster *clusterv1.Cluster, machine *clusterv1.Machine) (string, error) {
	if machine.ObjectMeta.Annotations != nil {
		if ip, ok := machine.ObjectMeta.Annotations[constants.OpenstackIPAnnotationKey]; ok {
			clusterProviderSpec, err := providerv1.ClusterSpecFromProviderSpec(cluster.Spec.ProviderSpec)
			if err != nil {
				return "", fmt.Errorf("could not get IP: %v", err)
			}
			var endpoint string
			if clusterProviderSpec.ManagedAPIServerLoadBalancer {
				endpoint = fmt.Sprintf("%s:%d", ip, clusterProviderSpec.APIServerLoadBalancerPort)
			} else {
				// TODO: replace hardcoded port 443 as soon as controlPlaneEndpoint is specified via
				// ClusterConfiguration (in Cluster CRD)
				endpoint = fmt.Sprintf("%s:443", ip)
			}
			klog.Infof("Returning endpoint from machine annotation %s", endpoint)
			return endpoint, nil
		}
	}

	return "", errors.New("could not get IP")
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

	var apiServerEndpoint string
	if clusterProviderSpec.ManagedAPIServerLoadBalancer {
		apiServerEndpoint = fmt.Sprintf("https://%s:%d", clusterProviderSpec.APIServerLoadBalancerFloatingIP, clusterProviderSpec.APIServerLoadBalancerPort)
	} else {
		var endpoint string
		if master != nil {
			endpoint, err = d.GetIP(cluster, master)
			if err != nil {
				return "", err
			}
		} else {
			// This case means no master has been created yet, we need get from cluster info anyway
			if len(clusterProviderSpec.MasterIP) == 0 {
				return "", errors.New("MasterIP in cluster spec not set")
			}
			endpoint = clusterProviderSpec.MasterIP
		}
		apiServerEndpoint = fmt.Sprintf("https://%s", endpoint)
	}

	cfg, err := certificates.NewKubeconfig(cluster.Name, apiServerEndpoint, cert, key)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate a kubeconfig")
	}

	yaml, err := clientcmd.Write(*cfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to serialize config to yaml")
	}

	return string(yaml), nil
}
