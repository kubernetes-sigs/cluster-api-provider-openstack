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
	"fmt"
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/golang/mock/gomock"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/servergroups"
	"k8s.io/utils/ptr"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

func TestService_GetServerGroupID(t *testing.T) {
	const serverGroupID1 = "ce96e584-7ebc-46d6-9e55-987d72e3806c"
	const serverGroupID2 = "8f536889-5198-42d7-8314-cb78f4f4755c"

	tests := []struct {
		testName         string
		serverGroupParam *infrav1.ServerGroupParam
		expect           func(m *mock.MockComputeClientMockRecorder)
		want             string
		wantErr          bool
	}{
		{
			testName:         "Return server group ID from filter if only filter (with ID) given",
			serverGroupParam: &infrav1.ServerGroupParam{ID: ptr.To(serverGroupID1)},
			expect: func(m *mock.MockComputeClientMockRecorder) {
			},
			want:    serverGroupID1,
			wantErr: false,
		},
		{
			testName:         "Return error if empty filter is given",
			serverGroupParam: &infrav1.ServerGroupParam{},
			expect: func(m *mock.MockComputeClientMockRecorder) {
			},
			want:    "",
			wantErr: true,
		},
		{
			testName:         "Return server group ID from filter if only filter (with name) given",
			serverGroupParam: &infrav1.ServerGroupParam{Filter: &infrav1.ServerGroupFilter{Name: ptr.To("test-server-group")}},
			expect: func(m *mock.MockComputeClientMockRecorder) {
				m.ListServerGroups().Return(
					[]servergroups.ServerGroup{{ID: serverGroupID1, Name: "test-server-group"}},
					nil)
			},
			want:    serverGroupID1,
			wantErr: false,
		},
		{
			testName:         "Return no results",
			serverGroupParam: &infrav1.ServerGroupParam{Filter: &infrav1.ServerGroupFilter{Name: ptr.To("test-server-group")}},
			expect: func(m *mock.MockComputeClientMockRecorder) {
				m.ListServerGroups().Return(
					[]servergroups.ServerGroup{},
					nil)
			},
			want:    "",
			wantErr: true,
		},
		{
			testName:         "Return multiple results",
			serverGroupParam: &infrav1.ServerGroupParam{Filter: &infrav1.ServerGroupFilter{Name: ptr.To("test-server-group")}},
			expect: func(m *mock.MockComputeClientMockRecorder) {
				m.ListServerGroups().Return(
					[]servergroups.ServerGroup{
						{ID: serverGroupID1, Name: "test-server-group"},
						{ID: serverGroupID2, Name: "test-server-group"},
					},
					nil)
			},
			want:    "",
			wantErr: true,
		},
		{
			testName:         "OpenStack returns error",
			serverGroupParam: &infrav1.ServerGroupParam{Filter: &infrav1.ServerGroupFilter{Name: ptr.To("test-server-group")}},
			expect: func(m *mock.MockComputeClientMockRecorder) {
				m.ListServerGroups().Return(
					nil,
					fmt.Errorf("test error"))
			},
			want:    "",
			wantErr: true,
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
			tt.expect(mockScopeFactory.ComputeClient.EXPECT())

			got, err := s.GetServerGroupID(tt.serverGroupParam)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.getServerGroupID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Service.getServerGroupID() = %v, want %v", got, tt.want)
			}
		})
	}
}
