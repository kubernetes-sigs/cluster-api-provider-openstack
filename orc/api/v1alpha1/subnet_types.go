/*
Copyright 2024 The ORC Authors.

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

package v1alpha1

// TODO validations:
//
// * IP addresses in CIDR, AllocationPools, Gateway, DNSNameserver(?), and
//   HostRoutes match the version in IPVersion (Spec and SubnetFilter)
// * IPv6 may only be set if IPVersion is 6 (Spec and SubnetFilter)
// * AllocationPools must be in CIDR

// SubnetFilter specifies a filter to select a subnet. At least one parameter must be specified.
// +kubebuilder:validation:MinProperties:=1
type SubnetFilter struct {
	Name        *OpenStackName        `json:"name,omitempty"`
	Description *OpenStackDescription `json:"description,omitempty"`
	ProjectID   *UUID                 `json:"projectID,omitempty"`
	IPVersion   *IPVersion            `json:"ipVersion,omitempty"`
	GatewayIP   *IPvAny               `json:"gatewayIP,omitempty"`
	CIDR        *CIDR                 `json:"cidr,omitempty"`
	IPv6        *IPv6Options          `json:"ipv6,omitempty"`

	FilterByNeutronTags `json:",inline"`
}

type SubnetResourceSpec struct {
	// NetworkRef is a reference to the ORC Network which this subnet is associated with.
	NetworkRef ORCNameRef `json:"networkRef"`

	// Name is a human-readable name of the subnet. If not set, the object's name will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// Description of the subnet.
	// +optional
	Description *OpenStackDescription `json:"description,omitempty"`

	// Tags is a list of tags which will be applied to the subnet.
	// +kubebuilder:validation:MaxItems:=32
	// +listType=set
	Tags []NeutronTag `json:"tags,omitempty"`

	// IPVersion is the IP version for the subnet.
	// +required
	IPVersion IPVersion `json:"ipVersion"`

	// CIDR is the address CIDR of the subnet. It must match the IP version specified in IPVersion.
	// +required
	CIDR CIDR `json:"cidr"`

	// ProjectID is the unique ID of the project which owns the Subnet. Only
	// administrative users can specify a project UUID other than their own.
	// +optional
	ProjectID *UUID `json:"projectID,omitempty"`

	// AllocationPools are IP Address pools that will be available for DHCP. IP
	// addresses must be in CIDR.
	// +kubebuilder:validation:MaxItems:=32
	// +listType=set
	AllocationPools []AllocationPool `json:"allocationPools,omitempty"`

	// Gateway specifies the default gateway of the subnet. If not specified,
	// neutron will add one automatically. To disable this behaviour, specify a
	// gateway with a type of None.
	// +optional
	Gateway *SubnetGateway `json:"gateway,omitempty"`

	// EnableDHCP will either enable to disable the DHCP service.
	// +optional
	EnableDHCP *bool `json:"enableDHCP,omitempty"`

	// DNSNameservers are the nameservers to be set via DHCP.
	// +kubebuilder:validation:MaxItems:=16
	// +listType=set
	DNSNameservers []IPvAny `json:"dnsNameservers,omitempty"`

	// DNSPublishFixedIP will either enable or disable the publication of fixed IPs to the DNS
	// +optional
	DNSPublishFixedIP *bool `json:"dnsPublishFixedIP,omitempty"`

	// HostRoutes are any static host routes to be set via DHCP.
	// +kubebuilder:validation:MaxItems:=256
	// +listType=set
	HostRoutes []HostRoute `json:"hostRoutes,omitempty"`

	// IPv6 contains IPv6-specific options. It may only be set if IPVersion is 6.
	IPv6 *IPv6Options `json:"ipv6,omitempty"`

	// TODO: Support service types
	// TODO: Support subnet pools
}

type SubnetResourceStatus struct {
	// UUID of the parent network.
	NetworkID UUID `json:"networkID"`

	// Name is the human-readable name of the subnet. Might not be unique.
	Name OpenStackName `json:"name"`

	// Description for the subnet.
	Description *OpenStackDescription `json:"description,omitempty"`

	// IPVersion specifies IP version, either `4' or `6'.
	IPVersion IPVersion `json:"ipVersion"`

	// CIDR representing IP range for this subnet, based on IP version.
	CIDR CIDR `json:"cidr"`

	// GatewayIP is the default gateway used by devices in this subnet, if any.
	GatewayIP *IPvAny `json:"gatewayIP,omitempty"`

	// DNSNameservers is a list of name servers used by hosts in this subnet.
	// +listType=atomic
	DNSNameservers []IPvAny `json:"dnsNameservers"`

	// DNSPublishFixedIP specifies whether the fixed IP addresses are published to the DNS.
	DNSPublishFixedIP bool `json:"dnsPublishFixedIP,omitempty"`

	// AllocationPools is a list of sub-ranges within CIDR available for dynamic
	// allocation to ports.
	// +listType=atomic
	AllocationPools []AllocationPool `json:"allocationPools,omitempty"`

	// HostRoutes is a list of routes that should be used by devices with IPs
	// from this subnet (not including local subnet route).
	// +listType=atomic
	HostRoutes []HostRoute `json:"hostRoutes,omitempty"`

	// Specifies whether DHCP is enabled for this subnet or not.
	EnableDHCP bool `json:"enableDHCP"`

	// ProjectID is the project owner of the subnet.
	ProjectID UUID `json:"projectID"`

	// The IPv6 address modes specifies mechanisms for assigning IPv6 IP addresses.
	IPv6AddressMode *IPv6AddressMode `json:"ipv6AddressMode,omitempty"`

	// The IPv6 router advertisement specifies whether the networking service
	// should transmit ICMPv6 packets.
	IPv6RAMode *IPv6RAMode `json:"ipv6RAMode,omitempty"`

	// SubnetPoolID is the id of the subnet pool associated with the subnet.
	SubnetPoolID *UUID `json:"subnetPoolID,omitempty"`

	// Tags optionally set via extensions/attributestags
	// +listType=atomic
	Tags []NeutronTag `json:"tags,omitempty"`

	// RevisionNumber optionally set via extensions/standard-attr-revisions
	RevisionNumber *int64 `json:"revisionNumber,omitempty"`
}

// +kubebuilder:validation:Enum:=slaac;dhcpv6-stateful;dhcpv6-stateless
type IPv6AddressMode string

const (
	IPv6AddressModeSLAAC           = "slaac"
	IPv6AddressModeDHCPv6Stateful  = "dhcpv6-stateful"
	IPv6AddressModeDHCPv6Stateless = "dhcpv6-stateless"
)

// +kubebuilder:validation:Enum:=slaac;dhcpv6-stateful;dhcpv6-stateless
type IPv6RAMode string

const (
	IPv6RAModeSLAAC           = "slaac"
	IPv6RAModeDHCPv6Stateful  = "dhcpv6-stateful"
	IPv6RAModeDHCPv6Stateless = "dhcpv6-stateless"
)

// +kubebuilder:validation:MinProperties:=1
type IPv6Options struct {
	// AddressMode specifies mechanisms for assigning IPv6 IP addresses.
	AddressMode *IPv6AddressMode `json:"addressMode,omitempty"`

	// RAMode specifies the IPv6 router advertisement mode. It specifies whether
	// the networking service should transmit ICMPv6 packets.
	RAMode *IPv6RAMode `json:"raMode,omitempty"`
}

type SubnetGateway struct {
	// Type specifies how the default gateway will be created. `Automatic`
	// specifies that neutron will automatically add a default gateway. This is
	// also the default if no Gateway is specified. `None` specifies that the
	// subnet will not have a default gateway. `IP` specifies that the subnet
	// will use a specific address as the default gateway, which must be
	// specified in `IP`.
	// +kubebuilder:validation:Enum:=None;Automatic;IP
	// +required
	Type string `json:"type"`

	// IP is the IP address of the default gateway, which must be specified if
	// Type is `IP`. It must be a valid IP address, either IPv4 or IPv6,
	// matching the IPVersion in SubnetResourceSpec.
	// +optional
	IP *IPvAny `json:"ip,omitempty"`
}

type AllocationPool struct {
	// +required
	Start IPvAny `json:"start"`

	// +required
	End IPvAny `json:"end"`
}

type HostRoute struct {
	Destination CIDR   `json:"destination"`
	NextHop     IPvAny `json:"nextHop"`
}
