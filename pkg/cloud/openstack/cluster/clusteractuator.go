package openstack

import (
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
	glog.Errorf("Not implemented yet")
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
