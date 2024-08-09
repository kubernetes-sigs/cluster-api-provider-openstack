/*
Copyright 2024 The Kubernetes Authors.

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

package image

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applyconfigv1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/pkg/utils/ssa"

	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
)

const (
	glanceOSHashAlgo  = "os_hash_algo"
	glanceOSHashValue = "os_hash_value"
)

// updateObject writes to the Image root resource, i.e. everything except status.
func (r *orcImageReconciler) updateObject(ctx context.Context, orcImage *orcv1alpha1.Image) error {
	applyConfig := orcapplyconfigv1alpha1.Image(orcImage.Name, orcImage.Namespace).
		WithFinalizers(orcv1alpha1.ImageControllerFinalizer)

	return r.client.Patch(ctx, orcImage, ssa.ApplyConfigPatch(applyConfig), client.ForceOwnership, client.FieldOwner(orcv1alpha1.ImageControllerFieldOwner))
}

type updateStatusOpts struct {
	glanceImage               *images.Image
	progressMessage           *string
	err                       error
	incrementDownloadAttempts bool
}

type updateStatusOpt func(*updateStatusOpts)

func withGlanceImage(glanceImage *images.Image) updateStatusOpt {
	return func(opts *updateStatusOpts) {
		opts.glanceImage = glanceImage
	}
}

func withError(err error) updateStatusOpt {
	return func(opts *updateStatusOpts) {
		opts.err = err
	}
}

// withProgressMessage sets a custom progressing message if and only if the reconcile is progressing.
func withProgressMessage(message string) updateStatusOpt {
	return func(opts *updateStatusOpts) {
		opts.progressMessage = &message
	}
}

func withIncrementDownloadAttempts() updateStatusOpt {
	return func(opts *updateStatusOpts) {
		opts.incrementDownloadAttempts = true
	}
}

// createStatusUpdate computes a complete status update based on the given
// observed state. This is separated from updateStatus to facilitate unit
// testing, as the version of k8s we currently import does not support patch
// apply in the fake client.
// Needs: https://github.com/kubernetes/kubernetes/pull/125560
func createStatusUpdate(ctx context.Context, orcImage *orcv1alpha1.Image, now metav1.Time, opts ...updateStatusOpt) *orcapplyconfigv1alpha1.ImageApplyConfiguration {
	log := ctrl.LoggerFrom(ctx)

	statusOpts := updateStatusOpts{}
	for i := range opts {
		opts[i](&statusOpts)
	}

	glanceImage := statusOpts.glanceImage
	err := statusOpts.err

	applyConfigStatus := orcapplyconfigv1alpha1.ImageStatus()
	applyConfig := orcapplyconfigv1alpha1.Image(orcImage.Name, orcImage.Namespace).WithStatus(applyConfigStatus)

	// We want to preserve a previously set image ID even if we were unable
	// to fetch the image from glance.
	if orcImage.Status.ImageID != nil {
		applyConfigStatus.WithImageID(*orcImage.Status.ImageID)
	}

	downloadAttempts := ptr.Deref(orcImage.Status.DownloadAttempts, 0)
	if statusOpts.incrementDownloadAttempts {
		downloadAttempts++
	}
	if downloadAttempts > 0 {
		applyConfigStatus.WithDownloadAttempts(downloadAttempts)
	}

	var glanceHash *orcv1alpha1.ImageHash
	if glanceImage != nil {
		applyConfigStatus.WithImageID(glanceImage.ID)
		applyConfigStatus.WithStatus(string(glanceImage.Status))

		if glanceImage.SizeBytes > 0 {
			applyConfigStatus.WithSizeB(glanceImage.SizeBytes)
		}
		if glanceImage.VirtualSize > 0 {
			applyConfigStatus.WithVirtualSizeB(glanceImage.VirtualSize)
		}

		osHashAlgo, _ := glanceImage.Properties[glanceOSHashAlgo].(string)
		osHashValue, _ := glanceImage.Properties[glanceOSHashValue].(string)
		if osHashAlgo != "" && osHashValue != "" {
			glanceHash = &orcv1alpha1.ImageHash{
				Algorithm: orcv1alpha1.ImageHashAlgorithm(osHashAlgo),
				Value:     osHashValue,
			}
			applyConfigStatus.WithHash(
				orcapplyconfigv1alpha1.ImageHash().
					WithAlgorithm(glanceHash.Algorithm).
					WithValue(glanceHash.Value))
		}
	}

	availableCondition := applyconfigv1.Condition().
		WithType(orcv1alpha1.OpenStackConditionAvailable).
		WithObservedGeneration(orcImage.Generation)
	progressingCondition := applyconfigv1.Condition().
		WithType(orcv1alpha1.OpenStackConditionProgressing).
		WithObservedGeneration(orcImage.Generation)

	available := false
	if glanceImage != nil && glanceImage.Status == images.ImageStatusActive {
		availableCondition.
			WithStatus(metav1.ConditionTrue).
			WithReason(orcv1alpha1.OpenStackConditionReasonSuccess).
			WithMessage("Glance image is available")
		available = true
	} else {
		// Image is not available. Reason and message will be copied from Progressing
		availableCondition.WithStatus(metav1.ConditionFalse)
	}

	// We are progressing until the image is available or there was an error
	if err == nil {
		if available {
			progressingCondition.
				WithStatus(metav1.ConditionFalse).
				WithReason(orcv1alpha1.OpenStackConditionReasonSuccess).
				WithMessage(*availableCondition.Message)
		} else {
			progressingCondition.
				WithStatus(metav1.ConditionTrue).
				WithReason(orcv1alpha1.OpenStackConditionReasonProgressing)

			if statusOpts.progressMessage == nil {
				progressingCondition.WithMessage("Reconciliation is progressing")
			} else {
				progressingCondition.WithMessage(*statusOpts.progressMessage)
			}
		}
	} else {
		progressingCondition.WithStatus(metav1.ConditionFalse)

		var terminalError *capoerrors.TerminalError
		if errors.As(err, &terminalError) {
			progressingCondition.
				WithReason(terminalError.Reason).
				WithMessage(terminalError.Message)
		} else {
			progressingCondition.
				WithReason(orcv1alpha1.OpenStackConditionReasonTransientError).
				WithMessage(err.Error())
		}
	}

	// Copy available status from progressing if it's not available yet
	if !available {
		availableCondition.
			WithReason(*progressingCondition.Reason).
			WithMessage(*progressingCondition.Message)
	}

	// Maintain condition timestamps if they haven't changed
	// This also ensures that we don't generate an update event if nothing has changed
	for _, condition := range []*applyconfigv1.ConditionApplyConfiguration{availableCondition, progressingCondition} {
		previous := meta.FindStatusCondition(orcImage.Status.Conditions, *condition.Type)
		if previous != nil && ssa.ConditionsEqual(previous, condition) {
			condition.WithLastTransitionTime(previous.LastTransitionTime)
		} else {
			condition.WithLastTransitionTime(now)
		}
	}

	applyConfigStatus.WithConditions(availableCondition, progressingCondition)

	if log.V(4).Enabled() {
		logValues := make([]any, 0, 12)
		addConditionValues := func(condition *applyconfigv1.ConditionApplyConfiguration) {
			if condition.Type == nil {
				bytes, _ := json.Marshal(condition)
				log.V(0).Info("Attempting to set condition with no type", "condition", string(bytes))
				return
			}

			for _, v := range []struct {
				name  string
				value *string
			}{
				{"status", (*string)(condition.Status)},
				{"reason", condition.Reason},
				{"message", condition.Message},
			} {
				logValues = append(logValues, *condition.Type+"."+v.name, ptr.Deref(v.value, ""))
			}
		}
		addConditionValues(availableCondition)
		addConditionValues(progressingCondition)
		log.V(4).Info("Setting image status", logValues...)
	}

	return applyConfig
}

// updateStatus computes a complete status based on the given observed state and writes it to status.
func (r *orcImageReconciler) updateStatus(ctx context.Context, orcImage *orcv1alpha1.Image, opts ...updateStatusOpt) error {
	now := metav1.NewTime(time.Now())

	statusUpdate := createStatusUpdate(ctx, orcImage, now, opts...)

	return r.client.Status().Patch(ctx, orcImage, ssa.ApplyConfigPatch(statusUpdate), client.ForceOwnership, client.FieldOwner(orcv1alpha1.ImageControllerFieldOwner))
}
