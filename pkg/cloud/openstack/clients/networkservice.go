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
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"k8s.io/klog"
	openstackconfigv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
)

const (
	defaultDNS    string = "8.8.8.8"
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
	if err := s.reconcileNetwork(networkName, desired, status); err != nil {
		return err
	}

	if err := s.reconcileSubnets(networkName, desired, status); err != nil {
		return err
	}

	if err := s.reconcileRouter(networkName, desired, status); err != nil {
		return err
	}

	return nil
}

func (s *NetworkService) reconcileNetwork(networkName string, desired openstackconfigv1.OpenstackClusterProviderSpec, status *openstackconfigv1.OpenstackClusterProviderStatus) error {
	klog.Infof("Reconciling network %s", networkName)
	in := &status.Network
	res, err := s.getNetworkByName(networkName)
	if err != nil {
		return err
	}
	if res.ID != "" {
		// Network exists
		in.ID = res.ID
		in.Name = res.Name
		return nil
	}

	opts := networks.CreateOpts{
		AdminStateUp: gophercloud.Enabled,
		Name:         networkName,
	}
	network, err := networks.Create(s.client, opts).Extract()
	if err != nil {
		return err
	}

	in.ID = network.ID
	in.Name = network.Name

	return nil
}

func (s *NetworkService) reconcileSubnets(name string, desired openstackconfigv1.OpenstackClusterProviderSpec, status *openstackconfigv1.OpenstackClusterProviderStatus) error {
	klog.Infof("Reconciling subnet %s", name)
	in := &status.Network
	if in.ID == "" {
		klog.V(3).Info("No need to reconcile subnets. There is no network.")
		return nil
	}
	allPages, err := subnets.List(s.client, subnets.ListOpts{
		NetworkID: in.ID,
		CIDR:      desired.NodeCIDR,
	}).AllPages()
	if err != nil {
		return err
	}

	subnetList, err := subnets.ExtractSubnets(allPages)
	if err != nil {
		return err
	}

	v1Subnets := []openstackconfigv1.Subnet{}
	if len(subnetList) == 0 {
		opts := subnets.CreateOpts{
			NetworkID:      in.ID,
			Name:           name,
			IPVersion:      4,
			DNSNameservers: []string{defaultDNS},

			CIDR: desired.NodeCIDR,
		}

		newSubnet, err := subnets.Create(s.client, opts).Extract()
		if err != nil {
			return err
		}
		v1Subnets = append(v1Subnets, openstackconfigv1.Subnet{
			ID:   newSubnet.ID,
			Name: newSubnet.Name,

			CIDR: newSubnet.CIDR,
		})
	}
	for _, sn := range subnetList {
		v1Subnets = append(v1Subnets, openstackconfigv1.Subnet{
			ID:   sn.ID,
			Name: sn.Name,

			CIDR: sn.CIDR,
		})
	}
	in.Subnets = v1Subnets

	return nil
}

func (s *NetworkService) reconcileRouter(name string, desired openstackconfigv1.OpenstackClusterProviderSpec, status *openstackconfigv1.OpenstackClusterProviderStatus) error {
	klog.Infof("Reconciling router %s", name)
	in := &status.Network
	if in.ID == "" {
		klog.V(3).Info("No need to reconcile router. There is no network.")
		return nil
	}
	if len(in.Subnets) == 0 {
		klog.V(3).Info("No need to reconcile router. There are no subnets.")
		return nil
	}
	if desired.ExternalNetworkID == "" {
		return errors.New("unable to create router, due to missing ExternalNetworkID")
	}

	allPages, err := routers.List(s.client, routers.ListOpts{
		Name: name,
	}).AllPages()
	if err != nil {
		return err
	}

	routerList, err := routers.ExtractRouters(allPages)
	if err != nil {
		return err
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
			return err
		}
		router = *newRouter
	} else {
		router = routerList[0]
	}

	in.Router = &openstackconfigv1.Router{
		Name: router.Name,
		ID:   router.ID,
	}

	routerInterfaces, err := s.getRouterInterfaces(router.ID)
	if err != nil {
		return err
	}

	// Get all subnets for our network...
	availableSubnets := make(map[string]openstackconfigv1.Subnet, len(in.Subnets))
	for _, net := range in.Subnets {
		availableSubnets[net.ID] = net
	}

	// ... and filter out all subnets, the router already has an interface in...
	for _, iface := range routerInterfaces {
		for _, ip := range iface.FixedIPs {
			if _, ok := availableSubnets[ip.SubnetID]; ok {
				delete(availableSubnets, ip.SubnetID)
			}
		}
	}

	// ... and create router interfaces for the remaining subnets.
	for _, net := range availableSubnets {
		klog.V(4).Infof("Creating RouterInterface on %s in subnet %s", router.ID, net.ID)
		iface, err := routers.AddInterface(s.client, router.ID, routers.AddInterfaceOpts{
			SubnetID: net.ID,
		}).Extract()
		if err != nil {
			return fmt.Errorf("unable to create router interface: %v", err)
		}
		klog.V(4).Infof("Created RouterInterface: %v", iface)
	}

	return nil
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
