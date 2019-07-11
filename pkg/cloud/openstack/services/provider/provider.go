package provider

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
	providerv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	CloudsSecretKey = "clouds.yaml"
	CaSecretKey     = "cacert"
)

func NewClientFromMachine(kubeClient kubernetes.Interface, machine *clusterv1.Machine) (*gophercloud.ProviderClient, *clientconfig.ClientOpts, error) {
	machineProviderSpec, err := providerv1.MachineSpecFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, nil, errors.Errorf("failed to load machine provider spec: %v", err)
	}
	var cloud clientconfig.Cloud
	var caCert []byte

	if machineProviderSpec.CloudsSecret != nil && machineProviderSpec.CloudsSecret.Name != "" {
		namespace := machineProviderSpec.CloudsSecret.Namespace
		if namespace == "" {
			namespace = machine.Namespace
		}
		cloud, caCert, err = getCloudFromSecret(kubeClient, namespace, machineProviderSpec.CloudsSecret.Name, machineProviderSpec.CloudName)
		if err != nil {
			return nil, nil, err
		}
	}
	return newClient(cloud, caCert)
}

func NewClientFromCluster(kubeClient kubernetes.Interface, cluster *clusterv1.Cluster) (*gophercloud.ProviderClient, *clientconfig.ClientOpts, error) {
	clusterProviderSpec, err := providerv1.ClusterSpecFromProviderSpec(cluster.Spec.ProviderSpec)
	if err != nil {
		return nil, nil, errors.Errorf("failed to load cluster provider spec: %v", err)
	}
	var cloud clientconfig.Cloud
	var caCert []byte

	if clusterProviderSpec.CloudsSecret != nil && clusterProviderSpec.CloudsSecret.Name != "" {
		namespace := clusterProviderSpec.CloudsSecret.Namespace
		if namespace == "" {
			namespace = cluster.Namespace
		}
		cloud, caCert, err = getCloudFromSecret(kubeClient, namespace, clusterProviderSpec.CloudsSecret.Name, clusterProviderSpec.CloudName)
		if err != nil {
			return nil, nil, err
		}
	}
	return newClient(cloud, caCert)
}

func newClient(cloud clientconfig.Cloud, caCert []byte) (*gophercloud.ProviderClient, *clientconfig.ClientOpts, error) {
	clientOpts := new(clientconfig.ClientOpts)
	if cloud.AuthInfo != nil {
		clientOpts.AuthInfo = cloud.AuthInfo
		clientOpts.AuthType = cloud.AuthType
		clientOpts.Cloud = cloud.Cloud
		clientOpts.RegionName = cloud.RegionName
	}

	opts, err := clientconfig.AuthOptions(clientOpts)
	if err != nil {
		return nil, nil, err
	}
	opts.AllowReauth = true

	provider, err := openstack.NewClient(opts.IdentityEndpoint)
	if err != nil {
		return nil, nil, fmt.Errorf("create providerClient err: %v", err)
	}

	config := &tls.Config{}
	cloudFromYaml, err := clientconfig.GetCloudFromYAML(clientOpts)
	if cloudFromYaml != nil {
		if cloudFromYaml.CACertFile != "" {
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			config.RootCAs = caCertPool
		}
		config.InsecureSkipVerify = !*cloudFromYaml.Verify
	}

	transport := &http.Transport{Proxy: http.ProxyFromEnvironment, TLSClientConfig: config}
	provider.HTTPClient.Transport = transport

	err = openstack.Authenticate(provider, *opts)
	if err != nil {
		return nil, nil, fmt.Errorf("providerClient authentication err: %v", err)
	}
	return provider, clientOpts, nil
}

// getCloudFromSecret extract a Cloud from the given namespace:secretName
func getCloudFromSecret(kubeClient kubernetes.Interface, namespace string, secretName string, cloudName string) (clientconfig.Cloud, []byte, error) {
	emptyCloud := clientconfig.Cloud{}

	if secretName == "" {
		return emptyCloud, nil, nil
	}

	if cloudName == "" {
		return emptyCloud, nil, fmt.Errorf("secret name set to %v but no cloud was specified. Please set cloud_name in your machine spec", secretName)
	}

	secret, err := kubeClient.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return emptyCloud, nil, err
	}

	content, ok := secret.Data[CloudsSecretKey]
	if !ok {
		return emptyCloud, nil, fmt.Errorf("OpenStack credentials secret %v did not contain key %v",
			secretName, CloudsSecretKey)
	}
	var clouds clientconfig.Clouds
	err = yaml.Unmarshal(content, &clouds)
	if err != nil {
		return emptyCloud, nil, fmt.Errorf("failed to unmarshal clouds credentials stored in secret %v: %v", secretName, err)
	}

	// get caCert
	caCert, ok := secret.Data[CaSecretKey]
	if !ok {
		return emptyCloud, nil, err
	}

	return clouds.Clouds[cloudName], caCert, nil
}
