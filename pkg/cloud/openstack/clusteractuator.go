package openstack

import (
	"github.com/golang/glog"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type OpenstackClusterClient struct{}

func NewClusterActuator() (*OpenstackClusterClient, error) {
	return &OpenstackClusterClient{}, nil
}

func (occ *OpenstackClusterClient) Reconcile(cluster *clusterv1.Cluster) error {
	glog.Errorf("Not implemented yet")
	return nil
}

func (occ *OpenstackClusterClient) Delete(cluster *clusterv1.Cluster) error {
	glog.Errorf("Not implemented yet")
	return nil
}
