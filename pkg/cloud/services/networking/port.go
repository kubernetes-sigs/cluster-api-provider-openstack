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
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsecurity"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha5"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
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

func (s *Service) GetOrCreatePort(eventObject runtime.Object, clusterName string, portName string, net infrav1.Network, instanceSecurityGroups *[]string, instanceTags []string) (*ports.Port, error) {
	existingPorts, err := s.client.ListPort(ports.ListOpts{
		Name:      portName,
		NetworkID: net.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("searching for existing port for server: %v", err)
	}

	if len(existingPorts) == 1 {
		return &existingPorts[0], nil
	}

	if len(existingPorts) > 1 {
		return nil, fmt.Errorf("multiple ports found with name \"%s\"", portName)
	}

	// no port found, so create the port
	portOpts := net.PortOpts
	if portOpts == nil {
		portOpts = &infrav1.PortOpts{}
	}

	description := portOpts.Description
	if description == "" {
		description = names.GetDescription(clusterName)
	}

	var securityGroups *[]string
	addressPairs := []ports.AddressPair{}
	if portOpts.DisablePortSecurity == nil || !*portOpts.DisablePortSecurity {
		for _, ap := range portOpts.AllowedAddressPairs {
			addressPairs = append(addressPairs, ports.AddressPair{
				IPAddress:  ap.IPAddress,
				MACAddress: ap.MACAddress,
			})
		}
		securityGroups, err = s.CollectPortSecurityGroups(eventObject, portOpts.SecurityGroups, portOpts.SecurityGroupFilters)
		if err != nil {
			return nil, err
		}
		// inherit port security groups from the instance if not explicitly specified
		if securityGroups == nil || len(*securityGroups) == 0 {
			securityGroups = instanceSecurityGroups
		}
	}

	var fixedIPs interface{}
	if len(portOpts.FixedIPs) > 0 {
		fips := make([]ports.IP, 0, len(portOpts.FixedIPs)+1)
		for _, fixedIP := range portOpts.FixedIPs {
			subnetID, err := s.getSubnetIDForFixedIP(fixedIP.Subnet, net.ID)
			if err != nil {
				return nil, err
			}
			fips = append(fips, ports.IP{
				SubnetID:  subnetID,
				IPAddress: fixedIP.IPAddress,
			})
		}
		if net.Subnet.ID != "" {
			fips = append(fips, ports.IP{SubnetID: net.Subnet.ID})
		}
		fixedIPs = fips
	}

	var createOpts ports.CreateOptsBuilder
	createOpts = ports.CreateOpts{
		Name:                portName,
		NetworkID:           net.ID,
		Description:         description,
		AdminStateUp:        portOpts.AdminStateUp,
		MACAddress:          portOpts.MACAddress,
		TenantID:            portOpts.TenantID,
		ProjectID:           portOpts.ProjectID,
		SecurityGroups:      securityGroups,
		AllowedAddressPairs: addressPairs,
		FixedIPs:            fixedIPs,
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
		HostID:            portOpts.HostID,
		VNICType:          portOpts.VNICType,
		Profile:           getPortProfile(portOpts.Profile),
	}

	port, err := s.client.CreatePort(createOpts)
	if err != nil {
		record.Warnf(eventObject, "FailedCreatePort", "Failed to create port %s: %v", portName, err)
		return nil, err
	}

	var tags []string
	tags = append(tags, instanceTags...)
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

	opts := subnet.ToListOpt()
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

func getPortProfile(p map[string]string) map[string]interface{} {
	portProfile := make(map[string]interface{})
	for k, v := range p {
		portProfile[k] = v
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

func (s *Service) DeletePort(eventObject runtime.Object, portID string) error {
	var err error
	err = util.PollImmediate(retryIntervalPortDelete, timeoutPortDelete, func() (bool, error) {
		err = s.client.DeletePort(portID)
		if err != nil {
			if capoerrors.IsNotFound(err) {
				record.Eventf(eventObject, "SuccessfulDeletePort", "Port with id %d did not exist", portID)
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

func (s *Service) GarbageCollectErrorInstancesPort(eventObject runtime.Object, instanceName string) error {
	portList, err := s.client.ListPort(ports.ListOpts{
		Name: instanceName,
	})
	if err != nil {
		return err
	}
	for _, p := range portList {
		if err := s.DeletePort(eventObject, p.ID); err != nil {
			return err
		}
	}

	return nil
}

// CollectPortSecurityGroups collects distinct securityGroups from port.SecurityGroups and port.SecurityGroupFilter fields.
func (s *Service) CollectPortSecurityGroups(eventObject runtime.Object, portSecurityGroups *[]string, portSecurityGroupFilters []infrav1.SecurityGroupParam) (*[]string, error) {
	var allSecurityGroupIDs []string
	// security groups provided with the portSecurityGroupFilters fields
	securityGroupFiltersByID, err := s.GetSecurityGroups(portSecurityGroupFilters)
	if err != nil {
		return portSecurityGroups, fmt.Errorf("error getting security groups: %v", err)
	}
	allSecurityGroupIDs = append(allSecurityGroupIDs, securityGroupFiltersByID...)
	securityGroupCount := 0
	// security groups provided with the portSecurityGroups fields
	if portSecurityGroups != nil {
		allSecurityGroupIDs = append(allSecurityGroupIDs, *portSecurityGroups...)
	}
	// generate unique values
	uids := make(map[string]int)
	for _, sg := range allSecurityGroupIDs {
		if sg == "" {
			continue
		}
		// count distinct values
		_, ok := uids[sg]
		if !ok {
			securityGroupCount++
		}
		uids[sg] = 1
	}
	distinctSecurityGroupIDs := make([]string, 0, securityGroupCount)
	// collect distict values
	for key := range uids {
		if key == "" {
			continue
		}
		distinctSecurityGroupIDs = append(distinctSecurityGroupIDs, key)
	}
	return &distinctSecurityGroupIDs, nil
}
