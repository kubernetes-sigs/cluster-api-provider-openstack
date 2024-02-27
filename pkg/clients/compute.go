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
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/attachinterfaces"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/servergroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/utils/openstack/clientconfig"
	uflavors "github.com/gophercloud/utils/openstack/compute/v2/flavors"

	"sigs.k8s.io/cluster-api-provider-openstack/pkg/metrics"
	openstackutil "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/openstack"
)

/*
Constants for specific microversion requirements.
2.60 corresponds to OpenStack Queens and 2.53 to OpenStack Pike,
2.38 is the maximum in OpenStack Newton.

For the canonical description of Nova microversions, see
https://docs.openstack.org/nova/latest/reference/api-microversion-history.html

CAPO uses server tags, which were first added in microversion 2.26 and then refined
in 2.52 so it is possible to apply them when creating a server (which is what CAPO does).
We round up to 2.53 here since that takes us to the maximum in Pike.

CAPO supports multiattach volume types, which were added in microversion 2.60.

2.38 was chosen as a base level since it is reasonably old, but not too old.
*/
const (
	MinimumNovaMicroversion = "2.38"
	NovaTagging             = "2.53"
	MultiAttachVolume       = "2.60"
)

// ServerExt is the base gophercloud Server with extensions used by InstanceStatus.
type ServerExt struct {
	servers.Server
	availabilityzones.ServerAvailabilityZoneExt
}

type ComputeClient interface {
	ListAvailabilityZones() ([]availabilityzones.AvailabilityZone, error)

	GetFlavorFromName(flavor string) (*flavors.Flavor, error)
	CreateServer(createOpts servers.CreateOptsBuilder) (*ServerExt, error)
	DeleteServer(serverID string) error
	GetServer(serverID string) (*ServerExt, error)
	ListServers(listOpts servers.ListOptsBuilder) ([]ServerExt, error)

	ListAttachedInterfaces(serverID string) ([]attachinterfaces.Interface, error)
	DeleteAttachedInterface(serverID, portID string) error

	ListServerGroups() ([]servergroups.ServerGroup, error)
	WithMicroversion(required string) (ComputeClient, error)
}

type computeClient struct {
	client     *gophercloud.ServiceClient
	minVersion string
	maxVersion string
}

// NewComputeClient returns a new compute client.
func NewComputeClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (ComputeClient, error) {
	compute, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service client: %v", err)
	}

	// Find the minimum and maximum versions supported by the server
	serviceMin, serviceMax, err := openstackutil.GetSupportedMicroversions(*compute)
	if err != nil {
		return nil, fmt.Errorf("unable to verify compatible server version: %w", err)
	}

	supported, err := openstackutil.MicroversionSupported(MinimumNovaMicroversion, serviceMin, serviceMax)
	if err != nil {
		return nil, fmt.Errorf("unable to verify compatible server version: %w", err)
	}
	if !supported {
		return nil, fmt.Errorf("no compatible server version. CAPO requires %s, but min=%s and max=%s",
			MinimumNovaMicroversion, serviceMin, serviceMax)
	}

	compute.Microversion = MinimumNovaMicroversion

	return &computeClient{client: compute, minVersion: serviceMin, maxVersion: serviceMax}, nil
}

func (c computeClient) ListAvailabilityZones() ([]availabilityzones.AvailabilityZone, error) {
	mc := metrics.NewMetricPrometheusContext("availability_zone", "list")
	allPages, err := availabilityzones.List(c.client).AllPages()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return availabilityzones.ExtractAvailabilityZones(allPages)
}

func (c computeClient) GetFlavorFromName(flavor string) (*flavors.Flavor, error) {
	mc := metrics.NewMetricPrometheusContext("flavor", "get")
	flavorID, err := uflavors.IDFromName(c.client, flavor)
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	f, err := flavors.Get(c.client, flavorID).Extract()
	return f, mc.ObserveRequest(err)
}

func (c computeClient) CreateServer(createOpts servers.CreateOptsBuilder) (*ServerExt, error) {
	var server ServerExt
	mc := metrics.NewMetricPrometheusContext("server", "create")
	err := servers.Create(c.client, createOpts).ExtractInto(&server)
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return &server, nil
}

func (c computeClient) DeleteServer(serverID string) error {
	mc := metrics.NewMetricPrometheusContext("server", "delete")
	err := servers.Delete(c.client, serverID).ExtractErr()
	return mc.ObserveRequestIgnoreNotFound(err)
}

func (c computeClient) GetServer(serverID string) (*ServerExt, error) {
	var server ServerExt
	mc := metrics.NewMetricPrometheusContext("server", "get")
	err := servers.Get(c.client, serverID).ExtractInto(&server)
	if mc.ObserveRequestIgnoreNotFound(err) != nil {
		return nil, err
	}
	return &server, nil
}

func (c computeClient) ListServers(listOpts servers.ListOptsBuilder) ([]ServerExt, error) {
	var serverList []ServerExt
	mc := metrics.NewMetricPrometheusContext("server", "list")
	allPages, err := servers.List(c.client, listOpts).AllPages()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	err = servers.ExtractServersInto(allPages, &serverList)
	return serverList, err
}

func (c computeClient) ListAttachedInterfaces(serverID string) ([]attachinterfaces.Interface, error) {
	mc := metrics.NewMetricPrometheusContext("server_os_interface", "list")
	interfaces, err := attachinterfaces.List(c.client, serverID).AllPages()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return attachinterfaces.ExtractInterfaces(interfaces)
}

func (c computeClient) DeleteAttachedInterface(serverID, portID string) error {
	mc := metrics.NewMetricPrometheusContext("server_os_interface", "delete")
	err := attachinterfaces.Delete(c.client, serverID, portID).ExtractErr()
	return mc.ObserveRequestIgnoreNotFoundorConflict(err)
}

func (c computeClient) ListServerGroups() ([]servergroups.ServerGroup, error) {
	mc := metrics.NewMetricPrometheusContext("server_group", "list")
	opts := servergroups.ListOpts{}
	allPages, err := servergroups.List(c.client, opts).AllPages()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return servergroups.ExtractServerGroups(allPages)
}

// WithMicroversion checks that the required Nova microversion is supported and sets it for
// the ComputeClient.
func (c computeClient) WithMicroversion(required string) (ComputeClient, error) {
	supported, err := openstackutil.MicroversionSupported(required, c.minVersion, c.maxVersion)
	if err != nil {
		return nil, err
	}
	if !supported {
		return nil, fmt.Errorf("microversion %s not supported. Min=%s, max=%s", required, c.minVersion, c.maxVersion)
	}
	versionedClient := c
	versionedClient.client.Microversion = required
	return versionedClient, nil
}

type computeErrorClient struct{ error }

// NewComputeErrorClient returns a ComputeClient in which every method returns the given error.
func NewComputeErrorClient(e error) ComputeClient {
	return computeErrorClient{e}
}

func (e computeErrorClient) ListAvailabilityZones() ([]availabilityzones.AvailabilityZone, error) {
	return nil, e.error
}

func (e computeErrorClient) GetFlavorFromName(_ string) (*flavors.Flavor, error) {
	return nil, e.error
}

func (e computeErrorClient) CreateServer(_ servers.CreateOptsBuilder) (*ServerExt, error) {
	return nil, e.error
}

func (e computeErrorClient) DeleteServer(_ string) error {
	return e.error
}

func (e computeErrorClient) GetServer(_ string) (*ServerExt, error) {
	return nil, e.error
}

func (e computeErrorClient) ListServers(_ servers.ListOptsBuilder) ([]ServerExt, error) {
	return nil, e.error
}

func (e computeErrorClient) ListAttachedInterfaces(_ string) ([]attachinterfaces.Interface, error) {
	return nil, e.error
}

func (e computeErrorClient) DeleteAttachedInterface(_, _ string) error {
	return e.error
}

func (e computeErrorClient) ListServerGroups() ([]servergroups.ServerGroup, error) {
	return nil, e.error
}

func (e computeErrorClient) WithMicroversion(_ string) (ComputeClient, error) {
	return nil, e.error
}
