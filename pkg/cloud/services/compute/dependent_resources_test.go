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

	"github.com/go-logr/logr/testr"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

func Test_ResolveDependentMachineResources(t *testing.T) {
	const networkID = "5e8e0d3b-7f3d-4f3e-8b3f-3e3e3e3e3e3e"
	const portID = "78e0d3b-7f3d-4f3e-8b3f-3e3e3e3e3e3e"

	tests := []struct {
		testName               string
		openStackCluster       *infrav1.OpenStackCluster
		openStackMachineStatus infrav1.OpenStackMachineStatus
		want                   *infrav1.DependentMachineResources
		wantErr                bool
	}{
		{
			testName:         "no Network ID yet and no ports in status",
			openStackCluster: &infrav1.OpenStackCluster{},
			want:             &infrav1.DependentMachineResources{},
			wantErr:          false,
		},
		{
			testName: "Network ID set but no ports in status",
			openStackCluster: &infrav1.OpenStackCluster{
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						NetworkStatus: infrav1.NetworkStatus{
							ID: networkID,
						},
					},
				},
			},
			want:    &infrav1.DependentMachineResources{},
			wantErr: false,
		},
		{
			testName: "Network ID set and ports in status",
			openStackCluster: &infrav1.OpenStackCluster{
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						NetworkStatus: infrav1.NetworkStatus{
							ID: networkID,
						},
					},
				},
			},
			openStackMachineStatus: infrav1.OpenStackMachineStatus{
				DependentResources: infrav1.DependentMachineResources{
					PortsStatus: []infrav1.PortStatus{
						{
							ID: portID,
						},
					},
				},
			},
			want: &infrav1.DependentMachineResources{
				PortsStatus: []infrav1.PortStatus{
					{
						ID: portID,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			g := NewWithT(t)
			log := testr.New(t)
			mockCtrl := gomock.NewController(t)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")

			defaultOpenStackMachine := &infrav1.OpenStackMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Status: tt.openStackMachineStatus,
			}

			_, err := ResolveDependentMachineResources(scope.NewWithLogger(mockScopeFactory, log), defaultOpenStackMachine)
			if tt.wantErr {
				g.Expect(err).Error()
				return
			}

			g.Expect(&defaultOpenStackMachine.Status.DependentResources).To(Equal(tt.want), cmp.Diff(&defaultOpenStackMachine.Status.DependentResources, tt.want))
		})
	}
}

func TestResolveDependentBastionResources(t *testing.T) {
	const networkID = "5e8e0d3b-7f3d-4f3e-8b3f-3e3e3e3e3e3e"
	const portID = "78e0d3b-7f3d-4f3e-8b3f-3e3e3e3e3e3e"
	const bastionName = "bastion"

	tests := []struct {
		testName         string
		openStackCluster *infrav1.OpenStackCluster
		want             *infrav1.DependentMachineResources
		wantErr          bool
	}{
		{
			testName: "no Network ID yet and no ports in status",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					Bastion: &infrav1.Bastion{
						Enabled: true,
					},
				},
			},
			want:    &infrav1.DependentMachineResources{},
			wantErr: false,
		},
		{
			testName: "Network ID set but no ports in status",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					Bastion: &infrav1.Bastion{
						Enabled: true,
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Network: &infrav1.NetworkStatusWithSubnets{
						NetworkStatus: infrav1.NetworkStatus{
							ID: networkID,
						},
					},
				},
			},
			want: &infrav1.DependentMachineResources{},
		},
		{
			testName: "Network ID set and ports in status",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					Bastion: &infrav1.Bastion{
						Enabled: true,
					},
				},
				Status: infrav1.OpenStackClusterStatus{
					Bastion: &infrav1.BastionStatus{
						DependentResources: infrav1.DependentMachineResources{
							PortsStatus: []infrav1.PortStatus{
								{
									ID: portID,
								},
							},
						},
					},
					Network: &infrav1.NetworkStatusWithSubnets{
						NetworkStatus: infrav1.NetworkStatus{
							ID: networkID,
						},
					},
				},
			},
			want: &infrav1.DependentMachineResources{
				PortsStatus: []infrav1.PortStatus{
					{
						ID: portID,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			g := NewWithT(t)
			log := testr.New(t)
			mockCtrl := gomock.NewController(t)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")

			_, err := ResolveDependentBastionResources(scope.NewWithLogger(mockScopeFactory, log), tt.openStackCluster, bastionName)
			if tt.wantErr {
				g.Expect(err).Error()
				return
			}

			defaultOpenStackCluster := &infrav1.OpenStackCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec:   tt.openStackCluster.Spec,
				Status: tt.openStackCluster.Status,
			}

			if tt.openStackCluster.Status.Bastion != nil {
				g.Expect(&defaultOpenStackCluster.Status.Bastion.DependentResources).To(Equal(tt.want), cmp.Diff(&defaultOpenStackCluster.Status.Bastion.DependentResources, tt.want))
			}
		})
	}
}
