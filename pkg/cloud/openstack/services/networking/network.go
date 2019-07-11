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
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/pkg/errors"
	"k8s.io/klog"
	openstackconfigv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
)

func (s *Service) ReconcileNetwork(clusterName string, clusterProviderSpec *openstackconfigv1.OpenstackClusterProviderSpec, clusterProviderStatus *openstackconfigv1.OpenstackClusterProviderStatus) error {

	networkName := fmt.Sprintf("%s-cluster-%s", networkPrefix, clusterName)
	klog.Infof("Reconciling network %s", networkName)

	res, err := s.getNetworkByName(networkName)
	if err != nil {
		return err
	}

	if res.ID != "" {
		// Network exists
		clusterProviderStatus.Network = &openstackconfigv1.Network{
			ID:   res.ID,
			Name: res.Name,
		}
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

	_, err = attributestags.ReplaceAll(s.client, "networks", network.ID, attributestags.ReplaceAllOpts{
		Tags: []string{
			"cluster-api-provider-openstack",
			clusterName,
		}}).Extract()
	if err != nil {
		return err
	}

	clusterProviderStatus.Network = &openstackconfigv1.Network{
		ID:   network.ID,
		Name: network.Name,
	}
	return nil
}

func (s *Service) ReconcileSubnet(clusterName string, clusterProviderSpec *openstackconfigv1.OpenstackClusterProviderSpec, clusterProviderStatus *openstackconfigv1.OpenstackClusterProviderStatus) error {

	if clusterProviderStatus.Network == nil || clusterProviderStatus.Network.ID == "" {
		klog.V(4).Infof("No need to reconcile network components since no network exists.")
		return nil
	}

	subnetName := fmt.Sprintf("%s-cluster-%s", networkPrefix, clusterName)
	klog.Infof("Reconciling subnet %s", subnetName)

	allPages, err := subnets.List(s.client, subnets.ListOpts{
		NetworkID: clusterProviderStatus.Network.ID,
		CIDR:      clusterProviderSpec.NodeCIDR,
	}).AllPages()
	if err != nil {
		return err
	}

	subnetList, err := subnets.ExtractSubnets(allPages)
	if err != nil {
		return err
	}

	var observedSubnet openstackconfigv1.Subnet
	if len(subnetList) > 1 {
		// Not panicing here, because every other cluster might work.
		return fmt.Errorf("found more than 1 network with the expected name (%d) and CIDR (%s), which should not be able to exist in OpenStack", len(subnetList), clusterProviderSpec.NodeCIDR)
	} else if len(subnetList) == 0 {
		opts := subnets.CreateOpts{
			NetworkID: clusterProviderStatus.Network.ID,
			Name:      subnetName,
			IPVersion: 4,

			CIDR:           clusterProviderSpec.NodeCIDR,
			DNSNameservers: clusterProviderSpec.DNSNameservers,
		}

		newSubnet, err := subnets.Create(s.client, opts).Extract()
		if err != nil {
			return err
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
		return err
	}

	clusterProviderStatus.Network.Subnet = &observedSubnet
	return nil
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
