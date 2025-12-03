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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"sigs.k8s.io/cluster-api/util/predicates"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

// OpenStackMachineTemplateReconciler reconciles a OpenStackMachineTemplate object.
// it only updates the .status field to allow auto-scaling
type OpenStackMachineTemplateReconciler struct {
	Client           client.Client
	Recorder         record.EventRecorder
	WatchFilterValue string
	ScopeFactory     scope.Factory
	CaCertificates   []byte // PEM encoded ca certificates.
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackmachinetemplatess,verbs=get;list;watch;create;
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackmachinetemplatess/status,verbs=get;update;patch

func (r *OpenStackMachineTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch the OpenStackMachine instance.
	openStackMachineTemplate := &infrav1.OpenStackMachine{}
	err := r.Client.Get(ctx, req.NamespacedName, openStackMachineTemplate)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	log = log.WithValues("openStackMachineTemplate", openStackMachineTemplate.Name)
	log.V(4).Info("Reconciling openStackMachineTemplate")

	return ctrl.Result{}, nil
}

func (r *OpenStackMachineTemplateReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrav1.OpenStackMachineTemplate{}).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(mgr.GetScheme(), log, r.WatchFilterValue)).
		// The filter below is required as we only want to reconcile objects created by cluster-api
		// and not users'
		WithEventFilter(predicates.ResourceIsTopologyOwned(mgr.GetScheme(), log)).
		Complete(r)
}
