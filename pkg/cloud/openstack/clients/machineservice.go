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
	"time"

	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/common/extensions"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/attachinterfaces"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/floatingips"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	netext "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/gophercloud/utils/openstack/clientconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	openstackconfigv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
)

const (
	CloudsSecretKey = "clouds.yaml"

	TimeoutTrunkDelete       = 3 * time.Minute
	RetryIntervalTrunkDelete = 5 * time.Second

	TimeoutPortDelete       = 3 * time.Minute
	RetryIntervalPortDelete = 5 * time.Second
)

type InstanceService struct {
	provider       *gophercloud.ProviderClient
	computeClient  *gophercloud.ServiceClient
	identityClient *gophercloud.ServiceClient
	networkClient  *gophercloud.ServiceClient
}

type Instance struct {
	servers.Server
}

type ServerNetwork struct {
	networkID string
	subnetID  string
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
	cloud := clientconfig.Cloud{}
	if machineSpec.CloudsSecret != nil && machineSpec.CloudsSecret.Name != "" {
		namespace := machineSpec.CloudsSecret.Namespace
		if namespace == "" {
			namespace = machine.Namespace
		}
		cloud, err = GetCloudFromSecret(kubeClient, namespace, machineSpec.CloudsSecret.Name, machineSpec.CloudName)
		if err != nil {
			return nil, err
		}
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

// A function for getting the id of a subnet by querying openstack with filters
func getSubnetsByFilter(is *InstanceService, opts *subnets.ListOpts) ([]subnets.Subnet, error) {
	if opts == nil {
		return []subnets.Subnet{}, fmt.Errorf("No Filters were passed")
	}
	pager := subnets.List(is.networkClient, opts)
	var snets []subnets.Subnet
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		subnetList, err := subnets.ExtractSubnets(page)
		if err != nil {
			return false, err
		} else if len(subnetList) == 0 {
			return false, fmt.Errorf("No subnets could be found with the filters provided")
		}
		for _, subnet := range subnetList {
			snets = append(snets, subnet)
		}
		return true, nil
	})
	if err != nil {
		return []subnets.Subnet{}, err
	}
	return snets, nil
}

func CreatePort(is *InstanceService, name string, net ServerNetwork, securityGroups *[]string) (ports.Port, error) {
	portCreateOpts := ports.CreateOpts{
		Name:           name,
		NetworkID:      net.networkID,
		SecurityGroups: securityGroups,
	}
	if net.subnetID != "" {
		portCreateOpts.FixedIPs = []ports.IP{{SubnetID: net.subnetID}}
	}
	newPort, err := ports.Create(is.networkClient, portCreateOpts).Extract()
	if err != nil {
		return ports.Port{}, fmt.Errorf("Create port for server err: %v", err)
	}
	return *newPort, nil
}

func (is *InstanceService) InstanceCreate(clusterName string, name string, config *openstackconfigv1.OpenstackProviderSpec, cmd string, keyName string) (instance *Instance, err error) {
	if config == nil {
		return nil, fmt.Errorf("create Options need be specified to create instace")
	}
	if config.Trunk == true {
		trunkSupport, err := GetTrunkSupport(is)
		if err != nil {
			return nil, fmt.Errorf("There was an issue verifying whether trunk support is available, please disable it: %v", err)
		}
		if trunkSupport == false {
			return nil, fmt.Errorf("There is no trunk support. Please disable it")
		}
	}
	clusterTags := []string{
		"cluster-api-provider-openstack",
		clusterName,
	}
	// Get all network UUIDs
	var nets []ServerNetwork
	for _, net := range config.Networks {
		opts := networks.ListOpts(net.Filter)
		opts.ID = net.UUID
		ids, err := getNetworkIDsByFilter(is, &opts)
		if err != nil {
			return nil, err
		}
		for _, netID := range ids {
			if net.Subnets == nil {
				nets = append(nets, ServerNetwork{
					networkID: netID,
				})
			}

			for _, snet := range net.Subnets {
				sopts := subnets.ListOpts(snet.Filter)
				sopts.ID = netID
				sopts.NetworkID = net.UUID
				snets, err := getSubnetsByFilter(is, &sopts)
				if err != nil {
					return nil, err
				}
				for _, snet := range snets {
					nets = append(nets, ServerNetwork{
						networkID: snet.NetworkID,
						subnetID:  snet.ID,
					})
				}
			}
		}

		if len(net.Filter.Tags) > 0 {
			clusterTags = append(clusterTags, net.Filter.Tags)
		}
	}
	userData := base64.StdEncoding.EncodeToString([]byte(cmd))
	var ports_list []servers.Network
	for _, net := range nets {
		if net.networkID == "" {
			return nil, fmt.Errorf("No network was found or provided. Please check your machine configuration and try again")
		}
		allPages, err := ports.List(is.networkClient, ports.ListOpts{
			Name:      name,
			NetworkID: net.networkID,
		}).AllPages()
		if err != nil {
			return nil, fmt.Errorf("Searching for existing port for server err: %v", err)
		}
		portList, err := ports.ExtractPorts(allPages)
		if err != nil {
			return nil, fmt.Errorf("Searching for existing port for server err: %v", err)
		}
		var port ports.Port
		if len(portList) == 0 {
			// create server port
			port, err = CreatePort(is, name, net, &config.SecurityGroups)
			if err != nil {
				return nil, fmt.Errorf("Failed to create port err: %v", err)
			}
		} else {
			port = portList[0]
		}

		_, err = attributestags.ReplaceAll(is.networkClient, "ports", port.ID, attributestags.ReplaceAllOpts{
			Tags: clusterTags}).Extract()
		if err != nil {
			return nil, fmt.Errorf("Tagging port for server err: %v", err)
		}
		ports_list = append(ports_list, servers.Network{
			Port: port.ID,
		})

		if config.Trunk == true {
			allPages, err := trunks.List(is.networkClient, trunks.ListOpts{
				Name:   name,
				PortID: port.ID,
			}).AllPages()
			if err != nil {
				return nil, fmt.Errorf("Searching for existing trunk for server err: %v", err)
			}
			trunkList, err := trunks.ExtractTrunks(allPages)
			if err != nil {
				return nil, fmt.Errorf("Searching for existing trunk for server err: %v", err)
			}
			var trunk trunks.Trunk
			if len(trunkList) == 0 {
				// create trunk with the previous port as parent
				trunkCreateOpts := trunks.CreateOpts{
					Name:   name,
					PortID: port.ID,
				}
				newTrunk, err := trunks.Create(is.networkClient, trunkCreateOpts).Extract()
				if err != nil {
					return nil, fmt.Errorf("Create trunk for server err: %v", err)
				}
				trunk = *newTrunk
			} else {
				trunk = trunkList[0]
			}

			_, err = attributestags.ReplaceAll(is.networkClient, "trunks", trunk.ID, attributestags.ReplaceAllOpts{
				Tags: clusterTags}).Extract()
			if err != nil {
				return nil, fmt.Errorf("Tagging trunk for server err: %v", err)
			}
		}
	}

	serverCreateOpts := servers.CreateOpts{
		Name:             name,
		ImageName:        config.Image,
		FlavorName:       config.Flavor,
		AvailabilityZone: config.AvailabilityZone,
		Networks:         ports_list,
		UserData:         []byte(userData),
		SecurityGroups:   config.SecurityGroups,
		ServiceClient:    is.computeClient,
	}
	server, err := servers.Create(is.computeClient, keypairs.CreateOptsExt{
		CreateOptsBuilder: serverCreateOpts,
		KeyName:           keyName,
	}).Extract()
	if err != nil {
		return nil, fmt.Errorf("Create new server err: %v", err)
	}
	return serverToInstance(server), nil
}

func (is *InstanceService) InstanceDelete(id string) error {
	// get instance port id
	allInterfaces, err := attachinterfaces.List(is.computeClient, id).AllPages()
	if err != nil {
		return err
	}
	instanceInterfaces, err := attachinterfaces.ExtractInterfaces(allInterfaces)
	if err != nil {
		return err
	}
	if len(instanceInterfaces) < 1 {
		return servers.Delete(is.computeClient, id).ExtractErr()
	}

	trunkSupport, err := GetTrunkSupport(is)
	if err != nil {
		return fmt.Errorf("Obtaining network extensions err: %v", err)
	}
	// get and delete trunks
	for _, port := range instanceInterfaces {
		err := attachinterfaces.Delete(is.computeClient, id, port.PortID).ExtractErr()
		if err != nil {
			return err
		}
		if trunkSupport {
			listOpts := trunks.ListOpts{
				PortID: port.PortID,
			}
			allTrunks, err := trunks.List(is.networkClient, listOpts).AllPages()
			if err != nil {
				return err
			}
			trunkInfo, err := trunks.ExtractTrunks(allTrunks)
			if err != nil {
				return err
			}
			if len(trunkInfo) == 1 {
				err = util.PollImmediate(RetryIntervalTrunkDelete, TimeoutTrunkDelete, func() (bool, error) {
					err := trunks.Delete(is.networkClient, trunkInfo[0].ID).ExtractErr()
					if err != nil {
						return false, nil
					}
					return true, nil
				})
				if err != nil {
					return fmt.Errorf("Error deleting the trunk %v", trunkInfo[0].ID)
				}
			}
		}

		// delete port
		err = util.PollImmediate(RetryIntervalPortDelete, TimeoutPortDelete, func() (bool, error) {
			err := ports.Delete(is.networkClient, port.PortID).ExtractErr()
			if err != nil {
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			return fmt.Errorf("Error deleting the port %v", port.PortID)
		}
	}

	// delete instance
	return servers.Delete(is.computeClient, id).ExtractErr()
}

func GetTrunkSupport(is *InstanceService) (bool, error) {
	allPages, err := netext.List(is.networkClient).AllPages()
	if err != nil {
		return false, err
	}

	allExts, err := extensions.ExtractExtensions(allPages)
	if err != nil {
		return false, err
	}

	for _, ext := range allExts {
		if ext.Alias == "trunk" {
			return true, nil
		}
	}
	return false, nil
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
