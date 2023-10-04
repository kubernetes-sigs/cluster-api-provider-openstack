package compute

/*
 * # Legacy
 *
 * If the server has been created, ports are attached to the server.
 * If the server has not been created, ports should have been deleted.
 *
 * When adopting ports, if server already exists add all attached ports to status and set PortsReady to true.
 *
 * # Current
 *
 * If PortsReady is true, status ports are canonical.
 * If PortsReady is false, we need to adopt/create ports.
 * Created ports are tagged with the uid of the machine and cluster which own them.
 *
 * Ports in status are in same order as machine spec.
 * Do not create a port if it already has an entry in status.
 *
 * If a port is not in status it may have been previously created but not added to status.
 * Check for existing port by name and tags.
 * If none, create a new port.
 * Add the new port to status.
 *
 * Fetch all ports. If they are all in `DOWN` state, set PortsReady to true.
 *
 * Invariants:
 * - If PortsReady is true, status contains all ports which have all reached `DOWN` state.
 * - If a port exists in status it and all lower numbered ports have been created in OpenStack.
 *
 * Prior to adoption check:
 * - If a port does not exist in status it MAY have been created in OpenStack.
 *
 * After adoption check:
 * - If a port does not exist in status is HAS NOT been created in OpenStack.
 */

/*
	// Ensure we delete the ports we created if we haven't created the server.
	defer func() {
		if server != nil {
			return
		}

		if err := s.deletePorts(eventObject, portList); err != nil {
			s.scope.Logger().V(4).Error(err, "Failed to clean up ports after failure")
		}
	}()

	ports, err := s.constructPorts(openStackCluster, instanceSpec)
	if err != nil {
		return nil, err
	}

	networkingService, err := s.getNetworkingService()
	if err != nil {
		return nil, err
	}

	securityGroups, err := networkingService.GetSecurityGroups(instanceSpec.SecurityGroups)
	if err != nil {
		return nil, fmt.Errorf("error getting security groups: %v", err)
	}

	for i := range ports {
		portOpts := &ports[i]
		iTags := []string{}
		if len(instanceSpec.Tags) > 0 {
			iTags = instanceSpec.Tags
		}
		portName := networking.GetPortName(instanceSpec.Name, portOpts, i)
		port, err := networkingService.GetOrCreatePort(eventObject, clusterName, portName, portOpts, securityGroups, iTags)
		if err != nil {
			return nil, err
		}

		portList = append(portList, servers.Network{
			Port: port.ID,
		})
	}
*/

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util/conditions"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha7"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
)

func (s *Service) adoptPorts() error {
}

/* TODO:
 * Remove clusterName params by populating port (and trunk?) descriptions in normalisePorts()
 * Where do we initialise security groups?
 */

func (s *Service) reconcilePorts(eventObject runtime.Object, clusterName string, openStackMachine *infrav1.OpenStackMachine, instanceSpec *InstanceSpec, desiredPorts []infrav1.PortOpts) error {
	portsReady := conditions.Get(openStackMachine, infrav1.PortsReadyCondition)
	if portsReady != nil && portsReady.Status == corev1.ConditionTrue {
		return nil
	}

	// Get the ports from Status
	statusPorts := openStackMachine.Status.Ports

	networkingService, err := s.getNetworkingService()
	if err != nil {
		return err
	}

	securityGroups, err := networkingService.GetSecurityGroups(instanceSpec.SecurityGroups)
	if err != nil {
		return fmt.Errorf("error getting security groups: %w", err)
	}

	for i := range desiredPorts {
		// check if the port is in status first and if it is, skip it
		if i < len(statusPorts) {
			continue
		}

		portOpts := &desiredPorts[i]
		iTags := []string{}
		if len(openStackMachine.Spec.Tags) > 0 {
			iTags = openStackMachine.Spec.Tags
		}
		portName := networking.GetPortName(openStackMachine.Name, portOpts, i)
		// XXX FIXME: PASS SECURITY GROUPS IN HERE!!!
		port, err := networkingService.GetOrCreatePort(eventObject, clusterName, portName, portOpts, securityGroups, iTags)
		if err != nil {
			return err
		}

	}

}

func (s *Service) reconcilePortsDelete() error {

}
