package cluster

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	providerv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	client "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
)

type Actuator struct {
	clustersGetter client.ClustersGetter
}

// ActuatorParams holds parameter information for Actuator
type ActuatorParams struct {
	ClustersGetter client.ClustersGetter
}

// NewActuator creates a new Actuator
func NewActuator(params ActuatorParams) (*Actuator, error) {
	res := &Actuator{
		clustersGetter: params.ClustersGetter,
	}

	return res, nil
}

func (a *Actuator) Reconcile(cluster *clusterv1.Cluster) error {
	glog.Infof("Reconciling cluster %v.", cluster.Name)

	// Load provider config.
	_, err := providerv1.ClusterConfigFromProviderConfig(cluster.Spec.ProviderConfig)
	if err != nil {
		return errors.Errorf("failed to load cluster provider config: %v", err)
	}

	// Load provider status.
	_, err = providerv1.ClusterStatusFromProviderStatus(cluster.Status.ProviderStatus)
	if err != nil {
		return errors.Errorf("failed to load cluster provider status: %v", err)
	}

	/* Uncomment when the clusterGetter is back to working
	defer func() {
		if err := a.storeClusterStatus(cluster, status); err != nil {
			glog.Errorf("failed to store provider status for cluster %q in namespace %q: %v", cluster.Name, cluster.Namespace, err)
		}
	}()*/
	return nil
}

// Delete deletes a cluster and is invoked by the Cluster Controller
func (a *Actuator) Delete(cluster *clusterv1.Cluster) error {
	glog.Infof("Deleting cluster %v.", cluster.Name)

	// Load provider config.
	_, err := providerv1.ClusterConfigFromProviderConfig(cluster.Spec.ProviderConfig)
	if err != nil {
		return errors.Errorf("failed to load cluster provider config: %v", err)
	}

	// Load provider status.
	_, err = providerv1.ClusterStatusFromProviderStatus(cluster.Status.ProviderStatus)
	if err != nil {
		return errors.Errorf("failed to load cluster provider status: %v", err)
	}

	// Delete other things

	return nil
}

func (a *Actuator) storeClusterStatus(cluster *clusterv1.Cluster, status *providerv1.OpenstackClusterProviderStatus) error {
	clusterClient := a.clustersGetter.Clusters(cluster.Namespace)

	ext, err := providerv1.EncodeClusterStatus(status)
	if err != nil {
		return fmt.Errorf("failed to update cluster status for cluster %q in namespace %q: %v", cluster.Name, cluster.Namespace, err)
	}

	cluster.Status.ProviderStatus = ext

	if _, err := clusterClient.UpdateStatus(cluster); err != nil {
		return fmt.Errorf("failed to update cluster status for cluster %q in namespace %q: %v", cluster.Name, cluster.Namespace, err)
	}

	return nil
}
