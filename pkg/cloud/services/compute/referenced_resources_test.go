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

package compute

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/servergroups"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	. "github.com/onsi/gomega"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha8"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

func Test_ResolveReferencedMachineResources(t *testing.T) {
	const serverGroupID1 = "ce96e584-7ebc-46d6-9e55-987d72e3806c"
	const imageID1 = "de96e584-7ebc-46d6-9e55-987d72e3806c"

	minimumReferences := &infrav1.ReferencedMachineResources{
		ImageID: imageID1,
	}

	tests := []struct {
		testName          string
		serverGroupFilter *infrav1.ServerGroupFilter
		imageFilter       *infrav1.ImageFilter
		expectComputeMock func(m *mock.MockComputeClientMockRecorder)
		expectImageMock   func(m *mock.MockImageClientMockRecorder)
		want              *infrav1.ReferencedMachineResources
		wantErr           bool
	}{
		{
			testName:          "Resources ID passed",
			serverGroupFilter: &infrav1.ServerGroupFilter{ID: serverGroupID1},
			imageFilter:       &infrav1.ImageFilter{ID: imageID1},
			expectComputeMock: func(m *mock.MockComputeClientMockRecorder) {},
			expectImageMock:   func(m *mock.MockImageClientMockRecorder) {},
			want:              &infrav1.ReferencedMachineResources{ImageID: imageID1, ServerGroupID: serverGroupID1},
			wantErr:           false,
		},
		{
			testName:          "Server group filter nil",
			serverGroupFilter: nil,
			expectComputeMock: func(m *mock.MockComputeClientMockRecorder) {},
			expectImageMock:   func(m *mock.MockImageClientMockRecorder) {},
			want:              minimumReferences,
			wantErr:           false,
		},
		{
			testName:          "Server group ID empty",
			serverGroupFilter: &infrav1.ServerGroupFilter{},
			expectComputeMock: func(m *mock.MockComputeClientMockRecorder) {},
			expectImageMock:   func(m *mock.MockImageClientMockRecorder) {},
			want:              minimumReferences,
			wantErr:           false,
		},
		{
			testName:          "Server group by Name not found",
			serverGroupFilter: &infrav1.ServerGroupFilter{Name: "test-server-group"},
			expectComputeMock: func(m *mock.MockComputeClientMockRecorder) {
				m.ListServerGroups().Return(
					[]servergroups.ServerGroup{},
					nil)
			},
			expectImageMock: func(m *mock.MockImageClientMockRecorder) {},
			want:            &infrav1.ReferencedMachineResources{},
			wantErr:         true,
		},
		{
			testName:          "Image by Name not found",
			imageFilter:       &infrav1.ImageFilter{Name: "test-image"},
			expectComputeMock: func(m *mock.MockComputeClientMockRecorder) {},
			expectImageMock: func(m *mock.MockImageClientMockRecorder) {
				m.ListImages(images.ListOpts{Name: "test-image"}).Return(
					[]images.Image{},
					nil)
			},
			want:    &infrav1.ReferencedMachineResources{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			g := NewWithT(t)
			mockCtrl := gomock.NewController(t)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "", logr.Discard())

			tt.expectComputeMock(mockScopeFactory.ComputeClient.EXPECT())
			tt.expectImageMock(mockScopeFactory.ImageClient.EXPECT())

			// Set defaults for required fields
			imageFilter := &infrav1.ImageFilter{ID: imageID1}
			if tt.imageFilter != nil {
				imageFilter = tt.imageFilter
			}

			machineSpec := &infrav1.OpenStackMachineSpec{
				ServerGroup: tt.serverGroupFilter,
				Image:       *imageFilter,
			}

			resources := &infrav1.ReferencedMachineResources{}

			err := ResolveReferencedMachineResources(mockScopeFactory, machineSpec, resources)
			if tt.wantErr {
				g.Expect(err).Error()
				return
			}

			g.Expect(resources).To(Equal(tt.want), cmp.Diff(resources, tt.want))
		})
	}
}
