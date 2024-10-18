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

package osclients

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type NetworkClient interface {
	ListFloatingIP(opts floatingips.ListOptsBuilder) ([]floatingips.FloatingIP, error)
	CreateFloatingIP(opts floatingips.CreateOptsBuilder) (*floatingips.FloatingIP, error)
	DeleteFloatingIP(id string) error
	GetFloatingIP(id string) (*floatingips.FloatingIP, error)
	UpdateFloatingIP(id string, opts floatingips.UpdateOptsBuilder) (*floatingips.FloatingIP, error)

	ListPort(opts ports.ListOptsBuilder) ([]ports.Port, error)
	CreatePort(opts ports.CreateOptsBuilder) (*ports.Port, error)
	DeletePort(id string) error
	GetPort(id string) (*ports.Port, error)
	UpdatePort(id string, opts ports.UpdateOptsBuilder) (*ports.Port, error)

	ListTrunk(opts trunks.ListOptsBuilder) ([]trunks.Trunk, error)
	CreateTrunk(opts trunks.CreateOptsBuilder) (*trunks.Trunk, error)
	DeleteTrunk(id string) error

	ListTrunkSubports(trunkID string) ([]trunks.Subport, error)
	RemoveSubports(id string, opts trunks.RemoveSubportsOpts) error

	ListRouter(opts routers.ListOpts) ([]routers.Router, error)
	CreateRouter(opts routers.CreateOptsBuilder) (*routers.Router, error)
	DeleteRouter(id string) error
	GetRouter(id string) (*routers.Router, error)
	UpdateRouter(id string, opts routers.UpdateOptsBuilder) (*routers.Router, error)
	AddRouterInterface(id string, opts routers.AddInterfaceOptsBuilder) (*routers.InterfaceInfo, error)
	RemoveRouterInterface(id string, opts routers.RemoveInterfaceOptsBuilder) (*routers.InterfaceInfo, error)

	ListSecGroup(opts groups.ListOpts) ([]groups.SecGroup, error)
	CreateSecGroup(opts groups.CreateOptsBuilder) (*groups.SecGroup, error)
	DeleteSecGroup(id string) error
	GetSecGroup(id string) (*groups.SecGroup, error)
	UpdateSecGroup(id string, opts groups.UpdateOptsBuilder) (*groups.SecGroup, error)

	ListSecGroupRule(opts rules.ListOpts) ([]rules.SecGroupRule, error)
	CreateSecGroupRule(opts rules.CreateOptsBuilder) (*rules.SecGroupRule, error)
	DeleteSecGroupRule(id string) error
	GetSecGroupRule(id string) (*rules.SecGroupRule, error)

	ListNetwork(opts networks.ListOptsBuilder) ([]networks.Network, error)
	CreateNetwork(opts networks.CreateOptsBuilder) (*networks.Network, error)
	DeleteNetwork(id string) error
	GetNetwork(id string) (*networks.Network, error)
	UpdateNetwork(id string, opts networks.UpdateOptsBuilder) (*networks.Network, error)

	ListSubnet(opts subnets.ListOptsBuilder) ([]subnets.Subnet, error)
	CreateSubnet(opts subnets.CreateOptsBuilder) (*subnets.Subnet, error)
	DeleteSubnet(id string) error
	GetSubnet(id string) (*subnets.Subnet, error)
	UpdateSubnet(id string, opts subnets.UpdateOptsBuilder) (*subnets.Subnet, error)

	ListExtensions() ([]extensions.Extension, error)

	ReplaceAllAttributesTags(resourceType string, resourceID string, opts attributestags.ReplaceAllOptsBuilder) ([]string, error)
}

type networkClient struct {
	serviceClient *gophercloud.ServiceClient
}

// NewNetworkClient returns an instance of the networking service.
func NewNetworkClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (NetworkClient, error) {
	serviceClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create networking service providerClient: %v", err)
	}

	return networkClient{serviceClient}, nil
}

func (c networkClient) AddRouterInterface(id string, opts routers.AddInterfaceOptsBuilder) (*routers.InterfaceInfo, error) {
	return routers.AddInterface(context.TODO(), c.serviceClient, id, opts).Extract()
}

func (c networkClient) RemoveRouterInterface(id string, opts routers.RemoveInterfaceOptsBuilder) (*routers.InterfaceInfo, error) {
	return routers.RemoveInterface(context.TODO(), c.serviceClient, id, opts).Extract()
}

func (c networkClient) ReplaceAllAttributesTags(resourceType string, resourceID string, opts attributestags.ReplaceAllOptsBuilder) ([]string, error) {
	return attributestags.ReplaceAll(context.TODO(), c.serviceClient, resourceType, resourceID, opts).Extract()
}

func (c networkClient) ListRouter(opts routers.ListOpts) ([]routers.Router, error) {
	allPages, err := routers.List(c.serviceClient, opts).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	return routers.ExtractRouters(allPages)
}

func (c networkClient) ListFloatingIP(opts floatingips.ListOptsBuilder) ([]floatingips.FloatingIP, error) {
	allPages, err := floatingips.List(c.serviceClient, opts).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	return floatingips.ExtractFloatingIPs(allPages)
}

func (c networkClient) CreateFloatingIP(opts floatingips.CreateOptsBuilder) (*floatingips.FloatingIP, error) {
	fip, err := floatingips.Create(context.TODO(), c.serviceClient, opts).Extract()
	if err != nil {
		return nil, err
	}
	return fip, nil
}

func (c networkClient) DeleteFloatingIP(id string) error {
	return floatingips.Delete(context.TODO(), c.serviceClient, id).ExtractErr()
}

func (c networkClient) GetFloatingIP(id string) (*floatingips.FloatingIP, error) {
	return floatingips.Get(context.TODO(), c.serviceClient, id).Extract()
}

func (c networkClient) UpdateFloatingIP(id string, opts floatingips.UpdateOptsBuilder) (*floatingips.FloatingIP, error) {
	return floatingips.Update(context.TODO(), c.serviceClient, id, opts).Extract()
}

func (c networkClient) ListPort(opts ports.ListOptsBuilder) ([]ports.Port, error) {
	allPages, err := ports.List(c.serviceClient, opts).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	return ports.ExtractPorts(allPages)
}

func (c networkClient) CreatePort(opts ports.CreateOptsBuilder) (*ports.Port, error) {
	return ports.Create(context.TODO(), c.serviceClient, opts).Extract()
}

func (c networkClient) DeletePort(id string) error {
	return ports.Delete(context.TODO(), c.serviceClient, id).ExtractErr()
}

func (c networkClient) GetPort(id string) (*ports.Port, error) {
	return ports.Get(context.TODO(), c.serviceClient, id).Extract()
}

func (c networkClient) UpdatePort(id string, opts ports.UpdateOptsBuilder) (*ports.Port, error) {
	return ports.Update(context.TODO(), c.serviceClient, id, opts).Extract()
}

func (c networkClient) CreateTrunk(opts trunks.CreateOptsBuilder) (*trunks.Trunk, error) {
	return trunks.Create(context.TODO(), c.serviceClient, opts).Extract()
}

func (c networkClient) DeleteTrunk(id string) error {
	return trunks.Delete(context.TODO(), c.serviceClient, id).ExtractErr()
}

func (c networkClient) ListTrunkSubports(trunkID string) ([]trunks.Subport, error) {
	return trunks.GetSubports(context.TODO(), c.serviceClient, trunkID).Extract()
}

func (c networkClient) RemoveSubports(id string, opts trunks.RemoveSubportsOpts) error {
	_, err := trunks.RemoveSubports(context.TODO(), c.serviceClient, id, opts).Extract()
	return err
}

func (c networkClient) ListTrunk(opts trunks.ListOptsBuilder) ([]trunks.Trunk, error) {
	allPages, err := trunks.List(c.serviceClient, opts).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	return trunks.ExtractTrunks(allPages)
}

func (c networkClient) CreateRouter(opts routers.CreateOptsBuilder) (*routers.Router, error) {
	return routers.Create(context.TODO(), c.serviceClient, opts).Extract()
}

func (c networkClient) DeleteRouter(id string) error {
	return routers.Delete(context.TODO(), c.serviceClient, id).ExtractErr()
}

func (c networkClient) GetRouter(id string) (*routers.Router, error) {
	return routers.Get(context.TODO(), c.serviceClient, id).Extract()
}

func (c networkClient) UpdateRouter(id string, opts routers.UpdateOptsBuilder) (*routers.Router, error) {
	return routers.Update(context.TODO(), c.serviceClient, id, opts).Extract()
}

func (c networkClient) ListSecGroup(opts groups.ListOpts) ([]groups.SecGroup, error) {
	allPages, err := groups.List(c.serviceClient, opts).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	return groups.ExtractGroups(allPages)
}

func (c networkClient) CreateSecGroup(opts groups.CreateOptsBuilder) (*groups.SecGroup, error) {
	return groups.Create(context.TODO(), c.serviceClient, opts).Extract()
}

func (c networkClient) DeleteSecGroup(id string) error {
	return groups.Delete(context.TODO(), c.serviceClient, id).ExtractErr()
}

func (c networkClient) GetSecGroup(id string) (*groups.SecGroup, error) {
	return groups.Get(context.TODO(), c.serviceClient, id).Extract()
}

func (c networkClient) UpdateSecGroup(id string, opts groups.UpdateOptsBuilder) (*groups.SecGroup, error) {
	return groups.Update(context.TODO(), c.serviceClient, id, opts).Extract()
}

func (c networkClient) ListSecGroupRule(opts rules.ListOpts) ([]rules.SecGroupRule, error) {
	allPages, err := rules.List(c.serviceClient, opts).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	return rules.ExtractRules(allPages)
}

func (c networkClient) CreateSecGroupRule(opts rules.CreateOptsBuilder) (*rules.SecGroupRule, error) {
	return rules.Create(context.TODO(), c.serviceClient, opts).Extract()
}

func (c networkClient) DeleteSecGroupRule(id string) error {
	return rules.Delete(context.TODO(), c.serviceClient, id).ExtractErr()
}

func (c networkClient) GetSecGroupRule(id string) (*rules.SecGroupRule, error) {
	return rules.Get(context.TODO(), c.serviceClient, id).Extract()
}

func (c networkClient) ListNetwork(opts networks.ListOptsBuilder) ([]networks.Network, error) {
	allPages, err := networks.List(c.serviceClient, opts).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	return networks.ExtractNetworks(allPages)
}

func (c networkClient) CreateNetwork(opts networks.CreateOptsBuilder) (*networks.Network, error) {
	return networks.Create(context.TODO(), c.serviceClient, opts).Extract()
}

func (c networkClient) DeleteNetwork(id string) error {
	return networks.Delete(context.TODO(), c.serviceClient, id).ExtractErr()
}

func (c networkClient) GetNetwork(id string) (*networks.Network, error) {
	return networks.Get(context.TODO(), c.serviceClient, id).Extract()
}

func (c networkClient) UpdateNetwork(id string, opts networks.UpdateOptsBuilder) (*networks.Network, error) {
	return networks.Update(context.TODO(), c.serviceClient, id, opts).Extract()
}

func (c networkClient) ListSubnet(opts subnets.ListOptsBuilder) ([]subnets.Subnet, error) {
	allPages, err := subnets.List(c.serviceClient, opts).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	return subnets.ExtractSubnets(allPages)
}

func (c networkClient) CreateSubnet(opts subnets.CreateOptsBuilder) (*subnets.Subnet, error) {
	return subnets.Create(context.TODO(), c.serviceClient, opts).Extract()
}

func (c networkClient) DeleteSubnet(id string) error {
	return subnets.Delete(context.TODO(), c.serviceClient, id).ExtractErr()
}

func (c networkClient) GetSubnet(id string) (*subnets.Subnet, error) {
	return subnets.Get(context.TODO(), c.serviceClient, id).Extract()
}

func (c networkClient) UpdateSubnet(id string, opts subnets.UpdateOptsBuilder) (*subnets.Subnet, error) {
	return subnets.Update(context.TODO(), c.serviceClient, id, opts).Extract()
}

func (c networkClient) ListExtensions() ([]extensions.Extension, error) {
	allPages, err := extensions.List(c.serviceClient).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	return extensions.ExtractExtensions(allPages)
}
