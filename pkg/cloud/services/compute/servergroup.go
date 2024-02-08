/*
Copyright 2023 The Kubernetes Authors.

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
	"fmt"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/servergroups"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha8"
)

// GetServerGroupID looks up a server group using the passed filter and returns
// its ID. It'll return an error when server group is not found or there are multiple.
func (s *Service) GetServerGroupID(serverGroupFilter *infrav1.ServerGroupFilter) (string, error) {
	// NOTE(dalees): This is an early exit if ID is set, and never hits the Nova API with the passed ID. This is not what the function description says it does.
	//               but it's probably okay; creating a server using the ID will surface the error.
	if serverGroupFilter.ID != "" {
		return serverGroupFilter.ID, nil
	}

	if serverGroupFilter.Name == "" {
		// empty filter produced no server group, but also no error
		return "", nil
	}

	// otherwise fallback to looking up by name, which is slower
	serverGroup, err := s.GetServerGroupByName(serverGroupFilter.Name, false)
	if err != nil {
		return "", err
	}

	return serverGroup.ID, nil
}

// TODO(dalees): This second parameter is messy. It would be better to differentiate 404, 'too many' and 'actual failure' in the caller.
func (s *Service) GetServerGroupByName(serverGroupName string, ignoreNotFound bool) (*servergroups.ServerGroup, error) {
	allServerGroups, err := s.getComputeClient().ListServerGroups()
	if err != nil {
		return nil, err
	}

	serverGroups := []servergroups.ServerGroup{}

	for _, serverGroup := range allServerGroups {
		if serverGroupName == serverGroup.Name {
			serverGroups = append(serverGroups, serverGroup)
		}
	}

	switch len(serverGroups) {
	case 0:
		if ignoreNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("no server group with name %s could be found", serverGroupName)
	case 1:
		return &serverGroups[0], nil
	default:
		// this will never happen due to duplicate IDs, only duplicate names, so our error message is worded accordingly
		return &serverGroups[0], fmt.Errorf("too many server groups with name %s were found", serverGroupName)
	}
}

func (s *Service) CreateServerGroup(serverGroupName string, policy string) (*servergroups.ServerGroup, error) {
	return s.getComputeClient().CreateServerGroup(serverGroupName, policy)
}

func (s *Service) DeleteServerGroup(id string) error {
	return s.getComputeClient().DeleteServerGroup(id)
}
