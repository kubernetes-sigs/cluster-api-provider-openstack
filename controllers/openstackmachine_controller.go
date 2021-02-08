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

package controllers

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha3"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/compute"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/loadbalancer"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/provider"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	InstanceIDIndex = ".spec.instanceID"

	waitForClusterInfrastructureReadyDuration = 15 * time.Second
)

// OpenStackMachineReconciler reconciles a OpenStackMachine object
type OpenStackMachineReconciler struct {
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch

func (r *OpenStackMachineReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx := context.TODO()
	logger := r.Log.WithValues("namespace", req.Namespace, "openStackMachine", req.Name)

	// Fetch the OpenStackMachine instance.
	openStackMachine := &infrav1.OpenStackMachine{}
	err := r.Get(ctx, req.NamespacedName, openStackMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, openStackMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		logger.Info("Machine Controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("machine", machine.Name)

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		logger.Info("Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("cluster", cluster.Name)

	openStackCluster := &infrav1.OpenStackCluster{}

	openStackClusterName := client.ObjectKey{
		Namespace: openStackMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Client.Get(ctx, openStackClusterName, openStackCluster); err != nil {
		logger.Info("OpenStackCluster is not available yet")
		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("openStackCluster", openStackCluster.Name)

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(openStackMachine, r)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Always patch the openStackMachine when exiting this function so we can persist any OpenStackMachine changes.
	defer func() {
		if err := patchHelper.Patch(ctx, openStackMachine); err != nil {
			if reterr == nil {
				reterr = err
			}
		}
	}()

	// Handle deleted machines
	if !openStackMachine.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, logger, patchHelper, machine, openStackMachine, cluster, openStackCluster)
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(ctx, logger, patchHelper, machine, openStackMachine, cluster, openStackCluster)
}

func (r *OpenStackMachineReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	controller, err := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrav1.OpenStackMachine{}).
		Watches(
			&source.Kind{Type: &clusterv1.Machine{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("OpenStackMachine")),
			},
		).
		Watches(
			&source.Kind{Type: &infrav1.OpenStackCluster{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(r.OpenStackClusterToOpenStackMachines)},
		).
		WithEventFilter(pausedPredicates(r.Log)).
		WithEventFilter(
			predicate.Funcs{
				// Avoid reconciling if the event triggering the reconciliation is related to incremental status updates
				// for OpenStackMachine resources only
				UpdateFunc: func(e event.UpdateEvent) bool {
					if e.ObjectOld.GetObjectKind().GroupVersionKind().Kind != "OpenStackMachine" {
						return true
					}

					oldMachine := e.ObjectOld.(*infrav1.OpenStackMachine).DeepCopy()
					newMachine := e.ObjectNew.(*infrav1.OpenStackMachine).DeepCopy()

					oldMachine.Status = infrav1.OpenStackMachineStatus{}
					newMachine.Status = infrav1.OpenStackMachineStatus{}

					oldMachine.ObjectMeta.ResourceVersion = ""
					newMachine.ObjectMeta.ResourceVersion = ""

					return !reflect.DeepEqual(oldMachine, newMachine)
				},
			},
		).
		Build(r)
	if err != nil {
		return err
	}

	// Add index to OpenStackMachine to find by providerID
	if err := mgr.GetFieldIndexer().IndexField(&infrav1.OpenStackMachine{},
		InstanceIDIndex,
		r.indexOpenStackMachineByInstanceID,
	); err != nil {
		return errors.Wrap(err, "error setting index fields")
	}

	return controller.Watch(
		&source.Kind{Type: &clusterv1.Cluster{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(r.requeueOpenStackMachinesForUnpausedCluster),
		},
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldCluster := e.ObjectOld.(*clusterv1.Cluster)
				newCluster := e.ObjectNew.(*clusterv1.Cluster)
				log := r.Log.WithValues("predicate", "updateEvent", "namespace", newCluster.Namespace, "cluster", newCluster.Name)

				switch {
				// never return true for a paused Cluster
				case newCluster.Spec.Paused:
					log.V(4).Info("Cluster is paused, will not attempt to map associated OpenStackMachine.")
					return false
				// return true if Cluster.Status.InfrastructureReady has changed from false to true
				case !oldCluster.Status.InfrastructureReady && newCluster.Status.InfrastructureReady:
					log.V(4).Info("Cluster InfrastructureReady became ready, will attempt to map associated OpenStackMachine.")
					return true
				// return true if Cluster.Spec.Paused has changed from true to false
				case oldCluster.Spec.Paused && !newCluster.Spec.Paused:
					log.V(4).Info("Cluster was unpaused, will attempt to map associated OpenStackMachine.")
					return true
				// otherwise, return false
				default:
					log.V(4).Info("Cluster did not match expected conditions, will not attempt to map associated OpenStackMachine.")
					return false
				}
			},
			CreateFunc: func(e event.CreateEvent) bool {
				cluster := e.Object.(*clusterv1.Cluster)
				log := r.Log.WithValues("predicateEvent", "create", "namespace", cluster.Namespace, "cluster", cluster.Name)

				// Only need to trigger a reconcile if the Cluster.Spec.Paused is false and
				// Cluster.Status.InfrastructureReady is true
				if !cluster.Spec.Paused && cluster.Status.InfrastructureReady {
					log.V(4).Info("Cluster is not paused and has infrastructure ready, will attempt to map associated OpenStackMachine.")
					return true
				}
				log.V(4).Info("Cluster did not match expected conditions, will not attempt to map associated OpenStackMachine.")
				return false
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				log := r.Log.WithValues("predicateEvent", "delete", "namespace", e.Meta.GetNamespace(), "cluster", e.Meta.GetName())
				log.V(4).Info("Cluster did not match expected conditions, will not attempt to map associated OpenStackMachine.")
				return false
			},
			GenericFunc: func(e event.GenericEvent) bool {
				log := r.Log.WithValues("predicateEvent", "generic", "namespace", e.Meta.GetNamespace(), "cluster", e.Meta.GetName())
				log.V(4).Info("Cluster did not match expected conditions, will not attempt to map associated OpenStackMachine.")
				return false
			},
		},
	)
}

func (r *OpenStackMachineReconciler) reconcileDelete(ctx context.Context, logger logr.Logger, patchHelper *patch.Helper, machine *clusterv1.Machine, openStackMachine *infrav1.OpenStackMachine, cluster *clusterv1.Cluster, openStackCluster *infrav1.OpenStackCluster) (ctrl.Result, error) {
	logger.Info("Handling deleted OpenStackMachine")

	clusterName := fmt.Sprintf("%s-%s", cluster.ObjectMeta.Namespace, cluster.Name)

	osProviderClient, clientOpts, err := provider.NewClientFromMachine(r.Client, openStackMachine)
	if err != nil {
		return ctrl.Result{}, err
	}

	computeService, err := compute.NewService(osProviderClient, clientOpts, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	networkingService, err := networking.NewService(osProviderClient, clientOpts, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	loadBalancerService, err := loadbalancer.NewService(osProviderClient, clientOpts, logger, openStackCluster.Spec.UseOctavia)
	if err != nil {
		return ctrl.Result{}, err
	}
	if openStackCluster.Spec.ManagedAPIServerLoadBalancer {
		err = loadBalancerService.DeleteLoadBalancerMember(clusterName, machine, openStackMachine, openStackCluster)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	instance, err := computeService.InstanceExists(openStackMachine.Name)
	if err != nil {
		return ctrl.Result{}, err
	}

	if instance == nil {
		logger.Info("Skipped deleting machine that is already deleted")
		controllerutil.RemoveFinalizer(openStackMachine, infrav1.MachineFinalizer)
		if err := patchHelper.Patch(ctx, openStackMachine); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// TODO(sbueringer) wait for instance deleted
	err = computeService.InstanceDelete(machine)
	if err != nil {
		handleUpdateMachineError(logger, openStackMachine, errors.Errorf("error deleting Openstack instance: %v", err))
		return ctrl.Result{}, nil
	}
	logger.Info("OpenStack machine deleted successfully")
	r.Recorder.Eventf(openStackMachine, corev1.EventTypeNormal, "SuccessfulDeleteServer", "Deleted server %s with id %s", instance.Name, instance.ID)

	if !openStackCluster.Spec.ManagedAPIServerLoadBalancer && util.IsControlPlaneMachine(machine) && openStackCluster.Spec.APIServerFloatingIP == "" && instance.FloatingIP != "" {
		if err = networkingService.DeleteFloatingIP(instance.FloatingIP); err != nil {
			handleUpdateMachineError(logger, openStackMachine, errors.Errorf("error deleting Openstack floating IP: %v", err))
			return ctrl.Result{}, nil
		}
		logger.Info("OpenStack floating IP deleted successfully", "Floating IP", instance.FloatingIP)
		r.Recorder.Eventf(openStackMachine, corev1.EventTypeNormal, "SuccessfulDeleteFloatingIP", "Deleted floating IP %s", instance.FloatingIP)
	}

	// Instance is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(openStackMachine, infrav1.MachineFinalizer)
	logger.Info("Reconciled Machine delete successfully")
	if err := patchHelper.Patch(ctx, openStackMachine); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *OpenStackMachineReconciler) reconcileNormal(ctx context.Context, logger logr.Logger, patchHelper *patch.Helper, machine *clusterv1.Machine, openStackMachine *infrav1.OpenStackMachine, cluster *clusterv1.Cluster, openStackCluster *infrav1.OpenStackCluster) (_ ctrl.Result, reterr error) {
	// If the OpenStackMachine is in an error state, return early.
	if openStackMachine.Status.FailureReason != nil || openStackMachine.Status.FailureMessage != nil {
		logger.Info("Error state detected, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// If the OpenStackMachine doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(openStackMachine, infrav1.MachineFinalizer)
	// Register the finalizer immediately to avoid orphaning OpenStack resources on delete
	if err := patchHelper.Patch(ctx, openStackMachine); err != nil {
		return ctrl.Result{}, err
	}

	if !cluster.Status.InfrastructureReady {
		logger.Info("Cluster infrastructure is not ready yet, requeuing machine")
		return ctrl.Result{RequeueAfter: waitForClusterInfrastructureReadyDuration}, nil
	}

	// Make sure bootstrap data is available and populated.
	if machine.Spec.Bootstrap.DataSecretName == nil {
		logger.Info("Waiting for bootstrap data to be available")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	userData, err := r.getBootstrapData(machine, openStackMachine)
	if err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("Reconciling Machine create started")

	clusterName := fmt.Sprintf("%s-%s", cluster.ObjectMeta.Namespace, cluster.Name)

	osProviderClient, clientOpts, err := provider.NewClientFromMachine(r.Client, openStackMachine)
	if err != nil {
		return ctrl.Result{}, err
	}

	computeService, err := compute.NewService(osProviderClient, clientOpts, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	networkingService, err := networking.NewService(osProviderClient, clientOpts, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	instance, err := r.getOrCreate(computeService, machine, openStackMachine, cluster, openStackCluster, userData, logger)
	if err != nil {
		handleUpdateMachineError(logger, openStackMachine, errors.Errorf("OpenStack instance cannot be created: %v", err))
		return ctrl.Result{}, err
	}

	// Set an error message if we couldn't find the instance.
	if instance == nil {
		handleUpdateMachineError(logger, openStackMachine, errors.New("OpenStack instance cannot be found"))
		return ctrl.Result{}, nil
	}

	// TODO(sbueringer) From CAPA: TODO(ncdc): move this validation logic into a validating webhook (for us: create validation logic in webhook)

	openStackMachine.Spec.ProviderID = pointer.StringPtr(fmt.Sprintf("openstack:///%s", instance.ID))
	openStackMachine.Spec.InstanceID = pointer.StringPtr(instance.ID)

	openStackMachine.Status.InstanceState = &instance.State

	address := []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: instance.IP}}
	if instance.FloatingIP != "" {
		address = append(address, []corev1.NodeAddress{{Type: corev1.NodeExternalIP, Address: instance.FloatingIP}}...)
	}
	openStackMachine.Status.Addresses = address

	// TODO(sbueringer) From CAPA: TODO(vincepri): Remove this annotation when clusterctl is no longer relevant.
	if openStackMachine.Annotations == nil {
		openStackMachine.Annotations = map[string]string{}
	}
	openStackMachine.Annotations["cluster-api-provider-openstack"] = "true"

	switch instance.State {
	case infrav1.InstanceStateActive:
		logger.Info("Machine instance is ACTIVE", "instance-id", instance.ID)
		openStackMachine.Status.Ready = true
	case infrav1.InstanceStateBuilding:
		logger.Info("Machine instance is BUILDING", "instance-id", instance.ID)
	default:
		handleUpdateMachineError(logger, openStackMachine, errors.Errorf("OpenStack instance state %q is unexpected", instance.State))
		return ctrl.Result{}, nil
	}

	if openStackCluster.Spec.ManagedAPIServerLoadBalancer {
		err = r.reconcileLoadBalancerMember(logger, osProviderClient, clientOpts, instance, clusterName, machine, openStackMachine, openStackCluster)
		if err != nil {
			handleUpdateMachineError(logger, openStackMachine, errors.Errorf("LoadBalancerMember cannot be reconciled: %v", err))
			return ctrl.Result{}, nil
		}
	} else if util.IsControlPlaneMachine(machine) {
		fp, err := networkingService.GetOrCreateFloatingIP(openStackCluster, openStackCluster.Spec.ControlPlaneEndpoint.Host)
		if err != nil {
			handleUpdateMachineError(logger, openStackMachine, errors.Errorf("Floating IP cannot be got or created: %v", err))
			return ctrl.Result{}, nil
		}
		err = computeService.AssociateFloatingIP(instance.ID, fp.FloatingIP)
		if err != nil {
			handleUpdateMachineError(logger, openStackMachine, errors.Errorf("Floating IP cannot be associated: %v", err))
			return ctrl.Result{}, nil
		}
	}

	logger.Info("Reconciled Machine create successfully")
	return ctrl.Result{}, nil
}

func (r *OpenStackMachineReconciler) getOrCreate(computeService *compute.Service, machine *clusterv1.Machine, openStackMachine *infrav1.OpenStackMachine, cluster *clusterv1.Cluster, openStackCluster *infrav1.OpenStackCluster, userData string, logger logr.Logger) (*infrav1.Instance, error) {

	instance, err := computeService.InstanceExists(openStackMachine.Name)
	if err != nil {
		return nil, err
	}

	if instance == nil {
		logger.Info("Machine not exist, Creating Machine", "Machine", openStackMachine.Name)
		instance, err = computeService.InstanceCreate(cluster.Name, machine, openStackMachine, openStackCluster, userData)
		if err != nil {
			return nil, errors.Errorf("error creating Openstack instance: %v", err)
		}
	}

	return instance, nil
}

func handleUpdateMachineError(logger logr.Logger, openstackMachine *infrav1.OpenStackMachine, message error) {
	err := capierrors.UpdateMachineError
	openstackMachine.Status.FailureReason = &err
	openstackMachine.Status.FailureMessage = pointer.StringPtr(message.Error())
	// TODO remove if this error is logged redundantly
	logger.Error(fmt.Errorf(string(err)), message.Error())
}

func (r *OpenStackMachineReconciler) reconcileLoadBalancerMember(logger logr.Logger, osProviderClient *gophercloud.ProviderClient, clientOpts *clientconfig.ClientOpts, instance *infrav1.Instance, clusterName string, machine *clusterv1.Machine, openStackMachine *infrav1.OpenStackMachine, openStackCluster *infrav1.OpenStackCluster) error {
	ip := instance.IP
	loadbalancerService, err := loadbalancer.NewService(osProviderClient, clientOpts, logger, openStackCluster.Spec.UseOctavia)
	if err != nil {
		return err
	}

	if err := loadbalancerService.ReconcileLoadBalancerMember(clusterName, machine, openStackMachine, openStackCluster, ip); err != nil {
		return err
	}
	return nil
}

// OpenStackClusterToOpenStackMachine is a handler.ToRequestsFunc to be used to enqeue requests for reconciliation
// of OpenStackMachines.
func (r *OpenStackMachineReconciler) OpenStackClusterToOpenStackMachines(o handler.MapObject) []ctrl.Request {
	result := []ctrl.Request{}

	c, ok := o.Object.(*infrav1.OpenStackCluster)
	if !ok {
		r.Log.Error(errors.Errorf("expected a OpenStackCluster but got a %T", o.Object), "failed to get OpenStackMachine for OpenStackCluster")
		return nil
	}
	log := r.Log.WithValues("OpenStackCluster", c.Name, "Namespace", c.Namespace)

	cluster, err := util.GetOwnerCluster(context.TODO(), r.Client, c.ObjectMeta)
	switch {
	case apierrors.IsNotFound(err) || cluster == nil:
		return result
	case err != nil:
		log.Error(err, "failed to get owning cluster")
		return result
	}

	labels := map[string]string{clusterv1.ClusterLabelName: cluster.Name}
	machineList := &clusterv1.MachineList{}
	if err := r.List(context.TODO(), machineList, client.InNamespace(c.Namespace), client.MatchingLabels(labels)); err != nil {
		log.Error(err, "failed to list Machines")
		return nil
	}
	for _, m := range machineList.Items {
		if m.Spec.InfrastructureRef.Name == "" {
			continue
		}
		name := client.ObjectKey{Namespace: m.Namespace, Name: m.Spec.InfrastructureRef.Name}
		result = append(result, ctrl.Request{NamespacedName: name})
	}

	return result
}

func (r *OpenStackMachineReconciler) getBootstrapData(machine *clusterv1.Machine, openStackMachine *infrav1.OpenStackMachine) (string, error) {
	if machine.Spec.Bootstrap.DataSecretName == nil {
		return "", errors.New("error retrieving bootstrap data: linked Machine's bootstrap.dataSecretName is nil")
	}

	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: machine.Namespace, Name: *machine.Spec.Bootstrap.DataSecretName}
	if err := r.Client.Get(context.TODO(), key, secret); err != nil {
		return "", errors.Wrapf(err, "failed to retrieve bootstrap data secret for Openstack Machine %s/%s", machine.Namespace, openStackMachine.Name)
	}

	value, ok := secret.Data["value"]
	if !ok {
		return "", errors.New("error retrieving bootstrap data: secret value key is missing")
	}

	return base64.StdEncoding.EncodeToString(value), nil
}

func (r *OpenStackMachineReconciler) requeueOpenStackMachinesForUnpausedCluster(o handler.MapObject) []ctrl.Request {
	c, ok := o.Object.(*clusterv1.Cluster)
	if !ok {
		r.Log.Error(errors.Errorf("expected a Cluster but got a %T", o.Object), "failed to get OpenStackMachines for unpaused Cluster")
		return nil
	}

	// Don't handle deleted clusters
	if !c.ObjectMeta.DeletionTimestamp.IsZero() {
		return nil
	}

	return r.requestsForCluster(c.Namespace, c.Name)
}

func (r *OpenStackMachineReconciler) requestsForCluster(namespace, name string) []ctrl.Request {
	log := r.Log.WithValues("Cluster", name, "Namespace", namespace)
	labels := map[string]string{clusterv1.ClusterLabelName: name}
	machineList := &clusterv1.MachineList{}
	if err := r.Client.List(context.TODO(), machineList, client.InNamespace(namespace), client.MatchingLabels(labels)); err != nil {
		log.Error(err, "failed to get owned Machines")
		return nil
	}

	result := make([]ctrl.Request, 0, len(machineList.Items))
	for _, m := range machineList.Items {
		if m.Spec.InfrastructureRef.Name != "" {
			result = append(result, ctrl.Request{NamespacedName: client.ObjectKey{Namespace: m.Namespace, Name: m.Spec.InfrastructureRef.Name}})
		}
	}
	return result
}

func (r *OpenStackMachineReconciler) indexOpenStackMachineByInstanceID(o runtime.Object) []string {
	openstackMachine, ok := o.(*infrav1.OpenStackMachine)
	if !ok {
		r.Log.Error(errors.New("incorrect type"), "expected an OpenStackMachine", "type", fmt.Sprintf("%T", o))
		return nil
	}

	if openstackMachine.Spec.InstanceID != nil {
		return []string{*openstackMachine.Spec.InstanceID}
	}

	return nil
}
