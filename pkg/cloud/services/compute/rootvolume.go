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

package compute

import (
	"fmt"

	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/attachments"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"k8s.io/apimachinery/pkg/runtime"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
)

func (s *Service) reconcileRootVolume(eventObject runtime.Object, instanceSpec *InstanceSpec, resources *infrav1.OpenStackMachineResources, instanceStatus *InstanceStatus) (bool, error) {
	rootVolumeSpec := instanceSpec.RootVolume
	rootVolumeStatus := &resources.RootVolume

	// Nothing to do if there's no root volume
	if !hasRootVolume(rootVolumeSpec) {
		return true, nil
	}

	err := s.adoptRootVolume(instanceSpec.Name, rootVolumeSpec, resources, instanceStatus)
	if err != nil {
		return false, fmt.Errorf("error adopting root volume for reconcile: %w", err)
	}

	// Nothing to do if the root volume is already created
	if rootVolumeStatus.Ready {
		return true, nil
	}

	// Volume has already been created. Check if it is available yet
	if rootVolumeStatus.ID != "" {
		volume, err := s.computeService.GetVolume(rootVolumeStatus.ID)
		if err != nil {
			return false, fmt.Errorf("error getting volume %s for machine %s: %w", rootVolumeStatus.ID, instanceSpec.Name, err)
		}

		if volume.Status == "available" {
			rootVolumeStatus.Ready = true
			record.Eventf(eventObject, "RootVolumeReady", "Root volume %s became ready", rootVolumeStatus.ID)
			return true, nil
		}

		if volume.Status == "error" {
			record.Eventf(eventObject, "RootVolumeError", "Root volume %s is in error state", rootVolumeStatus.ID)
			return true, fmt.Errorf("root volume %s is in error state", rootVolumeStatus.ID)
		}

		// Volume is still being created
		return false, nil
	}

	availabilityZone := instanceSpec.FailureDomain
	// Explicit root volume AZ overrides machine's failure domain
	if rootVolumeSpec.AvailabilityZone != "" {
		availabilityZone = rootVolumeSpec.AvailabilityZone
	}

	imageID, err := s.getImageID(instanceSpec.ImageUUID, instanceSpec.Image)
	if err != nil {
		return false, fmt.Errorf("error getting image ID: %v", err)
	}

	createOpts := volumes.CreateOpts{
		Size:             rootVolumeSpec.Size,
		Description:      fmt.Sprintf("Root volume for %s", instanceSpec.Name),
		Name:             rootVolumeName(instanceSpec.Name),
		ImageID:          imageID,
		Multiattach:      false,
		AvailabilityZone: availabilityZone,
		VolumeType:       rootVolumeSpec.VolumeType,
	}
	volume, err := s.computeService.CreateVolume(createOpts)
	if err != nil {
		record.Eventf(eventObject, "FailedCreateVolume", "Failed to create root volume; size=%d imageID=%s err=%v", rootVolumeSpec.Size, imageID, err)
		return false, fmt.Errorf("error creating root volume for machine %s: %w", instanceSpec.Name, err)
	}

	record.Eventf(eventObject, "SuccessfulCreateVolume", "Created root volume; id=%s", volume.ID)
	rootVolumeStatus.ID = volume.ID

	// Still need to wait for the volume to become available
	return false, nil
}

func (s *Service) adoptRootVolume(instanceName string, rootVolume *infrav1.RootVolume, resources *infrav1.OpenStackMachineResources, instanceStatus *InstanceStatus) error {
	// Nothing to do if the root volume is already adopted
	if resources.RootVolume != (infrav1.OpenStackResource{}) {
		return nil
	}

	// The primary purpose of storing the root volume ID after creation is
	// so we can ensure it is deleted with the machine. However, when the
	// server is created we set delete_on_termination on the volume.
	// Consequently, if the server exists we can get away with not trying to
	// work out the id of the root volume. This avoids additional API calls
	// and having to rely on heuristics for adoption.
	//
	// This is the primary mechanism for adopting a root volume from older
	// versions of CAPO.
	if instanceStatus != nil {
		// The root volume must be ready if the server exists
		resources.RootVolume.Ready = true

		s.logger.Info("not adopting root volume for existing server %s(%s)", instanceStatus.ID(), instanceStatus.Name())
		return nil
	}

	// Look for an existing volume with the expected name
	// This covers a previous crash of CAPO during instance creation, or a
	// previous failure of the API server when updating the machine status
	listOpts := volumes.ListOpts{
		Name: rootVolumeName(instanceName),
	}
	volumeList, err := s.computeService.ListVolumes(listOpts)
	if err != nil {
		return fmt.Errorf("error listing volumes: %w", err)
	}

	candidates := []*volumes.Volume{}
	for i := range volumeList {
		volume := &volumeList[i]

		if volume.Size != rootVolume.Size {
			s.logger.Info("not adopting volume %s machine %s because size %d does not match expected size %d", volume.ID, instanceName, volume.Size, rootVolume.Size)
			continue
		}

		attachmentsListOpts := attachments.ListOpts{
			VolumeID: volume.ID,
		}
		attachmentList, err := s.computeService.ListVolumeAttachments(attachmentsListOpts)
		if err != nil {
			return fmt.Errorf("error listing attachments for volume %s: %w", volume.ID, err)
		}
		if len(attachmentList) > 0 {
			s.logger.Info("not adopting volume %s for machine %s because it is already attached to instance %s", volume.ID, instanceName, attachmentList[0].Instance)
			continue
		}
		candidates = append(candidates, volume)
	}

	if len(candidates) == 0 {
		s.logger.Info("no existing volume found for machine %s", instanceName)
		return nil
	}

	if len(candidates) == 1 {
		resources.RootVolume.ID = candidates[0].ID
		resources.RootVolume.Ready = candidates[0].Status == "available"
		return nil
	}

	return fmt.Errorf("found multiple potential root volumes named %s for machine %s", rootVolumeName(instanceName), instanceName)
}

func (s *Service) deleteRootVolume(eventObject runtime.Object, openStackMachineSpec *infrav1.OpenStackMachineSpec, instanceName string, resources *infrav1.OpenStackMachineResources, instanceStatus *InstanceStatus) error {
	rootVolume := openStackMachineSpec.RootVolume

	// Nothing to do if there's no root volume
	if !hasRootVolume(rootVolume) {
		return nil
	}

	if err := s.adoptRootVolume(instanceName, rootVolume, resources, instanceStatus); err != nil {
		return err
	}

	if resources.RootVolume.ID == "" {
		// No need to delete this root volume explicitly
		return nil
	}

	s.logger.Info("deleting dangling root volume %s", resources.RootVolume.ID)
	if err := s.computeService.DeleteVolume(resources.RootVolume.ID, volumes.DeleteOpts{}); err != nil {
		record.Eventf(eventObject, "FailedDeleteRootVolume", "Failed to delete root volume %s: %v", resources.RootVolume.ID, err)
		return fmt.Errorf("error deleting root volume %s: %w", resources.RootVolume.ID, err)
	}

	record.Eventf(eventObject, "SuccessfulDeleteRootVolume", "Deleted root volume %s", resources.RootVolume.ID)
	return nil
}

func rootVolumeName(instanceName string) string {
	return fmt.Sprintf("%s-root", instanceName)
}

func hasRootVolume(rootVolume *infrav1.RootVolume) bool {
	return rootVolume != nil && rootVolume.Size > 0
}
