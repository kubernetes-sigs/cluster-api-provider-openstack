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
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha2"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/certificates"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/loadbalancer"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/provider"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/userdata"
	"sigs.k8s.io/cluster-api/api/v1alpha2"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strconv"
	"strings"
	"time"
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
		logger.Info("Waiting for Cluster Controller to set OwnerRef on OpenStackCluster")
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
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

	osProviderClient, clientOpts, err := provider.NewClientFromCluster(r.Client, openStackCluster)
	if err != nil {
		return reconcile.Result{}, err
	}

	networkingService, err := networking.NewService(osProviderClient, clientOpts)
	if err != nil {
		return reconcile.Result{}, err
	}

	certificatesService := certificates.NewService()

	loadbalancerService, err := loadbalancer.NewService(osProviderClient, clientOpts, openStackCluster.Spec.UseOctavia)
	if err != nil {
		return reconcile.Result{}, err
	}

	klog.Infof("Reconciling certificates for cluster %s", clusterName)
	// Store cert material in spec.
	if err := certificatesService.ReconcileCertificates(clusterName, openStackCluster); err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "failed to reconcile certificates for cluster %q", cluster.Name)
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
	controlPlaneURI := strings.Split(openStackCluster.Spec.ClusterConfiguration.ControlPlaneEndpoint, ":")
	apiServerHost := controlPlaneURI[0]
	apiServerPortStr := controlPlaneURI[1]
	apiServerPort, err := strconv.Atoi(apiServerPortStr)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "could not parse port of controlPlaneEndpoint %s", openStackCluster.Spec.ClusterConfiguration.ControlPlaneEndpoint)
	}
	openStackCluster.Status.APIEndpoints = []infrav1.APIEndpoint{
		{
			Host: apiServerHost,
			Port: apiServerPort,
		},
	}

	// No errors, so mark us ready so the Cluster API Cluster Controller can pull it
	openStackCluster.Status.Ready = true

	// TODO remove after migration to kubeadm bootstrapper
	// Upload kubeconfig (just for us) can be deleted after we migrated to kubeadm bootstrapper
	// because kubeadm bootstrapper already creates the secret "{cluster.Name}-kubeconfig"
	kubeConfig, err := userdata.GetKubeConfig(openStackCluster)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "failed to get kubeconfig for cluster %q", cluster.Name)
	}
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-tmp-kubeconfig", cluster.Name),
			Namespace: cluster.Namespace,
		},
		StringData: map[string]string{
			"value": kubeConfig,
		},
	}
	err = r.Client.Create(context.TODO(), secret)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			if err = r.Client.Update(context.TODO(), secret); err != nil {
				return reconcile.Result{}, errors.Wrapf(err, "failed to update kubeconfig secret for cluster %q", cluster.Name)
			}
		} else {
			return reconcile.Result{}, errors.Wrapf(err, "failed to create kubeconfig secret for cluster %q", cluster.Name)
		}
	}

	klog.Infof("Reconciled Cluster %s/%s successfully", cluster.Namespace, cluster.Name)
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
	return reconcile.Result{}, nil
}

func (r *OpenStackClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.OpenStackCluster{}).
		Complete(r)
}
