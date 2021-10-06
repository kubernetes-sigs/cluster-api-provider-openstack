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
	"os"
	"strconv"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/attachinterfaces"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/schedulerhints"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/utils/openstack/compute/v2/flavors"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha4"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/metrics"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
)

const (
	timeoutInstanceCreate       = 5
	retryIntervalInstanceStatus = 10 * time.Second
	timeoutInstanceDelete       = 5 * time.Minute
)

func (s *Service) CreateInstance(openStackCluster *infrav1.OpenStackCluster, machine *clusterv1.Machine, openStackMachine *infrav1.OpenStackMachine, clusterName string, userData string) (instance *InstanceStatus, err error) {
	if openStackMachine == nil {
		return nil, fmt.Errorf("create Options need be specified to create instace")
	}

	if machine.Spec.FailureDomain == nil {
		return nil, fmt.Errorf("failure domain not set")
	}

	instanceSpec := InstanceSpec{
		Name:          openStackMachine.Name,
		Image:         openStackMachine.Spec.Image,
		Flavor:        openStackMachine.Spec.Flavor,
		SSHKeyName:    openStackMachine.Spec.SSHKeyName,
		UserData:      userData,
		Metadata:      openStackMachine.Spec.ServerMetadata,
		ConfigDrive:   openStackMachine.Spec.ConfigDrive != nil && *openStackMachine.Spec.ConfigDrive,
		FailureDomain: *machine.Spec.FailureDomain,
		RootVolume:    openStackMachine.Spec.RootVolume,
		Subnet:        openStackMachine.Spec.Subnet,
		ServerGroupID: openStackMachine.Spec.ServerGroupID,
	}

	if openStackMachine.Spec.Trunk {
		trunkSupport, err := s.networkingService.GetTrunkSupport()
		if err != nil {
			return nil, fmt.Errorf("there was an issue verifying whether trunk support is available, please disable it: %v", err)
		}
		if !trunkSupport {
			return nil, fmt.Errorf("there is no trunk support. Please disable it")
		}
		instanceSpec.Trunk = trunkSupport
	}

	machineTags := []string{}

	// Append machine specific tags
	machineTags = append(machineTags, openStackMachine.Spec.Tags...)

	// Append cluster scope tags
	machineTags = append(machineTags, openStackCluster.Spec.Tags...)

	// tags need to be unique or the "apply tags" call will fail.
	machineTags = deduplicate(machineTags)

	instanceSpec.Tags = machineTags

	// Get security groups
	securityGroups, err := s.networkingService.GetSecurityGroups(openStackMachine.Spec.SecurityGroups)
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
	instanceSpec.SecurityGroups = securityGroups

	nets, err := s.constructNetworks(openStackCluster, openStackMachine)
	if err != nil {
		return nil, err
	}
	instanceSpec.Networks = nets

	return s.createInstance(openStackMachine, clusterName, &instanceSpec)
}

// constructNetworks builds an array of networks from the network, subnet and ports items in the machine spec.
// If no networks or ports are in the spec, returns a single network item for a network connection to the default cluster network.
func (s *Service) constructNetworks(openStackCluster *infrav1.OpenStackCluster, openStackMachine *infrav1.OpenStackMachine) ([]infrav1.Network, error) {
	var nets []infrav1.Network
	if len(openStackMachine.Spec.Networks) > 0 {
		var err error
		nets, err = s.getServerNetworks(openStackMachine.Spec.Networks)
		if err != nil {
			return nil, err
		}
	}
	for i, port := range openStackMachine.Spec.Ports {
		pOpts := &openStackMachine.Spec.Ports[i]
		// No Trunk field specified for the port, inherit openStackMachine.Spec.Trunk.
		if pOpts.Trunk == nil {
			pOpts.Trunk = &openStackMachine.Spec.Trunk
		}
		if port.NetworkID != "" {
			nets = append(nets, infrav1.Network{
				ID:       port.NetworkID,
				Subnet:   &infrav1.Subnet{},
				PortOpts: pOpts,
			})
		} else {
			nets = append(nets, infrav1.Network{
				ID: openStackCluster.Status.Network.ID,
				Subnet: &infrav1.Subnet{
					ID: openStackCluster.Status.Network.Subnet.ID,
				},
				PortOpts: pOpts,
			})
		}
	}
	// no networks or ports found in the spec, so create a port on the cluster network
	if len(nets) == 0 {
		nets = []infrav1.Network{{
			ID: openStackCluster.Status.Network.ID,
			Subnet: &infrav1.Subnet{
				ID: openStackCluster.Status.Network.Subnet.ID,
			},
			PortOpts: &infrav1.PortOpts{
				Trunk: &openStackMachine.Spec.Trunk,
			},
		}}
	}
	return nets, nil
}

func (s *Service) createInstance(eventObject runtime.Object, clusterName string, instanceSpec *InstanceSpec) (*InstanceStatus, error) {
	accessIPv4 := ""
	portList := []servers.Network{}

	for i, network := range instanceSpec.Networks {
		if network.ID == "" {
			return nil, fmt.Errorf("no network was found or provided. Please check your machine configuration and try again")
		}
		iTags := []string{}
		if len(instanceSpec.Tags) > 0 {
			iTags = instanceSpec.Tags
		}
		portName := getPortName(instanceSpec.Name, network.PortOpts, i)
		port, err := s.networkingService.GetOrCreatePort(eventObject, clusterName, portName, network, &instanceSpec.SecurityGroups, iTags)
		if err != nil {
			return nil, err
		}

		for _, fip := range port.FixedIPs {
			if fip.SubnetID == instanceSpec.Subnet {
				accessIPv4 = fip.IPAddress
			}
		}

		portList = append(portList, servers.Network{
			Port: port.ID,
		})
	}

	if instanceSpec.Subnet != "" && accessIPv4 == "" {
		if err := s.deletePorts(eventObject, portList); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("no ports with fixed IPs found on Subnet %q", instanceSpec.Subnet)
	}

	imageID, err := s.getImageID(instanceSpec.Image)
	if err != nil {
		return nil, fmt.Errorf("create new server: %v", err)
	}

	flavorID, err := flavors.IDFromName(s.computeClient, instanceSpec.Flavor)
	if err != nil {
		return nil, fmt.Errorf("error getting flavor id from flavor name %s: %v", instanceSpec.Flavor, err)
	}

	var serverCreateOpts servers.CreateOptsBuilder = servers.CreateOpts{
		Name:             instanceSpec.Name,
		ImageRef:         imageID,
		FlavorRef:        flavorID,
		AvailabilityZone: instanceSpec.FailureDomain,
		Networks:         portList,
		UserData:         []byte(instanceSpec.UserData),
		SecurityGroups:   instanceSpec.SecurityGroups,
		Tags:             instanceSpec.Tags,
		Metadata:         instanceSpec.Metadata,
		ConfigDrive:      &instanceSpec.ConfigDrive,
		AccessIPv4:       accessIPv4,
	}

	serverCreateOpts = applyRootVolume(serverCreateOpts, instanceSpec.RootVolume)

	serverCreateOpts = applyServerGroupID(serverCreateOpts, instanceSpec.ServerGroupID)

	mc := metrics.NewMetricPrometheusContext("server", "create")

	server, err := servers.Create(s.computeClient, keypairs.CreateOptsExt{
		CreateOptsBuilder: serverCreateOpts,
		KeyName:           instanceSpec.SSHKeyName,
	}).Extract()

	if mc.ObserveRequest(err) != nil {
		serverErr := err
		if err = s.deletePorts(eventObject, portList); err != nil {
			return nil, fmt.Errorf("error creating OpenStack instance: %v, error deleting ports: %v", serverErr, err)
		}
		return nil, fmt.Errorf("error creating Openstack instance: %v", serverErr)
	}
	instanceCreateTimeout := getTimeout("CLUSTER_API_OPENSTACK_INSTANCE_CREATE_TIMEOUT", timeoutInstanceCreate)
	instanceCreateTimeout *= time.Minute
	var createdInstance *InstanceStatus
	err = util.PollImmediate(retryIntervalInstanceStatus, instanceCreateTimeout, func() (bool, error) {
		createdInstance, err = s.GetInstanceStatus(server.ID)
		if err != nil {
			if capoerrors.IsRetryable(err) {
				return false, nil
			}
			return false, err
		}
		if createdInstance.State() == infrav1.InstanceStateError {
			return false, fmt.Errorf("error creating OpenStack instance %s, status changed to error", createdInstance.ID())
		}
		return createdInstance.State() == infrav1.InstanceStateActive, nil
	})
	if err != nil {
		record.Warnf(eventObject, "FailedCreateServer", "Failed to create server %s: %v", createdInstance.Name(), err)
		return nil, err
	}

	record.Eventf(eventObject, "SuccessfulCreateServer", "Created server %s with id %s", createdInstance.Name(), createdInstance.ID())
	return createdInstance, nil
}

// getPortName appends a suffix to an instance name in order to try and get a unique name per port.
func getPortName(instanceName string, opts *infrav1.PortOpts, netIndex int) string {
	if opts != nil && opts.NameSuffix != "" {
		return fmt.Sprintf("%s-%s", instanceName, opts.NameSuffix)
	}
	return fmt.Sprintf("%s-%d", instanceName, netIndex)
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

func (s *Service) getServerNetworks(networkParams []infrav1.NetworkParam) ([]infrav1.Network, error) {
	var nets []infrav1.Network

	addSubnet := func(netID, subnetID string) {
		nets = append(nets, infrav1.Network{
			ID: netID,
			Subnet: &infrav1.Subnet{
				ID: subnetID,
			},
		})
	}

	addSubnets := func(networkParam infrav1.NetworkParam, netID string) error {
		if len(networkParam.Subnets) == 0 && netID != "" {
			addSubnet(netID, "")
			return nil
		}

		for _, subnet := range networkParam.Subnets {
			// Don't lookup subnet if it was specified explicitly by UUID
			if subnet.UUID != "" {
				// If subnet was supplied by UUID then network
				// must also have been supplied by UUID.
				if netID == "" {
					return fmt.Errorf("validation error adding network for subnet %s: "+
						"network uuid must be specified when subnet uuid is specified", subnet.UUID)
				}

				addSubnet(netID, subnet.UUID)
			} else {
				subnetOpts := subnets.ListOpts(subnet.Filter)
				if netID != "" {
					subnetOpts.NetworkID = netID
				}
				subnetsByFilter, err := s.networkingService.GetSubnetsByFilter(&subnetOpts)
				if err != nil {
					return err
				}
				for _, subnetByFilter := range subnetsByFilter {
					addSubnet(subnetByFilter.NetworkID, subnetByFilter.ID)
				}
			}
		}

		return nil
	}

	for _, networkParam := range networkParams {
		// Don't lookup network if we specified one explicitly by UUID
		// Don't lookup network if we didn't specify a network filter
		// If there is no explicit network UUID and no network filter,
		// we will look for subnets matching any given subnet filters in
		// all networks.
		if networkParam.UUID != "" || networkParam.Filter == (infrav1.Filter{}) {
			if err := addSubnets(networkParam, networkParam.UUID); err != nil {
				return nil, err
			}
			continue
		}
		opts := networks.ListOpts(networkParam.Filter)
		ids, err := s.networkingService.GetNetworkIDsByFilter(&opts)
		if err != nil {
			return nil, err
		}
		for _, netID := range ids {
			if err := addSubnets(networkParam, netID); err != nil {
				return nil, err
			}
		}
	}
	return nets, nil
}

// Helper function for getting image ID from name.
func (s *Service) getImageID(imageName string) (string, error) {
	if imageName == "" {
		return "", nil
	}

	opts := images.ListOpts{
		Name: imageName,
	}

	mc := metrics.NewMetricPrometheusContext("image", "list")
	pages, err := images.List(s.imagesClient, opts).AllPages()
	if mc.ObserveRequest(err) != nil {
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

// GetManagementPort returns the port which is used for management and external
// traffic. Cluster floating IPs must be associated with this port.
func (s *Service) GetManagementPort(openStackCluster *infrav1.OpenStackCluster, instanceStatus *InstanceStatus) (*ports.Port, error) {
	ns, err := instanceStatus.NetworkStatus()
	if err != nil {
		return nil, err
	}
	allPorts, err := s.networkingService.GetPortFromInstanceIP(instanceStatus.ID(), ns.IP(openStackCluster.Status.Network.Name))
	if err != nil {
		return nil, fmt.Errorf("lookup management port for server %s: %w", instanceStatus.ID(), err)
	}
	if len(allPorts) < 1 {
		return nil, fmt.Errorf("did not find management port for server %s", instanceStatus.ID())
	}
	return &allPorts[0], nil
}

func (s *Service) DeleteInstance(eventObject runtime.Object, instance *InstanceStatus) error {
	mc := metrics.NewMetricPrometheusContext("server_os_interface", "list")
	instanceIdentifier := instance.InstanceIdentifier()
	allInterfaces, err := attachinterfaces.List(s.computeClient, instanceIdentifier.ID).AllPages()
	if mc.ObserveRequest(err) != nil {
		return err
	}
	instanceInterfaces, err := attachinterfaces.ExtractInterfaces(allInterfaces)
	if err != nil {
		return err
	}

	trunkSupport, err := s.networkingService.GetTrunkSupport()
	if err != nil {
		return fmt.Errorf("obtaining network extensions: %v", err)
	}
	// get and delete trunks
	for _, port := range instanceInterfaces {
		if err = s.deleteAttachInterface(eventObject, instanceIdentifier, port.PortID); err != nil {
			return err
		}

		if trunkSupport {
			if err = s.networkingService.DeleteTrunk(eventObject, port.PortID); err != nil {
				return err
			}
		}

		if err = s.networkingService.DeletePort(eventObject, port.PortID); err != nil {
			return err
		}
	}

	// delete port of error instance
	if instance.State() == infrav1.InstanceStateError {
		if err := s.networkingService.GarbageCollectErrorInstancesPort(eventObject, instance.Name()); err != nil {
			return err
		}
	}

	return s.deleteInstance(eventObject, instanceIdentifier)
}

func (s *Service) deletePorts(eventObject runtime.Object, nets []servers.Network) error {
	for _, n := range nets {
		if err := s.networkingService.DeletePort(eventObject, n.Port); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) deleteAttachInterface(eventObject runtime.Object, instance *InstanceIdentifier, portID string) error {
	mc := metrics.NewMetricPrometheusContext("server_os_interface", "delete")
	err := attachinterfaces.Delete(s.computeClient, instance.ID, portID).ExtractErr()
	if mc.ObserveRequestIgnoreNotFoundorConflict(err) != nil {
		if capoerrors.IsNotFound(err) {
			record.Eventf(eventObject, "SuccessfulDeleteAttachInterface", "Attach interface did not exist: instance %s, port %s", instance.ID, portID)
			return nil
		}
		if capoerrors.IsConflict(err) {
			// we don't want to block deletion because of Conflict
			// due to instance must be paused/active/shutoff in order to detach interface
			return nil
		}
		record.Warnf(eventObject, "FailedDeleteAttachInterface", "Failed to delete attach interface: instance %s, port %s: %v", instance.ID, portID, err)
		return err
	}

	record.Eventf(eventObject, "SuccessfulDeleteAttachInterface", "Deleted attach interface: instance %s, port %s", instance.ID, portID)
	return nil
}

func (s *Service) deleteInstance(eventObject runtime.Object, instance *InstanceIdentifier) error {
	mc := metrics.NewMetricPrometheusContext("server", "delete")
	err := servers.Delete(s.computeClient, instance.ID).ExtractErr()
	if mc.ObserveRequestIgnoreNotFound(err) != nil {
		if capoerrors.IsNotFound(err) {
			record.Eventf(eventObject, "SuccessfulDeleteServer", "Server %s with id %s did not exist", instance.Name, instance.ID)
			return nil
		}
		record.Warnf(eventObject, "FailedDeleteServer", "Failed to deleted server %s with id %s: %v", instance.Name, instance.ID, err)
		return err
	}

	err = util.PollImmediate(retryIntervalInstanceStatus, timeoutInstanceDelete, func() (bool, error) {
		i, err := s.GetInstanceStatus(instance.ID)
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

func (s *Service) GetInstanceStatus(resourceID string) (instance *InstanceStatus, err error) {
	if resourceID == "" {
		return nil, fmt.Errorf("resourceId should be specified to get detail")
	}

	mc := metrics.NewMetricPrometheusContext("server", "get")
	var server ServerExt
	err = servers.Get(s.computeClient, resourceID).ExtractInto(&server)
	if mc.ObserveRequestIgnoreNotFound(err) != nil {
		if capoerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get server %q detail failed: %v", resourceID, err)
	}

	return &InstanceStatus{&server, s.logger}, nil
}

func (s *Service) GetInstanceStatusByName(eventObject runtime.Object, name string) (instance *InstanceStatus, err error) {
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

	mc := metrics.NewMetricPrometheusContext("server", "list")
	allPages, err := servers.List(s.computeClient, listOpts).AllPages()
	if mc.ObserveRequest(err) != nil {
		return nil, fmt.Errorf("get server list: %v", err)
	}
	var serverList []ServerExt
	err = servers.ExtractServersInto(allPages, &serverList)
	if err != nil {
		return nil, fmt.Errorf("extract server list: %v", err)
	}

	if len(serverList) > 1 {
		record.Warnf(eventObject, "DuplicateServerNames", "Found %d servers with name '%s'. This is likely to cause errors.", len(serverList), name)
	}

	// Return the first returned server, if any
	for i := range serverList {
		return &InstanceStatus{&serverList[i], s.logger}, nil
	}
	return nil, nil
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
