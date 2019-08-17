/*

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

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
	"net"
	"os"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/compute"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/loadbalancer"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/provider"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/userdata"
	"sigs.k8s.io/cluster-api/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha2"
	capierrors "sigs.k8s.io/cluster-api/pkg/errors"
	"sigs.k8s.io/cluster-api/pkg/util"
)

const (
	waitForClusterInfrastructureReadyDuration = 15 * time.Second
	machineControllerName                     = "openstackmachine-controller"
	TimeoutInstanceCreate                     = 5
	TimeoutInstanceDelete                     = 5
	RetryIntervalInstanceStatus               = 10 * time.Second
)

// OpenStackMachineReconciler reconciles a OpenStackMachine object
type OpenStackMachineReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;machines,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch

func (r *OpenStackMachineReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	ctx := context.TODO()
	logger := log.Log.
		WithName(machineControllerName).
		WithName(fmt.Sprintf("namespace=%s", request.Namespace)).
		WithName(fmt.Sprintf("openStackMachine=%s", request.Name))

	// Fetch the OpenStackMachine instance.
	openStackMachine := &infrav1.OpenStackMachine{}
	err := r.Get(ctx, request.NamespacedName, openStackMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	logger = logger.WithName(openStackMachine.APIVersion)

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, openStackMachine.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if machine == nil {
		logger.Info("Waiting for Machine Controller to set OwnerRef on OpenStackMachine")
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	logger = logger.WithName(fmt.Sprintf("machine=%s", machine.Name))

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		logger.Info("Machine is missing cluster label or cluster does not exist")
		return reconcile.Result{}, nil
	}

	logger = logger.WithName(fmt.Sprintf("cluster=%s", cluster.Name))

	openStackCluster := &infrav1.OpenStackCluster{}
	openStackClusterName := types.NamespacedName{
		Namespace: openStackMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Client.Get(ctx, openStackClusterName, openStackCluster); err != nil {
		logger.Info("Waiting for OpenStackCluster")
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	logger = logger.WithName(fmt.Sprintf("openStackCluster=%s", openStackCluster.Name))

	// Handle deleted clusters
	if !openStackMachine.DeletionTimestamp.IsZero() {
		return r.reconcileMachineDelete(logger, machine, openStackMachine, cluster, openStackCluster)
	}

	// Handle non-deleted clusters
	return r.reconcileMachine(logger, machine, openStackMachine, cluster, openStackCluster)
}

func (r *OpenStackMachineReconciler) reconcileMachine(logger logr.Logger, machine *v1alpha2.Machine, openStackMachine *infrav1.OpenStackMachine, cluster *v1alpha2.Cluster, openStackCluster *infrav1.OpenStackCluster) (_ ctrl.Result, reterr error) {
	// If the OpenStackMachine is in an error state, return early.
	if openStackMachine.Status.ErrorReason != nil || openStackMachine.Status.ErrorMessage != nil {
		logger.Info("Error state detected, skipping reconciliation")
		return reconcile.Result{}, nil
	}

	// If the OpenStackMachine doesn't have our finalizer, add it.
	if !util.Contains(openStackMachine.Finalizers, infrav1.MachineFinalizer) {
		openStackMachine.Finalizers = append(openStackMachine.Finalizers, infrav1.MachineFinalizer)
	}

	if !cluster.Status.InfrastructureReady {
		logger.Info("Cluster infrastructure is not ready yet, requeuing machine")
		return reconcile.Result{RequeueAfter: waitForClusterInfrastructureReadyDuration}, nil
	}

	// TODO enable when using kubeadm bootstrapper
	// Make sure bootstrap data is available and populated.
	//if machine.Spec.Bootstrap.Data == nil {
	//	logger.Info("Waiting for bootstrap data to be available")
	//	return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	//}

	klog.Infof("Creating Machine %s/%s: %s", cluster.Namespace, cluster.Name, machine.Name)

	clusterName := fmt.Sprintf("%s-%s", cluster.ObjectMeta.Namespace, cluster.Name)

	openstackMachinePatch := client.MergeFrom(openStackMachine.DeepCopy())

	osProviderClient, clientOpts, err := provider.NewClientFromMachine(r.Client, openStackMachine)
	if err != nil {
		return reconcile.Result{}, err
	}

	computeService, err := compute.NewService(osProviderClient, clientOpts)
	if err != nil {
		return reconcile.Result{}, err
	}

	networkingService, err := networking.NewService(osProviderClient, clientOpts)
	if err != nil {
		return reconcile.Result{}, err
	}

	defer func() {
		if err := storeMachine(r.Client, openStackMachine, openstackMachinePatch); err != nil && reterr == nil {
			reterr = err
		}
	}()

	instance, err := r.getOrCreate(computeService, machine, openStackMachine, cluster, openStackCluster)
	if err != nil {
		handleMachineError(openStackMachine, capierrors.UpdateMachineError, errors.Errorf("OpenStack instance cannot be created: %v", err))
		return reconcile.Result{}, err
	}

	// Set an error message if we couldn't find the instance.
	if instance == nil {
		handleMachineError(openStackMachine, capierrors.UpdateMachineError, errors.New("OpenStack instance cannot be found"))
		return reconcile.Result{}, nil
	}

	// TODO(sbueringer) From CAPA: TODO(ncdc): move this validation logic into a validating webhook (for us: create validation logic in webhook)

	openStackMachine.Spec.ProviderID = pointer.StringPtr(fmt.Sprintf("openstack:////%s", instance.ID))

	openStackMachine.Status.InstanceState = &instance.State

	// TODO(sbueringer) From CAPA: TODO(vincepri): Remove this annotation when clusterctl is no longer relevant.
	if openStackMachine.Annotations == nil {
		openStackMachine.Annotations = map[string]string{}
	}
	openStackMachine.Annotations["cluster-api-provider-openstack"] = "true"

	switch instance.State {
	case infrav1.InstanceStateActive:
		logger.Info("Machine instance is ACTIVE", "instance-id", instance.ID)
	case infrav1.InstanceStateBuilding:
		logger.Info("Machine instance is BUILDING", "instance-id", instance.ID)
	default:
		handleMachineError(openStackMachine, capierrors.UpdateMachineError, errors.Errorf("OpenStack instance state %q is unexpected", instance.State))
		return reconcile.Result{}, nil
	}

	if openStackMachine.Spec.FloatingIP != "" {
		err = r.reconcileFloatingIP(computeService, networkingService, instance, openStackMachine, openStackCluster)
		if err != nil {
			handleMachineError(openStackMachine, capierrors.UpdateMachineError, errors.Errorf("FloatingIP cannot be reconciled: %v", err))
			return reconcile.Result{}, nil
		}
	}

	if openStackCluster.Spec.ManagedAPIServerLoadBalancer {
		err = r.reconcileLoadBalancerMember(osProviderClient, clientOpts, instance, clusterName, machine, openStackMachine, openStackCluster)
		if err != nil {
			handleMachineError(openStackMachine, capierrors.UpdateMachineError, errors.Errorf("LoadBalancerMember cannot be reconciled: %v", err))
			return reconcile.Result{}, nil
		}
	}

	// TODO(sbueringer) check if that's the right place to set the machine to ready
	openStackMachine.Status.Ready = true

	klog.Infof("Created Machine %s/%s: %s successfully", cluster.Namespace, cluster.Name, machine.Name)
	return reconcile.Result{}, nil
}

func (r *OpenStackMachineReconciler) reconcileMachineDelete(logger logr.Logger, machine *v1alpha2.Machine, openStackMachine *infrav1.OpenStackMachine, cluster *v1alpha2.Cluster, openStackCluster *infrav1.OpenStackCluster) (ctrl.Result, error) {

	klog.Infof("Deleting Machine %s/%s: %s", cluster.Namespace, cluster.Name, machine.Name)

	clusterName := fmt.Sprintf("%s-%s", cluster.ObjectMeta.Namespace, cluster.Name)

	osProviderClient, clientOpts, err := provider.NewClientFromMachine(r.Client, openStackMachine)
	if err != nil {
		return reconcile.Result{}, err
	}

	computeService, err := compute.NewService(osProviderClient, clientOpts)
	if err != nil {
		return reconcile.Result{}, err
	}

	loadbalancerService, err := loadbalancer.NewService(osProviderClient, clientOpts, openStackCluster.Spec.UseOctavia)
	if err != nil {
		return reconcile.Result{}, err
	}
	if openStackCluster.Spec.ManagedAPIServerLoadBalancer {
		err = loadbalancerService.DeleteLoadBalancerMember(clusterName, machine, openStackMachine, openStackCluster)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	instance, err := computeService.InstanceExists(openStackMachine)
	if err != nil {
		return reconcile.Result{}, err
	}

	if instance == nil {
		klog.Infof("Skipped deleting %s that is already deleted.\n", machine.Name)
		return reconcile.Result{}, nil
	}

	err = computeService.InstanceDelete(machine)
	if err != nil {
		handleMachineError(openStackMachine, capierrors.UpdateMachineError, errors.Errorf("error deleting Openstack instance: %v", err))
		return reconcile.Result{}, nil
	}
	return reconcile.Result{}, nil
}

func (r *OpenStackMachineReconciler) getOrCreate(computeService *compute.Service, machine *v1alpha2.Machine, openStackMachine *infrav1.OpenStackMachine, cluster *v1alpha2.Cluster, openStackCluster *infrav1.OpenStackCluster) (*compute.Instance, error) {

	instance, err := computeService.InstanceExists(openStackMachine)
	if err != nil {
		return nil, err
	}

	if instance == nil {
		userData, err := userdata.GetUserData(r.Client, machine, openStackMachine, cluster, openStackCluster)
		if err != nil {
			return nil, err
		}

		instance, err = computeService.InstanceCreate(cluster.Name, machine.Name, openStackCluster, openStackMachine, userData)
		if err != nil {
			return nil, errors.Errorf("error creating Openstack instance: %v", err)
		}
		instanceCreateTimeout := getTimeout("CLUSTER_API_OPENSTACK_INSTANCE_CREATE_TIMEOUT", TimeoutInstanceCreate)
		instanceCreateTimeout = instanceCreateTimeout * time.Minute
		// instance in PollImmediate has to overwrites instance of the outer scope to get an updated instance state,
		// which is then returned at the end of getOrCreate
		err = util.PollImmediate(RetryIntervalInstanceStatus, instanceCreateTimeout, func() (bool, error) {
			instance, err = computeService.GetInstance(instance.ID)
			if err != nil {
				return false, nil
			}
			return instance.Status == "ACTIVE", nil
		})
		if err != nil {
			return nil, errors.Errorf("error creating Openstack instance: %v", err)
		}
	}

	return instance, nil
}

func handleMachineError(openstackMachine *infrav1.OpenStackMachine, reason capierrors.MachineStatusError, message error) {
	openstackMachine.Status.ErrorReason = &reason
	openstackMachine.Status.ErrorMessage = pointer.StringPtr(message.Error())
	// TODO remove if this error is logged redundantly
	klog.Errorf("Machine error %s: %v", openstackMachine.Name, message.Error())
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

func storeMachine(ctrlClient client.Client, openStackMachine *infrav1.OpenStackMachine, openStackMachinePatch client.Patch) error {
	ctx := context.TODO()

	// Patch Cluster object.
	if err := ctrlClient.Patch(ctx, openStackMachine, openStackMachinePatch); err != nil {
		return errors.Wrapf(err, "error patching OpenStackMachine %s/%s", openStackMachine.Namespace, openStackMachine.Name)
	}

	// Patch Cluster status.
	if err := ctrlClient.Status().Patch(ctx, openStackMachine, openStackMachinePatch); err != nil {
		return errors.Wrapf(err, "error patching OpenStackMachine %s/%s status", openStackMachine.Namespace, openStackMachine.Name)
	}

	return nil
}

func (r *OpenStackMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.OpenStackMachine{}).
		Complete(r)
}

func (r *OpenStackMachineReconciler) reconcileFloatingIP(computeService *compute.Service, networkingService *networking.Service, instance *compute.Instance, openStackMachine *infrav1.OpenStackMachine, openStackCluster *infrav1.OpenStackCluster) error {
	err := networkingService.GetOrCreateFloatingIP(openStackCluster, openStackMachine.Spec.FloatingIP)
	if err != nil {
		return fmt.Errorf("error creating floatingIP: %v", err)
	}

	err = computeService.AssociateFloatingIP(instance.ID, openStackMachine.Spec.FloatingIP)
	if err != nil {
		return fmt.Errorf("error associationg floatingIP: %v", err)
	}
	return nil
}

func (r *OpenStackMachineReconciler) reconcileLoadBalancerMember(osProviderClient *gophercloud.ProviderClient, clientOpts *clientconfig.ClientOpts, instance *compute.Instance, clusterName string, machine *v1alpha2.Machine, openStackMachine *infrav1.OpenStackMachine, openStackCluster *infrav1.OpenStackCluster) error {
	ip, err := getIPFromInstance(instance)
	if err != nil {
		return err
	}
	loadbalancerService, err := loadbalancer.NewService(osProviderClient, clientOpts, openStackCluster.Spec.UseOctavia)
	if err != nil {
		return err
	}

	if err := loadbalancerService.ReconcileLoadBalancerMember(clusterName, machine, openStackMachine, openStackCluster, ip); err != nil {
		return err
	}
	return nil
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
