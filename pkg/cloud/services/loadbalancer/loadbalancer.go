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
	"cmp"
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/listeners"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/monitors"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/pools"
	"k8s.io/apimachinery/pkg/util/wait"
	utilsnet "k8s.io/utils/net"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/names"
	openstackutil "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/openstack"
	capostrings "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/strings"
)

const (
	networkPrefix           string = "k8s-clusterapi"
	kubeapiLBSuffix         string = "kubeapi"
	resolvedMsg             string = "ControlPlaneEndpoint.Host is not an IP address, using the first resolved IP address"
	waitForOctaviaLBCleanup        = 15 * time.Second
)

const (
	loadBalancerProvisioningStatusActive        = "ACTIVE"
	loadBalancerProvisioningStatusPendingDelete = "PENDING_DELETE"
)

// Default values for Monitor, sync with `kubebuilder:default` annotations on APIServerLoadBalancerMonitor object.
const (
	defaultMonitorDelay          = 10
	defaultMonitorTimeout        = 5
	defaultMonitorMaxRetries     = 5
	defaultMonitorMaxRetriesDown = 3
)

// Per-AZ reconciliation helper.
type azSubnet struct {
	az     *string
	subnet infrav1.Subnet
}

// We wrap the LookupHost function in a variable to allow overriding it in unit tests.
//
//nolint:gocritic
var lookupHost = func(host string) (*string, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return nil, err
	}
	if ip := net.ParseIP(ips[0]); ip == nil {
		return nil, fmt.Errorf("failed to resolve IP address for host %s", host)
	}
	return &ips[0], nil
}

// ReconcileLoadBalancers reconciles one load balancer for each APIServer LoadBalancer AvailabilityZone.
func (s *Service) ReconcileLoadBalancers(openStackCluster *infrav1.OpenStackCluster, clusterResourceName string, apiServerPort int) (bool, error) {
	if openStackCluster.Spec.APIServerLoadBalancer == nil {
		return false, nil
	}

	// Use the availability zone from the load balancer spec if it is set
	if openStackCluster.Spec.APIServerLoadBalancer.AvailabilityZone != nil &&
		openStackCluster.Spec.APIServerLoadBalancer.AvailabilityZones == nil {
		openStackCluster.Spec.APIServerLoadBalancer.AvailabilityZones = []string{
			*openStackCluster.Spec.APIServerLoadBalancer.AvailabilityZone,
		}
	}

	// Ensure API server load balancer network information is available
	if openStackCluster.Status.APIServerLoadBalancer == nil ||
		openStackCluster.Status.APIServerLoadBalancer.LoadBalancerNetwork == nil ||
		len(openStackCluster.Status.APIServerLoadBalancer.LoadBalancerNetwork.Subnets) == 0 {
		return false, fmt.Errorf("load balancer network information not available")
	}

	// Convert the availability zones to azSubnet structs
	// Prefer explicit AZ->Subnet mappings when provided; otherwise fallback to positional mapping.
	var azInfo []azSubnet
	lbStatus := openStackCluster.Status.APIServerLoadBalancer
	lbSpec := openStackCluster.Spec.APIServerLoadBalancer

	if lbStatus == nil || lbStatus.LoadBalancerNetwork == nil || len(lbStatus.LoadBalancerNetwork.Subnets) == 0 {
		return false, fmt.Errorf("load balancer network information not available")
	}

	if len(lbSpec.AvailabilityZoneSubnets) > 0 {
		// Controller resolves and orders LoadBalancerNetwork.Subnets to match AvailabilityZoneSubnets,
		// so we can index by the same position here.
		if len(lbSpec.AvailabilityZoneSubnets) > len(lbStatus.LoadBalancerNetwork.Subnets) {
			return false, fmt.Errorf("mismatch between availabilityZoneSubnets and resolved subnets: more mappings than subnets")
		}
		for i, m := range lbSpec.AvailabilityZoneSubnets {
			az := m.AvailabilityZone
			azInfo = append(azInfo, azSubnet{
				az:     &az,
				subnet: lbStatus.LoadBalancerNetwork.Subnets[i],
			})
		}
	} else if len(lbSpec.AvailabilityZones) == 0 {
		// Single-AZ default
		defaultAZ := "default"
		azInfo = append(azInfo, azSubnet{
			az:     &defaultAZ,
			subnet: lbStatus.LoadBalancerNetwork.Subnets[0],
		})
		s.scope.Logger().Info("No availability zones specified, using default")
	} else {
		// Positional fallback: AvailabilityZones[i] maps to Subnets[i]
		for i, az := range lbSpec.AvailabilityZones {
			if i >= len(lbStatus.LoadBalancerNetwork.Subnets) {
				return false, fmt.Errorf("mismatch between availability zones and subnets: more AZs than subnets")
			}
			// capture loop variable
			azCopy := az
			azInfo = append(azInfo, azSubnet{
				az:     &azCopy,
				subnet: lbStatus.LoadBalancerNetwork.Subnets[i],
			})
		}
	}

	// Always migrate the original load balancer to the AZ-named format
	if err := s.migrateAPIServerLoadBalancer(openStackCluster, clusterResourceName, azInfo[0], apiServerPort); err != nil {
		return false, fmt.Errorf("failed to migrate load balancer: %w", err)
	}

	// Reconcile the load balancer for each availability zone
	requeue := false
	for _, az := range azInfo {
		azClusterResourceName := fmt.Sprintf("%s-%s", clusterResourceName, *az.az)
		s.scope.Logger().Info("Reconciling load balancer for availability zone",
			"az", *az.az,
			"resourceName", azClusterResourceName)

		zoneRequeue, err := s.ReconcileLoadBalancer(openStackCluster, azClusterResourceName, az, apiServerPort)
		if err != nil {
			return zoneRequeue, err
		}
		requeue = requeue || zoneRequeue
	}

	return requeue, nil
}

// ReconcileLoadBalancer reconciles the load balancer for the given cluster.
func (s *Service) ReconcileLoadBalancer(openStackCluster *infrav1.OpenStackCluster, clusterResourceName string, azInfo azSubnet, apiServerPort int) (bool, error) {
	lbSpec := openStackCluster.Spec.APIServerLoadBalancer
	if !lbSpec.IsEnabled() {
		return false, nil
	}

	loadBalancerName := getLoadBalancerName(clusterResourceName)
	s.scope.Logger().Info("Reconciling load balancer", "name", loadBalancerName)

	lbStatus := openStackCluster.Status.APIServerLoadBalancer
	if lbStatus == nil {
		lbStatus = &infrav1.LoadBalancer{}
		openStackCluster.Status.APIServerLoadBalancer = lbStatus
	}

	lb, err := s.getOrCreateAPILoadBalancer(openStackCluster, clusterResourceName, azInfo.az, azInfo.subnet.ID)
	if err != nil {
		if errors.Is(err, capoerrors.ErrFilterMatch) {
			return true, err
		}
		return false, err
	}

	lbStatus.Name = lb.Name
	lbStatus.ID = lb.ID
	lbStatus.InternalIP = lb.VipAddress
	lbStatus.Tags = lb.Tags
	lbStatus.AvailabilityZone = ptr.Deref(azInfo.az, "")

	// Update the multi-AZ load balancer list
	s.updateMultiAZLoadBalancerStatus(openStackCluster, lb, ptr.Deref(azInfo.az, ""))

	if lb.ProvisioningStatus != loadBalancerProvisioningStatusActive {
		var err error
		lbID := lb.ID
		lb, err = s.waitForLoadBalancerActive(lbID)
		if err != nil {
			return false, fmt.Errorf("load balancer %q with id %s is not active after timeout: %v", loadBalancerName, lbID, err)
		}
	}

	if !ptr.Deref(openStackCluster.Spec.DisableAPIServerFloatingIP, false) {
		floatingIPAddress, err := getAPIServerFloatingIP(openStackCluster)
		if err != nil {
			return false, err
		}

		fp, err := s.networkingService.GetOrCreateFloatingIP(openStackCluster, openStackCluster, clusterResourceName, floatingIPAddress)
		if err != nil {
			if errors.Is(err, capoerrors.ErrFilterMatch) {
				return true, err
			}
			return false, err
		}

		// Write the floating IP to the status immediately so we won't
		// create a new floating IP on the next reconcile if something
		// fails below.
		lbStatus.IP = fp.FloatingIP

		// Also update the floating IP in the multi-AZ load balancer list
		s.updateMultiAZLoadBalancerFloatingIP(openStackCluster, ptr.Deref(azInfo.az, ""), fp.FloatingIP)

		if err = s.networkingService.AssociateFloatingIP(openStackCluster, fp, lb.VipPortID); err != nil {
			return false, err
		}
	}

	allowedCIDRsSupported, err := s.isAllowsCIDRSSupported(lb)
	if err != nil {
		return false, err
	}

	// AllowedCIDRs will be nil if allowed CIDRs is not supported by the Octavia provider
	if allowedCIDRsSupported {
		lbStatus.AllowedCIDRs = getCanonicalAllowedCIDRs(openStackCluster)
		// Also update the allowed CIDRs in the multi-AZ load balancer list
		s.updateMultiAZLoadBalancerAllowedCIDRs(openStackCluster, ptr.Deref(azInfo.az, ""), lbStatus.AllowedCIDRs)
	} else {
		lbStatus.AllowedCIDRs = nil
		// Also update the allowed CIDRs in the multi-AZ load balancer list
		s.updateMultiAZLoadBalancerAllowedCIDRs(openStackCluster, ptr.Deref(azInfo.az, ""), nil)
	}

	// Update the load balancer network information in the multi-AZ list
	if lbStatus.LoadBalancerNetwork != nil {
		s.updateMultiAZLoadBalancerNetwork(openStackCluster, ptr.Deref(azInfo.az, ""), lbStatus.LoadBalancerNetwork)
	}

	portList := []int{apiServerPort}
	portList = append(portList, lbSpec.AdditionalPorts...)
	for _, port := range portList {
		if err := s.reconcileAPILoadBalancerListener(lb, openStackCluster, clusterResourceName, port); err != nil {
			return false, err
		}
	}

	return false, nil
}

// getAPIServerVIPAddress gets the VIP address for the API server from wherever it is specified.
// Returns an empty string if the VIP address is not specified and it should be allocated automatically.
func getAPIServerVIPAddress(openStackCluster *infrav1.OpenStackCluster) (*string, error) {
	switch {
	// We only use call this function when creating the loadbalancer, so this case should never be used
	case openStackCluster.Status.APIServerLoadBalancer != nil && openStackCluster.Status.APIServerLoadBalancer.InternalIP != "":
		return &openStackCluster.Status.APIServerLoadBalancer.InternalIP, nil

	// Explicit fixed IP in the cluster spec
	case openStackCluster.Spec.APIServerFixedIP != nil:
		return openStackCluster.Spec.APIServerFixedIP, nil

	// If we are using the VIP as the control plane endpoint, use any value explicitly set on the control plane endpoint
	case ptr.Deref(openStackCluster.Spec.DisableAPIServerFloatingIP, false) && openStackCluster.Spec.ControlPlaneEndpoint != nil && openStackCluster.Spec.ControlPlaneEndpoint.IsValid():
		fixedIPAddress, err := lookupHost(openStackCluster.Spec.ControlPlaneEndpoint.Host)
		if err != nil {
			return nil, fmt.Errorf("lookup host: %w", err)
		}
		return fixedIPAddress, nil
	}

	return nil, nil
}

// getAPIServerFloatingIP gets the floating IP from wherever it is specified.
// Returns an empty string if the floating IP is not specified and it should be allocated automatically.
func getAPIServerFloatingIP(openStackCluster *infrav1.OpenStackCluster) (*string, error) {
	switch {
	// The floating IP was created previously
	case openStackCluster.Status.APIServerLoadBalancer != nil && openStackCluster.Status.APIServerLoadBalancer.IP != "":
		return &openStackCluster.Status.APIServerLoadBalancer.IP, nil

	// Explicit floating IP in the cluster spec
	case openStackCluster.Spec.APIServerFloatingIP != nil:
		return openStackCluster.Spec.APIServerFloatingIP, nil

	// An IP address is specified explicitly in the control plane endpoint
	case openStackCluster.Spec.ControlPlaneEndpoint != nil && openStackCluster.Spec.ControlPlaneEndpoint.IsValid():
		floatingIPAddress, err := lookupHost(openStackCluster.Spec.ControlPlaneEndpoint.Host)
		if err != nil {
			return nil, fmt.Errorf("lookup host: %w", err)
		}
		return floatingIPAddress, nil
	}

	return nil, nil
}

// getCanonicalAllowedCIDRs gets a filtered list of CIDRs which should be allowed to access the API server loadbalancer.
// Invalid CIDRs are filtered from the list and emil a warning event.
// It returns a canonical representation that can be directly compared with other canonicalized lists.
func getCanonicalAllowedCIDRs(openStackCluster *infrav1.OpenStackCluster) []string {
	allowedCIDRs := []string{}

	if openStackCluster.Spec.APIServerLoadBalancer != nil && len(openStackCluster.Spec.APIServerLoadBalancer.AllowedCIDRs) > 0 {
		allowedCIDRs = append(allowedCIDRs, openStackCluster.Spec.APIServerLoadBalancer.AllowedCIDRs...)

		// In the first reconciliation loop, only the Ready field is set in openStackCluster.Status
		// All other fields are empty/nil
		if openStackCluster.Status.Bastion != nil {
			if openStackCluster.Status.Bastion.FloatingIP != "" {
				allowedCIDRs = append(allowedCIDRs, openStackCluster.Status.Bastion.FloatingIP)
			}

			if openStackCluster.Status.Bastion.IP != "" {
				allowedCIDRs = append(allowedCIDRs, openStackCluster.Status.Bastion.IP)
			}
		}

		if openStackCluster.Status.Network != nil {
			for _, subnet := range openStackCluster.Status.Network.Subnets {
				if subnet.CIDR != "" {
					allowedCIDRs = append(allowedCIDRs, subnet.CIDR)
				}
			}

			if openStackCluster.Status.Router != nil && len(openStackCluster.Status.Router.IPs) > 0 {
				allowedCIDRs = append(allowedCIDRs, openStackCluster.Status.Router.IPs...)
			}
		}
	}

	// Filter invalid CIDRs and convert any IPs into CIDRs.
	validCIDRs := []string{}
	for _, v := range allowedCIDRs {
		switch {
		case utilsnet.IsIPv4String(v):
			validCIDRs = append(validCIDRs, v+"/32")
		case utilsnet.IsIPv4CIDRString(v):
			validCIDRs = append(validCIDRs, v)
		default:
			record.Warnf(openStackCluster, "FailedIPAddressValidation", "%s is not a valid IPv4 nor CIDR address and will not get applied to allowed_cidrs", v)
		}
	}

	// Sort and remove duplicates
	return capostrings.Canonicalize(validCIDRs)
}

// isAllowsCIDRSSupported returns true if Octavia supports allowed CIDRs for the loadbalancer provider in use.
func (s *Service) isAllowsCIDRSSupported(lb *loadbalancers.LoadBalancer) (bool, error) {
	octaviaVersions, err := s.loadbalancerClient.ListOctaviaVersions()
	if err != nil {
		return false, err
	}
	// The current version is always the last one in the list.
	octaviaVersion := octaviaVersions[len(octaviaVersions)-1].ID

	return openstackutil.IsOctaviaFeatureSupported(octaviaVersion, openstackutil.OctaviaFeatureVIPACL, lb.Provider), nil
}

// getOrCreateAPILoadBalancer returns an existing API loadbalancer if it already exists, or creates a new one if it does not.
func (s *Service) getOrCreateAPILoadBalancer(openStackCluster *infrav1.OpenStackCluster, clusterResourceName string, availabilityZone *string, vipSubnetIDOverride string) (*loadbalancers.LoadBalancer, error) {
	loadBalancerName := getLoadBalancerName(clusterResourceName)
	lb, err := s.checkIfLbExists(loadBalancerName)
	if err != nil {
		return nil, err
	}
	if lb != nil {
		return lb, nil
	}

	if openStackCluster.Status.Network == nil {
		return nil, fmt.Errorf("network is not yet available in OpenStackCluster.Status")
	}

	if openStackCluster.Status.APIServerLoadBalancer == nil {
		return nil, fmt.Errorf("apiserver loadbalancer network is not yet available in OpenStackCluster.Status")
	}

	lbNetwork := openStackCluster.Status.APIServerLoadBalancer.LoadBalancerNetwork
	if lbNetwork == nil {
		lbNetwork = &infrav1.NetworkStatusWithSubnets{}
		openStackCluster.Status.APIServerLoadBalancer.LoadBalancerNetwork = lbNetwork
	}

	var vipNetworkID, vipSubnetID string
	if lbNetwork.ID != "" {
		vipNetworkID = lbNetwork.ID
	}

	// Prefer the caller-provided VIP subnet ID (per-AZ mapping), otherwise fall back to the first subnet
	// from the LB network, and finally the first subnet of the cluster network.
	if vipSubnetIDOverride != "" {
		vipSubnetID = vipSubnetIDOverride
	} else if len(lbNetwork.Subnets) > 0 {
		vipSubnetID = lbNetwork.Subnets[0].ID
	}

	if vipNetworkID == "" && vipSubnetID == "" {
		// keep the default and create the VIP on the first cluster subnet
		vipSubnetID = openStackCluster.Status.Network.Subnets[0].ID
		s.scope.Logger().Info("No load balancer network specified, creating load balancer in the default subnet", "subnetID", vipSubnetID, "name", loadBalancerName)
	} else {
		s.scope.Logger().Info("Creating load balancer in subnet", "subnetID", vipSubnetID, "name", loadBalancerName)
	}

	providers, err := s.loadbalancerClient.ListLoadBalancerProviders()
	if err != nil {
		return nil, err
	}

	// Choose the selected provider and flavor if set in cluster spec, if not, omit these fields and Octavia will use the default values.
	lbProvider := ""
	lbFlavorID := ""
	if openStackCluster.Spec.APIServerLoadBalancer != nil {
		if openStackCluster.Spec.APIServerLoadBalancer.Provider != nil {
			for _, v := range providers {
				if v.Name == *openStackCluster.Spec.APIServerLoadBalancer.Provider {
					lbProvider = v.Name
					break
				}
			}
			if lbProvider == "" {
				record.Warnf(openStackCluster, "OctaviaProviderNotFound", "Provider specified for Octavia not found.")
				record.Eventf(openStackCluster, "OctaviaProviderNotFound", "Provider %s specified for Octavia not found, using the default provider.", openStackCluster.Spec.APIServerLoadBalancer.Provider)
			}
		}
		if openStackCluster.Spec.APIServerLoadBalancer.Flavor != nil {
			// Gophercloud does not support filtering loadbalancer flavors by name and status (enabled) so we have to get all available flavors
			// and filter them localy. There is a feature request in Gophercloud to implement this functionality:
			// https://github.com/gophercloud/gophercloud/v2/issues/3049
			flavors, err := s.loadbalancerClient.ListLoadBalancerFlavors()
			if err != nil {
				return nil, err
			}

			for _, v := range flavors {
				if v.Enabled && v.Name == *openStackCluster.Spec.APIServerLoadBalancer.Flavor {
					lbFlavorID = v.ID
					break
				}
			}
			if lbFlavorID == "" {
				record.Warnf(openStackCluster, "OctaviaFlavorNotFound", "Flavor %s specified for Octavia not found, using the default flavor.", *openStackCluster.Spec.APIServerLoadBalancer.Flavor)
			}
		}
	}

	vipAddress, err := getAPIServerVIPAddress(openStackCluster)
	if err != nil {
		return nil, err
	}

	lbCreateOpts := loadbalancers.CreateOpts{
		Name:         loadBalancerName,
		VipSubnetID:  vipSubnetID,
		VipNetworkID: vipNetworkID,
		Description:  names.GetDescription(clusterResourceName),
		Provider:     lbProvider,
		FlavorID:     lbFlavorID,
		Tags:         openStackCluster.Spec.Tags,
	}
	if availabilityZone != nil {
		lbCreateOpts.AvailabilityZone = *availabilityZone
	}
	if vipAddress != nil {
		lbCreateOpts.VipAddress = *vipAddress
	}

	lb, err = s.loadbalancerClient.CreateLoadBalancer(lbCreateOpts)
	if err != nil {
		record.Warnf(openStackCluster, "FailedCreateLoadBalancer", "Failed to create load balancer %s: %v", loadBalancerName, err)
		return nil, err
	}

	record.Eventf(openStackCluster, "SuccessfulCreateLoadBalancer", "Created load balancer %s with id %s", loadBalancerName, lb.ID)
	return lb, nil
}

// reconcileAPILoadBalancerListener ensures that the listener on the given port exists and is configured correctly.
func (s *Service) reconcileAPILoadBalancerListener(lb *loadbalancers.LoadBalancer, openStackCluster *infrav1.OpenStackCluster, clusterResourceName string, port int) error {
	loadBalancerName := getLoadBalancerName(clusterResourceName)
	lbPortObjectsName := fmt.Sprintf("%s-%d", loadBalancerName, port)

	if openStackCluster.Status.APIServerLoadBalancer == nil {
		return fmt.Errorf("APIServerLoadBalancer is not yet available in OpenStackCluster.Status")
	}

	allowedCIDRs := openStackCluster.Status.APIServerLoadBalancer.AllowedCIDRs

	listener, err := s.getOrCreateListener(openStackCluster, lbPortObjectsName, lb.ID, allowedCIDRs, port)
	if err != nil {
		return err
	}

	pool, err := s.getOrCreatePool(openStackCluster, lbPortObjectsName, listener.ID, lb.ID, lb.Provider)
	if err != nil {
		return err
	}
	if err := s.ensureMonitor(openStackCluster, lbPortObjectsName, pool.ID, lb.ID); err != nil {
		return err
	}

	// allowedCIDRs is nil if allowedCIDRs is not supported by the Octavia provider
	// A non-nil empty slice is an explicitly empty list
	if allowedCIDRs != nil {
		if err := s.getOrUpdateAllowedCIDRs(openStackCluster, listener, allowedCIDRs); err != nil {
			return err
		}
	}

	return nil
}

// getOrCreateListener returns an existing listener for the given loadbalancer
// and port if it already exists, or creates a new one if it does not.
func (s *Service) getOrCreateListener(openStackCluster *infrav1.OpenStackCluster, listenerName, lbID string, allowedCIDRs []string, port int) (*listeners.Listener, error) {
	listener, err := s.checkIfListenerExists(listenerName)
	if err != nil {
		return nil, err
	}

	if listener != nil {
		return listener, nil
	}

	s.scope.Logger().Info("Creating load balancer listener", "name", listenerName, "loadBalancerID", lbID)

	listenerCreateOpts := listeners.CreateOpts{
		Name:           listenerName,
		Protocol:       "TCP",
		ProtocolPort:   port,
		LoadbalancerID: lbID,
		Tags:           openStackCluster.Spec.Tags,
		AllowedCIDRs:   allowedCIDRs,
	}
	listener, err = s.loadbalancerClient.CreateListener(listenerCreateOpts)
	if err != nil {
		record.Warnf(openStackCluster, "FailedCreateListener", "Failed to create listener %s: %v", listenerName, err)
		return nil, err
	}

	if _, err := s.waitForLoadBalancerActive(lbID); err != nil {
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

// getOrUpdateAllowedCIDRs ensures that the allowed CIDRs configured on a listener correspond to the expected list.
func (s *Service) getOrUpdateAllowedCIDRs(openStackCluster *infrav1.OpenStackCluster, listener *listeners.Listener, allowedCIDRs []string) error {
	// Sort and remove duplicates
	listener.AllowedCIDRs = capostrings.Canonicalize(listener.AllowedCIDRs)

	if !slices.Equal(allowedCIDRs, listener.AllowedCIDRs) {
		s.scope.Logger().Info("CIDRs do not match, updating listener", "expectedCIDRs", allowedCIDRs, "currentCIDRs", listener.AllowedCIDRs)
		listenerUpdateOpts := listeners.UpdateOpts{
			AllowedCIDRs: &allowedCIDRs,
		}

		listenerID := listener.ID
		listener, err := s.loadbalancerClient.UpdateListener(listener.ID, listenerUpdateOpts)
		if err != nil {
			record.Warnf(openStackCluster, "FailedUpdateListener", "Failed to update listener %s: %v", listenerID, err)
			return err
		}

		if err := s.waitForListener(listener.ID, "ACTIVE"); err != nil {
			record.Warnf(openStackCluster, "FailedUpdateListener", "Failed to update listener %s with id %s: wait for listener active: %v", listener.Name, listener.ID, err)
			return err
		}

		record.Eventf(openStackCluster, "SuccessfulUpdateListener", "Updated allowed_cidrs %s for listener %s with id %s", listener.AllowedCIDRs, listener.Name, listener.ID)
	}
	return nil
}

func (s *Service) getOrCreatePool(openStackCluster *infrav1.OpenStackCluster, poolName, listenerID, lbID string, lbProvider string) (*pools.Pool, error) {
	pool, err := s.checkIfPoolExists(poolName)
	if err != nil {
		return nil, err
	}

	if pool != nil {
		return pool, nil
	}

	s.scope.Logger().Info("Creating load balancer pool for listener", "loadBalancerID", lbID, "listenerID", listenerID, "name", poolName)

	method := pools.LBMethodRoundRobin

	if lbProvider == "ovn" {
		method = pools.LBMethodSourceIpPort
	}

	poolCreateOpts := pools.CreateOpts{
		Name:       poolName,
		Protocol:   "TCP",
		LBMethod:   method,
		ListenerID: listenerID,
		Tags:       openStackCluster.Spec.Tags,
	}
	pool, err = s.loadbalancerClient.CreatePool(poolCreateOpts)
	if err != nil {
		record.Warnf(openStackCluster, "FailedCreatePool", "Failed to create pool %s: %v", poolName, err)
		return nil, err
	}

	if _, err := s.waitForLoadBalancerActive(lbID); err != nil {
		record.Warnf(openStackCluster, "FailedCreatePool", "Failed to create pool %s with id %s: wait for load balancer active %s: %v", poolName, pool.ID, lbID, err)
		return nil, err
	}

	record.Eventf(openStackCluster, "SuccessfulCreatePool", "Created pool %s with id %s", poolName, pool.ID)
	return pool, nil
}

func (s *Service) ensureMonitor(openStackCluster *infrav1.OpenStackCluster, monitorName, poolID, lbID string) error {
	var cfg infrav1.APIServerLoadBalancerMonitor

	if openStackCluster.Spec.APIServerLoadBalancer.Monitor != nil {
		cfg = *openStackCluster.Spec.APIServerLoadBalancer.Monitor
	}

	cfg.Delay = cmp.Or(cfg.Delay, defaultMonitorDelay)
	cfg.Timeout = cmp.Or(cfg.Timeout, defaultMonitorTimeout)
	cfg.MaxRetries = cmp.Or(cfg.MaxRetries, defaultMonitorMaxRetries)
	cfg.MaxRetriesDown = cmp.Or(cfg.MaxRetriesDown, defaultMonitorMaxRetriesDown)

	monitor, err := s.checkIfMonitorExists(monitorName)
	if err != nil {
		return err
	}

	if monitor != nil {
		needsUpdate := false
		monitorUpdateOpts := monitors.UpdateOpts{}

		if monitor.Delay != cfg.Delay {
			s.scope.Logger().Info("Monitor delay needs update", "current", monitor.Delay, "desired", cfg.Delay)
			monitorUpdateOpts.Delay = cfg.Delay
			needsUpdate = true
		}

		if monitor.Timeout != cfg.Timeout {
			s.scope.Logger().Info("Monitor timeout needs update", "current", monitor.Timeout, "desired", cfg.Timeout)
			monitorUpdateOpts.Timeout = cfg.Timeout
			needsUpdate = true
		}

		if monitor.MaxRetries != cfg.MaxRetries {
			s.scope.Logger().Info("Monitor maxRetries needs update", "current", monitor.MaxRetries, "desired", cfg.MaxRetries)
			monitorUpdateOpts.MaxRetries = cfg.MaxRetries
			needsUpdate = true
		}

		if monitor.MaxRetriesDown != cfg.MaxRetriesDown {
			s.scope.Logger().Info("Monitor maxRetriesDown needs update", "current", monitor.MaxRetriesDown, "desired", cfg.MaxRetriesDown)
			monitorUpdateOpts.MaxRetriesDown = cfg.MaxRetriesDown
			needsUpdate = true
		}

		if needsUpdate {
			s.scope.Logger().Info("Updating load balancer monitor", "loadBalancerID", lbID, "name", monitorName, "monitorID", monitor.ID)

			updatedMonitor, err := s.loadbalancerClient.UpdateMonitor(monitor.ID, monitorUpdateOpts)
			if err != nil {
				record.Warnf(openStackCluster, "FailedUpdateMonitor", "Failed to update monitor %s with id %s: %v", monitorName, monitor.ID, err)
				return err
			}

			if _, err = s.waitForLoadBalancerActive(lbID); err != nil {
				record.Warnf(openStackCluster, "FailedUpdateMonitor", "Failed to update monitor %s with id %s: wait for load balancer active %s: %v", monitorName, monitor.ID, lbID, err)
				return err
			}

			record.Eventf(openStackCluster, "SuccessfulUpdateMonitor", "Updated monitor %s with id %s", monitorName, updatedMonitor.ID)
		}

		return nil
	}

	s.scope.Logger().Info("Creating load balancer monitor for pool", "loadBalancerID", lbID, "name", monitorName, "poolID", poolID)

	monitor, err = s.loadbalancerClient.CreateMonitor(monitors.CreateOpts{
		Name:           monitorName,
		PoolID:         poolID,
		Type:           "TCP",
		Delay:          cfg.Delay,
		Timeout:        cfg.Timeout,
		MaxRetries:     cfg.MaxRetries,
		MaxRetriesDown: cfg.MaxRetriesDown,
	})
	if err != nil {
		if capoerrors.IsNotImplementedError(err) {
			record.Warnf(openStackCluster, "SkippedCreateMonitor", "Health Monitor is not created as it's not implemented with the current Octavia provider.")
			return nil
		}

		record.Warnf(openStackCluster, "FailedCreateMonitor", "Failed to create monitor %s: %v", monitorName, err)
		return err
	}

	if _, err = s.waitForLoadBalancerActive(lbID); err != nil {
		record.Warnf(openStackCluster, "FailedCreateMonitor", "Failed to create monitor %s with id %s: wait for load balancer active %s: %v", monitorName, monitor.ID, lbID, err)
		return err
	}

	record.Eventf(openStackCluster, "SuccessfulCreateMonitor", "Created monitor %s with id %s", monitorName, monitor.ID)
	return nil
}

func (s *Service) ReconcileLoadBalancerMember(openStackCluster *infrav1.OpenStackCluster, openStackMachine *infrav1.OpenStackMachine, clusterResourceName, ip, machineFailureDomain string) error {
	// Preconditions
	if openStackCluster.Status.Network == nil {
		return errors.New("network is not yet available in openStackCluster.Status")
	}
	if len(openStackCluster.Status.Network.Subnets) == 0 {
		return errors.New("network.Subnets are not yet available in openStackCluster.Status")
	}
	if openStackCluster.Status.APIServerLoadBalancer == nil &&
		len(openStackCluster.Status.APIServerLoadBalancers) == 0 {
		return errors.New("no load balancers available in openStackCluster.Status")
	}
	if openStackCluster.Spec.ControlPlaneEndpoint == nil || !openStackCluster.Spec.ControlPlaneEndpoint.IsValid() {
		return errors.New("ControlPlaneEndpoint is not yet set in openStackCluster.Spec")
	}

	// Determine machine AZ and cross-AZ behavior
	machineAZ := machineAZPtr(machineFailureDomain)

	allowsCrossAZ := false
	if openStackCluster.Spec.APIServerLoadBalancer != nil {
		allowsCrossAZ = openStackCluster.Spec.APIServerLoadBalancer.AllowsCrossAZLoadBalancerMembers()
	}

	if allowsCrossAZ {
		s.scope.Logger().Info("Cross-AZ load balancer members allowed, registering machine to all load balancers",
			"machineName", openStackMachine.Name,
			"machineAZ", machineAZ)
	} else {
		s.scope.Logger().Info("Cross-AZ load balancer members disabled, registering machine only to same-AZ load balancers",
			"machineName", openStackMachine.Name,
			"machineAZ", machineAZ)
	}

	// Compute target LBs
	targetLoadBalancers := s.selectTargetLoadBalancers(openStackCluster, machineAZ, allowsCrossAZ)

	if len(targetLoadBalancers) == 0 {
		s.scope.Logger().Info("No target load balancers found for machine registration",
			"machineName", openStackMachine.Name,
			"machineAZ", machineAZ,
			"allowsCrossAZ", allowsCrossAZ)
		return nil
	}

	s.scope.Logger().Info("Reconciling load balancer member registration",
		"machineName", openStackMachine.Name,
		"machineAZ", machineAZ,
		"targetLoadBalancers", len(targetLoadBalancers),
		"allowsCrossAZ", allowsCrossAZ)

	// Build port list once
	portList := s.buildAPIServerPortList(openStackCluster)

	// Reconcile per target LB
	for _, targetLB := range targetLoadBalancers {
		if targetLB.ID == "" {
			s.scope.Logger().Info("Skipping load balancer with empty ID", "lbName", targetLB.Name)
			continue
		}
		if err := s.reconcileMembersForLB(openStackCluster, openStackMachine, clusterResourceName, ip, targetLB, portList); err != nil {
			return err
		}
	}

	return nil
}

// machineAZPtr returns a pointer to the machine failure domain if non-empty.
func machineAZPtr(fd string) *string {
	if fd == "" {
		return nil
	}
	return ptr.To(fd)
}

// selectTargetLoadBalancers chooses the set of LBs the machine should register with
// based on cross-AZ setting and machine AZ.
func (s *Service) selectTargetLoadBalancers(openStackCluster *infrav1.OpenStackCluster, machineAZ *string, allowsCrossAZ bool) []infrav1.LoadBalancer {
	var targets []infrav1.LoadBalancer

	if allowsCrossAZ {
		// Register to all load balancers (both legacy and multi-AZ)
		if openStackCluster.Status.APIServerLoadBalancer != nil {
			targets = append(targets, *openStackCluster.Status.APIServerLoadBalancer)
		}
		if openStackCluster.Status.APIServerLoadBalancers != nil {
			targets = append(targets, openStackCluster.Status.APIServerLoadBalancers...)
		}
		return targets
	}

	// Same-AZ only
	if machineAZ != nil {
		// Find load balancers in the same AZ
		for _, lb := range openStackCluster.Status.APIServerLoadBalancers {
			if lb.AvailabilityZone == *machineAZ {
				targets = append(targets, lb)
			}
		}
		// Also check legacy load balancer if it doesn't have an AZ set (backward compatibility)
		if openStackCluster.Status.APIServerLoadBalancer != nil &&
			openStackCluster.Status.APIServerLoadBalancer.AvailabilityZone == "" {
			targets = append(targets, *openStackCluster.Status.APIServerLoadBalancer)
		}
	} else if openStackCluster.Status.APIServerLoadBalancer != nil {
		// Machine has no AZ, register to legacy load balancer only for backward compatibility
		targets = append(targets, *openStackCluster.Status.APIServerLoadBalancer)
	}

	return targets
}

// buildAPIServerPortList returns the list of ports to register on the LB.
func (s *Service) buildAPIServerPortList(openStackCluster *infrav1.OpenStackCluster) []int {
	var ports []int
	if openStackCluster.Spec.ControlPlaneEndpoint != nil {
		ports = append(ports, int(openStackCluster.Spec.ControlPlaneEndpoint.Port))
	}
	if openStackCluster.Spec.APIServerLoadBalancer != nil {
		ports = append(ports, openStackCluster.Spec.APIServerLoadBalancer.AdditionalPorts...)
	}
	return ports
}

// reconcileMembersForLB ensures the machine is registered as a member across all provided ports for a specific LB.
func (s *Service) reconcileMembersForLB(openStackCluster *infrav1.OpenStackCluster, openStackMachine *infrav1.OpenStackMachine, clusterResourceName, ip string, targetLB infrav1.LoadBalancer, ports []int) error {
	loadBalancerName := targetLB.Name
	if loadBalancerName == "" {
		// Fallback to legacy naming if name is not set
		loadBalancerName = getLoadBalancerName(clusterResourceName)
	}

	s.scope.Logger().Info("Reconciling load balancer member for specific LB",
		"loadBalancerName", loadBalancerName,
		"loadBalancerID", targetLB.ID,
		"targetAZ", targetLB.AvailabilityZone)

	for _, port := range ports {
		lbPortObjectsName := fmt.Sprintf("%s-%d", loadBalancerName, port)
		name := lbPortObjectsName + "-" + openStackMachine.Name

		pool, err := s.checkIfPoolExists(lbPortObjectsName)
		if err != nil {
			return err
		}
		if pool == nil {
			s.scope.Logger().Info("Load balancer pool does not exist yet, skipping",
				"poolName", lbPortObjectsName,
				"loadBalancerName", loadBalancerName)
			continue
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

			s.scope.Logger().Info("Deleting load balancer member because the IP of the machine changed",
				"memberName", name,
				"oldIP", lbMember.Address,
				"newIP", ip)

			// lb member changed so let's delete it so we can create it again with the correct IP
			if _, err := s.waitForLoadBalancerActive(targetLB.ID); err != nil {
				return err
			}
			if err := s.loadbalancerClient.DeletePoolMember(pool.ID, lbMember.ID); err != nil {
				return err
			}
			if _, err := s.waitForLoadBalancerActive(targetLB.ID); err != nil {
				return err
			}
		}

		s.scope.Logger().Info("Creating load balancer member",
			"memberName", name,
			"poolID", pool.ID,
			"loadBalancerID", targetLB.ID,
			"ip", ip)

		// if we got to this point we should either create or re-create the lb member
		lbMemberOpts := pools.CreateMemberOpts{
			Name:         name,
			ProtocolPort: port,
			Address:      ip,
			Tags:         openStackCluster.Spec.Tags,
		}

		if _, err := s.waitForLoadBalancerActive(targetLB.ID); err != nil {
			return err
		}

		if _, err := s.loadbalancerClient.CreatePoolMember(pool.ID, lbMemberOpts); err != nil {
			return err
		}

		if _, err := s.waitForLoadBalancerActive(targetLB.ID); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) DeleteLoadBalancer(openStackCluster *infrav1.OpenStackCluster, clusterResourceName string) (*reconcile.Result, error) {
	lbSpec := openStackCluster.Spec.APIServerLoadBalancer
	if lbSpec == nil || !lbSpec.IsEnabled() {
		return nil, nil
	}

	// Collect all load balancer names to delete (both legacy and multi-AZ formats)
	var loadBalancerNames []string

	// Add legacy single load balancer name
	legacyLoadBalancerName := getLoadBalancerName(clusterResourceName)
	loadBalancerNames = append(loadBalancerNames, legacyLoadBalancerName)

	// Add multi-AZ load balancer names
	availabilityZones := lbSpec.AvailabilityZones

	// Handle legacy single AZ case
	if lbSpec.AvailabilityZone != nil && len(availabilityZones) == 0 {
		availabilityZones = []string{*lbSpec.AvailabilityZone}
	}

	// If no AZ specified, check for default AZ load balancer
	if len(availabilityZones) == 0 {
		defaultAZName := fmt.Sprintf("%s-default", clusterResourceName)
		defaultLBName := getLoadBalancerName(defaultAZName)
		loadBalancerNames = append(loadBalancerNames, defaultLBName)
	} else {
		// Add AZ-specific load balancer names
		for _, az := range availabilityZones {
			azClusterResourceName := fmt.Sprintf("%s-%s", clusterResourceName, az)
			azLBName := getLoadBalancerName(azClusterResourceName)
			loadBalancerNames = append(loadBalancerNames, azLBName)
		}
	}

	// Track any pending deletions for requeue
	var pendingDeletions []string
	deletedLoadBalancers := make([]string, 0, len(loadBalancerNames))

	// Iterate through all potential load balancer names and delete them
	for _, loadBalancerName := range loadBalancerNames {
		lb, err := s.checkIfLbExists(loadBalancerName)
		if err != nil {
			return nil, err
		}

		if lb == nil {
			continue // Load balancer doesn't exist, skip
		}

		// If the load balancer is already in PENDING_DELETE state, we don't need to do anything.
		// However we should still wait for the load balancer to be deleted which is why we
		// request a new reconcile after a certain amount of time.
		if lb.ProvisioningStatus == loadBalancerProvisioningStatusPendingDelete {
			s.scope.Logger().Info("Load balancer is already in PENDING_DELETE state", "name", loadBalancerName)
			pendingDeletions = append(pendingDeletions, loadBalancerName)
			continue
		}

		if lb.VipPortID != "" {
			fip, err := s.networkingService.GetFloatingIPByPortID(lb.VipPortID)
			if err != nil {
				return nil, err
			}

			if fip != nil && fip.FloatingIP != "" {
				if err = s.networkingService.DisassociateFloatingIP(openStackCluster, fip.FloatingIP); err != nil {
					return nil, err
				}

				// If the floating is user-provider (BYO floating IP), don't delete it.
				if openStackCluster.Spec.APIServerFloatingIP == nil || *openStackCluster.Spec.APIServerFloatingIP != fip.FloatingIP {
					if err = s.networkingService.DeleteFloatingIP(openStackCluster, fip.FloatingIP); err != nil {
						return nil, err
					}
				} else {
					s.scope.Logger().Info("Skipping load balancer floating IP deletion as it's a user-provided resource", "name", loadBalancerName, "fip", fip.FloatingIP)
				}
			}
		}

		deleteOpts := loadbalancers.DeleteOpts{
			Cascade: true,
		}
		s.scope.Logger().Info("Deleting load balancer", "name", loadBalancerName, "cascade", deleteOpts.Cascade)
		err = s.loadbalancerClient.DeleteLoadBalancer(lb.ID, deleteOpts)
		if err != nil && !capoerrors.IsNotFound(err) {
			record.Warnf(openStackCluster, "FailedDeleteLoadBalancer", "Failed to delete load balancer %s with id %s: %v", lb.Name, lb.ID, err)
			return nil, err
		}

		record.Eventf(openStackCluster, "SuccessfulDeleteLoadBalancer", "Deleted load balancer %s with id %s", lb.Name, lb.ID)
		deletedLoadBalancers = append(deletedLoadBalancers, lb.Name)

		// Remove the load balancer from the multi-AZ status list
		s.removeLoadBalancerFromMultiAZStatus(openStackCluster, lb.ID)
	}

	// If there are pending deletions, requeue to wait for completion
	if len(pendingDeletions) > 0 {
		s.scope.Logger().Info("Load balancers still in PENDING_DELETE state, will requeue", "pendingDeletions", pendingDeletions)
		return &reconcile.Result{RequeueAfter: waitForOctaviaLBCleanup}, nil
	}

	// If we deleted any load balancers, requeue to ensure cleanup is complete
	if len(deletedLoadBalancers) > 0 {
		s.scope.Logger().Info("Load balancers deletion initiated, will requeue to ensure cleanup completion", "deletedLoadBalancers", deletedLoadBalancers)
		return &reconcile.Result{RequeueAfter: waitForOctaviaLBCleanup}, nil
	}

	return nil, nil
}

func (s *Service) DeleteLoadBalancerMember(openStackCluster *infrav1.OpenStackCluster, openStackMachine *infrav1.OpenStackMachine, clusterResourceName string) error {
	if openStackMachine == nil {
		return errors.New("openStackMachine is nil")
	}

	loadBalancerName := getLoadBalancerName(clusterResourceName)
	lb, err := s.checkIfLbExists(loadBalancerName)
	if err != nil {
		return err
	}
	if lb == nil {
		// nothing to do
		return nil
	}

	lbID := lb.ID

	var portList []int
	if openStackCluster.Spec.ControlPlaneEndpoint != nil {
		portList = append(portList, int(openStackCluster.Spec.ControlPlaneEndpoint.Port))
	}
	if openStackCluster.Spec.APIServerLoadBalancer != nil {
		portList = append(portList, openStackCluster.Spec.APIServerLoadBalancer.AdditionalPorts...)
	}
	for _, port := range portList {
		lbPortObjectsName := fmt.Sprintf("%s-%d", loadBalancerName, port)
		name := lbPortObjectsName + "-" + openStackMachine.Name

		pool, err := s.checkIfPoolExists(lbPortObjectsName)
		if err != nil {
			return err
		}
		if pool == nil {
			s.scope.Logger().Info("Load balancer pool does not exist", "name", lbPortObjectsName)
			continue
		}

		lbMember, err := s.checkIfLbMemberExists(pool.ID, name)
		if err != nil {
			return err
		}

		if lbMember != nil {
			// lb member changed so let's delete it so we can create it again with the correct IP
			_, err = s.waitForLoadBalancerActive(lbID)
			if err != nil {
				return err
			}
			if err := s.loadbalancerClient.DeletePoolMember(pool.ID, lbMember.ID); err != nil {
				return err
			}
			_, err = s.waitForLoadBalancerActive(lbID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getLoadBalancerName(clusterResourceName string) string {
	return fmt.Sprintf("%s-cluster-%s-%s", networkPrefix, clusterResourceName, kubeapiLBSuffix)
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
func (s *Service) waitForLoadBalancerActive(id string) (*loadbalancers.LoadBalancer, error) {
	var lb *loadbalancers.LoadBalancer

	s.scope.Logger().Info("Waiting for load balancer", "id", id, "targetStatus", "ACTIVE")
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		var err error
		lb, err = s.loadbalancerClient.GetLoadBalancer(id)
		if err != nil {
			return false, err
		}
		return lb.ProvisioningStatus == loadBalancerProvisioningStatusActive, nil
	})
	if err != nil {
		return nil, err
	}
	return lb, nil
}

func (s *Service) waitForListener(id, target string) error {
	s.scope.Logger().Info("Waiting for load balancer listener", "id", id, "targetStatus", target)
	return wait.ExponentialBackoff(backoff, func() (bool, error) {
		_, err := s.loadbalancerClient.GetListener(id)
		if err != nil {
			return false, err
		}
		// The listener resource has no Status attribute, so a successful Get is the best we can do
		return true, nil
	})
}

// migrateAPIServerLoadBalancer takes the old name format for a loadbalancer and converts it to the new format.
// The new format is based on the availability zone and the cluster resource name.
func (s *Service) migrateAPIServerLoadBalancer(openStackCluster *infrav1.OpenStackCluster, clusterResourceName string, azInfo azSubnet, apiServerPort int) error {
	lbSpec := openStackCluster.Spec.APIServerLoadBalancer
	if !lbSpec.IsEnabled() {
		return nil
	}

	// Determine the AZ-specific resource name
	azClusterResourceName := fmt.Sprintf("%s-default", clusterResourceName)
	if azInfo.az != nil {
		azClusterResourceName = fmt.Sprintf("%s-%s", clusterResourceName, *azInfo.az)
	}

	s.scope.Logger().Info("Migrating API server load balancer resources to AZ-specific naming",
		"clusterName", clusterResourceName,
		"azName", azInfo.az,
		"newResourceName", azClusterResourceName)

	// Rename the load balancer
	if err := s.renameAPIServerLoadBalancer(clusterResourceName, azClusterResourceName); err != nil {
		return fmt.Errorf("failed to rename load balancer: %w", err)
	}

	// Rename listeners, pools, and monitors for each port
	portList := append([]int{apiServerPort}, lbSpec.AdditionalPorts...)
	for _, port := range portList {
		if err := s.renameAPIServerListener(clusterResourceName, azClusterResourceName, port); err != nil {
			return fmt.Errorf("failed to rename listener for port %d: %w", port, err)
		}
		if err := s.renameAPIServerPool(clusterResourceName, azClusterResourceName, port); err != nil {
			return fmt.Errorf("failed to rename pool for port %d: %w", port, err)
		}
		if err := s.renameAPIServerMonitor(clusterResourceName, azClusterResourceName, port); err != nil {
			return fmt.Errorf("failed to rename monitor for port %d: %w", port, err)
		}
	}

	s.scope.Logger().Info("Successfully migrated API server load balancer resources",
		"clusterName", clusterResourceName,
		"azName", azInfo.az)
	return nil
}

func (s *Service) renameAPIServerLoadBalancer(clusterResourceName, azClusterResourceName string) error {
	loadBalancerName := getLoadBalancerName(clusterResourceName)
	lb, err := s.checkIfLbExists(loadBalancerName)
	if err != nil {
		return fmt.Errorf("error checking if load balancer exists: %w", err)
	}
	if lb == nil {
		s.scope.Logger().Info("Load balancer does not exist, skipping rename", "name", loadBalancerName)
		return nil
	}

	azLoadBalancerName := getLoadBalancerName(azClusterResourceName)
	if lb.Name == azLoadBalancerName {
		s.scope.Logger().Info("Load balancer already has the correct name",
			"name", lb.Name)
		return nil
	}

	s.scope.Logger().Info("Renaming load balancer", "oldName", lb.Name, "newName", azLoadBalancerName)
	updateOpts := loadbalancers.UpdateOpts{
		Name: &azLoadBalancerName,
	}
	if _, err := s.loadbalancerClient.UpdateLoadBalancer(lb.ID, updateOpts); err != nil {
		return fmt.Errorf("failed to update load balancer name: %w", err)
	}
	return nil
}

func (s *Service) renameAPIServerListener(clusterResourceName, azClusterResourceName string, port int) error {
	loadBalancerName := getLoadBalancerName(clusterResourceName)
	lbPortObjectsName := fmt.Sprintf("%s-%d", loadBalancerName, port)

	listener, err := s.checkIfListenerExists(lbPortObjectsName)
	if err != nil {
		return fmt.Errorf("error checking if listener exists: %w", err)
	}
	if listener == nil {
		s.scope.Logger().Info("Listener does not exist, skipping rename", "name", lbPortObjectsName)
		return nil
	}

	azLbPortObjectsName := fmt.Sprintf("%s-%d", azClusterResourceName, port)
	if listener.Name == azLbPortObjectsName {
		s.scope.Logger().Info("Listener already has the correct name", "name", listener.Name)
		return nil
	}

	s.scope.Logger().Info("Renaming listener", "oldName", listener.Name, "newName", azLbPortObjectsName)
	updateOpts := listeners.UpdateOpts{
		Name: &azLbPortObjectsName,
	}
	if _, err := s.loadbalancerClient.UpdateListener(listener.ID, updateOpts); err != nil {
		return fmt.Errorf("failed to update listener name: %w", err)
	}
	return nil
}

func (s *Service) renameAPIServerPool(clusterResourceName, azClusterResourceName string, port int) error {
	loadBalancerName := getLoadBalancerName(clusterResourceName)
	lbPortObjectsName := fmt.Sprintf("%s-%d", loadBalancerName, port)

	pool, err := s.checkIfPoolExists(lbPortObjectsName)
	if err != nil {
		return fmt.Errorf("error checking if pool exists: %w", err)
	}
	if pool == nil {
		s.scope.Logger().Info("Pool does not exist, skipping rename", "name", lbPortObjectsName)
		return nil
	}

	azLbPortObjectsName := fmt.Sprintf("%s-%d", azClusterResourceName, port)
	if pool.Name == azLbPortObjectsName {
		s.scope.Logger().Info("Pool already has the correct name", "name", pool.Name)
		return nil
	}

	s.scope.Logger().Info("Renaming pool", "oldName", pool.Name, "newName", azLbPortObjectsName)
	updateOpts := pools.UpdateOpts{
		Name: &azLbPortObjectsName,
	}
	if _, err := s.loadbalancerClient.UpdatePool(pool.ID, updateOpts); err != nil {
		return fmt.Errorf("failed to update pool name: %w", err)
	}
	return nil
}

func (s *Service) renameAPIServerMonitor(clusterResourceName, azClusterResourceName string, port int) error {
	loadBalancerName := getLoadBalancerName(clusterResourceName)
	lbPortObjectsName := fmt.Sprintf("%s-%d", loadBalancerName, port)

	monitor, err := s.checkIfMonitorExists(lbPortObjectsName)
	if err != nil {
		return fmt.Errorf("error checking if monitor exists: %w", err)
	}
	if monitor == nil {
		s.scope.Logger().Info("Monitor does not exist, skipping rename", "name", lbPortObjectsName)
		return nil
	}

	azLbPortObjectsName := fmt.Sprintf("%s-%d", azClusterResourceName, port)
	if monitor.Name == azLbPortObjectsName {
		s.scope.Logger().Info("Monitor already has the correct name", "name", monitor.Name)
		return nil
	}

	s.scope.Logger().Info("Renaming monitor", "oldName", monitor.Name, "newName", azLbPortObjectsName)
	updateOpts := monitors.UpdateOpts{
		Name: &azLbPortObjectsName,
	}
	if _, err := s.loadbalancerClient.UpdateMonitor(monitor.ID, updateOpts); err != nil {
		return fmt.Errorf("failed to update monitor name: %w", err)
	}
	return nil
}

// updateMultiAZLoadBalancerStatus updates the APIServerLoadBalancers list with the current load balancer status.
func (s *Service) updateMultiAZLoadBalancerStatus(openStackCluster *infrav1.OpenStackCluster, lb *loadbalancers.LoadBalancer, az string) {
	if openStackCluster.Status.APIServerLoadBalancers == nil {
		openStackCluster.Status.APIServerLoadBalancers = []infrav1.LoadBalancer{}
	}

	// Find existing entry or create new one
	var existingLB *infrav1.LoadBalancer
	for i := range openStackCluster.Status.APIServerLoadBalancers {
		if openStackCluster.Status.APIServerLoadBalancers[i].ID == lb.ID {
			existingLB = &openStackCluster.Status.APIServerLoadBalancers[i]
			break
		}
	}

	if existingLB == nil {
		// Create new entry
		newLB := infrav1.LoadBalancer{
			Name:             lb.Name,
			ID:               lb.ID,
			InternalIP:       lb.VipAddress,
			Tags:             lb.Tags,
			AvailabilityZone: az,
		}
		openStackCluster.Status.APIServerLoadBalancers = append(openStackCluster.Status.APIServerLoadBalancers, newLB)
	} else {
		// Update existing entry
		existingLB.Name = lb.Name
		existingLB.ID = lb.ID
		existingLB.InternalIP = lb.VipAddress
		existingLB.Tags = lb.Tags
		existingLB.AvailabilityZone = az
	}
}

// updateMultiAZLoadBalancerFloatingIP updates the floating IP for a specific load balancer in the multi-AZ list.
func (s *Service) updateMultiAZLoadBalancerFloatingIP(openStackCluster *infrav1.OpenStackCluster, az string, floatingIP string) {
	if openStackCluster.Status.APIServerLoadBalancers == nil {
		return
	}

	for i := range openStackCluster.Status.APIServerLoadBalancers {
		lb := &openStackCluster.Status.APIServerLoadBalancers[i]
		if lb.AvailabilityZone == az {
			lb.IP = floatingIP
			break
		}
	}
}

// updateMultiAZLoadBalancerAllowedCIDRs updates the allowed CIDRs for a specific load balancer in the multi-AZ list.
func (s *Service) updateMultiAZLoadBalancerAllowedCIDRs(openStackCluster *infrav1.OpenStackCluster, az string, allowedCIDRs []string) {
	if openStackCluster.Status.APIServerLoadBalancers == nil {
		return
	}

	for i := range openStackCluster.Status.APIServerLoadBalancers {
		lb := &openStackCluster.Status.APIServerLoadBalancers[i]
		if lb.AvailabilityZone == az {
			lb.AllowedCIDRs = allowedCIDRs
			break
		}
	}
}

// updateMultiAZLoadBalancerNetwork updates the load balancer network information for a specific load balancer in the multi-AZ list.
func (s *Service) updateMultiAZLoadBalancerNetwork(openStackCluster *infrav1.OpenStackCluster, az string, lbNetwork *infrav1.NetworkStatusWithSubnets) {
	if openStackCluster.Status.APIServerLoadBalancers == nil {
		return
	}

	for i := range openStackCluster.Status.APIServerLoadBalancers {
		lb := &openStackCluster.Status.APIServerLoadBalancers[i]
		if lb.AvailabilityZone == az {
			lb.LoadBalancerNetwork = lbNetwork
			break
		}
	}
}

// removeLoadBalancerFromMultiAZStatus removes a load balancer from the APIServerLoadBalancers list by ID.
func (s *Service) removeLoadBalancerFromMultiAZStatus(openStackCluster *infrav1.OpenStackCluster, lbID string) {
	if openStackCluster.Status.APIServerLoadBalancers == nil {
		return
	}

	// Find and remove the load balancer entry by ID
	for i, lb := range openStackCluster.Status.APIServerLoadBalancers {
		if lb.ID == lbID {
			// Remove this entry by slicing
			openStackCluster.Status.APIServerLoadBalancers = append(
				openStackCluster.Status.APIServerLoadBalancers[:i],
				openStackCluster.Status.APIServerLoadBalancers[i+1:]...,
			)
			s.scope.Logger().Info("Removed load balancer from multi-AZ status list", "lbID", lbID)
			break
		}
	}
}
