package cluster

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	"gopkg.in/yaml.v2"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"k8s.io/klog"
	providerv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	providerv1openstack "sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/clients"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	CloudsSecretKey = "clouds.yaml"
	CaSecretKey     = "cacert"
)

// Actuator controls cluster related infrastructure.
type Actuator struct {
	params providerv1openstack.ActuatorParams
}

// NewActuator creates a new Actuator
func NewActuator(params providerv1openstack.ActuatorParams) (*Actuator, error) {
	res := &Actuator{params: params}
	return res, nil
}

// Reconcile creates or applies updates to the cluster.
func (a *Actuator) Reconcile(cluster *clusterv1.Cluster) error {
	klog.Infof("Reconciling cluster %v.", cluster.Name)
	clusterName := fmt.Sprintf("%s-%s", cluster.Namespace, cluster.Name)

	client, err := a.getNetworkClient(cluster)
	if err != nil {
		return err
	}
	networkService, err := clients.NewNetworkService(client)
	if err != nil {
		return err
	}

	secGroupService, err := clients.NewSecGroupService(client)
	if err != nil {
		return err
	}

	// Load provider config.
	desired, err := providerv1.ClusterSpecFromProviderSpec(cluster.Spec.ProviderSpec)
	if err != nil {
		return errors.Errorf("failed to load cluster provider spec: %v", err)
	}

	// Load provider status.
	status, err := providerv1.ClusterStatusFromProviderStatus(cluster.Status.ProviderStatus)
	if err != nil {
		return errors.Errorf("failed to load cluster provider status: %v", err)
	}

	err = networkService.Reconcile(clusterName, *desired, status)
	if err != nil {
		return errors.Errorf("failed to reconcile network: %v", err)
	}

	err = secGroupService.Reconcile(clusterName, *desired, status)
	if err != nil {
		return errors.Errorf("failed to reconcile security groups: %v", err)
	}
	defer func() {
		if err := a.storeClusterStatus(cluster, status); err != nil {
			klog.Errorf("failed to store provider status for cluster %q in namespace %q: %v", cluster.Name, cluster.Namespace, err)
		}
	}()
	return nil
}

// Delete deletes a cluster and is invoked by the Cluster Controller
func (a *Actuator) Delete(cluster *clusterv1.Cluster) error {
	klog.Infof("Deleting cluster %v.", cluster.Name)

	client, err := a.getNetworkClient(cluster)
	if err != nil {
		return err
	}
	_, err = clients.NewNetworkService(client)
	if err != nil {
		return err
	}

	secGroupService, err := clients.NewSecGroupService(client)
	if err != nil {
		return err
	}

	// Load provider config.
	_, err = providerv1.ClusterSpecFromProviderSpec(cluster.Spec.ProviderSpec)
	if err != nil {
		return errors.Errorf("failed to load cluster provider config: %v", err)
	}

	// Load provider status.
	providerStatus, err := providerv1.ClusterStatusFromProviderStatus(cluster.Status.ProviderStatus)
	if err != nil {
		return errors.Errorf("failed to load cluster provider status: %v", err)
	}

	// Delete other things

	if providerStatus.GlobalSecurityGroup != nil {
		klog.Infof("Deleting global security group %q", providerStatus.GlobalSecurityGroup.Name)
		err := secGroupService.Delete(providerStatus.GlobalSecurityGroup)
		if err != nil {
			return errors.Errorf("failed to delete security group: %v", err)
		}
	}

	if providerStatus.ControlPlaneSecurityGroup != nil {
		klog.Infof("Deleting control plane security group %q", providerStatus.ControlPlaneSecurityGroup.Name)
		err := secGroupService.Delete(providerStatus.ControlPlaneSecurityGroup)
		if err != nil {
			return errors.Errorf("failed to delete security group: %v", err)
		}
	}

	return nil
}

func (a *Actuator) storeClusterStatus(cluster *clusterv1.Cluster, status *providerv1.OpenstackClusterProviderStatus) error {
	ext, err := providerv1.EncodeClusterStatus(status)
	if err != nil {
		return fmt.Errorf("failed to update cluster status for cluster %q in namespace %q: %v", cluster.Name, cluster.Namespace, err)
	}
	cluster.Status.ProviderStatus = ext

	statusClient := a.params.Client.Status()
	if err := statusClient.Update(context.Background(), cluster); err != nil {
		return fmt.Errorf("failed to update cluster status for cluster %q in namespace %q: %v", cluster.Name, cluster.Namespace, err)
	}

	return nil
}

func GetCloudFromSecret(kubeClient kubernetes.Interface, namespace string, secretName string, cloudName string) (clientconfig.Cloud, []byte, error) {
	emptyCloud := clientconfig.Cloud{}

	if secretName == "" {
		return emptyCloud, nil, nil
	}

	if secretName != "" && cloudName == "" {
		return emptyCloud, nil, fmt.Errorf("Secret name set to %v but no cloud was specified. Please set cloud_name in your machine spec.", secretName)
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

	// get cacert
	cacert, ok := secret.Data[CaSecretKey]
	if !ok {
		return emptyCloud, nil, err
	}

	return clouds.Clouds[cloudName], cacert, nil
}

// getNetworkClient returns an gophercloud.ServiceClient provided by openstack.NewNetworkV2
// TODO(chrigl) currently ignoring cluster, but in the future we might store OS-Credentials
// as secrets referenced by the cluster.
// See https://github.com/kubernetes-sigs/cluster-api-provider-openstack/pull/136
func (a *Actuator) getNetworkClient(cluster *clusterv1.Cluster) (*gophercloud.ServiceClient, error) {
	kubeClient := a.params.KubeClient
	clusterSpec, err := providerv1.ClusterSpecFromProviderSpec(cluster.Spec.ProviderSpec)
	if err != nil {
		return nil, errors.Errorf("failed to load cluster provider spec: %v", err)
	}
	cloud := clientconfig.Cloud{}
	var cacert []byte

	if clusterSpec.CloudsSecret != nil && clusterSpec.CloudsSecret.Name != "" {
		namespace := clusterSpec.CloudsSecret.Namespace
		if namespace == "" {
			namespace = cluster.Namespace
		}
		cloud, cacert, err = GetCloudFromSecret(kubeClient, namespace, clusterSpec.CloudsSecret.Name, clusterSpec.CloudName)
		if err != nil {
			return nil, err
		}
	}

	clientOpts := new(clientconfig.ClientOpts)
	var opts *gophercloud.AuthOptions

	if cloud.AuthInfo != nil {
		clientOpts.AuthInfo = cloud.AuthInfo
		clientOpts.AuthType = cloud.AuthType
		clientOpts.Cloud = cloud.Cloud
		clientOpts.RegionName = cloud.RegionName
	}

	opts, err = clientconfig.AuthOptions(clientOpts)
	if err != nil {
		return nil, err
	}

	opts.AllowReauth = true

	provider, err := openstack.NewClient(opts.IdentityEndpoint)
	if err != nil {
		return nil, fmt.Errorf("create providerClient err: %v", err)
	}

	config := &tls.Config{}
	cloudFromYaml, err := clientconfig.GetCloudFromYAML(clientOpts)
	if cloudFromYaml.CACertFile != "" {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(cacert)
		config.RootCAs = caCertPool
	}

	config.InsecureSkipVerify = *cloudFromYaml.Verify
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment, TLSClientConfig: config}
	provider.HTTPClient.Transport = transport

	err = openstack.Authenticate(provider, *opts)
	if err != nil {
		return nil, fmt.Errorf("providerClient authentication err: %v", err)
	}

	client, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}
