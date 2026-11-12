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

package clients

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

	"sigs.k8s.io/cluster-api-provider-openstack/pkg/metrics"
)

type NetworkClient interface {
	ListFloatingIP(ctx context.Context, opts floatingips.ListOptsBuilder) ([]floatingips.FloatingIP, error)
	CreateFloatingIP(ctx context.Context, opts floatingips.CreateOptsBuilder) (*floatingips.FloatingIP, error)
	DeleteFloatingIP(ctx context.Context, id string) error
	GetFloatingIP(ctx context.Context, id string) (*floatingips.FloatingIP, error)
	UpdateFloatingIP(ctx context.Context, id string, opts floatingips.UpdateOptsBuilder) (*floatingips.FloatingIP, error)

	ListPort(ctx context.Context, opts ports.ListOptsBuilder) ([]ports.Port, error)
	CreatePort(ctx context.Context, opts ports.CreateOptsBuilder) (*ports.Port, error)
	DeletePort(ctx context.Context, id string) error
	GetPort(ctx context.Context, id string) (*ports.Port, error)
	UpdatePort(ctx context.Context, id string, opts ports.UpdateOptsBuilder) (*ports.Port, error)

	ListTrunk(ctx context.Context, opts trunks.ListOptsBuilder) ([]trunks.Trunk, error)
	CreateTrunk(ctx context.Context, opts trunks.CreateOptsBuilder) (*trunks.Trunk, error)
	DeleteTrunk(ctx context.Context, id string) error

	ListTrunkSubports(ctx context.Context, trunkID string) ([]trunks.Subport, error)
	RemoveSubports(ctx context.Context, id string, opts trunks.RemoveSubportsOpts) error

	ListRouter(ctx context.Context, opts routers.ListOpts) ([]routers.Router, error)
	CreateRouter(ctx context.Context, opts routers.CreateOptsBuilder) (*routers.Router, error)
	DeleteRouter(ctx context.Context, id string) error
	GetRouter(ctx context.Context, id string) (*routers.Router, error)
	UpdateRouter(ctx context.Context, id string, opts routers.UpdateOptsBuilder) (*routers.Router, error)
	AddRouterInterface(ctx context.Context, id string, opts routers.AddInterfaceOptsBuilder) (*routers.InterfaceInfo, error)
	RemoveRouterInterface(ctx context.Context, id string, opts routers.RemoveInterfaceOptsBuilder) (*routers.InterfaceInfo, error)

	ListSecGroup(ctx context.Context, opts groups.ListOpts) ([]groups.SecGroup, error)
	CreateSecGroup(ctx context.Context, opts groups.CreateOptsBuilder) (*groups.SecGroup, error)
	DeleteSecGroup(ctx context.Context, id string) error
	GetSecGroup(ctx context.Context, id string) (*groups.SecGroup, error)
	UpdateSecGroup(ctx context.Context, id string, opts groups.UpdateOptsBuilder) (*groups.SecGroup, error)

	ListSecGroupRule(ctx context.Context, opts rules.ListOpts) ([]rules.SecGroupRule, error)
	CreateSecGroupRule(ctx context.Context, opts rules.CreateOptsBuilder) (*rules.SecGroupRule, error)
	DeleteSecGroupRule(ctx context.Context, id string) error
	GetSecGroupRule(ctx context.Context, id string) (*rules.SecGroupRule, error)

	ListNetwork(ctx context.Context, opts networks.ListOptsBuilder) ([]networks.Network, error)
	CreateNetwork(ctx context.Context, opts networks.CreateOptsBuilder) (*networks.Network, error)
	DeleteNetwork(ctx context.Context, id string) error
	GetNetwork(ctx context.Context, id string) (*networks.Network, error)
	UpdateNetwork(ctx context.Context, id string, opts networks.UpdateOptsBuilder) (*networks.Network, error)

	ListSubnet(ctx context.Context, opts subnets.ListOptsBuilder) ([]subnets.Subnet, error)
	CreateSubnet(ctx context.Context, opts subnets.CreateOptsBuilder) (*subnets.Subnet, error)
	DeleteSubnet(ctx context.Context, id string) error
	GetSubnet(ctx context.Context, id string) (*subnets.Subnet, error)
	UpdateSubnet(ctx context.Context, id string, opts subnets.UpdateOptsBuilder) (*subnets.Subnet, error)

	ListExtensions(ctx context.Context) ([]extensions.Extension, error)

	ReplaceAllAttributesTags(ctx context.Context, resourceType string, resourceID string, opts attributestags.ReplaceAllOptsBuilder) ([]string, error)
}

type networkClient struct {
	serviceClient *gophercloud.ServiceClient
}

var _ NetworkClient = &networkClient{}

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

func (c networkClient) AddRouterInterface(ctx context.Context, id string, opts routers.AddInterfaceOptsBuilder) (*routers.InterfaceInfo, error) {
	mc := metrics.NewMetricPrometheusContext("server_os_interface", "create")
	interfaceInfo, err := routers.AddInterface(ctx, c.serviceClient, id, opts).Extract()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return interfaceInfo, nil
}

func (c networkClient) RemoveRouterInterface(ctx context.Context, id string, opts routers.RemoveInterfaceOptsBuilder) (*routers.InterfaceInfo, error) {
	mc := metrics.NewMetricPrometheusContext("server_os_interface", "delete")
	interfaceInfo, err := routers.RemoveInterface(ctx, c.serviceClient, id, opts).Extract()
	if mc.ObserveRequestIgnoreNotFound(err) != nil {
		return nil, err
	}
	return interfaceInfo, nil
}

func (c networkClient) ReplaceAllAttributesTags(ctx context.Context, resourceType string, resourceID string, opts attributestags.ReplaceAllOptsBuilder) ([]string, error) {
	mc := metrics.NewMetricPrometheusContext("attributes_tags", "replace_all")
	tags, err := attributestags.ReplaceAll(ctx, c.serviceClient, resourceType, resourceID, opts).Extract()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return tags, nil
}

func (c networkClient) ListRouter(ctx context.Context, opts routers.ListOpts) ([]routers.Router, error) {
	mc := metrics.NewMetricPrometheusContext("router", "list")
	allPages, err := routers.List(c.serviceClient, opts).AllPages(ctx)
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return routers.ExtractRouters(allPages)
}

func (c networkClient) ListFloatingIP(ctx context.Context, opts floatingips.ListOptsBuilder) ([]floatingips.FloatingIP, error) {
	mc := metrics.NewMetricPrometheusContext("floating_ip", "list")
	allPages, err := floatingips.List(c.serviceClient, opts).AllPages(ctx)
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return floatingips.ExtractFloatingIPs(allPages)
}

func (c networkClient) CreateFloatingIP(ctx context.Context, opts floatingips.CreateOptsBuilder) (*floatingips.FloatingIP, error) {
	mc := metrics.NewMetricPrometheusContext("floating_ip", "create")
	fip, err := floatingips.Create(ctx, c.serviceClient, opts).Extract()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return fip, nil
}

func (c networkClient) DeleteFloatingIP(ctx context.Context, id string) error {
	mc := metrics.NewMetricPrometheusContext("floating_ip", "delete")
	return mc.ObserveRequestIgnoreNotFound(floatingips.Delete(ctx, c.serviceClient, id).ExtractErr())
}

func (c networkClient) GetFloatingIP(ctx context.Context, id string) (*floatingips.FloatingIP, error) {
	mc := metrics.NewMetricPrometheusContext("floating_ip", "list")
	fip, err := floatingips.Get(ctx, c.serviceClient, id).Extract()
	if mc.ObserveRequestIgnoreNotFound(err) != nil {
		return nil, err
	}
	return fip, nil
}

func (c networkClient) UpdateFloatingIP(ctx context.Context, id string, opts floatingips.UpdateOptsBuilder) (*floatingips.FloatingIP, error) {
	mc := metrics.NewMetricPrometheusContext("floating_ip", "update")
	fip, err := floatingips.Update(ctx, c.serviceClient, id, opts).Extract()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return fip, nil
}

func (c networkClient) ListPort(ctx context.Context, opts ports.ListOptsBuilder) ([]ports.Port, error) {
	mc := metrics.NewMetricPrometheusContext("port", "list")
	allPages, err := ports.List(c.serviceClient, opts).AllPages(ctx)
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return ports.ExtractPorts(allPages)
}

func (c networkClient) CreatePort(ctx context.Context, opts ports.CreateOptsBuilder) (*ports.Port, error) {
	mc := metrics.NewMetricPrometheusContext("port", "create")
	port, err := ports.Create(ctx, c.serviceClient, opts).Extract()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return port, nil
}

func (c networkClient) DeletePort(ctx context.Context, id string) error {
	mc := metrics.NewMetricPrometheusContext("port", "delete")
	return mc.ObserveRequestIgnoreNotFound(ports.Delete(ctx, c.serviceClient, id).ExtractErr())
}

func (c networkClient) GetPort(ctx context.Context, id string) (*ports.Port, error) {
	mc := metrics.NewMetricPrometheusContext("port", "get")
	port, err := ports.Get(ctx, c.serviceClient, id).Extract()
	if mc.ObserveRequestIgnoreNotFound(err) != nil {
		return nil, err
	}
	return port, nil
}

func (c networkClient) UpdatePort(ctx context.Context, id string, opts ports.UpdateOptsBuilder) (*ports.Port, error) {
	mc := metrics.NewMetricPrometheusContext("port", "update")
	port, err := ports.Update(ctx, c.serviceClient, id, opts).Extract()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return port, nil
}

func (c networkClient) CreateTrunk(ctx context.Context, opts trunks.CreateOptsBuilder) (*trunks.Trunk, error) {
	mc := metrics.NewMetricPrometheusContext("trunk", "create")
	trunk, err := trunks.Create(ctx, c.serviceClient, opts).Extract()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return trunk, nil
}

func (c networkClient) DeleteTrunk(ctx context.Context, id string) error {
	mc := metrics.NewMetricPrometheusContext("trunk", "delete")
	return mc.ObserveRequestIgnoreNotFound(trunks.Delete(ctx, c.serviceClient, id).ExtractErr())
}

func (c networkClient) ListTrunkSubports(ctx context.Context, trunkID string) ([]trunks.Subport, error) {
	mc := metrics.NewMetricPrometheusContext("trunk", "listsubports")
	subports, err := trunks.GetSubports(ctx, c.serviceClient, trunkID).Extract()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return subports, nil
}

func (c networkClient) RemoveSubports(ctx context.Context, id string, opts trunks.RemoveSubportsOpts) error {
	mc := metrics.NewMetricPrometheusContext("trunk", "deletesubports")
	_, err := trunks.RemoveSubports(ctx, c.serviceClient, id, opts).Extract()
	if mc.ObserveRequest(err) != nil {
		return err
	}
	return nil
}

func (c networkClient) ListTrunk(ctx context.Context, opts trunks.ListOptsBuilder) ([]trunks.Trunk, error) {
	mc := metrics.NewMetricPrometheusContext("trunk", "list")
	allPages, err := trunks.List(c.serviceClient, opts).AllPages(ctx)
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return trunks.ExtractTrunks(allPages)
}

func (c networkClient) CreateRouter(ctx context.Context, opts routers.CreateOptsBuilder) (*routers.Router, error) {
	mc := metrics.NewMetricPrometheusContext("router", "create")
	router, err := routers.Create(ctx, c.serviceClient, opts).Extract()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return router, nil
}

func (c networkClient) DeleteRouter(ctx context.Context, id string) error {
	mc := metrics.NewMetricPrometheusContext("router", "delete")
	return mc.ObserveRequestIgnoreNotFound(routers.Delete(ctx, c.serviceClient, id).ExtractErr())
}

func (c networkClient) GetRouter(ctx context.Context, id string) (*routers.Router, error) {
	mc := metrics.NewMetricPrometheusContext("router", "get")
	router, err := routers.Get(ctx, c.serviceClient, id).Extract()
	if mc.ObserveRequestIgnoreNotFound(err) != nil {
		return nil, err
	}
	return router, nil
}

func (c networkClient) UpdateRouter(ctx context.Context, id string, opts routers.UpdateOptsBuilder) (*routers.Router, error) {
	mc := metrics.NewMetricPrometheusContext("router", "update")
	router, err := routers.Update(ctx, c.serviceClient, id, opts).Extract()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return router, nil
}

func (c networkClient) ListSecGroup(ctx context.Context, opts groups.ListOpts) ([]groups.SecGroup, error) {
	mc := metrics.NewMetricPrometheusContext("group", "list")
	allPages, err := groups.List(c.serviceClient, opts).AllPages(ctx)
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return groups.ExtractGroups(allPages)
}

func (c networkClient) CreateSecGroup(ctx context.Context, opts groups.CreateOptsBuilder) (*groups.SecGroup, error) {
	mc := metrics.NewMetricPrometheusContext("security_group", "create")
	group, err := groups.Create(ctx, c.serviceClient, opts).Extract()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return group, nil
}

func (c networkClient) DeleteSecGroup(ctx context.Context, id string) error {
	mc := metrics.NewMetricPrometheusContext("security_group", "delete")
	return mc.ObserveRequestIgnoreNotFound(groups.Delete(ctx, c.serviceClient, id).ExtractErr())
}

func (c networkClient) GetSecGroup(ctx context.Context, id string) (*groups.SecGroup, error) {
	mc := metrics.NewMetricPrometheusContext("security_group", "get")
	group, err := groups.Get(ctx, c.serviceClient, id).Extract()
	if mc.ObserveRequestIgnoreNotFound(err) != nil {
		return nil, err
	}
	return group, nil
}

func (c networkClient) UpdateSecGroup(ctx context.Context, id string, opts groups.UpdateOptsBuilder) (*groups.SecGroup, error) {
	mc := metrics.NewMetricPrometheusContext("security_group", "update")
	group, err := groups.Update(ctx, c.serviceClient, id, opts).Extract()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return group, nil
}

func (c networkClient) ListSecGroupRule(ctx context.Context, opts rules.ListOpts) ([]rules.SecGroupRule, error) {
	mc := metrics.NewMetricPrometheusContext("security_group_rule", "list")
	allPages, err := rules.List(c.serviceClient, opts).AllPages(ctx)
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return rules.ExtractRules(allPages)
}

func (c networkClient) CreateSecGroupRule(ctx context.Context, opts rules.CreateOptsBuilder) (*rules.SecGroupRule, error) {
	mc := metrics.NewMetricPrometheusContext("security_group_rule", "create")
	rule, err := rules.Create(ctx, c.serviceClient, opts).Extract()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return rule, nil
}

func (c networkClient) DeleteSecGroupRule(ctx context.Context, id string) error {
	mc := metrics.NewMetricPrometheusContext("security_group_rule", "delete")
	return mc.ObserveRequestIgnoreNotFound(rules.Delete(ctx, c.serviceClient, id).ExtractErr())
}

func (c networkClient) GetSecGroupRule(ctx context.Context, id string) (*rules.SecGroupRule, error) {
	mc := metrics.NewMetricPrometheusContext("security_group_rule", "get")
	rule, err := rules.Get(ctx, c.serviceClient, id).Extract()
	if mc.ObserveRequestIgnoreNotFound(err) != nil {
		return nil, err
	}
	return rule, nil
}

func (c networkClient) ListNetwork(ctx context.Context, opts networks.ListOptsBuilder) ([]networks.Network, error) {
	mc := metrics.NewMetricPrometheusContext("network", "list")
	allPages, err := networks.List(c.serviceClient, opts).AllPages(ctx)
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return networks.ExtractNetworks(allPages)
}

func (c networkClient) CreateNetwork(ctx context.Context, opts networks.CreateOptsBuilder) (*networks.Network, error) {
	mc := metrics.NewMetricPrometheusContext("network", "create")
	net, err := networks.Create(ctx, c.serviceClient, opts).Extract()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return net, nil
}

func (c networkClient) DeleteNetwork(ctx context.Context, id string) error {
	mc := metrics.NewMetricPrometheusContext("network", "delete")
	return mc.ObserveRequestIgnoreNotFound(networks.Delete(ctx, c.serviceClient, id).ExtractErr())
}

func (c networkClient) GetNetwork(ctx context.Context, id string) (*networks.Network, error) {
	mc := metrics.NewMetricPrometheusContext("network", "get")
	net, err := networks.Get(ctx, c.serviceClient, id).Extract()
	if mc.ObserveRequestIgnoreNotFound(err) != nil {
		return nil, err
	}
	return net, nil
}

func (c networkClient) UpdateNetwork(ctx context.Context, id string, opts networks.UpdateOptsBuilder) (*networks.Network, error) {
	mc := metrics.NewMetricPrometheusContext("network", "update")
	net, err := networks.Update(ctx, c.serviceClient, id, opts).Extract()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return net, nil
}

func (c networkClient) ListSubnet(ctx context.Context, opts subnets.ListOptsBuilder) ([]subnets.Subnet, error) {
	mc := metrics.NewMetricPrometheusContext("subnet", "list")
	allPages, err := subnets.List(c.serviceClient, opts).AllPages(ctx)
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return subnets.ExtractSubnets(allPages)
}

func (c networkClient) CreateSubnet(ctx context.Context, opts subnets.CreateOptsBuilder) (*subnets.Subnet, error) {
	mc := metrics.NewMetricPrometheusContext("subnet", "create")
	subnet, err := subnets.Create(ctx, c.serviceClient, opts).Extract()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return subnet, nil
}

func (c networkClient) DeleteSubnet(ctx context.Context, id string) error {
	mc := metrics.NewMetricPrometheusContext("subnet", "delete")
	return mc.ObserveRequestIgnoreNotFound(subnets.Delete(ctx, c.serviceClient, id).ExtractErr())
}

func (c networkClient) GetSubnet(ctx context.Context, id string) (*subnets.Subnet, error) {
	mc := metrics.NewMetricPrometheusContext("subnet", "get")
	subnet, err := subnets.Get(ctx, c.serviceClient, id).Extract()
	if mc.ObserveRequestIgnoreNotFound(err) != nil {
		return nil, err
	}
	return subnet, nil
}

func (c networkClient) UpdateSubnet(ctx context.Context, id string, opts subnets.UpdateOptsBuilder) (*subnets.Subnet, error) {
	mc := metrics.NewMetricPrometheusContext("subnet", "update")
	subnet, err := subnets.Update(ctx, c.serviceClient, id, opts).Extract()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return subnet, nil
}

func (c networkClient) ListExtensions(ctx context.Context) ([]extensions.Extension, error) {
	mc := metrics.NewMetricPrometheusContext("network_extension", "list")
	allPages, err := extensions.List(c.serviceClient).AllPages(ctx)
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return extensions.ExtractExtensions(allPages)
}
