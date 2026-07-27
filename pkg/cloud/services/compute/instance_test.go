/*
Copyright 2018 The Kubernetes Authors.

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
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
)

func TestService_getImageID(t *testing.T) {
	const (
		imageID   = "ce96e584-7ebc-46d6-9e55-987d72e3806c"
		imageName = "test-image"
		namespace = "test-namespace"
	)
	imageTags := []string{"test-tag"}

	scheme := runtime.NewScheme()
	if err := orcv1alpha1.AddToScheme(scheme); err != nil {
		panic(err)
	}

	tests := []struct {
		testName          string
		image             infrav1.ImageParam
		fakeObjects       []runtime.Object
		expect            func(m *mock.MockImageClientMockRecorder)
		want              *string
		wantErr           bool
		wantTerminalError bool
	}{
		{
			testName: "Return image ID when ID given",
			image:    infrav1.ImageParam{ID: ptr.To(imageID)},
			want:     ptr.To(imageID),
			expect:   func(*mock.MockImageClientMockRecorder) {},
			wantErr:  false,
		},
		{
			testName: "Return image ID when name given",
			image: infrav1.ImageParam{
				Filter: &infrav1.ImageFilter{
					Name: ptr.To(imageName),
				},
			},
			want: ptr.To(imageID),
			expect: func(m *mock.MockImageClientMockRecorder) {
				m.ListImages(images.ListOpts{Name: imageName}).Return(
					[]images.Image{{ID: imageID, Name: imageName}},
					nil)
			},
			wantErr: false,
		},
		{
			testName: "Return image ID when tags given",
			image: infrav1.ImageParam{
				Filter: &infrav1.ImageFilter{
					Tags: imageTags,
				},
			},
			want: ptr.To(imageID),
			expect: func(m *mock.MockImageClientMockRecorder) {
				m.ListImages(images.ListOpts{Tags: imageTags}).Return(
					[]images.Image{{ID: imageID, Name: imageName, Tags: imageTags}},
					nil)
			},
			wantErr: false,
		},
		{
			testName: "Return no results",
			image: infrav1.ImageParam{
				Filter: &infrav1.ImageFilter{
					Name: ptr.To(imageName),
				},
			},
			expect: func(m *mock.MockImageClientMockRecorder) {
				m.ListImages(images.ListOpts{Name: imageName}).Return(
					[]images.Image{},
					nil)
			},
			want:    nil,
			wantErr: true,
		},
		{
			testName: "Return multiple results",
			image: infrav1.ImageParam{
				Filter: &infrav1.ImageFilter{
					Name: ptr.To(imageName),
				},
			},
			expect: func(m *mock.MockImageClientMockRecorder) {
				m.ListImages(images.ListOpts{Name: "test-image"}).Return(
					[]images.Image{
						{ID: imageID, Name: "test-image"},
						{ID: "123", Name: "test-image"},
					}, nil)
			},
			want:    nil,
			wantErr: true,
		},
		{
			testName: "OpenStack returns error",
			image: infrav1.ImageParam{
				Filter: &infrav1.ImageFilter{
					Name: ptr.To(imageName),
				},
			},
			expect: func(m *mock.MockImageClientMockRecorder) {
				m.ListImages(images.ListOpts{Name: "test-image"}).Return(
					nil,
					fmt.Errorf("test error"))
			},
			want:    nil,
			wantErr: true,
		},
		{
			testName: "Image by reference does not exist",
			image: infrav1.ImageParam{
				ImageRef: &infrav1.ResourceReference{
					Name: imageName,
				},
			},
			want: nil,
		},
		{
			testName: "Image by reference exists, is available",
			image: infrav1.ImageParam{
				ImageRef: &infrav1.ResourceReference{
					Name: imageName,
				},
			},
			fakeObjects: []runtime.Object{
				&orcv1alpha1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      imageName,
						Namespace: namespace,
					},
					Status: orcv1alpha1.ImageStatus{
						Conditions: []metav1.Condition{
							{
								Type:   orcv1alpha1.ConditionAvailable,
								Status: metav1.ConditionTrue,
							},
						},
						ID: ptr.To(imageID),
					},
				},
			},
			want: ptr.To(imageID),
		},
		{
			testName: "Image by reference exists, still reconciling",
			image: infrav1.ImageParam{
				ImageRef: &infrav1.ResourceReference{
					Name: imageName,
				},
			},
			fakeObjects: []runtime.Object{
				&orcv1alpha1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      imageName,
						Namespace: namespace,
					},
					Status: orcv1alpha1.ImageStatus{
						Conditions: []metav1.Condition{
							{
								Type:   orcv1alpha1.ConditionAvailable,
								Status: metav1.ConditionFalse,
							},
							{
								Type:   orcv1alpha1.ConditionProgressing,
								Status: metav1.ConditionTrue,
								Reason: orcv1alpha1.ConditionReasonProgressing,
							},
						},
						ID: ptr.To(imageID),
					},
				},
			},
			want: nil,
		},
		{
			testName: "Image by reference exists, terminal failure",
			image: infrav1.ImageParam{
				ImageRef: &infrav1.ResourceReference{
					Name: imageName,
				},
			},
			fakeObjects: []runtime.Object{
				&orcv1alpha1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      imageName,
						Namespace: namespace,
					},
					Status: orcv1alpha1.ImageStatus{
						Conditions: []metav1.Condition{
							{
								Type:   orcv1alpha1.ConditionAvailable,
								Status: metav1.ConditionFalse,
							},
							{
								Type:    orcv1alpha1.ConditionProgressing,
								Status:  metav1.ConditionFalse,
								Reason:  orcv1alpha1.ConditionReasonUnrecoverableError,
								Message: "test error",
							},
						},
						ID: ptr.To(imageID),
					},
				},
			},
			want:              nil,
			wantTerminalError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			log := testr.New(t)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")

			s, err := NewService(scope.NewWithLogger(mockScopeFactory, log))
			if err != nil {
				t.Fatalf("Failed to create service: %v", err)
			}
			if tt.expect != nil {
				tt.expect(mockScopeFactory.ImageClient.EXPECT())
			}

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.fakeObjects...).Build()

			got, err := s.GetImageID(context.TODO(), fakeClient, namespace, tt.image)

			if tt.wantTerminalError {
				tt.wantErr = true
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("Service.getImageID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			var terminalError *capoerrors.TerminalError
			if errors.As(err, &terminalError) != tt.wantTerminalError {
				t.Errorf("Terminal error: wanted = %v, got = %v", tt.wantTerminalError, !tt.wantTerminalError)
			}

			// NOTE(mdbooth): there must be a simpler way to write this!
			if (tt.want == nil && got != nil) || (tt.want != nil && (got == nil || *tt.want != *got)) {
				t.Errorf("Service.getImageID() = '%v', want '%v'", ptr.Deref(got, ""), ptr.Deref(tt.want, ""))
			}
		})
	}
}
