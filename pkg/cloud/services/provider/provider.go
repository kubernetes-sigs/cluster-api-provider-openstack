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
	osclient "github.com/gophercloud/utils/client"
	"github.com/gophercloud/utils/openstack/clientconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha5"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/metrics"
)

const (
	cloudsSecretKey = "clouds.yaml"
	caSecretKey     = "cacert"
)

func NewClientFromMachine(ctx context.Context, ctrlClient client.Client, openStackMachine *infrav1.OpenStackMachine) (*gophercloud.ProviderClient, *clientconfig.ClientOpts, error) {
	var cloud clientconfig.Cloud
	var caCert []byte

	if openStackMachine.Spec.IdentityRef != nil {
		var err error
		cloud, caCert, err = getCloudFromSecret(ctx, ctrlClient, openStackMachine.Namespace, openStackMachine.Spec.IdentityRef.Name, openStackMachine.Spec.CloudName)
		if err != nil {
			return nil, nil, err
		}
	}
	return NewClient(cloud, caCert)
}

func NewClientFromCluster(ctx context.Context, ctrlClient client.Client, openStackCluster *infrav1.OpenStackCluster) (*gophercloud.ProviderClient, *clientconfig.ClientOpts, error) {
	var cloud clientconfig.Cloud
	var caCert []byte

	if openStackCluster.Spec.IdentityRef != nil {
		var err error
		cloud, caCert, err = getCloudFromSecret(ctx, ctrlClient, openStackCluster.Namespace, openStackCluster.Spec.IdentityRef.Name, openStackCluster.Spec.CloudName)
		if err != nil {
			return nil, nil, err
		}
	}
	return NewClient(cloud, caCert)
}

func NewClient(cloud clientconfig.Cloud, caCert []byte) (*gophercloud.ProviderClient, *clientconfig.ClientOpts, error) {
	clientOpts := new(clientconfig.ClientOpts)
	if cloud.AuthInfo != nil {
		clientOpts.AuthInfo = cloud.AuthInfo
		clientOpts.AuthType = cloud.AuthType
		clientOpts.RegionName = cloud.RegionName
	}

	opts, err := clientconfig.AuthOptions(clientOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("auth option failed for cloud %v: %v", cloud.Cloud, err)
	}
	opts.AllowReauth = true

	provider, err := openstack.NewClient(opts.IdentityEndpoint)
	if err != nil {
		return nil, nil, fmt.Errorf("create providerClient err: %v", err)
	}

	config := &tls.Config{
		RootCAs:    x509.NewCertPool(),
		MinVersion: tls.VersionTLS12,
	}
	if cloud.Verify != nil {
		config.InsecureSkipVerify = !*cloud.Verify
	}
	if caCert != nil {
		config.RootCAs.AppendCertsFromPEM(caCert)
	}

	provider.HTTPClient.Transport = &http.Transport{Proxy: http.ProxyFromEnvironment, TLSClientConfig: config}
	if klog.V(6).Enabled() {
		provider.HTTPClient.Transport = &osclient.RoundTripper{
			Rt:     provider.HTTPClient.Transport,
			Logger: &defaultLogger{},
		}
	}
	err = openstack.Authenticate(provider, *opts)
	if err != nil {
		return nil, nil, fmt.Errorf("providerClient authentication err: %v", err)
	}
	return provider, clientOpts, nil
}

type defaultLogger struct{}

// Printf is a default Printf method.
func (defaultLogger) Printf(format string, args ...interface{}) {
	klog.V(6).Infof(format, args...)
}

// getCloudFromSecret extract a Cloud from the given namespace:secretName.
func getCloudFromSecret(ctx context.Context, ctrlClient client.Client, secretNamespace string, secretName string, cloudName string) (clientconfig.Cloud, []byte, error) {
	emptyCloud := clientconfig.Cloud{}

	if secretName == "" {
		return emptyCloud, nil, nil
	}

	if cloudName == "" {
		return emptyCloud, nil, fmt.Errorf("secret name set to %v but no cloud was specified. Please set cloud_name in your machine spec", secretName)
	}

	secret := &corev1.Secret{}
	err := ctrlClient.Get(ctx, types.NamespacedName{
		Namespace: secretNamespace,
		Name:      secretName,
	}, secret)
	if err != nil {
		return emptyCloud, nil, err
	}

	content, ok := secret.Data[cloudsSecretKey]
	if !ok {
		return emptyCloud, nil, fmt.Errorf("OpenStack credentials secret %v did not contain key %v",
			secretName, cloudsSecretKey)
	}
	var clouds clientconfig.Clouds
	if err = yaml.Unmarshal(content, &clouds); err != nil {
		return emptyCloud, nil, fmt.Errorf("failed to unmarshal clouds credentials stored in secret %v: %v", secretName, err)
	}

	// get caCert
	caCert, ok := secret.Data[caSecretKey]
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
	mc := metrics.NewMetricPrometheusContext("project", "get")
	resp, err := c.Get(c.ServiceURL("auth", "projects"), &jsonResp, &gophercloud.RequestOpts{OkCodes: []int{200}})
	if mc.ObserveRequest(err) != nil {
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
