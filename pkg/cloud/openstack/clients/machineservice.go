/*
Copyright 2018 The Kubernetes Authors.

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

package clients

import (
	"encoding/base64"
	"fmt"

	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/floatingips"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/gophercloud/utils/openstack/clientconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	openstackconfigv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const CloudsSecretKey = "clouds.yaml"

type InstanceService struct {
	provider       *gophercloud.ProviderClient
	computeClient  *gophercloud.ServiceClient
	identityClient *gophercloud.ServiceClient
	networkClient  *gophercloud.ServiceClient
}

type Instance struct {
	servers.Server
}

type InstanceListOpts struct {
	// Name of the image in URL format.
	Image string `q:"image"`

	// Name of the flavor in URL format.
	Flavor string `q:"flavor"`

	// Name of the server as a string; can be queried with regular expressions.
	// Realize that ?name=bob returns both bob and bobb. If you need to match bob
	// only, you can use a regular expression matching the syntax of the
	// underlying database server implemented for Compute.
	Name string `q:"name"`
}

func GetCloudFromSecret(kubeClient kubernetes.Interface, namespace string, secretName string, cloudName string) (clientconfig.Cloud, error) {
	emptyCloud := clientconfig.Cloud{}

	if secretName == "" {
		return emptyCloud, nil
	}

	if secretName != "" && cloudName == "" {
		return emptyCloud, fmt.Errorf("Secret name set to %v but no cloud was specified. Please set cloud_name in your machine spec.", secretName)
	}

	secret, err := kubeClient.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return emptyCloud, err
	}

	content, ok := secret.Data[CloudsSecretKey]
	if !ok {
		return emptyCloud, fmt.Errorf("OpenStack credentials secret %v did not contain key %v",
			secretName, CloudsSecretKey)
	}
	var clouds clientconfig.Clouds
	err = yaml.Unmarshal(content, &clouds)
	if err != nil {
		return emptyCloud, fmt.Errorf("failed to unmarshal clouds credentials stored in secret %v: %v", secretName, err)
	}

	return clouds.Clouds[cloudName], nil
}

// TODO: Eventually we'll have a NewInstanceServiceFromCluster too
func NewInstanceServiceFromMachine(kubeClient kubernetes.Interface, machine *clusterv1.Machine) (*InstanceService, error) {
	machineSpec, err := openstackconfigv1.MachineSpecFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, err
	}
	namespace := machineSpec.CloudsSecret.Namespace
	if namespace == "" {
		namespace = machine.Namespace
	}
	cloud, err := GetCloudFromSecret(kubeClient, namespace, machineSpec.CloudsSecret.Name, machineSpec.CloudName)
	if err != nil {
		return nil, err
	}
	return NewInstanceServiceFromCloud(cloud)
}

func NewInstanceService() (*InstanceService, error) {
	cloud := clientconfig.Cloud{}
	return NewInstanceServiceFromCloud(cloud)
}

func NewInstanceServiceFromCloud(cloud clientconfig.Cloud) (*InstanceService, error) {
	clientOpts := new(clientconfig.ClientOpts)
	var opts *gophercloud.AuthOptions

	if cloud.AuthInfo != nil {
		clientOpts.AuthInfo = cloud.AuthInfo
		clientOpts.AuthType = cloud.AuthType
		clientOpts.Cloud = cloud.Cloud
		clientOpts.RegionName = cloud.RegionName
	}

	opts, err := clientconfig.AuthOptions(clientOpts)

	if err != nil {
		return nil, err
	}

	opts.AllowReauth = true

	provider, err := openstack.AuthenticatedClient(*opts)
	if err != nil {
		return nil, fmt.Errorf("Create providerClient err: %v", err)
	}

	identityClient, err := openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{
		Region: "",
	})
	if err != nil {
		return nil, fmt.Errorf("Create identityClient err: %v", err)
	}
	serverClient, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("Create serviceClient err: %v", err)
	}
	networkingClient, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("Create networkingClient err: %v", err)
	}

	return &InstanceService{
		provider:       provider,
		identityClient: identityClient,
		computeClient:  serverClient,
		networkClient:  networkingClient,
	}, nil
}

// UpdateToken to update token if need.
func (is *InstanceService) UpdateToken() error {
	token := is.provider.Token()
	result, err := tokens.Validate(is.identityClient, token)
	if err != nil {
		return fmt.Errorf("Validate token err: %v", err)
	}
	if result {
		return nil
	}
	klog.V(2).Infof("Token is out of date, getting new token.")
	reAuthFunction := is.provider.ReauthFunc
	if reAuthFunction() != nil {
		return fmt.Errorf("reAuth err: %v", err)
	}
	return nil
}

func (is *InstanceService) AssociateFloatingIP(instanceID, floatingIP string) error {
	opts := floatingips.AssociateOpts{
		FloatingIP: floatingIP,
	}
	return floatingips.AssociateInstance(is.computeClient, instanceID, opts).ExtractErr()
}

func (is *InstanceService) GetAcceptableFloatingIP() (string, error) {
	page, err := floatingips.List(is.computeClient).AllPages()
	if err != nil {
		return "", fmt.Errorf("Get floating IP list failed: %v", err)
	}
	list, err := floatingips.ExtractFloatingIPs(page)
	if err != nil {
		return "", err
	}
	for _, floatingIP := range list {
		if floatingIP.FixedIP == "" {
			return floatingIP.IP, nil
		}
	}
	return "", fmt.Errorf("Don't have acceptable floating IP")
}

// A function for getting the id of a network by querying openstack with filters
func getNetworkIDsByFilter(is *InstanceService, opts *networks.ListOpts) ([]string, error) {
	if opts == nil {
		return []string{}, fmt.Errorf("No Filters were passed")
	}
	pager := networks.List(is.networkClient, opts)
	var uuids []string
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		networkList, err := networks.ExtractNetworks(page)
		if err != nil {
			return false, err
		} else if len(networkList) == 0 {
			return false, fmt.Errorf("No networks could be found with the filters provided")
		}
		for _, network := range networkList {
			uuids = append(uuids, network.ID)
		}
		return true, nil
	})
	if err != nil {
		return []string{}, err
	}
	return uuids, nil
}

func (is *InstanceService) InstanceCreate(name string, config *openstackconfigv1.OpenstackProviderSpec, cmd string, keyName string) (instance *Instance, err error) {
	var createOpts servers.CreateOpts
	if config == nil {
		return nil, fmt.Errorf("create Options need be specified to create instace")
	}
	var nets []servers.Network
	for _, net := range config.Networks {
		if net.UUID == "" {
			opts := networks.ListOpts(net.Filter)
			ids, err := getNetworkIDsByFilter(is, &opts)
			if err != nil {
				return nil, err
			}
			for _, netID := range ids {
				nets = append(nets, servers.Network{
					UUID: netID,
				})
			}
		} else {
			nets = append(nets, servers.Network{
				UUID: net.UUID,
			})
		}
	}
	userData := base64.StdEncoding.EncodeToString([]byte(cmd))
	createOpts = servers.CreateOpts{
		Name:             name,
		ImageName:        config.Image,
		FlavorName:       config.Flavor,
		AvailabilityZone: config.AvailabilityZone,
		Networks:         nets,
		UserData:         []byte(userData),
		SecurityGroups:   config.SecurityGroups,
		ServiceClient:    is.computeClient,
	}
	server, err := servers.Create(is.computeClient, keypairs.CreateOptsExt{
		CreateOptsBuilder: createOpts,
		KeyName:           keyName,
	}).Extract()
	if err != nil {
		return nil, fmt.Errorf("Create new server err: %v", err)
	}
	return serverToInstance(server), nil
}

func (is *InstanceService) InstanceDelete(id string) error {
	return servers.Delete(is.computeClient, id).ExtractErr()
}

func (is *InstanceService) GetInstanceList(opts *InstanceListOpts) ([]*Instance, error) {
	var listOpts servers.ListOpts
	if opts != nil {
		listOpts = servers.ListOpts{
			Name: opts.Name,
		}
	} else {
		listOpts = servers.ListOpts{}
	}

	allPages, err := servers.List(is.computeClient, listOpts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("Get service list err: %v", err)
	}
	serverList, err := servers.ExtractServers(allPages)
	if err != nil {
		return nil, fmt.Errorf("Extract services list err: %v", err)
	}
	var instanceList []*Instance
	for _, server := range serverList {
		instanceList = append(instanceList, serverToInstance(&server))
	}
	return instanceList, nil
}

func (is *InstanceService) GetInstance(resourceId string) (instance *Instance, err error) {
	if resourceId == "" {
		return nil, fmt.Errorf("ResourceId should be specified to  get detail.")
	}
	server, err := servers.Get(is.computeClient, resourceId).Extract()
	if err != nil {
		return nil, fmt.Errorf("Get server %q detail failed: %v", resourceId, err)
	}
	return serverToInstance(server), err
}

func serverToInstance(server *servers.Server) *Instance {
	return &Instance{*server}
}
