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
	"reflect"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha4"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/compute"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/loadbalancer"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/provider"
)

// OpenStackClusterReconciler reconciles a OpenStackCluster object.
type OpenStackClusterReconciler struct {
	Client           client.Client
	Recorder         record.EventRecorder
	WatchFilterValue string
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

func (r *OpenStackClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
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
		log.Info("OpenStackCluster or linked Cluster is marked as paused. Won't reconcile")
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
				reterr = errors.Wrapf(err, "error patching OpenStackCluster %s/%s", openStackCluster.Namespace, openStackCluster.Name)
			}
		}
	}()

	// Handle deleted clusters
	if !openStackCluster.DeletionTimestamp.IsZero() {
		return reconcileDelete(ctx, log, r.Client, patchHelper, cluster, openStackCluster)
	}

	// Handle non-deleted clusters
	return reconcileNormal(ctx, log, r.Client, patchHelper, cluster, openStackCluster)
}

func reconcileDelete(ctx context.Context, log logr.Logger, client client.Client, patchHelper *patch.Helper, cluster *clusterv1.Cluster, openStackCluster *infrav1.OpenStackCluster) (ctrl.Result, error) {
	log.Info("Reconciling Cluster delete")

	osProviderClient, clientOpts, err := provider.NewClientFromCluster(ctx, client, openStackCluster)
	if err != nil {
		return reconcile.Result{}, err
	}

	if err = deleteBastion(log, osProviderClient, clientOpts, cluster, openStackCluster); err != nil {
		return reconcile.Result{}, err
	}

	networkingService, err := networking.NewService(osProviderClient, clientOpts, log)
	if err != nil {
		return reconcile.Result{}, err
	}

	loadBalancerService, err := loadbalancer.NewService(osProviderClient, clientOpts, log)
	if err != nil {
		return reconcile.Result{}, err
	}

	clusterName := fmt.Sprintf("%s-%s", cluster.Namespace, cluster.Name)

	if openStackCluster.Spec.ManagedAPIServerLoadBalancer {
		if err = loadBalancerService.DeleteLoadBalancer(openStackCluster, clusterName); err != nil {
			handleUpdateOSCError(openStackCluster, errors.Errorf("failed to delete load balancer: %v", err))
			return reconcile.Result{}, errors.Errorf("failed to delete load balancer: %v", err)
		}
	}

	if err = networkingService.DeleteSecurityGroups(openStackCluster, clusterName); err != nil {
		handleUpdateOSCError(openStackCluster, errors.Errorf("failed to delete security groups: %v", err))
		return reconcile.Result{}, errors.Errorf("failed to delete security groups: %v", err)
	}

	// if NodeCIDR was not set, no network was created.
	if openStackCluster.Spec.NodeCIDR != "" {
		if err = networkingService.DeleteRouter(openStackCluster, clusterName); err != nil {
			handleUpdateOSCError(openStackCluster, errors.Errorf("failed to delete router: %v", err))
			return ctrl.Result{}, errors.Errorf("failed to delete router: %v", err)
		}

		if err = networkingService.DeleteNetwork(openStackCluster, clusterName); err != nil {
			handleUpdateOSCError(openStackCluster, errors.Errorf("failed to delete network: %v", err))
			return ctrl.Result{}, errors.Errorf("failed to delete network: %v", err)
		}
	}

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

func deleteBastion(log logr.Logger, osProviderClient *gophercloud.ProviderClient, clientOpts *clientconfig.ClientOpts, cluster *clusterv1.Cluster, openStackCluster *infrav1.OpenStackCluster) error {
	computeService, err := compute.NewService(osProviderClient, clientOpts, log)
	if err != nil {
		return err
	}
	networkingService, err := networking.NewService(osProviderClient, clientOpts, log)
	if err != nil {
		return err
	}

	instanceStatus, err := computeService.GetInstanceStatusByName(openStackCluster, fmt.Sprintf("%s-bastion", cluster.Name))
	if err != nil {
		return err
	}

	if instanceStatus != nil {
		instanceNS, err := instanceStatus.NetworkStatus()
		if err != nil {
			return err
		}

		floatingIP := instanceNS.FloatingIP()
		if floatingIP != "" {
			if err = networkingService.DisassociateFloatingIP(openStackCluster, floatingIP); err != nil {
				handleUpdateOSCError(openStackCluster, errors.Errorf("failed to disassociate floating IP: %v", err))
				return errors.Errorf("failed to disassociate floating IP: %v", err)
			}
			if err = networkingService.DeleteFloatingIP(openStackCluster, floatingIP); err != nil {
				handleUpdateOSCError(openStackCluster, errors.Errorf("failed to delete floating IP: %v", err))
				return errors.Errorf("failed to delete floating IP: %v", err)
			}
		}

		if err = computeService.DeleteInstance(openStackCluster, instanceStatus.InstanceIdentifier()); err != nil {
			handleUpdateOSCError(openStackCluster, errors.Errorf("failed to delete bastion: %v", err))
			return errors.Errorf("failed to delete bastion: %v", err)
		}
	}

	openStackCluster.Status.Bastion = nil

	if err = networkingService.DeleteBastionSecurityGroup(openStackCluster, fmt.Sprintf("%s-%s", cluster.Namespace, cluster.Name)); err != nil {
		handleUpdateOSCError(openStackCluster, errors.Errorf("failed to delete bastion security group: %v", err))
		return errors.Errorf("failed to delete bastion security group: %v", err)
	}
	openStackCluster.Status.BastionSecurityGroup = nil

	return nil
}

func reconcileNormal(ctx context.Context, log logr.Logger, client client.Client, patchHelper *patch.Helper, cluster *clusterv1.Cluster, openStackCluster *infrav1.OpenStackCluster) (ctrl.Result, error) {
	log.Info("Reconciling Cluster")

	// If the OpenStackCluster doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(openStackCluster, infrav1.ClusterFinalizer)
	// Register the finalizer immediately to avoid orphaning OpenStack resources on delete
	if err := patchHelper.Patch(ctx, openStackCluster); err != nil {
		return reconcile.Result{}, err
	}

	osProviderClient, clientOpts, err := provider.NewClientFromCluster(ctx, client, openStackCluster)
	if err != nil {
		return reconcile.Result{}, err
	}

	computeService, err := compute.NewService(osProviderClient, clientOpts, log)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = reconcileNetworkComponents(log, osProviderClient, clientOpts, cluster, openStackCluster)
	if err != nil {
		return reconcile.Result{}, err
	}

	if err = reconcileBastion(log, osProviderClient, clientOpts, cluster, openStackCluster); err != nil {
		return reconcile.Result{}, err
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
	openStackCluster.Status.FailureMessage = nil
	openStackCluster.Status.FailureReason = nil
	log.Info("Reconciled Cluster create successfully")
	return reconcile.Result{}, nil
}

func reconcileBastion(log logr.Logger, osProviderClient *gophercloud.ProviderClient, clientOpts *clientconfig.ClientOpts, cluster *clusterv1.Cluster, openStackCluster *infrav1.OpenStackCluster) error {
	log.Info("Reconciling Bastion")

	if openStackCluster.Spec.Bastion == nil || !openStackCluster.Spec.Bastion.Enabled {
		return deleteBastion(log, osProviderClient, clientOpts, cluster, openStackCluster)
	}

	computeService, err := compute.NewService(osProviderClient, clientOpts, log)
	if err != nil {
		return err
	}

	instanceStatus, err := computeService.GetInstanceStatusByName(openStackCluster, fmt.Sprintf("%s-bastion", cluster.Name))
	if err != nil {
		return err
	}
	if instanceStatus != nil {
		bastion, err := instanceStatus.APIInstance()
		if err != nil {
			return err
		}
		openStackCluster.Status.Bastion = bastion
		return nil
	}

	instanceStatus, err = computeService.CreateBastion(openStackCluster, cluster.Name)
	if err != nil {
		handleUpdateOSCError(openStackCluster, errors.Errorf("failed to reconcile bastion: %v", err))
		return errors.Errorf("failed to reconcile bastion: %v", err)
	}

	networkingService, err := networking.NewService(osProviderClient, clientOpts, log)
	if err != nil {
		return err
	}
	clusterName := fmt.Sprintf("%s-%s", cluster.Namespace, cluster.Name)
	fp, err := networkingService.GetOrCreateFloatingIP(openStackCluster, clusterName, openStackCluster.Spec.Bastion.Instance.FloatingIP)
	if err != nil {
		handleUpdateOSCError(openStackCluster, errors.Errorf("failed to get or create floating IP for bastion: %v", err))
		return errors.Errorf("failed to get or create floating IP for bastion: %v", err)
	}
	port, err := computeService.GetManagementPort(instanceStatus)
	if err != nil {
		err = errors.Errorf("getting management port for bastion: %v", err)
		handleUpdateOSCError(openStackCluster, err)
		return err
	}
	err = networkingService.AssociateFloatingIP(openStackCluster, fp, port.ID)
	if err != nil {
		handleUpdateOSCError(openStackCluster, errors.Errorf("failed to associate floating IP with bastion: %v", err))
		return errors.Errorf("failed to associate floating IP with bastion: %v", err)
	}

	bastion, err := instanceStatus.APIInstance()
	if err != nil {
		return err
	}
	bastion.FloatingIP = fp.FloatingIP
	openStackCluster.Status.Bastion = bastion
	return nil
}

func reconcileNetworkComponents(log logr.Logger, osProviderClient *gophercloud.ProviderClient, clientOpts *clientconfig.ClientOpts, cluster *clusterv1.Cluster, openStackCluster *infrav1.OpenStackCluster) error {
	clusterName := fmt.Sprintf("%s-%s", cluster.Namespace, cluster.Name)

	networkingService, err := networking.NewService(osProviderClient, clientOpts, log)
	if err != nil {
		return err
	}

	loadBalancerService, err := loadbalancer.NewService(osProviderClient, clientOpts, log)
	if err != nil {
		return err
	}

	log.Info("Reconciling network components")

	err = networkingService.ReconcileExternalNetwork(openStackCluster)
	if err != nil {
		handleUpdateOSCError(openStackCluster, errors.Errorf("failed to reconcile external network: %v", err))
		return errors.Errorf("failed to reconcile external network: %v", err)
	}

	if openStackCluster.Spec.NodeCIDR == "" {
		log.V(4).Info("No need to reconcile network, searching network and subnet instead")

		netOpts := networks.ListOpts(openStackCluster.Spec.Network)
		networkList, err := networkingService.GetNetworksByFilter(&netOpts)
		if err != nil {
			handleUpdateOSCError(openStackCluster, errors.Errorf("failed to find network: %v", err))
			return errors.Errorf("failed to find network: %v", err)
		}
		if len(networkList) == 0 {
			handleUpdateOSCError(openStackCluster, errors.Errorf("failed to find any network: %v", err))
			return errors.Errorf("failed to find any network: %v", err)
		}
		if len(networkList) > 1 {
			handleUpdateOSCError(openStackCluster, errors.Errorf("failed to find only one network (result: %v): %v", networkList, err))
			return errors.Errorf("failed to find only one network (result: %v): %v", networkList, err)
		}
		openStackCluster.Status.Network = &infrav1.Network{
			ID:   networkList[0].ID,
			Name: networkList[0].Name,
			Tags: networkList[0].Tags,
		}

		subnetOpts := subnets.ListOpts(openStackCluster.Spec.Subnet)
		subnetOpts.NetworkID = networkList[0].ID
		subnetList, err := networkingService.GetSubnetsByFilter(&subnetOpts)
		if err != nil || len(subnetList) == 0 {
			handleUpdateOSCError(openStackCluster, errors.Errorf("failed to find subnet: %v", err))
			return errors.Errorf("failed to find subnet: %v", err)
		}
		if len(subnetList) > 1 {
			handleUpdateOSCError(openStackCluster, errors.Errorf("failed to find only one subnet (result: %v): %v", subnetList, err))
			return errors.Errorf("failed to find only one subnet (result: %v): %v", subnetList, err)
		}
		openStackCluster.Status.Network.Subnet = &infrav1.Subnet{
			ID:   subnetList[0].ID,
			Name: subnetList[0].Name,
			CIDR: subnetList[0].CIDR,
			Tags: subnetList[0].Tags,
		}
	} else {
		err := networkingService.ReconcileNetwork(openStackCluster, clusterName)
		if err != nil {
			handleUpdateOSCError(openStackCluster, errors.Errorf("failed to reconcile network: %v", err))
			return errors.Errorf("failed to reconcile network: %v", err)
		}
		err = networkingService.ReconcileSubnet(openStackCluster, clusterName)
		if err != nil {
			handleUpdateOSCError(openStackCluster, errors.Errorf("failed to reconcile subnets: %v", err))
			return errors.Errorf("failed to reconcile subnets: %v", err)
		}
		err = networkingService.ReconcileRouter(openStackCluster, clusterName)
		if err != nil {
			handleUpdateOSCError(openStackCluster, errors.Errorf("failed to reconcile router: %v", err))
			return errors.Errorf("failed to reconcile router: %v", err)
		}
	}
	if !openStackCluster.Spec.ControlPlaneEndpoint.IsValid() {
		var port int32
		if openStackCluster.Spec.APIServerPort == 0 {
			port = 6443
		} else {
			port = int32(openStackCluster.Spec.APIServerPort)
		}
		fp, err := networkingService.GetOrCreateFloatingIP(openStackCluster, clusterName, openStackCluster.Spec.APIServerFloatingIP)
		if err != nil {
			handleUpdateOSCError(openStackCluster, errors.Errorf("Floating IP cannot be got or created: %v", err))
			return errors.Errorf("Floating IP cannot be got or created: %v", err)
		}
		// Set APIEndpoints so the Cluster API Cluster Controller can pull them
		openStackCluster.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{
			Host: fp.FloatingIP,
			Port: port,
		}
	}

	err = networkingService.ReconcileSecurityGroups(openStackCluster, clusterName)
	if err != nil {
		handleUpdateOSCError(openStackCluster, errors.Errorf("failed to reconcile security groups: %v", err))
		return errors.Errorf("failed to reconcile security groups: %v", err)
	}

	if openStackCluster.Spec.ManagedAPIServerLoadBalancer {
		err = loadBalancerService.ReconcileLoadBalancer(openStackCluster, clusterName)
		if err != nil {
			handleUpdateOSCError(openStackCluster, errors.Errorf("failed to reconcile load balancer: %v", err))
			return errors.Errorf("failed to reconcile load balancer: %v", err)
		}
	}

	return nil
}

func (r *OpenStackClusterReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	clusterToInfraFn := util.ClusterToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("OpenStackCluster"))
	log := ctrl.LoggerFrom(ctx)

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrav1.OpenStackCluster{},
			builder.WithPredicates(
				predicate.Funcs{
					// Avoid reconciling if the event triggering the reconciliation is related to incremental status updates
					UpdateFunc: func(e event.UpdateEvent) bool {
						oldCluster := e.ObjectOld.(*infrav1.OpenStackCluster).DeepCopy()
						newCluster := e.ObjectNew.(*infrav1.OpenStackCluster).DeepCopy()
						oldCluster.Status = infrav1.OpenStackClusterStatus{}
						newCluster.Status = infrav1.OpenStackClusterStatus{}
						oldCluster.ObjectMeta.ResourceVersion = ""
						newCluster.ObjectMeta.ResourceVersion = ""
						return !reflect.DeepEqual(oldCluster, newCluster)
					},
				},
			),
		).
		Watches(
			&source.Kind{Type: &clusterv1.Cluster{}},
			handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
				requests := clusterToInfraFn(o)
				if len(requests) < 1 {
					return nil
				}

				c := &infrav1.OpenStackCluster{}
				if err := r.Client.Get(ctx, requests[0].NamespacedName, c); err != nil {
					log.V(4).Error(err, "Failed to get OpenStack cluster")
					return nil
				}

				if annotations.IsExternallyManaged(c) {
					log.V(4).Info("OpenStackCluster is externally managed, skipping mapping.")
					return nil
				}
				return requests
			}),
			builder.WithPredicates(predicates.ClusterUnpaused(ctrl.LoggerFrom(ctx))),
		).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(ctrl.LoggerFrom(ctx), r.WatchFilterValue)).
		WithEventFilter(predicates.ResourceIsNotExternallyManaged(ctrl.LoggerFrom(ctx))).
		Complete(r)
}

func handleUpdateOSCError(openstackCluster *infrav1.OpenStackCluster, message error) {
	err := capierrors.UpdateClusterError
	openstackCluster.Status.FailureReason = &err
	openstackCluster.Status.FailureMessage = pointer.StringPtr(message.Error())
}
