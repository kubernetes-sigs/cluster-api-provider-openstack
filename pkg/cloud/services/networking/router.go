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

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/pagination"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha2"
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
	if openStackCluster.Spec.ExternalNetworkID == "" {
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
				NetworkID: openStackCluster.Spec.ExternalNetworkID,
			}
		}
		newRouter, err := routers.Create(s.client, opts).Extract()
		if err != nil {
			record.Warnf(openStackCluster, "FailedCreateRouter", "Failed to create router %s: %v", routerName, err)
			return err
		}
		record.Eventf(openStackCluster, "SuccessfulCreateRouter", "Created router %s with id %s", routerName, newRouter.ID)
		router = *newRouter
	} else {
		router = routerList[0]
	}

	if len(openStackCluster.Spec.ExternalRouterIPs) > 0 {
		var updateOpts routers.UpdateOpts
		updateOpts.GatewayInfo = &routers.GatewayInfo{
			NetworkID: openStackCluster.Spec.ExternalNetworkID,
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

	observedRouter := infrav1.Router{
		Name: router.Name,
		ID:   router.ID,
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

	_, err = attributestags.ReplaceAll(s.client, "routers", observedRouter.ID, attributestags.ReplaceAllOpts{
		Tags: []string{
			"cluster-api-provider-openstack",
			clusterName,
		}}).Extract()
	if err != nil {
		return err
	}

	if observedRouter.ID != "" {
		openStackCluster.Status.Network.Router = &observedRouter
	}
	return nil
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
