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
	"fmt"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha2"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/loadbalancer"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/provider"
	"sigs.k8s.io/cluster-api/api/v1alpha2"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	clusterControllerName = "openstackcluster-controller"
)

// OpenStackClusterReconciler reconciles a OpenStackCluster object
type OpenStackClusterReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;list;watch

func (r *OpenStackClusterReconciler) Reconcile(request ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx := context.TODO()
	logger := log.Log.WithName(clusterControllerName).
		WithName(fmt.Sprintf("namespace=%s", request.Namespace)).
		WithName(fmt.Sprintf("openStackCluster=%s", request.Name))

	// Fetch the OpenStackCluster instance
	openStackCluster := &infrav1.OpenStackCluster{}
	err := r.Get(ctx, request.NamespacedName, openStackCluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	logger = logger.WithName(openStackCluster.APIVersion)

	// Fetch the Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, openStackCluster.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if cluster == nil {
		logger.Info("Cluster Controller has not yet set OwnerRef")
		return reconcile.Result{}, nil
	}

	logger = logger.WithName(fmt.Sprintf("cluster=%s", cluster.Name))

	patchHelper, err := patch.NewHelper(openStackCluster, r)
	if err != nil {
		return ctrl.Result{}, err
	}
	defer func() {
		if err := patchHelper.Patch(ctx, openStackCluster); err != nil {
			if reterr == nil {
				reterr = errors.Wrapf(err, "error patching OpenStackCluster %s/%s", openStackCluster.Namespace, openStackCluster.Name)
			}
		}
	}()

	// Handle deleted clusters
	if !openStackCluster.DeletionTimestamp.IsZero() {
		return r.reconcileClusterDelete(logger, cluster, openStackCluster)
	}

	// Handle non-deleted clusters
	return r.reconcileCluster(logger, cluster, openStackCluster)
}

func (r *OpenStackClusterReconciler) reconcileCluster(logger logr.Logger, cluster *v1alpha2.Cluster, openStackCluster *infrav1.OpenStackCluster) (_ ctrl.Result, reterr error) {
	klog.Infof("Reconciling Cluster %s/%s", cluster.Namespace, cluster.Name)

	clusterName := fmt.Sprintf("%s-%s", cluster.Namespace, cluster.Name)

	// If the OpenStackCluster doesn't have our finalizer, add it.
	if !util.Contains(openStackCluster.Finalizers, infrav1.ClusterFinalizer) {
		openStackCluster.Finalizers = append(openStackCluster.Finalizers, infrav1.ClusterFinalizer)
	}

	osProviderClient, clientOpts, err := provider.NewClientFromCluster(r.Client, openStackCluster)
	if err != nil {
		return reconcile.Result{}, err
	}

	networkingService, err := networking.NewService(osProviderClient, clientOpts)
	if err != nil {
		return reconcile.Result{}, err
	}

	loadbalancerService, err := loadbalancer.NewService(osProviderClient, clientOpts, openStackCluster.Spec.UseOctavia)
	if err != nil {
		return reconcile.Result{}, err
	}

	klog.Infof("Reconciling network components for cluster %s", clusterName)
	if openStackCluster.Spec.NodeCIDR == "" {
		klog.V(4).Infof("No need to reconcile network for cluster %s", clusterName)
	} else {
		err := networkingService.ReconcileNetwork(clusterName, openStackCluster)
		if err != nil {
			return reconcile.Result{}, errors.Errorf("failed to reconcile network: %v", err)
		}
		err = networkingService.ReconcileSubnet(clusterName, openStackCluster)
		if err != nil {
			return reconcile.Result{}, errors.Errorf("failed to reconcile subnets: %v", err)
		}
		err = networkingService.ReconcileRouter(clusterName, openStackCluster)
		if err != nil {
			return reconcile.Result{}, errors.Errorf("failed to reconcile router: %v", err)
		}
		if openStackCluster.Spec.ManagedAPIServerLoadBalancer {
			err = loadbalancerService.ReconcileLoadBalancer(clusterName, openStackCluster)
			if err != nil {
				return reconcile.Result{}, errors.Errorf("failed to reconcile load balancer: %v", err)
			}
		}
	}

	err = networkingService.ReconcileSecurityGroups(clusterName, openStackCluster)
	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to reconcile security groups: %v", err)
	}

	// Set APIEndpoints so the Cluster API Cluster Controller can pull them
	if openStackCluster.Spec.ManagedAPIServerLoadBalancer {
		openStackCluster.Status.APIEndpoints = []infrav1.APIEndpoint{
			{
				Host: openStackCluster.Spec.APIServerLoadBalancerFloatingIP,
				Port: openStackCluster.Spec.APIServerLoadBalancerPort,
			},
		}
	} else {
		controlPlaneMachine, err := r.getControlPlaneMachine()
		if err != nil {
			return reconcile.Result{}, errors.Errorf("failed to get control plane machine: %v", err)
		}
		if controlPlaneMachine != nil {
			openStackCluster.Status.APIEndpoints = []infrav1.APIEndpoint{
				{
					Host: controlPlaneMachine.Spec.FloatingIP,
					Port: int(*cluster.Spec.ClusterNetwork.APIServerPort),
				},
			}
		} else {
			klog.Info("No control plane node found yet, could not write OpenStackCluster.Status.APIEndpoints")
		}
	}

	// No errors, so mark us ready so the Cluster API Cluster Controller can pull it
	openStackCluster.Status.Ready = true

	klog.Infof("Reconciled Cluster create %s/%s successfully", cluster.Namespace, cluster.Name)
	return reconcile.Result{}, nil
}

func (r *OpenStackClusterReconciler) reconcileClusterDelete(logger logr.Logger, cluster *v1alpha2.Cluster, openStackCluster *infrav1.OpenStackCluster) (ctrl.Result, error) {

	klog.Infof("Reconcile Cluster delete %s/%s", cluster.Namespace, cluster.Name)
	clusterName := fmt.Sprintf("%s-%s", cluster.Namespace, cluster.Name)
	osProviderClient, clientOpts, err := provider.NewClientFromCluster(r.Client, openStackCluster)
	if err != nil {
		return reconcile.Result{}, err
	}

	networkingService, err := networking.NewService(osProviderClient, clientOpts)
	if err != nil {
		return reconcile.Result{}, err
	}

	loadbalancerService, err := loadbalancer.NewService(osProviderClient, clientOpts, openStackCluster.Spec.UseOctavia)
	if err != nil {
		return reconcile.Result{}, err
	}

	if openStackCluster.Spec.ManagedAPIServerLoadBalancer {
		err = loadbalancerService.DeleteLoadBalancer(clusterName, openStackCluster)
		if err != nil {
			return reconcile.Result{}, errors.Errorf("failed to delete load balancer: %v", err)
		}
	}

	// Delete other things
	if openStackCluster.Status.GlobalSecurityGroup != nil {
		klog.Infof("Deleting global security group %q", openStackCluster.Status.GlobalSecurityGroup.Name)
		err := networkingService.DeleteSecurityGroups(openStackCluster.Status.GlobalSecurityGroup)
		if err != nil {
			return reconcile.Result{}, errors.Errorf("failed to delete security group: %v", err)
		}
	}

	if openStackCluster.Status.ControlPlaneSecurityGroup != nil {
		klog.Infof("Deleting control plane security group %q", openStackCluster.Status.ControlPlaneSecurityGroup.Name)
		err := networkingService.DeleteSecurityGroups(openStackCluster.Status.ControlPlaneSecurityGroup)
		if err != nil {
			return reconcile.Result{}, errors.Errorf("failed to delete security group: %v", err)
		}
	}

	// TODO(sbueringer) Delete network/subnet/router/... if created by CAPO

	klog.Infof("Reconciled Cluster delete %s/%s successfully", cluster.Namespace, cluster.Name)
	// Cluster is deleted so remove the finalizer.
	openStackCluster.Finalizers = util.Filter(openStackCluster.Finalizers, infrav1.ClusterFinalizer)
	return reconcile.Result{}, nil
}

func (r *OpenStackClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.OpenStackCluster{}).
		Complete(r)
}

func (r *OpenStackClusterReconciler) getControlPlaneMachine() (*infrav1.OpenStackMachine, error) {
	machines := &v1alpha2.MachineList{}
	if err := r.Client.List(context.Background(), machines); err != nil {
		return nil, err
	}
	openStackMachines := &infrav1.OpenStackMachineList{}
	if err := r.Client.List(context.Background(), openStackMachines); err != nil {
		return nil, err
	}

	var controlPlaneMachine *v1alpha2.Machine
	for _, machine := range machines.Items {
		if util.IsControlPlaneMachine(&machine) {
			controlPlaneMachine = &machine
			break
		}
	}
	if controlPlaneMachine == nil {
		return nil, nil
	}

	for _, openStackMachine := range openStackMachines.Items {
		if openStackMachine.Name == controlPlaneMachine.Name {
			return &openStackMachine, nil
		}
	}
	return nil, nil
}
