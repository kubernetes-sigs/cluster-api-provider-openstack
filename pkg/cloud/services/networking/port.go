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

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha4"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/names"
)

const (
	timeoutPortDelete       = 3 * time.Minute
	retryIntervalPortDelete = 5 * time.Second
)

func (s *Service) getPort(portID string) (port *ports.Port, err error) {
	if portID == "" {
		return nil, fmt.Errorf("portID should be specified to get detail")
	}
	port, err = s.client.GetPort(portID)
	if err != nil {
		if capoerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get port %q detail failed: %v", portID, err)
	}
	return port, nil
}

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

func (s *Service) GetOrCreatePort(eventObject runtime.Object, clusterName string, portName string, net infrav1.Network, instanceSecurityGroups *[]string, tags []string) (*ports.Port, error) {
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

		securityGroups = portOpts.SecurityGroups

		// inherit port security groups from the instance if not explicitly specified
		if securityGroups == nil {
			securityGroups = instanceSecurityGroups
		}
	}

	var fixedIPs interface{}
	if len(portOpts.FixedIPs) > 0 {
		fips := make([]ports.IP, 0, len(portOpts.FixedIPs)+1)
		for _, fixedIP := range portOpts.FixedIPs {
			fips = append(fips, ports.IP{
				SubnetID:  fixedIP.SubnetID,
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

	record.Eventf(eventObject, "SuccessfulCreatePort", "Created port %s with id %s", port.Name, port.ID)
	if portOpts.Trunk != nil && *portOpts.Trunk {
		trunk, err := s.getOrCreateTrunk(eventObject, clusterName, port.Name, port.ID)
		if err != nil {
			record.Warnf(eventObject, "FailedCreateTrunk", "Failed to create trunk for port %s: %v", portName, err)
			return nil, err
		}
		if err = s.replaceAllAttributesTags(eventObject, trunk.ID, tags); err != nil {
			record.Warnf(eventObject, "FailedReplaceTags", "Failed to replace trunk tags %s: %v", portName, err)
			return nil, err
		}
	}

	return port, nil
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
	port, err := s.getPort(portID)
	if err != nil {
		return err
	}
	if port == nil {
		return nil
	}

	err = util.PollImmediate(retryIntervalPortDelete, timeoutPortDelete, func() (bool, error) {
		err := s.client.DeletePort(port.ID)
		if err != nil {
			if capoerrors.IsNotFound(err) {
				record.Eventf(eventObject, "SuccessfulDeletePort", "Port %s with id %d did not exist", port.Name, port.ID)
			}
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
