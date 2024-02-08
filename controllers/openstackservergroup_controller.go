/*
Copyright 2024 The Kubernetes Authors.

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
	"reflect"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/servergroups"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha8"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/compute"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

// OpenStackServerGroupReconciler reconciles a OpenstackServerGroup object
type OpenStackServerGroupReconciler struct {
	client.Client
	Recorder         record.EventRecorder
	WatchFilterValue string
	ScopeFactory     scope.Factory
	CaCertificates   []byte // PEM encoded ca certificates.
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackservergroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackservergroups/status,verbs=get;update;patch

func (r *OpenStackServerGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch the OpenStackMachine instance.
	openStackServerGroup := &infrav1.OpenStackServerGroup{}
	err := r.Client.Get(ctx, req.NamespacedName, openStackServerGroup)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	log = log.WithValues("openStackServerGroup", openStackServerGroup.Name)

	log.Info("OpenStackServerGroup is about to reconcile")

	if annotations.HasPaused(openStackServerGroup) {
		log.Info("OpenStackServerGroup is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(openStackServerGroup, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Always patch the openStackServerGroup when exiting this function so we can persist any changes.
	defer func() {
		if err := patchServerGroup(ctx, patchHelper, openStackServerGroup); err != nil {
			result = ctrl.Result{}
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}
	}()

	scope, err := r.ScopeFactory.NewClientScopeFromServerGroup(ctx, r.Client, openStackServerGroup, r.CaCertificates, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Handle deleted servergroups
	if !openStackServerGroup.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(scope, openStackServerGroup)
	}

	// Handle non-deleted servergroups
	return r.reconcileNormal(ctx, scope, openStackServerGroup)
}

func patchServerGroup(ctx context.Context, patchHelper *patch.Helper, openStackServerGroup *infrav1.OpenStackServerGroup, options ...patch.Option) error {
	return patchHelper.Patch(ctx, openStackServerGroup, options...)
}

func (r *OpenStackServerGroupReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&infrav1.OpenStackServerGroup{},
			builder.WithPredicates(
				predicate.Funcs{
					// Avoid reconciling if the event triggering the reconciliation is related to incremental status updates
					UpdateFunc: func(e event.UpdateEvent) bool {
						oldServerGroup := e.ObjectOld.(*infrav1.OpenStackServerGroup).DeepCopy()
						newServerGroup := e.ObjectNew.(*infrav1.OpenStackServerGroup).DeepCopy()
						oldServerGroup.Status = infrav1.OpenStackServerGroupStatus{}
						newServerGroup.Status = infrav1.OpenStackServerGroupStatus{}
						oldServerGroup.ObjectMeta.ResourceVersion = ""
						newServerGroup.ObjectMeta.ResourceVersion = ""
						return !reflect.DeepEqual(oldServerGroup, newServerGroup)
					},
				},
			),
		).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(ctrl.LoggerFrom(ctx), r.WatchFilterValue)).
		Complete(r)
}

func (r *OpenStackServerGroupReconciler) reconcileDelete(scope scope.Scope, openStackServerGroup *infrav1.OpenStackServerGroup) (result ctrl.Result, reterr error) {
	scope.Logger().Info("Reconciling ServerGroup delete")

	serverGroupName := openStackServerGroup.Name

	computeService, err := compute.NewService(scope)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Get the servergroup by name, even if our K8s resource already has the ID field set.
	// TODO(dalees): If this returns a 404 do we try to delete with existing UUID? Do we just assume success?
	serverGroup, err := computeService.GetServerGroupByName(serverGroupName, true)
	// Retry if the failure was anything other than Not Found.
	if err != nil {
		return ctrl.Result{}, err
	}

	if serverGroup != nil {
		err = computeService.DeleteServerGroup(serverGroup.ID)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	controllerutil.RemoveFinalizer(openStackServerGroup, infrav1.ServerGroupFinalizer)
	scope.Logger().Info("Reconciled ServerGroup delete successfully")
	return ctrl.Result{}, nil
}

func (r *OpenStackServerGroupReconciler) reconcileNormal(ctx context.Context, scope scope.Scope, openStackServerGroup *infrav1.OpenStackServerGroup) (result ctrl.Result, reterr error) {

	// If the OpenStackServerGroup doesn't have our finalizer, add it.
	if controllerutil.AddFinalizer(openStackServerGroup, infrav1.ServerGroupFinalizer) {
		scope.Logger().Info("Reconciling ServerGroup: Adding finalizer")
		// Register the finalizer immediately to avoid orphaning OpenStack resources on delete
		// NOTE(dalees): This return without Requeue relies on patchServerGroup to persist the change, and the watch triggers an immediate reconcile.
		return ctrl.Result{}, nil
	}

	scope.Logger().Info("Reconciling ServerGroup")

	computeService, err := compute.NewService(scope)
	if err != nil {
		return ctrl.Result{}, err
	}

	serverGroupStatus, err := r.getOrCreate(scope.Logger(), openStackServerGroup, computeService)
	if err != nil || serverGroupStatus == nil {
		return ctrl.Result{}, err
	}

	// Update the resource with any new information.
	openStackServerGroup.Status.Ready = true
	openStackServerGroup.Status.ID = serverGroupStatus.ID

	scope.Logger().Info("Reconciled ServerGroup successfully")
	return ctrl.Result{}, nil
}

func (r *OpenStackServerGroupReconciler) getOrCreate(logger logr.Logger, openStackServerGroup *infrav1.OpenStackServerGroup, computeService *compute.Service) (*servergroups.ServerGroup, error) {

	serverGroupName := openStackServerGroup.Name

	serverGroup, err := computeService.GetServerGroupByName(serverGroupName, false)
	if err != nil && serverGroup != nil {
		// More than one server group was found with the same name.
		// We should not create another, nor should we use the first found.
		return nil, err
	}
	if err == nil {
		return serverGroup, nil
	}

	logger.Info("Unable to get ServerGroup instance, we need to create it.", "name", serverGroupName, "policy", openStackServerGroup.Spec.Policy)

	serverGroup, err = computeService.CreateServerGroup(serverGroupName, openStackServerGroup.Spec.Policy)
	if err != nil {
		return nil, err
	}

	return serverGroup, err
}
