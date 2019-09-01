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

package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ClusterFinalizer allows ReconcileOpenStackCluster to clean up OpenStack resources associated with OpenStackCluster before
	// removing it from the apiserver.
	ClusterFinalizer = "openstackcluster.infrastructure.cluster.x-k8s.io"
)

// OpenStackClusterSpec defines the desired state of OpenStackCluster
type OpenStackClusterSpec struct {

	// The name of the secret containing the openstack credentials
	// +optional
	CloudsSecret *corev1.SecretReference `json:"cloudsSecret"`

	// The name of the cloud to use from the clouds secret
	// +optional
	CloudName string `json:"cloudName"`

	// NodeCIDR is the OpenStack Subnet to be created. Cluster actuator will create a
	// network, a subnet with NodeCIDR, and a router connected to this subnet.
	// If you leave this empty, no network will be created.
	NodeCIDR string `json:"nodeCidr,omitempty"`
	// DNSNameservers is the list of nameservers for OpenStack Subnet being created.
	DNSNameservers []string `json:"dnsNameservers,omitempty"`
	// ExternalRouterIPs is an array of externalIPs on the respective subnets.
	// This is necessary if the router needs a fixed ip in a specific subnet.
	ExternalRouterIPs []ExternalRouterIPParam `json:"externalRouterIPs,omitempty"`
	// ExternalNetworkID is the ID of an external OpenStack Network. This is necessary
	// to get public internet to the VMs.
	ExternalNetworkID string `json:"externalNetworkId,omitempty"`

	// UseOctavia is weather LoadBalancer Service is Octavia or not
	// +optional
	UseOctavia bool `json:"useOctavia,omitempty"`

	// ManagedAPIServerLoadBalancer defines whether a LoadBalancer for the
	// APIServer should be created. If set to true the following properties are
	// mandatory: APIServerLoadBalancerFloatingIP, APIServerLoadBalancerPort
	// +optional
	ManagedAPIServerLoadBalancer bool `json:"managedAPIServerLoadBalancer"`

	// APIServerLoadBalancerFloatingIP is the floatingIP which will be associated
	// to the APIServer loadbalancer. The floatingIP will be created if it not
	// already exists.
	APIServerLoadBalancerFloatingIP string `json:"apiServerLoadBalancerFloatingIP,omitempty"`

	// APIServerLoadBalancerPort is the port on which the listener on the APIServer
	// loadbalancer will be created
	APIServerLoadBalancerPort int `json:"apiServerLoadBalancerPort,omitempty"`

	// APIServerLoadBalancerAdditionalPorts adds additional ports to the APIServerLoadBalancer
	APIServerLoadBalancerAdditionalPorts []int `json:"apiServerLoadBalancerAdditionalPorts,omitempty"`

	// ManagedSecurityGroups defines that kubernetes manages the OpenStack security groups
	// for now, that means that we'll create two security groups, one allowing SSH
	// and API access from everywhere, and another one that allows all traffic to/from
	// machines belonging to that group. In the future, we could make this more flexible.
	// +optional
	ManagedSecurityGroups bool `json:"managedSecurityGroups"`

	// DisablePortSecurity disables the port security of the network created for the
	// Kubernetes cluster, which also disables SecurityGroups
	DisablePortSecurity bool `json:"disablePortSecurity,omitempty"`

	// Tags for all resources in cluster
	Tags []string `json:"tags,omitempty"`

	// Default: True. In case of server tag errors, set to False
	DisableServerTags bool `json:"disableServerTags,omitempty"`

	// CAKeyPair is the key pair for ca certs.
	CAKeyPair KeyPair `json:"caKeyPair,omitempty"`

	//EtcdCAKeyPair is the key pair for etcd.
	EtcdCAKeyPair KeyPair `json:"etcdCAKeyPair,omitempty"`

	// FrontProxyCAKeyPair is the key pair for FrontProxyKeyPair.
	FrontProxyCAKeyPair KeyPair `json:"frontProxyCAKeyPair,omitempty"`

	// SAKeyPair is the service account key pair.
	SAKeyPair KeyPair `json:"saKeyPair,omitempty"`
}

// OpenStackClusterStatus defines the observed state of OpenStackCluster
type OpenStackClusterStatus struct {
	Ready bool `json:"ready"`
	// APIEndpoints represents the endpoints to communicate with the control plane.
	// +optional
	APIEndpoints []APIEndpoint `json:"apiEndpoints,omitempty"`

	// Network contains all information about the created OpenStack Network.
	// It includes Subnets and Router.
	Network *Network `json:"network,omitempty"`

	// ControlPlaneSecurityGroups contains all the information about the OpenStack
	// Security Group that needs to be applied to control plane nodes.
	// TODO: Maybe instead of two properties, we add a property to the group?
	ControlPlaneSecurityGroup *SecurityGroup `json:"controlPlaneSecurityGroup,omitempty"`

	// GlobalSecurityGroup contains all the information about the OpenStack Security
	// Group that needs to be applied to all nodes, both control plane and worker nodes.
	GlobalSecurityGroup *SecurityGroup `json:"globalSecurityGroup,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=openstackclusters,scope=Namespaced
// +kubebuilder:storageversion
// +kubebuilder:subresource:status

// OpenStackCluster is the Schema for the openstackclusters API
type OpenStackCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackClusterSpec   `json:"spec,omitempty"`
	Status OpenStackClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OpenStackClusterList contains a list of OpenStackCluster
type OpenStackClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackCluster{}, &OpenStackClusterList{})
}
