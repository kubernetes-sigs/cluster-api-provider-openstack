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
	"regexp"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/attachinterfaces"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"k8s.io/utils/pointer"

	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients"
)

type (
	ListAvailabilityZonesPreHook    func() (bool, []availabilityzones.AvailabilityZone, error)
	ListAvailabilityZonesPostHook   func([]availabilityzones.AvailabilityZone, error)
	GetFlavorIDFromNamePreHook      func(flavor string) (bool, string, error)
	GetFlavorIDFromNamePostHook     func(string, string, error)
	CreateServerPreHook             func(createOpts servers.CreateOptsBuilder) (bool, *clients.ServerExt, error)
	CreateServerPostHook            func(servers.CreateOptsBuilder, *clients.ServerExt, error)
	DeleteServerPreHook             func(serverID string) (bool, error)
	DeleteServerPostHook            func(string, error)
	GetServerPreHook                func(serverID string) (bool, *clients.ServerExt, error)
	GetServerPostHook               func(string, *clients.ServerExt, error)
	ListServersPreHook              func(listOpts servers.ListOptsBuilder) (bool, []clients.ServerExt, error)
	ListServersPostHook             func(servers.ListOptsBuilder, []clients.ServerExt, error)
	ListAttachedInterfacesPreHook   func(serverID string) (bool, []attachinterfaces.Interface, error)
	ListAttachedInterfacesPostHook  func(string, []attachinterfaces.Interface, error)
	DeleteAttachedInterfacePreHook  func(serverID, portID string) (bool, error)
	DeleteAttachedInterfacePostHook func(string, string, error)
)

// ServerExt stores only values the code needs to retrieve. The simulator also
// needs to store important values we set but never retrieve. SimServer is a
// superset of ServerExt which also holds import set values.
type SimServer struct {
	clients.ServerExt

	ConfigDrive *bool
	UserData    *string
}

type ComputeSimulator struct {
	Simulator *OpenStackSimulator

	Servers           []SimServer
	Flavors           []flavors.Flavor
	AvailabilityZones []string

	ListAvailabilityZonesPreHook    ListAvailabilityZonesPreHook
	ListAvailabilityZonesPostHook   ListAvailabilityZonesPostHook
	GetFlavorIDFromNamePreHook      GetFlavorIDFromNamePreHook
	GetFlavorIDFromNamePostHook     GetFlavorIDFromNamePostHook
	CreateServerPreHook             CreateServerPreHook
	CreateServerPostHook            CreateServerPostHook
	DeleteServerPreHook             DeleteServerPreHook
	DeleteServerPostHook            DeleteServerPostHook
	GetServerPreHook                GetServerPreHook
	GetServerPostHook               GetServerPostHook
	ListServersPreHook              ListServersPreHook
	ListServersPostHook             ListServersPostHook
	ListAttachedInterfacesPreHook   ListAttachedInterfacesPreHook
	ListAttachedInterfacesPostHook  ListAttachedInterfacesPostHook
	DeleteAttachedInterfacePreHook  DeleteAttachedInterfacePreHook
	DeleteAttachedInterfacePostHook DeleteAttachedInterfacePostHook
}

func NewComputeSimulator(p *OpenStackSimulator) *ComputeSimulator {
	c := &ComputeSimulator{Simulator: p}

	c.CreateServerPostHook = c.CallBackCreateServerSetActive

	return c
}

const (
	ServerStatusActive       = "ACTIVE"
	ServerStatusError        = "ERROR"
	ServerStatusDeleted      = "DELETED"
	ServerStatusBuild        = "BUILD"
	ServerStatusRebuild      = "REBUILD"
	ServerStatusResize       = "RESIZE"
	ServerStatusVerifyResize = "VERIFY_RESIZE"
	ServerStatusMigrating    = "MIGRATING"
)

/*
 * Simulator implementation methods
 */

func (c *ComputeSimulator) ImplListAvailabilityZones() ([]availabilityzones.AvailabilityZone, error) {
	azs := []availabilityzones.AvailabilityZone{}
	for _, az := range c.AvailabilityZones {
		azs = append(azs, availabilityzones.AvailabilityZone{
			ZoneName: az,
			ZoneState: availabilityzones.ZoneState{
				Available: true,
			},
		})
	}
	return azs, nil
}

func (c *ComputeSimulator) ImplGetFlavorIDFromName(name string) (string, error) {
	for _, flavor := range c.Flavors {
		if flavor.Name == name {
			return flavor.ID, nil
		}
	}

	return "", &gophercloud.ErrResourceNotFound{Name: name, ResourceType: "flavor"}
}

func (c *ComputeSimulator) ImplCreateServer(createOpts servers.CreateOptsBuilder) (*clients.ServerExt, error) {
	createMap, err := createOpts.ToServerCreateMap()
	if err != nil {
		return nil, fmt.Errorf("CreateServer: creating server map: %w", err)
	}
	createMap, ok := createMap["server"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("CreateServer: create map does not contain server")
	}

	server := SimServer{}
	server.ID = generateUUID()
	server.Status = ServerStatusBuild

	if flavorRef, ok := createMap["flavorRef"]; ok {
		flavor := func() *flavors.Flavor {
			for _, flavor := range c.Flavors {
				if flavor.ID == flavorRef {
					return &flavor
				}
			}
			return nil
		}()
		if flavor == nil {
			// XXX: 400 return is a guess
			return nil, &gophercloud.ErrDefault400{
				ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
					BaseError: gophercloud.BaseError{
						Info: fmt.Sprintf("Unknown flavorRef: %s", flavorRef),
					},
				},
			}
		}

		// XXX: This flavor map is not correct. We will need to fix it if we ever have code using it.
		server.Flavor = make(map[string]interface{})
		server.Flavor["id"] = flavor.ID
		server.Flavor["name"] = flavor.Name

		delete(createMap, "flavorRef")
	} else {
		// XXX: 400 return is a guess
		return nil, &gophercloud.ErrDefault400{
			ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
				BaseError: gophercloud.BaseError{
					Info: "Server create without flavor",
				},
			},
		}
	}

	if imageRef, ok := createMap["imageRef"]; ok {
		image := func() *images.Image {
			for _, image := range c.Simulator.Image.Images {
				if image.ID == imageRef {
					return &image
				}
			}
			return nil
		}()
		if image == nil {
			// XXX: 400 return is a guess
			return nil, &gophercloud.ErrDefault400{
				ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
					BaseError: gophercloud.BaseError{
						Info: fmt.Sprintf("Unknown imageRef: %s", imageRef),
					},
				},
			}
		}

		// XXX: This image map is not correct. We will need to fix it if we ever have code using it.
		server.Image = make(map[string]interface{})
		server.Image["id"] = image.ID
		server.Image["name"] = image.Name

		delete(createMap, "imageRef")
	}

	var ports []*ports.Port
	for k, v := range createMap {
		switch k {
		case "config_drive":
			server.ConfigDrive = pointer.Bool(v.(bool))
		case "name":
			server.Name = v.(string)
		case "networks":
			if createMap[k] == nil {
				break
			}
			networks := createMap[k].([]map[string]interface{})
			for _, networkDef := range networks {
				for k, v := range networkDef {
					if k != "port" {
						panic(fmt.Errorf("CreateServer: simulator only supports networks by port id, given %+v", networkDef))
					}
					portID := v.(string)
					port, err := c.Simulator.Network.GetPort(portID)
					if err != nil {
						return nil, fmt.Errorf("CreateServer: %w", err)
					}
					ports = append(ports, port)
				}
			}
		case "user_data":
			server.UserData = v.(*string)
		case "key_name":
			server.KeyName = v.(string)
		case "availability_zone":
			server.AvailabilityZone = v.(string)
		default:
			panic(fmt.Errorf("CreateServer: simulator does not support %s:%+v", k, v))
		}
	}

	addresses := make(map[string]interface{})
	server.Addresses = addresses
	for _, port := range ports {
		c.Simulator.Network.SimAttachPort(port.ID, server.ID)

		network, err := c.Simulator.Network.GetNetwork(port.NetworkID)
		if err != nil {
			// This should only be a simulator bug
			panic(fmt.Errorf("CreateServer: port with id %s references network with id %s, which doesn't exist", port.ID, port.NetworkID))
		}

		networkAddresses := func() []map[string]interface{} {
			if networkAddresses, ok := addresses[network.Name]; ok {
				return networkAddresses.([]map[string]interface{})
			}
			return nil
		}()

		for _, fixedIP := range port.FixedIPs {
			networkAddresses = append(networkAddresses, map[string]interface{}{
				"addr":            fixedIP.IPAddress,
				"version":         4,
				"OS-EXT-IPS:type": "fixed",
			})
		}
		addresses[network.Name] = networkAddresses
	}

	c.Servers = append(c.Servers, server)
	return &server.ServerExt, nil
}

func (c *ComputeSimulator) ImplDeleteServer(serverID string) error {
	for i := range c.Servers {
		server := &c.Servers[i]
		if server.ID == serverID {
			c.Servers = append(c.Servers[:i], c.Servers[i+1:]...)
			return nil
		}
	}

	err := &gophercloud.ErrDefault404{}
	err.Info = fmt.Sprintf("DeleteServer: Server %s not found", serverID)
	return err
}

func (c *ComputeSimulator) ImplGetServer(serverID string) (*clients.ServerExt, error) {
	for _, server := range c.Servers {
		if server.ID == serverID {
			retCopy := server.ServerExt
			return &retCopy, nil
		}
	}

	return nil, &gophercloud.ErrDefault404{
		ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
			BaseError: gophercloud.BaseError{
				Info: fmt.Sprintf("no server with ID %s", serverID),
			},
		},
	}
}

func (c *ComputeSimulator) ImplListServers(listOpts servers.ListOptsBuilder) ([]clients.ServerExt, error) {
	query, err := listOpts.ToServerListQuery()
	if err != nil {
		return nil, fmt.Errorf("creating server list query: %w", err)
	}
	name, err := getNameFromQuery(query)
	if err != nil {
		return nil, fmt.Errorf("ListServers: %w", err)
	}
	nameRegexp, err := regexp.Compile(name)
	if err != nil {
		return nil, fmt.Errorf("compiling name regexp in ListServers: %w", err)
	}

	ret := []clients.ServerExt{}
	for _, server := range c.Servers {
		if nameRegexp.MatchString(server.Name) {
			ret = append(ret, server.ServerExt)
		}
	}

	return ret, nil
}

func (c *ComputeSimulator) ImplListAttachedInterfaces(serverID string) ([]attachinterfaces.Interface, error) {
	interfaces := []attachinterfaces.Interface{}

	for _, port := range c.Simulator.Network.Ports {
		if port.DeviceID == serverID {
			fixedIPs := []attachinterfaces.FixedIP{}
			for _, fixedIP := range port.FixedIPs {
				fixedIPs = append(fixedIPs, attachinterfaces.FixedIP{
					SubnetID: fixedIP.SubnetID,
				})
			}
			interfaces = append(interfaces, attachinterfaces.Interface{
				PortState: port.Status,
				FixedIPs:  fixedIPs,
				PortID:    port.ID,
				NetID:     port.NetworkID,
				// MACAddr:   "",
			})
		}
	}

	return interfaces, nil
}

func (c *ComputeSimulator) ImplDeleteAttachedInterface(serverID, portID string) error {
	port, err := c.Simulator.Network.GetPort(portID)
	if err != nil {
		return fmt.Errorf("DeleteAttachedInterface: %w", err)
	}

	if port.DeviceID != serverID {
		return fmt.Errorf("DeleteAttachedInterface: port %s is not attached to server %s", portID, serverID)
	}

	return c.Simulator.Network.DeletePort(portID)
}

/*
 * Callback handler stubs
 */

func (c *ComputeSimulator) ListAvailabilityZones() ([]availabilityzones.AvailabilityZone, error) {
	if c.ListAvailabilityZonesPreHook != nil {
		handled, azs, err := c.ListAvailabilityZonesPreHook()
		if handled {
			return azs, err
		}
	}
	azs, err := c.ImplListAvailabilityZones()
	if c.ListAvailabilityZonesPostHook != nil {
		c.ListAvailabilityZonesPostHook(azs, err)
	}
	return azs, err
}

func (c *ComputeSimulator) GetFlavorIDFromName(name string) (string, error) {
	if c.GetFlavorIDFromNamePreHook != nil {
		handled, flavorID, err := c.GetFlavorIDFromNamePreHook(name)
		if handled {
			return flavorID, err
		}
	}
	flavorID, err := c.ImplGetFlavorIDFromName(name)
	if c.GetFlavorIDFromNamePostHook != nil {
		c.GetFlavorIDFromNamePostHook(name, flavorID, err)
	}
	return flavorID, err
}

func (c *ComputeSimulator) CreateServer(createOpts servers.CreateOptsBuilder) (*clients.ServerExt, error) {
	if c.CreateServerPreHook != nil {
		handled, server, err := c.CreateServerPreHook(createOpts)
		if handled {
			return server, err
		}
	}
	server, err := c.ImplCreateServer(createOpts)
	if c.CreateServerPostHook != nil {
		c.CreateServerPostHook(createOpts, server, err)
	}
	return server, err
}

func (c *ComputeSimulator) DeleteServer(serverID string) error {
	if c.DeleteServerPreHook != nil {
		handled, err := c.DeleteServerPreHook(serverID)
		if handled {
			return err
		}
	}
	err := c.ImplDeleteServer(serverID)
	if c.DeleteServerPostHook != nil {
		c.DeleteServerPostHook(serverID, err)
	}
	return err
}

func (c *ComputeSimulator) GetServer(serverID string) (*clients.ServerExt, error) {
	if c.GetServerPreHook != nil {
		handled, server, err := c.GetServerPreHook(serverID)
		if handled {
			return server, err
		}
	}
	server, err := c.ImplGetServer(serverID)
	if c.GetServerPostHook != nil {
		c.GetServerPostHook(serverID, server, err)
	}
	return server, err
}

func (c *ComputeSimulator) ListServers(listOpts servers.ListOptsBuilder) ([]clients.ServerExt, error) {
	if c.ListServersPreHook != nil {
		handled, servers, err := c.ListServersPreHook(listOpts)
		if handled {
			return servers, err
		}
	}
	servers, err := c.ImplListServers(listOpts)
	if c.ListServersPostHook != nil {
		c.ListServersPostHook(listOpts, servers, err)
	}
	return servers, err
}

func (c *ComputeSimulator) ListAttachedInterfaces(serverID string) ([]attachinterfaces.Interface, error) {
	if c.ListAttachedInterfacesPreHook != nil {
		handled, interfaces, err := c.ListAttachedInterfacesPreHook(serverID)
		if handled {
			return interfaces, err
		}
	}
	interfaces, err := c.ImplListAttachedInterfaces(serverID)
	if c.ListAttachedInterfacesPostHook != nil {
		c.ListAttachedInterfacesPostHook(serverID, interfaces, err)
	}
	return interfaces, err
}

func (c *ComputeSimulator) DeleteAttachedInterface(serverID, portID string) error {
	if c.DeleteAttachedInterfacePreHook != nil {
		handled, err := c.DeleteAttachedInterfacePreHook(serverID, portID)
		if handled {
			return err
		}
	}
	err := c.ImplDeleteAttachedInterface(serverID, portID)
	if c.DeleteAttachedInterfacePostHook != nil {
		c.DeleteAttachedInterfacePostHook(serverID, portID, err)
	}
	return err
}

/*
 * Simulator state helpers
 */

func (c *ComputeSimulator) SimAddFlavor(name, uuid string) {
	c.Flavors = append(c.Flavors, flavors.Flavor{
		Name: name,
		ID:   uuid,
	})
}

func (c *ComputeSimulator) SimAddAvailabilityZone(name string) {
	c.AvailabilityZones = append(c.AvailabilityZones, name)
}

func (c *ComputeSimulator) SimGetServer(id string) *SimServer {
	for _, server := range c.Servers {
		if server.ID == id {
			return &server
		}
	}
	return nil
}

/*
 * Default callbacks
 */

func (c *ComputeSimulator) CallBackCreateServerSetActive(_ servers.CreateOptsBuilder, createdServer *clients.ServerExt, err error) {
	if err != nil || createdServer.Status != ServerStatusBuild {
		return
	}

	go func() {
		c.Simulator.DefaultDelay()

		for i := range c.Servers {
			server := &c.Servers[i]
			if server.ID == createdServer.ID {
				if server.Status == ServerStatusBuild {
					server.Status = ServerStatusActive
				}
				return
			}
		}
	}()
}
