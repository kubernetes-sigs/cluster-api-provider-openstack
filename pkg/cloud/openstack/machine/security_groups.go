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
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	openstackconfigv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (oc *OpenstackClient) updateSecurityGroups(cluster *clusterv1.Cluster, machine *clusterv1.Machine, providerConfig *openstackconfigv1.OpenstackProviderConfig) (err error) {

	serverID := machine.ObjectMeta.Annotations[OpenstackIdAnnotationKey]

	annotationSecurityGroups, err := oc.machineAnnotationJSON(machine, LastKnownSecurityGroups)
	if err != nil {
		return err
	}

	specSecurityGroups := providerConfig.SecurityGroups
	openstackServerSecurityGroups, err := oc.machineService.GetOpenstackSecurityGroups(serverID)
	if err != nil {
		return err
	}

	updatedSG := false
	updatedAnnotation := copyAnnotation(annotationSecurityGroups)

	// add security groups added to spec
	addedSecurityGroups := []string{}
	for _, sgName := range specSecurityGroups {
		if _, ok := openstackServerSecurityGroups[sgName]; !ok {
			addedSecurityGroups = append(addedSecurityGroups, sgName)
		}
	}

	openstackProjectSecurityGroups := map[string]secgroups.SecurityGroup{}
	if len(addedSecurityGroups) > 0 {
		// need to get all openstack project security groups to correlate the sg name and sg ID
		openstackProjectSecurityGroups, err = oc.machineService.GetAllOpenstackSecurityGroups()
		if err != nil {
			return err
		}
	}

	for _, sgName := range addedSecurityGroups {
		if updatedSG, err = oc.machineService.AddSecurityGroup(sgName, serverID, openstackProjectSecurityGroups, updatedAnnotation); err != nil {
			return err
		}
	}

	// remove security groups deleted from spec
	for sgName, sg := range openstackServerSecurityGroups {
		_, previouslyExisted := annotationSecurityGroups[sgName]
		if ok := !contains(specSecurityGroups, sgName) && previouslyExisted; ok {
			if updatedSG, err = oc.machineService.RemoveSecurityGroup(sgName, sg.ID, serverID, updatedAnnotation); err != nil {
				if updatedSG {
					// If adding SG succeeds, but removing SG fails, still update annotation. Otherwise we can never remove that SG
					oc.updateMachineAnnotationJSON(machine, LastKnownSecurityGroups, updatedAnnotation)
				}
				return err
			}
		}
	}

	if updatedSG {
		if err := oc.updateMachineAnnotationJSON(machine, LastKnownSecurityGroups, updatedAnnotation); err != nil {
			return err
		}
	}

	return nil
}

func copyAnnotation(annotationSecurityGroups map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{}, len(annotationSecurityGroups))
	for key, value := range annotationSecurityGroups {
		copy[key] = value
	}
	return copy
}

func contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}
