/*
Copyright 2019 The Kubernetes Authors.

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
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha3"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/compute"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/loadbalancer"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/provider"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// OpenStackClusterReconciler reconciles a OpenStackCluster object
type OpenStackClusterReconciler struct {
	client.Client
	Recorder record.EventRecorder
	Log      logr.Logger
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

func (r *OpenStackClusterReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx := context.TODO()
	log := r.Log.WithValues("namespace", req.Namespace, "openStackCluster", req.Name)

	// Fetch the OpenStackCluster instance
	openStackCluster := &infrav1.OpenStackCluster{}
	err := r.Get(ctx, req.NamespacedName, openStackCluster)
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

	if isPaused(cluster, openStackCluster) {
		log.Info("OpenStackCluster or linked Cluster is marked as paused. Won't reconcile")
		return reconcile.Result{}, nil
	}

	if cluster == nil {
		log.Info("Cluster Controller has not yet set OwnerRef")
		return reconcile.Result{}, nil
	}

	log = log.WithValues("cluster", cluster.Name)

	patchHelper, err := patch.NewHelper(openStackCluster, r)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Always patch the openStackCluster when exiting this function so we can persist any OpenStackCluster changes.
	defer func() {
		if err := patchHelper.Patch(ctx, openStackCluster); err != nil {
			if reterr == nil {
				reterr = errors.Wrapf(err, "error patching OpenStackCluster %s/%s", openStackCluster.Namespace, openStackCluster.Name)
			}
		}
	}()

	// Handle deleted clusters
	if !openStackCluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, log, patchHelper, cluster, openStackCluster)
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(ctx, log, patchHelper, cluster, openStackCluster)
}

func (r *OpenStackClusterReconciler) reconcileDelete(ctx context.Context, log logr.Logger, patchHelper *patch.Helper, cluster *clusterv1.Cluster, openStackCluster *infrav1.OpenStackCluster) (ctrl.Result, error) {
	log.Info("Reconciling Cluster delete")

	clusterName := fmt.Sprintf("%s-%s", cluster.Namespace, cluster.Name)
	osProviderClient, clientOpts, err := provider.NewClientFromCluster(r.Client, openStackCluster)
	if err != nil {
		return reconcile.Result{}, err
	}

	networkingService, err := networking.NewService(osProviderClient, clientOpts, log)
	if err != nil {
		return reconcile.Result{}, err
	}

	loadBalancerService, err := loadbalancer.NewService(osProviderClient, clientOpts, log, openStackCluster.Spec.UseOctavia)
	if err != nil {
		return reconcile.Result{}, err
	}

	if openStackCluster.Spec.ManagedAPIServerLoadBalancer {
		err = loadBalancerService.DeleteLoadBalancer(clusterName, openStackCluster)
		if err != nil {
			return reconcile.Result{}, errors.Errorf("failed to delete load balancer: %v", err)
		}
	}

	// Delete other things
	if openStackCluster.Status.GlobalSecurityGroup != nil {
		log.Info("Deleting global security group", "name", openStackCluster.Status.GlobalSecurityGroup.Name)
		err := networkingService.DeleteSecurityGroups(openStackCluster.Status.GlobalSecurityGroup)
		if err != nil {
			return reconcile.Result{}, errors.Errorf("failed to delete security group: %v", err)
		}
	}

	if openStackCluster.Status.ControlPlaneSecurityGroup != nil {
		log.Info("Deleting control plane security group", "name", openStackCluster.Status.ControlPlaneSecurityGroup.Name)
		err := networkingService.DeleteSecurityGroups(openStackCluster.Status.ControlPlaneSecurityGroup)
		if err != nil {
			return reconcile.Result{}, errors.Errorf("failed to delete security group: %v", err)
		}
	}

	if openStackCluster.Status.Network.Router != nil {
		log.Info("Deleting router", "name", openStackCluster.Status.Network.Router.Name)
		if err := networkingService.DeleteRouter(openStackCluster.Status.Network); err != nil {
			return ctrl.Result{}, errors.Errorf("failed to delete router: %v", err)
		}
		log.Info("OpenStack router deleted successfully")
	}

	if openStackCluster.Status.Network != nil {
		log.Info("Deleting network", "name", openStackCluster.Status.Network.Name)
		if err := networkingService.DeleteNetwork(openStackCluster.Status.Network); err != nil {
			return ctrl.Result{}, errors.Errorf("failed to delete network: %v", err)
		}
		log.Info("OpenStack network deleted successfully")
	}

	log.Info("OpenStack cluster deleted successfully")

	// Cluster is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(openStackCluster, infrav1.ClusterFinalizer)
	log.Info("Reconciled Cluster delete successfully")
	if err := patchHelper.Patch(ctx, openStackCluster); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func contains(arr []string, target string) bool {
	for _, a := range arr {
		if a == target {
			return true
		}
	}
	return false
}

func (r *OpenStackClusterReconciler) reconcileNormal(ctx context.Context, log logr.Logger, patchHelper *patch.Helper, cluster *clusterv1.Cluster, openStackCluster *infrav1.OpenStackCluster) (ctrl.Result, error) {
	log.Info("Reconciling Cluster")

	clusterName := fmt.Sprintf("%s-%s", cluster.Namespace, cluster.Name)

	// If the OpenStackCluster doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(openStackCluster, infrav1.ClusterFinalizer)
	// Register the finalizer immediately to avoid orphaning OpenStack resources on delete
	if err := patchHelper.Patch(ctx, openStackCluster); err != nil {
		return reconcile.Result{}, err
	}

	osProviderClient, clientOpts, err := provider.NewClientFromCluster(r.Client, openStackCluster)
	if err != nil {
		return reconcile.Result{}, err
	}

	computeService, err := compute.NewService(osProviderClient, clientOpts, log)
	if err != nil {
		return reconcile.Result{}, err
	}

	networkingService, err := networking.NewService(osProviderClient, clientOpts, log)
	if err != nil {
		return reconcile.Result{}, err
	}

	loadBalancerService, err := loadbalancer.NewService(osProviderClient, clientOpts, log, openStackCluster.Spec.UseOctavia)
	if err != nil {
		return reconcile.Result{}, err
	}

	log.Info("Reconciling network components")
	if openStackCluster.Spec.NodeCIDR == "" {
		log.V(4).Info("No need to reconcile network, searching network and subnet instead")

		netOpts := networks.ListOpts(openStackCluster.Spec.Network)
		networkList, err := networkingService.GetNetworksByFilter(&netOpts)
		if err != nil && len(networkList) == 0 {
			return reconcile.Result{}, errors.Errorf("failed to find network: %v", err)
		}
		if len(networkList) > 1 {
			return reconcile.Result{}, errors.Errorf("failed to find only one network (result: %v): %v", networkList, err)
		}
		openStackCluster.Status.Network = &infrav1.Network{
			ID:   networkList[0].ID,
			Name: networkList[0].Name,
		}

		subnetOpts := subnets.ListOpts(openStackCluster.Spec.Subnet)
		subnetOpts.NetworkID = networkList[0].ID
		subnetList, err := networkingService.GetSubnetsByFilter(&subnetOpts)
		if err != nil || len(subnetList) == 0 {
			return reconcile.Result{}, errors.Errorf("failed to find subnet: %v", err)
		}
		if len(subnetList) > 1 {
			return reconcile.Result{}, errors.Errorf("failed to find only one subnet (result: %v): %v", subnetList, err)
		}
		openStackCluster.Status.Network.Subnet = &infrav1.Subnet{
			ID:   subnetList[0].ID,
			Name: subnetList[0].Name,
			CIDR: subnetList[0].CIDR,
		}
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
			err = loadBalancerService.ReconcileLoadBalancer(clusterName, openStackCluster)
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
		openStackCluster.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{
			Host: openStackCluster.Spec.APIServerLoadBalancerFloatingIP,
			Port: int32(openStackCluster.Spec.APIServerLoadBalancerPort),
		}
	} else {
		controlPlaneMachine, err := getControlPlaneMachine(r.Client)
		if err != nil {
			return ctrl.Result{}, errors.Errorf("failed to get control plane machine: %v", err)
		}
		if controlPlaneMachine != nil {
			var apiPort int
			if cluster.Spec.ClusterNetwork.APIServerPort == nil {
				log.Info("No API endpoint given, default to 6443")
				apiPort = 6443
			} else {
				apiPort = int(*cluster.Spec.ClusterNetwork.APIServerPort)
			}

			openStackCluster.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{
				Host: controlPlaneMachine.Spec.FloatingIP,
				Port: int32(apiPort),
			}
		} else {
			log.Info("No control plane node found yet, could not write OpenStackCluster.Spec.ControlPlaneEndpoint")
		}
	}

	availabilityZones, err := computeService.GetAvailabilityZones()
	if err != nil {
		return ctrl.Result{}, err
	}
	if openStackCluster.Status.FailureDomains == nil {
		openStackCluster.Status.FailureDomains = make(clusterv1.FailureDomains)
	}
	for _, az := range availabilityZones {
		// I'm actually not sure if that's just my local devstack,
		// but we probably shouldn't use the "internal" AZ
		if az.ZoneName == "internal" {
			continue
		}

		found := true
		// If Az given, then check whether it's in the allow list
		// If no Az given, then by default put into allow list
		if len(openStackCluster.Spec.ControlPlaneAvailabilityZones) > 0 {
			if contains(openStackCluster.Spec.ControlPlaneAvailabilityZones, az.ZoneName) {
				found = true
			} else {
				found = false
			}
		}

		openStackCluster.Status.FailureDomains[az.ZoneName] = clusterv1.FailureDomainSpec{
			ControlPlane: found,
		}
	}

	openStackCluster.Status.Ready = true
	log.Info("Reconciled Cluster create successfully")
	return ctrl.Result{}, nil
}

func (r *OpenStackClusterReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrav1.OpenStackCluster{}).
		WithEventFilter(pausePredicates).
		Complete(r)
}

func getControlPlaneMachine(client client.Client) (*infrav1.OpenStackMachine, error) {
	machines := &clusterv1.MachineList{}
	if err := client.List(context.TODO(), machines); err != nil {
		return nil, err
	}
	openStackMachines := &infrav1.OpenStackMachineList{}
	if err := client.List(context.TODO(), openStackMachines); err != nil {
		return nil, err
	}

	var controlPlaneMachine *clusterv1.Machine
	for _, machine := range machines.Items {
		m := machine
		if util.IsControlPlaneMachine(&m) {
			controlPlaneMachine = &m
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
