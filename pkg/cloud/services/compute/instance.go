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
	"github.com/gophercloud/utils/openstack/compute/v2/flavors"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha4"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
)

const (
	TimeoutInstanceCreate       = 5
	RetryIntervalInstanceStatus = 10 * time.Second

	TimeoutTrunkDelete       = 3 * time.Minute
	RetryIntervalTrunkDelete = 5 * time.Second

	TimeoutPortDelete       = 3 * time.Minute
	RetryIntervalPortDelete = 5 * time.Second

	TimeoutInstanceDelete = 5 * time.Minute
)

func (s *Service) CreateInstance(openStackCluster *infrav1.OpenStackCluster, machine *clusterv1.Machine, openStackMachine *infrav1.OpenStackMachine, clusterName string, userData string) (instance *infrav1.Instance, err error) {
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
		Subnet:        openStackMachine.Spec.Subnet,
	}

	if openStackMachine.Spec.Trunk {
		trunkSupport, err := s.getTrunkSupport()
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
	securityGroups, err := s.getSecurityGroups(openStackMachine.Spec.SecurityGroups)
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
		nets, err = s.getServerNetworks(openStackMachine.Spec.Networks)
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

	out, err := s.createInstance(openStackMachine, clusterName, input)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) createInstance(eventObject runtime.Object, clusterName string, i *infrav1.Instance) (*infrav1.Instance, error) {
	accessIPv4 := ""
	portList := []servers.Network{}

	for _, network := range *i.Networks {
		if network.ID == "" {
			return nil, fmt.Errorf("no network was found or provided. Please check your machine configuration and try again")
		}

		port, err := s.getOrCreatePort(eventObject, clusterName, i.Name, network, i.SecurityGroups)
		if err != nil {
			return nil, err
		}

		if i.Trunk {
			trunk, err := s.getOrCreateTrunk(eventObject, i.Name, port.ID)
			if err != nil {
				return nil, err
			}

			if err = s.replaceAllAttributesTags(eventObject, trunk.ID, i.Tags); err != nil {
				return nil, err
			}
		}

		for _, fip := range port.FixedIPs {
			if fip.SubnetID == i.Subnet {
				accessIPv4 = fip.IPAddress
			}
		}

		portList = append(portList, servers.Network{
			Port: port.ID,
		})
	}

	if i.Subnet != "" && accessIPv4 == "" {
		if err := s.deletePorts(eventObject, portList); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("no ports with fixed IPs found on Subnet %q", i.Subnet)
	}

	imageID, err := s.getImageID(i.Image)
	if err != nil {
		return nil, fmt.Errorf("create new server: %v", err)
	}

	flavorID, err := flavors.IDFromName(s.computeClient, i.Flavor)
	if err != nil {
		return nil, fmt.Errorf("error getting flavor id from flavor name %s: %v", i.Flavor, err)
	}

	var serverCreateOpts servers.CreateOptsBuilder = servers.CreateOpts{
		Name:             i.Name,
		ImageRef:         imageID,
		FlavorRef:        flavorID,
		AvailabilityZone: i.FailureDomain,
		Networks:         portList,
		UserData:         []byte(i.UserData),
		SecurityGroups:   *i.SecurityGroups,
		Tags:             i.Tags,
		Metadata:         i.Metadata,
		ConfigDrive:      i.ConfigDrive,
		AccessIPv4:       accessIPv4,
	}

	serverCreateOpts = applyRootVolume(serverCreateOpts, i.RootVolume)

	serverCreateOpts = applyServerGroupID(serverCreateOpts, i.ServerGroupID)

	server, err := servers.Create(s.computeClient, keypairs.CreateOptsExt{
		CreateOptsBuilder: serverCreateOpts,
		KeyName:           i.SSHKeyName,
	}).Extract()
	if err != nil {
		if err = s.deletePorts(eventObject, portList); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("error creating Openstack instance: %v", err)
	}
	instanceCreateTimeout := getTimeout("CLUSTER_API_OPENSTACK_INSTANCE_CREATE_TIMEOUT", TimeoutInstanceCreate)
	instanceCreateTimeout *= time.Minute
	var instance *infrav1.Instance
	err = util.PollImmediate(RetryIntervalInstanceStatus, instanceCreateTimeout, func() (bool, error) {
		instance, err = s.GetInstance(server.ID)
		if err != nil {
			if capoerrors.IsRetryable(err) {
				return false, nil
			}
			return false, err
		}
		return instance.State == infrav1.InstanceStateActive, nil
	})
	if err != nil {
		record.Warnf(eventObject, "FailedCreateServer", "Failed to create server %s: %v", instance.Name, err)
		return nil, err
	}

	record.Eventf(eventObject, "SuccessfulCreateServer", "Created server %s with id %s", instance.Name, instance.ID)
	return instance, nil
}

// applyRootVolume sets a root volume if the root volume Size is not 0.
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
// spec contains a server group ID.
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

func (s *Service) getTrunkSupport() (bool, error) {
	allPages, err := netext.List(s.networkClient).AllPages()
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

func (s *Service) getSecurityGroups(securityGroupParams []infrav1.SecurityGroupParam) ([]string, error) {
	var sgIDs []string
	for _, sg := range securityGroupParams {
		listOpts := groups.ListOpts(sg.Filter)
		if listOpts.ProjectID == "" {
			listOpts.ProjectID = s.projectID
		}
		listOpts.Name = sg.Name
		listOpts.ID = sg.UUID
		pages, err := groups.List(s.networkClient, listOpts).AllPages()
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

func (s *Service) getServerNetworks(networkParams []infrav1.NetworkParam) ([]infrav1.Network, error) {
	var nets []infrav1.Network
	for _, networkParam := range networkParams {
		opts := networks.ListOpts(networkParam.Filter)
		opts.ID = networkParam.UUID
		ids, err := s.networkingService.GetNetworkIDsByFilter(&opts)
		if err != nil {
			return nil, err
		}
		for _, netID := range ids {
			if networkParam.Subnets == nil {
				nets = append(nets, infrav1.Network{
					ID: netID,
				})
				continue
			}

			for _, subnet := range networkParam.Subnets {
				subnetOpts := subnets.ListOpts(subnet.Filter)
				subnetOpts.ID = subnet.UUID
				subnetOpts.NetworkID = netID
				subnetsByFilter, err := s.networkingService.GetSubnetsByFilter(&subnetOpts)
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

func (s *Service) getOrCreatePort(eventObject runtime.Object, clusterName string, portName string, net infrav1.Network, securityGroups *[]string) (*ports.Port, error) {
	allPages, err := ports.List(s.networkClient, ports.ListOpts{
		Name:      portName,
		NetworkID: net.ID,
	}).AllPages()
	if err != nil {
		return nil, fmt.Errorf("searching for existing port for server: %v", err)
	}
	portList, err := ports.ExtractPorts(allPages)
	if err != nil {
		return nil, fmt.Errorf("searching for existing port for server: %v", err)
	}

	if len(portList) != 0 {
		return &portList[0], nil
	}

	portCreateOpts := ports.CreateOpts{
		Name:           portName,
		NetworkID:      net.ID,
		SecurityGroups: securityGroups,
		Description:    fmt.Sprintf("Created by cluster-api-provider-openstack cluster %s", clusterName),
	}
	if net.Subnet.ID != "" {
		portCreateOpts.FixedIPs = []ports.IP{{SubnetID: net.Subnet.ID}}
	}
	port, err := ports.Create(s.networkClient, portCreateOpts).Extract()
	if err != nil {
		record.Warnf(eventObject, "FailedCreatePort", "Failed to create port %s: %v", portName, err)
		return nil, err
	}

	record.Eventf(eventObject, "SuccessfulCreatePort", "Created port %s with id %s", port.Name, port.ID)
	return port, nil
}

func (s *Service) getOrCreateTrunk(eventObject runtime.Object, trunkName, portID string) (*trunks.Trunk, error) {
	allPages, err := trunks.List(s.networkClient, trunks.ListOpts{
		Name:   trunkName,
		PortID: portID,
	}).AllPages()
	if err != nil {
		return nil, fmt.Errorf("searching for existing trunk for server: %v", err)
	}
	trunkList, err := trunks.ExtractTrunks(allPages)
	if err != nil {
		return nil, fmt.Errorf("searching for existing trunk for server: %v", err)
	}

	if len(trunkList) != 0 {
		return &trunkList[0], nil
	}

	trunkCreateOpts := trunks.CreateOpts{
		Name:   trunkName,
		PortID: portID,
	}
	trunk, err := trunks.Create(s.networkClient, trunkCreateOpts).Extract()
	if err != nil {
		record.Warnf(eventObject, "FailedCreateTrunk", "Failed to create trunk %s: %v", trunkName, err)
		return nil, err
	}

	record.Eventf(eventObject, "SuccessfulCreateTrunk", "Created trunk %s with id %s", trunk.Name, trunk.ID)
	return trunk, nil
}

func (s *Service) replaceAllAttributesTags(eventObject runtime.Object, trunkID string, tags []string) error {
	_, err := attributestags.ReplaceAll(s.networkClient, "trunks", trunkID, attributestags.ReplaceAllOpts{
		Tags: tags,
	}).Extract()
	if err != nil {
		record.Warnf(eventObject, "FailedReplaceAllAttributesTags", "Failed to replace all attributestags, trunk %s: %v", trunkID, err)
		return err
	}

	record.Eventf(eventObject, "SuccessfulReplaceAllAttributeTags", "Replaced all attributestags %s with tags %s", trunkID, tags)
	return nil
}

// Helper function for getting image ID from name.
func (s *Service) getImageID(imageName string) (string, error) {
	if imageName == "" {
		return "", nil
	}

	opts := images.ListOpts{
		Name: imageName,
	}

	pages, err := images.List(s.imagesClient, opts).AllPages()
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

func (s *Service) DeleteInstance(eventObject runtime.Object, instanceName string) error {
	instance, err := s.InstanceExists(instanceName)
	if err != nil {
		return err
	}

	if instance == nil || instance.ID == "" {
		return nil
	}

	allInterfaces, err := attachinterfaces.List(s.computeClient, instance.ID).AllPages()
	if err != nil {
		return err
	}
	instanceInterfaces, err := attachinterfaces.ExtractInterfaces(allInterfaces)
	if err != nil {
		return err
	}
	if len(instanceInterfaces) < 1 {
		return s.deleteInstance(eventObject, instance.ID)
	}

	trunkSupport, err := s.getTrunkSupport()
	if err != nil {
		return fmt.Errorf("obtaining network extensions: %v", err)
	}
	// get and delete trunks
	for _, port := range instanceInterfaces {
		if err = s.deleteAttachInterface(eventObject, instance.ID, port.PortID); err != nil {
			return err
		}

		if trunkSupport {
			if err = s.deleteTrunk(eventObject, port.PortID); err != nil {
				return err
			}
		}

		if err = s.deletePort(eventObject, port.PortID); err != nil {
			return err
		}
	}

	return s.deleteInstance(eventObject, instance.ID)
}

func (s *Service) deletePort(eventObject runtime.Object, portID string) error {
	port, err := s.getPort(portID)
	if err != nil {
		return err
	}
	if port == nil {
		return nil
	}

	err = util.PollImmediate(RetryIntervalPortDelete, TimeoutPortDelete, func() (bool, error) {
		err := ports.Delete(s.networkClient, port.ID).ExtractErr()
		if err != nil {
			if capoerrors.IsRetryable(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if err != nil {
		record.Warnf(eventObject, "FailedDeletePort", "Failed to delete port %s with id %s: %v", port.Name, port.ID, err)
		return err
	}

	record.Eventf(eventObject, "SuccessfulDeletePort", "Deleted port %s with id %s", port.Name, port.ID)
	return nil
}

func (s *Service) deletePorts(eventObject runtime.Object, nets []servers.Network) error {
	for _, n := range nets {
		if err := s.deletePort(eventObject, n.Port); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) deleteAttachInterface(eventObject runtime.Object, instanceID, portID string) error {
	instance, err := s.GetInstance(instanceID)
	if err != nil {
		return err
	}
	if instance == nil || instance.ID == "" {
		return nil
	}

	port, err := s.getPort(portID)
	if err != nil {
		return err
	}
	if port == nil {
		return nil
	}

	err = attachinterfaces.Delete(s.computeClient, instanceID, portID).ExtractErr()
	if err != nil {
		if capoerrors.IsNotFound(err) {
			return nil
		}
		record.Warnf(eventObject, "FailedDeleteAttachInterface", "Failed to delete attach interface: instance %s, port %s: %v", instance.ID, port.ID, err)
		return err
	}

	record.Eventf(eventObject, "SuccessfulDeleteAttachInterface", "Deleted attach interface: instance %s, port %s", instance.ID, port.ID)
	return nil
}

func (s *Service) deleteTrunk(eventObject runtime.Object, portID string) error {
	port, err := s.getPort(portID)
	if err != nil {
		return err
	}
	if port == nil {
		return nil
	}

	listOpts := trunks.ListOpts{
		PortID: port.ID,
	}
	trunkList, err := trunks.List(s.networkClient, listOpts).AllPages()
	if err != nil {
		return err
	}
	trunkInfo, err := trunks.ExtractTrunks(trunkList)
	if err != nil {
		return err
	}
	if len(trunkInfo) != 1 {
		return nil
	}

	err = util.PollImmediate(RetryIntervalTrunkDelete, TimeoutTrunkDelete, func() (bool, error) {
		if err := trunks.Delete(s.networkClient, trunkInfo[0].ID).ExtractErr(); err != nil {
			if capoerrors.IsRetryable(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if err != nil {
		record.Warnf(eventObject, "FailedDeleteTrunk", "Failed to delete trunk %s with id %s: %v", trunkInfo[0].Name, trunkInfo[0].ID, err)
		return err
	}

	record.Eventf(eventObject, "SuccessfulDeleteTrunk", "Deleted trunk %s with id %s", trunkInfo[0].Name, trunkInfo[0].ID)
	return nil
}

func (s *Service) getPort(portID string) (port *ports.Port, err error) {
	if portID == "" {
		return nil, fmt.Errorf("portID should be specified to get detail")
	}
	port, err = ports.Get(s.networkClient, portID).Extract()
	if err != nil {
		if capoerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get port %q detail failed: %v", portID, err)
	}
	return port, nil
}

func (s *Service) deleteInstance(eventObject runtime.Object, instanceID string) error {
	instance, err := s.GetInstance(instanceID)
	if err != nil {
		return err
	}

	if instance == nil || instance.ID == "" {
		return nil
	}

	if err = servers.Delete(s.computeClient, instance.ID).ExtractErr(); err != nil {
		record.Warnf(eventObject, "FailedDeleteServer", "Failed to deleted server %s with id %s: %v", instance.Name, instance.ID, err)
		return err
	}

	err = util.PollImmediate(RetryIntervalInstanceStatus, TimeoutInstanceDelete, func() (bool, error) {
		i, err := s.GetInstance(instance.ID)
		if err != nil {
			return false, err
		}
		if i != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		record.Warnf(eventObject, "FailedDeleteServer", "Failed to delete server %s with id %s: %v", instance.Name, instance.ID, err)
		return err
	}

	record.Eventf(eventObject, "SuccessfulDeleteServer", "Deleted server %s with id %s", instance.Name, instance.ID)
	return nil
}

func (s *Service) GetInstance(resourceID string) (instance *infrav1.Instance, err error) {
	if resourceID == "" {
		return nil, fmt.Errorf("resourceId should be specified to get detail")
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
			// The name parameter to /servers is a regular expression. Unless we
			// explicitly specify a whole string match this will be a substring
			// match.
			Name: fmt.Sprintf("^%s$", name),
		}
	} else {
		listOpts = servers.ListOpts{}
	}

	allPages, err := servers.List(s.computeClient, listOpts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("get server list: %v", err)
	}
	serverList, err := servers.ExtractServers(allPages)
	if err != nil {
		return nil, fmt.Errorf("extract server list: %v", err)
	}
	instanceList := []*infrav1.Instance{}
	for _, server := range serverList {
		server := server
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
	addrMap, err := GetIPFromInstance(*v)
	if err != nil {
		return i, err
	}
	i.IP = addrMap["internal"]
	if addrMap["floating"] != "" {
		i.FloatingIP = addrMap["floating"]
	}
	return i, nil
}

func GetIPFromInstance(v servers.Server) (map[string]string, error) {
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
		var networkList []interface{}
		err = json.Unmarshal(list, &networkList)
		if err != nil {
			return nil, fmt.Errorf("extract IP from instance err: %v", err)
		}
		for _, network := range networkList {
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
