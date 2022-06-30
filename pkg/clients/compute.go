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
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/attachinterfaces"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/utils/openstack/compute/v2/flavors"

	"sigs.k8s.io/cluster-api-provider-openstack/pkg/metrics"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

/*
NovaMinimumMicroversion is the minimum Nova microversion supported by CAPO
2.53 corresponds to OpenStack Pike

For the canonical description of Nova microversions, see
https://docs.openstack.org/nova/latest/reference/api-microversion-history.html

CAPO uses server tags, which were added in microversion 2.52.
*/
const NovaMinimumMicroversion = "2.53"

// ServerExt is the base gophercloud Server with extensions used by InstanceStatus.
type ServerExt struct {
	servers.Server
	availabilityzones.ServerAvailabilityZoneExt
}

type ComputeClient interface {
	ListAvailabilityZones() ([]availabilityzones.AvailabilityZone, error)

	ListImages(listOpts images.ListOptsBuilder) ([]images.Image, error)

	GetFlavorIDFromName(flavor string) (string, error)
	CreateServer(createOpts servers.CreateOptsBuilder) (*ServerExt, error)
	DeleteServer(serverID string) error
	GetServer(serverID string) (*ServerExt, error)
	ListServers(listOpts servers.ListOptsBuilder) ([]ServerExt, error)

	ListAttachedInterfaces(serverID string) ([]attachinterfaces.Interface, error)
	DeleteAttachedInterface(serverID, portID string) error

	ListVolumes(opts volumes.ListOptsBuilder) ([]volumes.Volume, error)
	CreateVolume(opts volumes.CreateOptsBuilder) (*volumes.Volume, error)
	DeleteVolume(volumeID string, opts volumes.DeleteOptsBuilder) error
	GetVolume(volumeID string) (*volumes.Volume, error)
}

type computeClient struct {
	compute *gophercloud.ServiceClient
	images  *gophercloud.ServiceClient
	volume  *gophercloud.ServiceClient
}

// NewComputeClient returns a new compute client.
func NewComputeClient(scope *scope.Scope) (ComputeClient, error) {
	compute, err := openstack.NewComputeV2(scope.ProviderClient, gophercloud.EndpointOpts{
		Region: scope.ProviderClientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service client: %v", err)
	}
	compute.Microversion = NovaMinimumMicroversion

	images, err := openstack.NewImageServiceV2(scope.ProviderClient, gophercloud.EndpointOpts{
		Region: scope.ProviderClientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create image service client: %v", err)
	}

	volume, err := openstack.NewBlockStorageV3(scope.ProviderClient, gophercloud.EndpointOpts{
		Region: scope.ProviderClientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create volume service client: %v", err)
	}

	return &computeClient{compute, images, volume}, nil
}

func (s computeClient) ListAvailabilityZones() ([]availabilityzones.AvailabilityZone, error) {
	mc := metrics.NewMetricPrometheusContext("availability_zone", "list")
	allPages, err := availabilityzones.List(s.compute).AllPages()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return availabilityzones.ExtractAvailabilityZones(allPages)
}

func (s computeClient) ListImages(listOpts images.ListOptsBuilder) ([]images.Image, error) {
	mc := metrics.NewMetricPrometheusContext("image", "list")
	pages, err := images.List(s.images, listOpts).AllPages()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return images.ExtractImages(pages)
}

func (s computeClient) GetFlavorIDFromName(flavor string) (string, error) {
	mc := metrics.NewMetricPrometheusContext("flavor", "get")
	flavorID, err := flavors.IDFromName(s.compute, flavor)
	return flavorID, mc.ObserveRequest(err)
}

func (s computeClient) CreateServer(createOpts servers.CreateOptsBuilder) (*ServerExt, error) {
	var server ServerExt
	mc := metrics.NewMetricPrometheusContext("server", "create")
	err := servers.Create(s.compute, createOpts).ExtractInto(&server)
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return &server, nil
}

func (s computeClient) DeleteServer(serverID string) error {
	mc := metrics.NewMetricPrometheusContext("server", "delete")
	err := servers.Delete(s.compute, serverID).ExtractErr()
	return mc.ObserveRequestIgnoreNotFound(err)
}

func (s computeClient) GetServer(serverID string) (*ServerExt, error) {
	var server ServerExt
	mc := metrics.NewMetricPrometheusContext("server", "get")
	err := servers.Get(s.compute, serverID).ExtractInto(&server)
	if mc.ObserveRequestIgnoreNotFound(err) != nil {
		return nil, err
	}
	return &server, nil
}

func (s computeClient) ListServers(listOpts servers.ListOptsBuilder) ([]ServerExt, error) {
	var serverList []ServerExt
	mc := metrics.NewMetricPrometheusContext("server", "list")
	allPages, err := servers.List(s.compute, listOpts).AllPages()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	err = servers.ExtractServersInto(allPages, &serverList)
	return serverList, err
}

func (s computeClient) ListAttachedInterfaces(serverID string) ([]attachinterfaces.Interface, error) {
	mc := metrics.NewMetricPrometheusContext("server_os_interface", "list")
	interfaces, err := attachinterfaces.List(s.compute, serverID).AllPages()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return attachinterfaces.ExtractInterfaces(interfaces)
}

func (s computeClient) DeleteAttachedInterface(serverID, portID string) error {
	mc := metrics.NewMetricPrometheusContext("server_os_interface", "delete")
	err := attachinterfaces.Delete(s.compute, serverID, portID).ExtractErr()
	return mc.ObserveRequestIgnoreNotFoundorConflict(err)
}

func (s computeClient) ListVolumes(opts volumes.ListOptsBuilder) ([]volumes.Volume, error) {
	mc := metrics.NewMetricPrometheusContext("volume", "list")
	pages, err := volumes.List(s.volume, opts).AllPages()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return volumes.ExtractVolumes(pages)
}

func (s computeClient) CreateVolume(opts volumes.CreateOptsBuilder) (*volumes.Volume, error) {
	mc := metrics.NewMetricPrometheusContext("volume", "create")
	volume, err := volumes.Create(s.volume, opts).Extract()
	return volume, mc.ObserveRequest(err)
}

func (s computeClient) DeleteVolume(volumeID string, opts volumes.DeleteOptsBuilder) error {
	mc := metrics.NewMetricPrometheusContext("volume", "delete")
	err := volumes.Delete(s.volume, volumeID, opts).ExtractErr()
	return mc.ObserveRequestIgnoreNotFound(err)
}

func (s computeClient) GetVolume(volumeID string) (*volumes.Volume, error) {
	mc := metrics.NewMetricPrometheusContext("volume", "get")
	volume, err := volumes.Get(s.volume, volumeID).Extract()
	return volume, mc.ObserveRequestIgnoreNotFound(err)
}
