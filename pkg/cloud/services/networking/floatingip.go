/*
Copyright 2020 The Kubernetes Authors.

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
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha3"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
)

func (s *Service) GetOrCreateFloatingIP(openStackCluster *infrav1.OpenStackCluster, ip string) (*floatingips.FloatingIP, error) {
	var fp *floatingips.FloatingIP
	var err error
	if ip != "" {
		fp, err = checkIfFloatingIPExists(s.client, ip)
		if err != nil {
			return nil, err
		}
	}
	if fp == nil {
		fpCreateOpts := &floatingips.CreateOpts{
			FloatingNetworkID: openStackCluster.Status.ExternalNetwork.ID,
		}
		if ip != "" {
			// only admin can add ip address
			fpCreateOpts.FloatingIP = ip
		}
		fp, err = floatingips.Create(s.client, fpCreateOpts).Extract()
		if err != nil {
			return nil, fmt.Errorf("error creating floating IP: %s", err)
		}
		record.Eventf(openStackCluster, "SuccessfulCreateFloatingIP", "Created floating IP %s with id %s", fp.FloatingIP, fp.ID)

	}
	return fp, nil
}

func checkIfFloatingIPExists(client *gophercloud.ServiceClient, ip string) (*floatingips.FloatingIP, error) {
	allPages, err := floatingips.List(client, floatingips.ListOpts{FloatingIP: ip}).AllPages()
	if err != nil {
		return nil, err
	}
	fpList, err := floatingips.ExtractFloatingIPs(allPages)
	if err != nil {
		return nil, err
	}
	if len(fpList) == 0 {
		return nil, nil
	}
	return &fpList[0], nil
}
