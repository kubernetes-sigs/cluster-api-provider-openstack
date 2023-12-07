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

package networking

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"

	. "github.com/onsi/gomega"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha7"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

func Test_ReconcileSecurityGroups(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	clusterName := "test-cluster"
	expectedControlPlaneSecurityGroupName := getSecControlPlaneGroupName(clusterName)
	expectedWorkerSecurityGroupName := getSecWorkerGroupName(clusterName)
	expectedBastionSecurityGroupName := getSecBastionGroupName(clusterName)
	fakeControlPlaneSecurityGroupID := "d08803fc-2fa5-4179-b9f7-8c43d0af2fe6"
	fakeWorkerSecurityGroupID := "619e34ec-b7ce-4a07-a0f4-450322c7ba40"
	fakeBastionSecurityGroupID := "91beedf0-1020-4443-9cbd-95412f527245"

	tests := []struct {
		name             string
		openStackCluster *infrav1.OpenStackCluster
		expect           func(m *mock.MockNetworkClientMockRecorder)
		want             *infrav1.OpenStackCluster
		wantError        bool
	}{
		{
			name: "ensure nothing to reconcile without security group",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {},
			want: &infrav1.OpenStackCluster{
				Spec:   infrav1.OpenStackClusterSpec{},
				Status: infrav1.OpenStackClusterStatus{},
			},
			wantError: false,
		},
		{
			name: "ensures status set when reconciling legacy managed security groups",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ManagedSecurityGroups: true,
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListSecGroup(gomock.Any()).Return([]groups.SecGroup{}, nil).AnyTimes()

				m.
					CreateSecGroup(groups.CreateOpts{
						Name:        expectedControlPlaneSecurityGroupName,
						Description: "Cluster API managed group",
					}).
					Return(&groups.SecGroup{
						Name:        expectedControlPlaneSecurityGroupName,
						Description: "Cluster API managed group",
					}, nil)

				m.
					CreateSecGroup(groups.CreateOpts{
						Name:        expectedWorkerSecurityGroupName,
						Description: "Cluster API managed group",
					}).
					Return(&groups.SecGroup{
						Name:        expectedWorkerSecurityGroupName,
						Description: "Cluster API managed group",
					}, nil)
			},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{},
				Status: infrav1.OpenStackClusterStatus{
					ControlPlaneSecurityGroup: &infrav1.SecurityGroup{
						ID:   fakeControlPlaneSecurityGroupID,
						Name: expectedControlPlaneSecurityGroupName,
					},
					WorkerSecurityGroup: &infrav1.SecurityGroup{
						Name: expectedWorkerSecurityGroupName,
					},
				},
			},
			wantError: false,
		},
		{
			name: "ensures status set when reconciling security groups",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					SecurityGroups: &infrav1.SecurityGroupsSpec{
						AdditionalControlPlaneSecurityGroupRules: []infrav1.SecurityGroupRule{
							{
								Description:   "Etcd",
								Direction:     "ingress",
								EtherType:     "IPv4",
								PortRangeMin:  2379,
								PortRangeMax:  2380,
								Protocol:      "tcp",
								RemoteGroupID: "",
							},
						},
						AdditionalWorkerSecurityGroupRules: []infrav1.SecurityGroupRule{
							{
								Description:   "ssh",
								Direction:     "ingress",
								EtherType:     "IPv4",
								PortRangeMin:  22,
								PortRangeMax:  22,
								Protocol:      "tcp",
								RemoteGroupID: "",
							},
						},
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				// create security groups first if they don't exist
				m.
					ListSecGroup(gomock.Any()).Return([]groups.SecGroup{}, nil).AnyTimes()

				m.
					CreateSecGroup(groups.CreateOpts{
						Name:        expectedControlPlaneSecurityGroupName,
						Description: "Cluster API managed group",
					}).
					Return(&groups.SecGroup{
						Name:        expectedControlPlaneSecurityGroupName,
						Description: "Cluster API managed group",
						ID:          fakeControlPlaneSecurityGroupID,
					}, nil)

				m.
					CreateSecGroup(groups.CreateOpts{
						Name:        expectedWorkerSecurityGroupName,
						Description: "Cluster API managed group",
					}).
					Return(&groups.SecGroup{
						Name:        expectedWorkerSecurityGroupName,
						Description: "Cluster API managed group",
						ID:          fakeWorkerSecurityGroupID,
					}, nil)

			},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{},
				Status: infrav1.OpenStackClusterStatus{
					ControlPlaneSecurityGroup: &infrav1.SecurityGroup{
						ID:   fakeControlPlaneSecurityGroupID,
						Name: expectedControlPlaneSecurityGroupName,
					},
					WorkerSecurityGroup: &infrav1.SecurityGroup{
						ID:   fakeWorkerSecurityGroupID,
						Name: expectedWorkerSecurityGroupName,
					},
				},
			},
			wantError: false,
		},
		{
			name: "ensures status set when reconciling legacy managed security groups with bastion",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ManagedSecurityGroups: true,
					Bastion: &infrav1.Bastion{
						Enabled: true,
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				m.
					ListSecGroup(gomock.Any()).Return([]groups.SecGroup{}, nil).AnyTimes()

				m.
					CreateSecGroup(groups.CreateOpts{
						Name:        expectedControlPlaneSecurityGroupName,
						Description: "Cluster API managed group",
					}).
					Return(&groups.SecGroup{
						Name:        expectedControlPlaneSecurityGroupName,
						Description: "Cluster API managed group",
					}, nil)

				m.
					CreateSecGroup(groups.CreateOpts{
						Name:        expectedWorkerSecurityGroupName,
						Description: "Cluster API managed group",
					}).
					Return(&groups.SecGroup{
						Name:        expectedWorkerSecurityGroupName,
						Description: "Cluster API managed group",
					}, nil)

				m.
					CreateSecGroup(groups.CreateOpts{
						Name:        expectedBastionSecurityGroupName,
						Description: "Cluster API managed group",
					}).
					Return(&groups.SecGroup{
						Name:        expectedBastionSecurityGroupName,
						Description: "Cluster API managed group",
					}, nil)
			},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{},
				Status: infrav1.OpenStackClusterStatus{
					ControlPlaneSecurityGroup: &infrav1.SecurityGroup{
						ID:   fakeControlPlaneSecurityGroupID,
						Name: expectedControlPlaneSecurityGroupName,
					},
					WorkerSecurityGroup: &infrav1.SecurityGroup{
						Name: expectedWorkerSecurityGroupName,
					},
					BastionSecurityGroup: &infrav1.SecurityGroup{
						Name: expectedBastionSecurityGroupName,
					},
				},
			},
			wantError: false,
		},
		{
			name: "ensures status set when reconciling security groups with bastion",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					SecurityGroups: &infrav1.SecurityGroupsSpec{
						AdditionalControlPlaneSecurityGroupRules: []infrav1.SecurityGroupRule{
							{
								Description:   "Etcd",
								Direction:     "ingress",
								EtherType:     "IPv4",
								PortRangeMin:  2379,
								PortRangeMax:  2380,
								Protocol:      "tcp",
								RemoteGroupID: "",
							},
						},
						AdditionalWorkerSecurityGroupRules: []infrav1.SecurityGroupRule{
							{
								Description:   "ssh",
								Direction:     "ingress",
								EtherType:     "IPv4",
								PortRangeMin:  22,
								PortRangeMax:  22,
								Protocol:      "tcp",
								RemoteGroupID: "",
							},
						},
						AdditionalBastionSecurityGroupRules: []infrav1.SecurityGroupRule{
							{
								Description:   "ssh",
								Direction:     "ingress",
								EtherType:     "IPv4",
								PortRangeMin:  22,
								PortRangeMax:  22,
								Protocol:      "tcp",
								RemoteGroupID: "",
							},
						},
					},
					Bastion: &infrav1.Bastion{
						Enabled: true,
					},
				},
			},
			expect: func(m *mock.MockNetworkClientMockRecorder) {
				// create security groups first if they don't exist
				m.
					ListSecGroup(gomock.Any()).Return([]groups.SecGroup{}, nil).AnyTimes()

				m.
					CreateSecGroup(groups.CreateOpts{
						Name:        expectedControlPlaneSecurityGroupName,
						Description: "Cluster API managed group",
					}).
					Return(&groups.SecGroup{
						Name:        expectedControlPlaneSecurityGroupName,
						Description: "Cluster API managed group",
						ID:          fakeControlPlaneSecurityGroupID,
					}, nil)

				m.
					CreateSecGroup(groups.CreateOpts{
						Name:        expectedWorkerSecurityGroupName,
						Description: "Cluster API managed group",
					}).
					Return(&groups.SecGroup{
						Name:        expectedWorkerSecurityGroupName,
						Description: "Cluster API managed group",
						ID:          fakeWorkerSecurityGroupID,
					}, nil)

				m.
					CreateSecGroup(groups.CreateOpts{
						Name:        expectedBastionSecurityGroupName,
						Description: "Cluster API managed group",
					}).
					Return(&groups.SecGroup{
						Name:        expectedBastionSecurityGroupName,
						Description: "Cluster API managed group",
					}, nil)
			},
			want: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{},
				Status: infrav1.OpenStackClusterStatus{
					ControlPlaneSecurityGroup: &infrav1.SecurityGroup{
						ID:   fakeControlPlaneSecurityGroupID,
						Name: expectedControlPlaneSecurityGroupName,
					},
					WorkerSecurityGroup: &infrav1.SecurityGroup{
						ID:   fakeWorkerSecurityGroupID,
						Name: expectedWorkerSecurityGroupName,
					},
					BastionSecurityGroup: &infrav1.SecurityGroup{
						ID:   fakeBastionSecurityGroupID,
						Name: expectedBastionSecurityGroupName,
					},
				},
			},
			wantError: false,
		},
		{
			name: "ensures error when both legacy managed rules and security groups are set",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ManagedSecurityGroups: true,
					SecurityGroups: &infrav1.SecurityGroupsSpec{
						AdditionalControlPlaneSecurityGroupRules: []infrav1.SecurityGroupRule{
							{
								Description:   "Etcd",
								Direction:     "ingress",
								EtherType:     "IPv4",
								PortRangeMin:  2379,
								PortRangeMax:  2380,
								Protocol:      "tcp",
								RemoteGroupID: "",
							},
						},
						AdditionalWorkerSecurityGroupRules: []infrav1.SecurityGroupRule{
							{
								Description:   "ssh",
								Direction:     "ingress",
								EtherType:     "IPv4",
								PortRangeMin:  22,
								PortRangeMax:  22,
								Protocol:      "tcp",
								RemoteGroupID: "",
							},
						},
					},
				},
			},
			expect:    func(m *mock.MockNetworkClientMockRecorder) {},
			want:      &infrav1.OpenStackCluster{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			mockClient := mock.NewMockNetworkClient(mockCtrl)
			tt.expect(mockClient.EXPECT())
			s := Service{
				client: mockClient,
				scope:  scope.NewMockScopeFactory(mockCtrl, "", logr.Discard()),
			}
			err := s.ReconcileSecurityGroups(tt.openStackCluster, clusterName)
			if tt.wantError {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
		})
	}
}
