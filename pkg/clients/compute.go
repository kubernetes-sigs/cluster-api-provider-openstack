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
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/attachinterfaces"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/availabilityzones"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servergroups"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"

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
	NovaMultiAttachVolume   = "2.60"
)

type ComputeClient interface {
	ListAvailabilityZones(ctx context.Context) ([]availabilityzones.AvailabilityZone, error)

	ListFlavors(ctx context.Context) ([]flavors.Flavor, error)
	GetFlavor(ctx context.Context, flavorID string) (*flavors.Flavor, error)

	CreateServer(ctx context.Context, createOpts servers.CreateOptsBuilder, schedulerHints servers.SchedulerHintOptsBuilder) (*servers.Server, error)
	DeleteServer(ctx context.Context, serverID string) error
	GetServer(ctx context.Context, serverID string) (*servers.Server, error)
	ListServers(ctx context.Context, listOpts servers.ListOptsBuilder) ([]servers.Server, error)

	ListAttachedInterfaces(ctx context.Context, serverID string) ([]attachinterfaces.Interface, error)
	DeleteAttachedInterface(ctx context.Context, serverID, portID string) error

	ListServerGroups(ctx context.Context) ([]servergroups.ServerGroup, error)
	GetConsoleOutput(ctx context.Context, serverID string) (string, error)
	WithMicroversion(required string) (ComputeClient, error)
}

var _ ComputeClient = &computeClient{}

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

func (c computeClient) ListAvailabilityZones(ctx context.Context) ([]availabilityzones.AvailabilityZone, error) {
	mc := metrics.NewMetricPrometheusContext("availability_zone", "list")
	allPages, err := availabilityzones.List(c.client).AllPages(ctx)
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return availabilityzones.ExtractAvailabilityZones(allPages)
}

func (c computeClient) ListFlavors(ctx context.Context) ([]flavors.Flavor, error) {
	mc := metrics.NewMetricPrometheusContext("flavor", "list")
	allPages, err := flavors.ListDetail(c.client, &flavors.ListOpts{}).AllPages(ctx)
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return flavors.ExtractFlavors(allPages)
}

func (c computeClient) GetFlavor(ctx context.Context, flavorID string) (*flavors.Flavor, error) {
	mc := metrics.NewMetricPrometheusContext("flavor", "get")
	flavor, err := flavors.Get(ctx, c.client, flavorID).Extract()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return flavor, nil
}

func (c computeClient) CreateServer(ctx context.Context, createOpts servers.CreateOptsBuilder, schedulerHints servers.SchedulerHintOptsBuilder) (*servers.Server, error) {
	mc := metrics.NewMetricPrometheusContext("server", "create")
	server, err := servers.Create(ctx, c.client, createOpts, schedulerHints).Extract()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return server, nil
}

func (c computeClient) DeleteServer(ctx context.Context, serverID string) error {
	mc := metrics.NewMetricPrometheusContext("server", "delete")
	err := servers.Delete(ctx, c.client, serverID).ExtractErr()
	return mc.ObserveRequestIgnoreNotFound(err)
}

func (c computeClient) GetServer(ctx context.Context, serverID string) (*servers.Server, error) {
	var server servers.Server
	mc := metrics.NewMetricPrometheusContext("server", "get")
	err := servers.Get(ctx, c.client, serverID).ExtractInto(&server)
	if mc.ObserveRequestIgnoreNotFound(err) != nil {
		return nil, err
	}
	return &server, nil
}

func (c computeClient) ListServers(ctx context.Context, listOpts servers.ListOptsBuilder) ([]servers.Server, error) {
	var serverList []servers.Server
	mc := metrics.NewMetricPrometheusContext("server", "list")
	allPages, err := servers.List(c.client, listOpts).AllPages(ctx)
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	err = servers.ExtractServersInto(allPages, &serverList)
	return serverList, err
}

func (c computeClient) ListAttachedInterfaces(ctx context.Context, serverID string) ([]attachinterfaces.Interface, error) {
	mc := metrics.NewMetricPrometheusContext("server_os_interface", "list")
	interfaces, err := attachinterfaces.List(c.client, serverID).AllPages(ctx)
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return attachinterfaces.ExtractInterfaces(interfaces)
}

func (c computeClient) DeleteAttachedInterface(ctx context.Context, serverID, portID string) error {
	mc := metrics.NewMetricPrometheusContext("server_os_interface", "delete")
	err := attachinterfaces.Delete(ctx, c.client, serverID, portID).ExtractErr()
	return mc.ObserveRequestIgnoreNotFoundorConflict(err)
}

func (c computeClient) ListServerGroups(ctx context.Context) ([]servergroups.ServerGroup, error) {
	mc := metrics.NewMetricPrometheusContext("server_group", "list")
	opts := servergroups.ListOpts{}
	allPages, err := servergroups.List(c.client, opts).AllPages(ctx)
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return servergroups.ExtractServerGroups(allPages)
}

func (c computeClient) GetConsoleOutput(ctx context.Context, serverID string) (string, error) {
	opts := servers.ShowConsoleOutputOpts{}
	return servers.ShowConsoleOutput(ctx, c.client, serverID, opts).Extract()
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

var _ ComputeClient = &computeClient{}

// NewComputeErrorClient returns a ComputeClient in which every method returns the given error.
func NewComputeErrorClient(e error) ComputeClient {
	return computeErrorClient{e}
}

func (e computeErrorClient) ListAvailabilityZones(_ context.Context) ([]availabilityzones.AvailabilityZone, error) {
	return nil, e.error
}

func (e computeErrorClient) ListFlavors(_ context.Context) ([]flavors.Flavor, error) {
	return nil, e.error
}

func (e computeErrorClient) GetFlavor(_ context.Context, _ string) (*flavors.Flavor, error) {
	return nil, e.error
}

func (e computeErrorClient) CreateServer(_ context.Context, _ servers.CreateOptsBuilder, _ servers.SchedulerHintOptsBuilder) (*servers.Server, error) {
	return nil, e.error
}

func (e computeErrorClient) DeleteServer(_ context.Context, _ string) error {
	return e.error
}

func (e computeErrorClient) GetServer(_ context.Context, _ string) (*servers.Server, error) {
	return nil, e.error
}

func (e computeErrorClient) ListServers(_ context.Context, _ servers.ListOptsBuilder) ([]servers.Server, error) {
	return nil, e.error
}

func (e computeErrorClient) ListAttachedInterfaces(_ context.Context, _ string) ([]attachinterfaces.Interface, error) {
	return nil, e.error
}

func (e computeErrorClient) DeleteAttachedInterface(_ context.Context, _, _ string) error {
	return e.error
}

func (e computeErrorClient) ListServerGroups(_ context.Context) ([]servergroups.ServerGroup, error) {
	return nil, e.error
}

func (e computeErrorClient) GetConsoleOutput(_ context.Context, _ string) (string, error) {
	return "", e.error
}

func (e computeErrorClient) WithMicroversion(_ string) (ComputeClient, error) {
	return nil, e.error
}
