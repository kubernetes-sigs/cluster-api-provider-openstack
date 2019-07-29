package cluster

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"reflect"
	providerv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/certificates"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/networking"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/provider"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/deployer"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	clientclusterv1 "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/controller/remote"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/patch"
)

// Actuator controls cluster related infrastructure.
type Actuator struct {
	*deployer.Deployer

	params ActuatorParams
}

// ActuatorParams holds parameter information for Actuator
type ActuatorParams struct {
	KubeClient    kubernetes.Interface
	Client        client.Client
	ClusterClient clientclusterv1.ClusterV1alpha1Interface
	EventRecorder record.EventRecorder
	Scheme        *runtime.Scheme
}

// NewActuator creates a new Actuator
func NewActuator(params ActuatorParams) *Actuator {
	return &Actuator{
		Deployer: deployer.New(),
		params:   params,
	}
}

// Reconcile creates or applies updates to the cluster.
func (a *Actuator) Reconcile(cluster *clusterv1.Cluster) error {

	if cluster == nil {
		return fmt.Errorf("the cluster is nil, check your cluster configuration")
	}
	klog.Infof("Reconciling Cluster %s/%s", cluster.Namespace, cluster.Name)

	// ClusterCopy is used for patch generation during storeCluster
	clusterCopy := cluster.DeepCopy()
	clusterName := fmt.Sprintf("%s-%s", cluster.Namespace, cluster.Name)

	osProviderClient, clientOpts, err := provider.NewClientFromCluster(a.params.KubeClient, cluster)
	if err != nil {
		return err
	}

	networkingService, err := networking.NewService(osProviderClient, clientOpts)
	if err != nil {
		return err
	}

	certificatesService := certificates.NewService()

	// Load provider spec & status.
	clusterProviderSpec, clusterProviderStatus, err := providerv1.ClusterSpecAndStatusFromProviderSpec(cluster)
	if err != nil {
		return err
	}

	defer func() {
		if err := a.storeCluster(cluster, clusterCopy, clusterProviderSpec, clusterProviderStatus); err != nil {
			klog.Errorf("failed to store cluster %q in namespace %q: %v", cluster.Name, cluster.Namespace, err)
		}
	}()

	klog.Infof("Reconciling certificates for cluster %s", clusterName)
	// Store cert material in spec.
	if err := certificatesService.ReconcileCertificates(clusterName, clusterProviderSpec); err != nil {
		return errors.Wrapf(err, "failed to reconcile certificates for cluster %q", cluster.Name)
	}

	klog.Infof("Reconciling network components for cluster %s", clusterName)
	if clusterProviderSpec.NodeCIDR == "" {
		klog.V(4).Infof("No need to reconcile network for cluster %s", clusterName)
	} else {
		err := networkingService.ReconcileNetwork(clusterName, clusterProviderSpec, clusterProviderStatus)
		if err != nil {
			return errors.Errorf("failed to reconcile network: %v", err)
		}
		err = networkingService.ReconcileSubnet(clusterName, clusterProviderSpec, clusterProviderStatus)
		if err != nil {
			return errors.Errorf("failed to reconcile subnets: %v", err)
		}
		err = networkingService.ReconcileRouter(clusterName, clusterProviderSpec, clusterProviderStatus)
		if err != nil {
			return errors.Errorf("failed to reconcile router: %v", err)
		}
	}

	err = networkingService.ReconcileSecurityGroups(clusterName, *clusterProviderSpec, clusterProviderStatus)
	if err != nil {
		return errors.Errorf("failed to reconcile security groups: %v", err)
	}

	// Store KubeConfig for Cluster API NodeRef controller to use.
	kubeConfigSecretName := remote.KubeConfigSecretName(cluster.Name)
	if _, err := a.params.KubeClient.CoreV1().Secrets(cluster.Namespace).Get(kubeConfigSecretName, metav1.GetOptions{}); err != nil && apierrors.IsNotFound(err) {
		kubeConfig, err := a.Deployer.GetKubeConfig(cluster, nil)
		if err != nil {
			return errors.Wrapf(err, "failed to get kubeconfig for cluster %q", cluster.Name)
		}

		kubeConfigSecret := &apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubeConfigSecretName,
			},
			StringData: map[string]string{
				"value": kubeConfig,
			},
		}

		if _, err := a.params.KubeClient.CoreV1().Secrets(cluster.Namespace).Create(kubeConfigSecret); err != nil {
			return errors.Wrapf(err, "failed to create kubeconfig secret for cluster %q", cluster.Name)
		}
	} else if err != nil {
		return errors.Wrapf(err, "failed to get kubeconfig secret for cluster %q", cluster.Name)
	}

	return nil
}

// Delete deletes a cluster and is invoked by the Cluster Controller
func (a *Actuator) Delete(cluster *clusterv1.Cluster) error {
	klog.Infof("Deleting Cluster %s/%s", cluster.Namespace, cluster.Name)

	osProviderClient, clientOpts, err := provider.NewClientFromCluster(a.params.KubeClient, cluster)
	if err != nil {
		return err
	}

	networkingService, err := networking.NewService(osProviderClient, clientOpts)
	if err != nil {
		return err
	}

	// Load provider spec & status.
	_, clusterProviderStatus, err := providerv1.ClusterSpecAndStatusFromProviderSpec(cluster)
	if err != nil {
		return err
	}

	// Delete other things
	if clusterProviderStatus.GlobalSecurityGroup != nil {
		klog.Infof("Deleting global security group %q", clusterProviderStatus.GlobalSecurityGroup.Name)
		err := networkingService.DeleteSecurityGroups(clusterProviderStatus.GlobalSecurityGroup)
		if err != nil {
			return errors.Errorf("failed to delete security group: %v", err)
		}
	}

	if clusterProviderStatus.ControlPlaneSecurityGroup != nil {
		klog.Infof("Deleting control plane security group %q", clusterProviderStatus.ControlPlaneSecurityGroup.Name)
		err := networkingService.DeleteSecurityGroups(clusterProviderStatus.ControlPlaneSecurityGroup)
		if err != nil {
			return errors.Errorf("failed to delete security group: %v", err)
		}
	}
	return nil
}

func (a *Actuator) storeCluster(cluster *clusterv1.Cluster, clusterCopy *clusterv1.Cluster, spec *providerv1.OpenstackClusterProviderSpec, status *providerv1.OpenstackClusterProviderStatus) error {

	rawSpec, rawStatus, err := providerv1.EncodeClusterSpecAndStatus(cluster, spec, status)
	if err != nil {
		return err
	}

	cluster.Spec.ProviderSpec.Value = rawSpec

	// Build a patch and marshal that patch to something the client will understand.
	p, err := patch.NewJSONPatch(clusterCopy, cluster)
	if err != nil {
		return fmt.Errorf("failed to create new JSONPatch: %v", err)
	}

	clusterClient := a.params.ClusterClient.Clusters(cluster.Namespace)

	// Do not update Cluster if nothing has changed
	if len(p) != 0 {
		pb, err := json.MarshalIndent(p, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to json marshal patch: %v", err)
		}
		klog.Infof("Patching cluster %s", cluster.Name)
		result, err := clusterClient.Patch(cluster.Name, types.JSONPatchType, pb)
		if err != nil {
			return fmt.Errorf("failed to patch cluster: %v", err)
		}
		// Keep the resource version updated so the status update can succeed
		cluster.ResourceVersion = result.ResourceVersion
	}

	cluster.Status.ProviderStatus = rawStatus
	if !reflect.DeepEqual(cluster.Status, clusterCopy.Status) {
		klog.Infof("Updating cluster status %s", cluster.Name)
		if _, err := clusterClient.UpdateStatus(cluster); err != nil {
			return fmt.Errorf("failed to update cluster status: %v", err)
		}
	}
	return nil
}
