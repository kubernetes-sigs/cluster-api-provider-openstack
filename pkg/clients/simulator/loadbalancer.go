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

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/apiversions"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/listeners"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/monitors"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/pools"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/providers"
)

type (
	CreateLoadBalancerPreHook         func(loadbalancers.CreateOptsBuilder) (bool, *loadbalancers.LoadBalancer, error)
	CreateLoadBalancerPostHook        func(loadbalancers.CreateOptsBuilder, *loadbalancers.LoadBalancer, error)
	ListLoadBalancersPreHook          func(opts loadbalancers.ListOptsBuilder) (bool, []loadbalancers.LoadBalancer, error)
	ListLoadBalancersPostHook         func(loadbalancers.ListOptsBuilder, []loadbalancers.LoadBalancer, error)
	GetLoadBalancerPreHook            func(id string) (bool, *loadbalancers.LoadBalancer, error)
	GetLoadBalancerPostHook           func(string, *loadbalancers.LoadBalancer, error)
	DeleteLoadBalancerPreHook         func(id string, opts loadbalancers.DeleteOptsBuilder) (bool, error)
	DeleteLoadBalancerPostHook        func(string, loadbalancers.DeleteOptsBuilder, error)
	CreateListenerPreHook             func(opts listeners.CreateOptsBuilder) (bool, *listeners.Listener, error)
	CreateListenerPostHook            func(listeners.CreateOptsBuilder, *listeners.Listener, error)
	ListListenersPreHook              func(opts listeners.ListOptsBuilder) (bool, []listeners.Listener, error)
	ListListenersPostHook             func(listeners.ListOptsBuilder, []listeners.Listener, error)
	UpdateListenerPreHook             func(id string, opts listeners.UpdateOpts) (bool, *listeners.Listener, error)
	UpdateListenerPostHook            func(string, listeners.UpdateOpts, *listeners.Listener, error)
	GetListenerPreHook                func(id string) (bool, *listeners.Listener, error)
	GetListenerPostHook               func(string, *listeners.Listener, error)
	DeleteListenerPreHook             func(id string) (bool, error)
	DeleteListenerPostHook            func(string, error)
	CreatePoolPreHook                 func(opts pools.CreateOptsBuilder) (bool, *pools.Pool, error)
	CreatePoolPostHook                func(pools.CreateOptsBuilder, *pools.Pool, error)
	ListPoolsPreHook                  func(opts pools.ListOptsBuilder) (bool, []pools.Pool, error)
	ListPoolsPostHook                 func(pools.ListOptsBuilder, []pools.Pool, error)
	GetPoolPreHook                    func(id string) (bool, *pools.Pool, error)
	GetPoolPostHook                   func(string, *pools.Pool, error)
	DeletePoolPreHook                 func(id string) (bool, error)
	DeletePoolPostHook                func(string, error)
	CreatePoolMemberPreHook           func(poolID string, opts pools.CreateMemberOptsBuilder) (bool, *pools.Member, error)
	CreatePoolMemberPostHook          func(string, pools.CreateMemberOptsBuilder, *pools.Member, error)
	ListPoolMemberPreHook             func(poolID string, opts pools.ListMembersOptsBuilder) (bool, []pools.Member, error)
	ListPoolMemberPostHook            func(string, pools.ListMembersOptsBuilder, []pools.Member, error)
	DeletePoolMemberPreHook           func(poolID string, lbMemberID string) (bool, error)
	DeletePoolMemberPostHook          func(string, string, error)
	CreateMonitorPreHook              func(opts monitors.CreateOptsBuilder) (bool, *monitors.Monitor, error)
	CreateMonitorPostHook             func(monitors.CreateOptsBuilder, *monitors.Monitor, error)
	ListMonitorsPreHook               func(opts monitors.ListOptsBuilder) (bool, []monitors.Monitor, error)
	ListMonitorsPostHook              func(monitors.ListOptsBuilder, []monitors.Monitor, error)
	DeleteMonitorPreHook              func(id string) (bool, error)
	DeleteMonitorPostHook             func(string, error)
	ListLoadBalancerProvidersPreHook  func() (bool, []providers.Provider, error)
	ListLoadBalancerProvidersPostHook func([]providers.Provider, error)
	ListOctaviaVersionsPreHook        func() (bool, []apiversions.APIVersion, error)
	ListOctaviaVersionsPostHook       func([]apiversions.APIVersion, error)
)

type LbSimulator struct {
	Simulator *OpenStackSimulator

	Listeners             []listeners.Listener
	LoadBalancers         []loadbalancers.LoadBalancer
	Monitors              []monitors.Monitor
	Pools                 []pools.Pool
	Providers             []providers.Provider
	OctaviaCurrentVersion string

	CreateLoadBalancerPreHook         CreateLoadBalancerPreHook
	CreateLoadBalancerPostHook        CreateLoadBalancerPostHook
	ListLoadBalancersPreHook          ListLoadBalancersPreHook
	ListLoadBalancersPostHook         ListLoadBalancersPostHook
	GetLoadBalancerPreHook            GetLoadBalancerPreHook
	GetLoadBalancerPostHook           GetLoadBalancerPostHook
	DeleteLoadBalancerPreHook         DeleteLoadBalancerPreHook
	DeleteLoadBalancerPostHook        DeleteLoadBalancerPostHook
	CreateListenerPreHook             CreateListenerPreHook
	CreateListenerPostHook            CreateListenerPostHook
	ListListenersPreHook              ListListenersPreHook
	ListListenersPostHook             ListListenersPostHook
	UpdateListenerPreHook             UpdateListenerPreHook
	UpdateListenerPostHook            UpdateListenerPostHook
	GetListenerPreHook                GetListenerPreHook
	GetListenerPostHook               GetListenerPostHook
	DeleteListenerPreHook             DeleteListenerPreHook
	DeleteListenerPostHook            DeleteListenerPostHook
	CreatePoolPreHook                 CreatePoolPreHook
	CreatePoolPostHook                CreatePoolPostHook
	ListPoolsPreHook                  ListPoolsPreHook
	ListPoolsPostHook                 ListPoolsPostHook
	GetPoolPreHook                    GetPoolPreHook
	GetPoolPostHook                   GetPoolPostHook
	DeletePoolPreHook                 DeletePoolPreHook
	DeletePoolPostHook                DeletePoolPostHook
	CreatePoolMemberPreHook           CreatePoolMemberPreHook
	CreatePoolMemberPostHook          CreatePoolMemberPostHook
	ListPoolMemberPreHook             ListPoolMemberPreHook
	ListPoolMemberPostHook            ListPoolMemberPostHook
	DeletePoolMemberPreHook           DeletePoolMemberPreHook
	DeletePoolMemberPostHook          DeletePoolMemberPostHook
	CreateMonitorPreHook              CreateMonitorPreHook
	CreateMonitorPostHook             CreateMonitorPostHook
	ListMonitorsPreHook               ListMonitorsPreHook
	ListMonitorsPostHook              ListMonitorsPostHook
	DeleteMonitorPreHook              DeleteMonitorPreHook
	DeleteMonitorPostHook             DeleteMonitorPostHook
	ListLoadBalancerProvidersPreHook  ListLoadBalancerProvidersPreHook
	ListLoadBalancerProvidersPostHook ListLoadBalancerProvidersPostHook
	ListOctaviaVersionsPreHook        ListOctaviaVersionsPreHook
	ListOctaviaVersionsPostHook       ListOctaviaVersionsPostHook
}

const (
	OperatingStatusOnline           = "ONLINE"
	OperatingStatusDraining         = "DRAINING"
	OperatingStatusOffline          = "OFFLINE"
	OperatingStatusDegraded         = "DEGRADED"
	OperatingStatusError            = "ERROR"
	OperatingStatusNoMonitor        = "NO_MONITOR"
	ProvisioningStatusActive        = "ACTIVE"
	ProvisioningStatusDeleted       = "DELETED"
	ProvisioningStatusError         = "ERROR"
	ProvisioningStatusPendingCreate = "PENDING_CREATE"
	ProvisioningStatusPendingUpdate = "PENDING_UPDATE"
	ProvisioningStatusPendingDelete = "PENDING_DELETE"
)

func NewLbSimulator(p *OpenStackSimulator) *LbSimulator {
	s := LbSimulator{Simulator: p}

	s.SimAddLoadBalancerProvider("amphora")
	s.SimSetOctaviaCurrentVersion("v2.14")
	s.CreateLoadBalancerPostHook = s.CallBackCreateLoadbalancerSetActive
	s.CreateListenerPostHook = s.CallBackCreateListenerSetActive

	return &s
}

/*
 * Simulator implementation methods
 */

func (c *LbSimulator) ImplCreateLoadBalancer(opts loadbalancers.CreateOptsBuilder) (*loadbalancers.LoadBalancer, error) {
	createMap, err := opts.ToLoadBalancerCreateMap()
	if err != nil {
		return nil, fmt.Errorf("CreateLoadBalancer: creating loadbalancer map: %w", err)
	}
	createMap, ok := createMap["loadbalancer"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("CreateLoadBalancer: create map doesn't contain loadbalancer")
	}

	loadBalancer := loadbalancers.LoadBalancer{}
	loadBalancer.ID = generateUUID()
	loadBalancer.ProvisioningStatus = ProvisioningStatusPendingCreate

	for k, v := range createMap {
		switch k {
		case "description":
			loadBalancer.Description = v.(string)
		case "name":
			loadBalancer.Name = v.(string)
		case "provider":
			loadBalancer.Provider = v.(string)
		case "vip_subnet_id":
			subnet, err := c.Simulator.Network.GetSubnet(v.(string))
			if err != nil {
				return nil, fmt.Errorf("CreateLoadBalancer: getting VIP subnet: %w", err)
			}

			loadBalancer.VipSubnetID = subnet.ID
		default:
			panic(fmt.Errorf("CreateLoadBalancer: unsupported create parameter %s", k))
		}
	}

	c.LoadBalancers = append(c.LoadBalancers, loadBalancer)
	return &loadBalancer, nil
}

func (c *LbSimulator) ImplListLoadBalancers(opts loadbalancers.ListOptsBuilder) ([]loadbalancers.LoadBalancer, error) {
	query, err := opts.ToLoadBalancerListQuery()
	if err != nil {
		return nil, fmt.Errorf("ListLoadBalancers: creating loadbalancer query: %w", err)
	}
	values, err := getValuesFromQuery(query)
	if err != nil {
		return nil, fmt.Errorf("ListLoadBalancers: %w", err)
	}

	ret := []loadbalancers.LoadBalancer{}
loadbalancers:
	for _, lb := range c.LoadBalancers {
		for k, v := range values {
			switch k {
			case "name":
				if lb.Name != v {
					continue loadbalancers
				}
			default:
				panic(fmt.Errorf("ListLoadBalancers: unsupported query parameter %s", k))
			}
		}

		ret = append(ret, lb)
	}

	return ret, nil
}

func (c *LbSimulator) ImplGetLoadBalancer(id string) (*loadbalancers.LoadBalancer, error) {
	for _, lb := range c.LoadBalancers {
		if lb.ID == id {
			retCopy := lb
			return &retCopy, nil
		}
	}

	return nil, &gophercloud.ErrDefault404{
		ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
			BaseError: gophercloud.BaseError{
				Info: fmt.Sprintf("GetLoadBalancer: Loadbalancer %s not found", id),
			},
		},
	}
}

func (c *LbSimulator) ImplDeleteLoadBalancer(id string, opts loadbalancers.DeleteOptsBuilder) error {
	query, err := opts.ToLoadBalancerDeleteQuery()
	if err != nil {
		return fmt.Errorf("DeleteLoadBalancer: creating loadbalancer delete opts: %w", err)
	}
	values, err := getValuesFromQuery(query)
	if err != nil {
		return fmt.Errorf("DeleteLoadBalancer: %w", err)
	}

	getListeners := func() []listeners.Listener {
		var listeners []listeners.Listener
		for _, listener := range c.Listeners {
			for _, lb := range listener.Loadbalancers {
				if lb.ID == id {
					listeners = append(listeners, listener)
				}
			}
		}
		return listeners
	}

	getPools := func() []pools.Pool {
		var pools []pools.Pool
		for _, pool := range c.Pools {
			for _, lb := range pool.Loadbalancers {
				if lb.ID == id {
					pools = append(pools, pool)
				}
			}
		}
		return pools
	}

	// XXX: Does cascade delete anything else? Monitors?
	// Are there any cases where we wouldn't delete resources because they are shared?
	cascade := func() error {
		for _, listener := range getListeners() {
			err := c.DeleteListener(listener.ID)
			if err != nil {
				return fmt.Errorf("DeleteLoadBalancer: deleting listener %s: %w", listener.ID, err)
			}
		}

		for _, pool := range getPools() {
			err := c.DeletePool(pool.ID)
			if err != nil {
				return fmt.Errorf("DeleteLoadBalancer: deleting pool %s: %w", pool.ID, err)
			}
		}

		return nil
	}

	for k, v := range values {
		switch k {
		case "cascade":
			if v == "true" {
				err = cascade()
				if err != nil {
					return fmt.Errorf("DeleteLoadBalancer: %w", err)
				}
			}
		default:
			panic(fmt.Errorf("DeleteLoadBalancer: unsupported query parameter %s", k))
		}
	}

	if len(getPools()) > 0 || len(getListeners()) > 0 {
		return &gophercloud.ErrDefault409{
			ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
				BaseError: gophercloud.BaseError{
					Info: fmt.Sprintf("DeleteLoadBalancer: Loadbalancer %s has associated resources", id),
				},
			},
		}
	}

	for i, lb := range c.LoadBalancers {
		if lb.ID == id {
			c.LoadBalancers = append(c.LoadBalancers[:i], c.LoadBalancers[i+1:]...)
			return nil
		}
	}

	return &gophercloud.ErrDefault404{
		ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
			BaseError: gophercloud.BaseError{
				Info: fmt.Sprintf("DeleteLoadBalancer: Loadbalancer %s not found", id),
			},
		},
	}
}

func (c *LbSimulator) ImplCreateListener(opts listeners.CreateOptsBuilder) (*listeners.Listener, error) {
	createMap, err := opts.ToListenerCreateMap()
	if err != nil {
		return nil, fmt.Errorf("CreateListener: creating listener map: %w", err)
	}
	createMap, ok := createMap["listener"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("CreateListener: create map doesn't contain listener")
	}

	listener := listeners.Listener{}
	listener.ID = generateUUID()
	listener.ProvisioningStatus = ProvisioningStatusPendingCreate

	for k, v := range createMap {
		switch k {
		case "loadbalancer_id":
			lb, err := c.GetLoadBalancer(v.(string))
			if err != nil {
				return nil, fmt.Errorf("CreateListener: %w", err)
			}
			// Not sure why this is a list
			listener.Loadbalancers = []listeners.LoadBalancerID{{ID: lb.ID}}
		case "name":
			listener.Name = v.(string)
		case "protocol":
			listener.Protocol = v.(string)
		case "protocol_port":
			listener.ProtocolPort = int(v.(float64))
		default:
			panic(fmt.Errorf("CreateListener: unsupported create parameter %s", k))
		}
	}

	c.Listeners = append(c.Listeners, listener)
	return &listener, nil
}

func (c *LbSimulator) ImplListListeners(opts listeners.ListOptsBuilder) ([]listeners.Listener, error) {
	query, err := opts.ToListenerListQuery()
	if err != nil {
		return nil, fmt.Errorf("ListListeners: creating listener query: %w", err)
	}
	values, err := getValuesFromQuery(query)
	if err != nil {
		return nil, fmt.Errorf("ListListeners: %w", err)
	}

	listeners := []listeners.Listener{}
listeners:
	for _, listener := range c.Listeners {
		for k, v := range values {
			switch k {
			case "name":
				if listener.Name != v {
					continue listeners
				}
			default:
				panic(fmt.Errorf("ListListeners: unsupported query parameter %s", k))
			}
		}
		listeners = append(listeners, listener)
	}

	return listeners, nil
}

func (c *LbSimulator) ImplUpdateListener(id string, opts listeners.UpdateOpts) (*listeners.Listener, error) {
	panic(fmt.Errorf("UpdateListener not implemented"))
}

func (c *LbSimulator) ImplGetListener(id string) (*listeners.Listener, error) {
	for _, listener := range c.Listeners {
		if listener.ID == id {
			retCopy := listener
			return &retCopy, nil
		}
	}

	return nil, &gophercloud.ErrDefault404{
		ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
			BaseError: gophercloud.BaseError{
				Info: fmt.Sprintf("GetListener: Listener %s not found", id),
			},
		},
	}
}

func (c *LbSimulator) ImplDeleteListener(id string) error {
	for i, listener := range c.Listeners {
		if listener.ID == id {
			c.Listeners = append(c.Listeners[:i], c.Listeners[i+1:]...)
			return nil
		}
	}

	return &gophercloud.ErrDefault404{
		ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
			BaseError: gophercloud.BaseError{
				Info: fmt.Sprintf("DeleteListener: Listener %s not found", id),
			},
		},
	}
}

func (c *LbSimulator) ImplCreatePool(opts pools.CreateOptsBuilder) (*pools.Pool, error) {
	createMap, err := opts.ToPoolCreateMap()
	if err != nil {
		return nil, fmt.Errorf("CreatePool: creating pool map: %w", err)
	}
	createMap, ok := createMap["pool"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("CreatePool: create map doesn't contain pool")
	}

	pool := pools.Pool{}
	pool.ID = generateUUID()
	pool.ProvisioningStatus = ProvisioningStatusPendingCreate

	for k, v := range createMap {
		switch k {
		case "name":
			pool.Name = v.(string)
		case "protocol":
			pool.Protocol = v.(string)
		case "lb_algorithm":
			pool.LBMethod = v.(string)
		case "listener_id":
			listener, err := c.GetListener(v.(string))
			if err != nil {
				return nil, fmt.Errorf("CreatePool: %w", err)
			}
			pool.Listeners = []pools.ListenerID{{ID: listener.ID}}
		default:
			panic(fmt.Errorf("CreatePool: unsupported create parameter %s", k))
		}
	}

	c.Pools = append(c.Pools, pool)
	return &pool, nil
}

func (c *LbSimulator) ImplListPools(opts pools.ListOptsBuilder) ([]pools.Pool, error) {
	query, err := opts.ToPoolListQuery()
	if err != nil {
		return nil, fmt.Errorf("ListPools: creating pool query: %w", err)
	}
	values, err := getValuesFromQuery(query)
	if err != nil {
		return nil, fmt.Errorf("ListPools: %w", err)
	}

	pools := []pools.Pool{}
pools:
	for _, pool := range c.Pools {
		for k, v := range values {
			switch k {
			case "name":
				if pool.Name != v {
					continue pools
				}
			default:
				panic(fmt.Errorf("ListPools: unsupported query parameter %s", k))
			}
		}
		pools = append(pools, pool)
	}

	return pools, nil
}

func (c *LbSimulator) ImplGetPool(id string) (*pools.Pool, error) {
	for _, pool := range c.Pools {
		if pool.ID == id {
			retCopy := pool
			return &retCopy, nil
		}
	}

	return nil, &gophercloud.ErrDefault404{
		ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
			BaseError: gophercloud.BaseError{
				Info: fmt.Sprintf("GetPool: Pool %s not found", id),
			},
		},
	}
}

func (c *LbSimulator) ImplDeletePool(id string) error {
	for i, pool := range c.Pools {
		if pool.ID == id {
			c.Pools = append(c.Pools[:i], c.Pools[i+1:]...)
			return nil
		}
	}

	return &gophercloud.ErrDefault404{
		ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
			BaseError: gophercloud.BaseError{
				Info: fmt.Sprintf("DeletePool: Pool %s not found", id),
			},
		},
	}
}

func (c *LbSimulator) ImplCreatePoolMember(poolID string, opts pools.CreateMemberOptsBuilder) (*pools.Member, error) {
	panic(fmt.Errorf("CreatePoolMember not implemented"))
}

func (c *LbSimulator) ImplListPoolMember(poolID string, opts pools.ListMembersOptsBuilder) ([]pools.Member, error) {
	panic(fmt.Errorf("ListPoolMember not implemented"))
}

func (c *LbSimulator) ImplDeletePoolMember(poolID string, lbMemberID string) error {
	panic(fmt.Errorf("DeletePoolMember not implemented"))
}

func (c *LbSimulator) ImplCreateMonitor(opts monitors.CreateOptsBuilder) (*monitors.Monitor, error) {
	createMap, err := opts.ToMonitorCreateMap()
	if err != nil {
		return nil, fmt.Errorf("CreateMonitor: creating monitor map: %w", err)
	}
	createMap, ok := createMap["healthmonitor"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("CreateMonitor: create map doesn't contain healthmonitor")
	}

	monitor := monitors.Monitor{}
	monitor.ID = generateUUID()
	monitor.ProvisioningStatus = ProvisioningStatusPendingCreate

	for k, v := range createMap {
		switch k {
		case "name":
			monitor.Name = v.(string)
		case "type":
			monitor.Type = v.(string)
		case "delay":
			monitor.Delay = int(v.(float64))
		case "timeout":
			monitor.Timeout = int(v.(float64))
		case "max_retries":
			monitor.MaxRetries = int(v.(float64))
		case "max_retries_down":
			monitor.MaxRetriesDown = int(v.(float64))
		case "http_method":
			monitor.HTTPMethod = v.(string)
		case "url_path":
			monitor.URLPath = v.(string)
		case "expected_codes":
			monitor.ExpectedCodes = v.(string)
		case "pool_id":
			pool, err := c.GetPool(v.(string))
			if err != nil {
				return nil, fmt.Errorf("CreateMonitor: %w", err)
			}
			monitor.Pools = []monitors.PoolID{{ID: pool.ID}}
		default:
			panic(fmt.Errorf("CreateMonitor: unsupported create parameter %s", k))
		}
	}

	c.Monitors = append(c.Monitors, monitor)
	return &monitor, nil
}

func (c *LbSimulator) ImplListMonitors(opts monitors.ListOptsBuilder) ([]monitors.Monitor, error) {
	query, err := opts.ToMonitorListQuery()
	if err != nil {
		return nil, fmt.Errorf("ListMonitors: creating monitor query: %w", err)
	}
	values, err := getValuesFromQuery(query)
	if err != nil {
		return nil, fmt.Errorf("ListMonitors: %w", err)
	}

	monitors := []monitors.Monitor{}
monitors:
	for _, monitor := range c.Monitors {
		for k, v := range values {
			switch k {
			case "name":
				if monitor.Name != v {
					continue monitors
				}
			default:
				panic(fmt.Errorf("ListMonitors: unsupported query parameter %s", k))
			}
		}
		monitors = append(monitors, monitor)
	}

	return monitors, nil
}

func (c *LbSimulator) ImplDeleteMonitor(id string) error {
	panic(fmt.Errorf("DeleteMonitor not implemented"))
}

func (c *LbSimulator) ImplListLoadBalancerProviders() ([]providers.Provider, error) {
	providers := []providers.Provider{}
	providers = append(providers, c.Providers...)

	return providers, nil
}

func (c *LbSimulator) ImplListOctaviaVersions() ([]apiversions.APIVersion, error) {
	// This list can be obtained with the following command:
	// curl -s (openstack catalog show octavia -f json | jq -re '.endpoints | map(select(.interface=="public")) | .[0].url')
	versions := []string{
		"v2.0",
		"v2.1",
		"v2.2",
		"v2.3",
		"v2.4",
		"v2.5",
		"v2.6",
		"v2.7",
		"v2.8",
		"v2.9",
		"v2.10",
		"v2.11",
		"v2.12",
		"v2.13",
		"v2.14",
	}

	foundCurrent := false
	ret := []apiversions.APIVersion{}
	for _, v := range versions {
		apiversion := apiversions.APIVersion{ID: v}
		if v == c.OctaviaCurrentVersion {
			apiversion.Status = "CURRENT"
			foundCurrent = true
			break
		} else {
			apiversion.Status = "SUPPORTED"
		}
		ret = append(ret, apiversion)
	}

	if !foundCurrent {
		panic(fmt.Errorf("ListOctaviaVersions: current version %s not found in list of supported versions", c.OctaviaCurrentVersion))
	}

	return ret, nil
}

/*
 * Callback handler stubs
 */

func (c *LbSimulator) CreateLoadBalancer(opts loadbalancers.CreateOptsBuilder) (*loadbalancers.LoadBalancer, error) {
	if c.CreateLoadBalancerPreHook != nil {
		handled, lb, err := c.CreateLoadBalancerPreHook(opts)
		if handled {
			return lb, err
		}
	}
	lb, err := c.ImplCreateLoadBalancer(opts)
	if c.CreateLoadBalancerPostHook != nil {
		c.CreateLoadBalancerPostHook(opts, lb, err)
	}
	return lb, err
}

func (c *LbSimulator) ListLoadBalancers(opts loadbalancers.ListOptsBuilder) ([]loadbalancers.LoadBalancer, error) {
	if c.ListLoadBalancersPreHook != nil {
		handled, lbs, err := c.ListLoadBalancersPreHook(opts)
		if handled {
			return lbs, err
		}
	}
	lbs, err := c.ImplListLoadBalancers(opts)
	if c.ListLoadBalancersPostHook != nil {
		c.ListLoadBalancersPostHook(opts, lbs, err)
	}
	return lbs, err
}

func (c *LbSimulator) GetLoadBalancer(id string) (*loadbalancers.LoadBalancer, error) {
	if c.GetLoadBalancerPreHook != nil {
		handled, lb, err := c.GetLoadBalancerPreHook(id)
		if handled {
			return lb, err
		}
	}
	lb, err := c.ImplGetLoadBalancer(id)
	if c.GetLoadBalancerPostHook != nil {
		c.GetLoadBalancerPostHook(id, lb, err)
	}
	return lb, err
}

func (c *LbSimulator) DeleteLoadBalancer(id string, opts loadbalancers.DeleteOptsBuilder) error {
	if c.DeleteLoadBalancerPreHook != nil {
		handled, err := c.DeleteLoadBalancerPreHook(id, opts)
		if handled {
			return err
		}
	}
	err := c.ImplDeleteLoadBalancer(id, opts)
	if c.DeleteLoadBalancerPostHook != nil {
		c.DeleteLoadBalancerPostHook(id, opts, err)
	}
	return nil
}

func (c *LbSimulator) CreateListener(opts listeners.CreateOptsBuilder) (*listeners.Listener, error) {
	if c.CreateListenerPreHook != nil {
		handled, listener, err := c.CreateListenerPreHook(opts)
		if handled {
			return listener, err
		}
	}
	listener, err := c.ImplCreateListener(opts)
	if c.CreateListenerPostHook != nil {
		c.CreateListenerPostHook(opts, listener, err)
	}
	return listener, err
}

func (c *LbSimulator) ListListeners(opts listeners.ListOptsBuilder) ([]listeners.Listener, error) {
	if c.ListListenersPreHook != nil {
		handled, listeners, err := c.ListListenersPreHook(opts)
		if handled {
			return listeners, err
		}
	}
	listeners, err := c.ImplListListeners(opts)
	if c.ListListenersPostHook != nil {
		c.ListListenersPostHook(opts, listeners, err)
	}
	return listeners, err
}

func (c *LbSimulator) UpdateListener(id string, opts listeners.UpdateOpts) (*listeners.Listener, error) {
	if c.UpdateListenerPreHook != nil {
		handled, listener, err := c.UpdateListenerPreHook(id, opts)
		if handled {
			return listener, err
		}
	}
	listener, err := c.ImplUpdateListener(id, opts)
	if c.UpdateListenerPostHook != nil {
		c.UpdateListenerPostHook(id, opts, listener, err)
	}
	return listener, err
}

func (c *LbSimulator) GetListener(id string) (*listeners.Listener, error) {
	if c.GetListenerPreHook != nil {
		handled, listener, err := c.GetListenerPreHook(id)
		if handled {
			return listener, err
		}
	}
	listener, err := c.ImplGetListener(id)
	if c.GetListenerPostHook != nil {
		c.GetListenerPostHook(id, listener, err)
	}
	return listener, err
}

func (c *LbSimulator) DeleteListener(id string) error {
	if c.DeleteListenerPreHook != nil {
		handled, err := c.DeleteListenerPreHook(id)
		if handled {
			return err
		}
	}
	err := c.ImplDeleteListener(id)
	if c.DeleteListenerPostHook != nil {
		c.DeleteListenerPostHook(id, err)
	}
	return err
}

func (c *LbSimulator) CreatePool(opts pools.CreateOptsBuilder) (*pools.Pool, error) {
	if c.CreatePoolPreHook != nil {
		handled, pool, err := c.CreatePoolPreHook(opts)
		if handled {
			return pool, err
		}
	}
	pool, err := c.ImplCreatePool(opts)
	if c.CreatePoolPostHook != nil {
		c.CreatePoolPostHook(opts, pool, err)
	}
	return pool, nil
}

func (c *LbSimulator) ListPools(opts pools.ListOptsBuilder) ([]pools.Pool, error) {
	if c.ListPoolsPreHook != nil {
		handled, pools, err := c.ListPoolsPreHook(opts)
		if handled {
			return pools, err
		}
	}
	pools, err := c.ImplListPools(opts)
	if c.ListPoolsPostHook != nil {
		c.ListPoolsPostHook(opts, pools, err)
	}
	return pools, err
}

func (c *LbSimulator) GetPool(id string) (*pools.Pool, error) {
	if c.GetPoolPreHook != nil {
		handled, pool, err := c.GetPoolPreHook(id)
		if handled {
			return pool, err
		}
	}
	pool, err := c.ImplGetPool(id)
	if c.GetPoolPostHook != nil {
		c.GetPoolPostHook(id, pool, err)
	}
	return pool, err
}

func (c *LbSimulator) DeletePool(id string) error {
	if c.DeletePoolPreHook != nil {
		handled, err := c.DeletePoolPreHook(id)
		if handled {
			return err
		}
	}
	err := c.ImplDeletePool(id)
	if c.DeletePoolPostHook != nil {
		c.DeletePoolPostHook(id, err)
	}
	return err
}

func (c *LbSimulator) CreatePoolMember(poolID string, opts pools.CreateMemberOptsBuilder) (*pools.Member, error) {
	if c.CreatePoolMemberPreHook != nil {
		handled, member, err := c.CreatePoolMemberPreHook(poolID, opts)
		if handled {
			return member, err
		}
	}
	member, err := c.ImplCreatePoolMember(poolID, opts)
	if c.CreatePoolMemberPostHook != nil {
		c.CreatePoolMemberPostHook(poolID, opts, member, err)
	}
	return member, err
}

func (c *LbSimulator) ListPoolMember(poolID string, opts pools.ListMembersOptsBuilder) ([]pools.Member, error) {
	if c.ListPoolMemberPreHook != nil {
		handled, members, err := c.ListPoolMemberPreHook(poolID, opts)
		if handled {
			return members, err
		}
	}
	members, err := c.ImplListPoolMember(poolID, opts)
	if c.ListPoolMemberPostHook != nil {
		c.ListPoolMemberPostHook(poolID, opts, members, err)
	}
	return members, err
}

func (c *LbSimulator) DeletePoolMember(poolID string, lbMemberID string) error {
	if c.DeletePoolMemberPreHook != nil {
		handled, err := c.DeletePoolMemberPreHook(poolID, lbMemberID)
		if handled {
			return err
		}
	}
	err := c.ImplDeletePoolMember(poolID, lbMemberID)
	if c.DeletePoolMemberPostHook != nil {
		c.DeletePoolMemberPostHook(poolID, lbMemberID, err)
	}
	return err
}

func (c *LbSimulator) CreateMonitor(opts monitors.CreateOptsBuilder) (*monitors.Monitor, error) {
	if c.CreateMonitorPreHook != nil {
		handled, monitor, err := c.CreateMonitorPreHook(opts)
		if handled {
			return monitor, err
		}
	}
	monitor, err := c.ImplCreateMonitor(opts)
	if c.CreateMonitorPostHook != nil {
		c.CreateMonitorPostHook(opts, monitor, err)
	}
	return monitor, err
}

func (c *LbSimulator) ListMonitors(opts monitors.ListOptsBuilder) ([]monitors.Monitor, error) {
	if c.ListMonitorsPreHook != nil {
		handled, monitors, err := c.ListMonitorsPreHook(opts)
		if handled {
			return monitors, err
		}
	}
	monitors, err := c.ImplListMonitors(opts)
	if c.ListMonitorsPostHook != nil {
		c.ListMonitorsPostHook(opts, monitors, err)
	}
	return monitors, err
}

func (c *LbSimulator) DeleteMonitor(id string) error {
	if c.DeleteMonitorPreHook != nil {
		handled, err := c.DeleteMonitorPreHook(id)
		if handled {
			return err
		}
	}
	err := c.ImplDeleteMonitor(id)
	if c.DeleteMonitorPostHook != nil {
		c.DeleteMonitorPostHook(id, err)
	}
	return err
}

func (c *LbSimulator) ListLoadBalancerProviders() ([]providers.Provider, error) {
	if c.ListLoadBalancerProvidersPreHook != nil {
		handled, providers, err := c.ListLoadBalancerProvidersPreHook()
		if handled {
			return providers, err
		}
	}
	providers, err := c.ImplListLoadBalancerProviders()
	if c.ListLoadBalancerProvidersPostHook != nil {
		c.ListLoadBalancerProvidersPostHook(providers, err)
	}
	return providers, err
}

func (c *LbSimulator) ListOctaviaVersions() ([]apiversions.APIVersion, error) {
	if c.ListOctaviaVersionsPreHook != nil {
		handled, versions, err := c.ListOctaviaVersionsPreHook()
		if handled {
			return versions, err
		}
	}
	versions, err := c.ImplListOctaviaVersions()
	if c.ListOctaviaVersionsPostHook != nil {
		c.ListOctaviaVersionsPostHook(versions, err)
	}
	return versions, err
}

/*
 * Simulator state helpers
 */

func (c *LbSimulator) SimAddLoadBalancerProvider(name string) {
	c.Providers = append(c.Providers, providers.Provider{
		Name: name,
	})
}

func (c *LbSimulator) SimSetOctaviaCurrentVersion(version string) {
	c.OctaviaCurrentVersion = version
}

/*
 * Default callbacks
 */

func (c *LbSimulator) CallBackCreateLoadbalancerSetActive(_ loadbalancers.CreateOptsBuilder, createdLb *loadbalancers.LoadBalancer, err error) {
	if err != nil || createdLb.ProvisioningStatus != ProvisioningStatusPendingCreate {
		return
	}

	go func() {
		c.Simulator.DefaultDelay()

		for i := range c.LoadBalancers {
			lb := &c.LoadBalancers[i]
			if lb.ID == createdLb.ID {
				if lb.ProvisioningStatus == ProvisioningStatusPendingCreate {
					lb.ProvisioningStatus = ProvisioningStatusActive
					lb.OperatingStatus = OperatingStatusOnline
				}
				return
			}
		}
	}()
}

func (c *LbSimulator) CallBackCreateListenerSetActive(_ listeners.CreateOptsBuilder, createdListener *listeners.Listener, err error) {
	if err != nil || createdListener.ProvisioningStatus != ProvisioningStatusPendingCreate {
		return
	}

	go func() {
		c.Simulator.DefaultDelay()

		for i := range c.Listeners {
			listener := &c.Listeners[i]
			if listener.ID == createdListener.ID {
				if listener.ProvisioningStatus == ProvisioningStatusPendingCreate {
					listener.ProvisioningStatus = ProvisioningStatusActive
				}
				return
			}
		}
	}()
}
