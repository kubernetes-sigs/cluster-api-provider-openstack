/*
Copyright 2021 The Kubernetes Authors.

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

package compute

import (
	"encoding/json"
	"fmt"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha4"
)

// InstanceSpec defines the fields which can be set on a new OpenStack instance.
//
// InstanceSpec does not contain all of the fields of infrav1.Instance, as not
// all of them can be set on a new instance.
type InstanceSpec struct {
	Name           string
	Image          string
	Flavor         string
	SSHKeyName     string
	UserData       string
	Metadata       map[string]string
	ConfigDrive    bool
	FailureDomain  string
	RootVolume     *infrav1.RootVolume
	Subnet         string
	ServerGroupID  string
	Trunk          bool
	Tags           []string
	SecurityGroups []string
	Networks       []infrav1.Network
}

// InstanceIdentifier describes an instance which has not necessarily been fetched.
type InstanceIdentifier struct {
	ID   string
	Name string
}

// InstanceStatus represents instance data which has been returned by OpenStack.
type InstanceStatus struct {
	server *servers.Server
}

func NewInstanceStatusFromServer(server *servers.Server) *InstanceStatus {
	return &InstanceStatus{server}
}

type networkInterface struct {
	Address string  `json:"addr"`
	Version float64 `json:"version"`
	Type    string  `json:"OS-EXT-IPS:type"`
}

// InstanceNetworkStatus represents the network status of an OpenStack instance
// as used by CAPO. Therefore it may use more context than just data which was
// returned by OpenStack.
type InstanceNetworkStatus struct {
	addresses map[string][]networkInterface
}

func (is *InstanceStatus) ID() string {
	return is.server.ID
}

func (is *InstanceStatus) Name() string {
	return is.server.Name
}

func (is *InstanceStatus) State() infrav1.InstanceState {
	return infrav1.InstanceState(is.server.Status)
}

func (is *InstanceStatus) SSHKeyName() string {
	return is.server.KeyName
}

// APIInstance returns an infrav1.Instance object for use by the API.
func (is *InstanceStatus) APIInstance() (*infrav1.Instance, error) {
	i := infrav1.Instance{
		ID:         is.ID(),
		Name:       is.Name(),
		SSHKeyName: is.SSHKeyName(),
		State:      is.State(),
	}

	ns, err := is.NetworkStatus()
	if err != nil {
		return nil, err
	}

	i.IP = ns.IP()
	i.FloatingIP = ns.FloatingIP()

	return &i, nil
}

// InstanceIdentifier returns an InstanceIdentifier object for an InstanceStatus.
func (is *InstanceStatus) InstanceIdentifier() *InstanceIdentifier {
	return &InstanceIdentifier{
		ID:   is.ID(),
		Name: is.Name(),
	}
}

// NetworkStatus returns an InstanceNetworkStatus object for an InstanceStatus.
func (is *InstanceStatus) NetworkStatus() (*InstanceNetworkStatus, error) {
	addresses := make(map[string][]networkInterface)

	for networkName, b := range is.server.Addresses {
		list, err := json.Marshal(b)
		if err != nil {
			return nil, fmt.Errorf("error marshalling addresses for instance %s: %w", is.ID(), err)
		}
		var networkList []networkInterface
		err = json.Unmarshal(list, &networkList)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling addresses for instance %s: %w", is.ID(), err)
		}

		addresses[networkName] = networkList
	}

	return &InstanceNetworkStatus{addresses}, nil
}

func (ns *InstanceNetworkStatus) IP() string {
	// Return the last listed non-floating IPv4 from the last listed network
	// This behaviour is wrong, but consistent with the previous behaviour
	// https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/4debc1fc4742e483302b0c36b16c076977bd165d/pkg/cloud/services/compute/instance.go#L973-L998
	// XXX: Fix this
	for _, vifs := range ns.addresses {
		for _, vif := range vifs {
			if vif.Version == 4.0 && vif.Type != "floating" {
				return vif.Address
			}
		}
	}
	return ""
}

func (ns *InstanceNetworkStatus) FloatingIP() string {
	// Return the last listed floating IPv4 from the last listed network
	// This behaviour is wrong, but consistent with the previous behaviour
	// https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/4debc1fc4742e483302b0c36b16c076977bd165d/pkg/cloud/services/compute/instance.go#L973-L998
	// XXX: Fix this
	for _, vifs := range ns.addresses {
		for _, vif := range vifs {
			if vif.Version == 4.0 && vif.Type == "floating" {
				return vif.Address
			}
		}
	}
	return ""
}
