package networking

import (
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"k8s.io/klog"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha2"
)

func (s *Service) GetOrCreateFloatingIP(openStackCluster *infrav1.OpenStackCluster, ip string) error {
	fp, err := checkIfFloatingIPExists(s.client, ip)
	if err != nil {
		return err
	}
	if fp == nil {
		klog.Infof("Creating floating ip %s", ip)
		fpCreateOpts := &floatingips.CreateOpts{
			FloatingIP:        ip,
			FloatingNetworkID: openStackCluster.Spec.ExternalNetworkID,
		}
		fp, err = floatingips.Create(s.client, fpCreateOpts).Extract()
		if err != nil {
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
