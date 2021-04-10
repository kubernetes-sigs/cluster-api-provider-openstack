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

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/pkg/errors"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha4"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
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

func (s *Service) ReconcileNetwork(openStackCluster *infrav1.OpenStackCluster, clusterName string) error {
	networkName := getNetworkName(clusterName)
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
			Tags: openStackCluster.Spec.Tags,
		}).Extract()
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

func (s *Service) DeleteNetwork(openStackCluster *infrav1.OpenStackCluster, clusterName string) error {
	networkName := getNetworkName(clusterName)
	network, err := s.getNetworkByName(networkName)
	if err != nil {
		return err
	}
	if network.ID == "" {
		// nothing to do
		return nil
	}

	if err = networks.Delete(s.client, network.ID).ExtractErr(); err != nil {
		record.Warnf(openStackCluster, "FailedDeleteNetwork", "Failed to delete network %s with id %s: %v", network.Name, network.ID, err)
		return err
	}

	record.Eventf(openStackCluster, "SuccessfulDeleteNetwork", "Deleted network %s with id %s", network.Name, network.ID)
	return nil
}

func (s *Service) ReconcileSubnet(openStackCluster *infrav1.OpenStackCluster, clusterName string) error {
	if openStackCluster.Status.Network == nil || openStackCluster.Status.Network.ID == "" {
		s.logger.V(4).Info("No need to reconcile network components since no network exists.")
		return nil
	}

	subnetName := getSubnetName(clusterName)
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

	if len(subnetList) > 1 {
		return fmt.Errorf("found %d subnets with the name %s, which should not happen", len(subnetList), subnetName)
	}

	var subnet *subnets.Subnet
	if len(subnetList) == 0 {
		var err error
		subnet, err = createSubnet(s.client, openStackCluster, subnetName)
		if err != nil {
			return err
		}
	} else if len(subnetList) == 1 {
		subnet = &subnetList[0]
		s.logger.V(6).Info(fmt.Sprintf("Reuse existing subnet %s with id %s", subnetName, subnet.ID))
	}

	openStackCluster.Status.Network.Subnet = &infrav1.Subnet{
		ID:   subnet.ID,
		Name: subnet.Name,
		CIDR: subnet.CIDR,
		Tags: subnet.Tags,
	}
	return nil
}

func createSubnet(client *gophercloud.ServiceClient, openStackCluster *infrav1.OpenStackCluster, name string) (*subnets.Subnet, error) {
	opts := subnets.CreateOpts{
		NetworkID:      openStackCluster.Status.Network.ID,
		Name:           name,
		IPVersion:      4,
		CIDR:           openStackCluster.Spec.NodeCIDR,
		DNSNameservers: openStackCluster.Spec.DNSNameservers,
	}
	subnet, err := subnets.Create(client, opts).Extract()
	if err != nil {
		record.Warnf(openStackCluster, "FailedCreateSubnet", "Failed to create subnet %s: %v", name, err)
		return nil, err
	}
	record.Eventf(openStackCluster, "SuccessfulCreateSubnet", "Created subnet %s with id %s", name, subnet.ID)

	if len(openStackCluster.Spec.Tags) > 0 {
		_, err = attributestags.ReplaceAll(client, "subnets", subnet.ID, attributestags.ReplaceAllOpts{
			Tags: openStackCluster.Spec.Tags,
		}).Extract()
		if err != nil {
			return nil, err
		}
	}

	return subnet, nil
}

func (s *Service) getNetworkByID(networkID string) (networks.Network, error) {
	opts := networks.ListOpts{
		ID: networkID,
	}

	allPages, err := networks.List(s.client, opts).AllPages()
	if err != nil {
		return networks.Network{}, err
	}

	networkList, err := networks.ExtractNetworks(allPages)
	if err != nil {
		return networks.Network{}, err
	}

	switch len(networkList) {
	case 0:
		return networks.Network{}, nil
	case 1:
		return networkList[0], nil
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

	networkList, err := networks.ExtractNetworks(allPages)
	if err != nil {
		return networks.Network{}, err
	}

	switch len(networkList) {
	case 0:
		return networks.Network{}, nil
	case 1:
		return networkList[0], nil
	}
	return networks.Network{}, errors.New("too many resources")
}

func (s *Service) GetNetworksByFilter(opts networks.ListOptsBuilder) ([]networks.Network, error) {
	return GetNetworksByFilter(s.client, opts)
}

// getNetworkIDsByFilter retrieves network ids by querying openstack with filters.
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

// GetNetworksByFilter retrieves networks by querying openstack with filters.
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

// A function for getting the id of a subnet by querying openstack with filters.
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

func getSubnetName(clusterName string) string {
	return fmt.Sprintf("%s-cluster-%s", networkPrefix, clusterName)
}

func getNetworkName(clusterName string) string {
	return fmt.Sprintf("%s-cluster-%s", networkPrefix, clusterName)
}
