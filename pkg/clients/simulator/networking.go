/*
Copyright 2022 The Kubernetes Authors.

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

package simulator

import (
	"fmt"
	"net"
	"regexp"
	"strconv"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	netutils "k8s.io/utils/net"
	"k8s.io/utils/pointer"

	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
)

type (
	ListFloatingIPPreHook            func(opts floatingips.ListOptsBuilder) (bool, []floatingips.FloatingIP, error)
	ListFloatingIPPostHook           func(floatingips.ListOptsBuilder, []floatingips.FloatingIP, error)
	CreateFloatingIPPreHook          func(opts floatingips.CreateOptsBuilder) (bool, *floatingips.FloatingIP, error)
	CreateFloatingIPPostHook         func(floatingips.CreateOptsBuilder, *floatingips.FloatingIP, error)
	DeleteFloatingIPPreHook          func(id string) (bool, error)
	DeleteFloatingIPPostHook         func(string, error)
	GetFloatingIPPreHook             func(id string) (bool, *floatingips.FloatingIP, error)
	GetFloatingIPPostHook            func(string, *floatingips.FloatingIP, error)
	UpdateFloatingIPPreHook          func(id string, opts floatingips.UpdateOptsBuilder) (bool, *floatingips.FloatingIP, error)
	UpdateFloatingIPPostHook         func(string, floatingips.UpdateOptsBuilder, *floatingips.FloatingIP, error)
	ListPortPreHook                  func(opts ports.ListOptsBuilder) (bool, []ports.Port, error)
	ListPortPostHook                 func(ports.ListOptsBuilder, []ports.Port, error)
	CreatePortPreHook                func(opts ports.CreateOptsBuilder) (bool, *ports.Port, error)
	CreatePortPostHook               func(ports.CreateOptsBuilder, *ports.Port, error)
	DeletePortPreHook                func(id string) (bool, error)
	DeletePortPostHook               func(string, error)
	GetPortPreHook                   func(id string) (bool, *ports.Port, error)
	GetPortPostHook                  func(string, *ports.Port, error)
	UpdatePortPreHook                func(id string, opts ports.UpdateOptsBuilder) (bool, *ports.Port, error)
	UpdatePortPostHook               func(string, ports.UpdateOptsBuilder, *ports.Port, error)
	ListTrunkPreHook                 func(opts trunks.ListOptsBuilder) (bool, []trunks.Trunk, error)
	ListTrunkPostHook                func(trunks.ListOptsBuilder, []trunks.Trunk, error)
	CreateTrunkPreHook               func(opts trunks.CreateOptsBuilder) (bool, *trunks.Trunk, error)
	CreateTrunkPostHook              func(trunks.CreateOptsBuilder, *trunks.Trunk, error)
	DeleteTrunkPreHook               func(id string) (bool, error)
	DeleteTrunkPostHook              func(string, error)
	ListRouterPreHook                func(opts routers.ListOpts) (bool, []routers.Router, error)
	ListRouterPostHook               func(routers.ListOpts, []routers.Router, error)
	CreateRouterPreHook              func(opts routers.CreateOptsBuilder) (bool, *routers.Router, error)
	CreateRouterPostHook             func(routers.CreateOptsBuilder, *routers.Router, error)
	DeleteRouterPreHook              func(id string) (bool, error)
	DeleteRouterPostHook             func(string, error)
	GetRouterPreHook                 func(id string) (bool, *routers.Router, error)
	GetRouterPostHook                func(string, *routers.Router, error)
	UpdateRouterPreHook              func(id string, opts routers.UpdateOptsBuilder) (bool, *routers.Router, error)
	UpdateRouterPostHook             func(string, routers.UpdateOptsBuilder, *routers.Router, error)
	AddRouterInterfacePreHook        func(id string, opts routers.AddInterfaceOptsBuilder) (bool, *routers.InterfaceInfo, error)
	AddRouterInterfacePostHook       func(string, routers.AddInterfaceOptsBuilder, *routers.InterfaceInfo, error)
	RemoveRouterInterfacePreHook     func(id string, opts routers.RemoveInterfaceOptsBuilder) (bool, *routers.InterfaceInfo, error)
	RemoveRouterInterfacePostHook    func(string, routers.RemoveInterfaceOptsBuilder, *routers.InterfaceInfo, error)
	ListSecGroupPreHook              func(opts groups.ListOpts) (bool, []groups.SecGroup, error)
	ListSecGroupPostHook             func(groups.ListOpts, []groups.SecGroup, error)
	CreateSecGroupPreHook            func(opts groups.CreateOptsBuilder) (bool, *groups.SecGroup, error)
	CreateSecGroupPostHook           func(groups.CreateOptsBuilder, *groups.SecGroup, error)
	DeleteSecGroupPreHook            func(id string) (bool, error)
	DeleteSecGroupPostHook           func(string, error)
	GetSecGroupPreHook               func(id string) (bool, *groups.SecGroup, error)
	GetSecGroupPostHook              func(string, *groups.SecGroup, error)
	UpdateSecGroupPreHook            func(id string, opts groups.UpdateOptsBuilder) (bool, *groups.SecGroup, error)
	UpdateSecGroupPostHook           func(string, groups.UpdateOptsBuilder, *groups.SecGroup, error)
	ListSecGroupRulePreHook          func(opts rules.ListOpts) (bool, []rules.SecGroupRule, error)
	ListSecGroupRulePostHook         func(rules.ListOpts, []rules.SecGroupRule, error)
	CreateSecGroupRulePreHook        func(opts rules.CreateOptsBuilder) (bool, *rules.SecGroupRule, error)
	CreateSecGroupRulePostHook       func(rules.CreateOptsBuilder, *rules.SecGroupRule, error)
	DeleteSecGroupRulePreHook        func(id string) (bool, error)
	DeleteSecGroupRulePostHook       func(string, error)
	GetSecGroupRulePreHook           func(id string) (bool, *rules.SecGroupRule, error)
	GetSecGroupRulePostHook          func(string, *rules.SecGroupRule, error)
	ListNetworkPreHook               func(opts networks.ListOptsBuilder) (bool, []networks.Network, error)
	ListNetworkPostHook              func(networks.ListOptsBuilder, []networks.Network, error)
	CreateNetworkPreHook             func(opts networks.CreateOptsBuilder) (bool, *networks.Network, error)
	CreateNetworkPostHook            func(networks.CreateOptsBuilder, *networks.Network, error)
	DeleteNetworkPreHook             func(id string) (bool, error)
	DeleteNetworkPostHook            func(string, error)
	GetNetworkPreHook                func(id string) (bool, *networks.Network, error)
	GetNetworkPostHook               func(string, *networks.Network, error)
	UpdateNetworkPreHook             func(id string, opts networks.UpdateOptsBuilder) (bool, *networks.Network, error)
	UpdateNetworkPostHook            func(string, networks.UpdateOptsBuilder, *networks.Network, error)
	ListSubnetPreHook                func(opts subnets.ListOptsBuilder) (bool, []subnets.Subnet, error)
	ListSubnetPostHook               func(subnets.ListOptsBuilder, []subnets.Subnet, error)
	CreateSubnetPreHook              func(opts subnets.CreateOptsBuilder) (bool, *subnets.Subnet, error)
	CreateSubnetPostHook             func(subnets.CreateOptsBuilder, *subnets.Subnet, error)
	DeleteSubnetPreHook              func(id string) (bool, error)
	DeleteSubnetPostHook             func(string, error)
	GetSubnetPreHook                 func(id string) (bool, *subnets.Subnet, error)
	GetSubnetPostHook                func(string, *subnets.Subnet, error)
	UpdateSubnetPreHook              func(id string, opts subnets.UpdateOptsBuilder) (bool, *subnets.Subnet, error)
	UpdateSubnetPostHook             func(string, subnets.UpdateOptsBuilder, *subnets.Subnet, error)
	ListExtensionsPreHook            func() (bool, []extensions.Extension, error)
	ListExtensionsPostHook           func([]extensions.Extension, error)
	ReplaceAllAttributesTagsPreHook  func(resourceType string, resourceID string, opts attributestags.ReplaceAllOptsBuilder) (bool, []string, error)
	ReplaceAllAttributesTagsPostHook func(string, string, attributestags.ReplaceAllOptsBuilder, []string, error)
)

type SimNetwork struct {
	networks.Network
	external.NetworkExternalExt
}

type SimSubnet struct {
	subnets.Subnet

	SimIPNet  *net.IPNet
	SimLastIP int
}

type NetworkSimulator struct {
	Simulator *OpenStackSimulator

	Extensions       []extensions.Extension
	FloatingIPs      []floatingips.FloatingIP
	Networks         []SimNetwork
	Ports            []ports.Port
	Routers          []routers.Router
	RouterInterfaces map[string][]string
	SecGroupRules    []rules.SecGroupRule
	SecGroups        []groups.SecGroup
	Subnets          []SimSubnet
	Trunks           []trunks.Trunk

	ListFloatingIPPreHook            ListFloatingIPPreHook
	ListFloatingIPPostHook           ListFloatingIPPostHook
	CreateFloatingIPPreHook          CreateFloatingIPPreHook
	CreateFloatingIPPostHook         CreateFloatingIPPostHook
	DeleteFloatingIPPreHook          DeleteFloatingIPPreHook
	DeleteFloatingIPPostHook         DeleteFloatingIPPostHook
	GetFloatingIPPreHook             GetFloatingIPPreHook
	GetFloatingIPPostHook            GetFloatingIPPostHook
	UpdateFloatingIPPreHook          UpdateFloatingIPPreHook
	UpdateFloatingIPPostHook         UpdateFloatingIPPostHook
	ListPortPreHook                  ListPortPreHook
	ListPortPostHook                 ListPortPostHook
	CreatePortPreHook                CreatePortPreHook
	CreatePortPostHook               CreatePortPostHook
	DeletePortPreHook                DeletePortPreHook
	DeletePortPostHook               DeletePortPostHook
	GetPortPreHook                   GetPortPreHook
	GetPortPostHook                  GetPortPostHook
	UpdatePortPreHook                UpdatePortPreHook
	UpdatePortPostHook               UpdatePortPostHook
	ListTrunkPreHook                 ListTrunkPreHook
	ListTrunkPostHook                ListTrunkPostHook
	CreateTrunkPreHook               CreateTrunkPreHook
	CreateTrunkPostHook              CreateTrunkPostHook
	DeleteTrunkPreHook               DeleteTrunkPreHook
	DeleteTrunkPostHook              DeleteTrunkPostHook
	ListRouterPreHook                ListRouterPreHook
	ListRouterPostHook               ListRouterPostHook
	CreateRouterPreHook              CreateRouterPreHook
	CreateRouterPostHook             CreateRouterPostHook
	DeleteRouterPreHook              DeleteRouterPreHook
	DeleteRouterPostHook             DeleteRouterPostHook
	GetRouterPreHook                 GetRouterPreHook
	GetRouterPostHook                GetRouterPostHook
	UpdateRouterPreHook              UpdateRouterPreHook
	UpdateRouterPostHook             UpdateRouterPostHook
	AddRouterInterfacePreHook        AddRouterInterfacePreHook
	AddRouterInterfacePostHook       AddRouterInterfacePostHook
	RemoveRouterInterfacePreHook     RemoveRouterInterfacePreHook
	RemoveRouterInterfacePostHook    RemoveRouterInterfacePostHook
	ListSecGroupPreHook              ListSecGroupPreHook
	ListSecGroupPostHook             ListSecGroupPostHook
	CreateSecGroupPreHook            CreateSecGroupPreHook
	CreateSecGroupPostHook           CreateSecGroupPostHook
	DeleteSecGroupPreHook            DeleteSecGroupPreHook
	DeleteSecGroupPostHook           DeleteSecGroupPostHook
	GetSecGroupPreHook               GetSecGroupPreHook
	GetSecGroupPostHook              GetSecGroupPostHook
	UpdateSecGroupPreHook            UpdateSecGroupPreHook
	UpdateSecGroupPostHook           UpdateSecGroupPostHook
	ListSecGroupRulePreHook          ListSecGroupRulePreHook
	ListSecGroupRulePostHook         ListSecGroupRulePostHook
	CreateSecGroupRulePreHook        CreateSecGroupRulePreHook
	CreateSecGroupRulePostHook       CreateSecGroupRulePostHook
	DeleteSecGroupRulePreHook        DeleteSecGroupRulePreHook
	DeleteSecGroupRulePostHook       DeleteSecGroupRulePostHook
	GetSecGroupRulePreHook           GetSecGroupRulePreHook
	GetSecGroupRulePostHook          GetSecGroupRulePostHook
	ListNetworkPreHook               ListNetworkPreHook
	ListNetworkPostHook              ListNetworkPostHook
	CreateNetworkPreHook             CreateNetworkPreHook
	CreateNetworkPostHook            CreateNetworkPostHook
	DeleteNetworkPreHook             DeleteNetworkPreHook
	DeleteNetworkPostHook            DeleteNetworkPostHook
	GetNetworkPreHook                GetNetworkPreHook
	GetNetworkPostHook               GetNetworkPostHook
	UpdateNetworkPreHook             UpdateNetworkPreHook
	UpdateNetworkPostHook            UpdateNetworkPostHook
	ListSubnetPreHook                ListSubnetPreHook
	ListSubnetPostHook               ListSubnetPostHook
	CreateSubnetPreHook              CreateSubnetPreHook
	CreateSubnetPostHook             CreateSubnetPostHook
	DeleteSubnetPreHook              DeleteSubnetPreHook
	DeleteSubnetPostHook             DeleteSubnetPostHook
	GetSubnetPreHook                 GetSubnetPreHook
	GetSubnetPostHook                GetSubnetPostHook
	UpdateSubnetPreHook              UpdateSubnetPreHook
	UpdateSubnetPostHook             UpdateSubnetPostHook
	ListExtensionsPreHook            ListExtensionsPreHook
	ListExtensionsPostHook           ListExtensionsPostHook
	ReplaceAllAttributesTagsPreHook  ReplaceAllAttributesTagsPreHook
	ReplaceAllAttributesTagsPostHook ReplaceAllAttributesTagsPostHook
}

const (
	FloatingIPStatusActive = "ACTIVE"
	FloatingIPStatusDown   = "DOWN"
	FloatingIPStatusError  = "ERROR"
)

func NewNetworkSimulator(p *OpenStackSimulator) *NetworkSimulator {
	sim := &NetworkSimulator{
		Simulator:        p,
		RouterInterfaces: make(map[string][]string),
	}

	sim.UpdateFloatingIPPostHook = sim.CallBackFloatingIPSetActiveAfterAssociate
	return sim
}

/*
 * Simulator implementation methods
 */

func (c *NetworkSimulator) ImplListFloatingIP(opts floatingips.ListOptsBuilder) ([]floatingips.FloatingIP, error) {
	query, err := opts.ToFloatingIPListQuery()
	if err != nil {
		return nil, err
	}
	values, err := getValuesFromQuery(query)
	if err != nil {
		return nil, fmt.Errorf("ListFloatingIP: %w", err)
	}

	var result []floatingips.FloatingIP
fips:
	for _, fip := range c.FloatingIPs {
		for k, v := range values {
			switch k {
			case "floating_ip_address":
				if fip.FloatingIP != v {
					continue fips
				}
			default:
				panic(fmt.Sprintf("ListFloatingIP: unknown query parameter: %s:%+v", k, v))
			}
		}
		result = append(result, fip)
	}

	return result, nil
}

func (c *NetworkSimulator) ImplCreateFloatingIP(opts floatingips.CreateOptsBuilder) (*floatingips.FloatingIP, error) {
	createMap, err := opts.ToFloatingIPCreateMap()
	if err != nil {
		return nil, fmt.Errorf("CreateFloatingIP: creating floating ip map: %w", err)
	}
	createMap, ok := createMap["floatingip"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("CreateFloatingIP: create map doesn't contain floatingip: %w", err)
	}

	floatingip := floatingips.FloatingIP{}
	floatingip.ID = generateUUID()
	floatingip.Status = FloatingIPStatusDown

	for k, v := range createMap {
		switch k {
		case "description":
			floatingip.Description = v.(string)
		case "floating_network_id":
			floatingip.FloatingNetworkID = v.(string)
		default:
			panic(fmt.Errorf("CreateFloatingIP: unsupported field %s", k))
		}
	}

	subnet := c.firstSubnetForNetwork(floatingip.FloatingNetworkID)
	if subnet == nil {
		panic(fmt.Errorf("CreateFloatingIP: no subnet for network %s", floatingip.FloatingNetworkID))
	}
	ip, err := c.SimNextIPForSubnet(subnet.ID)
	if err != nil {
		return nil, fmt.Errorf("CreateFloatingIP: getting next ip for subnet %s: %w", subnet.ID, err)
	}
	floatingip.FloatingIP = ip

	c.FloatingIPs = append(c.FloatingIPs, floatingip)
	return &floatingip, nil
}

func (c *NetworkSimulator) ImplDeleteFloatingIP(id string) error {
	var fip *floatingips.FloatingIP
	for i := range c.FloatingIPs {
		o := &c.FloatingIPs[i]
		if o.ID == id {
			c.FloatingIPs = append(c.FloatingIPs[:i], c.FloatingIPs[i+1:]...)
			fip = o
			break
		}
	}

	if fip == nil {
		err := &gophercloud.ErrDefault404{}
		err.Info = fmt.Sprintf("DeleteFloatingIP: Floating IP %s not found", id)
		return err
	}

	if fip.PortID == "" {
		return nil
	}

	var port *ports.Port
	for i := range c.Ports {
		o := &c.Ports[i]
		if o.ID == fip.PortID {
			port = o
			break
		}
	}
	if port == nil {
		panic(fmt.Errorf("DeleteFloatingIP: referenced port %s not found", fip.PortID))
	}
	if port.DeviceID == "" {
		return nil
	}

	var server *SimServer
	for i := range c.Simulator.Compute.Servers {
		o := &c.Simulator.Compute.Servers[i]
		if o.ID == port.DeviceID {
			server = o
			break
		}
	}
	if server == nil {
		panic(fmt.Errorf("DeleteFloatingIP: referenced server %s not found", port.DeviceID))
	}
	if server.Addresses == nil {
		return nil
	}

	networkName := func() string {
		for _, n := range c.Networks {
			if n.ID == port.NetworkID {
				return n.Name
			}
		}
		panic(fmt.Errorf("DeleteFloatingIP: referenced network %s not found", port.NetworkID))
	}()

	ifAddresses := server.Addresses[networkName].([]map[string]interface{})
	for i := range ifAddresses {
		ifAddress := ifAddresses[i]
		if ifAddress["OS-EXT-IPS:type"] == "floating" && ifAddress["addr"] == fip.FloatingIP {
			ifAddresses = append(ifAddresses[:i], ifAddresses[i+1:]...)
			break
		}
	}
	server.Addresses[networkName] = ifAddresses

	return nil
}

func (c *NetworkSimulator) ImplGetFloatingIP(id string) (*floatingips.FloatingIP, error) {
	for _, floatingip := range c.FloatingIPs {
		if floatingip.ID == id {
			return &floatingip, nil
		}
	}

	err := &gophercloud.ErrDefault404{}
	err.Info = fmt.Sprintf("GetFloatingIP: Floating IP %s not found", id)
	return nil, err
}

func (c *NetworkSimulator) ImplUpdateFloatingIP(id string, opts floatingips.UpdateOptsBuilder) (*floatingips.FloatingIP, error) {
	updateMap, err := opts.ToFloatingIPUpdateMap()
	if err != nil {
		return nil, fmt.Errorf("UpdateFloatingIP: creating floating ip update map: %w", err)
	}
	updateMap, ok := updateMap["floatingip"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("UpdateFloatingIP: update map doesn't contain floatingip: %w", err)
	}

	floatingip := func() *floatingips.FloatingIP {
		for i := range c.FloatingIPs {
			if c.FloatingIPs[i].ID == id {
				return &c.FloatingIPs[i]
			}
		}
		return nil
	}()
	if floatingip == nil {
		err := &gophercloud.ErrDefault404{}
		err.Info = fmt.Sprintf("UpdateFloatingIP: Floating IP %s not found", id)
		return nil, err
	}

	portAddFloatingIP := func(portID string) error {
		port := func() *ports.Port {
			for i := range c.Ports {
				if c.Ports[i].ID == portID {
					return &c.Ports[i]
				}
			}
			return nil
		}()
		if port == nil {
			err := &gophercloud.ErrDefault404{}
			err.Info = fmt.Sprintf("UpdateFloatingIP: Port %s not found", portID)
			return err
		}

		port.FixedIPs = append(port.FixedIPs, ports.IP{
			SubnetID:  id,
			IPAddress: floatingip.FloatingIP,
		})

		// Add the port to an attached server
		// See also population of server.Addresses in CreateServer
		if port.DeviceID != "" {
			server := func() *SimServer {
				for i := range c.Simulator.Compute.Servers {
					server := &c.Simulator.Compute.Servers[i]
					if server.ID == port.DeviceID {
						return server
					}
				}
				return nil
			}()
			if server == nil {
				panic(fmt.Errorf("UpdateFloatingIP: port referenced server %s not found", port.DeviceID))
			}

			networkName := func() string {
				for _, network := range c.Networks {
					if network.ID == port.NetworkID {
						return network.Name
					}
				}
				panic(fmt.Sprintf("UpdateFloatingIP: port %s has network %s, but no such network", port.ID, port.NetworkID))
			}()
			if server.Addresses == nil {
				server.Addresses = make(map[string]interface{})
			}
			addresses := func() []map[string]interface{} {
				if addresses, ok := server.Addresses[networkName]; ok {
					return addresses.([]map[string]interface{})
				}
				return nil
			}()
			addresses = append(addresses, map[string]interface{}{
				"addr":            floatingip.FloatingIP,
				"version":         4,
				"OS-EXT-IPS:type": "floating",
			})
			server.Addresses[networkName] = addresses
		}
		return nil
	}

	for k, v := range updateMap {
		switch k {
		case "port_id":
			portID := v.(string)
			err = portAddFloatingIP(portID)
			if err != nil {
				return nil, fmt.Errorf("UpdateFloatingIP: adding floating ip to port: %w", err)
			}
			floatingip.PortID = portID
		default:
			panic(fmt.Errorf("UpdateFloatingIP: unsupported field %s", k))
		}
	}

	retCopy := *floatingip
	return &retCopy, nil
}

func (c *NetworkSimulator) ImplListPort(opts ports.ListOptsBuilder) ([]ports.Port, error) {
	query, err := opts.ToPortListQuery()
	if err != nil {
		return nil, fmt.Errorf("creating port list query: %w", err)
	}
	queryValues, err := getValuesFromQuery(query)
	if err != nil {
		return nil, fmt.Errorf("ListPort: parsing values from query: %w", err)
	}

	limit, err := func() (int, error) {
		limit, ok := queryValues["limit"]
		if !ok {
			return 0, nil
		}
		delete(queryValues, "limit")

		limitInt, err := strconv.Atoi(limit)
		if err != nil {
			return 0, fmt.Errorf("ListPort: parsing limit: %w", err)
		}
		return limitInt, nil
	}()
	if err != nil {
		return nil, &gophercloud.ErrDefault400{
			ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
				BaseError: gophercloud.BaseError{
					Info: err.Error(),
				},
			},
		}
	}

	ret := []ports.Port{}
ports:
	for _, port := range c.Ports {
		for k, v := range queryValues {
			switch k {
			case "name":
				if port.Name != v {
					continue ports
				}
			case "network_id":
				if port.NetworkID != v {
					continue ports
				}
			case "device_id":
				if port.DeviceID != v {
					continue ports
				}
			case "fixed_ips":
				// Port must have all of the fixed IPs specified in the query
				// I don't know what the query format is except
				// for a single entry containing an ip address,
				// so for now we panic on anything else
				ipaddress := regexp.MustCompile(`^ip_address=([0-9.]+)$`)
				matches := ipaddress.FindStringSubmatch(v)
				if len(matches) != 2 {
					panic(fmt.Errorf("ListPort: unsupported fixed_ips query: %s", v))
				}
				findip := func() bool {
					for _, fixedip := range port.FixedIPs {
						if fixedip.IPAddress == matches[1] {
							return true
						}
					}
					return false
				}
				if !findip() {
					continue ports
				}
			default:
				panic(fmt.Errorf("ListPort: unsupported query parameter %s:%+v", k, v))
			}
		}

		ret = append(ret, port)
		if len(ret) == limit {
			break
		}
	}

	return ret, nil
}

func (c *NetworkSimulator) ImplCreatePort(opts ports.CreateOptsBuilder) (*ports.Port, error) {
	createMap, err := opts.ToPortCreateMap()
	if err != nil {
		return nil, fmt.Errorf("CreatePort: creating port map: %w", err)
	}
	createMap, ok := createMap["port"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("CreatePort: create map does not contain port")
	}

	port := ports.Port{}
	port.ID = generateUUID()
	for k, v := range createMap {
		switch k {
		case "description":
			port.Description = v.(string)
		case "name":
			port.Name = v.(string)
		case "network_id":
			networkID := v.(string)
			_, err = c.GetNetwork(networkID)
			if err != nil {
				return nil, fmt.Errorf("CreatePort: %w", err)
			}
			port.NetworkID = networkID
		case "security_groups":
			if v == nil {
				continue
			}

			var sgs []string
			for _, sg := range v.([]interface{}) {
				sgs = append(sgs, sg.(string))
			}
			port.SecurityGroups = sgs
		case "fixed_ips":
			for _, fixedIP := range v.([]interface{}) {
				params := fixedIP.(map[string]interface{})
				ip := ports.IP{}
				for k, v := range params {
					switch k {
					case "ip_address":
						ip.IPAddress = v.(string)
					case "subnet_id":
						subnet, err := c.GetSubnet(v.(string))
						if err != nil {
							return nil, fmt.Errorf("CreatePort: %w", err)
						}
						ip.SubnetID = subnet.ID
					default:
						panic(fmt.Errorf("CreatePort: unsupported fixed IP field %s", k))
					}
				}
				port.FixedIPs = append(port.FixedIPs, ip)
			}
		case "device_id":
			port.DeviceID = v.(string)
		default:
			panic(fmt.Errorf("CreatePort: unsupported create parameter: %s:%+v", k, v))
		}
	}

	// If no fixed IPs were requested explicitly, create one implicitly from the first subnet on the given network
	if len(port.FixedIPs) == 0 {
		subnet := c.firstSubnetForNetwork(port.NetworkID)
		if subnet != nil {
			port.FixedIPs = append(port.FixedIPs, ports.IP{SubnetID: subnet.ID})
		}
	}

	// Allocate IP addresses for all fixed IPs
	for i := range port.FixedIPs {
		fixedIP := &port.FixedIPs[i]
		if fixedIP.IPAddress != "" {
			continue
		}

		ip, err := c.SimNextIPForSubnet(fixedIP.SubnetID)
		if err != nil {
			return nil, fmt.Errorf("CreatePort: %w", err)
		}
		fixedIP.IPAddress = ip
	}

	c.Ports = append(c.Ports, port)
	return &port, nil
}

func (c *NetworkSimulator) ImplDeletePort(id string) error {
	for i := range c.Ports {
		if c.Ports[i].ID == id {
			c.Ports = append(c.Ports[:i], c.Ports[i+i:]...)
			return nil
		}
	}
	return &gophercloud.ErrDefault404{
		ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
			BaseError: gophercloud.BaseError{
				Info: fmt.Sprintf("DeletePort: port with id %s not found", id),
			},
		},
	}
}

func (c *NetworkSimulator) ImplGetPort(id string) (*ports.Port, error) {
	for _, port := range c.Ports {
		if port.ID == id {
			retCopy := port
			return &retCopy, nil
		}
	}

	return nil, &gophercloud.ErrDefault404{
		ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
			BaseError: gophercloud.BaseError{
				Info: fmt.Sprintf("GetPort: port with id %s not found", id),
			},
		},
	}
}

func (c *NetworkSimulator) ImplUpdatePort(id string, opts ports.UpdateOptsBuilder) (*ports.Port, error) {
	panic(fmt.Errorf("UpdatePort not implemented"))
}

func (c *NetworkSimulator) ImplListTrunk(opts trunks.ListOptsBuilder) ([]trunks.Trunk, error) {
	query, err := opts.ToTrunkListQuery()
	if err != nil {
		return nil, fmt.Errorf("ListTrunk: create list query: %w", err)
	}
	queryValues, err := getValuesFromQuery(query)
	if err != nil {
		return nil, fmt.Errorf("ListTrunk: %w", err)
	}

	var ret []trunks.Trunk
trunks:
	for _, trunk := range c.Trunks {
		for k, v := range queryValues {
			switch k {
			case "port_id":
				if trunk.PortID != v {
					continue trunks
				}
			case "name":
				if trunk.Name != v {
					continue trunks
				}
			default:
				panic(fmt.Errorf("ListTrunk: simulator doesn't support query by %s", k))
			}
		}
		ret = append(ret, trunk)
	}

	return ret, nil
}

func (c *NetworkSimulator) ImplCreateTrunk(opts trunks.CreateOptsBuilder) (*trunks.Trunk, error) {
	panic(fmt.Errorf("CreateTrunk not implemented"))
}

func (c *NetworkSimulator) ImplDeleteTrunk(id string) error {
	panic(fmt.Errorf("DeleteTrunk not implemented"))
}

func (c *NetworkSimulator) ImplListRouter(opts routers.ListOpts) ([]routers.Router, error) {
	ret := []routers.Router{}
	for _, router := range c.Routers {
		if opts.ID != "" && router.ID != opts.ID {
			continue
		}
		if opts.Name != "" && router.Name != opts.Name {
			continue
		}
		if opts.Description != "" && router.Description != opts.Description {
			continue
		}
		if opts.AdminStateUp != nil && router.AdminStateUp != *opts.AdminStateUp {
			continue
		}
		if opts.Distributed != nil && router.Distributed != *opts.Distributed {
			continue
		}
		if opts.Status != "" && router.Status != opts.Status {
			continue
		}
		if opts.TenantID != "" && router.TenantID != opts.TenantID {
			continue
		}
		if opts.ProjectID != "" && router.ProjectID != opts.ProjectID {
			continue
		}
		if opts.Tags != "" || opts.TagsAny != "" ||
			opts.NotTags != "" || opts.NotTagsAny != "" ||
			opts.Limit != 0 || opts.Marker != "" ||
			opts.SortKey != "" || opts.SortDir != "" {
			panic(fmt.Errorf("ListRouter: simulator doesn't support query by tags, limit, marker, sort_key, or sort_dir"))
		}

		ret = append(ret, router)
	}

	return ret, nil
}

func (c *NetworkSimulator) ImplCreateRouter(opts routers.CreateOptsBuilder) (*routers.Router, error) {
	createMap, err := opts.ToRouterCreateMap()
	if err != nil {
		return nil, fmt.Errorf("CreateRouter: %w", err)
	}
	createMap, ok := createMap["router"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("CreateRouter: router not found in create map")
	}

	router := routers.Router{}
	router.ID = generateUUID()

	for k, v := range createMap {
		switch k {
		case "name":
			router.Name = v.(string)
		case "admin_state_up":
			router.AdminStateUp = v.(bool)
		case "distributed":
			router.Distributed = v.(bool)
		case "availability_zone_hints":
			router.AvailabilityZoneHints = v.([]string)
		case "external_gateway_info":
			for k1, v1 := range v.(map[string]interface{}) {
				switch k1 {
				case "network_id":
					network, err := c.GetNetwork(v1.(string))
					if err != nil {
						return nil, fmt.Errorf("CreateRouter: %w", err)
					}
					router.GatewayInfo.NetworkID = network.ID
				case "enable_snat":
					router.GatewayInfo.EnableSNAT = pointer.Bool(v1.(bool))
				default:
					panic(fmt.Errorf("CreateRouter: simulator doesn't support external_gateway_info.%s", k1))
				}
			}
		case "tags":
			router.Tags = v.([]string)
		case "tenant_id":
			router.TenantID = v.(string)
		case "project_id":
			router.ProjectID = v.(string)
		case "description":
			router.Description = v.(string)
		default:
			panic(fmt.Errorf("CreateRouter: unsupported create parameter: %s:%+v", k, v))
		}
	}

	c.Routers = append(c.Routers, router)
	return &router, nil
}

func (c *NetworkSimulator) ImplDeleteRouter(id string) error {
	for _, portID := range c.RouterInterfaces[id] {
		port, err := c.GetPort(portID)
		if err != nil && !capoerrors.IsNotFound(err) {
			return fmt.Errorf("DeleteRouter: %w", err)
		}
		if port != nil {
			err := &gophercloud.ErrDefault404{}
			err.Info = fmt.Sprintf("Router %s still has ports", id)
			return err
		}
	}

	for i, router := range c.Routers {
		if router.ID == id {
			c.Routers = append(c.Routers[:i], c.Routers[i+1:]...)
			return nil
		}
	}

	err := &gophercloud.ErrDefault404{}
	err.Info = fmt.Sprintf("Router %s not found", id)
	return err
}

func (c *NetworkSimulator) ImplGetRouter(id string) (*routers.Router, error) {
	for _, router := range c.Routers {
		if router.ID == id {
			retCopy := router
			return &retCopy, nil
		}
	}

	err := &gophercloud.ErrDefault404{}
	err.Info = fmt.Sprintf("GetRouter: Router %s not found", id)
	return nil, err
}

func (c *NetworkSimulator) ImplUpdateRouter(id string, opts routers.UpdateOptsBuilder) (*routers.Router, error) {
	panic(fmt.Errorf("UpdateRouter not implemented"))
}

func (c *NetworkSimulator) ImplAddRouterInterface(id string, opts routers.AddInterfaceOptsBuilder) (*routers.InterfaceInfo, error) {
	_, err := c.GetRouter(id)
	if err != nil {
		return nil, fmt.Errorf("AddRouterInterface: %w", err)
	}

	createMap, err := opts.ToRouterAddInterfaceMap()
	if err != nil {
		return nil, fmt.Errorf("AddRouterInterface: creating router interface map: %w", err)
	}

	interfaceInfo := routers.InterfaceInfo{}
	for k, v := range createMap {
		switch k {
		case "subnet_id":
			interfaceInfo.SubnetID = v.(string)
		case "port_id":
			interfaceInfo.PortID = v.(string)
		default:
			panic(fmt.Errorf("AddRouterInterface: simulator doesn't support %s", k))
		}
	}

	var port *ports.Port
	if interfaceInfo.PortID != "" {
		port, err = c.GetPort(interfaceInfo.PortID)
		if err != nil {
			return nil, fmt.Errorf("AddRouterInterface: %w", err)
		}
	} else if interfaceInfo.SubnetID != "" {
		subnet, err := c.GetSubnet(interfaceInfo.SubnetID)
		if err != nil {
			return nil, fmt.Errorf("AddRouterInterface: %w", err)
		}
		var createOpts ports.CreateOptsBuilder = ports.CreateOpts{
			Name:      fmt.Sprintf("router %s port", id),
			FixedIPs:  []ports.IP{{SubnetID: interfaceInfo.SubnetID}},
			NetworkID: subnet.NetworkID,
			DeviceID:  id,
		}

		port, err = c.CreatePort(createOpts)
		if err != nil {
			return nil, fmt.Errorf("AddRouterInterface: %w", err)
		}
	} else {
		err := &gophercloud.ErrDefault400{}
		err.Info = "AddRouterInterface: port_id or subnet_id must be specified"
		return nil, err
	}

	routerInterfaces := c.RouterInterfaces[id]
	routerInterfaces = append(routerInterfaces, port.ID)
	c.RouterInterfaces[id] = routerInterfaces

	return &interfaceInfo, nil
}

func (c *NetworkSimulator) ImplRemoveRouterInterface(id string, opts routers.RemoveInterfaceOptsBuilder) (*routers.InterfaceInfo, error) {
	router, err := c.GetRouter(id)
	if err != nil {
		return nil, fmt.Errorf("RemoveRouterInterface: %w", err)
	}

	removeMap, err := opts.ToRouterRemoveInterfaceMap()
	if err != nil {
		return nil, fmt.Errorf("RemoveRouterInterface: creating router interface map: %w", err)
	}

	interfaceInfo := routers.InterfaceInfo{}
	for k, v := range removeMap {
		switch k {
		case "subnet_id":
			interfaceInfo.SubnetID = v.(string)
		case "port_id":
			interfaceInfo.PortID = v.(string)
		default:
			panic(fmt.Errorf("RemoveRouterInterface: simulator doesn't support %s", k))
		}
	}

	portIDs := c.RouterInterfaces[router.ID]
	i, err := func() (int, error) {
		for i, portID := range portIDs {
			port, err := c.GetPort(portID)
			if err != nil && !capoerrors.IsNotFound(err) {
				return -1, fmt.Errorf("RemoveRouterInterface: %w", err)
			}
			if port == nil {
				continue
			}

			matchSubnet := func() bool {
				// Check if any of the port's fixed IPs match the subnet ID
				for _, fixedIP := range port.FixedIPs {
					if fixedIP.SubnetID == interfaceInfo.SubnetID {
						return true
					}
				}
				return false
			}

			if (interfaceInfo.PortID != "" && interfaceInfo.PortID != port.ID) ||
				(interfaceInfo.SubnetID != "" && !matchSubnet()) {
				continue
			}

			return i, nil
		}
		return -1, nil
	}()
	if err != nil {
		return nil, err
	}

	if i == -1 {
		err := &gophercloud.ErrDefault404{}
		err.Info = fmt.Sprintf("RemoveRouterInterface: Router interface %+v for router %s not found", removeMap, id)
		return nil, err
	}

	c.RouterInterfaces[router.ID] = append(portIDs[:i], portIDs[i+1:]...)
	return &interfaceInfo, nil
}

func (c *NetworkSimulator) ImplListSecGroup(opts groups.ListOpts) ([]groups.SecGroup, error) {
	ret := []groups.SecGroup{}
	for _, secGroup := range c.SecGroups {
		if opts.ID != "" && secGroup.ID != opts.ID {
			continue
		}
		if opts.Name != "" && secGroup.Name != opts.Name {
			continue
		}
		if opts.Description != "" && secGroup.Description != opts.Description {
			continue
		}
		if opts.TenantID != "" && secGroup.TenantID != opts.TenantID {
			continue
		}
		if opts.ProjectID != "" && secGroup.ProjectID != opts.ProjectID {
			continue
		}

		unsup := func(f string) {
			panic(fmt.Errorf("ListSecGroup: unsupported query parameter %s", f))
		}
		if opts.Tags != "" {
			unsup("tags")
		}
		if opts.TagsAny != "" {
			unsup("tags-any")
		}
		if opts.NotTags != "" {
			unsup("not-tags")
		}
		if opts.NotTagsAny != "" {
			unsup("not-tags-any")
		}
		if opts.Limit != 0 {
			unsup("limit")
		}
		if opts.Marker != "" {
			unsup("marker")
		}
		if opts.SortKey != "" {
			unsup("sortkey")
		}
		if opts.SortDir != "" {
			unsup("sortdir")
		}

		ret = append(ret, secGroup)
	}

	return ret, nil
}

func (c *NetworkSimulator) ImplCreateSecGroup(opts groups.CreateOptsBuilder) (*groups.SecGroup, error) {
	createMap, err := opts.ToSecGroupCreateMap()
	if err != nil {
		return nil, fmt.Errorf("CreateSecGroup: creating secgroup map: %w", err)
	}
	createMap, ok := createMap["security_group"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("CreateSecGroup: create map does not contain security_group")
	}

	secGroup := groups.SecGroup{}
	secGroup.ID = generateUUID()

	for k, v := range createMap {
		switch k {
		case "description":
			secGroup.Description = v.(string)
		case "name":
			secGroup.Name = v.(string)
		default:
			panic(fmt.Errorf("CreateSecGroup: unsupported create parameter %s", k))
		}
	}

	c.SecGroups = append(c.SecGroups, secGroup)
	return &secGroup, nil
}

func (c *NetworkSimulator) ImplDeleteSecGroup(id string) error {
	for i, secGroup := range c.SecGroups {
		if secGroup.ID == id {
			c.SecGroups = append(c.SecGroups[:i], c.SecGroups[i+1:]...)
			return nil
		}
	}

	return &gophercloud.ErrDefault404{
		ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
			BaseError: gophercloud.BaseError{
				Info: fmt.Sprintf("DeleteSecGroup: secgroup %s not found", id),
			},
		},
	}
}

func (c *NetworkSimulator) ImplGetSecGroup(id string) (*groups.SecGroup, error) {
	for _, secGroup := range c.SecGroups {
		if secGroup.ID == id {
			retCopy := secGroup
			return &retCopy, nil
		}
	}

	return nil, &gophercloud.ErrDefault404{
		ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
			BaseError: gophercloud.BaseError{
				Info: fmt.Sprintf("GetSecGroup: secgroup with id %s does not exist", id),
			},
		},
	}
}

func (c *NetworkSimulator) ImplUpdateSecGroup(id string, opts groups.UpdateOptsBuilder) (*groups.SecGroup, error) {
	panic(fmt.Errorf("UpdateSecGroup not implemented"))
}

func (c *NetworkSimulator) ImplListSecGroupRule(opts rules.ListOpts) ([]rules.SecGroupRule, error) {
	panic(fmt.Errorf("ListSecGroupRule not implemented"))
}

func (c *NetworkSimulator) ImplCreateSecGroupRule(opts rules.CreateOptsBuilder) (*rules.SecGroupRule, error) {
	createMap, err := opts.ToSecGroupRuleCreateMap()
	if err != nil {
		return nil, fmt.Errorf("CreateSecGroupRule: creating secgrouprule map: %w", err)
	}
	createMap, ok := createMap["security_group_rule"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("CreateSecGroupRule: create map does not contain security_group_rule")
	}

	secGroupRule := rules.SecGroupRule{}
	secGroupRule.ID = generateUUID()

	for k, v := range createMap {
		switch k {
		case "description":
			secGroupRule.Description = v.(string)
		case "direction":
			secGroupRule.Direction = v.(string)
		case "ethertype":
			secGroupRule.EtherType = v.(string)
		case "security_group_id":
			secGroupRule.SecGroupID = v.(string)
		case "port_range_min":
			secGroupRule.PortRangeMin = int(v.(float64))
		case "port_range_max":
			secGroupRule.PortRangeMax = int(v.(float64))
		case "protocol":
			secGroupRule.Protocol = v.(string)
		case "remote_group_id":
			secGroupRule.RemoteGroupID = v.(string)
		default:
			panic(fmt.Errorf("CreateSecGroupRule: unsupported create parameter: %s", k))
		}
	}

	_, err = c.GetSecGroup(secGroupRule.SecGroupID)
	if err != nil {
		return nil, fmt.Errorf("CreateSecGroupRule: %w", err)
	}

	c.SecGroupRules = append(c.SecGroupRules, secGroupRule)
	return &secGroupRule, nil
}

func (c *NetworkSimulator) ImplDeleteSecGroupRule(id string) error {
	panic(fmt.Errorf("DeleteSecGroupRule not implemented"))
}

func (c *NetworkSimulator) ImplGetSecGroupRule(id string) (*rules.SecGroupRule, error) {
	panic(fmt.Errorf("GetSecGroupRule not implemented"))
}

func (c *NetworkSimulator) ImplListNetwork(opts networks.ListOptsBuilder) ([]networks.Network, error) {
	query, err := opts.ToNetworkListQuery()
	if err != nil {
		return nil, fmt.Errorf("ListNetwork: creating network query: %w", err)
	}
	values, err := getValuesFromQuery(query)
	if err != nil {
		return nil, fmt.Errorf("ListNetwork: %w", err)
	}

	networks := []networks.Network{}
networks:
	for _, network := range c.Networks {
		for k, v := range values {
			switch k {
			case "id":
				if network.ID != v {
					continue networks
				}
			case "name":
				if network.Name != v {
					continue networks
				}
			case "router:external":
				external := v == "true"
				if network.External != external {
					continue networks
				}
			default:
				panic(fmt.Errorf("ListNetwork: unsupported query param %s", k))
			}
		}

		networks = append(networks, network.Network)
	}

	return networks, nil
}

func (c *NetworkSimulator) ImplCreateNetwork(opts networks.CreateOptsBuilder) (*networks.Network, error) {
	createMap, err := opts.ToNetworkCreateMap()
	if err != nil {
		return nil, fmt.Errorf("CreateNetwork: creating network map: %w", err)
	}
	createMap, ok := createMap["network"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("CreateNetwork: create map does not contain network")
	}

	network := SimNetwork{}
	network.ID = generateUUID()
	network.External = false

	for k, v := range createMap {
		switch k {
		case "admin_state_up":
			network.AdminStateUp = v.(bool)
		case "name":
			network.Name = v.(string)
		default:
			panic(fmt.Errorf("CreateNetwork: unsupported create parameter %s", k))
		}
	}

	c.Networks = append(c.Networks, network)
	return &network.Network, nil
}

func (c *NetworkSimulator) ImplDeleteNetwork(id string) error {
	for i, network := range c.Networks {
		if network.ID == id {
			c.Networks = append(c.Networks[:i], c.Networks[i+1:]...)
			return nil
		}
	}

	return &gophercloud.ErrDefault404{
		ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
			BaseError: gophercloud.BaseError{
				Info: fmt.Sprintf("DeleteNetwork: network with id %s does not exist", id),
			},
		},
	}
}

func (c *NetworkSimulator) ImplGetNetwork(id string) (*networks.Network, error) {
	for _, network := range c.Networks {
		if network.ID == id {
			retCopy := network.Network
			return &retCopy, nil
		}
	}

	return nil, &gophercloud.ErrDefault404{
		ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
			BaseError: gophercloud.BaseError{
				Info: fmt.Sprintf("GetNetwork: network with id %s does not exist", id),
			},
		},
	}
}

func (c *NetworkSimulator) ImplUpdateNetwork(id string, opts networks.UpdateOptsBuilder) (*networks.Network, error) {
	panic(fmt.Errorf("UpdateNetwork not implemented"))
}

func (c *NetworkSimulator) ImplListSubnet(opts subnets.ListOptsBuilder) ([]subnets.Subnet, error) {
	query, err := opts.ToSubnetListQuery()
	if err != nil {
		return nil, fmt.Errorf("ListSubnet: creating subnet query: %w", err)
	}
	values, err := getValuesFromQuery(query)
	if err != nil {
		return nil, fmt.Errorf("ListSubnet: %w", err)
	}

	ret := []subnets.Subnet{}
subnets:
	for _, subnet := range c.Subnets {
		for k, v := range values {
			switch k {
			case "name":
				if subnet.Name != v {
					continue subnets
				}
			case "cidr":
				if subnet.CIDR != v {
					continue subnets
				}
			case "network_id":
				if subnet.NetworkID != v {
					continue subnets
				}
			default:
				panic(fmt.Errorf("ListSubnet: unsupported parameter %s", k))
			}
		}

		ret = append(ret, subnet.Subnet)
	}

	return ret, nil
}

func (c *NetworkSimulator) ImplCreateSubnet(opts subnets.CreateOptsBuilder) (*subnets.Subnet, error) {
	createMap, err := opts.ToSubnetCreateMap()
	if err != nil {
		return nil, fmt.Errorf("CreateSubnet: creating subnet map: %w", err)
	}
	createMap, ok := createMap["subnet"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("CreateSubnet: create map does not contain subnet")
	}

	subnet := SimSubnet{}
	subnet.ID = generateUUID()

	for k, v := range createMap {
		switch k {
		case "cidr":
			subnet.CIDR = v.(string)
			_, ipNet, err := netutils.ParseCIDRSloppy(subnet.CIDR)
			if err != nil {
				return nil, fmt.Errorf("CreateSubnet: parsing cidr: %w", err)
			}
			subnet.SimIPNet = ipNet
		case "description":
			subnet.Description = v.(string)
		case "dns_nameservers":
			var nameservers []string
			for _, v := range v.([]interface{}) {
				nameservers = append(nameservers, v.(string))
			}
			subnet.DNSNameservers = nameservers
		case "ip_version":
			subnet.IPVersion = int(v.(float64))
		case "name":
			subnet.Name = v.(string)
		case "network_id":
			subnet.NetworkID = v.(string)
		default:
			panic(fmt.Errorf("CreateSubnet: unsupported create parameter %s", k))
		}
	}

	_, err = c.GetNetwork(subnet.NetworkID)
	if err != nil {
		return nil, fmt.Errorf("CreateSubnet: %w", err)
	}

	c.Subnets = append(c.Subnets, subnet)
	return &subnet.Subnet, nil
}

func (c *NetworkSimulator) ImplDeleteSubnet(id string) error {
	panic(fmt.Errorf("DeleteSubnet not implemented"))
}

func (c *NetworkSimulator) ImplGetSubnet(id string) (*subnets.Subnet, error) {
	for _, subnet := range c.Subnets {
		if subnet.ID == id {
			retCopy := subnet.Subnet
			return &retCopy, nil
		}
	}

	return nil, &gophercloud.ErrDefault404{
		ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
			BaseError: gophercloud.BaseError{
				Info: fmt.Sprintf("GetSubnet: subnet with id %s does not exist", id),
			},
		},
	}
}

func (c *NetworkSimulator) ImplUpdateSubnet(id string, opts subnets.UpdateOptsBuilder) (*subnets.Subnet, error) {
	panic(fmt.Errorf("UpdateSubnet not implemented"))
}

func (c *NetworkSimulator) ImplListExtensions() ([]extensions.Extension, error) {
	return c.Extensions, nil
}

func (c *NetworkSimulator) ImplReplaceAllAttributesTags(resourceType string, resourceID string, opts attributestags.ReplaceAllOptsBuilder) ([]string, error) {
	tagOpts, err := opts.ToAttributeTagsReplaceAllMap()
	if err != nil {
		return nil, fmt.Errorf("ReplaceAllAttributesTags: creating tags map: %w", err)
	}
	tags, ok := tagOpts["tags"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("ReplaceAllAttributesTags: tags map does not contain tags")
	}
	tagStrings := []string{}
	for _, tag := range tags {
		tagStrings = append(tagStrings, tag.(string))
	}

	tagNetwork := func() ([]string, error) {
		for i := range c.Networks {
			network := &c.Networks[i]
			if network.ID == resourceID {
				oldTags := network.Tags
				network.Tags = tagStrings
				return oldTags, nil
			}
		}

		err := &gophercloud.ErrDefault404{}
		err.Info = fmt.Sprintf("ReplaceAllAttributesTags: network with id %s does not exist", resourceID)
		return nil, err
	}

	tagSubnets := func() ([]string, error) {
		for i := range c.Subnets {
			subnet := &c.Subnets[i]
			if subnet.ID == resourceID {
				oldTags := subnet.Tags
				subnet.Tags = tagStrings
				return oldTags, nil
			}
		}

		err := &gophercloud.ErrDefault404{}
		err.Info = fmt.Sprintf("ReplaceAllAttributesTags: subnet with id %s does not exist", resourceID)
		return nil, err
	}

	tagRouters := func() ([]string, error) {
		for i := range c.Routers {
			router := &c.Routers[i]
			if router.ID == resourceID {
				oldTags := router.Tags
				router.Tags = tagStrings
				return oldTags, nil
			}
		}

		err := &gophercloud.ErrDefault404{}
		err.Info = fmt.Sprintf("ReplaceAllAttributesTags: router with id %s does not exist", resourceID)
		return nil, err
	}

	tagSecurityGroups := func() ([]string, error) {
		for i := range c.SecGroups {
			securityGroup := &c.SecGroups[i]
			if securityGroup.ID == resourceID {
				oldTags := securityGroup.Tags
				securityGroup.Tags = tagStrings
				return oldTags, nil
			}
		}

		err := &gophercloud.ErrDefault404{}
		err.Info = fmt.Sprintf("ReplaceAllAttributesTags: security group with id %s does not exist", resourceID)
		return nil, err
	}

	tagFloatingIPs := func() ([]string, error) {
		for i := range c.FloatingIPs {
			floatingIP := &c.FloatingIPs[i]
			if floatingIP.ID == resourceID {
				oldTags := floatingIP.Tags
				floatingIP.Tags = tagStrings
				return oldTags, nil
			}
		}

		err := &gophercloud.ErrDefault404{}
		err.Info = fmt.Sprintf("ReplaceAllAttributesTags: floating IP with id %s does not exist", resourceID)
		return nil, err
	}

	switch resourceType {
	case "networks":
		return tagNetwork()
	case "subnets":
		return tagSubnets()
	case "routers":
		return tagRouters()
	case "security-groups":
		return tagSecurityGroups()
	case "floatingips":
		return tagFloatingIPs()
	default:
		panic(fmt.Errorf("ReplaceAllAttributesTags: unsupported resource type %s", resourceType))
	}
}

/*
 * Callback handler stubs
 */

func (c *NetworkSimulator) ListFloatingIP(opts floatingips.ListOptsBuilder) ([]floatingips.FloatingIP, error) {
	if c.ListFloatingIPPreHook != nil {
		handled, fips, err := c.ListFloatingIPPreHook(opts)
		if handled {
			return fips, err
		}
	}
	fips, err := c.ImplListFloatingIP(opts)
	if c.ListFloatingIPPostHook != nil {
		c.ListFloatingIPPostHook(opts, fips, err)
	}
	return fips, err
}

func (c *NetworkSimulator) CreateFloatingIP(opts floatingips.CreateOptsBuilder) (*floatingips.FloatingIP, error) {
	if c.CreateFloatingIPPreHook != nil {
		handled, fip, err := c.CreateFloatingIPPreHook(opts)
		if handled {
			return fip, err
		}
	}
	fip, err := c.ImplCreateFloatingIP(opts)
	if c.CreateFloatingIPPostHook != nil {
		c.CreateFloatingIPPostHook(opts, fip, err)
	}
	return fip, err
}

func (c *NetworkSimulator) DeleteFloatingIP(id string) error {
	if c.DeleteFloatingIPPreHook != nil {
		handled, err := c.DeleteFloatingIPPreHook(id)
		if handled {
			return err
		}
	}
	err := c.ImplDeleteFloatingIP(id)
	if c.DeleteFloatingIPPostHook != nil {
		c.DeleteFloatingIPPostHook(id, err)
	}
	return err
}

func (c *NetworkSimulator) GetFloatingIP(id string) (*floatingips.FloatingIP, error) {
	if c.GetFloatingIPPreHook != nil {
		handled, fip, err := c.GetFloatingIPPreHook(id)
		if handled {
			return fip, err
		}
	}
	fip, err := c.ImplGetFloatingIP(id)
	if c.GetFloatingIPPostHook != nil {
		c.GetFloatingIPPostHook(id, fip, err)
	}
	return fip, err
}

func (c *NetworkSimulator) UpdateFloatingIP(id string, opts floatingips.UpdateOptsBuilder) (*floatingips.FloatingIP, error) {
	if c.UpdateFloatingIPPreHook != nil {
		handled, fip, err := c.UpdateFloatingIPPreHook(id, opts)
		if handled {
			return fip, err
		}
	}
	fip, err := c.ImplUpdateFloatingIP(id, opts)
	if c.UpdateFloatingIPPostHook != nil {
		c.UpdateFloatingIPPostHook(id, opts, fip, err)
	}
	return fip, err
}

func (c *NetworkSimulator) ListPort(opts ports.ListOptsBuilder) ([]ports.Port, error) {
	if c.ListPortPreHook != nil {
		handled, ports, err := c.ListPortPreHook(opts)
		if handled {
			return ports, err
		}
	}
	ports, err := c.ImplListPort(opts)
	if c.ListPortPostHook != nil {
		c.ListPortPostHook(opts, ports, err)
	}
	return ports, err
}

func (c *NetworkSimulator) CreatePort(opts ports.CreateOptsBuilder) (*ports.Port, error) {
	if c.CreatePortPreHook != nil {
		handled, port, err := c.CreatePortPreHook(opts)
		if handled {
			return port, err
		}
	}
	port, err := c.ImplCreatePort(opts)
	if c.CreatePortPostHook != nil {
		c.CreatePortPostHook(opts, port, err)
	}
	return port, err
}

func (c *NetworkSimulator) DeletePort(id string) error {
	if c.DeletePortPreHook != nil {
		handled, err := c.DeletePortPreHook(id)
		if handled {
			return err
		}
	}
	err := c.ImplDeletePort(id)
	if c.DeletePortPostHook != nil {
		c.DeletePortPostHook(id, err)
	}
	return err
}

func (c *NetworkSimulator) GetPort(id string) (*ports.Port, error) {
	if c.GetPortPreHook != nil {
		handled, port, err := c.GetPortPreHook(id)
		if handled {
			return port, err
		}
	}
	port, err := c.ImplGetPort(id)
	if c.GetPortPostHook != nil {
		c.GetPortPostHook(id, port, err)
	}
	return port, err
}

func (c *NetworkSimulator) UpdatePort(id string, opts ports.UpdateOptsBuilder) (*ports.Port, error) {
	if c.UpdatePortPreHook != nil {
		handled, port, err := c.UpdatePortPreHook(id, opts)
		if handled {
			return port, err
		}
	}
	port, err := c.ImplUpdatePort(id, opts)
	if c.UpdatePortPostHook != nil {
		c.UpdatePortPostHook(id, opts, port, err)
	}
	return port, err
}

func (c *NetworkSimulator) ListTrunk(opts trunks.ListOptsBuilder) ([]trunks.Trunk, error) {
	if c.ListTrunkPreHook != nil {
		handled, trunks, err := c.ListTrunkPreHook(opts)
		if handled {
			return trunks, err
		}
	}
	trunks, err := c.ImplListTrunk(opts)
	if c.ListTrunkPostHook != nil {
		c.ListTrunkPostHook(opts, trunks, err)
	}
	return trunks, err
}

func (c *NetworkSimulator) CreateTrunk(opts trunks.CreateOptsBuilder) (*trunks.Trunk, error) {
	if c.CreateTrunkPreHook != nil {
		handled, trunk, err := c.CreateTrunkPreHook(opts)
		if handled {
			return trunk, err
		}
	}
	trunk, err := c.ImplCreateTrunk(opts)
	if c.CreateTrunkPostHook != nil {
		c.CreateTrunkPostHook(opts, trunk, err)
	}
	return trunk, err
}

func (c *NetworkSimulator) DeleteTrunk(id string) error {
	if c.DeleteTrunkPreHook != nil {
		handled, err := c.DeleteTrunkPreHook(id)
		if handled {
			return err
		}
	}
	err := c.ImplDeleteTrunk(id)
	if c.DeleteTrunkPostHook != nil {
		c.DeleteTrunkPostHook(id, err)
	}
	return err
}

func (c *NetworkSimulator) ListRouter(opts routers.ListOpts) ([]routers.Router, error) {
	if c.ListRouterPreHook != nil {
		handled, routers, err := c.ListRouterPreHook(opts)
		if handled {
			return routers, err
		}
	}
	routers, err := c.ImplListRouter(opts)
	if c.ListRouterPostHook != nil {
		c.ListRouterPostHook(opts, routers, err)
	}
	return routers, err
}

func (c *NetworkSimulator) CreateRouter(opts routers.CreateOptsBuilder) (*routers.Router, error) {
	if c.CreateRouterPreHook != nil {
		handled, router, err := c.CreateRouterPreHook(opts)
		if handled {
			return router, err
		}
	}
	router, err := c.ImplCreateRouter(opts)
	if c.CreateRouterPostHook != nil {
		c.CreateRouterPostHook(opts, router, err)
	}
	return router, err
}

func (c *NetworkSimulator) DeleteRouter(id string) error {
	if c.DeleteRouterPreHook != nil {
		handled, err := c.DeleteRouterPreHook(id)
		if handled {
			return err
		}
	}
	err := c.ImplDeleteRouter(id)
	if c.DeleteRouterPostHook != nil {
		c.DeleteRouterPostHook(id, err)
	}
	return err
}

func (c *NetworkSimulator) GetRouter(id string) (*routers.Router, error) {
	if c.GetRouterPreHook != nil {
		handled, router, err := c.GetRouterPreHook(id)
		if handled {
			return router, err
		}
	}
	router, err := c.ImplGetRouter(id)
	if c.GetRouterPostHook != nil {
		c.GetRouterPostHook(id, router, err)
	}
	return router, err
}

func (c *NetworkSimulator) UpdateRouter(id string, opts routers.UpdateOptsBuilder) (*routers.Router, error) {
	if c.UpdateRouterPreHook != nil {
		handled, router, err := c.UpdateRouterPreHook(id, opts)
		if handled {
			return router, err
		}
	}
	router, err := c.ImplUpdateRouter(id, opts)
	if c.UpdateRouterPostHook != nil {
		c.UpdateRouterPostHook(id, opts, router, err)
	}
	return router, err
}

func (c *NetworkSimulator) AddRouterInterface(id string, opts routers.AddInterfaceOptsBuilder) (*routers.InterfaceInfo, error) {
	if c.AddRouterInterfacePreHook != nil {
		handled, info, err := c.AddRouterInterfacePreHook(id, opts)
		if handled {
			return info, err
		}
	}
	info, err := c.ImplAddRouterInterface(id, opts)
	if c.AddRouterInterfacePostHook != nil {
		c.AddRouterInterfacePostHook(id, opts, info, err)
	}
	return info, err
}

func (c *NetworkSimulator) RemoveRouterInterface(id string, opts routers.RemoveInterfaceOptsBuilder) (*routers.InterfaceInfo, error) {
	if c.RemoveRouterInterfacePreHook != nil {
		handled, ifInfo, err := c.RemoveRouterInterfacePreHook(id, opts)
		if handled {
			return ifInfo, err
		}
	}
	ifInfo, err := c.ImplRemoveRouterInterface(id, opts)
	if c.RemoveRouterInterfacePostHook != nil {
		c.RemoveRouterInterfacePostHook(id, opts, ifInfo, err)
	}
	return ifInfo, err
}

func (c *NetworkSimulator) ListSecGroup(opts groups.ListOpts) ([]groups.SecGroup, error) {
	if c.ListSecGroupPreHook != nil {
		handled, groups, err := c.ListSecGroupPreHook(opts)
		if handled {
			return groups, err
		}
	}
	groups, err := c.ImplListSecGroup(opts)
	if c.ListSecGroupPostHook != nil {
		c.ListSecGroupPostHook(opts, groups, err)
	}
	return groups, err
}

func (c *NetworkSimulator) CreateSecGroup(opts groups.CreateOptsBuilder) (*groups.SecGroup, error) {
	if c.CreateSecGroupPreHook != nil {
		handled, group, err := c.CreateSecGroupPreHook(opts)
		if handled {
			return group, err
		}
	}
	group, err := c.ImplCreateSecGroup(opts)
	if c.CreateSecGroupPostHook != nil {
		c.CreateSecGroupPostHook(opts, group, err)
	}
	return group, err
}

func (c *NetworkSimulator) DeleteSecGroup(id string) error {
	if c.DeleteSecGroupPreHook != nil {
		handled, err := c.DeleteSecGroupPreHook(id)
		if handled {
			return err
		}
	}
	err := c.ImplDeleteSecGroup(id)
	if c.DeleteSecGroupPostHook != nil {
		c.DeleteSecGroupPostHook(id, err)
	}
	return err
}

func (c *NetworkSimulator) GetSecGroup(id string) (*groups.SecGroup, error) {
	if c.GetSecGroupPreHook != nil {
		handled, group, err := c.GetSecGroupPreHook(id)
		if handled {
			return group, err
		}
	}
	group, err := c.ImplGetSecGroup(id)
	if c.GetSecGroupPostHook != nil {
		c.GetSecGroupPostHook(id, group, err)
	}
	return group, err
}

func (c *NetworkSimulator) UpdateSecGroup(id string, opts groups.UpdateOptsBuilder) (*groups.SecGroup, error) {
	if c.UpdateSecGroupPreHook != nil {
		handled, group, err := c.UpdateSecGroupPreHook(id, opts)
		if handled {
			return group, err
		}
	}
	group, err := c.ImplUpdateSecGroup(id, opts)
	if c.UpdateSecGroupPostHook != nil {
		c.UpdateSecGroupPostHook(id, opts, group, err)
	}
	return group, err
}

func (c *NetworkSimulator) ListSecGroupRule(opts rules.ListOpts) ([]rules.SecGroupRule, error) {
	if c.ListSecGroupRulePreHook != nil {
		handled, rules, err := c.ListSecGroupRulePreHook(opts)
		if handled {
			return rules, err
		}
	}
	rules, err := c.ImplListSecGroupRule(opts)
	if c.ListSecGroupRulePostHook != nil {
		c.ListSecGroupRulePostHook(opts, rules, err)
	}
	return rules, err
}

func (c *NetworkSimulator) CreateSecGroupRule(opts rules.CreateOptsBuilder) (*rules.SecGroupRule, error) {
	if c.CreateSecGroupRulePreHook != nil {
		handled, rule, err := c.CreateSecGroupRulePreHook(opts)
		if handled {
			return rule, err
		}
	}
	rule, err := c.ImplCreateSecGroupRule(opts)
	if c.CreateSecGroupRulePostHook != nil {
		c.CreateSecGroupRulePostHook(opts, rule, err)
	}
	return rule, err
}

func (c *NetworkSimulator) DeleteSecGroupRule(id string) error {
	if c.DeleteSecGroupRulePreHook != nil {
		handled, err := c.DeleteSecGroupRulePreHook(id)
		if handled {
			return err
		}
	}
	err := c.ImplDeleteSecGroupRule(id)
	if c.DeleteSecGroupRulePostHook != nil {
		c.DeleteSecGroupRulePostHook(id, err)
	}
	return err
}

func (c *NetworkSimulator) GetSecGroupRule(id string) (*rules.SecGroupRule, error) {
	if c.GetSecGroupRulePreHook != nil {
		handled, rule, err := c.GetSecGroupRulePreHook(id)
		if handled {
			return rule, err
		}
	}
	rule, err := c.ImplGetSecGroupRule(id)
	if c.GetSecGroupRulePostHook != nil {
		c.GetSecGroupRulePostHook(id, rule, err)
	}
	return rule, err
}

func (c *NetworkSimulator) ListNetwork(opts networks.ListOptsBuilder) ([]networks.Network, error) {
	if c.ListNetworkPreHook != nil {
		handled, networks, err := c.ListNetworkPreHook(opts)
		if handled {
			return networks, err
		}
	}
	networks, err := c.ImplListNetwork(opts)
	if c.ListNetworkPostHook != nil {
		c.ListNetworkPostHook(opts, networks, err)
	}
	return networks, err
}

func (c *NetworkSimulator) CreateNetwork(opts networks.CreateOptsBuilder) (*networks.Network, error) {
	if c.CreateNetworkPreHook != nil {
		handled, network, err := c.CreateNetworkPreHook(opts)
		if handled {
			return network, err
		}
	}
	network, err := c.ImplCreateNetwork(opts)
	if c.CreateNetworkPostHook != nil {
		c.CreateNetworkPostHook(opts, network, err)
	}
	return network, err
}

func (c *NetworkSimulator) DeleteNetwork(id string) error {
	if c.DeleteNetworkPreHook != nil {
		handled, err := c.DeleteNetworkPreHook(id)
		if handled {
			return err
		}
	}
	err := c.ImplDeleteNetwork(id)
	if c.DeleteNetworkPostHook != nil {
		c.DeleteNetworkPostHook(id, err)
	}
	return err
}

func (c *NetworkSimulator) GetNetwork(id string) (*networks.Network, error) {
	if c.GetNetworkPreHook != nil {
		handled, network, err := c.GetNetworkPreHook(id)
		if handled {
			return network, err
		}
	}
	network, err := c.ImplGetNetwork(id)
	if c.GetNetworkPostHook != nil {
		c.GetNetworkPostHook(id, network, err)
	}
	return network, err
}

func (c *NetworkSimulator) UpdateNetwork(id string, opts networks.UpdateOptsBuilder) (*networks.Network, error) {
	if c.UpdateNetworkPreHook != nil {
		handled, network, err := c.UpdateNetworkPreHook(id, opts)
		if handled {
			return network, err
		}
	}
	network, err := c.ImplUpdateNetwork(id, opts)
	if c.UpdateNetworkPostHook != nil {
		c.UpdateNetworkPostHook(id, opts, network, err)
	}
	return network, err
}

func (c *NetworkSimulator) ListSubnet(opts subnets.ListOptsBuilder) ([]subnets.Subnet, error) {
	if c.ListSubnetPreHook != nil {
		handled, subnets, err := c.ListSubnetPreHook(opts)
		if handled {
			return subnets, err
		}
	}
	subnets, err := c.ImplListSubnet(opts)
	if c.ListSubnetPostHook != nil {
		c.ListSubnetPostHook(opts, subnets, err)
	}
	return subnets, err
}

func (c *NetworkSimulator) CreateSubnet(opts subnets.CreateOptsBuilder) (*subnets.Subnet, error) {
	if c.CreateSubnetPreHook != nil {
		handled, subnet, err := c.CreateSubnetPreHook(opts)
		if handled {
			return subnet, err
		}
	}
	subnet, err := c.ImplCreateSubnet(opts)
	if c.CreateSubnetPostHook != nil {
		c.CreateSubnetPostHook(opts, subnet, err)
	}
	return subnet, err
}

func (c *NetworkSimulator) DeleteSubnet(id string) error {
	if c.DeleteSubnetPreHook != nil {
		handled, err := c.DeleteSubnetPreHook(id)
		if handled {
			return err
		}
	}
	err := c.ImplDeleteSubnet(id)
	if c.DeleteSubnetPostHook != nil {
		c.DeleteSubnetPostHook(id, err)
	}
	return err
}

func (c *NetworkSimulator) GetSubnet(id string) (*subnets.Subnet, error) {
	if c.GetSubnetPreHook != nil {
		handled, subnet, err := c.GetSubnetPreHook(id)
		if handled {
			return subnet, err
		}
	}
	subnet, err := c.ImplGetSubnet(id)
	if c.GetSubnetPostHook != nil {
		c.GetSubnetPostHook(id, subnet, err)
	}
	return subnet, err
}

func (c *NetworkSimulator) UpdateSubnet(id string, opts subnets.UpdateOptsBuilder) (*subnets.Subnet, error) {
	if c.UpdateSubnetPreHook != nil {
		handled, subnet, err := c.UpdateSubnetPreHook(id, opts)
		if handled {
			return subnet, err
		}
	}
	subnet, err := c.ImplUpdateSubnet(id, opts)
	if c.UpdateSubnetPostHook != nil {
		c.UpdateSubnetPostHook(id, opts, subnet, err)
	}
	return subnet, err
}

func (c *NetworkSimulator) ListExtensions() ([]extensions.Extension, error) {
	if c.ListExtensionsPreHook != nil {
		handled, extensions, err := c.ListExtensionsPreHook()
		if handled {
			return extensions, err
		}
	}
	extensions, err := c.ImplListExtensions()
	if c.ListExtensionsPostHook != nil {
		c.ListExtensionsPostHook(extensions, err)
	}
	return extensions, err
}

func (c *NetworkSimulator) ReplaceAllAttributesTags(resourceType string, resourceID string, opts attributestags.ReplaceAllOptsBuilder) ([]string, error) {
	if c.ReplaceAllAttributesTagsPreHook != nil {
		handled, tags, err := c.ReplaceAllAttributesTagsPreHook(resourceType, resourceID, opts)
		if handled {
			return tags, err
		}
	}
	tags, err := c.ImplReplaceAllAttributesTags(resourceType, resourceID, opts)
	if c.ReplaceAllAttributesTagsPostHook != nil {
		c.ReplaceAllAttributesTagsPostHook(resourceType, resourceID, opts, tags, err)
	}
	return tags, err
}

/*
 * Simulator state helpers
 */

func (c *NetworkSimulator) firstSubnetForNetwork(networkID string) *SimSubnet {
	for _, subnet := range c.Subnets {
		if subnet.NetworkID == networkID {
			return &subnet
		}
	}
	return nil
}

func (c *NetworkSimulator) SimAttachPort(portID, deviceID string) {
	for i := range c.Ports {
		port := &c.Ports[i]
		if port.ID == portID {
			port.DeviceID = deviceID
			return
		}
	}
}

func (c *NetworkSimulator) SimAddTrunkSupport() {
	trunkExtension := extensions.Extension{}
	trunkExtension.Alias = "trunk"

	c.Extensions = append(c.Extensions, trunkExtension)
}

func (c *NetworkSimulator) SimAddNetworkWithSubnet(name, uuid string, cidr string, external bool) {
	network := SimNetwork{}
	network.Name = name
	network.ID = uuid
	network.External = external

	subnet := SimSubnet{}
	subnet.Name = fmt.Sprintf("%s-subnet", name)
	subnet.NetworkID = uuid
	subnet.CIDR = cidr
	_, ipNet, err := netutils.ParseCIDRSloppy(subnet.CIDR)
	if err != nil {
		panic(err)
	}
	subnet.SimIPNet = ipNet

	c.Subnets = append(c.Subnets, subnet)
	c.Networks = append(c.Networks, network)
}

func (c *NetworkSimulator) SimNextIPForSubnet(id string) (string, error) {
	for i := range c.Subnets {
		subnet := &c.Subnets[i]
		if subnet.ID == id {
			subnet.SimLastIP++
			ip, err := netutils.GetIndexedIP(subnet.SimIPNet, subnet.SimLastIP)
			if err != nil {
				return "", err
			}
			return ip.String(), nil
		}
	}

	// This is an error in the simulator or the test
	panic("Subnet not found")
}

func (c *NetworkSimulator) SimSetActiveFloatingIP(id string) {
	for i := range c.FloatingIPs {
		fip := &c.FloatingIPs[i]
		if fip.ID == id {
			fip.Status = FloatingIPStatusActive
			return
		}
	}
}

/*
 * Default callbacks
 */

func (c *NetworkSimulator) CallBackFloatingIPSetActiveAfterAssociate(floatingIP string, _ floatingips.UpdateOptsBuilder, _ *floatingips.FloatingIP, err error) {
	if err != nil {
		return
	}

	go func() {
		c.Simulator.DefaultDelay()
		c.SimSetActiveFloatingIP(floatingIP)
	}()
}
