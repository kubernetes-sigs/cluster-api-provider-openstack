// Copyright © 2018 The Kubernetes Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package machine

import (
	"encoding/json"
	"fmt"
	"net"

	openstackconfigv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/clients"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	OpenstackIPAnnotationKey = "openstack-ip-address"
	OpenstackIdAnnotationKey = "openstack-resourceId"
	// LastKnownSecurityGroups is the key to fetch the machine object annotation data,
	// which contains the security groups that the machine actuator manages.
	LastKnownSecurityGroups = "openstack-security-groups"
)

func (oc *OpenstackClient) updateAnnotation(machine *clusterv1.Machine, instance *clients.Instance, providerConfig *openstackconfigv1.OpenstackProviderConfig) error {

	if machine.ObjectMeta.Annotations == nil {
		machine.ObjectMeta.Annotations = make(map[string]string)
	}

	ip, err := getIPFromInstance(instance)
	if err != nil {
		return err
	}

	oc.updateMachineAnnotation(machine, OpenstackIPAnnotationKey, ip)
	oc.updateMachineAnnotation(machine, OpenstackIdAnnotationKey, instance.ID)
	if err := oc.updateMachineAnnotationJSON(machine, LastKnownSecurityGroups, listToMap(providerConfig.SecurityGroups)); err != nil {
		return err
	}

	return oc.updateInstanceStatus(machine)
}

// updateMachineAnnotationJSON updates the `annotation` on `machine` with
// `content`. `content` in this case should be a `map[string]interface{}`
// suitable for turning into JSON. This `content` map will be marshalled into a
// JSON string before being set as the given `annotation`.
func (oc *OpenstackClient) updateMachineAnnotationJSON(machine *clusterv1.Machine, annotation string, content map[string]interface{}) error {
	b, err := json.Marshal(content)
	if err != nil {
		return err
	}

	oc.updateMachineAnnotation(machine, annotation, string(b))

	if err := oc.client.Update(nil, machine); err != nil {
		return err
	}
	return nil
}

// updateMachineAnnotation updates the `annotation` on the given `machine` with
// `content`.
func (oc *OpenstackClient) updateMachineAnnotation(machine *clusterv1.Machine, annotation string, content string) {
	// Get the annotations
	annotations := machine.GetAnnotations()

	// Set our annotation to the given content.
	annotations[annotation] = content

	// Update the machine object with these annotations
	machine.SetAnnotations(annotations)
}

// Returns a map[string]interface from a JSON annotation.
// This method gets the given `annotation` from the `machine` and unmarshalls it
// from a JSON string into a `map[string]interface{}`.
func (oc *OpenstackClient) machineAnnotationJSON(machine *clusterv1.Machine, annotation string) (map[string]interface{}, error) {
	out := map[string]interface{}{}

	jsonAnnotation := oc.machineAnnotation(machine, annotation)
	if len(jsonAnnotation) == 0 {
		return out, nil
	}

	err := json.Unmarshal([]byte(jsonAnnotation), &out)
	if err != nil {
		return out, err
	}

	return out, nil
}

// Fetches the specific machine annotation.
func (oc *OpenstackClient) machineAnnotation(machine *clusterv1.Machine, annotation string) string {
	return machine.GetAnnotations()[annotation]
}

func getIPFromInstance(instance *clients.Instance) (string, error) {
	if instance.AccessIPv4 != "" && net.ParseIP(instance.AccessIPv4) != nil {
		return instance.AccessIPv4, nil
	}
	type networkInterface struct {
		Address string  `json:"addr"`
		Version float64 `json:"version"`
		Type    string  `json:"OS-EXT-IPS:type"`
	}
	var addrList []string

	for _, b := range instance.Addresses {
		list, err := json.Marshal(b)
		if err != nil {
			return "", fmt.Errorf("extract IP from instance err: %v", err)
		}
		var networks []interface{}
		json.Unmarshal(list, &networks)
		for _, network := range networks {
			var netInterface networkInterface
			b, _ := json.Marshal(network)
			json.Unmarshal(b, &netInterface)
			if netInterface.Version == 4.0 {
				if netInterface.Type == "floating" {
					return netInterface.Address, nil
				}
				addrList = append(addrList, netInterface.Address)
			}
		}
	}
	if len(addrList) != 0 {
		return addrList[0], nil
	}
	return "", fmt.Errorf("extract IP from instance err")
}

// TODO make this a utility method elsewhere?
func listToMap(content []string) map[string]interface{} {
	mapK := make(map[string]interface{}, len(content))
	for _, k := range content {
		mapK[k] = struct{}{}
	}
	return mapK
}
