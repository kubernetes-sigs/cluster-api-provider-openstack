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

package compute

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/attachinterfaces"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/utils/openstack/compute/v2/flavors"

	"sigs.k8s.io/cluster-api-provider-openstack/pkg/metrics"
)

//go:generate mockgen -package=compute -self_package sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/compute -destination=client_mock.go sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/compute Client
//go:generate /usr/bin/env bash -c "cat ../../../../hack/boilerplate/boilerplate.generatego.txt client_mock.go > _client_mock.go && mv _client_mock.go client_mock.go"

// ServerExt is the base gophercloud Server with extensions used by InstanceStatus.
type ServerExt struct {
	servers.Server
	availabilityzones.ServerAvailabilityZoneExt
}

type Client interface {
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

type serviceClient struct {
	compute *gophercloud.ServiceClient
	images  *gophercloud.ServiceClient
	volume  *gophercloud.ServiceClient
}

func (s serviceClient) ListAvailabilityZones() ([]availabilityzones.AvailabilityZone, error) {
	mc := metrics.NewMetricPrometheusContext("availability_zone", "list")
	allPages, err := availabilityzones.List(s.compute).AllPages()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return availabilityzones.ExtractAvailabilityZones(allPages)
}

func (s serviceClient) ListImages(listOpts images.ListOptsBuilder) ([]images.Image, error) {
	mc := metrics.NewMetricPrometheusContext("image", "list")
	pages, err := images.List(s.images, listOpts).AllPages()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return images.ExtractImages(pages)
}

func (s serviceClient) GetFlavorIDFromName(flavor string) (string, error) {
	mc := metrics.NewMetricPrometheusContext("flavor", "get")
	flavorID, err := flavors.IDFromName(s.compute, flavor)
	return flavorID, mc.ObserveRequest(err)
}

func (s serviceClient) CreateServer(createOpts servers.CreateOptsBuilder) (*ServerExt, error) {
	var server ServerExt
	mc := metrics.NewMetricPrometheusContext("server", "create")
	err := servers.Create(s.compute, createOpts).ExtractInto(&server)
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return &server, nil
}

func (s serviceClient) DeleteServer(serverID string) error {
	mc := metrics.NewMetricPrometheusContext("server", "delete")
	err := servers.Delete(s.compute, serverID).ExtractErr()
	return mc.ObserveRequestIgnoreNotFound(err)
}

func (s serviceClient) GetServer(serverID string) (*ServerExt, error) {
	var server ServerExt
	mc := metrics.NewMetricPrometheusContext("server", "get")
	err := servers.Get(s.compute, serverID).ExtractInto(&server)
	if mc.ObserveRequestIgnoreNotFound(err) != nil {
		return nil, err
	}
	return &server, nil
}

func (s serviceClient) ListServers(listOpts servers.ListOptsBuilder) ([]ServerExt, error) {
	var serverList []ServerExt
	mc := metrics.NewMetricPrometheusContext("server", "list")
	allPages, err := servers.List(s.compute, listOpts).AllPages()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	err = servers.ExtractServersInto(allPages, &serverList)
	return serverList, err
}

func (s serviceClient) ListAttachedInterfaces(serverID string) ([]attachinterfaces.Interface, error) {
	mc := metrics.NewMetricPrometheusContext("server_os_interface", "list")
	interfaces, err := attachinterfaces.List(s.compute, serverID).AllPages()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return attachinterfaces.ExtractInterfaces(interfaces)
}

func (s serviceClient) DeleteAttachedInterface(serverID, portID string) error {
	mc := metrics.NewMetricPrometheusContext("server_os_interface", "delete")
	err := attachinterfaces.Delete(s.compute, serverID, portID).ExtractErr()
	return mc.ObserveRequestIgnoreNotFoundorConflict(err)
}

func (s serviceClient) ListVolumes(opts volumes.ListOptsBuilder) ([]volumes.Volume, error) {
	mc := metrics.NewMetricPrometheusContext("volume", "list")
	pages, err := volumes.List(s.volume, opts).AllPages()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return volumes.ExtractVolumes(pages)
}

func (s serviceClient) CreateVolume(opts volumes.CreateOptsBuilder) (*volumes.Volume, error) {
	mc := metrics.NewMetricPrometheusContext("volume", "create")
	volume, err := volumes.Create(s.volume, opts).Extract()
	return volume, mc.ObserveRequest(err)
}

func (s serviceClient) DeleteVolume(volumeID string, opts volumes.DeleteOptsBuilder) error {
	mc := metrics.NewMetricPrometheusContext("volume", "delete")
	err := volumes.Delete(s.volume, volumeID, opts).ExtractErr()
	return mc.ObserveRequestIgnoreNotFound(err)
}

func (s serviceClient) GetVolume(volumeID string) (*volumes.Volume, error) {
	mc := metrics.NewMetricPrometheusContext("volume", "get")
	volume, err := volumes.Get(s.volume, volumeID).Extract()
	return volume, mc.ObserveRequestIgnoreNotFound(err)
}
