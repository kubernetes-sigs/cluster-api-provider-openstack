/*
Copyright 2023 The Kubernetes Authors.

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
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/collections"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha7"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

// SecurityGroupsReconciler reconciles managed security groups in OpenStackCluster and OpenStackMachine objects.
type SecurityGroupsReconciler struct {
	Client           client.Client
	Recorder         record.EventRecorder
	WatchFilterValue string
	ScopeFactory     scope.Factory
	CaCertificates   []byte // PEM encoded ca certificates.
}

func (r *SecurityGroupsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch the OpenStackCluster instance
	openStackCluster := &infrav1.OpenStackCluster{}
	err := r.Client.Get(ctx, req.NamespacedName, openStackCluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Fetch the Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, openStackCluster.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}

	if cluster == nil {
		log.Info("Cluster Controller has not yet set OwnerRef")
		return reconcile.Result{}, nil
	}

	log = log.WithValues("cluster", cluster.Name)

	if annotations.IsPaused(cluster, openStackCluster) {
		log.Info("OpenStackCluster or linked Cluster is marked as paused. Not reconciling")
		return reconcile.Result{}, nil
	}

	patchHelper, err := patch.NewHelper(openStackCluster, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Always patch the openStackCluster when exiting this function so we can persist any OpenStackCluster changes.
	defer func() {
		if err := patchHelper.Patch(ctx, openStackCluster); err != nil {
			if reterr == nil {
				reterr = fmt.Errorf("error patching OpenStackCluster %s/%s: %w", openStackCluster.Namespace, openStackCluster.Name, err)
			}
		}
	}()

	scope, err := r.ScopeFactory.NewClientScopeFromCluster(ctx, r.Client, openStackCluster, r.CaCertificates, log)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Handle deleted SecurityGroups
	if !openStackCluster.DeletionTimestamp.IsZero() {
		return r.reconcileDeleteSecurityGroups(ctx, scope, cluster, openStackCluster)
	}

	// Handle non-deleted SecurityGroups
	return reconcileSecurityGroups(scope, cluster, openStackCluster)
}

func reconcileSecurityGroups(scope scope.Scope, cluster *clusterv1.Cluster, openStackCluster *infrav1.OpenStackCluster) (ctrl.Result, error) { //nolint:unparam
	scope.Logger().Info("Reconciling SecurityGroups")

	clusterName := fmt.Sprintf("%s-%s", cluster.Namespace, cluster.Name)

	networkingService, err := networking.NewService(scope)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = networkingService.ReconcileSecurityGroups(openStackCluster, clusterName)
	if err != nil {
		handleUpdateOSCError(openStackCluster, fmt.Errorf("failed to reconcile security groups: %w", err))
		return ctrl.Result{}, fmt.Errorf("failed to reconcile security groups: %w", err)
	}

	scope.Logger().Info("Reconciled SecurityGroups successfully")
	return ctrl.Result{}, nil
}

func (r *SecurityGroupsReconciler) reconcileDeleteSecurityGroups(ctx context.Context, scope scope.Scope, cluster *clusterv1.Cluster, openStackCluster *infrav1.OpenStackCluster) (ctrl.Result, error) {
	scope.Logger().Info("Reconciling SecurityGroups delete")

	// Wait for machines to be deleted before removing the finalizer as they
	// depend on this resource to deprovision.  Additionally it appears that
	// allowing the Kubernetes API to vanish too quickly will upset the capi
	// kubeadm control plane controller.
	machines, err := collections.GetFilteredMachinesForCluster(ctx, r.Client, cluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(machines) != 0 {
		scope.Logger().Info("Waiting for machines to be deleted", "remaining", len(machines))
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	networkingService, err := networking.NewService(scope)
	if err != nil {
		return reconcile.Result{}, err
	}

	clusterName := fmt.Sprintf("%s-%s", cluster.Namespace, cluster.Name)

	if err = networkingService.DeleteBastionSecurityGroup(openStackCluster, fmt.Sprintf("%s-%s", cluster.Namespace, cluster.Name)); err != nil {
		handleUpdateOSCError(openStackCluster, fmt.Errorf("failed to delete bastion security group: %w", err))
		return reconcile.Result{}, fmt.Errorf("failed to delete bastion security group: %w", err)
	}
	openStackCluster.Status.BastionSecurityGroup = nil

	if err = networkingService.DeleteControlPlaneSecurityGroup(openStackCluster, clusterName); err != nil {
		handleUpdateOSCError(openStackCluster, fmt.Errorf("failed to delete control plane security group: %w", err))
		return reconcile.Result{}, fmt.Errorf("failed to delete control plane security group: %w", err)
	}
	openStackCluster.Status.ControlPlaneSecurityGroup = nil

	if err = networkingService.DeleteWorkerSecurityGroup(openStackCluster, clusterName); err != nil {
		handleUpdateOSCError(openStackCluster, fmt.Errorf("failed to delete worker security group: %w", err))
		return reconcile.Result{}, fmt.Errorf("failed to delete worker security group: %w", err)
	}
	openStackCluster.Status.WorkerSecurityGroup = nil

	scope.Logger().Info("Reconciled SecurityGroups deleted successfully")
	return ctrl.Result{}, nil
}

func (r *SecurityGroupsReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(
			&infrav1.OpenStackMachine{},
			builder.WithPredicates(
				predicate.Funcs{
					// Avoid reconciling if the event triggering the reconciliation is related to incremental status updates
					UpdateFunc: func(e event.UpdateEvent) bool {
						oldMachine := e.ObjectOld.(*infrav1.OpenStackMachine).DeepCopy()
						newMachine := e.ObjectNew.(*infrav1.OpenStackMachine).DeepCopy()
						oldMachine.Status = infrav1.OpenStackMachineStatus{}
						newMachine.Status = infrav1.OpenStackMachineStatus{}
						oldMachine.ObjectMeta.ResourceVersion = ""
						newMachine.ObjectMeta.ResourceVersion = ""
						return !reflect.DeepEqual(oldMachine, newMachine)
					},
				},
			),
		).
		Watches(
			&clusterv1.Machine{},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("OpenStackMachine"))),
		).
		Watches(
			&infrav1.OpenStackCluster{},
			handler.EnqueueRequestsFromMapFunc(r.OpenStackClusterToOpenStackMachines(ctx)),
		).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(ctrl.LoggerFrom(ctx), r.WatchFilterValue)).
		Watches(
			&clusterv1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(r.requeueOpenStackMachinesForUnpausedCluster(ctx)),
			builder.WithPredicates(predicates.ClusterUnpausedAndInfrastructureReady(ctrl.LoggerFrom(ctx))),
		).
		Complete(r)
}

// OpenStackClusterToOpenStackMachines is a handler.ToRequestsFunc to be used to enqeue requests for reconciliation
// of OpenStackMachines.
func (r *SecurityGroupsReconciler) OpenStackClusterToOpenStackMachines(ctx context.Context) handler.MapFunc {
	log := ctrl.LoggerFrom(ctx)
	return func(ctx context.Context, o client.Object) []ctrl.Request {
		c, ok := o.(*infrav1.OpenStackCluster)
		if !ok {
			panic(fmt.Sprintf("Expected a OpenStackCluster but got a %T", o))
		}

		log := log.WithValues("objectMapper", "openStackClusterToOpenStackMachine", "namespace", c.Namespace, "openStackCluster", c.Name)

		// Don't handle deleted OpenStackClusters
		if !c.ObjectMeta.DeletionTimestamp.IsZero() {
			log.V(4).Info("OpenStackClusters has a deletion timestamp, skipping mapping.")
			return nil
		}

		cluster, err := util.GetOwnerCluster(ctx, r.Client, c.ObjectMeta)
		switch {
		case apierrors.IsNotFound(err) || cluster == nil:
			log.V(4).Info("Cluster for OpenStackCluster not found, skipping mapping.")
			return nil
		case err != nil:
			log.Error(err, "Failed to get owning cluster, skipping mapping.")
			return nil
		}

		return r.requestsForCluster(ctx, log, cluster.Namespace, cluster.Name)
	}
}

func (r *SecurityGroupsReconciler) requestsForCluster(ctx context.Context, log logr.Logger, namespace, name string) []ctrl.Request {
	labels := map[string]string{clusterv1.ClusterNameLabel: name}
	machineList := &clusterv1.MachineList{}
	if err := r.Client.List(ctx, machineList, client.InNamespace(namespace), client.MatchingLabels(labels)); err != nil {
		log.Error(err, "Failed to get owned Machines, skipping mapping.")
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

func (r *SecurityGroupsReconciler) requeueOpenStackMachinesForUnpausedCluster(ctx context.Context) handler.MapFunc {
	log := ctrl.LoggerFrom(ctx)
	return func(ctx context.Context, o client.Object) []ctrl.Request {
		c, ok := o.(*clusterv1.Cluster)
		if !ok {
			panic(fmt.Sprintf("Expected a Cluster but got a %T", o))
		}

		log := log.WithValues("objectMapper", "clusterToOpenStackMachine", "namespace", c.Namespace, "cluster", c.Name)

		// Don't handle deleted clusters
		if !c.ObjectMeta.DeletionTimestamp.IsZero() {
			log.V(4).Info("Cluster has a deletion timestamp, skipping mapping.")
			return nil
		}

		return r.requestsForCluster(ctx, log, c.Namespace, c.Name)
	}
}
