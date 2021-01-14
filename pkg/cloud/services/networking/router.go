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
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/pagination"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha3"
)

func (s *Service) ReconcileRouter(clusterName string, openStackCluster *infrav1.OpenStackCluster) error {

	if openStackCluster.Status.Network == nil || openStackCluster.Status.Network.ID == "" {
		s.logger.V(3).Info("No need to reconcile router since no network exists.")
		return nil
	}
	if openStackCluster.Status.Network.Subnet == nil || openStackCluster.Status.Network.Subnet.ID == "" {
		s.logger.V(4).Info("No need to reconcile router since no subnet exists.")
		return nil
	}
	if openStackCluster.Status.ExternalNetwork == nil || openStackCluster.Status.ExternalNetwork.ID == "" {
		s.logger.V(3).Info("No need to create router, due to missing ExternalNetworkID.")
		return nil
	}

	routerName := fmt.Sprintf("%s-cluster-%s", networkPrefix, clusterName)
	s.logger.Info("Reconciling router", "name", routerName)

	allPages, err := routers.List(s.client, routers.ListOpts{
		Name: routerName,
	}).AllPages()
	if err != nil {
		return err
	}

	routerList, err := routers.ExtractRouters(allPages)
	if err != nil {
		return err
	}
	var router routers.Router
	var observedRouter infrav1.Router
	if len(routerList) == 0 {
		opts := routers.CreateOpts{
			Name: routerName,
		}
		// only set the GatewayInfo right now when no externalIPs
		// should be configured because at least in our environment
		// we can only set the routerIP via gateway update not during create
		// That's also the same way terraform provider OpenStack does it
		if len(openStackCluster.Spec.ExternalRouterIPs) == 0 {
			opts.GatewayInfo = &routers.GatewayInfo{
				NetworkID: openStackCluster.Status.ExternalNetwork.ID,
			}
		}
		newRouter, err := routers.Create(s.client, opts).Extract()
		if err != nil {
			record.Warnf(openStackCluster, "FailedCreateRouter", "Failed to create router %s: %v", routerName, err)
			return err
		}
		record.Eventf(openStackCluster, "SuccessfulCreateRouter", "Created router %s with id %s", routerName, newRouter.ID)
		if len(openStackCluster.Spec.Tags) > 0 {
			_, err = attributestags.ReplaceAll(s.client, "routers", newRouter.ID, attributestags.ReplaceAllOpts{
				Tags: openStackCluster.Spec.Tags}).Extract()
			if err != nil {
				return err
			}
		}
		router = *newRouter
		observedRouter = infrav1.Router{
			Name: router.Name,
			ID:   router.ID,
			Tags: openStackCluster.Spec.Tags,
		}

	} else {
		router = routerList[0]
		sInfo := fmt.Sprintf("Reuse Existing Router %s with id %s", routerName, router.ID)
		s.logger.V(6).Info(sInfo)
		observedRouter = infrav1.Router{
			Name: router.Name,
			ID:   router.ID,
			Tags: router.Tags,
		}

	}

	if len(openStackCluster.Spec.ExternalRouterIPs) > 0 {
		var updateOpts routers.UpdateOpts
		updateOpts.GatewayInfo = &routers.GatewayInfo{
			NetworkID: openStackCluster.Status.ExternalNetwork.ID,
		}
		for _, externalRouterIP := range openStackCluster.Spec.ExternalRouterIPs {
			subnetID := externalRouterIP.Subnet.UUID
			if subnetID == "" {
				sopts := subnets.ListOpts(externalRouterIP.Subnet.Filter)
				snets, err := GetSubnetsByFilter(s.client, &sopts)
				if err != nil {
					return err
				}
				if len(snets) != 1 {
					return fmt.Errorf("subnetParam didn't exactly match one subnet")
				}
				subnetID = snets[0].ID
			}
			updateOpts.GatewayInfo.ExternalFixedIPs = append(updateOpts.GatewayInfo.ExternalFixedIPs, routers.ExternalFixedIP{
				IPAddress: externalRouterIP.FixedIP,
				SubnetID:  subnetID,
			})
		}

		_, err = routers.Update(s.client, router.ID, updateOpts).Extract()
		if err != nil {
			record.Warnf(openStackCluster, "FailedUpdateRouter", "Failed to update router %s: %v", routerName, err)
			return fmt.Errorf("error updating OpenStack Neutron Router: %s", err)
		}
		record.Eventf(openStackCluster, "SuccessfulUpdateRouter", "Updated router %s with id %s", routerName, router.ID)
	}

	routerInterfaces, err := s.getRouterInterfaces(router.ID)
	if err != nil {
		return err
	}

	createInterface := true
	// check all router interfaces for an existing port in our subnet.
INTERFACE_LOOP:
	for _, iface := range routerInterfaces {
		for _, ip := range iface.FixedIPs {
			if ip.SubnetID == openStackCluster.Status.Network.Subnet.ID {
				createInterface = false
				break INTERFACE_LOOP
			}
		}
	}

	// ... and create a router interface for our subnet.
	if createInterface {
		s.logger.V(4).Info("Creating RouterInterface", "routerID", router.ID, "subnetID", openStackCluster.Status.Network.Subnet.ID)
		routerInterface, err := routers.AddInterface(s.client, router.ID, routers.AddInterfaceOpts{
			SubnetID: openStackCluster.Status.Network.Subnet.ID,
		}).Extract()
		if err != nil {
			return fmt.Errorf("unable to create router interface: %v", err)
		}
		s.logger.V(4).Info("Created RouterInterface", "id", routerInterface.ID)
	}

	if observedRouter.ID != "" {
		openStackCluster.Status.Network.Router = &observedRouter
	}
	return nil
}

func (s *Service) DeleteRouter(network *infrav1.Network) error {
	if network.Router == nil || network.Router.ID == "" {
		s.logger.V(4).Info("No need to delete router since no router exists.")
		return nil
	}
	exists, err := s.existsRouter(network.Router.ID)
	if err != nil {
		return err
	}
	if !exists {
		s.logger.Info("Skipping router deletion because router doesn't exist", "router", network.Router.ID)
		return nil
	}
	if network.Subnet == nil || network.Subnet.ID == "" {
		s.logger.V(4).Info("Skipping removing router interface since no subnet exists.")
	} else {
		_, err = routers.RemoveInterface(s.client, network.Router.ID, routers.RemoveInterfaceOpts{
			SubnetID: network.Subnet.ID,
		}).Extract()
		if err != nil {
			if errors.IsNotFound(err) {
				s.logger.V(4).Info("Router Interface already removed, no actions", "id", network.Router.ID)
				return nil
			}
			return fmt.Errorf("unable to remove router interface: %v", err)
		}
		s.logger.V(4).Info("Removed RouterInterface of Router", "id", network.Router.ID)
	}
	return routers.Delete(s.client, network.Router.ID).ExtractErr()
}

func (s *Service) getRouterInterfaces(routerID string) ([]ports.Port, error) {
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

func (s *Service) existsRouter(routerID string) (bool, error) {
	if routerID == "" {
		return false, nil
	}
	router, err := routers.Get(s.client, routerID).Extract()
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if router == nil {
		return false, nil
	}
	return true, nil
}
