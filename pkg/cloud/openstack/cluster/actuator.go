package cluster

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/pkg/errors"
	"k8s.io/klog"
	providerv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	providerv1openstack "sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/clients"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// Actuator controls cluster related infrastructure.
type Actuator struct {
	params providerv1openstack.ActuatorParams
}

// NewActuator creates a new Actuator
func NewActuator(params providerv1openstack.ActuatorParams) (*Actuator, error) {
	res := &Actuator{params: params}
	return res, nil
}

// Reconcile creates or applies updates to the cluster.
func (a *Actuator) Reconcile(cluster *clusterv1.Cluster) error {
	klog.Infof("Reconciling cluster %v.", cluster.Name)
	clusterName := fmt.Sprintf("%s/%s", cluster.ObjectMeta.Name, cluster.Name)

	client, err := a.getNetworkClient(cluster)
	if err != nil {
		return err
	}
	networkService, err := clients.NewNetworkService(client)
	if err != nil {
		return err
	}

	secGroupService, err := clients.NewSecGroupService(client)
	if err != nil {
		return err
	}

	// Load provider config.
	desired, err := providerv1.ClusterSpecFromProviderSpec(cluster.Spec.ProviderSpec)
	if err != nil {
		return errors.Errorf("failed to load cluster provider spec: %v", err)
	}

	// Load provider status.
	status, err := providerv1.ClusterStatusFromProviderStatus(cluster.Status.ProviderStatus)
	if err != nil {
		return errors.Errorf("failed to load cluster provider status: %v", err)
	}

	err = networkService.Reconcile(clusterName, *desired, status)
	if err != nil {
		return errors.Errorf("failed to reconcile network: %v", err)
	}

	err = secGroupService.Reconcile(clusterName, *desired, status)
	if err != nil {
		return errors.Errorf("failed to reconcile security groups: %v", err)
	}
	defer func() {
		if err := a.storeClusterStatus(cluster, status); err != nil {
			klog.Errorf("failed to store provider status for cluster %q in namespace %q: %v", cluster.Name, cluster.Namespace, err)
		}
	}()
	return nil
}

// Delete deletes a cluster and is invoked by the Cluster Controller
func (a *Actuator) Delete(cluster *clusterv1.Cluster) error {
	klog.Infof("Deleting cluster %v.", cluster.Name)

	client, err := a.getNetworkClient(cluster)
	if err != nil {
		return err
	}
	_, err = clients.NewNetworkService(client)
	if err != nil {
		return err
	}

	// Load provider config.
	_, err = providerv1.ClusterSpecFromProviderSpec(cluster.Spec.ProviderSpec)
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
	ext, err := providerv1.EncodeClusterStatus(status)
	if err != nil {
		return fmt.Errorf("failed to update cluster status for cluster %q in namespace %q: %v", cluster.Name, cluster.Namespace, err)
	}
	cluster.Status.ProviderStatus = ext

	statusClient := a.params.Client.Status()
	if err := statusClient.Update(context.Background(), cluster); err != nil {
		return fmt.Errorf("failed to update cluster status for cluster %q in namespace %q: %v", cluster.Name, cluster.Namespace, err)
	}

	return nil
}

// getNetworkClient returns an gophercloud.ServiceClient provided by openstack.NewNetworkV2
// TODO(chrigl) currently ignoring cluster, but in the future we might store OS-Credentials
// as secrets referenced by the cluster.
// See https://github.com/kubernetes-sigs/cluster-api-provider-openstack/pull/136
func (a *Actuator) getNetworkClient(cluster *clusterv1.Cluster) (*gophercloud.ServiceClient, error) {
	clientOpts := new(clientconfig.ClientOpts)
	opts, err := clientconfig.AuthOptions(clientOpts)
	if err != nil {
		return nil, err
	}

	provider, err := openstack.AuthenticatedClient(*opts)
	if err != nil {
		return nil, fmt.Errorf("create providerClient err: %v", err)
	}

	client, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}
