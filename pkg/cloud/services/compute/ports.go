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
 *
 *
 * STATUS
 *
 * Blocked because we have nowhere to store port status for the bastion.
 * We're updating CreateInstance to take a list of pre-created ports, and
 * DeleteInstance to assume that it can rely on ports status. However, this
 * breaks the bastion so we can't do it yet.
 *
 * Options:
 * - Define an interface which can be implemented by both OpenStackCluster and OpenStackMachine which stores ports status.
 * - Reimplement the bastion to create an OpenStackMachine object and rely on the machine controller instead.
 *
 * Matt's preference is the latter.
 *
 * VOLUMES
 *
 * We were mistaken that we can lookup volumes by metadata, but we should still
 * add it. We should calculate tags for volumes in the same way we do for other
 * resources: cluster tags + machine tags. We should also add an additional tag
 * containing the uid of the kubernetes openstackmachine object. We can add all
 * of these as server metadata. This will allow us to distinguish in the case
 * where we find 2 volumes with the same name.
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

	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util/conditions"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha7"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
)

func (s *Service) adoptPorts(eventObject runtime.Object, clusterName string, openStackMachine *infrav1.OpenStackMachine, instanceSpec *InstanceSpec, desiredPorts []infrav1.PortOpts) error {
	portsReady := conditions.Get(openStackMachine, infrav1.PortsReadyCondition)
	if portsReady != nil && portsReady.Status == corev1.ConditionTrue {
		return nil
	}

	statusPorts := openStackMachine.Status.Ports

	networkingClient, err := s.getNetworkingClient()
	if err != nil {
		return err
	}

	for i, port := range desiredPorts {
		// check if the port is in status first and if it is, skip it
		if i < len(statusPorts) {
			continue
		}

		portOpts := &desiredPorts[i]
		portName := networking.GetPortName(openStackMachine.Name, portOpts, i)
		// List ports by name and tags
		ports, err := networkingClient.ListPort(ports.ListOpts{
			Name:      portName,
			NetworkID: port.Network.ID,
		})
		if err != nil {
			return err
		}
		if len(ports) == 0 {
			return nil
		}
		if len(ports) > 1 {
			return fmt.Errorf("found multiple ports with name %s", portName)
		}

		openStackMachine.Status.Ports = append(openStackMachine.Status.Ports, ports[0].ID)
	}

	return nil
}

/* TODO:
 * Remove clusterName params by populating port (and trunk?) descriptions in normalisePorts()
 * Use CreatePort instead of GetOrCreatePort because we already called AdoptPorts and we know the ports don't exist.
 * If this function returns an error, we should set that error into the PortsReady condition.
 */
func (s *Service) reconcilePorts(eventObject runtime.Object, clusterName string, openStackMachine *infrav1.OpenStackMachine, instanceSpec *InstanceSpec, desiredPorts []infrav1.PortOpts) error {
	portsReady := conditions.Get(openStackMachine, infrav1.PortsReadyCondition)
	if portsReady != nil && portsReady.Status == corev1.ConditionTrue {
		return nil
	}

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
		port, err := networkingService.GetOrCreatePort(eventObject, clusterName, portName, portOpts, securityGroups, iTags)
		if err != nil {
			return err
		}
		openStackMachine.Status.Ports = append(openStackMachine.Status.Ports, port.ID)
	}

	// XXX TODO: After creating *all* ports and add them to status, check that they have all reached `DOWN` state.
	// * If they have, set PortsReady to true.

	// Sets PortsReady to true anyway for now
	conditions.MarkTrue(openStackMachine, infrav1.PortsReadyCondition)

	return nil
}

func (s *Service) reconcilePortsDelete(eventObject runtime.Object, openStackMachine *infrav1.OpenStackMachine) error {
	// Delete all of the ports in status
	networkingClient, err := s.getNetworkingClient()
	if err != nil {
		return err
	}

	for _, portID := range openStackMachine.Status.Ports {
		if err := networkingClient.DeletePort(portID); err != nil && !capoerrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}
