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

package clients

import (
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"k8s.io/klog/v2"
	openstackconfigv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
)

const (
	networkPrefix string = "k8s"
)

// NetworkService interfaces with the OpenStack Networking API.
// It will create a network related infrastructure for the cluster, like network, subnet, router.
type NetworkService struct {
	client *gophercloud.ServiceClient
}

// NewNetworkService returns an instance for the OpenStack Networking API
func NewNetworkService(client *gophercloud.ServiceClient) (*NetworkService, error) {
	return &NetworkService{
		client: client,
	}, nil
}

// Reconcile the Network for a given cluster
func (s *NetworkService) Reconcile(clusterName string, desired openstackconfigv1.OpenstackClusterProviderSpec, status *openstackconfigv1.OpenstackClusterProviderStatus) error {
	klog.Infof("Reconciling network components for cluster %s", clusterName)
	if desired.NodeCIDR == "" {
		klog.V(4).Infof("No need to reconcile network for cluster %s", clusterName)
		return nil
	}
	networkName := fmt.Sprintf("%s-cluster-%s", networkPrefix, clusterName)
	network, err := s.reconcileNetwork(clusterName, networkName, desired)
	if err != nil {
		return err
	}
	if network.ID == "" {
		klog.V(4).Infof("No need to reconcile network components since no network exists.")
		status.Network = nil
		return nil
	}
	status.Network = &network

	observedSubnet, err := s.reconcileSubnets(clusterName, networkName, desired, network)
	if err != nil {
		return err
	}
	if observedSubnet.ID == "" {
		klog.V(4).Infof("No need to reconcile further network components since no subnet exists.")
		status.Network.Subnet = nil
		return nil
	}
	network.Subnet = &observedSubnet

	observerdRouter, err := s.reconcileRouter(clusterName, networkName, desired, network)
	if err != nil {
		return err
	}
	if observerdRouter.ID != "" {
		// Only appending the router if it has an actual id
		network.Router = &observerdRouter
	} else {
		status.Network.Router = nil
	}

	return nil
}

func (s *NetworkService) reconcileNetwork(clusterName, networkName string, desired openstackconfigv1.OpenstackClusterProviderSpec) (openstackconfigv1.Network, error) {
	klog.Infof("Reconciling network %s", networkName)
	emptyNetwork := openstackconfigv1.Network{}
	res, err := s.getNetworkByName(networkName)
	if err != nil {
		return emptyNetwork, err
	}

	if res.ID != "" {
		// Network exists
		return openstackconfigv1.Network{
			ID:   res.ID,
			Name: res.Name,
		}, nil
	}

	opts := networks.CreateOpts{
		AdminStateUp: gophercloud.Enabled,
		Name:         networkName,
	}
	network, err := networks.Create(s.client, opts).Extract()
	if err != nil {
		return emptyNetwork, err
	}

	_, err = attributestags.ReplaceAll(s.client, "networks", network.ID, attributestags.ReplaceAllOpts{
		Tags: []string{
			"cluster-api-provider-openstack",
			clusterName,
		}}).Extract()
	if err != nil {
		return emptyNetwork, err
	}

	return openstackconfigv1.Network{
		ID:   network.ID,
		Name: network.Name,
	}, nil
}

func (s *NetworkService) reconcileSubnets(clusterName, name string, desired openstackconfigv1.OpenstackClusterProviderSpec, network openstackconfigv1.Network) (openstackconfigv1.Subnet, error) {
	klog.Infof("Reconciling subnet %s", name)
	emptySubnet := openstackconfigv1.Subnet{}
	allPages, err := subnets.List(s.client, subnets.ListOpts{
		NetworkID: network.ID,
		CIDR:      desired.NodeCIDR,
	}).AllPages()
	if err != nil {
		return emptySubnet, err
	}

	subnetList, err := subnets.ExtractSubnets(allPages)
	if err != nil {
		return emptySubnet, err
	}

	var observedSubnet openstackconfigv1.Subnet
	if len(subnetList) > 1 {
		// Not panicing here, because every other cluster might work.
		return emptySubnet, fmt.Errorf("found more than 1 network with the expected name (%d) and CIDR (%s), which should not be able to exist in OpenStack", len(subnetList), desired.NodeCIDR)
	} else if len(subnetList) == 0 {
		opts := subnets.CreateOpts{
			NetworkID: network.ID,
			Name:      name,
			IPVersion: 4,

			CIDR:           desired.NodeCIDR,
			DNSNameservers: desired.DNSNameservers,
		}

		newSubnet, err := subnets.Create(s.client, opts).Extract()
		if err != nil {
			return emptySubnet, err
		}
		observedSubnet = openstackconfigv1.Subnet{
			ID:   newSubnet.ID,
			Name: newSubnet.Name,

			CIDR: newSubnet.CIDR,
		}
	} else if len(subnetList) == 1 {
		observedSubnet = openstackconfigv1.Subnet{
			ID:   subnetList[0].ID,
			Name: subnetList[0].Name,

			CIDR: subnetList[0].CIDR,
		}
	}

	_, err = attributestags.ReplaceAll(s.client, "subnets", observedSubnet.ID, attributestags.ReplaceAllOpts{
		Tags: []string{
			"cluster-api-provider-openstack",
			clusterName,
		}}).Extract()
	if err != nil {
		return emptySubnet, err
	}

	return observedSubnet, nil
}

func (s *NetworkService) reconcileRouter(clusterName, name string, desired openstackconfigv1.OpenstackClusterProviderSpec, network openstackconfigv1.Network) (openstackconfigv1.Router, error) {
	klog.Infof("Reconciling router %s", name)
	emptyRouter := openstackconfigv1.Router{}
	if network.ID == "" {
		klog.V(3).Info("No need to reconcile router. There is no network.")
		return emptyRouter, nil
	}
	if network.Subnet == nil {
		klog.V(3).Info("No need to reconcile router. There is no subnet.")
		return emptyRouter, nil
	}
	if desired.ExternalNetworkID == "" {
		klog.V(3).Info("No need to create router, due to missing ExternalNetworkID")
		return emptyRouter, nil
	}

	allPages, err := routers.List(s.client, routers.ListOpts{
		Name: name,
	}).AllPages()
	if err != nil {
		return emptyRouter, err
	}

	routerList, err := routers.ExtractRouters(allPages)
	if err != nil {
		return emptyRouter, err
	}
	var router routers.Router
	if len(routerList) == 0 {
		opts := routers.CreateOpts{
			Name: name,
			GatewayInfo: &routers.GatewayInfo{
				NetworkID: desired.ExternalNetworkID,
			},
		}
		newRouter, err := routers.Create(s.client, opts).Extract()
		if err != nil {
			return emptyRouter, err
		}
		router = *newRouter
	} else {
		router = routerList[0]
	}

	observedRouter := openstackconfigv1.Router{
		Name: router.Name,
		ID:   router.ID,
	}

	routerInterfaces, err := s.getRouterInterfaces(router.ID)
	if err != nil {
		return emptyRouter, err
	}

	createInterface := true
	// check all router interfaces for an existing port in our subnet.
INTERFACE_LOOP:
	for _, iface := range routerInterfaces {
		for _, ip := range iface.FixedIPs {
			if ip.SubnetID == network.Subnet.ID {
				createInterface = false
				break INTERFACE_LOOP
			}
		}
	}

	// ... and create a router interface for our subnet.
	if createInterface {
		klog.V(4).Infof("Creating RouterInterface on %s in subnet %s", router.ID, network.Subnet.ID)
		iface, err := routers.AddInterface(s.client, router.ID, routers.AddInterfaceOpts{
			SubnetID: network.Subnet.ID,
		}).Extract()
		if err != nil {
			return observedRouter, fmt.Errorf("unable to create router interface: %v", err)
		}
		klog.V(4).Infof("Created RouterInterface: %v", iface)
	}

	_, err = attributestags.ReplaceAll(s.client, "routers", observedRouter.ID, attributestags.ReplaceAllOpts{
		Tags: []string{
			"cluster-api-provider-openstack",
			clusterName,
		}}).Extract()
	if err != nil {
		return emptyRouter, err
	}

	return observedRouter, nil
}

func (s *NetworkService) getRouterInterfaces(routerID string) ([]ports.Port, error) {
	allPages, err := ports.List(s.client, ports.ListOpts{
		DeviceID: routerID,
	}).AllPages()
	if err != nil {
		return []ports.Port{}, err
	}

	portList, err := ports.ExtractPorts(allPages)
	if err != nil {
		return []ports.Port{}, err
	}

	return portList, nil
}

func (s *NetworkService) getNetworkByName(networkName string) (networks.Network, error) {
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
