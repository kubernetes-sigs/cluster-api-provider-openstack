package provider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CloudsSecretKey = "clouds.yaml"
	CaSecretKey     = "cacert"
)

func NewClientFromMachine(ctrlClient client.Client, openStackMachine *infrav1.OpenStackMachine) (*gophercloud.ProviderClient, *clientconfig.ClientOpts, error) {
	var cloud clientconfig.Cloud
	var caCert []byte

	if openStackMachine.Spec.CloudsSecret != nil && openStackMachine.Spec.CloudsSecret.Name != "" {
		namespace := openStackMachine.Spec.CloudsSecret.Namespace
		if namespace == "" {
			namespace = openStackMachine.Namespace
		}
		var err error
		cloud, caCert, err = getCloudFromSecret(ctrlClient, namespace, openStackMachine.Spec.CloudsSecret.Name, openStackMachine.Spec.CloudName)
		if err != nil {
			return nil, nil, err
		}
	}
	return newClient(cloud, caCert)
}

func NewClientFromCluster(ctrlClient client.Client, openStackCluster *infrav1.OpenStackCluster) (*gophercloud.ProviderClient, *clientconfig.ClientOpts, error) {
	var cloud clientconfig.Cloud
	var caCert []byte

	if openStackCluster.Spec.CloudsSecret != nil && openStackCluster.Spec.CloudsSecret.Name != "" {
		namespace := openStackCluster.Spec.CloudsSecret.Namespace
		if namespace == "" {
			namespace = openStackCluster.Namespace
		}
		var err error
		cloud, caCert, err = getCloudFromSecret(ctrlClient, namespace, openStackCluster.Spec.CloudsSecret.Name, openStackCluster.Spec.CloudName)
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
func getCloudFromSecret(ctrlClient client.Client, secretNamespace string, secretName string, cloudName string) (clientconfig.Cloud, []byte, error) {
	ctx := context.TODO()
	emptyCloud := clientconfig.Cloud{}

	if secretName == "" {
		return emptyCloud, nil, nil
	}

	if cloudName == "" {
		return emptyCloud, nil, fmt.Errorf("secret name set to %v but no cloud was specified. Please set cloud_name in your machine spec", secretName)
	}

	secret := &v1.Secret{}
	err := ctrlClient.Get(ctx, types.NamespacedName{
		Namespace: secretNamespace,
		Name:      secretName,
	}, secret)
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
