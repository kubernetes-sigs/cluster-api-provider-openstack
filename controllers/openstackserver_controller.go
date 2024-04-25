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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/compute"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

// OpenStackServerReconciler reconciles a OpenstackServer object
type OpenStackServerReconciler struct {
	client.Client
	Recorder         record.EventRecorder
	WatchFilterValue string
	ScopeFactory     scope.Factory
	CaCertificates   []byte // PEM encoded ca certificates.
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackservers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackservers/status,verbs=get;update;patch

func (r *OpenStackServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch the OpenStackServer instance.
	openStackServer := &infrav1.OpenStackServer{}
	err := r.Client.Get(ctx, req.NamespacedName, openStackServer)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	log = log.WithValues("openStackServer", openStackServer.Name)
	log.V(4).Info("Reconciling OpenStackServer")

	if annotations.HasPaused(openStackServer) {
		log.Info("OpenStackServer is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(openStackServer, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Always patch the openStackServer when exiting this function so we can persist any changes.
	defer func() {
		if err := patchServer(ctx, patchHelper, openStackServer); err != nil {
			result = ctrl.Result{}
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}
	}()

	clientScope, err := r.ScopeFactory.NewClientScopeFromServer(ctx, r.Client, openStackServer, r.CaCertificates, log)
	if err != nil {
		return ctrl.Result{}, err
	}
	scope := scope.NewWithLogger(clientScope, log)

	// Handle deleted servers
	if !openStackServer.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(scope, openStackServer)
	}

	// Handle non-deleted servers
	//return r.reconcileNormal(ctx, scope, openStackServer)
}

func patchServer(ctx context.Context, patchHelper *patch.Helper, openStackServer *infrav1.OpenStackServer, options ...patch.Option) error {
	// TODO (emilien): add conditions
	return patchHelper.Patch(ctx, openStackServer, options...)
}

func (r *OpenStackServerReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(
			&infrav1.OpenStackServer{},
			builder.WithPredicates(
				predicate.Funcs{
					// Avoid reconciling if the event triggering the reconciliation is related to incremental status updates
					UpdateFunc: func(e event.UpdateEvent) bool {
						oldServer := e.ObjectOld.(*infrav1.OpenStackServer).DeepCopy()
						newServer := e.ObjectNew.(*infrav1.OpenStackServer).DeepCopy()
						oldServer.Status = infrav1.OpenStackServerStatus{}
						newServer.Status = infrav1.OpenStackServerStatus{}
						oldServer.ObjectMeta.ResourceVersion = ""
						newServer.ObjectMeta.ResourceVersion = ""
						return !reflect.DeepEqual(oldServer, newServer)
					},
				},
			),
		).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(ctrl.LoggerFrom(ctx), r.WatchFilterValue)).
		Complete(r)
}

func (r *OpenStackServerReconciler) reconcileDelete(scope *scope.WithLogger, openStackServer *infrav1.OpenStackServer) (result ctrl.Result, reterr error) {
	scope.Logger().Info("Reconciling Server delete")

	ServerName := openStackServer.Name

	computeService, err := compute.NewService(scope)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Get the Server by name, even if our K8s resource already has the ID field set.
	// TODO(dalees): If this returns a 404 do we try to delete with existing UUID? Do we just assume success?
	server, err := computeService.GetServerByName(ServerName, true)
	// Retry if the failure was anything other than Not Found.
	if err != nil {
		return ctrl.Result{}, err
	}

	if server != nil {
		err = computeService.DeleteServer(Server.ID)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	controllerutil.RemoveFinalizer(openStackServer, infrav1.ServerFinalizer)
	scope.Logger().Info("Reconciled Server delete successfully")
	return ctrl.Result{}, nil
}

func (r *OpenStackServerReconciler) reconcileNormal(ctx context.Context, scope scope.WithLogger, openStackServer *infrav1.OpenStackServer) (result ctrl.Result, reterr error) {
	return ctrl.Result{}, nil
}

func resolveServerResources(scope *scope.WithLogger, openStackServer *infrav1.OpenStackServer) (bool, error) {
	// Resolve and store resources for the server
	resolved := openStackServer.Status.Resolved
	if resolved == nil {
		resolved = &infrav1.ResolvedServerSpec{}
		openStackServer.Status.Resolved = resolved
	}
	changed, err := compute.ResolveMachineSpec(scope,
		openStackCluster.Spec.Bastion.Spec, resolved,
		clusterResourceName, bastionName(clusterResourceName),
		openStackCluster, getBastionSecurityGroupID(openStackCluster))
	if err != nil {
		return false, err
	}
	if changed {
		// If the resolved machine spec changed we need to restart the reconcile to avoid inconsistencies between reconciles.
		return true, nil
	}
	resources := openStackCluster.Status.Bastion.Resources
	if resources == nil {
		resources = &infrav1.ServerResources{}
		openStackCluster.Status.Bastion.Resources = resources
	}

	err = compute.AdoptMachineResources(scope, resolved, resources)
	if err != nil {
		return false, err
	}
	return false, nil
}
