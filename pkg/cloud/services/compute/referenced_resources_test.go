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
	"reflect"
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

func Test_getInstanceTags(t *testing.T) {
	tests := []struct {
		name             string
		spec             func() *infrav1.OpenStackMachineSpec
		openStackCluster func() *infrav1.OpenStackCluster
		wantMachineTags  []string
	}{
		{
			name: "No tags",
			spec: func() *infrav1.OpenStackMachineSpec {
				return &infrav1.OpenStackMachineSpec{}
			},
			openStackCluster: func() *infrav1.OpenStackCluster {
				return &infrav1.OpenStackCluster{
					Spec: infrav1.OpenStackClusterSpec{},
				}
			},
			wantMachineTags: []string{},
		},
		{
			name: "Machine tags only",
			spec: func() *infrav1.OpenStackMachineSpec {
				return &infrav1.OpenStackMachineSpec{
					Tags: []string{"machine-tag1", "machine-tag2"},
				}
			},
			openStackCluster: func() *infrav1.OpenStackCluster {
				return &infrav1.OpenStackCluster{
					Spec: infrav1.OpenStackClusterSpec{},
				}
			},
			wantMachineTags: []string{"machine-tag1", "machine-tag2"},
		},
		{
			name: "Cluster tags only",
			spec: func() *infrav1.OpenStackMachineSpec {
				return &infrav1.OpenStackMachineSpec{}
			},
			openStackCluster: func() *infrav1.OpenStackCluster {
				return &infrav1.OpenStackCluster{
					Spec: infrav1.OpenStackClusterSpec{
						Tags: []string{"cluster-tag1", "cluster-tag2"},
					},
				}
			},
			wantMachineTags: []string{"cluster-tag1", "cluster-tag2"},
		},
		{
			name: "Machine and cluster tags",
			spec: func() *infrav1.OpenStackMachineSpec {
				return &infrav1.OpenStackMachineSpec{
					Tags: []string{"machine-tag1", "machine-tag2"},
				}
			},
			openStackCluster: func() *infrav1.OpenStackCluster {
				return &infrav1.OpenStackCluster{
					Spec: infrav1.OpenStackClusterSpec{
						Tags: []string{"cluster-tag1", "cluster-tag2"},
					},
				}
			},
			wantMachineTags: []string{"machine-tag1", "machine-tag2", "cluster-tag1", "cluster-tag2"},
		},
		{
			name: "Duplicate tags",
			spec: func() *infrav1.OpenStackMachineSpec {
				return &infrav1.OpenStackMachineSpec{
					Tags: []string{"tag1", "tag2", "tag1"},
				}
			},
			openStackCluster: func() *infrav1.OpenStackCluster {
				return &infrav1.OpenStackCluster{
					Spec: infrav1.OpenStackClusterSpec{
						Tags: []string{"tag2", "tag3", "tag3"},
					},
				}
			},
			wantMachineTags: []string{"tag1", "tag2", "tag3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMachineTags := InstanceTags(tt.spec(), tt.openStackCluster())
			if !reflect.DeepEqual(gotMachineTags, tt.wantMachineTags) {
				t.Errorf("getInstanceTags() = %v, want %v", gotMachineTags, tt.wantMachineTags)
			}
		})
	}
}

func newFlavorTestService(t *testing.T, mockCtrl *gomock.Controller) (*Service, *scope.MockScopeFactory) {
	t.Helper()
	log := testr.New(t)
	mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
	svc, err := NewService(scope.NewWithLogger(mockScopeFactory, log))
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc, mockScopeFactory
}

// --- ID path ---

func TestGetFlavorID_ByID_NoLookup(t *testing.T) {
	g := NewWithT(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	svc, mockScopeFactory := newFlavorTestService(t, mockCtrl)

	// ListFlavors must NOT be called when an ID is provided directly.
	mockScopeFactory.ComputeClient.EXPECT().ListFlavors().Times(0)

	id, err := svc.GetFlavorID(infrav1.FlavorParam{
		ID: ptr.To("flavor-uuid-direct"),
	})

	// GetFlavorID returns *string — use HaveValue to unwrap.
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(id).To(HaveValue(Equal("flavor-uuid-direct")))
}

// --- Filter path: success ---

func TestGetFlavorID_ByFilter_Name_ExactlyOne(t *testing.T) {
	g := NewWithT(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	svc, mockScopeFactory := newFlavorTestService(t, mockCtrl)
	mockScopeFactory.ComputeClient.EXPECT().ListFlavors().Return([]flavors.Flavor{
		{ID: "aaa-111", Name: "m1.small"},
	}, nil)

	id, err := svc.GetFlavorID(infrav1.FlavorParam{
		Filter: &infrav1.FlavorFilter{
			Name: ptr.To("m1.small"),
		},
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(id).To(HaveValue(Equal("aaa-111")))
}

// --- Filter path: error cases ---

func TestGetFlavorID_ByFilter_NoResults(t *testing.T) {
	g := NewWithT(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	svc, mockScopeFactory := newFlavorTestService(t, mockCtrl)
	mockScopeFactory.ComputeClient.EXPECT().ListFlavors().Return([]flavors.Flavor{}, nil)

	_, err := svc.GetFlavorID(infrav1.FlavorParam{
		Filter: &infrav1.FlavorFilter{
			Name: ptr.To("nonexistent-flavor"),
		},
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("nonexistent-flavor"))
}

func TestGetFlavorID_ByFilter_ListError(t *testing.T) {
	g := NewWithT(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	svc, mockScopeFactory := newFlavorTestService(t, mockCtrl)
	mockScopeFactory.ComputeClient.EXPECT().ListFlavors().Return(
		nil, fmt.Errorf("nova unavailable"),
	)

	_, err := svc.GetFlavorID(infrav1.FlavorParam{
		Filter: &infrav1.FlavorFilter{
			Name: ptr.To("m1.small"),
		},
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("nova unavailable"))
}
