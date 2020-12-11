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

package loadbalancer

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/listeners"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/monitors"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/pools"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha3"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
)

func (s *Service) ReconcileLoadBalancer(clusterName string, openStackCluster *infrav1.OpenStackCluster) error {

	loadBalancerName := fmt.Sprintf("%s-cluster-%s-%s", networkPrefix, clusterName, kubeapiLBSuffix)
	s.logger.Info("Reconciling loadbalancer", "name", loadBalancerName)

	// lb
	lb, err := checkIfLbExists(s.loadbalancerClient, loadBalancerName)
	if err != nil {
		return err
	}
	if lb == nil {
		s.logger.Info("Creating loadbalancer", "name", loadBalancerName)
		lbCreateOpts := loadbalancers.CreateOpts{
			Name:        loadBalancerName,
			VipSubnetID: openStackCluster.Status.Network.Subnet.ID,
		}

		lb, err = loadbalancers.Create(s.loadbalancerClient, lbCreateOpts).Extract()
		if err != nil {
			return fmt.Errorf("error creating loadbalancer: %s", err)
		}
	}
	if err := waitForLoadBalancerActive(s.logger, s.loadbalancerClient, lb.ID); err != nil {
		return err
	}

	if !openStackCluster.Spec.UseOctavia {
		err := s.assignNeutronLbaasAPISecGroup(clusterName, lb)
		if err != nil {
			return err
		}
	}

	fp, err := s.networkingService.GetOrCreateFloatingIP(openStackCluster, openStackCluster.Spec.ControlPlaneEndpoint.Host)
	if err != nil {
		return err
	}
	err = s.networkingService.AssociateFloatingIP(fp, lb.VipPortID)
	if err != nil {
		return err
	}
	record.Eventf(openStackCluster, "SuccessfulAssociateFloatingIP", "Associate floating IP %s with port %s", fp.FloatingIP, lb.VipPortID)

	// lb listener
	portList := []int{int(openStackCluster.Spec.ControlPlaneEndpoint.Port)}
	portList = append(portList, openStackCluster.Spec.APIServerLoadBalancerAdditionalPorts...)
	for _, port := range portList {
		lbPortObjectsName := fmt.Sprintf("%s-%d", loadBalancerName, port)

		listener, err := checkIfListenerExists(s.loadbalancerClient, lbPortObjectsName)
		if err != nil {
			return err
		}
		if listener == nil {
			s.logger.Info("Creating lb listener", "name", lbPortObjectsName)
			listenerCreateOpts := listeners.CreateOpts{
				Name:           lbPortObjectsName,
				Protocol:       "TCP",
				ProtocolPort:   port,
				LoadbalancerID: lb.ID,
			}
			listener, err = listeners.Create(s.loadbalancerClient, listenerCreateOpts).Extract()
			if err != nil {
				return fmt.Errorf("error creating listener: %s", err)
			}
		}
		if err := waitForLoadBalancerActive(s.logger, s.loadbalancerClient, lb.ID); err != nil {
			return err
		}

		if err := waitForListener(s.logger, s.loadbalancerClient, listener.ID, "ACTIVE"); err != nil {
			return err
		}

		// lb pool
		pool, err := checkIfPoolExists(s.loadbalancerClient, lbPortObjectsName)
		if err != nil {
			return err
		}
		if pool == nil {
			s.logger.Info("Creating lb pool", "name", lbPortObjectsName)
			poolCreateOpts := pools.CreateOpts{
				Name:       lbPortObjectsName,
				Protocol:   "TCP",
				LBMethod:   pools.LBMethodRoundRobin,
				ListenerID: listener.ID,
			}
			pool, err = pools.Create(s.loadbalancerClient, poolCreateOpts).Extract()
			if err != nil {
				return fmt.Errorf("error creating pool: %s", err)
			}
		}
		if err := waitForLoadBalancerActive(s.logger, s.loadbalancerClient, lb.ID); err != nil {
			return err
		}

		// lb monitor
		monitor, err := checkIfMonitorExists(s.loadbalancerClient, lbPortObjectsName)
		if err != nil {
			return err
		}
		if monitor == nil {
			s.logger.Info("Creating lb monitor", "name", lbPortObjectsName)
			monitorCreateOpts := monitors.CreateOpts{
				Name:       lbPortObjectsName,
				PoolID:     pool.ID,
				Type:       "TCP",
				Delay:      30,
				Timeout:    5,
				MaxRetries: 3,
			}
			_, err = monitors.Create(s.loadbalancerClient, monitorCreateOpts).Extract()
			if err != nil {
				return fmt.Errorf("error creating monitor: %s", err)
			}
		}
		if err = waitForLoadBalancerActive(s.logger, s.loadbalancerClient, lb.ID); err != nil {
			return err
		}
	}

	openStackCluster.Status.Network.APIServerLoadBalancer = &infrav1.LoadBalancer{
		Name:       lb.Name,
		ID:         lb.ID,
		InternalIP: lb.VipAddress,
		IP:         fp.FloatingIP,
	}
	return nil
}

func (s *Service) assignNeutronLbaasAPISecGroup(clusterName string, lb *loadbalancers.LoadBalancer) error {
	neutronLbaasSecGroupName := networking.GetNeutronLBaasSecGroupName(clusterName)
	listOpts := groups.ListOpts{
		Name: neutronLbaasSecGroupName,
	}
	allPages, err := groups.List(s.loadbalancerClient, listOpts).AllPages()
	if err != nil {
		return err
	}

	neutronLbaasGroups, err := groups.ExtractGroups(allPages)
	if err != nil {
		return err
	}

	if len(neutronLbaasGroups) != 1 {
		return fmt.Errorf("error found %v securitygroups with name %v", len(neutronLbaasGroups), neutronLbaasSecGroupName)
	}

	updateOpts := ports.UpdateOpts{
		SecurityGroups: &[]string{neutronLbaasGroups[0].ID},
	}

	_, err = ports.Update(s.loadbalancerClient, lb.VipPortID, updateOpts).Extract()
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) ReconcileLoadBalancerMember(clusterName string, machine *clusterv1.Machine, openStackMachine *infrav1.OpenStackMachine, openStackCluster *infrav1.OpenStackCluster, ip string) error {
	if !util.IsControlPlaneMachine(machine) {
		return nil
	}

	if openStackCluster.Status.Network == nil {
		return errors.New("network is not yet available in openStackCluster.Status")
	}
	if openStackCluster.Status.Network.Subnet == nil {
		return errors.New("network.Subnet is not yet available in openStackCluster.Status")
	}
	if openStackCluster.Status.Network.APIServerLoadBalancer == nil {
		return errors.New("network.APIServerLoadBalancer is not yet available in openStackCluster.Status")
	}

	loadBalancerName := fmt.Sprintf("%s-cluster-%s-%s", networkPrefix, clusterName, kubeapiLBSuffix)
	s.logger.Info("Reconciling loadbalancer", "name", loadBalancerName)

	lbID := openStackCluster.Status.Network.APIServerLoadBalancer.ID
	subnetID := openStackCluster.Status.Network.Subnet.ID
	portList := []int{int(openStackCluster.Spec.ControlPlaneEndpoint.Port)}
	portList = append(portList, openStackCluster.Spec.APIServerLoadBalancerAdditionalPorts...)
	for _, port := range portList {
		lbPortObjectsName := fmt.Sprintf("%s-%d", loadBalancerName, port)
		name := lbPortObjectsName + "-" + openStackMachine.Name

		pool, err := checkIfPoolExists(s.loadbalancerClient, lbPortObjectsName)
		if err != nil {
			return err
		}
		if pool == nil {
			return errors.New("loadbalancer pool does not exist yet")
		}

		lbMember, err := checkIfLbMemberExists(s.loadbalancerClient, pool.ID, name)
		if err != nil {
			return err
		}

		if lbMember != nil {
			// check if we have to recreate the LB Member
			if lbMember.Address == ip {
				// nothing to do return
				return nil
			}

			s.logger.Info("Deleting lb member (because the IP of the machine changed)", "name", name)

			// lb member changed so let's delete it so we can create it again with the correct IP
			err = waitForLoadBalancerActive(s.logger, s.loadbalancerClient, lbID)
			if err != nil {
				return err
			}
			err = pools.DeleteMember(s.loadbalancerClient, pool.ID, lbMember.ID).ExtractErr()
			if err != nil {
				return fmt.Errorf("error deleting lbmember: %s", err)
			}
			err = waitForLoadBalancerActive(s.logger, s.loadbalancerClient, lbID)
			if err != nil {
				return err
			}
		}

		s.logger.Info("Creating lb member", "name", name)

		// if we got to this point we should either create or re-create the lb member
		lbMemberOpts := pools.CreateMemberOpts{
			Name:         name,
			ProtocolPort: port,
			Address:      ip,
			SubnetID:     subnetID,
		}

		if err := waitForLoadBalancerActive(s.logger, s.loadbalancerClient, lbID); err != nil {
			return err
		}
		if _, err := pools.CreateMember(s.loadbalancerClient, pool.ID, lbMemberOpts).Extract(); err != nil {
			return fmt.Errorf("error create lbmember: %s", err)
		}
		if err := waitForLoadBalancerActive(s.logger, s.loadbalancerClient, lbID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) DeleteLoadBalancer(loadBalancerName string, openStackCluster *infrav1.OpenStackCluster) error {
	lb, err := checkIfLbExists(s.loadbalancerClient, loadBalancerName)
	if err != nil {
		return err
	}
	if lb == nil {
		// nothing to do
		return nil
	}

	// only Octavia supports Cascade
	if openStackCluster.Spec.UseOctavia {
		deleteOpts := loadbalancers.DeleteOpts{
			Cascade: true,
		}
		s.logger.Info("Deleting loadbalancer", "name", loadBalancerName)
		err = loadbalancers.Delete(s.loadbalancerClient, lb.ID, deleteOpts).ExtractErr()
		if err != nil {
			return fmt.Errorf("error deleting loadbalancer: %s", err)
		}
	} else if err := s.deleteLoadBalancerNeutronV2(lb.ID); err != nil {
		return fmt.Errorf("error deleting loadbalancer: %s", err)
	}

	return nil
}

// ref: https://github.com/kubernetes/kubernetes/blob/7f23a743e8c23ac6489340bbb34fa6f1d392db9d/pkg/cloudprovider/providers/openstack/openstack_loadbalancer.go#L1452
func (s *Service) deleteLoadBalancerNeutronV2(id string) error {

	lb, err := loadbalancers.Get(s.loadbalancerClient, id).Extract()
	if err != nil {
		return fmt.Errorf("unable to get loadbalancer: %v", err)
	}

	// get all listeners
	r, err := listeners.List(s.loadbalancerClient, listeners.ListOpts{LoadbalancerID: lb.ID}).AllPages()
	if err != nil {
		return fmt.Errorf("unable to list listeners of loadbalancer %s: %v", lb.ID, err)
	}
	lbListeners, err := listeners.ExtractListeners(r)
	if err != nil {
		return fmt.Errorf("unable to extract listeners: %v", err)
	}

	// get all pools and healthmonitors for this lb
	r, err = pools.List(s.loadbalancerClient, pools.ListOpts{LoadbalancerID: lb.ID}).AllPages()
	if err != nil {
		return fmt.Errorf("unable to list pools for laodbalancer %s: %v", lb.ID, err)
	}
	lbPools, err := pools.ExtractPools(r)
	if err != nil {
		return fmt.Errorf("unable to extract pools for laodbalancer %s: %v", lb.ID, err)
	}

	for _, pool := range lbPools {
		// delete the monitors
		if pool.MonitorID != "" {
			s.logger.Info("Deleting lb monitor", "id", pool.MonitorID)
			err := monitors.Delete(s.loadbalancerClient, pool.MonitorID).ExtractErr()
			if err != nil {
				return fmt.Errorf("error deleting lbaas monitor %s: %v", pool.MonitorID, err)
			}
			if err = waitForLoadBalancerActive(s.logger, s.loadbalancerClient, lb.ID); err != nil {
				return fmt.Errorf("loadbalancer %s did not get back to %s state in time", lb.ID, "Active")
			}
		}

		// get all members of pool
		r, err := pools.ListMembers(s.loadbalancerClient, pool.ID, pools.ListMembersOpts{}).AllPages()
		if err != nil {
			return fmt.Errorf("error listing loadbalancer members of pool %s: %v", pool.ID, err)
		}
		members, err := pools.ExtractMembers(r)
		if err != nil {
			return fmt.Errorf("unable to extract members: %v", err)
		}
		// delete all members of pool
		for _, member := range members {
			s.logger.Info("Deleting lb member", "name", member.Name, "id", member.ID)
			err := pools.DeleteMember(s.loadbalancerClient, pool.ID, member.ID).ExtractErr()
			if err != nil {
				return fmt.Errorf("error deleting lbaas member %s on pool %s: %v", member.ID, pool.ID, err)
			}
			if err = waitForLoadBalancerActive(s.logger, s.loadbalancerClient, lb.ID); err != nil {
				return fmt.Errorf("loadbalancer %s did not get back to %s state in time", lb.ID, "ACTIVE")
			}
		}

		// delete pool
		s.logger.Info("Deleting lb pool", "name", pool.Name, "id", pool.ID)
		err = pools.Delete(s.loadbalancerClient, pool.ID).ExtractErr()
		if err != nil {
			return fmt.Errorf("error deleting lbaas pool %s: %v", pool.ID, err)
		}
		if err = waitForLoadBalancerActive(s.logger, s.loadbalancerClient, lb.ID); err != nil {
			return fmt.Errorf("loadbalancer %s did not get back to %s state in time", lb.ID, "ACTIVE")
		}
	}

	// delete all listeners
	for _, listener := range lbListeners {
		s.logger.Info("Deleting lb listener", "name", listener.Name, "id", listener.ID)
		err = listeners.Delete(s.loadbalancerClient, listener.ID).ExtractErr()
		if err != nil {
			return fmt.Errorf("error deleting lbaas listener %s: %v", listener.ID, err)
		}
		if err = waitForLoadBalancerActive(s.logger, s.loadbalancerClient, lb.ID); err != nil {
			return fmt.Errorf("loadbalancer %s did not get back to %s state in time", lb.ID, "ACTIVE")
		}
	}

	// delete loadbalancer
	s.logger.Info("Deleting loadbalancer", "name", lb.Name, "id", lb.ID)
	if err = loadbalancers.Delete(s.loadbalancerClient, lb.ID, loadbalancers.DeleteOpts{}).ExtractErr(); err != nil {
		return fmt.Errorf("error deleting lbaas %s: %v", lb.ID, err)
	}

	return nil
}

func (s *Service) DeleteLoadBalancerMember(clusterName string, machine *clusterv1.Machine, openStackMachine *infrav1.OpenStackMachine, openStackCluster *infrav1.OpenStackCluster) error {

	if openStackMachine == nil || !util.IsControlPlaneMachine(machine) {
		return nil
	}

	loadBalancerName := fmt.Sprintf("%s-cluster-%s-%s", networkPrefix, clusterName, kubeapiLBSuffix)
	s.logger.Info("Reconciling loadbalancer", "name", loadBalancerName)

	lbID := openStackCluster.Status.Network.APIServerLoadBalancer.ID

	portList := []int{int(openStackCluster.Spec.ControlPlaneEndpoint.Port)}
	portList = append(portList, openStackCluster.Spec.APIServerLoadBalancerAdditionalPorts...)
	for _, port := range portList {
		lbPortObjectsName := fmt.Sprintf("%s-%d", loadBalancerName, port)
		name := lbPortObjectsName + "-" + openStackMachine.Name

		pool, err := checkIfPoolExists(s.loadbalancerClient, lbPortObjectsName)
		if err != nil {
			return err
		}
		if pool == nil {
			s.logger.Info("Pool does not exist", "name", lbPortObjectsName)
			continue
		}

		lbMember, err := checkIfLbMemberExists(s.loadbalancerClient, pool.ID, name)
		if err != nil {
			return err
		}

		if lbMember != nil {

			// lb member changed so let's delete it so we can create it again with the correct IP
			err = waitForLoadBalancerActive(s.logger, s.loadbalancerClient, lbID)
			if err != nil {
				return err
			}
			err = pools.DeleteMember(s.loadbalancerClient, pool.ID, lbMember.ID).ExtractErr()
			if err != nil {
				return fmt.Errorf("error deleting lbmember: %s", err)
			}
			err = waitForLoadBalancerActive(s.logger, s.loadbalancerClient, lbID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func checkIfLbExists(client *gophercloud.ServiceClient, name string) (*loadbalancers.LoadBalancer, error) {
	allPages, err := loadbalancers.List(client, loadbalancers.ListOpts{Name: name}).AllPages()
	if err != nil {
		return nil, err
	}
	lbList, err := loadbalancers.ExtractLoadBalancers(allPages)
	if err != nil {
		return nil, err
	}
	if len(lbList) == 0 {
		return nil, nil
	}
	return &lbList[0], nil
}

func checkIfListenerExists(client *gophercloud.ServiceClient, name string) (*listeners.Listener, error) {
	allPages, err := listeners.List(client, listeners.ListOpts{Name: name}).AllPages()
	if err != nil {
		return nil, err
	}
	listenerList, err := listeners.ExtractListeners(allPages)
	if err != nil {
		return nil, err
	}
	if len(listenerList) == 0 {
		return nil, nil
	}
	return &listenerList[0], nil
}

func checkIfPoolExists(client *gophercloud.ServiceClient, name string) (*pools.Pool, error) {
	allPages, err := pools.List(client, pools.ListOpts{Name: name}).AllPages()
	if err != nil {
		return nil, err
	}
	poolList, err := pools.ExtractPools(allPages)
	if err != nil {
		return nil, err
	}
	if len(poolList) == 0 {
		return nil, nil
	}
	return &poolList[0], nil
}

func checkIfMonitorExists(client *gophercloud.ServiceClient, name string) (*monitors.Monitor, error) {
	allPages, err := monitors.List(client, monitors.ListOpts{Name: name}).AllPages()
	if err != nil {
		return nil, err
	}
	monitorList, err := monitors.ExtractMonitors(allPages)
	if err != nil {
		return nil, err
	}
	if len(monitorList) == 0 {
		return nil, nil
	}
	return &monitorList[0], nil
}

func checkIfLbMemberExists(client *gophercloud.ServiceClient, poolID, name string) (*pools.Member, error) {
	allPages, err := pools.ListMembers(client, poolID, pools.ListMembersOpts{Name: name}).AllPages()
	if err != nil {
		return nil, err
	}
	lbMemberList, err := pools.ExtractMembers(allPages)
	if err != nil {
		return nil, err
	}
	if len(lbMemberList) == 0 {
		return nil, nil
	}
	return &lbMemberList[0], nil
}

var backoff = wait.Backoff{
	Steps:    10,
	Duration: 30 * time.Second,
	Factor:   1.0,
	Jitter:   0.1,
}

// Possible LoadBalancer states are documented here: https://developer.openstack.org/api-ref/network/v2/?expanded=show-load-balancer-status-tree-detail#load-balancer-statuses
func waitForLoadBalancerActive(logger logr.Logger, client *gophercloud.ServiceClient, id string) error {
	logger.Info("Waiting for loadbalancer", "id", id, "targetStatus", "ACTIVE")
	return wait.ExponentialBackoff(backoff, func() (bool, error) {
		lb, err := loadbalancers.Get(client, id).Extract()
		if err != nil {
			return false, err
		}
		return lb.ProvisioningStatus == "ACTIVE", nil
	})
}

func waitForListener(logger logr.Logger, client *gophercloud.ServiceClient, id, target string) error {
	logger.Info("Waiting for listener", "id", id, "targetStatus", target)
	return wait.ExponentialBackoff(backoff, func() (bool, error) {
		_, err := listeners.Get(client, id).Extract()
		if err != nil {
			return false, err
		}
		// The listener resource has no Status attribute, so a successful Get is the best we can do
		return true, nil
	})
}
