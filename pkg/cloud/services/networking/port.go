/*
Copyright 2021 The Kubernetes Authors.

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

package networking

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsecurity"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/filterconvert"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/names"
)

const (
	timeoutPortDelete       = 3 * time.Minute
	retryIntervalPortDelete = 5 * time.Second
)

// GetPortFromInstanceIP returns at most one port attached to the instance with given ID
// and with the IP address provided.
func (s *Service) GetPortFromInstanceIP(instanceID string, ip string) ([]ports.Port, error) {
	portOpts := ports.ListOpts{
		DeviceID: instanceID,
		FixedIPs: []ports.FixedIPOpts{
			{
				IPAddress: ip,
			},
		},
		Limit: 1,
	}
	return s.client.ListPort(portOpts)
}

func (s *Service) GetPortForExternalNetwork(instanceID string, externalNetworkID string) (*ports.Port, error) {
	instancePortsOpts := ports.ListOpts{
		DeviceID: instanceID,
	}
	instancePorts, err := s.client.ListPort(instancePortsOpts)
	if err != nil {
		return nil, fmt.Errorf("lookup ports for server %s: %w", instanceID, err)
	}

	for _, instancePort := range instancePorts {
		networkPortsOpts := ports.ListOpts{
			NetworkID:   instancePort.NetworkID,
			DeviceOwner: "network:router_interface",
		}

		networkPorts, err := s.client.ListPort(networkPortsOpts)
		if err != nil {
			return nil, fmt.Errorf("lookup ports for network %s: %w", instancePort.NetworkID, err)
		}

		for _, networkPort := range networkPorts {
			// Check if the instance port and the network port share a subnet
			matchingSubnet := false
			for _, fixedIP := range instancePort.FixedIPs {
				for _, networkFixedIP := range networkPort.FixedIPs {
					if fixedIP.SubnetID == networkFixedIP.SubnetID {
						matchingSubnet = true
						break
					}
				}
				if matchingSubnet {
					break
				}
			}
			if !matchingSubnet {
				continue
			}

			router, err := s.client.GetRouter(networkPort.DeviceID)
			if err != nil {
				return nil, fmt.Errorf("lookup router %s: %w", networkPort.DeviceID, err)
			}

			if router.GatewayInfo.NetworkID == externalNetworkID {
				return &instancePort, nil
			}
		}
	}
	return nil, nil
}

func (s *Service) CreatePort(eventObject runtime.Object, clusterName string, portName string, portOpts *infrav1.PortOpts, defaultSecurityGroups []string, baseTags []string) (*ports.Port, error) {
	var err error
	networkID := portOpts.Network.ID

	var description string
	if portOpts.Description != nil {
		description = *portOpts.Description
	} else {
		description = names.GetDescription(clusterName)
	}

	var securityGroups []string
	addressPairs := []ports.AddressPair{}
	if portOpts.DisablePortSecurity == nil || !*portOpts.DisablePortSecurity {
		for _, ap := range portOpts.AllowedAddressPairs {
			addressPairs = append(addressPairs, ports.AddressPair{
				IPAddress:  ap.IPAddress,
				MACAddress: pointer.StringDeref(ap.MACAddress, ""),
			})
		}
		if portOpts.SecurityGroups != nil {
			securityGroups, err = s.GetSecurityGroups(portOpts.SecurityGroups)
			if err != nil {
				return nil, fmt.Errorf("error getting security groups: %v", err)
			}
		}
		// inherit port security groups from the instance if not explicitly specified
		if len(securityGroups) == 0 {
			securityGroups = defaultSecurityGroups
		}
	}

	var fixedIPs interface{}
	if len(portOpts.FixedIPs) > 0 {
		fips := make([]ports.IP, 0, len(portOpts.FixedIPs)+1)
		for _, fixedIP := range portOpts.FixedIPs {
			subnetID, err := s.getSubnetIDForFixedIP(fixedIP.Subnet, networkID)
			if err != nil {
				return nil, err
			}
			fips = append(fips, ports.IP{
				SubnetID:  subnetID,
				IPAddress: pointer.StringDeref(fixedIP.IPAddress, ""),
			})
		}
		fixedIPs = fips
	}

	var valueSpecs *map[string]string
	if len(portOpts.ValueSpecs) > 0 {
		vs := make(map[string]string, len(portOpts.ValueSpecs))
		for _, valueSpec := range portOpts.ValueSpecs {
			vs[valueSpec.Key] = valueSpec.Value
		}
		valueSpecs = &vs
	}

	var createOpts ports.CreateOptsBuilder

	// Gophercloud expects a *[]string. We translate a nil slice to a nil pointer.
	var securityGroupsPtr *[]string
	if securityGroups != nil {
		securityGroupsPtr = &securityGroups
	}

	createOpts = ports.CreateOpts{
		Name:                  portName,
		NetworkID:             networkID,
		Description:           description,
		AdminStateUp:          portOpts.AdminStateUp,
		MACAddress:            pointer.StringDeref(portOpts.MACAddress, ""),
		SecurityGroups:        securityGroupsPtr,
		AllowedAddressPairs:   addressPairs,
		FixedIPs:              fixedIPs,
		ValueSpecs:            valueSpecs,
		PropagateUplinkStatus: portOpts.PropagateUplinkStatus,
	}

	if portOpts.DisablePortSecurity != nil {
		portSecurity := !*portOpts.DisablePortSecurity
		createOpts = portsecurity.PortCreateOptsExt{
			CreateOptsBuilder:   createOpts,
			PortSecurityEnabled: &portSecurity,
		}
	}

	createOpts = portsbinding.CreateOptsExt{
		CreateOptsBuilder: createOpts,
		HostID:            pointer.StringDeref(portOpts.HostID, ""),
		VNICType:          pointer.StringDeref(portOpts.VNICType, ""),
		Profile:           getPortProfile(portOpts.Profile),
	}

	port, err := s.client.CreatePort(createOpts)
	if err != nil {
		record.Warnf(eventObject, "FailedCreatePort", "Failed to create port %s: %v", portName, err)
		return nil, err
	}

	var tags []string
	tags = append(tags, baseTags...)
	tags = append(tags, portOpts.Tags...)
	if len(tags) > 0 {
		if err = s.replaceAllAttributesTags(eventObject, portResource, port.ID, tags); err != nil {
			record.Warnf(eventObject, "FailedReplaceTags", "Failed to replace port tags %s: %v", portName, err)
			return nil, err
		}
	}
	record.Eventf(eventObject, "SuccessfulCreatePort", "Created port %s with id %s", port.Name, port.ID)
	if portOpts.Trunk != nil && *portOpts.Trunk {
		trunk, err := s.getOrCreateTrunk(eventObject, clusterName, port.Name, port.ID)
		if err != nil {
			record.Warnf(eventObject, "FailedCreateTrunk", "Failed to create trunk for port %s: %v", portName, err)
			return nil, err
		}
		if err = s.replaceAllAttributesTags(eventObject, trunkResource, trunk.ID, tags); err != nil {
			record.Warnf(eventObject, "FailedReplaceTags", "Failed to replace trunk tags %s: %v", portName, err)
			return nil, err
		}
	}

	return port, nil
}

func (s *Service) getSubnetIDForFixedIP(subnet *infrav1.SubnetFilter, networkID string) (string, error) {
	if subnet == nil {
		return "", nil
	}
	// Do not query for subnets if UUID is already provided
	if subnet.ID != "" {
		return subnet.ID, nil
	}

	opts := filterconvert.SubnetFilterToListOpts(subnet)
	opts.NetworkID = networkID
	subnets, err := s.client.ListSubnet(opts)
	if err != nil {
		return "", err
	}

	switch len(subnets) {
	case 0:
		return "", fmt.Errorf("subnet query %v, returns no subnets", *subnet)
	case 1:
		return subnets[0].ID, nil
	default:
		return "", fmt.Errorf("subnet query %v, returns too many subnets: %v", *subnet, subnets)
	}
}

func getPortProfile(p *infrav1.BindingProfile) map[string]interface{} {
	if p == nil {
		return nil
	}

	portProfile := make(map[string]interface{})

	// if p.OVSHWOffload is true, we need to set the profile
	// to enable hardware offload for the port
	if pointer.BoolDeref(p.OVSHWOffload, false) {
		portProfile["capabilities"] = []string{"switchdev"}
	}
	if pointer.BoolDeref(p.TrustedVF, false) {
		portProfile["trusted"] = true
	}

	// We need return nil if there is no profiles
	// to have backward compatible defaults.
	// To set profiles, your tenant needs this permission:
	// rule:create_port and rule:create_port:binding:profile
	if len(portProfile) == 0 {
		return nil
	}
	return portProfile
}

// DeletePort deletes the Neutron port with the given ID.
func (s *Service) DeletePort(eventObject runtime.Object, portID string) error {
	var err error
	err = wait.PollUntilContextTimeout(context.TODO(), retryIntervalPortDelete, timeoutPortDelete, true, func(_ context.Context) (bool, error) {
		err = s.client.DeletePort(portID)
		if err != nil {
			if capoerrors.IsNotFound(err) {
				record.Eventf(eventObject, "SuccessfulDeletePort", "Port with id %d did not exist", portID)
				// this is success so we return without another try
				return true, nil
			}
			if capoerrors.IsRetryable(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if err != nil {
		record.Warnf(eventObject, "FailedDeletePort", "Failed to delete port with id %s: %v", portID, err)
		return err
	}

	record.Eventf(eventObject, "SuccessfulDeletePort", "Deleted port with id %s", portID)
	return nil
}

// DeleteTrunk deletes the Neutron trunk and port with the given ID.
func (s *Service) DeleteInstanceTrunkAndPort(eventObject runtime.Object, port infrav1.PortStatus, trunkSupported bool) error {
	if trunkSupported {
		if err := s.DeleteTrunk(eventObject, port.ID); err != nil {
			return fmt.Errorf("error deleting trunk of port %s: %v", port.ID, err)
		}
	}
	if err := s.DeletePort(eventObject, port.ID); err != nil {
		return fmt.Errorf("error deleting port %s: %v", port.ID, err)
	}

	return nil
}

// DeleteClusterPorts deletes all ports created for the cluster.
func (s *Service) DeleteClusterPorts(openStackCluster *infrav1.OpenStackCluster) error {
	// If the network is not ready, do nothing
	if openStackCluster.Status.Network == nil || openStackCluster.Status.Network.ID == "" {
		return nil
	}
	networkID := openStackCluster.Status.Network.ID

	portList, err := s.client.ListPort(ports.ListOpts{
		NetworkID:   networkID,
		DeviceOwner: "",
	})
	s.scope.Logger().Info("Deleting cluster ports", "networkID", networkID, "portList", portList)
	if err != nil {
		if capoerrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("list ports of network %q: %v", networkID, err)
	}

	for _, port := range portList {
		if strings.HasPrefix(port.Name, openStackCluster.Name) {
			if err := s.DeletePort(openStackCluster, port.ID); err != nil {
				return fmt.Errorf("error deleting port %s: %v", port.ID, err)
			}
		}
	}

	return nil
}

// GetPortName appends a suffix to an instance name in order to try and get a unique name per port.
func GetPortName(instanceName string, opts *infrav1.PortOpts, netIndex int) string {
	if opts != nil && opts.NameSuffix != nil {
		return fmt.Sprintf("%s-%s", instanceName, *opts.NameSuffix)
	}
	return fmt.Sprintf("%s-%d", instanceName, netIndex)
}

func (s *Service) CreatePorts(eventObject runtime.Object, clusterName, baseName string, securityGroups []infrav1.SecurityGroupFilter, baseTags []string, desiredPorts []infrav1.PortOpts, dependentResources *infrav1.DependentMachineResources) error {
	defaultSecurityGroups, err := s.GetSecurityGroups(securityGroups)
	if err != nil {
		return fmt.Errorf("error getting security groups: %v", err)
	}

	for i := range desiredPorts {
		// Skip creation of ports which already exist
		if i < len(dependentResources.Ports) {
			continue
		}

		portOpts := &desiredPorts[i]
		portName := GetPortName(baseName, portOpts, i)
		// Events are recorded in CreatePort
		port, err := s.CreatePort(eventObject, clusterName, portName, portOpts, defaultSecurityGroups, baseTags)
		if err != nil {
			return err
		}

		dependentResources.Ports = append(dependentResources.Ports, infrav1.PortStatus{
			ID: port.ID,
		})
	}

	return nil
}

// ConstructPorts builds an array of ports from the instance spec.
// If no ports are in the spec, returns a single port for a network connection to the default cluster network.
func (s *Service) ConstructPorts(openStackCluster *infrav1.OpenStackCluster, ports []infrav1.PortOpts, trunkEnabled bool, trunkSupported bool) ([]infrav1.PortOpts, error) {
	// If no network is specified, return nil
	if openStackCluster.Status.Network == nil {
		return nil, nil
	}

	// Ensure user-specified ports have all required fields
	ports, err := s.normalizePorts(ports, openStackCluster, trunkEnabled)
	if err != nil {
		return nil, err
	}

	// no networks or ports found in the spec, so create a port on the cluster network
	if len(ports) == 0 {
		port := infrav1.PortOpts{
			Network: &infrav1.NetworkFilter{
				ID: openStackCluster.Status.Network.ID,
			},
			Trunk: &trunkEnabled,
		}
		for _, subnet := range openStackCluster.Status.Network.Subnets {
			port.FixedIPs = append(port.FixedIPs, infrav1.FixedIP{
				Subnet: &infrav1.SubnetFilter{
					ID: subnet.ID,
				},
			})
		}
		ports = []infrav1.PortOpts{port}
	}

	// trunk support is required if any port has trunk enabled
	portUsesTrunk := func() bool {
		for _, port := range ports {
			if port.Trunk != nil && *port.Trunk {
				return true
			}
		}
		return false
	}
	if portUsesTrunk() {
		if !trunkSupported {
			return nil, fmt.Errorf("there is no trunk support. please ensure that the trunk extension is enabled in your OpenStack deployment")
		}
	}

	return ports, nil
}

// normalizePorts ensures that a user-specified PortOpts has all required fields set. Specifically it:
// - sets the Trunk field to the instance spec default if not specified
// - sets the Network ID field if not specified.
func (s *Service) normalizePorts(ports []infrav1.PortOpts, openStackCluster *infrav1.OpenStackCluster, trunkEnabled bool) ([]infrav1.PortOpts, error) {
	normalizedPorts := make([]infrav1.PortOpts, 0, len(ports))
	for i := range ports {
		// Deep copy the port to avoid mutating the original
		port := ports[i].DeepCopy()

		// No Trunk field specified for the port, inherit the machine default
		if port.Trunk == nil {
			port.Trunk = &trunkEnabled
		}

		if err := s.normalizePortTarget(port, openStackCluster, i); err != nil {
			return nil, err
		}

		normalizedPorts = append(normalizedPorts, *port)
	}
	return normalizedPorts, nil
}

// normalizePortTarget ensures that the port has a network ID.
func (s *Service) normalizePortTarget(port *infrav1.PortOpts, openStackCluster *infrav1.OpenStackCluster, portIdx int) error {
	// Treat no Network and empty Network the same
	noNetwork := port.Network.IsEmpty()

	// No network or subnets defined: use cluster defaults
	if noNetwork && len(port.FixedIPs) == 0 {
		port.Network = &infrav1.NetworkFilter{
			ID: openStackCluster.Status.Network.ID,
		}
		for _, subnet := range openStackCluster.Status.Network.Subnets {
			port.FixedIPs = append(port.FixedIPs, infrav1.FixedIP{
				Subnet: &infrav1.SubnetFilter{
					ID: subnet.ID,
				},
			})
		}

		return nil
	}

	// No network, but fixed IPs are defined(we handled the no fixed
	// IPs case above): try to infer network from a subnet
	if noNetwork {
		s.scope.Logger().V(4).Info("No network defined for port, attempting to infer from subnet", "port", portIdx)

		// Look for a unique subnet defined in FixedIPs.  If we find one
		// we can use it to infer the network ID. We don't need to worry
		// here about the case where different FixedIPs have different
		// networks because that will cause an error later when we try
		// to create the port.
		networkID, err := func() (string, error) {
			for i, fixedIP := range port.FixedIPs {
				if fixedIP.Subnet == nil {
					continue
				}

				subnet, err := s.GetSubnetByFilter(fixedIP.Subnet)
				if err != nil {
					// Multiple matches might be ok later when we restrict matches to a single network
					if errors.Is(err, ErrMultipleMatches) {
						s.scope.Logger().V(4).Info("Couldn't infer network from subnet", "subnetIndex", i, "err", err)
						continue
					}

					return "", err
				}

				// Cache the subnet ID in the FixedIP
				fixedIP.Subnet.ID = subnet.ID
				return subnet.NetworkID, nil
			}

			// TODO: This is a spec error: it should set the machine to failed
			return "", fmt.Errorf("port %d has no network and unable to infer from fixed IPs", portIdx)
		}()
		if err != nil {
			return err
		}

		port.Network = &infrav1.NetworkFilter{
			ID: networkID,
		}

		return nil
	}

	// Nothing to do if network ID is already set
	if port.Network.ID != "" {
		return nil
	}

	// Network is defined by Filter
	networkListOpts := filterconvert.NetworkFilterToListOpts(port.Network)
	netIDs, err := s.GetNetworkIDsByFilter(networkListOpts)
	if err != nil {
		return err
	}

	// TODO: These are spec errors: they should set the machine to failed
	if len(netIDs) > 1 {
		return fmt.Errorf("network filter for port %d returns more than one result", portIdx)
	} else if len(netIDs) == 0 {
		return fmt.Errorf("network filter for port %d returns no networks", portIdx)
	}

	port.Network.ID = netIDs[0]

	return nil
}

// IsTrunkExtSupported verifies trunk setup on the OpenStack deployment.
func (s *Service) IsTrunkExtSupported() (trunknSupported bool, err error) {
	trunkSupport, err := s.GetTrunkSupport()
	if err != nil {
		return false, fmt.Errorf("there was an issue verifying whether trunk support is available, Please try again later: %v", err)
	}
	if !trunkSupport {
		return false, nil
	}
	return true, nil
}

// AdoptPorts looks for ports in desiredPorts which were previously created, and adds them to dependentResources.Ports.
// A port matches if it has the same name and network ID as the desired port.
func (s *Service) AdoptPorts(scope *scope.WithLogger, baseName string, desiredPorts []infrav1.PortOpts, dependentResources *infrav1.DependentMachineResources) error {
	// We can skip adoption if the ports are already in the status
	if len(desiredPorts) == len(dependentResources.Ports) {
		return nil
	}

	scope.Logger().V(5).Info("Adopting ports")

	// We create ports in order and adopt them in order in PortsStatus.
	// This means that if port N doesn't exist we know that ports >N don't exist.
	// We can therefore stop searching for ports once we find one that doesn't exist.
	for i, port := range desiredPorts {
		// check if the port is in status first and if it is, skip it
		if i < len(dependentResources.Ports) {
			scope.Logger().V(5).Info("Port already in status, skipping it", "port index", i)
			continue
		}

		portOpts := &desiredPorts[i]
		portName := GetPortName(baseName, portOpts, i)
		ports, err := s.client.ListPort(ports.ListOpts{
			Name:      portName,
			NetworkID: port.Network.ID,
		})
		if err != nil {
			return fmt.Errorf("searching for existing port %s in network %s: %v", portName, port.Network.ID, err)
		}
		// if the port is not found, we stop the adoption of ports since the rest of the ports will not be found either
		// and will be created after the adoption
		if len(ports) == 0 {
			scope.Logger().V(5).Info("Port not found, stopping the adoption of ports", "port index", i)
			return nil
		}
		if len(ports) > 1 {
			return fmt.Errorf("found multiple ports with name %s", portName)
		}

		// The desired port was found, so we add it to the status
		portID := ports[0].ID
		scope.Logger().Info("Adopted previously created port which was not in status", "port index", i, "portID", portID)
		dependentResources.Ports = append(dependentResources.Ports, infrav1.PortStatus{ID: portID})
	}

	return nil
}
