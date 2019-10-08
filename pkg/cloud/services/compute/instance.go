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
	"fmt"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	"time"

	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha2"
	"sigs.k8s.io/cluster-api/controllers/noderefutil"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"

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
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha2"
	"sigs.k8s.io/cluster-api/util"
)

const (
	TimeoutTrunkDelete       = 3 * time.Minute
	RetryIntervalTrunkDelete = 5 * time.Second

	TimeoutPortDelete       = 3 * time.Minute
	RetryIntervalPortDelete = 5 * time.Second
)

// TODO(sbueringer) We should probably wrap the OpenStack object completely (see CAPA)
type Instance struct {
	servers.Server
	State infrav1.InstanceState
}

type ServerNetwork struct {
	networkID string
	subnetID  string
}

// InstanceCreate creates a compute instance
func (s *Service) InstanceCreate(clusterName string, machine *clusterv1.Machine, openStackMachine *infrav1.OpenStackMachine, openStackCluster *infrav1.OpenStackCluster) (instance *Instance, err error) {
	if openStackMachine == nil {
		return nil, fmt.Errorf("create Options need be specified to create instace")
	}
	if openStackMachine.Spec.Trunk {
		trunkSupport, err := getTrunkSupport(s)
		if err != nil {
			return nil, fmt.Errorf("there was an issue verifying whether trunk support is available, please disable it: %v", err)
		}
		if !trunkSupport {
			return nil, fmt.Errorf("there is no trunk support. Please disable it")
		}
	}

	// Set default Tags
	machineTags := []string{
		"cluster-api-provider-openstack",
		clusterName,
	}

	// Append machine specific tags
	machineTags = append(machineTags, openStackMachine.Spec.Tags...)

	// Append cluster scope tags
	machineTags = append(machineTags, openStackCluster.Spec.Tags...)

	// Get security groups
	securityGroups, err := getSecurityGroups(s, openStackMachine.Spec.SecurityGroups)
	if err != nil {
		return nil, err
	}
	// Get all network UUIDs
	var nets []ServerNetwork
	if len(openStackMachine.Spec.Networks) > 0 {
		for _, net := range openStackMachine.Spec.Networks {
			opts := networks.ListOpts(net.Filter)
			opts.ID = net.UUID
			ids, err := networking.GetNetworkIDsByFilter(s.networkClient, &opts)
			if err != nil {
				return nil, err
			}
			for _, netID := range ids {
				if net.Subnets == nil {
					nets = append(nets, ServerNetwork{
						networkID: netID,
					})
					continue
				}

				for _, subnet := range net.Subnets {
					subnetOpts := subnets.ListOpts(subnet.Filter)
					subnetOpts.ID = subnet.UUID
					subnetOpts.NetworkID = netID
					subnetsByFilter, err := networking.GetSubnetsByFilter(s.networkClient, &subnetOpts)
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
	} else {
		if openStackCluster.Status.Network == nil {
			return nil, fmt.Errorf(".spec.networks not set in Machine and also no network was found in .status.network in OpenStackCluster")
		}
		if openStackCluster.Status.Network.Subnet == nil {
			return nil, fmt.Errorf(".spec.networks not set in Machine and also no subnet was found in .status.network.subnet in OpenStackCluster")
		}

		nets = []ServerNetwork{{
			networkID: openStackCluster.Status.Network.ID,
			subnetID:  openStackCluster.Status.Network.Subnet.ID,
		}}
	}
	if len(nets) == 0 {
		return nil, fmt.Errorf("no network was found or provided. Please check your machine configuration and try again")
	}

	portsList := []servers.Network{}
	for _, net := range nets {
		if net.networkID == "" {
			return nil, fmt.Errorf("no network was found or provided. Please check your machine configuration and try again")
		}
		allPages, err := ports.List(s.networkClient, ports.ListOpts{
			Name:      openStackMachine.Name,
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
			port, err = createPort(s, openStackMachine.Name, net, &securityGroups)
			if err != nil {
				return nil, fmt.Errorf("failed to create port err: %v", err)
			}
		} else {
			port = portList[0]
		}

		_, err = attributestags.ReplaceAll(s.networkClient, "ports", port.ID, attributestags.ReplaceAllOpts{
			Tags: machineTags}).Extract()
		if err != nil {
			return nil, fmt.Errorf("tagging port for server err: %v", err)
		}
		portsList = append(portsList, servers.Network{
			Port: port.ID,
		})

		if openStackMachine.Spec.Trunk {
			allPages, err := trunks.List(s.networkClient, trunks.ListOpts{
				Name:   openStackMachine.Name,
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
					Name:   openStackMachine.Name,
					PortID: port.ID,
				}
				newTrunk, err := trunks.Create(s.networkClient, trunkCreateOpts).Extract()
				if err != nil {
					return nil, fmt.Errorf("create trunk for server err: %v", err)
				}
				trunk = *newTrunk
			} else {
				trunk = trunkList[0]
			}

			_, err = attributestags.ReplaceAll(s.networkClient, "trunks", trunk.ID, attributestags.ReplaceAllOpts{
				Tags: machineTags}).Extract()
			if err != nil {
				return nil, fmt.Errorf("tagging trunk for server err: %v", err)
			}
		}
	}

	var serverTags []string
	if !openStackCluster.Spec.DisableServerTags {
		serverTags = machineTags
		// NOTE(flaper87): This is the minimum required version
		// to use tags.
		s.computeClient.Microversion = "2.52"
		defer func(s *Service) {
			s.computeClient.Microversion = ""
		}(s)
	}

	// Get image ID
	imageID, err := getImageID(s, openStackMachine.Spec.Image)
	if err != nil {
		return nil, fmt.Errorf("create new server err: %v", err)
	}

	var serverCreateOpts servers.CreateOptsBuilder = servers.CreateOpts{
		Name:             openStackMachine.Name,
		ImageRef:         imageID,
		FlavorName:       openStackMachine.Spec.Flavor,
		AvailabilityZone: openStackMachine.Spec.AvailabilityZone,
		Networks:         portsList,
		UserData:         []byte(*machine.Spec.Bootstrap.Data),
		SecurityGroups:   securityGroups,
		ServiceClient:    s.computeClient,
		Tags:             serverTags,
		Metadata:         openStackMachine.Spec.ServerMetadata,
		ConfigDrive:      openStackMachine.Spec.ConfigDrive,
	}

	// If the root volume Size is not 0, means boot from volume
	if openStackMachine.Spec.RootVolume != nil && openStackMachine.Spec.RootVolume.Size != 0 {
		block := bootfromvolume.BlockDevice{
			SourceType:          bootfromvolume.SourceType(openStackMachine.Spec.RootVolume.SourceType),
			BootIndex:           0,
			UUID:                openStackMachine.Spec.RootVolume.SourceUUID,
			DeleteOnTermination: true,
			DestinationType:     bootfromvolume.DestinationVolume,
			VolumeSize:          openStackMachine.Spec.RootVolume.Size,
			DeviceType:          openStackMachine.Spec.RootVolume.DeviceType,
		}
		serverCreateOpts = bootfromvolume.CreateOptsExt{
			CreateOptsBuilder: serverCreateOpts,
			BlockDevice:       []bootfromvolume.BlockDevice{block},
		}
	}

	server, err := servers.Create(s.computeClient, keypairs.CreateOptsExt{
		CreateOptsBuilder: serverCreateOpts,
		KeyName:           openStackMachine.Spec.SSHKeyName,
	}).Extract()
	if err != nil {
		record.Warnf(openStackMachine, "FailedCreateServer", "Failed to create server: %v", err)
		return nil, fmt.Errorf("create new server err: %v", err)
	}
	record.Eventf(openStackMachine, "SuccessfulCreateServer", "Created server %s with id %s", openStackMachine.Name, server.ID)

	return &Instance{Server: *server, State: infrav1.InstanceState(server.Status)}, nil
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

func getSecurityGroups(is *Service, securityGroupParams []infrav1.SecurityGroupParam) ([]string, error) {
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
	if len(list) == 0 {
		return false
	}
	for _, element := range list {
		if element == name {
			return true
		}
	}
	return false
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

func (s *Service) AssociateFloatingIP(instanceID, floatingIP string) error {
	opts := floatingips.AssociateOpts{
		FloatingIP: floatingIP,
	}
	return floatingips.AssociateInstance(s.computeClient, instanceID, opts).ExtractErr()
}

func (s *Service) InstanceDelete(machine *clusterv1.Machine) error {

	if machine.Spec.ProviderID == nil {
		// nothing to do
		return nil
	}

	parsed, err := noderefutil.NewProviderID(*machine.Spec.ProviderID)
	if err != nil {
		return err
	}

	// get instance port id
	allInterfaces, err := attachinterfaces.List(s.computeClient, parsed.ID()).AllPages()
	if err != nil {
		return err
	}
	instanceInterfaces, err := attachinterfaces.ExtractInterfaces(allInterfaces)
	if err != nil {
		return err
	}
	if len(instanceInterfaces) < 1 {
		return servers.Delete(s.computeClient, parsed.ID()).ExtractErr()
	}

	trunkSupport, err := getTrunkSupport(s)
	if err != nil {
		return fmt.Errorf("obtaining network extensions: %v", err)
	}
	// get and delete trunks
	for _, port := range instanceInterfaces {
		err := attachinterfaces.Delete(s.computeClient, parsed.ID(), port.PortID).ExtractErr()
		if err != nil {
			return err
		}
		if trunkSupport {
			listOpts := trunks.ListOpts{
				PortID: port.PortID,
			}
			allTrunks, err := trunks.List(s.networkClient, listOpts).AllPages()
			if err != nil {
				return err
			}
			trunkInfo, err := trunks.ExtractTrunks(allTrunks)
			if err != nil {
				return err
			}
			if len(trunkInfo) == 1 {
				err = util.PollImmediate(RetryIntervalTrunkDelete, TimeoutTrunkDelete, func() (bool, error) {
					err := trunks.Delete(s.networkClient, trunkInfo[0].ID).ExtractErr()
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
			err := ports.Delete(s.networkClient, port.PortID).ExtractErr()
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
	return servers.Delete(s.computeClient, parsed.ID()).ExtractErr()
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

func (s *Service) GetInstanceList(opts *InstanceListOpts) ([]*Instance, error) {
	var listOpts servers.ListOpts
	if opts != nil {
		listOpts = servers.ListOpts{
			Name: opts.Name,
		}
	} else {
		listOpts = servers.ListOpts{}
	}

	allPages, err := servers.List(s.computeClient, listOpts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("get service list: %v", err)
	}
	serverList, err := servers.ExtractServers(allPages)
	if err != nil {
		return nil, fmt.Errorf("extract services list: %v", err)
	}
	instanceList := []*Instance{}
	for _, server := range serverList {
		instanceList = append(instanceList, &Instance{
			Server: server,
			State:  infrav1.InstanceState(server.Status),
		})
	}
	return instanceList, nil
}

func (s *Service) GetInstance(resourceID string) (instance *Instance, err error) {
	if resourceID == "" {
		return nil, fmt.Errorf("resourceId should be specified to  get detail")
	}
	server, err := servers.Get(s.computeClient, resourceID).Extract()
	if err != nil {
		return nil, fmt.Errorf("get server %q detail failed: %v", resourceID, err)
	}
	return &Instance{Server: *server, State: infrav1.InstanceState(server.Status)}, err
}

func (s *Service) InstanceExists(openStackMachine *infrav1.OpenStackMachine) (instance *Instance, err error) {
	opts := &InstanceListOpts{
		Name:   openStackMachine.Name,
		Image:  openStackMachine.Spec.Image,
		Flavor: openStackMachine.Spec.Flavor,
	}

	instanceList, err := s.GetInstanceList(opts)
	if err != nil {
		return nil, err
	}
	if len(instanceList) == 0 {
		return nil, nil
	}
	return instanceList[0], nil
}
