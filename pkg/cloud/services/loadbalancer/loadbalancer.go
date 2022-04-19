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

	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/listeners"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/monitors"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/pools"
	"k8s.io/apimachinery/pkg/util/wait"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha5"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/names"
)

const (
	networkPrefix   string = "k8s-clusterapi"
	kubeapiLBSuffix string = "kubeapi"
)

const loadBalancerProvisioningStatusActive = "ACTIVE"

func (s *Service) ReconcileLoadBalancer(openStackCluster *infrav1.OpenStackCluster, clusterName string, apiServerPort int) error {
	loadBalancerName := getLoadBalancerName(clusterName)
	s.scope.Logger.Info("Reconciling load balancer", "name", loadBalancerName)

	var fixedIPAddress string
	switch {
	case openStackCluster.Spec.APIServerFixedIP != "":
		fixedIPAddress = openStackCluster.Spec.APIServerFixedIP
	case openStackCluster.Spec.DisableAPIServerFloatingIP && openStackCluster.Spec.ControlPlaneEndpoint.IsValid():
		fixedIPAddress = openStackCluster.Spec.ControlPlaneEndpoint.Host
	}

	lb, err := s.getOrCreateLoadBalancer(openStackCluster, loadBalancerName, openStackCluster.Status.Network.Subnet.ID, clusterName, fixedIPAddress)
	if err != nil {
		return err
	}
	if lb.ProvisioningStatus != loadBalancerProvisioningStatusActive {
		return fmt.Errorf("load balancer %q is not in expected state %s, current state is %s", lb.ID, loadBalancerProvisioningStatusActive, lb.ProvisioningStatus)
	}

	var lbFloatingIP string
	if !openStackCluster.Spec.DisableAPIServerFloatingIP {
		var floatingIPAddress string
		switch {
		case openStackCluster.Spec.APIServerFloatingIP != "":
			floatingIPAddress = openStackCluster.Spec.APIServerFloatingIP
		case openStackCluster.Spec.ControlPlaneEndpoint.IsValid():
			floatingIPAddress = openStackCluster.Spec.ControlPlaneEndpoint.Host
		}
		fp, err := s.networkingService.GetOrCreateFloatingIP(openStackCluster, openStackCluster, clusterName, floatingIPAddress)
		if err != nil {
			return err
		}
		if err = s.networkingService.AssociateFloatingIP(openStackCluster, fp, lb.VipPortID); err != nil {
			return err
		}
		lbFloatingIP = fp.FloatingIP
	}

	portList := []int{apiServerPort}
	portList = append(portList, openStackCluster.Spec.APIServerLoadBalancer.AdditionalPorts...)
	for _, port := range portList {
		lbPortObjectsName := fmt.Sprintf("%s-%d", loadBalancerName, port)
		listener, err := s.getOrCreateListener(openStackCluster, lbPortObjectsName, lb.ID, port)
		if err != nil {
			return err
		}

		pool, err := s.getOrCreatePool(openStackCluster, lbPortObjectsName, listener.ID, lb.ID)
		if err != nil {
			return err
		}

		if err := s.getOrCreateMonitor(openStackCluster, lbPortObjectsName, pool.ID, lb.ID); err != nil {
			return err
		}
	}

	openStackCluster.Status.Network.APIServerLoadBalancer = &infrav1.LoadBalancer{
		Name:       lb.Name,
		ID:         lb.ID,
		InternalIP: lb.VipAddress,
		IP:         lbFloatingIP,
	}
	return nil
}

func (s *Service) getOrCreateLoadBalancer(openStackCluster *infrav1.OpenStackCluster, loadBalancerName, subnetID, clusterName string, vipAddress string) (*loadbalancers.LoadBalancer, error) {
	lb, err := s.checkIfLbExists(loadBalancerName)
	if err != nil {
		return nil, err
	}

	if lb != nil {
		return lb, nil
	}

	s.scope.Logger.Info(fmt.Sprintf("Creating load balancer in subnet: %q", subnetID), "name", loadBalancerName)

	lbCreateOpts := loadbalancers.CreateOpts{
		Name:        loadBalancerName,
		VipSubnetID: subnetID,
		VipAddress:  vipAddress,
		Description: names.GetDescription(clusterName),
	}
	lb, err = s.loadbalancerClient.CreateLoadBalancer(lbCreateOpts)
	if err != nil {
		record.Warnf(openStackCluster, "FailedCreateLoadBalancer", "Failed to create load balancer %s: %v", loadBalancerName, err)
		return nil, err
	}

	if err := s.waitForLoadBalancerActive(lb.ID); err != nil {
		record.Warnf(openStackCluster, "FailedCreateLoadBalancer", "Failed to create load balancer %s with id %s: wait for load balancer active: %v", loadBalancerName, lb.ID, err)
		return nil, err
	}

	record.Eventf(openStackCluster, "SuccessfulCreateLoadBalancer", "Created load balancer %s with id %s", loadBalancerName, lb.ID)
	return lb, nil
}

func (s *Service) getOrCreateListener(openStackCluster *infrav1.OpenStackCluster, listenerName, lbID string, port int) (*listeners.Listener, error) {
	listener, err := s.checkIfListenerExists(listenerName)
	if err != nil {
		return nil, err
	}

	if listener != nil {
		return listener, nil
	}

	s.scope.Logger.Info("Creating load balancer listener", "name", listenerName, "lb-id", lbID)

	listenerCreateOpts := listeners.CreateOpts{
		Name:           listenerName,
		Protocol:       "TCP",
		ProtocolPort:   port,
		LoadbalancerID: lbID,
	}
	listener, err = s.loadbalancerClient.CreateListener(listenerCreateOpts)
	if err != nil {
		record.Warnf(openStackCluster, "FailedCreateListener", "Failed to create listener %s: %v", listenerName, err)
		return nil, err
	}

	if err := s.waitForLoadBalancerActive(lbID); err != nil {
		record.Warnf(openStackCluster, "FailedCreateListener", "Failed to create listener %s with id %s: wait for load balancer active %s: %v", listenerName, listener.ID, lbID, err)
		return nil, err
	}

	if err := s.waitForListener(listener.ID, "ACTIVE"); err != nil {
		record.Warnf(openStackCluster, "FailedCreateListener", "Failed to create listener %s with id %s: wait for listener active: %v", listenerName, listener.ID, err)
		return nil, err
	}

	record.Eventf(openStackCluster, "SuccessfulCreateListener", "Created listener %s with id %s", listenerName, listener.ID)
	return listener, nil
}

func (s *Service) getOrCreatePool(openStackCluster *infrav1.OpenStackCluster, poolName, listenerID, lbID string) (*pools.Pool, error) {
	pool, err := s.checkIfPoolExists(poolName)
	if err != nil {
		return nil, err
	}

	if pool != nil {
		return pool, nil
	}

	s.scope.Logger.Info(fmt.Sprintf("Creating load balancer pool for listener %q", listenerID), "name", poolName, "lb-id", lbID)

	poolCreateOpts := pools.CreateOpts{
		Name:       poolName,
		Protocol:   "TCP",
		LBMethod:   pools.LBMethodRoundRobin,
		ListenerID: listenerID,
	}
	pool, err = s.loadbalancerClient.CreatePool(poolCreateOpts)
	if err != nil {
		record.Warnf(openStackCluster, "FailedCreatePool", "Failed to create pool %s: %v", poolName, err)
		return nil, err
	}

	if err := s.waitForLoadBalancerActive(lbID); err != nil {
		record.Warnf(openStackCluster, "FailedCreatePool", "Failed to create pool %s with id %s: wait for load balancer active %s: %v", poolName, pool.ID, lbID, err)
		return nil, err
	}

	record.Eventf(openStackCluster, "SuccessfulCreatePool", "Created pool %s with id %s", poolName, pool.ID)
	return pool, nil
}

func (s *Service) getOrCreateMonitor(openStackCluster *infrav1.OpenStackCluster, monitorName, poolID, lbID string) error {
	monitor, err := s.checkIfMonitorExists(monitorName)
	if err != nil {
		return err
	}

	if monitor != nil {
		return nil
	}

	s.scope.Logger.Info(fmt.Sprintf("Creating load balancer monitor for pool %q", poolID), "name", monitorName, "lb-id", lbID)

	monitorCreateOpts := monitors.CreateOpts{
		Name:       monitorName,
		PoolID:     poolID,
		Type:       "TCP",
		Delay:      30,
		Timeout:    5,
		MaxRetries: 3,
	}
	monitor, err = s.loadbalancerClient.CreateMonitor(monitorCreateOpts)
	if err != nil {
		record.Warnf(openStackCluster, "FailedCreateMonitor", "Failed to create monitor %s: %v", monitorName, err)
		return err
	}

	if err = s.waitForLoadBalancerActive(lbID); err != nil {
		record.Warnf(openStackCluster, "FailedCreateMonitor", "Failed to create monitor %s with id %s: wait for load balancer active %s: %v", monitorName, monitor.ID, lbID, err)
		return err
	}

	record.Eventf(openStackCluster, "SuccessfulCreateMonitor", "Created monitor %s with id %s", monitorName, monitor.ID)
	return nil
}

func (s *Service) ReconcileLoadBalancerMember(openStackCluster *infrav1.OpenStackCluster, machine *clusterv1.Machine, openStackMachine *infrav1.OpenStackMachine, clusterName, ip string) error {
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

	loadBalancerName := getLoadBalancerName(clusterName)
	s.scope.Logger.Info("Reconciling load balancer member", "name", loadBalancerName)

	lbID := openStackCluster.Status.Network.APIServerLoadBalancer.ID
	portList := []int{int(openStackCluster.Spec.ControlPlaneEndpoint.Port)}
	portList = append(portList, openStackCluster.Spec.APIServerLoadBalancer.AdditionalPorts...)
	for _, port := range portList {
		lbPortObjectsName := fmt.Sprintf("%s-%d", loadBalancerName, port)
		name := lbPortObjectsName + "-" + openStackMachine.Name

		pool, err := s.checkIfPoolExists(lbPortObjectsName)
		if err != nil {
			return err
		}
		if pool == nil {
			return errors.New("load balancer pool does not exist yet")
		}

		lbMember, err := s.checkIfLbMemberExists(pool.ID, name)
		if err != nil {
			return err
		}

		if lbMember != nil {
			// check if we have to recreate the LB Member
			if lbMember.Address == ip {
				// nothing to do continue to next port
				continue
			}

			s.scope.Logger.Info("Deleting load balancer member (because the IP of the machine changed)", "name", name)

			// lb member changed so let's delete it so we can create it again with the correct IP
			err = s.waitForLoadBalancerActive(lbID)
			if err != nil {
				return err
			}
			if err := s.loadbalancerClient.DeletePoolMember(pool.ID, lbMember.ID); err != nil {
				return err
			}
			err = s.waitForLoadBalancerActive(lbID)
			if err != nil {
				return err
			}
		}

		s.scope.Logger.Info("Creating load balancer member", "name", name)

		// if we got to this point we should either create or re-create the lb member
		lbMemberOpts := pools.CreateMemberOpts{
			Name:         name,
			ProtocolPort: port,
			Address:      ip,
		}

		if err := s.waitForLoadBalancerActive(lbID); err != nil {
			return err
		}

		if _, err := s.loadbalancerClient.CreatePoolMember(pool.ID, lbMemberOpts); err != nil {
			return err
		}

		if err := s.waitForLoadBalancerActive(lbID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) DeleteLoadBalancer(openStackCluster *infrav1.OpenStackCluster, clusterName string) error {
	loadBalancerName := getLoadBalancerName(clusterName)
	lb, err := s.checkIfLbExists(loadBalancerName)
	if err != nil {
		return err
	}

	if lb == nil {
		return nil
	}

	if lb.VipPortID != "" {
		fip, err := s.networkingService.GetFloatingIPByPortID(lb.VipPortID)
		if err != nil {
			return err
		}

		if fip != nil && fip.FloatingIP != "" {
			if err = s.networkingService.DisassociateFloatingIP(openStackCluster, fip.FloatingIP); err != nil {
				return err
			}
			if err = s.networkingService.DeleteFloatingIP(openStackCluster, fip.FloatingIP); err != nil {
				return err
			}
		}
	}

	deleteOpts := loadbalancers.DeleteOpts{
		Cascade: true,
	}
	s.scope.Logger.Info("Deleting load balancer", "name", loadBalancerName, "cascade", deleteOpts.Cascade)
	err = s.loadbalancerClient.DeleteLoadBalancer(lb.ID, deleteOpts)
	if err != nil && !capoerrors.IsNotFound(err) {
		record.Warnf(openStackCluster, "FailedDeleteLoadBalancer", "Failed to delete load balancer %s with id %s: %v", lb.Name, lb.ID, err)
		return err
	}

	record.Eventf(openStackCluster, "SuccessfulDeleteLoadBalancer", "Deleted load balancer %s with id %s", lb.Name, lb.ID)
	return nil
}

func (s *Service) DeleteLoadBalancerMember(openStackCluster *infrav1.OpenStackCluster, machine *clusterv1.Machine, openStackMachine *infrav1.OpenStackMachine, clusterName string) error {
	if openStackMachine == nil || !util.IsControlPlaneMachine(machine) {
		return nil
	}

	loadBalancerName := getLoadBalancerName(clusterName)
	lb, err := s.checkIfLbExists(loadBalancerName)
	if err != nil {
		return err
	}
	if lb == nil {
		// nothing to do
		return nil
	}

	lbID := lb.ID

	portList := []int{int(openStackCluster.Spec.ControlPlaneEndpoint.Port)}
	portList = append(portList, openStackCluster.Spec.APIServerLoadBalancer.AdditionalPorts...)
	for _, port := range portList {
		lbPortObjectsName := fmt.Sprintf("%s-%d", loadBalancerName, port)
		name := lbPortObjectsName + "-" + openStackMachine.Name

		pool, err := s.checkIfPoolExists(lbPortObjectsName)
		if err != nil {
			return err
		}
		if pool == nil {
			s.scope.Logger.Info("Load balancer pool does not exist", "name", lbPortObjectsName)
			continue
		}

		lbMember, err := s.checkIfLbMemberExists(pool.ID, name)
		if err != nil {
			return err
		}

		if lbMember != nil {
			// lb member changed so let's delete it so we can create it again with the correct IP
			err = s.waitForLoadBalancerActive(lbID)
			if err != nil {
				return err
			}
			if err := s.loadbalancerClient.DeletePoolMember(pool.ID, lbMember.ID); err != nil {
				return err
			}
			err = s.waitForLoadBalancerActive(lbID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getLoadBalancerName(clusterName string) string {
	return fmt.Sprintf("%s-cluster-%s-%s", networkPrefix, clusterName, kubeapiLBSuffix)
}

func (s *Service) checkIfLbExists(name string) (*loadbalancers.LoadBalancer, error) {
	lbList, err := s.loadbalancerClient.ListLoadBalancers(loadbalancers.ListOpts{Name: name})
	if err != nil {
		return nil, err
	}
	if len(lbList) == 0 {
		return nil, nil
	}
	return &lbList[0], nil
}

func (s *Service) checkIfListenerExists(name string) (*listeners.Listener, error) {
	listenerList, err := s.loadbalancerClient.ListListeners(listeners.ListOpts{Name: name})
	if err != nil {
		return nil, err
	}
	if len(listenerList) == 0 {
		return nil, nil
	}
	return &listenerList[0], nil
}

func (s *Service) checkIfPoolExists(name string) (*pools.Pool, error) {
	poolList, err := s.loadbalancerClient.ListPools(pools.ListOpts{Name: name})
	if err != nil {
		return nil, err
	}
	if len(poolList) == 0 {
		return nil, nil
	}
	return &poolList[0], nil
}

func (s *Service) checkIfMonitorExists(name string) (*monitors.Monitor, error) {
	monitorList, err := s.loadbalancerClient.ListMonitors(monitors.ListOpts{Name: name})
	if err != nil {
		return nil, err
	}
	if len(monitorList) == 0 {
		return nil, nil
	}
	return &monitorList[0], nil
}

func (s *Service) checkIfLbMemberExists(poolID, name string) (*pools.Member, error) {
	lbMemberList, err := s.loadbalancerClient.ListPoolMember(poolID, pools.ListMembersOpts{Name: name})
	if err != nil {
		return nil, err
	}
	if len(lbMemberList) == 0 {
		return nil, nil
	}
	return &lbMemberList[0], nil
}

var backoff = wait.Backoff{
	Steps:    20,
	Duration: time.Second,
	Factor:   1.25,
	Jitter:   0.1,
}

// Possible LoadBalancer states are documented here: https://docs.openstack.org/api-ref/load-balancer/v2/index.html#prov-status
func (s *Service) waitForLoadBalancerActive(id string) error {
	s.scope.Logger.Info("Waiting for load balancer", "id", id, "targetStatus", "ACTIVE")
	return wait.ExponentialBackoff(backoff, func() (bool, error) {
		lb, err := s.loadbalancerClient.GetLoadBalancer(id)
		if err != nil {
			return false, err
		}
		return lb.ProvisioningStatus == loadBalancerProvisioningStatusActive, nil
	})
}

func (s *Service) waitForListener(id, target string) error {
	s.scope.Logger.Info("Waiting for load balancer listener", "id", id, "targetStatus", target)
	return wait.ExponentialBackoff(backoff, func() (bool, error) {
		_, err := s.loadbalancerClient.GetListener(id)
		if err != nil {
			return false, err
		}
		// The listener resource has no Status attribute, so a successful Get is the best we can do
		return true, nil
	})
}
