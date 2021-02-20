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

package networking

import (
	"fmt"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/pkg/errors"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha4"
)

type createOpts struct {
	AdminStateUp        *bool  `json:"admin_state_up,omitempty"`
	Name                string `json:"name,omitempty"`
	PortSecurityEnabled *bool  `json:"port_security_enabled,omitempty"`
}

func (c createOpts) ToNetworkCreateMap() (map[string]interface{}, error) {
	return gophercloud.BuildRequestBody(c, "network")
}

func (s *Service) ReconcileExternalNetwork(openStackCluster *infrav1.OpenStackCluster) error {

	if openStackCluster.Spec.ExternalNetworkID != "" {
		externalNetwork, err := s.getNetworkByID(openStackCluster.Spec.ExternalNetworkID)
		if err != nil {
			return err
		}
		if externalNetwork.ID != "" {
			openStackCluster.Status.ExternalNetwork = &infrav1.Network{
				ID:   externalNetwork.ID,
				Name: externalNetwork.Name,
				Tags: externalNetwork.Tags,
			}
			return nil
		}
	}

	// ExternalNetworkID is not given
	iTrue := true
	networkListOpts := networks.ListOpts{}
	listOpts := external.ListOptsExt{
		ListOptsBuilder: networkListOpts,
		External:        &iTrue,
	}

	allPages, err := networks.List(s.client, listOpts).AllPages()
	if err != nil {
		return err
	}
	allNetworks, err := networks.ExtractNetworks(allPages)
	if err != nil {
		return err
	}

	switch len(allNetworks) {
	case 0:
		return fmt.Errorf("external network not found")
	case 1:
		openStackCluster.Status.ExternalNetwork = &infrav1.Network{
			ID:   allNetworks[0].ID,
			Name: allNetworks[0].Name,
			Tags: allNetworks[0].Tags,
		}
		s.logger.Info("External network found", "network id", allNetworks[0].ID)
		return nil
	}
	return errors.New("too many resources")
}

func (s *Service) ReconcileNetwork(clusterName string, openStackCluster *infrav1.OpenStackCluster) error {

	networkName := fmt.Sprintf("%s-cluster-%s", networkPrefix, clusterName)
	s.logger.Info("Reconciling network", "name", networkName)

	res, err := s.getNetworkByName(networkName)
	if err != nil {
		return err
	}

	if res.ID != "" {
		// Network exists
		openStackCluster.Status.Network = &infrav1.Network{
			ID:   res.ID,
			Name: res.Name,
			Tags: res.Tags,
		}
		sInfo := fmt.Sprintf("Reuse Existing Network %s with id %s", res.Name, res.ID)
		s.logger.V(6).Info(sInfo)
		return nil
	}

	var opts createOpts
	if openStackCluster.Spec.DisablePortSecurity {
		opts = createOpts{
			AdminStateUp:        gophercloud.Enabled,
			Name:                networkName,
			PortSecurityEnabled: gophercloud.Disabled,
		}
	} else {
		opts = createOpts{
			AdminStateUp: gophercloud.Enabled,
			Name:         networkName,
		}
	}
	network, err := networks.Create(s.client, opts).Extract()
	if err != nil {
		record.Warnf(openStackCluster, "FailedCreateNetwork", "Failed to create network %s: %v", networkName, err)
		return err
	}
	record.Eventf(openStackCluster, "SuccessfulCreateNetwork", "Created network %s with id %s", networkName, network.ID)

	if len(openStackCluster.Spec.Tags) > 0 {
		_, err = attributestags.ReplaceAll(s.client, "networks", network.ID, attributestags.ReplaceAllOpts{
			Tags: openStackCluster.Spec.Tags}).Extract()
		if err != nil {
			return err
		}
	}

	openStackCluster.Status.Network = &infrav1.Network{
		ID:   network.ID,
		Name: network.Name,
		Tags: openStackCluster.Spec.Tags,
	}
	return nil
}

func (s *Service) DeleteNetwork(network *infrav1.Network) error {
	if network == nil || network.ID == "" {
		s.logger.V(4).Info("No need to delete network since no network exists.")
		return nil
	}
	exists, err := s.existsNetwork(network.ID)
	if err != nil {
		return err
	}
	if !exists {
		s.logger.Info("Skipping network deletion because network doesn't exist", "network", network.ID)
		return nil
	}
	return networks.Delete(s.client, network.ID).ExtractErr()
}

func (s *Service) ReconcileSubnet(clusterName string, openStackCluster *infrav1.OpenStackCluster) error {

	if openStackCluster.Status.Network == nil || openStackCluster.Status.Network.ID == "" {
		s.logger.V(4).Info("No need to reconcile network components since no network exists.")
		return nil
	}

	subnetName := fmt.Sprintf("%s-cluster-%s", networkPrefix, clusterName)
	s.logger.Info("Reconciling subnet", "name", subnetName)

	allPages, err := subnets.List(s.client, subnets.ListOpts{
		NetworkID: openStackCluster.Status.Network.ID,
		CIDR:      openStackCluster.Spec.NodeCIDR,
	}).AllPages()
	if err != nil {
		return err
	}

	subnetList, err := subnets.ExtractSubnets(allPages)
	if err != nil {
		return err
	}

	var observedSubnet infrav1.Subnet

	if len(subnetList) > 1 {
		// Not panicing here, because every other cluster might work.
		return fmt.Errorf("found more than 1 network with the expected name (%d) and CIDR (%s), which should not be able to exist in OpenStack", len(subnetList), openStackCluster.Spec.NodeCIDR)
	}

	if len(subnetList) == 0 {
		opts := subnets.CreateOpts{
			NetworkID: openStackCluster.Status.Network.ID,
			Name:      subnetName,
			IPVersion: 4,

			CIDR:           openStackCluster.Spec.NodeCIDR,
			DNSNameservers: openStackCluster.Spec.DNSNameservers,
		}
		newSubnet, err := subnets.Create(s.client, opts).Extract()
		if err != nil {
			record.Warnf(openStackCluster, "FailedCreateSubnet", "Failed to create subnet %s: %v", subnetName, err)
			return err
		}
		record.Eventf(openStackCluster, "SuccessfulCreateSubnet", "Created subnet %s with id %s", subnetName, newSubnet.ID)

		if len(openStackCluster.Spec.Tags) > 0 {
			_, err = attributestags.ReplaceAll(s.client, "subnets", newSubnet.ID, attributestags.ReplaceAllOpts{
				Tags: openStackCluster.Spec.Tags}).Extract()
			if err != nil {
				return err
			}
		}

		observedSubnet = infrav1.Subnet{
			ID:   newSubnet.ID,
			Name: newSubnet.Name,
			CIDR: newSubnet.CIDR,
			Tags: openStackCluster.Spec.Tags,
		}
	} else if len(subnetList) == 1 {
		observedSubnet = infrav1.Subnet{
			ID:   subnetList[0].ID,
			Name: subnetList[0].Name,
			CIDR: subnetList[0].CIDR,
			Tags: subnetList[0].Tags,
		}
	}

	openStackCluster.Status.Network.Subnet = &observedSubnet
	return nil
}

func (s *Service) getNetworkByID(networkID string) (networks.Network, error) {
	opts := networks.ListOpts{
		ID: networkID,
	}

	allPages, err := networks.List(s.client, opts).AllPages()
	if err != nil {
		return networks.Network{}, err
	}

	allNetworks, err := networks.ExtractNetworks(allPages)
	if err != nil {
		return networks.Network{}, err
	}

	switch len(allNetworks) {
	case 0:
		return networks.Network{}, nil
	case 1:
		return allNetworks[0], nil
	}
	return networks.Network{}, errors.New("multiple external network found")
}

func (s *Service) getNetworkByName(networkName string) (networks.Network, error) {
	opts := networks.ListOpts{
		Name: networkName,
	}

	allPages, err := networks.List(s.client, opts).AllPages()
	if err != nil {
		return networks.Network{}, err
	}

	allNetworks, err := networks.ExtractNetworks(allPages)
	if err != nil {
		return networks.Network{}, err
	}

	switch len(allNetworks) {
	case 0:
		return networks.Network{}, nil
	case 1:
		return allNetworks[0], nil
	}
	return networks.Network{}, errors.New("too many resources")
}

func (s *Service) existsNetwork(networkID string) (bool, error) {
	if networkID == "" {
		return false, nil
	}
	network, err := networks.Get(s.client, networkID).Extract()
	if err != nil {
		if capoerrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if network == nil {
		return false, nil
	}
	return true, nil
}

func (s *Service) GetNetworksByFilter(opts networks.ListOptsBuilder) ([]networks.Network, error) {
	return GetNetworksByFilter(s.client, opts)
}

// getNetworkIDsByFilter retrieves network ids by querying openstack with filters
func GetNetworkIDsByFilter(networkClient *gophercloud.ServiceClient, opts networks.ListOptsBuilder) ([]string, error) {
	nets, err := GetNetworksByFilter(networkClient, opts)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for _, network := range nets {
		ids = append(ids, network.ID)
	}
	return ids, nil
}

// GetNetworksByFilter retrieves networks by querying openstack with filters
func GetNetworksByFilter(networkClient *gophercloud.ServiceClient, opts networks.ListOptsBuilder) ([]networks.Network, error) {
	if opts == nil {
		return nil, fmt.Errorf("no Filters were passed")
	}
	pager := networks.List(networkClient, opts)
	var nets []networks.Network
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		networkList, err := networks.ExtractNetworks(page)
		if err != nil {
			return false, err
		} else if len(networkList) == 0 {
			return false, fmt.Errorf("no networks could be found with the filters provided")
		}
		nets = networkList
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return nets, nil
}

func (s *Service) GetSubnetsByFilter(opts subnets.ListOptsBuilder) ([]subnets.Subnet, error) {
	return GetSubnetsByFilter(s.client, opts)
}

// A function for getting the id of a subnet by querying openstack with filters
func GetSubnetsByFilter(networkClient *gophercloud.ServiceClient, opts subnets.ListOptsBuilder) ([]subnets.Subnet, error) {
	if opts == nil {
		return []subnets.Subnet{}, fmt.Errorf("no Filters were passed")
	}
	pager := subnets.List(networkClient, opts)
	var snets []subnets.Subnet
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		subnetList, err := subnets.ExtractSubnets(page)
		if err != nil {
			return false, err
		} else if len(subnetList) == 0 {
			return false, fmt.Errorf("no subnets could be found with the filters provided")
		}
		snets = append(snets, subnetList...)
		return true, nil
	})
	if err != nil {
		return []subnets.Subnet{}, err
	}
	return snets, nil
}
