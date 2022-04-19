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

	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/schedulerhints"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha5"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
)

const (
	retryIntervalInstanceStatus = 10 * time.Second
	timeoutInstanceCreate       = 5
	timeoutInstanceDelete       = 5 * time.Minute
)

// constructNetworks builds an array of networks from the network, subnet and ports items in the instance spec.
// If no networks or ports are in the spec, returns a single network item for a network connection to the default cluster network.
func (s *Service) constructNetworks(openStackCluster *infrav1.OpenStackCluster, instanceSpec *InstanceSpec) ([]infrav1.Network, error) {
	trunkRequired := false

	nets, err := s.getServerNetworks(instanceSpec.Networks)
	if err != nil {
		return nil, err
	}

	for i := range instanceSpec.Ports {
		port := &instanceSpec.Ports[i]
		// No Trunk field specified for the port, inherit openStackMachine.Spec.Trunk.
		if port.Trunk == nil {
			port.Trunk = &instanceSpec.Trunk
		}
		if *port.Trunk {
			trunkRequired = true
		}
		if port.Network != nil {
			netID := port.Network.ID
			if netID == "" {
				netIDs, err := s.networkingService.GetNetworkIDsByFilter(port.Network.ToListOpt())
				if err != nil {
					return nil, err
				}
				if len(netIDs) > 1 {
					return nil, fmt.Errorf("network filter for port %s returns more than one result", port.NameSuffix)
				} else if len(netIDs) == 0 {
					return nil, fmt.Errorf("network filter for port %s returns no networks", port.NameSuffix)
				}
				netID = netIDs[0]
			}
			nets = append(nets, infrav1.Network{
				ID:       netID,
				Subnet:   &infrav1.Subnet{},
				PortOpts: port,
			})
		} else {
			nets = append(nets, infrav1.Network{
				ID: openStackCluster.Status.Network.ID,
				Subnet: &infrav1.Subnet{
					ID: openStackCluster.Status.Network.Subnet.ID,
				},
				PortOpts: port,
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
				Trunk: &instanceSpec.Trunk,
			},
		}}
		trunkRequired = instanceSpec.Trunk
	}

	if trunkRequired {
		trunkSupported, err := s.isTrunkExtSupported()
		if err != nil {
			return nil, err
		}
		if !trunkSupported {
			return nil, fmt.Errorf("there is no trunk support. please ensure that the trunk extension is enabled in your OpenStack deployment")
		}
	}

	return nets, nil
}

func (s *Service) CreateInstance(eventObject runtime.Object, openStackCluster *infrav1.OpenStackCluster, instanceSpec *InstanceSpec, clusterName string) (*InstanceStatus, error) {
	return s.createInstanceImpl(eventObject, openStackCluster, instanceSpec, clusterName, retryIntervalInstanceStatus)
}

func (s *Service) createInstanceImpl(eventObject runtime.Object, openStackCluster *infrav1.OpenStackCluster, instanceSpec *InstanceSpec, clusterName string, retryInterval time.Duration) (*InstanceStatus, error) {
	var server *ServerExt
	accessIPv4 := ""
	portList := []servers.Network{}

	if instanceSpec.Subnet != "" && accessIPv4 == "" {
		return nil, fmt.Errorf("no ports with fixed IPs found on Subnet %q", instanceSpec.Subnet)
	}

	imageID, err := s.getImageID(instanceSpec.ImageUUID, instanceSpec.Image)
	if err != nil {
		return nil, fmt.Errorf("error getting image ID: %v", err)
	}

	flavorID, err := s.computeService.GetFlavorIDFromName(instanceSpec.Flavor)
	if err != nil {
		return nil, fmt.Errorf("error getting flavor id from flavor name %s: %v", instanceSpec.Flavor, err)
	}

	// Ensure we delete the ports we created if we haven't created the server.
	defer func() {
		if server != nil {
			return
		}

		if err := s.deletePorts(eventObject, portList); err != nil {
			s.scope.Logger.V(4).Error(err, "Failed to clean up ports after failure")
		}
	}()

	nets, err := s.constructNetworks(openStackCluster, instanceSpec)
	if err != nil {
		return nil, err
	}

	securityGroups, err := s.networkingService.GetSecurityGroups(instanceSpec.SecurityGroups)
	if err != nil {
		return nil, fmt.Errorf("error getting security groups: %v", err)
	}

	for i, network := range nets {
		if network.ID == "" {
			return nil, fmt.Errorf("no network was found or provided. Please check your machine configuration and try again")
		}
		iTags := []string{}
		if len(instanceSpec.Tags) > 0 {
			iTags = instanceSpec.Tags
		}
		portName := getPortName(instanceSpec.Name, network.PortOpts, i)
		port, err := s.networkingService.GetOrCreatePort(eventObject, clusterName, portName, network, &securityGroups, iTags)
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

	var serverCreateOpts servers.CreateOptsBuilder = servers.CreateOpts{
		Name:             instanceSpec.Name,
		ImageRef:         imageID,
		FlavorRef:        flavorID,
		AvailabilityZone: instanceSpec.FailureDomain,
		Networks:         portList,
		UserData:         []byte(instanceSpec.UserData),
		SecurityGroups:   securityGroups,
		Tags:             instanceSpec.Tags,
		Metadata:         instanceSpec.Metadata,
		ConfigDrive:      &instanceSpec.ConfigDrive,
		AccessIPv4:       accessIPv4,
	}

	volume, err := s.getOrCreateRootVolume(eventObject, instanceSpec, imageID)
	if err != nil {
		return nil, fmt.Errorf("error in get or create root volume: %w", err)
	}

	instanceCreateTimeout := getTimeout("CLUSTER_API_OPENSTACK_INSTANCE_CREATE_TIMEOUT", timeoutInstanceCreate)
	instanceCreateTimeout *= time.Minute

	// Wait for volume to become available
	if volume != nil {
		err = util.PollImmediate(retryIntervalInstanceStatus, instanceCreateTimeout, func() (bool, error) {
			createdVolume, err := s.computeService.GetVolume(volume.ID)
			if err != nil {
				if capoerrors.IsRetryable(err) {
					return false, nil
				}
				return false, err
			}

			switch createdVolume.Status {
			case "available":
				return true, nil
			case "error":
				return false, fmt.Errorf("volume %s is in error state", volume.ID)
			default:
				return false, nil
			}
		})
		if err != nil {
			return nil, fmt.Errorf("volume %s did not become available: %w", volume.ID, err)
		}
	}

	serverCreateOpts = applyRootVolume(serverCreateOpts, volume)

	serverCreateOpts = applyServerGroupID(serverCreateOpts, instanceSpec.ServerGroupID)

	server, err = s.computeService.CreateServer(keypairs.CreateOptsExt{
		CreateOptsBuilder: serverCreateOpts,
		KeyName:           instanceSpec.SSHKeyName,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating Openstack instance: %v", err)
	}

	var createdInstance *InstanceStatus
	err = util.PollImmediate(retryInterval, instanceCreateTimeout, func() (bool, error) {
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

func rootVolumeName(instanceName string) string {
	return fmt.Sprintf("%s-root", instanceName)
}

func hasRootVolume(rootVolume *infrav1.RootVolume) bool {
	return rootVolume != nil && rootVolume.Size > 0
}

func (s *Service) getVolumeByName(name string) (*volumes.Volume, error) {
	listOpts := volumes.ListOpts{
		AllTenants: false,
		Name:       name,
		TenantID:   s.scope.ProjectID,
	}
	volumeList, err := s.computeService.ListVolumes(listOpts)
	if err != nil {
		return nil, fmt.Errorf("error listing volumes: %w", err)
	}
	if len(volumeList) > 1 {
		return nil, fmt.Errorf("expected to find a single volume called %s; found %d", name, len(volumeList))
	}
	if len(volumeList) == 0 {
		return nil, nil
	}
	return &volumeList[0], nil
}

func (s *Service) getOrCreateRootVolume(eventObject runtime.Object, instanceSpec *InstanceSpec, imageID string) (*volumes.Volume, error) {
	rootVolume := instanceSpec.RootVolume
	if !hasRootVolume(rootVolume) {
		return nil, nil
	}

	name := rootVolumeName(instanceSpec.Name)
	size := rootVolume.Size

	volume, err := s.getVolumeByName(name)
	if err != nil {
		return nil, err
	}
	if volume != nil {
		if volume.Size != size {
			return nil, fmt.Errorf("exected to find volume %s with size %d; found size %d", name, size, volume.Size)
		}

		s.scope.Logger.Info("using existing root volume %s", name)
		return volume, nil
	}

	availabilityZone := instanceSpec.FailureDomain
	if rootVolume.AvailabilityZone != "" {
		availabilityZone = rootVolume.AvailabilityZone
	}

	createOpts := volumes.CreateOpts{
		Size:             rootVolume.Size,
		Description:      fmt.Sprintf("Root volume for %s", instanceSpec.Name),
		Name:             rootVolumeName(instanceSpec.Name),
		ImageID:          imageID,
		Multiattach:      false,
		AvailabilityZone: availabilityZone,
		VolumeType:       rootVolume.VolumeType,
	}
	volume, err = s.computeService.CreateVolume(createOpts)
	if err != nil {
		record.Eventf(eventObject, "FailedCreateVolume", "Failed to create root volume; size=%d imageID=%s err=%v", size, imageID, err)
		return nil, err
	}
	record.Eventf(eventObject, "SuccessfulCreateVolume", "Created root volume; id=%s", volume.ID)
	return volume, err
}

// applyRootVolume sets a root volume if the root volume Size is not 0.
func applyRootVolume(opts servers.CreateOptsBuilder, volume *volumes.Volume) servers.CreateOptsBuilder {
	if volume == nil {
		return opts
	}

	block := bootfromvolume.BlockDevice{
		SourceType:          bootfromvolume.SourceVolume,
		BootIndex:           0,
		UUID:                volume.ID,
		DeleteOnTermination: true,
		DestinationType:     bootfromvolume.DestinationVolume,
	}
	return bootfromvolume.CreateOptsExt{
		CreateOptsBuilder: opts,
		BlockDevice:       []bootfromvolume.BlockDevice{block},
	}
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
				subnetOpts := subnet.Filter.ToListOpt()
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
		if networkParam.UUID != "" || networkParam.Filter == (infrav1.NetworkFilter{}) {
			if err := addSubnets(networkParam, networkParam.UUID); err != nil {
				return nil, err
			}
			continue
		}
		opts := networkParam.Filter.ToListOpt()
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

// Helper function for getting image id from name.
func (s *Service) getImageIDFromName(imageName string) (string, error) {
	var opts images.ListOpts

	opts.Name = imageName

	allImages, err := s.computeService.ListImages(opts)
	if err != nil {
		return "", err
	}

	switch len(allImages) {
	case 0:
		return "", fmt.Errorf("no image with the Name %s could be found", imageName)
	case 1:
		return allImages[0].ID, nil
	default:
		// this should never happen
		return "", fmt.Errorf("too many images with the name, %s, were found", imageName)
	}
}

// Helper function for getting image ID from name or ID.
func (s *Service) getImageID(imageUUID, imageName string) (string, error) {
	if imageUUID != "" {
		// we return imageUUID without check
		return imageUUID, nil
	} else if imageName != "" {
		return s.getImageIDFromName(imageName)
	}

	return "", nil
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

func (s *Service) DeleteInstance(eventObject runtime.Object, instanceSpec *InstanceSpec, instanceStatus *InstanceStatus) error {
	if instanceStatus == nil {
		/*
			We create a boot-from-volume instance in 2 steps:
			1. Create the volume
			2. Create the instance with the created root volume and set DeleteOnTermination

			This introduces a new failure mode which has implications for safely deleting instances: we
			might create the volume, but the instance create fails. This would leave us with a dangling
			volume with no instance.

			To handle this safely, we ensure that we never remove a machine finalizer until all resources
			associated with the instance, including a root volume, have been deleted. To achieve this:
			* We always call DeleteInstance when reconciling a delete, regardless of
			  whether the instance exists or not.
			* If the instance was already deleted we check that the volume is also gone.

			Note that we don't need to separately delete the root volume when deleting the instance because
			DeleteOnTermination will ensure it is deleted in that case.
		*/
		rootVolume := instanceSpec.RootVolume
		if hasRootVolume(rootVolume) {
			name := rootVolumeName(instanceSpec.Name)
			volume, err := s.getVolumeByName(name)
			if err != nil {
				return err
			}
			if volume == nil {
				return nil
			}

			s.scope.Logger.Info("deleting dangling root volume %s(%s)", volume.Name, volume.ID)
			return s.computeService.DeleteVolume(volume.ID, volumes.DeleteOpts{})
		}

		return nil
	}

	instanceInterfaces, err := s.computeService.ListAttachedInterfaces(instanceStatus.ID())
	if err != nil {
		return err
	}

	trunkSupported, err := s.isTrunkExtSupported()
	if err != nil {
		return fmt.Errorf("obtaining network extensions: %v", err)
	}

	// get and delete trunks
	for _, port := range instanceInterfaces {
		if err = s.deleteAttachInterface(eventObject, instanceStatus.InstanceIdentifier(), port.PortID); err != nil {
			return err
		}

		if trunkSupported {
			if err = s.networkingService.DeleteTrunk(eventObject, port.PortID); err != nil {
				return err
			}
		}
		if err = s.networkingService.DeletePort(eventObject, port.PortID); err != nil {
			return err
		}
	}

	// delete port of error instance
	if instanceStatus.State() == infrav1.InstanceStateError {
		if err := s.networkingService.GarbageCollectErrorInstancesPort(eventObject, instanceStatus.Name()); err != nil {
			return err
		}
	}

	return s.deleteInstance(eventObject, instanceStatus.InstanceIdentifier())
}

func (s *Service) deletePorts(eventObject runtime.Object, nets []servers.Network) error {
	trunkSupported, err := s.isTrunkExtSupported()
	if err != nil {
		return err
	}

	for _, n := range nets {
		if trunkSupported {
			if err = s.networkingService.DeleteTrunk(eventObject, n.Port); err != nil {
				return err
			}
		}
		if err := s.networkingService.DeletePort(eventObject, n.Port); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) deleteAttachInterface(eventObject runtime.Object, instance *InstanceIdentifier, portID string) error {
	err := s.computeService.DeleteAttachedInterface(instance.ID, portID)
	if err != nil {
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
	err := s.computeService.DeleteServer(instance.ID)
	if err != nil {
		if capoerrors.IsNotFound(err) {
			record.Eventf(eventObject, "SuccessfulDeleteServer", "Server %s with id %s did not exist", instance.Name, instance.ID)
			return nil
		}
		record.Warnf(eventObject, "FailedDeleteServer", "Failed to delete server %s with id %s: %v", instance.Name, instance.ID, err)
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

	server, err := s.computeService.GetServer(resourceID)
	if err != nil {
		if capoerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get server %q detail failed: %v", resourceID, err)
	}

	return &InstanceStatus{server, s.scope.Logger}, nil
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

	serverList, err := s.computeService.ListServers(listOpts)
	if err != nil {
		return nil, fmt.Errorf("get server list: %v", err)
	}

	if len(serverList) > 1 {
		record.Warnf(eventObject, "DuplicateServerNames", "Found %d servers with name '%s'. This is likely to cause errors.", len(serverList), name)
	}

	// Return the first returned server, if any
	for i := range serverList {
		return &InstanceStatus{&serverList[i], s.scope.Logger}, nil
	}
	return nil, nil
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

// isTrunkExtSupported verifies trunk setup on the OpenStack deployment.
func (s *Service) isTrunkExtSupported() (trunknSupported bool, err error) {
	trunkSupport, err := s.networkingService.GetTrunkSupport()
	if err != nil {
		return false, fmt.Errorf("there was an issue verifying whether trunk support is available, Please try again later: %v", err)
	}
	if !trunkSupport {
		return false, nil
	}
	return true, nil
}
