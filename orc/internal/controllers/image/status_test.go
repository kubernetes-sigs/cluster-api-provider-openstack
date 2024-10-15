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
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applyconfigv1 "k8s.io/client-go/applyconfigurations/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

func Test_orcImageReconciler_updateStatus(t *testing.T) {
	const (
		progressingMsg      = "Reconciliation is progressing"
		successMsg          = "Glance image is available"
		hashVerifyFailedMsg = "Hash published by glance does not match expected image hash"

		imageID         = "dc4d187c-a93c-4700-ab32-975c783c4b97"
		testSHA512value = "acbca8ab710885fb4a62e236491276b59178e6c7b3ec17b1578f784bcc8d946accee686be8477aa911b7e0266e956eaea177afdbf337f03397bb811ab59db142"
		testSize        = 100
		testVirtualSize = 200
	)

	notAvailableImage := func() *images.Image {
		return &images.Image{
			ID:               imageID,
			Name:             "test-image",
			Status:           images.ImageStatusQueued,
			Tags:             []string{"foo", "bar"},
			ContainerFormat:  "bare",
			DiskFormat:       "qcow2",
			MinDiskGigabytes: 1,
			MinRAMMegabytes:  2,
			Protected:        true,
			Visibility:       images.ImageVisibilityPublic,
			Hidden:           true,
			Properties: map[string]interface{}{
				"hw_cpu_cores": 2,
			},
		}
	}

	availableImage := func() *images.Image {
		return &images.Image{
			ID:               imageID,
			Name:             "test-image",
			Status:           images.ImageStatusActive,
			Tags:             []string{"foo", "bar"},
			ContainerFormat:  "bare",
			DiskFormat:       "qcow2",
			MinDiskGigabytes: 1,
			MinRAMMegabytes:  2,
			Protected:        true,
			Visibility:       images.ImageVisibilityPublic,
			Hidden:           true,
			SizeBytes:        testSize,
			Properties: map[string]interface{}{
				"hw_cpu_cores":  2,
				"os_hash_algo":  "sha512",
				"os_hash_value": testSHA512value,
			},
			VirtualSize: testVirtualSize,
		}
	}

	type args struct {
		orcImage    func(*orcv1alpha1.Image, metav1.Time)
		glanceImage func() *images.Image
		err         error
		opts        []updateStatusOpt
	}
	tests := []struct {
		name            string
		args            args
		wantStatus      func() *orcapplyconfigv1alpha1.ImageStatusApplyConfiguration
		wantAvailable   func(metav1.Time) *applyconfigv1.ConditionApplyConfiguration
		wantProgressing func(metav1.Time) *applyconfigv1.ConditionApplyConfiguration
		isComplete      bool
	}{
		{
			name:       "No image, no status, no error",
			args:       args{},
			wantStatus: orcapplyconfigv1alpha1.ImageStatus,
			wantAvailable: func(now metav1.Time) *applyconfigv1.ConditionApplyConfiguration {
				return applyconfigv1.Condition().
					WithStatus(metav1.ConditionFalse).
					WithReason(orcv1alpha1.OpenStackConditionReasonProgressing).
					WithMessage(progressingMsg).
					WithLastTransitionTime(now)
			},
			wantProgressing: func(now metav1.Time) *applyconfigv1.ConditionApplyConfiguration {
				return applyconfigv1.Condition().
					WithStatus(metav1.ConditionTrue).
					WithReason(orcv1alpha1.OpenStackConditionReasonProgressing).
					WithMessage(progressingMsg).
					WithLastTransitionTime(now)
			},
			isComplete: false,
		},
		{
			name: "No image, no status, transient error",
			args: args{
				err: fmt.Errorf("test-error"),
			},
			wantStatus: orcapplyconfigv1alpha1.ImageStatus,
			wantAvailable: func(now metav1.Time) *applyconfigv1.ConditionApplyConfiguration {
				return applyconfigv1.Condition().
					WithStatus(metav1.ConditionFalse).
					WithReason(orcv1alpha1.OpenStackConditionReasonTransientError).
					WithMessage("test-error").
					WithLastTransitionTime(now)
			},
			wantProgressing: func(now metav1.Time) *applyconfigv1.ConditionApplyConfiguration {
				return applyconfigv1.Condition().
					WithStatus(metav1.ConditionFalse).
					WithReason(orcv1alpha1.OpenStackConditionReasonTransientError).
					WithMessage("test-error").
					WithLastTransitionTime(now)
			},
			isComplete: false,
		},
		{
			name: "No image, no status, fatal error",
			args: args{
				err: orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, "invalid configuration", fmt.Errorf("test-error")),
			},
			wantStatus: orcapplyconfigv1alpha1.ImageStatus,
			wantAvailable: func(now metav1.Time) *applyconfigv1.ConditionApplyConfiguration {
				return applyconfigv1.Condition().
					WithStatus(metav1.ConditionFalse).
					WithReason(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration).
					WithMessage("invalid configuration").
					WithLastTransitionTime(now)
			},
			wantProgressing: func(now metav1.Time) *applyconfigv1.ConditionApplyConfiguration {
				return applyconfigv1.Condition().
					WithStatus(metav1.ConditionFalse).
					WithReason(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration).
					WithMessage("invalid configuration").
					WithLastTransitionTime(now)
			},
			isComplete: true,
		},
		{
			name: "Image (not available), no status, no error",
			args: args{
				glanceImage: notAvailableImage,
			},
			wantStatus: func() *orcapplyconfigv1alpha1.ImageStatusApplyConfiguration {
				return orcapplyconfigv1alpha1.ImageStatus().
					WithResource(orcapplyconfigv1alpha1.ImageResourceStatus().
						WithStatus(string(images.ImageStatusQueued)))
			},
			wantAvailable: func(now metav1.Time) *applyconfigv1.ConditionApplyConfiguration {
				return applyconfigv1.Condition().
					WithStatus(metav1.ConditionFalse).
					WithReason(orcv1alpha1.OpenStackConditionReasonProgressing).
					WithMessage(progressingMsg).
					WithLastTransitionTime(now)
			},
			wantProgressing: func(now metav1.Time) *applyconfigv1.ConditionApplyConfiguration {
				return applyconfigv1.Condition().
					WithStatus(metav1.ConditionTrue).
					WithReason(orcv1alpha1.OpenStackConditionReasonProgressing).
					WithMessage(progressingMsg).
					WithLastTransitionTime(now)
			},
			isComplete: false,
		},
		{
			name: "Image (available), no status, no error",
			args: args{
				glanceImage: availableImage,
			},
			wantStatus: func() *orcapplyconfigv1alpha1.ImageStatusApplyConfiguration {
				return orcapplyconfigv1alpha1.ImageStatus().
					WithResource(orcapplyconfigv1alpha1.ImageResourceStatus().
						WithStatus(string(images.ImageStatusActive)).
						WithSizeB(testSize).
						WithVirtualSizeB(testVirtualSize).
						WithHash(orcapplyconfigv1alpha1.ImageHash().
							WithAlgorithm("sha512").
							WithValue(testSHA512value)))
			},
			wantAvailable: func(now metav1.Time) *applyconfigv1.ConditionApplyConfiguration {
				return applyconfigv1.Condition().
					WithStatus(metav1.ConditionTrue).
					WithReason(orcv1alpha1.OpenStackConditionReasonSuccess).
					WithMessage(successMsg).
					WithLastTransitionTime(now)
			},
			wantProgressing: func(now metav1.Time) *applyconfigv1.ConditionApplyConfiguration {
				return applyconfigv1.Condition().
					WithStatus(metav1.ConditionFalse).
					WithReason(orcv1alpha1.OpenStackConditionReasonSuccess).
					WithMessage(successMsg).
					WithLastTransitionTime(now)
			},
			isComplete: true,
		},
		{
			name: "Image (available), no status, no error",
			args: args{
				glanceImage: availableImage,
			},
			wantStatus: func() *orcapplyconfigv1alpha1.ImageStatusApplyConfiguration {
				return orcapplyconfigv1alpha1.ImageStatus().
					WithResource(orcapplyconfigv1alpha1.ImageResourceStatus().
						WithStatus(string(images.ImageStatusActive)).
						WithSizeB(testSize).
						WithVirtualSizeB(testVirtualSize).
						WithHash(orcapplyconfigv1alpha1.ImageHash().
							WithAlgorithm("sha512").
							WithValue(testSHA512value)))
			},
			wantAvailable: func(now metav1.Time) *applyconfigv1.ConditionApplyConfiguration {
				return applyconfigv1.Condition().
					WithStatus(metav1.ConditionTrue).
					WithReason(orcv1alpha1.OpenStackConditionReasonSuccess).
					WithMessage(successMsg).
					WithLastTransitionTime(now)
			},
			wantProgressing: func(now metav1.Time) *applyconfigv1.ConditionApplyConfiguration {
				return applyconfigv1.Condition().
					WithStatus(metav1.ConditionFalse).
					WithReason(orcv1alpha1.OpenStackConditionReasonSuccess).
					WithMessage(successMsg).
					WithLastTransitionTime(now)
			},
			isComplete: true,
		},
		{
			name: "Image (available), status previously available, no error",
			args: args{
				glanceImage: availableImage,
				orcImage: func(orcImage *orcv1alpha1.Image, now metav1.Time) {
					hourAgo := metav1.NewTime(now.Add(-time.Hour))

					orcImage.Status.Conditions = []metav1.Condition{
						{
							Type:               orcv1alpha1.OpenStackConditionAvailable,
							Status:             metav1.ConditionTrue,
							ObservedGeneration: 1,
							LastTransitionTime: hourAgo,
							Reason:             orcv1alpha1.OpenStackConditionReasonSuccess,
							Message:            successMsg,
						},
						{
							Type:               orcv1alpha1.OpenStackConditionProgressing,
							Status:             metav1.ConditionFalse,
							ObservedGeneration: 1,
							LastTransitionTime: hourAgo,
							Reason:             orcv1alpha1.OpenStackConditionReasonSuccess,
							Message:            successMsg,
						},
					}
				},
			},
			wantStatus: func() *orcapplyconfigv1alpha1.ImageStatusApplyConfiguration {
				return orcapplyconfigv1alpha1.ImageStatus().
					WithResource(orcapplyconfigv1alpha1.ImageResourceStatus().
						WithStatus(string(images.ImageStatusActive)).
						WithSizeB(testSize).
						WithVirtualSizeB(testVirtualSize).
						WithHash(orcapplyconfigv1alpha1.ImageHash().
							WithAlgorithm("sha512").
							WithValue(testSHA512value)))
			},
			wantAvailable: func(now metav1.Time) *applyconfigv1.ConditionApplyConfiguration {
				// LastTransitionTime should be copied from previous condition
				hourAgo := metav1.NewTime(now.Add(-time.Hour))
				return applyconfigv1.Condition().
					WithStatus(metav1.ConditionTrue).
					WithReason(orcv1alpha1.OpenStackConditionReasonSuccess).
					WithMessage(successMsg).
					WithLastTransitionTime(hourAgo)
			},
			wantProgressing: func(now metav1.Time) *applyconfigv1.ConditionApplyConfiguration {
				hourAgo := metav1.NewTime(now.Add(-time.Hour))
				return applyconfigv1.Condition().
					WithStatus(metav1.ConditionFalse).
					WithReason(orcv1alpha1.OpenStackConditionReasonSuccess).
					WithMessage(successMsg).
					WithLastTransitionTime(hourAgo)
			},
			isComplete: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)
			ctx := context.TODO()

			now := metav1.NewTime(time.Now())

			orcImage := &orcv1alpha1.Image{}
			orcImage.SetName("test-image")
			orcImage.SetGeneration(1)
			if tt.args.orcImage != nil {
				tt.args.orcImage(orcImage, now)
			}

			wantAvailable := tt.wantAvailable(now).
				WithType(orcv1alpha1.OpenStackConditionAvailable).
				WithObservedGeneration(1)
			wantProgressing := tt.wantProgressing(now).
				WithType(orcv1alpha1.OpenStackConditionProgressing).
				WithObservedGeneration(1)
			wantStatusUpdate := orcapplyconfigv1alpha1.Image(orcImage.Name, orcImage.Namespace).
				WithStatus(tt.wantStatus().WithConditions(wantAvailable, wantProgressing))

			var glanceImage *images.Image
			if tt.args.glanceImage != nil {
				glanceImage = tt.args.glanceImage()
			}

			opts := append(tt.args.opts, withGlanceImage(glanceImage), withError(tt.args.err))

			// TODO: Consider rewriting to this to test
			// updateStatus() using fake client when we have
			// https://github.com/kubernetes/kubernetes/pull/125560
			statusUpdate := createStatusUpdate(ctx, orcImage, now, opts...)

			g.Expect(statusUpdate).To(gomega.Equal(wantStatusUpdate), cmp.Diff(statusUpdate, wantStatusUpdate))

			// This would be better done with fake client. As we
			// can't use that, we take advantage of the fact that an
			// apply configuration marshals identically to the
			// object it's based on. We marshal it to json, then
			// unmarshal it over the original object.
			b, err := json.Marshal(statusUpdate)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(json.Unmarshal(b, orcImage)).To(gomega.Succeed())

			g.Expect(orcv1alpha1.IsReconciliationComplete(orcImage)).To(gomega.Equal(tt.isComplete), "isComplete")
		})
	}
}
