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

package compute

import (
	"encoding/base64"
	"fmt"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"k8s.io/klog"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/networking"
	"time"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/common/extensions"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/attachinterfaces"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/floatingips"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	netext "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/pagination"
	openstackconfigv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
)

const (
	TimeoutTrunkDelete       = 3 * time.Minute
	RetryIntervalTrunkDelete = 5 * time.Second

	TimeoutPortDelete       = 3 * time.Minute
	RetryIntervalPortDelete = 5 * time.Second
)

type Instance struct {
	servers.Server
}

type ServerNetwork struct {
	networkID string
	subnetID  string
}

// InstanceCreate creates a compute instance
func (is *Service) InstanceCreate(clusterName string, name string, clusterSpec *openstackconfigv1.OpenstackClusterProviderSpec, config *openstackconfigv1.OpenstackProviderSpec, userdata string, keyName string) (instance *Instance, err error) {
	var createOpts servers.CreateOptsBuilder
	if config == nil {
		return nil, fmt.Errorf("create Options need be specified to create instace")
	}
	if config.Trunk == true {
		trunkSupport, err := getTrunkSupport(is)
		if err != nil {
			return nil, fmt.Errorf("there was an issue verifying whether trunk support is available, please disable it: %v", err)
		}
		if trunkSupport == false {
			return nil, fmt.Errorf("there is no trunk support. Please disable it")
		}
	}

	// Set default Tags
	machineTags := []string{
		"cluster-api-provider-openstack",
		clusterName,
	}

	// Append machine specific tags
	machineTags = append(machineTags, config.Tags...)

	// Append cluster scope tags
	machineTags = append(machineTags, clusterSpec.Tags...)

	// Get security groups
	securityGroups, err := getSecurityGroups(is, config.SecurityGroups)
	if err != nil {
		return nil, err
	}
	// Get all network UUIDs
	var nets []ServerNetwork
	for _, net := range config.Networks {
		opts := networks.ListOpts(net.Filter)
		opts.ID = net.UUID
		ids, err := getNetworkIDsByFilter(is.networkClient, &opts)
		if err != nil {
			return nil, err
		}
		for _, netID := range ids {
			if net.Subnets == nil {
				nets = append(nets, ServerNetwork{
					networkID: netID,
				})
			}

			for _, subnet := range net.Subnets {
				subnetOpts := subnets.ListOpts(subnet.Filter)
				subnetOpts.ID = subnet.UUID
				subnetOpts.NetworkID = netID
				subnetsByFilter, err := networking.GetSubnetsByFilter(is.networkClient, &subnetOpts)
				if err != nil {
					return nil, err
				}
				for _, subnetByFilter := range subnetsByFilter {
					nets = append(nets, ServerNetwork{
						networkID: subnetByFilter.NetworkID,
						subnetID:  subnetByFilter.ID,
					})
				}
			}
		}
	}
	userData := base64.StdEncoding.EncodeToString([]byte(userdata))
	var portsList []servers.Network
	for _, net := range nets {
		if net.networkID == "" {
			return nil, fmt.Errorf("no network was found or provided. Please check your machine configuration and try again")
		}
		allPages, err := ports.List(is.networkClient, ports.ListOpts{
			Name:      name,
			NetworkID: net.networkID,
		}).AllPages()
		if err != nil {
			return nil, fmt.Errorf("searching for existing port for server: %v", err)
		}
		portList, err := ports.ExtractPorts(allPages)
		if err != nil {
			return nil, fmt.Errorf("searching for existing port for server err: %v", err)
		}
		var port ports.Port
		if len(portList) == 0 {
			// create server port
			port, err = createPort(is, name, net, &securityGroups)
			if err != nil {
				return nil, fmt.Errorf("failed to create port err: %v", err)
			}
		} else {
			port = portList[0]
		}

		_, err = attributestags.ReplaceAll(is.networkClient, "ports", port.ID, attributestags.ReplaceAllOpts{
			Tags: machineTags}).Extract()
		if err != nil {
			return nil, fmt.Errorf("tagging port for server err: %v", err)
		}
		portsList = append(portsList, servers.Network{
			Port: port.ID,
		})

		if config.Trunk == true {
			allPages, err := trunks.List(is.networkClient, trunks.ListOpts{
				Name:   name,
				PortID: port.ID,
			}).AllPages()
			if err != nil {
				return nil, fmt.Errorf("searching for existing trunk for server err: %v", err)
			}
			trunkList, err := trunks.ExtractTrunks(allPages)
			if err != nil {
				return nil, fmt.Errorf("searching for existing trunk for server err: %v", err)
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
					return nil, fmt.Errorf("create trunk for server err: %v", err)
				}
				trunk = *newTrunk
			} else {
				trunk = trunkList[0]
			}

			_, err = attributestags.ReplaceAll(is.networkClient, "trunks", trunk.ID, attributestags.ReplaceAllOpts{
				Tags: machineTags}).Extract()
			if err != nil {
				return nil, fmt.Errorf("tagging trunk for server err: %v", err)
			}
		}
	}

	var serverTags []string
	if clusterSpec.DisableServerTags == false {
		serverTags = machineTags
		// NOTE(flaper87): This is the minimum required version
		// to use tags.
		is.computeClient.Microversion = "2.52"
	}

	// Get image ID
	imageID, err := getImageID(is, config.Image)
	if err != nil {
		return nil, fmt.Errorf("create new server err: %v", err)
	}

	serverCreateOpts := servers.CreateOpts{
		Name:             name,
		ImageRef:         imageID,
		FlavorName:       config.Flavor,
		AvailabilityZone: config.AvailabilityZone,
		Networks:         portsList,
		UserData:         []byte(userData),
		SecurityGroups:   securityGroups,
		ServiceClient:    is.computeClient,
		Tags:             serverTags,
		Metadata:         config.ServerMetadata,
		ConfigDrive:      config.ConfigDrive,
	}

	// If the root volume Size is not 0, means boot from volume
	if config.RootVolume != nil && config.RootVolume.Size != 0 {
		var blocks []bootfromvolume.BlockDevice

		block := bootfromvolume.BlockDevice{
			SourceType:          bootfromvolume.SourceType(config.RootVolume.SourceType),
			BootIndex:           0,
			UUID:                config.RootVolume.SourceUUID,
			DeleteOnTermination: true,
			DestinationType:     bootfromvolume.DestinationVolume,
			VolumeSize:          config.RootVolume.Size,
			DeviceType:          config.RootVolume.DeviceType,
		}
		blocks = append(blocks, block)

		createOpts = bootfromvolume.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			BlockDevice:       blocks,
		}
	}

	server, err := servers.Create(is.computeClient, keypairs.CreateOptsExt{
		CreateOptsBuilder: serverCreateOpts,
		KeyName:           keyName,
	}).Extract()
	if err != nil {
		return nil, fmt.Errorf("create new server err: %v", err)
	}
	is.computeClient.Microversion = ""
	return &Instance{*server}, nil
}

func getTrunkSupport(is *Service) (bool, error) {
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

func getSecurityGroups(is *Service, securityGroupParams []openstackconfigv1.SecurityGroupParam) ([]string, error) {
	var sgIDs []string
	for _, sg := range securityGroupParams {
		listOpts := groups.ListOpts(sg.Filter)
		listOpts.Name = sg.Name
		listOpts.ID = sg.UUID
		pages, err := groups.List(is.networkClient, listOpts).AllPages()
		if err != nil {
			return nil, err
		}

		SGList, err := groups.ExtractGroups(pages)
		if err != nil {
			return nil, err
		}

		for _, group := range SGList {
			if isDuplicate(sgIDs, group.ID) {
				continue
			}
			sgIDs = append(sgIDs, group.ID)
		}
	}
	return sgIDs, nil
}

func isDuplicate(list []string, name string) bool {
	if list == nil || len(list) == 0 {
		return false
	}
	for _, element := range list {
		if element == name {
			return true
		}
	}
	return false
}

// A function for getting the id of a network by querying openstack with filters
func getNetworkIDsByFilter(networkClient *gophercloud.ServiceClient, opts *networks.ListOpts) ([]string, error) {
	if opts == nil {
		return []string{}, fmt.Errorf("no Filters were passed")
	}
	pager := networks.List(networkClient, opts)
	var uuids []string
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		networkList, err := networks.ExtractNetworks(page)
		if err != nil {
			return false, err
		} else if len(networkList) == 0 {
			return false, fmt.Errorf("no networks could be found with the filters provided")
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

func createPort(is *Service, name string, net ServerNetwork, securityGroups *[]string) (ports.Port, error) {
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
		return ports.Port{}, fmt.Errorf("create port for server: %v", err)
	}
	return *newPort, nil
}

// Helper function for getting image ID from name
func getImageID(is *Service, imageName string) (string, error) {
	if imageName == "" {
		return "", nil
	}

	opts := images.ListOpts{
		Name: imageName,
	}

	pages, err := images.List(is.imagesClient, opts).AllPages()
	if err != nil {
		return "", err
	}

	allImages, err := images.ExtractImages(pages)
	if err != nil {
		return "", err
	}

	switch len(allImages) {
	case 0:
		return "", fmt.Errorf("no image with the name %s could be found", imageName)
	case 1:
		return allImages[0].ID, nil
	default:
		return "", fmt.Errorf("too many images with the name, %s, were found", imageName)
	}
}

func (is *Service) AssociateFloatingIP(instanceID, floatingIP string) error {
	opts := floatingips.AssociateOpts{
		FloatingIP: floatingIP,
	}
	return floatingips.AssociateInstance(is.computeClient, instanceID, opts).ExtractErr()
}

func (is *Service) InstanceDelete(id string) error {
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

	trunkSupport, err := getTrunkSupport(is)
	if err != nil {
		return fmt.Errorf("obtaining network extensions: %v", err)
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
					return fmt.Errorf("error deleting the trunk %v", trunkInfo[0].ID)
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
			return fmt.Errorf("error deleting the port %v", port.PortID)
		}
	}

	// delete instance
	return servers.Delete(is.computeClient, id).ExtractErr()
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

func (is *Service) GetInstanceList(opts *InstanceListOpts) ([]*Instance, error) {
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
		return nil, fmt.Errorf("get service list: %v", err)
	}
	serverList, err := servers.ExtractServers(allPages)
	if err != nil {
		return nil, fmt.Errorf("extract services list: %v", err)
	}
	var instanceList []*Instance
	for _, server := range serverList {
		instanceList = append(instanceList, &Instance{server})
	}
	return instanceList, nil
}

func (is *Service) GetInstance(resourceId string) (instance *Instance, err error) {
	if resourceId == "" {
		return nil, fmt.Errorf("ResourceId should be specified to  get detail.")
	}
	server, err := servers.Get(is.computeClient, resourceId).Extract()
	if err != nil {
		return nil, fmt.Errorf("get server %q detail failed: %v", resourceId, err)
	}
	return &Instance{*server}, err
}

// UpdateToken to update token if need.
func (is *Service) UpdateToken() error {
	token := is.provider.Token()
	result, err := tokens.Validate(is.identityClient, token)
	if err != nil {
		return fmt.Errorf("validate token: %v", err)
	}
	if result {
		return nil
	}
	klog.V(2).Infof("Token is out of date, getting new token.")
	reAuthFunction := is.provider.ReauthFunc
	if reAuthFunction() != nil {
		return fmt.Errorf("reAuth: %v", err)
	}
	return nil
}
