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
	"errors"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	ipamv1 "sigs.k8s.io/cluster-api/api/ipam/v1beta2"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	v1beta1conditions "sigs.k8s.io/cluster-api/util/deprecated/v1beta1/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1alpha1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

const (
	openStackFloatingIPPool = "OpenStackFloatingIPPool"
)

var errMaxIPsReached = errors.New("maximum number of IPs reached")

var backoff = wait.Backoff{
	Steps:    4,
	Duration: 10 * time.Millisecond,
	Factor:   5.0,
	Jitter:   0.1,
}

// OpenStackFloatingIPPoolReconciler reconciles a OpenStackFloatingIPPool object.
type OpenStackFloatingIPPoolReconciler struct {
	Client           client.Client
	Recorder         record.EventRecorder
	WatchFilterValue string
	ScopeFactory     scope.Factory
	CaCertificates   []byte // PEM encoded ca certificates.

	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackfloatingippools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackfloatingippools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddressclaims;ipaddressclaims/status,verbs=get;list;watch;update;create;delete
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddresses;ipaddresses/status,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

func (r *OpenStackFloatingIPPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)
	pool := &infrav1alpha1.OpenStackFloatingIPPool{}
	if err := r.Client.Get(ctx, req.NamespacedName, pool); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	patchHelper, err := patch.NewHelper(pool, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	defer func() {
		if err := patchHelper.Patch(ctx, pool); err != nil {
			if reterr == nil {
				reterr = fmt.Errorf("error patching OpenStackFloatingIPPool %s/%s: %w", pool.Namespace, pool.Name, err)
			}
		}
	}()

	clientScope, err := r.ScopeFactory.NewClientScopeFromObject(ctx, r.Client, r.CaCertificates, log, pool)
	if err != nil {
		v1beta1conditions.MarkFalse(pool, infrav1.OpenStackAuthenticationSucceeded, infrav1.OpenStackAuthenticationFailedReason, clusterv1beta1.ConditionSeverityError, "Failed to create OpenStack client scope: %v", err)
		return reconcile.Result{}, err
	}
	v1beta1conditions.MarkTrue(pool, infrav1.OpenStackAuthenticationSucceeded)
	scope := scope.NewWithLogger(clientScope, log)

	// This is done before deleting the pool, because we want to handle deleted IPs before we delete the pool
	if err := r.reconcileIPAddresses(ctx, scope, pool); err != nil {
		return ctrl.Result{}, err
	}

	if pool.DeletionTimestamp.IsZero() {
		// Add finalizer if it does not exist
		if controllerutil.AddFinalizer(pool, infrav1alpha1.OpenStackFloatingIPPoolFinalizer) {
			return ctrl.Result{}, r.Client.Update(ctx, pool)
		}
	} else {
		// Handle deletion
		return ctrl.Result{}, r.reconcileDelete(ctx, scope, pool)
	}

	if err := r.reconcileFloatingIPNetwork(scope, pool); err != nil {
		return ctrl.Result{}, err
	}

	claims := &ipamv1.IPAddressClaimList{}
	if err := r.Client.List(context.Background(), claims, client.InNamespace(req.Namespace), client.MatchingFields{infrav1alpha1.OpenStackFloatingIPPoolNameIndex: pool.Name}); err != nil {
		return ctrl.Result{}, err
	}

	for _, claim := range claims.Items {
		log := log.WithValues("claim", claim.Name)

		cluster, err := util.GetClusterFromMetadata(ctx, r.Client, claim.ObjectMeta)
		if err != nil {
			log.Error(err, "Failed to get owning cluster, skipping claim", "claim", claim.Name)
			continue
		}

		// Add finalizer if it does not exist
		if controllerutil.AddFinalizer(&claim, infrav1alpha1.OpenStackFloatingIPPoolFinalizer) {
			return ctrl.Result{}, r.Client.Update(ctx, &claim)
		}

		if annotations.IsPaused(cluster, &claim) {
			log.V(4).Info("IPAddressClaim or linked Cluster is paused, skipping reconcile", "claim", claim.Name, "namespace", claim.Namespace)
			return reconcile.Result{}, nil
		}

		if !claim.DeletionTimestamp.IsZero() {
			controllerutil.RemoveFinalizer(&claim, infrav1alpha1.OpenStackFloatingIPPoolFinalizer)
			if err := r.Client.Update(ctx, &claim); err != nil {
				return ctrl.Result{}, err
			}
			continue
		}

		if claim.Status.AddressRef.Name == "" {
			ipAddress := &ipamv1.IPAddress{}
			err := r.Client.Get(ctx, client.ObjectKey{Name: claim.Name, Namespace: claim.Namespace}, ipAddress)
			if client.IgnoreNotFound(err) != nil {
				return ctrl.Result{}, err
			}
			if apierrors.IsNotFound(err) {
				ip, err := r.getIP(ctx, scope, pool)
				if err != nil {
					if errors.Is(err, errMaxIPsReached) {
						log.Info("Maximum number of IPs reached, will not allocate more IPs.")
						return ctrl.Result{}, nil
					}
					return ctrl.Result{}, err
				}

				ipAddress = &ipamv1.IPAddress{
					ObjectMeta: ctrl.ObjectMeta{
						Name:      claim.Name,
						Namespace: claim.Namespace,
						Finalizers: []string{
							infrav1alpha1.DeleteFloatingIPFinalizer,
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: claim.APIVersion,
								Kind:       claim.Kind,
								Name:       claim.Name,
								UID:        claim.UID,
							},
						},
					},
					Spec: ipamv1.IPAddressSpec{
						ClaimRef: ipamv1.IPAddressClaimReference{
							Name: claim.Name,
						},
						PoolRef: ipamv1.IPPoolReference{
							APIGroup: infrav1alpha1.SchemeGroupVersion.Group,
							Kind:     pool.Kind,
							Name:     pool.Name,
						},
						Address: ip,
						Prefix:  ptr.To(int32(32)),
					},
				}

				// Retry creating the IPAddress object
				err = wait.ExponentialBackoffWithContext(ctx, backoff, func(ctx context.Context) (bool, error) {
					if err := r.Client.Create(ctx, ipAddress); err != nil {
						return false, err
					}
					return true, nil
				})
				if err != nil {
					// If we failed to create the IPAddress, there might be an IP leak in OpenStack if we also failed to tag the IP after creation
					scope.Logger().Error(err, "Failed to create IPAddress", "ip", ip)
					return ctrl.Result{}, err
				}
			}
			claim.Status.AddressRef.Name = ipAddress.Name
			if err = r.Client.Status().Update(ctx, &claim); err != nil {
				log.Error(err, "Failed to update IPAddressClaim status", "claim", claim.Name, "ipaddress", ipAddress.Name)
				return ctrl.Result{}, err
			}
			scope.Logger().Info("Claimed IP", "ip", ipAddress.Spec.Address)
		}
	}
	v1beta1conditions.MarkTrue(pool, infrav1alpha1.OpenstackFloatingIPPoolReadyCondition)
	return ctrl.Result{}, r.Client.Status().Update(ctx, pool)
}

func (r *OpenStackFloatingIPPoolReconciler) reconcileDelete(ctx context.Context, scope *scope.WithLogger, pool *infrav1alpha1.OpenStackFloatingIPPool) error {
	log := ctrl.LoggerFrom(ctx)
	ipAddresses := &ipamv1.IPAddressList{}
	if err := r.Client.List(ctx, ipAddresses, client.InNamespace(pool.Namespace), client.MatchingFields{infrav1alpha1.OpenStackFloatingIPPoolNameIndex: pool.Name}); err != nil {
		return err
	}

	// If there are still IPAddress objects that are not deleted, there are still claims on this pool and we should not delete the
	// pool because it is needed to clean up the addresses from openstack
	if len(ipAddresses.Items) > 0 {
		log.Info("Waiting for IPAddress to be deleted before deleting OpenStackFloatingIPPool")
		return errors.New("waiting for IPAddress to be deleted, until we can delete the OpenStackFloatingIPPool")
	}

	networkingService, err := networking.NewService(scope)
	if err != nil {
		return err
	}

	for _, ip := range diff(pool.Status.AvailableIPs, pool.Spec.PreAllocatedFloatingIPs) {
		if err := networkingService.DeleteFloatingIP(pool, ip); err != nil {
			return fmt.Errorf("delete floating IP: %w", err)
		}
		// Remove the IP from the available IPs, so we don't try to delete it again if the reconcile loop runs again
		pool.Status.AvailableIPs = diff(pool.Status.AvailableIPs, []string{ip})
	}

	if controllerutil.RemoveFinalizer(pool, infrav1alpha1.OpenStackFloatingIPPoolFinalizer) {
		log.Info("Removing finalizer from OpenStackFloatingIPPool")
		return r.Client.Update(ctx, pool)
	}
	return nil
}

func union(a []string, b []string) []string {
	m := make(map[string]struct{})
	for _, item := range a {
		m[item] = struct{}{}
	}
	for _, item := range b {
		m[item] = struct{}{}
	}
	result := make([]string, 0, len(m))
	for item := range m {
		result = append(result, item)
	}
	return result
}

func diff(a []string, b []string) []string {
	m := make(map[string]struct{})
	for _, item := range a {
		m[item] = struct{}{}
	}
	for _, item := range b {
		delete(m, item)
	}
	result := make([]string, 0, len(m))
	for item := range m {
		result = append(result, item)
	}
	return result
}

func (r *OpenStackFloatingIPPoolReconciler) reconcileIPAddresses(ctx context.Context, scope *scope.WithLogger, pool *infrav1alpha1.OpenStackFloatingIPPool) error {
	ipAddresses := &ipamv1.IPAddressList{}
	if err := r.Client.List(ctx, ipAddresses, client.InNamespace(pool.Namespace), client.MatchingFields{infrav1alpha1.OpenStackFloatingIPPoolNameIndex: pool.Name}); err != nil {
		return err
	}

	networkingService, err := networking.NewService(scope)
	if err != nil {
		return err
	}
	pool.Status.ClaimedIPs = []string{}
	if pool.Status.AvailableIPs == nil {
		pool.Status.AvailableIPs = []string{}
	}

	for i := 0; i < len(ipAddresses.Items); i++ {
		ipAddress := &(ipAddresses.Items[i])
		if ipAddress.DeletionTimestamp.IsZero() {
			pool.Status.ClaimedIPs = append(pool.Status.ClaimedIPs, ipAddress.Spec.Address)
			continue
		}

		// Check if the owning claim or its cluster is paused before processing deletion,
		// and clear the claim's AddressRef so it can be re-reconciled once unpaused or re-created.
		if ipAddress.Spec.ClaimRef.Name == "" {
			continue
		}

		claim := &ipamv1.IPAddressClaim{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: ipAddress.Spec.ClaimRef.Name, Namespace: ipAddress.Namespace}, claim); err != nil {
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to get IPAddressClaim %q: %w", ipAddress.Spec.ClaimRef.Name, err)
			}
			claim = nil
		} else {
			cluster, err := util.GetClusterFromMetadata(ctx, r.Client, claim.ObjectMeta)
			if err != nil {
				return fmt.Errorf("failed to get owning cluster for claim %q: %w", claim.Name, err)
			}
			if annotations.IsPaused(cluster, claim) {
				scope.Logger().V(4).Info("IPAddress owner IPAddressClaim or linked Cluster is paused, skipping deletion", "ipAddress", ipAddress.Name, "claim", claim.Name)
				continue
			}
		}

		if controllerutil.ContainsFinalizer(ipAddress, infrav1alpha1.DeleteFloatingIPFinalizer) {
			if pool.Spec.ReclaimPolicy == infrav1alpha1.ReclaimDelete && !contains(pool.Spec.PreAllocatedFloatingIPs, ipAddress.Spec.Address) {
				if err = networkingService.DeleteFloatingIP(pool, ipAddress.Spec.Address); err != nil {
					return fmt.Errorf("delete floating IP %q: %w", ipAddress.Spec.Address, err)
				}
			} else {
				pool.Status.AvailableIPs = append(pool.Status.AvailableIPs, ipAddress.Spec.Address)
			}
		}
		// Clear AddressRef so the claim will be re-assigned an IP on the next reconcile.
		if claim != nil && claim.Status.AddressRef.Name != "" {
			claim.Status.AddressRef.Name = ""
			if err := r.Client.Status().Update(ctx, claim); err != nil {
				return fmt.Errorf("failed to clear AddressRef for claim %q: %w", claim.Name, err)
			}
		}

		controllerutil.RemoveFinalizer(ipAddress, infrav1alpha1.DeleteFloatingIPFinalizer)
		if err := r.Client.Update(ctx, ipAddress); err != nil {
			return err
		}
	}
	allIPs := union(pool.Status.AvailableIPs, pool.Spec.PreAllocatedFloatingIPs)
	unclaimedIPs := diff(allIPs, pool.Status.ClaimedIPs)
	pool.Status.AvailableIPs = diff(unclaimedIPs, pool.Status.FailedIPs)
	return nil
}

func (r *OpenStackFloatingIPPoolReconciler) getIP(ctx context.Context, scope *scope.WithLogger, pool *infrav1alpha1.OpenStackFloatingIPPool) (string, error) {
	// There's a potential leak of IPs here, if the reconcile loop fails after we claim an IP but before we create the IPAddress object.
	var ip string

	networkingService, err := networking.NewService(scope)
	if err != nil {
		scope.Logger().Error(err, "Failed to create networking service")
		return "", err
	}

	// Get tagged floating IPs and add them to the available IPs if they are not present in either the available IPs or the claimed IPs
	// This is done to prevent leaking floating IPs if the floating IP was created but the IPAddress object was not
	if len(pool.Status.AvailableIPs) == 0 {
		taggedIPs, err := networkingService.GetFloatingIPsByTag(pool.GetFloatingIPTag())
		if err != nil {
			scope.Logger().Error(err, "Failed to get floating IPs by tag", "pool", pool.Name)
			return "", err
		}
		for _, taggedIP := range taggedIPs {
			if contains(pool.Status.AvailableIPs, taggedIP.FloatingIP) || contains(pool.Status.ClaimedIPs, taggedIP.FloatingIP) {
				continue
			}
			scope.Logger().Info("Tagged floating IP found that was not known to the pool, adding it to the pool", "ip", taggedIP.FloatingIP)
			pool.Status.AvailableIPs = append(pool.Status.AvailableIPs, taggedIP.FloatingIP)
		}
	}

	if len(pool.Status.AvailableIPs) > 0 {
		ip = pool.Status.AvailableIPs[0]
		pool.Status.AvailableIPs = pool.Status.AvailableIPs[1:]
		pool.Status.ClaimedIPs = append(pool.Status.ClaimedIPs, ip)
	}

	if ip != "" {
		fp, err := networkingService.GetFloatingIP(ip)
		if err != nil {
			return "", fmt.Errorf("get floating IP: %w", err)
		}
		if fp != nil {
			pool.Status.ClaimedIPs = append(pool.Status.ClaimedIPs, fp.FloatingIP)
			return fp.FloatingIP, nil
		}
		pool.Status.FailedIPs = append(pool.Status.FailedIPs, ip)
	}
	maxIPs := ptr.Deref(pool.Spec.MaxIPs, -1)
	// If we have reached the maximum number of IPs, we should not create more IPs
	if maxIPs != -1 && len(pool.Status.ClaimedIPs) >= maxIPs {
		scope.Logger().Info("MaxIPs reached", "pool", pool.Name)
		v1beta1conditions.MarkFalse(pool, infrav1alpha1.OpenstackFloatingIPPoolReadyCondition, infrav1alpha1.MaxIPsReachedReason, clusterv1beta1.ConditionSeverityError, "Maximum number of IPs reached, we will not allocate more IPs for this pool")
		return "", errMaxIPsReached
	}

	fp, err := networkingService.CreateFloatingIPForPool(pool)
	if err != nil {
		scope.Logger().Error(err, "Failed to create floating IP", "pool", pool.Name)
		v1beta1conditions.MarkFalse(pool, infrav1alpha1.OpenstackFloatingIPPoolReadyCondition, infrav1.OpenStackErrorReason, clusterv1beta1.ConditionSeverityError, "Failed to create floating IP: %v", err)
		return "", err
	}
	defer func() {
		tag := pool.GetFloatingIPTag()

		err := wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
			if err := networkingService.TagFloatingIP(fp.FloatingIP, tag); err != nil {
				scope.Logger().Error(err, "Failed to tag floating IP, retrying", "ip", fp.FloatingIP, "tag", tag)
				return false, err
			}
			return true, nil
		})
		if err != nil {
			scope.Logger().Error(err, "Failed to tag floating IP", "ip", fp.FloatingIP, "tag", tag)
		}
	}()

	v1beta1conditions.MarkTrue(pool, infrav1alpha1.OpenstackFloatingIPPoolReadyCondition)
	ip = fp.FloatingIP
	pool.Status.ClaimedIPs = append(pool.Status.ClaimedIPs, ip)
	return ip, nil
}

func (r *OpenStackFloatingIPPoolReconciler) reconcileFloatingIPNetwork(scope *scope.WithLogger, pool *infrav1alpha1.OpenStackFloatingIPPool) error {
	// If the pool already has a network, we don't need to do anything
	if pool.Status.FloatingIPNetwork != nil {
		return nil
	}

	networkingService, err := networking.NewService(scope)
	if err != nil {
		return err
	}

	// If the pool does not have a network, we default to a external network if there's only one
	var networkParam *infrav1.NetworkParam
	if pool.Spec.FloatingIPNetwork == nil {
		networkParam = &infrav1.NetworkParam{
			Filter: &infrav1.NetworkFilter{},
		}
	} else {
		networkParam = pool.Spec.FloatingIPNetwork
	}

	network, err := networkingService.GetNetworkByParam(networkParam, networking.ExternalNetworksOnly)
	if err != nil {
		v1beta1conditions.MarkFalse(pool, infrav1alpha1.OpenstackFloatingIPPoolReadyCondition, infrav1alpha1.UnableToFindNetwork, clusterv1beta1.ConditionSeverityError, "Failed to find network: %v", err)
		return fmt.Errorf("failed to find network: %w", err)
	}

	pool.Status.FloatingIPNetwork = &infrav1.NetworkStatus{
		ID:   network.ID,
		Name: network.Name,
		Tags: network.Tags,
	}
	return nil
}

func (r *OpenStackFloatingIPPoolReconciler) ipAddressClaimToPoolMapper(ctx context.Context, o client.Object) []ctrl.Request {
	log := ctrl.LoggerFrom(ctx)
	claim, ok := o.(*ipamv1.IPAddressClaim)
	if !ok {
		panic(fmt.Sprintf("Expected a IPAddressClaim but got a %T", o))
	}
	if claim.Spec.PoolRef.Kind != openStackFloatingIPPool {
		return nil
	}

	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, claim.ObjectMeta)
	if err != nil {
		log.Error(err, "Failed to get owning cluster, skipping mapping", "claim", claim.Name, "namespace", claim.Namespace)
		return nil
	}

	if annotations.IsPaused(cluster, claim) {
		log.V(4).Info("IPAddressClaim or linked Cluster is paused, skipping mapping", "claim", claim.Name, "namespace", claim.Namespace)
		return nil
	}

	return []ctrl.Request{
		{
			NamespacedName: client.ObjectKey{
				Name:      claim.Spec.PoolRef.Name,
				Namespace: claim.Namespace,
			},
		},
	}
}

func (r *OpenStackFloatingIPPoolReconciler) clusterToPoolMapper(ctx context.Context, o client.Object) []ctrl.Request {
	log := ctrl.LoggerFrom(ctx)
	cluster, ok := o.(*clusterv1.Cluster)
	if !ok {
		panic(fmt.Sprintf("Expected a Cluster but got a %T", o))
	}

	claims := &ipamv1.IPAddressClaimList{}
	if err := r.Client.List(ctx, claims, client.InNamespace(cluster.Namespace), client.MatchingLabels{clusterv1.ClusterNameLabel: cluster.Name}); err != nil {
		log.Error(err, "Failed to list IPAddressClaims for cluster, skipping mapping", "cluster", cluster.Name, "namespace", cluster.Namespace)
		return nil
	}

	requestsByPool := make(map[client.ObjectKey]struct{})
	for i := range claims.Items {
		claim := &claims.Items[i]
		if claim.Spec.PoolRef.Kind != openStackFloatingIPPool {
			continue
		}
		if annotations.IsPaused(cluster, claim) {
			continue
		}
		requestsByPool[client.ObjectKey{Name: claim.Spec.PoolRef.Name, Namespace: claim.Namespace}] = struct{}{}
	}

	requests := make([]ctrl.Request, 0, len(requestsByPool))
	for key := range requestsByPool {
		requests = append(requests, ctrl.Request{NamespacedName: key})
	}

	return requests
}

func (r *OpenStackFloatingIPPoolReconciler) ipAddressToPoolMapper(_ context.Context, o client.Object) []ctrl.Request {
	ip, ok := o.(*ipamv1.IPAddress)
	if !ok {
		panic(fmt.Sprintf("Expected a IPAddress but got a %T", o))
	}
	if ip.Spec.PoolRef.Kind != openStackFloatingIPPool {
		return nil
	}
	return []ctrl.Request{
		{
			NamespacedName: client.ObjectKey{
				Name:      ip.Spec.PoolRef.Name,
				Namespace: ip.Namespace,
			},
		},
	}
}

func (r *OpenStackFloatingIPPoolReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &ipamv1.IPAddressClaim{}, infrav1alpha1.OpenStackFloatingIPPoolNameIndex, func(rawObj client.Object) []string {
		claim := rawObj.(*ipamv1.IPAddressClaim)
		if claim.Spec.PoolRef.Kind != openStackFloatingIPPool {
			return nil
		}
		return []string{claim.Spec.PoolRef.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &ipamv1.IPAddress{}, infrav1alpha1.OpenStackFloatingIPPoolNameIndex, func(rawObj client.Object) []string {
		ip := rawObj.(*ipamv1.IPAddress)
		if ip.Spec.PoolRef.Kind != openStackFloatingIPPool {
			return nil
		}
		return []string{ip.Spec.PoolRef.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.OpenStackFloatingIPPool{}).
		Watches(
			&ipamv1.IPAddressClaim{},
			handler.EnqueueRequestsFromMapFunc(r.ipAddressClaimToPoolMapper),
		).
		Watches(
			&clusterv1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(r.clusterToPoolMapper),
			builder.WithPredicates(predicates.ClusterUnpaused(mgr.GetScheme(), ctrl.LoggerFrom(ctx))),
		).
		Watches(
			&ipamv1.IPAddress{},
			handler.EnqueueRequestsFromMapFunc(r.ipAddressToPoolMapper),
		).
		Complete(r)
}
