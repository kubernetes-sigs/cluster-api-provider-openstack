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
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/common/extensions"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/attachinterfaces"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/floatingips"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/schedulerhints"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	netext "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha3"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/controllers/noderefutil"
	"sigs.k8s.io/cluster-api/util"
)

const (
	TimeoutInstanceCreate       = 5
	RetryIntervalInstanceStatus = 10 * time.Second

	TimeoutTrunkDelete       = 3 * time.Minute
	RetryIntervalTrunkDelete = 5 * time.Second

	TimeoutPortDelete       = 3 * time.Minute
	RetryIntervalPortDelete = 5 * time.Second
)

// InstanceCreate creates a compute instance
func (s *Service) InstanceCreate(clusterName string, machine *clusterv1.Machine, openStackMachine *infrav1.OpenStackMachine, openStackCluster *infrav1.OpenStackCluster, userData string) (instance *infrav1.Instance, err error) {
	if openStackMachine == nil {
		return nil, fmt.Errorf("create Options need be specified to create instace")
	}

	if machine.Spec.FailureDomain == nil {
		return nil, fmt.Errorf("failure domain not set")
	}

	input := &infrav1.Instance{
		Name:          openStackMachine.Name,
		Image:         openStackMachine.Spec.Image,
		Flavor:        openStackMachine.Spec.Flavor,
		SSHKeyName:    openStackMachine.Spec.SSHKeyName,
		UserData:      userData,
		Metadata:      openStackMachine.Spec.ServerMetadata,
		ConfigDrive:   openStackMachine.Spec.ConfigDrive,
		FailureDomain: *machine.Spec.FailureDomain,
		RootVolume:    openStackMachine.Spec.RootVolume,
	}

	if openStackMachine.Spec.Trunk {
		trunkSupport, err := getTrunkSupport(s)
		if err != nil {
			return nil, fmt.Errorf("there was an issue verifying whether trunk support is available, please disable it: %v", err)
		}
		if !trunkSupport {
			return nil, fmt.Errorf("there is no trunk support. Please disable it")
		}
		input.Trunk = trunkSupport
	}

	machineTags := []string{}

	// Append machine specific tags
	machineTags = append(machineTags, openStackMachine.Spec.Tags...)

	// Append cluster scope tags
	machineTags = append(machineTags, openStackCluster.Spec.Tags...)

	// tags need to be unique or the "apply tags" call will fail.
	machineTags = deduplicate(machineTags)

	input.Tags = machineTags

	// Get security groups
	securityGroups, err := getSecurityGroups(s, openStackMachine.Spec.SecurityGroups)
	if err != nil {
		return nil, err
	}
	if openStackCluster.Spec.ManagedSecurityGroups {
		if util.IsControlPlaneMachine(machine) {
			securityGroups = append(securityGroups, openStackCluster.Status.ControlPlaneSecurityGroup.ID)
		} else {
			securityGroups = append(securityGroups, openStackCluster.Status.WorkerSecurityGroup.ID)
		}
	}
	input.SecurityGroups = &securityGroups

	var nets []infrav1.Network
	if len(openStackMachine.Spec.Networks) > 0 {
		var err error
		nets, err = getServerNetworks(s.networkClient, openStackMachine.Spec.Networks)
		if err != nil {
			return nil, err
		}
	} else {
		nets = []infrav1.Network{{
			ID: openStackCluster.Status.Network.ID,
			Subnet: &infrav1.Subnet{
				ID: openStackCluster.Status.Network.Subnet.ID,
			},
		}}
	}
	input.Networks = &nets

	out, err := s.createInstance(clusterName, input)
	if err != nil {
		record.Warnf(openStackMachine, "FailedCreateServer", "Failed to create server: %v", err)
		return nil, fmt.Errorf("create new server err: %v", err)
	}
	record.Eventf(openStackMachine, "SuccessfulCreateServer", "Created server %s with id %s", out.Name, out.ID)
	return out, nil
}

func (s *Service) createInstance(clusterName string, i *infrav1.Instance) (*infrav1.Instance, error) {
	// Get image ID
	imageID, err := getImageID(s, i.Image)
	if err != nil {
		return nil, fmt.Errorf("create new server err: %v", err)
	}

	nets := i.Networks
	portsList := []servers.Network{}
	for _, net := range *nets {
		net := net
		if net.ID == "" {
			return nil, fmt.Errorf("no network was found or provided. Please check your machine configuration and try again")
		}
		allPages, err := ports.List(s.networkClient, ports.ListOpts{
			Name:      i.Name,
			NetworkID: net.ID,
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
			port, err = createPort(s, clusterName, i.Name, &net, i.SecurityGroups)
			if err != nil {
				return nil, fmt.Errorf("failed to create port err: %v", err)
			}
		} else {
			port = portList[0]
		}

		portsList = append(portsList, servers.Network{
			Port: port.ID,
		})

		if i.Trunk {
			allPages, err := trunks.List(s.networkClient, trunks.ListOpts{
				Name:   i.Name,
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
					Name:   i.Name,
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
				Tags: i.Tags}).Extract()
			if err != nil {
				return nil, fmt.Errorf("tagging trunk for server err: %v", err)
			}
		}
	}
	var serverCreateOpts servers.CreateOptsBuilder = servers.CreateOpts{
		Name:             i.Name,
		ImageRef:         imageID,
		FlavorName:       i.Flavor,
		AvailabilityZone: i.FailureDomain,
		Networks:         portsList,
		UserData:         []byte(i.UserData),
		SecurityGroups:   *i.SecurityGroups,
		ServiceClient:    s.computeClient,
		Tags:             i.Tags,
		Metadata:         i.Metadata,
		ConfigDrive:      i.ConfigDrive,
	}

	serverCreateOpts = applyRootVolume(serverCreateOpts, i.RootVolume)

	serverCreateOpts = applyServerGroupID(serverCreateOpts, i.ServerGroupID)

	server, err := servers.Create(s.computeClient, keypairs.CreateOptsExt{
		CreateOptsBuilder: serverCreateOpts,
		KeyName:           i.SSHKeyName,
	}).Extract()
	if err != nil {
		return nil, fmt.Errorf("error creating Openstack instance: %v", err)
	}
	instanceCreateTimeout := getTimeout("CLUSTER_API_OPENSTACK_INSTANCE_CREATE_TIMEOUT", TimeoutInstanceCreate)
	instanceCreateTimeout *= time.Minute
	var instance *infrav1.Instance
	err = util.PollImmediate(RetryIntervalInstanceStatus, instanceCreateTimeout, func() (bool, error) {
		instance, err = s.GetInstance(server.ID)
		if err != nil {
			return false, nil

		}
		return instance.State == infrav1.InstanceStateActive, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error creating Openstack instance %s, %v", server.ID, err)
	}
	return instance, nil

}

func serverToInstance(v *servers.Server) (*infrav1.Instance, error) {
	if v == nil {
		return nil, nil
	}
	i := &infrav1.Instance{
		ID:         v.ID,
		Name:       v.Name,
		SSHKeyName: v.KeyName,
		State:      infrav1.InstanceState(v.Status),
	}
	addrMap, err := getIPFromInstance(v)
	if err != nil {
		return i, err
	}
	i.IP = addrMap["internal"]
	if addrMap["floating"] != "" {
		i.FloatingIP = addrMap["floating"]
	}
	return i, nil
}

func getIPFromInstance(v *servers.Server) (map[string]string, error) {
	addrMap := make(map[string]string)
	if v.AccessIPv4 != "" && net.ParseIP(v.AccessIPv4) != nil {
		addrMap["internal"] = v.AccessIPv4
		return addrMap, nil
	}
	type networkInterface struct {
		Address string  `json:"addr"`
		Version float64 `json:"version"`
		Type    string  `json:"OS-EXT-IPS:type"`
	}

	for _, b := range v.Addresses {
		list, err := json.Marshal(b)
		if err != nil {
			return nil, fmt.Errorf("extract IP from instance err: %v", err)
		}
		var networks []interface{}
		err = json.Unmarshal(list, &networks)
		if err != nil {
			return nil, fmt.Errorf("extract IP from instance err: %v", err)
		}
		for _, network := range networks {
			var netInterface networkInterface
			b, _ := json.Marshal(network)
			err = json.Unmarshal(b, &netInterface)
			if err != nil {
				return nil, fmt.Errorf("extract IP from instance err: %v", err)
			}
			if netInterface.Version == 4.0 {
				if netInterface.Type == "floating" {
					addrMap["floating"] = netInterface.Address
				} else {
					addrMap["internal"] = netInterface.Address
				}
			}
		}
	}

	return addrMap, nil
}

// applyRootVolume sets a root volume if the root volume Size is not 0
func applyRootVolume(opts servers.CreateOptsBuilder, rootVolume *infrav1.RootVolume) servers.CreateOptsBuilder {
	if rootVolume != nil && rootVolume.Size != 0 {
		block := bootfromvolume.BlockDevice{
			SourceType:          bootfromvolume.SourceType(rootVolume.SourceType),
			BootIndex:           0,
			UUID:                rootVolume.SourceUUID,
			DeleteOnTermination: true,
			DestinationType:     bootfromvolume.DestinationVolume,
			VolumeSize:          rootVolume.Size,
			DeviceType:          rootVolume.DeviceType,
		}
		return bootfromvolume.CreateOptsExt{
			CreateOptsBuilder: opts,
			BlockDevice:       []bootfromvolume.BlockDevice{block},
		}
	}
	return opts
}

// applyServerGroupID adds a scheduler hint to the CreateOptsBuilder, if the
// spec contains a server group ID
func applyServerGroupID(opts servers.CreateOptsBuilder, serverGroupID string) servers.CreateOptsBuilder {
	if serverGroupID != "" {
		return schedulerhints.CreateOptsExt{
			CreateOptsBuilder: opts,
			SchedulerHints: schedulerhints.SchedulerHints{
				Group: serverGroupID,
			},
		}
	}
	return opts
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
		if listOpts.ProjectID == "" {
			listOpts.ProjectID = is.projectID
		}
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

		if len(SGList) == 0 {
			return nil, fmt.Errorf("security group %s not found", sg.Name)
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

func getServerNetworks(networkClient *gophercloud.ServiceClient, networkParams []infrav1.NetworkParam) ([]infrav1.Network, error) {
	var nets []infrav1.Network
	for _, net := range networkParams {
		opts := networks.ListOpts(net.Filter)
		opts.ID = net.UUID
		ids, err := networking.GetNetworkIDsByFilter(networkClient, &opts)
		if err != nil {
			return nil, err
		}
		for _, netID := range ids {
			if net.Subnets == nil {
				nets = append(nets, infrav1.Network{
					ID: netID,
				})
				continue
			}

			for _, subnet := range net.Subnets {
				subnetOpts := subnets.ListOpts(subnet.Filter)
				subnetOpts.ID = subnet.UUID
				subnetOpts.NetworkID = netID
				subnetsByFilter, err := networking.GetSubnetsByFilter(networkClient, &subnetOpts)
				if err != nil {
					return nil, err
				}
				for _, subnetByFilter := range subnetsByFilter {
					nets = append(nets, infrav1.Network{
						ID: subnetByFilter.NetworkID,
						Subnet: &infrav1.Subnet{
							ID: subnetByFilter.ID,
						},
					})
				}
			}
		}
	}
	return nets, nil
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

func createPort(is *Service, clusterName string, name string, net *infrav1.Network, securityGroups *[]string) (ports.Port, error) {
	portCreateOpts := ports.CreateOpts{
		Name:           name,
		NetworkID:      net.ID,
		SecurityGroups: securityGroups,
		Description:    fmt.Sprintf("Created by cluster-api-provider-openstack cluster %s", clusterName),
	}
	if net.Subnet.ID != "" {
		portCreateOpts.FixedIPs = []ports.IP{{SubnetID: net.Subnet.ID}}
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
	err := floatingips.AssociateInstance(s.computeClient, instanceID, opts).ExtractErr()
	if err != nil {
		return err
	}
	return nil
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

	return s.deleteInstance(parsed.ID())
}

func (s *Service) deleteInstance(serverID string) error {
	// get instance port id
	allInterfaces, err := attachinterfaces.List(s.computeClient, serverID).AllPages()
	if err != nil {
		return err
	}
	instanceInterfaces, err := attachinterfaces.ExtractInterfaces(allInterfaces)
	if err != nil {
		return err
	}
	if len(instanceInterfaces) < 1 {
		return servers.Delete(s.computeClient, serverID).ExtractErr()
	}

	trunkSupport, err := getTrunkSupport(s)
	if err != nil {
		return fmt.Errorf("obtaining network extensions: %v", err)
	}
	// get and delete trunks
	for _, port := range instanceInterfaces {
		err := attachinterfaces.Delete(s.computeClient, serverID, port.PortID).ExtractErr()
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
	return servers.Delete(s.computeClient, serverID).ExtractErr()
}

func (s *Service) GetInstance(resourceID string) (instance *infrav1.Instance, err error) {
	if resourceID == "" {
		return nil, fmt.Errorf("resourceId should be specified to  get detail")
	}
	server, err := servers.Get(s.computeClient, resourceID).Extract()
	if err != nil {
		if capoerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get server %q detail failed: %v", resourceID, err)

	}
	i, err := serverToInstance(server)
	if err != nil {
		return nil, err
	}
	return i, err
}

func (s *Service) InstanceExists(name string) (instance *infrav1.Instance, err error) {

	var listOpts servers.ListOpts
	if name != "" {
		listOpts = servers.ListOpts{
			Name: name,
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
	instanceList := []*infrav1.Instance{}
	for _, server := range serverList {
		var server = server
		i, err := serverToInstance(&server)
		if err != nil {
			return nil, err
		}
		instanceList = append(instanceList, i)
	}
	if len(instanceList) == 0 {
		return nil, nil
	}
	return instanceList[0], nil
}

// deduplicate takes a slice of input strings and filters out any duplicate
// string occurrences, for example making ["a", "b", "a", "c"] become ["a", "b",
// "c"].
func deduplicate(sequence []string) []string {
	var unique []string
	set := make(map[string]bool)

	for _, s := range sequence {
		if _, ok := set[s]; !ok {
			unique = append(unique, s)
			set[s] = true
		}
	}

	return unique
}

func getTimeout(name string, timeout int) time.Duration {
	if v := os.Getenv(name); v != "" {
		timeout, err := strconv.Atoi(v)
		if err == nil {
			return time.Duration(timeout)
		}
	}
	return time.Duration(timeout)
}
