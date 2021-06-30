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

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha4"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/metrics"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/names"
)

func (s *Service) ReconcileRouter(openStackCluster *infrav1.OpenStackCluster, clusterName string) error {
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

	routerName := getRouterName(clusterName)
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

	if len(routerList) > 1 {
		return fmt.Errorf("found %d router with the name %s, which should not happen", len(routerList), routerName)
	}

	var router *routers.Router
	if len(routerList) == 0 {
		var err error
		router, err = s.createRouter(openStackCluster, clusterName, routerName)
		if err != nil {
			return err
		}
	} else {
		router = &routerList[0]
		s.logger.V(6).Info(fmt.Sprintf("Reuse existing Router %s with id %s", routerName, router.ID))
	}

	openStackCluster.Status.Network.Router = &infrav1.Router{
		Name: router.Name,
		ID:   router.ID,
		Tags: router.Tags,
	}

	if len(openStackCluster.Spec.ExternalRouterIPs) > 0 {
		if err := s.setRouterExternalIPs(openStackCluster, router); err != nil {
			return err
		}
	}

	routerInterfaces, err := s.getRouterInterfaces(router.ID)
	if err != nil {
		return err
	}

	//nolint:ifshort
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
		mc := metrics.NewMetricPrometheusContext("router_interface", "create")
		routerInterface, err := routers.AddInterface(s.client, router.ID, routers.AddInterfaceOpts{
			SubnetID: openStackCluster.Status.Network.Subnet.ID,
		}).Extract()
		if mc.ObserveRequest(err) != nil {
			return fmt.Errorf("unable to create router interface: %v", err)
		}
		s.logger.V(4).Info("Created RouterInterface", "id", routerInterface.ID)
	}
	return nil
}

func (s *Service) createRouter(openStackCluster *infrav1.OpenStackCluster, clusterName, name string) (*routers.Router, error) {
	opts := routers.CreateOpts{
		Description: names.GetDescription(clusterName),
		Name:        name,
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

	mc := metrics.NewMetricPrometheusContext("router", "create")

	router, err := routers.Create(s.client, opts).Extract()

	if mc.ObserveRequest(err) != nil {
		record.Warnf(openStackCluster, "FailedCreateRouter", "Failed to create router %s: %v", name, err)
		return nil, err
	}
	record.Eventf(openStackCluster, "SuccessfulCreateRouter", "Created router %s with id %s", name, router.ID)

	if len(openStackCluster.Spec.Tags) > 0 {
		_, err = attributestags.ReplaceAll(s.client, "routers", router.ID, attributestags.ReplaceAllOpts{
			Tags: openStackCluster.Spec.Tags,
		}).Extract()
		if err != nil {
			return nil, err
		}
	}

	return router, nil
}

func (s *Service) setRouterExternalIPs(openStackCluster *infrav1.OpenStackCluster, router *routers.Router) error {
	updateOpts := routers.UpdateOpts{
		GatewayInfo: &routers.GatewayInfo{
			NetworkID: openStackCluster.Status.ExternalNetwork.ID,
		},
	}

	for _, externalRouterIP := range openStackCluster.Spec.ExternalRouterIPs {
		subnetID := externalRouterIP.Subnet.UUID
		if subnetID == "" {
			listOpts := subnets.ListOpts(externalRouterIP.Subnet.Filter)
			subnetsByFilter, err := s.GetSubnetsByFilter(&listOpts)
			if err != nil {
				return err
			}
			if len(subnetsByFilter) != 1 {
				return fmt.Errorf("subnetParam didn't exactly match one subnet")
			}
			subnetID = subnetsByFilter[0].ID
		}
		updateOpts.GatewayInfo.ExternalFixedIPs = append(updateOpts.GatewayInfo.ExternalFixedIPs, routers.ExternalFixedIP{
			IPAddress: externalRouterIP.FixedIP,
			SubnetID:  subnetID,
		})
	}

	mc := metrics.NewMetricPrometheusContext("router", "update")
	_, err := routers.Update(s.client, router.ID, updateOpts).Extract()
	if mc.ObserveRequest(err) != nil {
		record.Warnf(openStackCluster, "FailedUpdateRouter", "Failed to update router %s with id %s: %v", router.Name, router.ID, err)
		return err
	}

	record.Eventf(openStackCluster, "SuccessfulUpdateRouter", "Updated router %s with id %s", router.Name, router.ID)
	return nil
}

func (s *Service) DeleteRouter(openStackCluster *infrav1.OpenStackCluster, clusterName string) error {
	router, subnet, err := s.getRouter(clusterName)
	if err != nil {
		return err
	}

	if router.ID == "" {
		return nil
	}

	if subnet.ID != "" {
		mc := metrics.NewMetricPrometheusContext("router_interface", "delete")
		_, err = routers.RemoveInterface(s.client, router.ID, routers.RemoveInterfaceOpts{
			SubnetID: subnet.ID,
		}).Extract()
		if mc.ObserveRequest(err) != nil {
			if !capoerrors.IsNotFound(err) {
				return fmt.Errorf("unable to remove router interface: %v", err)
			}
			s.logger.V(4).Info("Router Interface already removed, nothing to do", "id", router.ID)
		} else {
			s.logger.V(4).Info("Removed RouterInterface of Router", "id", router.ID)
		}
	}

	mc := metrics.NewMetricPrometheusContext("router", "delete")
	err = routers.Delete(s.client, router.ID).ExtractErr()
	if mc.ObserveRequest(err) != nil {
		record.Warnf(openStackCluster, "FailedDeleteRouter", "Failed to delete router %s with id %s: %v", router.Name, router.ID, err)
		return err
	}

	record.Eventf(openStackCluster, "SuccessfulDeleteRouter", "Deleted router %s with id %s", router.Name, router.ID)
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

func (s *Service) getRouter(clusterName string) (routers.Router, subnets.Subnet, error) {
	routerName := getRouterName(clusterName)
	router, err := s.getRouterByName(routerName)
	if err != nil {
		return routers.Router{}, subnets.Subnet{}, err
	}

	subnetName := getSubnetName(clusterName)
	subnet, err := s.getSubnetByName(subnetName)
	if err != nil {
		return router, subnets.Subnet{}, err
	}

	return router, subnet, nil
}

func (s *Service) getRouterByName(routerName string) (routers.Router, error) {
	allPages, err := routers.List(s.client, routers.ListOpts{
		Name: routerName,
	}).AllPages()
	if err != nil {
		return routers.Router{}, err
	}

	routerList, err := routers.ExtractRouters(allPages)
	if err != nil {
		return routers.Router{}, err
	}

	switch len(routerList) {
	case 0:
		return routers.Router{}, nil
	case 1:
		return routerList[0], nil
	}
	return routers.Router{}, fmt.Errorf("found %d router with the name %s, which should not happen", len(routerList), routerName)
}

func (s *Service) getSubnetByName(subnetName string) (subnets.Subnet, error) {
	opts := subnets.ListOpts{
		Name: subnetName,
	}

	allPages, err := subnets.List(s.client, opts).AllPages()
	if err != nil {
		return subnets.Subnet{}, err
	}

	subnetList, err := subnets.ExtractSubnets(allPages)
	if err != nil {
		return subnets.Subnet{}, err
	}

	switch len(subnetList) {
	case 0:
		return subnets.Subnet{}, nil
	case 1:
		return subnetList[0], nil
	}
	return subnets.Subnet{}, fmt.Errorf("found %d subnets with the name %s, which should not happen", len(subnetList), subnetName)
}

func getRouterName(clusterName string) string {
	return fmt.Sprintf("%s-cluster-%s", networkPrefix, clusterName)
}
