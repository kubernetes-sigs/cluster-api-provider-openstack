/*
Copyright 2022 The Kubernetes Authors.

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

package v1alpha6

// OpenStackMachineTemplateResource describes the data needed to create a OpenStackMachine from a template.
type OpenStackMachineTemplateResource struct {
	// Spec is the specification of the desired behavior of the machine.
	Spec OpenStackMachineSpec `json:"spec"`
}

type ExternalRouterIPParam struct {
	// The FixedIP in the corresponding subnet
	FixedIP string `json:"fixedIP,omitempty"`
	// The subnet in which the FixedIP is used for the Gateway of this router
	Subnet SubnetParam `json:"subnet"`
}

type SecurityGroupParam struct {
	// Security Group UID
	UUID string `json:"uuid,omitempty"`
	// Security Group name
	Name string `json:"name,omitempty"`
	// Filters used to query security groups in openstack
	Filter SecurityGroupFilter `json:"filter,omitempty"`
}

type SecurityGroupFilter struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	TenantID    string `json:"tenantId,omitempty"`
	ProjectID   string `json:"projectId,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Marker      string `json:"marker,omitempty"`
	SortKey     string `json:"sortKey,omitempty"`
	SortDir     string `json:"sortDir,omitempty"`
	Tags        string `json:"tags,omitempty"`
	TagsAny     string `json:"tagsAny,omitempty"`
	NotTags     string `json:"notTags,omitempty"`
	NotTagsAny  string `json:"notTagsAny,omitempty"`
}

type NetworkParam struct {
	// Optional UUID of the network.
	// If specified this will not be validated prior to server creation.
	// Required if `Subnets` specifies a subnet by UUID.
	UUID string `json:"uuid,omitempty"`
	// A fixed IPv4 address for the NIC.
	FixedIP string `json:"fixedIP,omitempty"`
	// Filters for optional network query
	Filter NetworkFilter `json:"filter,omitempty"`
	// Subnet within a network to use
	Subnets []SubnetParam `json:"subnets,omitempty"`
}

type NetworkFilter struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	ProjectID   string `json:"projectId,omitempty"`
	ID          string `json:"id,omitempty"`
	Tags        string `json:"tags,omitempty"`
	TagsAny     string `json:"tagsAny,omitempty"`
	NotTags     string `json:"notTags,omitempty"`
	NotTagsAny  string `json:"notTagsAny,omitempty"`
}

type SubnetParam struct {
	// Optional UUID of the subnet.
	// If specified this will not be validated prior to server creation.
	// If specified, the enclosing `NetworkParam` must also be specified by UUID.
	UUID string `json:"uuid,omitempty"`

	// Filters for optional subnet query
	Filter SubnetFilter `json:"filter,omitempty"`
}

type SubnetFilter struct {
	Name            string `json:"name,omitempty"`
	Description     string `json:"description,omitempty"`
	ProjectID       string `json:"projectId,omitempty"`
	IPVersion       int    `json:"ipVersion,omitempty"`
	GatewayIP       string `json:"gateway_ip,omitempty"`
	CIDR            string `json:"cidr,omitempty"`
	IPv6AddressMode string `json:"ipv6AddressMode,omitempty"`
	IPv6RAMode      string `json:"ipv6RaMode,omitempty"`
	ID              string `json:"id,omitempty"`
	Tags            string `json:"tags,omitempty"`
	TagsAny         string `json:"tagsAny,omitempty"`
	NotTags         string `json:"notTags,omitempty"`
	NotTagsAny      string `json:"notTagsAny,omitempty"`
}

type PortOpts struct {
	// Network is a query for an openstack network that the port will be created or discovered on.
	// This will fail if the query returns more than one network.
	Network *NetworkFilter `json:"network,omitempty"`
	// Used to make the name of the port unique. If unspecified, instead the 0-based index of the port in the list is used.
	NameSuffix   string `json:"nameSuffix,omitempty"`
	Description  string `json:"description,omitempty"`
	AdminStateUp *bool  `json:"adminStateUp,omitempty"`
	MACAddress   string `json:"macAddress,omitempty"`
	// Specify pairs of subnet and/or IP address. These should be subnets of the network with the given NetworkID.
	FixedIPs  []FixedIP `json:"fixedIPs,omitempty"`
	TenantID  string    `json:"tenantId,omitempty"`
	ProjectID string    `json:"projectId,omitempty"`
	// The uuids of the security groups to assign to the instance
	// +listType=set
	SecurityGroups *[]string `json:"securityGroups,omitempty"`
	// The names, uuids, filters or any combination these of the security groups to assign to the instance
	SecurityGroupFilters []SecurityGroupParam `json:"securityGroupFilters,omitempty"`
	AllowedAddressPairs  []AddressPair        `json:"allowedAddressPairs,omitempty"`
	// Enables and disables trunk at port level. If not provided, openStackMachine.Spec.Trunk is inherited.
	Trunk *bool `json:"trunk,omitempty"`

	// The ID of the host where the port is allocated
	HostID string `json:"hostId,omitempty"`

	// The virtual network interface card (vNIC) type that is bound to the neutron port.
	VNICType string `json:"vnicType,omitempty"`

	// A dictionary that enables the application running on the specified
	// host to pass and receive virtual network interface (VIF) port-specific
	// information to the plug-in.
	Profile map[string]string `json:"profile,omitempty"`

	// DisablePortSecurity enables or disables the port security when set.
	// When not set, it takes the value of the corresponding field at the network level.
	DisablePortSecurity *bool `json:"disablePortSecurity,omitempty"`

	// Tags applied to the port (and corresponding trunk, if a trunk is configured.)
	// These tags are applied in addition to the instance's tags, which will also be applied to the port.
	// +listType=set
	Tags []string `json:"tags,omitempty"`
}

type FixedIP struct {
	// Subnet is an openstack subnet query that will return the id of a subnet to create
	// the fixed IP of a port in. This query must not return more than one subnet.
	Subnet    *SubnetFilter `json:"subnet"`
	IPAddress string        `json:"ipAddress,omitempty"`
}

type AddressPair struct {
	IPAddress  string `json:"ipAddress,omitempty"`
	MACAddress string `json:"macAddress,omitempty"`
}

type Instance struct {
	ID             string            `json:"id,omitempty"`
	Name           string            `json:"name,omitempty"`
	Trunk          bool              `json:"trunk,omitempty"`
	FailureDomain  string            `json:"failureDomain,omitempty"`
	SecurityGroups *[]string         `json:"securigyGroups,omitempty"`
	Networks       *[]Network        `json:"networks,omitempty"`
	Subnet         string            `json:"subnet,omitempty"`
	Tags           []string          `json:"tags,omitempty"`
	Image          string            `json:"image,omitempty"`
	ImageUUID      string            `json:"imageUUID,omitempty"`
	Flavor         string            `json:"flavor,omitempty"`
	SSHKeyName     string            `json:"sshKeyName,omitempty"`
	UserData       string            `json:"userData,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	ConfigDrive    *bool             `json:"configDrive,omitempty"`
	RootVolume     *RootVolume       `json:"rootVolume,omitempty"`
	ServerGroupID  string            `json:"serverGroupID,omitempty"`
	State          InstanceState     `json:"state,omitempty"`
	IP             string            `json:"ip,omitempty"`
	FloatingIP     string            `json:"floatingIP,omitempty"`
}

type RootVolume struct {
	Size             int    `json:"diskSize,omitempty"`
	VolumeType       string `json:"volumeType,omitempty"`
	AvailabilityZone string `json:"availabilityZone,omitempty"`
}

// Network represents basic information about an OpenStack Neutron Network associated with an instance's port.
type Network struct {
	Name string `json:"name"`
	ID   string `json:"id"`

	//+optional
	Tags []string `json:"tags,omitempty"`

	Subnet   *Subnet   `json:"subnet,omitempty"`
	PortOpts *PortOpts `json:"port,omitempty"`
	Router   *Router   `json:"router,omitempty"`

	// Be careful when using APIServerLoadBalancer, because this field is optional and therefore not
	// set in all cases
	APIServerLoadBalancer *LoadBalancer `json:"apiServerLoadBalancer,omitempty"`
}

// Subnet represents basic information about the associated OpenStack Neutron Subnet.
type Subnet struct {
	Name string `json:"name"`
	ID   string `json:"id"`

	CIDR string `json:"cidr"`

	//+optional
	Tags []string `json:"tags,omitempty"`
}

// Router represents basic information about the associated OpenStack Neutron Router.
type Router struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	//+optional
	Tags []string `json:"tags,omitempty"`
	//+optional
	IPs []string `json:"ips,omitempty"`
}

// LoadBalancer represents basic information about the associated OpenStack LoadBalancer.
type LoadBalancer struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	IP         string `json:"ip"`
	InternalIP string `json:"internalIP"`
	//+optional
	AllowedCIDRs []string `json:"allowedCIDRs,omitempty"`
	//+optional
	Tags []string `json:"tags,omitempty"`
}

// SecurityGroup represents the basic information of the associated
// OpenStack Neutron Security Group.
type SecurityGroup struct {
	Name  string              `json:"name"`
	ID    string              `json:"id"`
	Rules []SecurityGroupRule `json:"rules"`
}

// SecurityGroupRule represent the basic information of the associated OpenStack
// Security Group Role.
type SecurityGroupRule struct {
	Description     string `json:"description"`
	ID              string `json:"name"`
	Direction       string `json:"direction"`
	EtherType       string `json:"etherType"`
	SecurityGroupID string `json:"securityGroupID"`
	PortRangeMin    int    `json:"portRangeMin"`
	PortRangeMax    int    `json:"portRangeMax"`
	Protocol        string `json:"protocol"`
	RemoteGroupID   string `json:"remoteGroupID"`
	RemoteIPPrefix  string `json:"remoteIPPrefix"`
}

// Equal checks if two SecurityGroupRules are the same.
func (r SecurityGroupRule) Equal(x SecurityGroupRule) bool {
	return (r.Direction == x.Direction &&
		r.Description == x.Description &&
		r.EtherType == x.EtherType &&
		r.PortRangeMin == x.PortRangeMin &&
		r.PortRangeMax == x.PortRangeMax &&
		r.Protocol == x.Protocol &&
		r.RemoteGroupID == x.RemoteGroupID &&
		r.RemoteIPPrefix == x.RemoteIPPrefix)
}

// InstanceState describes the state of an OpenStack instance.
type InstanceState string

var (
	// InstanceStateBuilding is the string representing an instance in a building state.
	InstanceStateBuilding = InstanceState("BUILDING")

	// InstanceStateActive is the string representing an instance in an active state.
	InstanceStateActive = InstanceState("ACTIVE")

	// InstanceStateError is the string representing an instance in an error state.
	InstanceStateError = InstanceState("ERROR")

	// InstanceStateStopped is the string representing an instance in a stopped state.
	InstanceStateStopped = InstanceState("STOPPED")

	// InstanceStateShutoff is the string representing an instance in a shutoff state.
	InstanceStateShutoff = InstanceState("SHUTOFF")

	// InstanceStateDeleted is the string representing an instance in a deleted state.
	InstanceStateDeleted = InstanceState("DELETED")
)

// Bastion represents basic information about the bastion node.
type Bastion struct {
	//+optional
	Enabled bool `json:"enabled"`

	// Instance for the bastion itself
	Instance OpenStackMachineSpec `json:"instance,omitempty"`

	//+optional
	AvailabilityZone string `json:"availabilityZone,omitempty"`
}

type APIServerLoadBalancer struct {
	// Enabled defines whether a load balancer should be created.
	Enabled bool `json:"enabled,omitempty"`
	// AdditionalPorts adds additional tcp ports to the load balancer.
	AdditionalPorts []int `json:"additionalPorts,omitempty"`
	// AllowedCIDRs restrict access to all API-Server listeners to the given address CIDRs.
	AllowedCIDRs []string `json:"allowedCidrs,omitempty"`
}

// FailureDomainMachinePlacement is an enumeration of the possible machine placement strategies for a failure domain. It controls whether the failure domain is for all machines, or only workers.
// kubebuilder:validation:Enum:="All";"WorkerOnly"
type FailureDomainMachinePlacement string

const (
	// FailureDomainMachinePlacementAll denotes that a failure domain is suitable for both control plane and worker machines.
	FailureDomainMachinePlacementAll FailureDomainMachinePlacement = "All"

	// FailureDomainMachinePlacementNoControlPlane denotes that a failure domain will not be used for control plane machines.
	FailureDomainMachinePlacementNoControlPlane = "NoControlPlane"
)

const (
	FailureDomainType        = "Type"
	FailureDomainTypeAZ      = "AvailabilityZone"
	FailureDomainTypeCluster = "Cluster"
)

type FailureDomainDefinition struct {
	// Name is a string by which a failure domain is referenced at creation
	// time.
	// As this is only a reference, it is not safe to assume that all
	// machines created using this name were also using the same failure
	// domain.
	// +required
	Name string `json:"name"`

	// MachinePlacement defines which machines this failure domain is suitable for.
	// 'All' specifies that the failure domain is suitable for all machines. Control plane machines will be automatically distributed across failure domains with a MachinePlacement of All.
	// 'NoControlPlane' specifies that the failure domain will not be used by control plane machines. The failure domain may be referenced by worker machines, but will not be used by control plane machines.
	// If not specified, the default is 'All'.
	// +kubebuilder:default:="All"
	// +optional
	MachinePlacement FailureDomainMachinePlacement `json:"machinePlacement,omitempty"`

	FailureDomain `json:",inline"`
}

type FailureDomain struct {
	// ComputeAvailabilityZone is the name of a valid nova availability zone. The server will be created in this availability zone.
	// +optional
	ComputeAvailabilityZone string `json:"computeAvailabilityZone,omitempty"`

	// StorageAvailabilityZone is the name of a valid cinder availability
	// zone. This will be the availability zone of the root volume if one is
	// specified.
	// +optional
	StorageAvailabilityZone string `json:"storageAvailabilityZone,omitempty"`

	// Ports defines a set of ports and their attached networks. These will be prepended to any other ports attached to the server.
	// +optional
	Ports []PortOpts `json:"ports,omitempty"`
}
