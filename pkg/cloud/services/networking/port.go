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

	"github.com/gophercloud/gophercloud/openstack/common/extensions"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	netext "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha4"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/metrics"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/names"
)

const (
	TimeoutTrunkDelete       = 3 * time.Minute
	RetryIntervalTrunkDelete = 5 * time.Second

	TimeoutPortDelete       = 3 * time.Minute
	RetryIntervalPortDelete = 5 * time.Second
)

func (s *Service) GetPort(portID string) (port *ports.Port, err error) {
	if portID == "" {
		return nil, fmt.Errorf("portID should be specified to get detail")
	}
	port, err = ports.Get(s.client, portID).Extract()
	if err != nil {
		if capoerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get port %q detail failed: %v", portID, err)
	}
	return port, nil
}

func (s *Service) GetTrunkSupport() (bool, error) {
	allPages, err := netext.List(s.client).AllPages()
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

func (s *Service) GetOrCreatePort(eventObject runtime.Object, clusterName string, portName string, net infrav1.Network, instanceSecurityGroups *[]string) (*ports.Port, error) {
	allPages, err := ports.List(s.client, ports.ListOpts{
		Name:      portName,
		NetworkID: net.ID,
	}).AllPages()
	if err != nil {
		return nil, fmt.Errorf("searching for existing port for server: %v", err)
	}
	existingPorts, err := ports.ExtractPorts(allPages)
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

	// inherit port security groups from the instance if not explicitly specified
	securityGroups := portOpts.SecurityGroups
	if securityGroups == nil {
		securityGroups = instanceSecurityGroups
	}

	createOpts := ports.CreateOpts{
		Name:                portName,
		NetworkID:           net.ID,
		Description:         description,
		AdminStateUp:        portOpts.AdminStateUp,
		MACAddress:          portOpts.MACAddress,
		TenantID:            portOpts.TenantID,
		ProjectID:           portOpts.ProjectID,
		SecurityGroups:      securityGroups,
		AllowedAddressPairs: []ports.AddressPair{},
	}

	for _, ap := range portOpts.AllowedAddressPairs {
		createOpts.AllowedAddressPairs = append(createOpts.AllowedAddressPairs, ports.AddressPair{
			IPAddress:  ap.IPAddress,
			MACAddress: ap.MACAddress,
		})
	}

	fixedIPs := make([]ports.IP, 0, len(portOpts.FixedIPs)+1)
	for _, fixedIP := range portOpts.FixedIPs {
		fixedIPs = append(fixedIPs, ports.IP{
			SubnetID:  fixedIP.SubnetID,
			IPAddress: fixedIP.IPAddress,
		})
	}

	if net.Subnet.ID != "" {
		fixedIPs = append(fixedIPs, ports.IP{SubnetID: net.Subnet.ID})
	}
	if len(fixedIPs) > 0 {
		createOpts.FixedIPs = fixedIPs
	}
	mc := metrics.NewMetricPrometheusContext("port", "create")

	port, err := ports.Create(s.client, portsbinding.CreateOptsExt{
		CreateOptsBuilder: createOpts,
		HostID:            portOpts.HostID,
		VNICType:          portOpts.VNICType,
		Profile:           nil,
	}).Extract()
	if mc.ObserveRequest(err) != nil {
		record.Warnf(eventObject, "FailedCreatePort", "Failed to create port %s: %v", portName, err)
		return nil, err
	}

	record.Eventf(eventObject, "SuccessfulCreatePort", "Created port %s with id %s", port.Name, port.ID)
	return port, nil
}

func (s *Service) GetOrCreateTrunk(eventObject runtime.Object, clusterName, trunkName, portID string) (*trunks.Trunk, error) {
	allPages, err := trunks.List(s.client, trunks.ListOpts{
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
		Name:        trunkName,
		PortID:      portID,
		Description: names.GetDescription(clusterName),
	}
	mc := metrics.NewMetricPrometheusContext("trunk", "create")

	trunk, err := trunks.Create(s.client, trunkCreateOpts).Extract()
	if mc.ObserveRequest(err) != nil {
		record.Warnf(eventObject, "FailedCreateTrunk", "Failed to create trunk %s: %v", trunkName, err)
		return nil, err
	}

	record.Eventf(eventObject, "SuccessfulCreateTrunk", "Created trunk %s with id %s", trunk.Name, trunk.ID)
	return trunk, nil
}

func (s *Service) ReplaceAllAttributesTags(eventObject runtime.Object, trunkID string, tags []string) error {
	mc := metrics.NewMetricPrometheusContext("trunk", "update")
	_, err := attributestags.ReplaceAll(s.client, "trunks", trunkID, attributestags.ReplaceAllOpts{
		Tags: tags,
	}).Extract()
	if mc.ObserveRequest(err) != nil {
		record.Warnf(eventObject, "FailedReplaceAllAttributesTags", "Failed to replace all attributestags, trunk %s: %v", trunkID, err)
		return err
	}

	record.Eventf(eventObject, "SuccessfulReplaceAllAttributeTags", "Replaced all attributestags %s with tags %s", trunkID, tags)
	return nil
}

func (s *Service) DeletePort(eventObject runtime.Object, portID string) error {
	port, err := s.GetPort(portID)
	if err != nil {
		return err
	}
	if port == nil {
		return nil
	}

	err = util.PollImmediate(RetryIntervalPortDelete, TimeoutPortDelete, func() (bool, error) {
		mc := metrics.NewMetricPrometheusContext("port", "delete")
		err := ports.Delete(s.client, port.ID).ExtractErr()
		if mc.ObserveRequest(err) != nil {
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

func (s *Service) DeletePorts(eventObject runtime.Object, nets []servers.Network) error {
	for _, n := range nets {
		if err := s.DeletePort(eventObject, n.Port); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) DeleteTrunk(eventObject runtime.Object, portID string) error {
	port, err := s.GetPort(portID)
	if err != nil {
		return err
	}
	if port == nil {
		return nil
	}

	listOpts := trunks.ListOpts{
		PortID: port.ID,
	}
	trunkList, err := trunks.List(s.client, listOpts).AllPages()
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
		if err := trunks.Delete(s.client, trunkInfo[0].ID).ExtractErr(); err != nil {
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
