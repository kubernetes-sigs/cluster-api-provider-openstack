package networking

import (
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	providerv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	"time"
)

func (s *Service) GetOrCreateFloatingIP(clusterProviderSpec *providerv1.OpenstackClusterProviderSpec, ip string) error {
	fp, err := checkIfFloatingIPExists(s.client, ip)
	if err != nil {
		return err
	}
	if fp == nil {
		klog.Infof("Creating floating ip %s", ip)
		fpCreateOpts := &floatingips.CreateOpts{
			FloatingIP:        ip,
			FloatingNetworkID: clusterProviderSpec.ExternalNetworkID,
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

var backoff = wait.Backoff{
	Steps:    10,
	Duration: 30 * time.Second,
	Factor:   1.0,
	Jitter:   0.1,
}

func waitForFloatingIP(client *gophercloud.ServiceClient, id, target string) error {
	klog.Infof("Waiting for floatingip %s to become %s.", id, target)
	return wait.ExponentialBackoff(backoff, func() (bool, error) {
		fp, err := floatingips.Get(client, id).Extract()
		if err != nil {
			return false, err
		}
		return fp.Status == target, nil
	})
}
