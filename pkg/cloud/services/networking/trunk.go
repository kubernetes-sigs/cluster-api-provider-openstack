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

package networking

import (
	"context"
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"

	infrav1alpha1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
)

const (
	timeoutTrunkDelete         = 3 * time.Minute
	retryIntervalTrunkDelete   = 5 * time.Second
	retryIntervalSubportDelete = 30 * time.Second
)

func (s *Service) GetTrunkSupport() (bool, error) {
	allExts, err := s.client.ListExtensions()
	if err != nil {
		return false, err
	}

	for _, ext := range allExts {
		if ext.Alias == "trunk" {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) getOrCreateTrunkForPort(eventObject runtime.Object, port *ports.Port) (*trunks.Trunk, error) {
	trunkList, err := s.client.ListTrunk(trunks.ListOpts{
		Name:   port.Name,
		PortID: port.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("searching for existing trunk for server: %v", err)
	}

	if len(trunkList) != 0 {
		return &trunkList[0], nil
	}

	trunkCreateOpts := trunks.CreateOpts{
		Name:        port.Name,
		PortID:      port.ID,
		Description: port.Description,
	}

	trunk, err := s.client.CreateTrunk(trunkCreateOpts)
	if err != nil {
		return nil, err
	}

	record.Eventf(eventObject, "SuccessfulCreateTrunk", "Created trunk %s with id %s", trunk.Name, trunk.ID)
	return trunk, nil
}

func (s *Service) RemoveTrunkSubports(trunkID string) error {
	subports, err := s.client.ListTrunkSubports(trunkID)
	if err != nil {
		return err
	}

	if len(subports) == 0 {
		return nil
	}

	portList := make([]trunks.RemoveSubport, len(subports))
	for i, subport := range subports {
		portList[i] = trunks.RemoveSubport{
			PortID: subport.PortID,
		}
	}

	removeSubportsOpts := trunks.RemoveSubportsOpts{
		Subports: portList,
	}

	err = s.client.RemoveSubports(trunkID, removeSubportsOpts)
	if err != nil {
		return err
	}

	for _, subPort := range subports {
		err := s.client.DeletePort(subPort.PortID)
		if err != nil {
			return err
		}
	}

	return nil
}

// EnsurePorts ensures that every one of desiredPorts is created and has
// expected trunk and tags.
func (s *Service) EnsureTrunkSubPorts(eventObject runtime.Object, desiredPorts []infrav1.ResolvedPortSpec, resources *infrav1alpha1.ServerResources) error {
	for i := range desiredPorts {
		// Fail here if trunk port has not been created
		if i >= len(resources.Ports) {
			return fmt.Errorf("ports have not been created")
		}

		var newSubports []trunks.Subport
		if ptr.Deref(desiredPorts[i].Trunk, false) {
			// Retrieve trunk subports and transform it to map for faster validation
			trnks, err := s.client.ListTrunk(trunks.ListOpts{PortID: resources.Ports[i].ID})
			if err != nil {
				return fmt.Errorf("searching trunk: %v", err)
			}

			if len(trnks) > 1 {
				return fmt.Errorf("multiple trunks found for port \"%s\"", resources.Ports[i].ID)
			}

			if len(trnks) == 0 {
				return fmt.Errorf("trunks not found for port \"%s\"", resources.Ports[i].ID)
			}

			existingSubports, err := s.client.ListTrunkSubports(trnks[0].ID)
			if err != nil {
				return fmt.Errorf("searching for existing port for server: %v", err)
			}
			subportMap := make(map[string]trunks.Subport)
			for _, existringSubport := range existingSubports {
				subportMap[existringSubport.PortID] = existringSubport
			}

			// Ensure trunk subports are created as ports
			for j, desiredSubport := range desiredPorts[i].Subports {
				subportStatus := infrav1.PortStatus{}
				if j < len(resources.Ports[i].Subports) {
					subportStatus.ID = resources.Ports[i].Subports[j].ID
				}
				subport, err := s.EnsurePort(eventObject, &infrav1.ResolvedPortSpec{CommonResolvedPortSpec: desiredSubport.CommonResolvedPortSpec}, subportStatus)
				if err != nil {
					return err
				}

				if _, found := subportMap[subport.ID]; !found {
					newSubports = append(newSubports, trunks.Subport{
						SegmentationID:   desiredSubport.SegmentationID,
						SegmentationType: desiredSubport.SegmentationType,
						PortID:           subport.ID,
					})
				}

				if j < len(resources.Ports[i].Subports) {
					resources.Ports[i].Subports[j].ID = subportStatus.ID
				} else {
					resources.Ports[i].Subports = append(resources.Ports[i].Subports, infrav1.SubPortStatus{ID: subport.ID})
				}
			}

			// Append subports to trunk
			if len(newSubports) > 0 {
				_, err = s.client.AddSubports(trnks[0].ID, trunks.AddSubportsOpts{Subports: newSubports})
				if err != nil {
					return fmt.Errorf("failed to add subports to trunk %s: %v", trnks[0].ID, err)
				}
			}
		}
	}

	return nil
}

func (s *Service) DeleteTrunk(eventObject runtime.Object, portID string) error {
	listOpts := trunks.ListOpts{
		PortID: portID,
	}
	trunkInfo, err := s.client.ListTrunk(listOpts)
	if err != nil {
		return err
	}
	if len(trunkInfo) != 1 {
		return nil
	}
	// Delete sub-ports if trunk is associated with sub-ports
	err = wait.PollUntilContextTimeout(context.TODO(), retryIntervalSubportDelete, timeoutTrunkDelete, true, func(_ context.Context) (bool, error) {
		if err := s.RemoveTrunkSubports(trunkInfo[0].ID); err != nil {
			if capoerrors.IsNotFound(err) || capoerrors.IsConflict(err) || capoerrors.IsRetryable(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if err != nil {
		record.Warnf(eventObject, "FailedRemoveTrunkSubports", "Failed to delete sub ports trunk %s with id %s: %v", trunkInfo[0].Name, trunkInfo[0].ID, err)
		return err
	}

	record.Eventf(eventObject, "SuccessfulRemoveTrunkSubports", "Removed trunk sub ports %s with id %s", trunkInfo[0].Name, trunkInfo[0].ID)

	err = wait.PollUntilContextTimeout(context.TODO(), retryIntervalTrunkDelete, timeoutTrunkDelete, true, func(_ context.Context) (bool, error) {
		if err := s.client.DeleteTrunk(trunkInfo[0].ID); err != nil {
			if capoerrors.IsNotFound(err) {
				record.Eventf(eventObject, "SuccessfulDeleteTrunk", "Trunk %s with id %s did not exist", trunkInfo[0].Name, trunkInfo[0].ID)
				return true, nil
			}
			if capoerrors.IsConflict(err) {
				return false, nil
			}
			if capoerrors.IsRetryable(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if err != nil {
		record.Warnf(eventObject, "FailedDeleteTrunk", "Failed to delete trunk %s with id %s: %v", trunkInfo[0].Name, trunkInfo[0].ID, err)
		return err
	}

	record.Eventf(eventObject, "SuccessfulDeleteTrunk", "Deleted trunk %s with id %s", trunkInfo[0].Name, trunkInfo[0].ID)
	return nil
}
