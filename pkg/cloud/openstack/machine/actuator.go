/*
Copyright 2018 The Kubernetes Authors.

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

package machine

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"net"
	"os"
	"reflect"
	constants "sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/contants"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/compute"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/provider"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/userdata"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/deployer"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	clientgorecord "k8s.io/client-go/tools/record"
	"k8s.io/klog"
	providerv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	clientclusterv1 "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
	apierrors "sigs.k8s.io/cluster-api/pkg/errors"
	"sigs.k8s.io/cluster-api/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TimeoutInstanceCreate       = 5
	TimeoutInstanceDelete       = 5
	RetryIntervalInstanceStatus = 10 * time.Second
)

type Actuator struct {
	*deployer.Deployer

	params ActuatorParams
	scheme *runtime.Scheme
	client client.Client
}

// ActuatorParams holds parameter information for Actuator
type ActuatorParams struct {
	KubeClient    kubernetes.Interface
	Client        client.Client
	ClusterClient clientclusterv1.ClusterV1alpha1Interface
	EventRecorder clientgorecord.EventRecorder
	Scheme        *runtime.Scheme
}

func NewActuator(params ActuatorParams) *Actuator {
	return &Actuator{
		Deployer: deployer.New(),
		params:   params,
		client:   params.Client,
		scheme:   params.Scheme,
	}
}

func (a *Actuator) Create(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {

	if cluster == nil {
		return fmt.Errorf("the cluster is nil, check your cluster configuration")
	}
	klog.Infof("Creating Machine %s/%s: %s", cluster.Namespace, cluster.Name, machine.Name)

	clusterName := fmt.Sprintf("%s-%s", cluster.ObjectMeta.Namespace, cluster.Name)

	osProviderClient, clientOpts, err := provider.NewClientFromMachine(a.params.KubeClient, machine)
	if err != nil {
		return err
	}

	computeService, err := compute.NewService(osProviderClient, clientOpts)
	if err != nil {
		return err
	}

	clusterProviderSpec, err := providerv1.ClusterSpecFromProviderSpec(cluster.Spec.ProviderSpec)
	if err != nil {
		return a.handleMachineError(machine, apierrors.CreateMachine(
			"error creating Openstack instance: %v", err))
	}
	machineProviderSpec, err := providerv1.MachineSpecFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return a.handleMachineError(machine, apierrors.InvalidMachineConfiguration(
			"Cannot unmarshal machineProviderSpec field: %v", err))
	}

	if vErr := a.validateMachine(machine, machineProviderSpec); vErr != nil {
		return a.handleMachineError(machine, vErr)
	}

	instance, err := a.instanceExists(machine)
	if err != nil {
		return err
	}
	if instance != nil {
		klog.Infof("Skipped creating a VM that already exists.\n")
		return nil
	}

	userData, err := userdata.GetUserData(a.params.Client, a.params.KubeClient, machineProviderSpec, cluster, machine)
	if err != nil {
		if machineError, ok := err.(*apierrors.MachineError); ok {
			return a.handleMachineError(machine, machineError)
		}
		return err
	}

	instance, err = computeService.InstanceCreate(clusterName, machine.Name, clusterProviderSpec, machineProviderSpec, userData, machineProviderSpec.KeyName)

	if err != nil {
		return a.handleMachineError(machine, apierrors.CreateMachine(
			"error creating Openstack instance: %v", err))
	}
	instanceCreateTimeout := getTimeout("CLUSTER_API_OPENSTACK_INSTANCE_CREATE_TIMEOUT", TimeoutInstanceCreate)
	instanceCreateTimeout = instanceCreateTimeout * time.Minute
	err = util.PollImmediate(RetryIntervalInstanceStatus, instanceCreateTimeout, func() (bool, error) {
		instance, err := computeService.GetInstance(instance.ID)
		if err != nil {
			return false, nil
		}
		return instance.Status == "ACTIVE", nil
	})
	if err != nil {
		return a.handleMachineError(machine, apierrors.CreateMachine(
			"error creating Openstack instance: %v", err))
	}

	if machineProviderSpec.FloatingIP != "" {
		err := computeService.AssociateFloatingIP(instance.ID, machineProviderSpec.FloatingIP)
		if err != nil {
			return a.handleMachineError(machine, apierrors.CreateMachine(
				"Associate floatingIP err: %v", err))
		}

	}

	// update Annotation below will store machine spec
	providerID := fmt.Sprintf("openstack:////%s", instance.ID)
	machine.Spec.ProviderID = &providerID

	klog.Infof("updating status of machine of %s", machine.Name)
	ext, _ := providerv1.EncodeMachineStatus(&providerv1.OpenstackMachineProviderStatus{})
	machine.Status.ProviderStatus = ext
	err = a.updateMachine(cluster, machine)
	if err != nil {
		klog.Infof("updated status of machine failed: %v", err)
	}

	record.Eventf(machine, "CreatedInstance", "Created new instance with id: %s", instance.ID)
	return a.updateAnnotation(machine, instance.ID)
}

func (a *Actuator) updateMachine(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	machineClient := a.params.ClusterClient.Machines(cluster.Namespace)
	_, err := machineClient.UpdateStatus(machine)
	return err
}

func (a *Actuator) Delete(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {

	if cluster == nil {
		return fmt.Errorf("the cluster is nil, check your cluster configuration")
	}
	klog.Infof("Deleting Machine %s/%s: %s", cluster.Namespace, cluster.Name, machine.Name)

	osProviderClient, clientOpts, err := provider.NewClientFromMachine(a.params.KubeClient, machine)
	if err != nil {
		return err
	}

	computeService, err := compute.NewService(osProviderClient, clientOpts)
	if err != nil {
		return err
	}

	instance, err := a.instanceExists(machine)
	if err != nil {
		return err
	}

	if instance == nil {
		klog.Infof("Skipped deleting %s that is already deleted.\n", machine.Name)
		return nil
	}

	id := machine.ObjectMeta.Annotations[constants.OpenstackIdAnnotationKey]
	err = computeService.InstanceDelete(id)
	if err != nil {
		return a.handleMachineError(machine, apierrors.DeleteMachine(
			"error deleting Openstack instance: %v", err))
	}
	record.Eventf(machine, "DeletedInstance", "Deleted instance with id: %s", instance.ID)
	return nil
}

func (a *Actuator) Update(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {

	if cluster == nil {
		return fmt.Errorf("the cluster is nil, check your cluster configuration")
	}
	klog.Infof("Updating Machine %s/%s: %s", cluster.Namespace, cluster.Name, machine.Name)

	status, err := a.instanceStatus(machine)
	if err != nil {
		return err
	}

	// FIXME: Sometimes the master node ProviderStatus update of bootstrap cluster didn't reflected on the new cluster
	klog.Infof("updating status of machine of %s", machine.Name)
	ext, _ := providerv1.EncodeMachineStatus(&providerv1.OpenstackMachineProviderStatus{})
	machine.Status.ProviderStatus = ext
	err = a.updateMachine(cluster, machine)
	if err != nil {
		klog.Infof("updated status of machine failed: %v", err)
	}

	currentMachine := (*clusterv1.Machine)(status)
	if currentMachine == nil {
		instance, err := a.instanceExists(machine)
		if err != nil {
			return err
		}
		if instance != nil && instance.Status == "ACTIVE" {
			klog.Infof("Populating current state for boostrap machine %v", machine.ObjectMeta.Name)
			return a.updateAnnotation(machine, instance.ID)
		} else {
			return fmt.Errorf("cannot retrieve current state to update machine %v", machine.ObjectMeta.Name)
		}
	}

	if !requiresUpdate(currentMachine, machine) {
		return nil
	}

	if util.IsControlPlaneMachine(currentMachine) {
		// TODO: add master inplace
		klog.Errorf("master inplace update failed: not support master in place update now")
	} else {
		klog.Infof("re-creating machine %s for update.", currentMachine.ObjectMeta.Name)
		err = a.Delete(ctx, cluster, currentMachine)
		if err != nil {
			klog.Errorf("delete machine %s for update failed: %v", currentMachine.ObjectMeta.Name, err)
		} else {
			instanceDeleteTimeout := getTimeout("CLUSTER_API_OPENSTACK_INSTANCE_DELETE_TIMEOUT", TimeoutInstanceDelete)
			instanceDeleteTimeout = instanceDeleteTimeout * time.Minute
			err = util.PollImmediate(RetryIntervalInstanceStatus, instanceDeleteTimeout, func() (bool, error) {
				instance, err := a.instanceExists(machine)
				if err != nil {
					return false, nil
				}
				return instance == nil, nil
			})
			if err != nil {
				return a.handleMachineError(machine, apierrors.DeleteMachine(
					"error deleting Openstack instance: %v", err))
			}

			err = a.Create(ctx, cluster, machine)
			if err != nil {
				klog.Errorf("create machine %s for update failed: %v", machine.ObjectMeta.Name, err)
			}
			klog.Infof("Successfully updated machine %s", currentMachine.ObjectMeta.Name)
		}
	}

	return nil
}

func (a *Actuator) Exists(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) (bool, error) {
	instance, err := a.instanceExists(machine)
	if err != nil {
		return false, err
	}

	if (instance != nil) && (machine.Spec.ProviderID == nil || *machine.Spec.ProviderID == "") {
		providerID := fmt.Sprintf("openstack:////%s", instance.ID)
		machine.Spec.ProviderID = &providerID

		a.client.Update(nil, machine)
	}
	return instance != nil, err
}

func getIPFromInstance(instance *compute.Instance) (string, error) {
	if instance.AccessIPv4 != "" && net.ParseIP(instance.AccessIPv4) != nil {
		return instance.AccessIPv4, nil
	}
	type networkInterface struct {
		Address string  `json:"addr"`
		Version float64 `json:"version"`
		Type    string  `json:"OS-EXT-IPS:type"`
	}
	var addrList []string

	for _, b := range instance.Addresses {
		list, err := json.Marshal(b)
		if err != nil {
			return "", fmt.Errorf("extract IP from instance err: %v", err)
		}
		var networks []interface{}
		json.Unmarshal(list, &networks)
		for _, network := range networks {
			var netInterface networkInterface
			b, _ := json.Marshal(network)
			json.Unmarshal(b, &netInterface)
			if netInterface.Version == 4.0 {
				if netInterface.Type == "floating" {
					return netInterface.Address, nil
				}
				addrList = append(addrList, netInterface.Address)
			}
		}
	}
	if len(addrList) != 0 {
		return addrList[0], nil
	}
	return "", fmt.Errorf("extract IP from instance err")
}

// If the Actuator has a client for updating Machine objects, this will set
// the appropriate reason/message on the Machine.Status. If not, such as during
// cluster installation, it will operate as a no-op. It also returns the
// original error for convenience, so callers can do "return handleMachineError(...)".
func (a *Actuator) handleMachineError(machine *clusterv1.Machine, err *apierrors.MachineError) error {
	if a.client != nil {
		reason := err.Reason
		message := err.Message
		machine.Status.ErrorReason = &reason
		machine.Status.ErrorMessage = &message
		if err := a.client.Update(nil, machine); err != nil {
			return fmt.Errorf("unable to update machine status: %v", err)
		}
	}

	klog.Errorf("Machine error %s: %v", machine.Name, err.Message)
	return err
}

func (a *Actuator) updateAnnotation(machine *clusterv1.Machine, id string) error {
	if machine.ObjectMeta.Annotations == nil {
		machine.ObjectMeta.Annotations = make(map[string]string)
	}
	machine.ObjectMeta.Annotations[constants.OpenstackIdAnnotationKey] = id
	instance, _ := a.instanceExists(machine)
	ip, err := getIPFromInstance(instance)
	if err != nil {
		return err
	}
	machine.ObjectMeta.Annotations[constants.OpenstackIPAnnotationKey] = ip
	if err := a.client.Update(nil, machine); err != nil {
		return err
	}
	return a.updateInstanceStatus(machine)
}

func requiresUpdate(current *clusterv1.Machine, new *clusterv1.Machine) bool {
	if current == nil || new == nil {
		return true
	}
	// Do not want status changes. Do want changes that impact machine provisioning
	return !reflect.DeepEqual(current.Spec.ObjectMeta, new.Spec.ObjectMeta) ||
		!reflect.DeepEqual(current.Spec.ProviderSpec, new.Spec.ProviderSpec) ||
		!reflect.DeepEqual(current.Spec.Versions, new.Spec.Versions) ||
		current.ObjectMeta.Name != new.ObjectMeta.Name
}

func (a *Actuator) instanceExists(machine *clusterv1.Machine) (instance *compute.Instance, err error) {
	machineSpec, err := providerv1.MachineSpecFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, err
	}
	opts := &compute.InstanceListOpts{
		Name:   machine.Name,
		Image:  machineSpec.Image,
		Flavor: machineSpec.Flavor,
	}

	osProviderClient, clientOpts, err := provider.NewClientFromMachine(a.params.KubeClient, machine)
	if err != nil {
		return nil, err
	}

	computeService, err := compute.NewService(osProviderClient, clientOpts)
	if err != nil {
		return nil, err
	}

	instanceList, err := computeService.GetInstanceList(opts)
	if err != nil {
		return nil, err
	}
	if len(instanceList) == 0 {
		return nil, nil
	}
	return instanceList[0], nil
}

func (a *Actuator) validateMachine(machine *clusterv1.Machine, config *providerv1.OpenstackProviderSpec) *apierrors.MachineError {
	// TODO: other validate of openstackCloud
	return nil
}

func getTimeout(name string, timeout int) time.Duration {
	if v := os.Getenv(name); v != "" {
		timeout, err := strconv.Atoi(v)
		if err == nil {
			return time.Duration(timeout)
		}
	}
	return time.Duration(timeout)
}
