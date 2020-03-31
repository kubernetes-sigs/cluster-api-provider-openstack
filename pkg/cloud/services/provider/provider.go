/*
Copyright 2020 The Kubernetes Authors.

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

package provider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha3"
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

	config := &tls.Config{
		RootCAs: x509.NewCertPool(),
	}
	if cloud.Verify != nil {
		config.InsecureSkipVerify = !*cloud.Verify
	}
	if caCert != nil {
		config.RootCAs.AppendCertsFromPEM(caCert)
	}

	provider.HTTPClient.Transport = &http.Transport{Proxy: http.ProxyFromEnvironment, TLSClientConfig: config}
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
		return clouds.Clouds[cloudName], nil, nil
	}

	return clouds.Clouds[cloudName], caCert, nil
}

type project struct {
	ID   string `json:"id"`
	Name string
}

type projects struct {
	Projects []project `json:"projects"`
}

func GetProjectID(client *gophercloud.ProviderClient, name string) (string, error) {
	c, err := openstack.NewIdentityV3(client, gophercloud.EndpointOpts{})
	if err != nil {
		return "", fmt.Errorf("failed to create identity service client: %v", err)
	}

	jsonResp := projects{}
	resp, err := c.Get(c.ServiceURL("auth", "projects"), &jsonResp, &gophercloud.RequestOpts{OkCodes: []int{200}})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	for _, project := range jsonResp.Projects {
		if project.Name == name {
			return project.ID, nil
		}
	}
	return "", fmt.Errorf("project %s not found", name)
}
