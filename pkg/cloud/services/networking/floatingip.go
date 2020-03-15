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
)

func (s *Service) CreateFloatingIPIfNecessary(openStackCluster *infrav1.OpenStackCluster, ip string) error {
	fp, err := checkIfFloatingIPExists(s.client, ip)
	if err != nil {
		return err
	}
	if fp == nil {
		s.logger.Info("Creating floating ip", "ip", ip)
		fpCreateOpts := &floatingips.CreateOpts{
			FloatingIP:        ip,
			FloatingNetworkID: openStackCluster.Spec.ExternalNetworkID,
		}
		if _, err = floatingips.Create(s.client, fpCreateOpts).Extract(); err != nil {
			return fmt.Errorf("error allocating floating IP: %s", err)
		}
	}
	return nil
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
