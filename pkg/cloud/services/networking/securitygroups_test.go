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
package networking

import (
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules"
	. "github.com/onsi/gomega" //nolint:revive
	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

func TestValidateRemoteManagedGroups(t *testing.T) {
	tests := []struct {
		name                string
		rule                infrav1.SecurityGroupRuleSpec
		remoteManagedGroups map[string]string
		wantErr             bool
	}{
		{
			name: "Invalid rule with unknown remoteManagedGroup",
			rule: infrav1.SecurityGroupRuleSpec{
				RemoteManagedGroups: []infrav1.ManagedSecurityGroupName{"unknownGroup"},
			},
			wantErr: true,
		},
		{
			name: "Valid rule with no remoteManagedGroups",
			rule: infrav1.SecurityGroupRuleSpec{
				PortRangeMin:   ptr.To(22),
				PortRangeMax:   ptr.To(22),
				Protocol:       ptr.To("tcp"),
				RemoteIPPrefix: ptr.To("0.0.0.0/0"),
			},
			wantErr: false,
		},
		{
			name: "Valid rule with remoteManagedGroups",
			rule: infrav1.SecurityGroupRuleSpec{
				RemoteManagedGroups: []infrav1.ManagedSecurityGroupName{"controlplane", "worker", "bastion"},
			},
			remoteManagedGroups: map[string]string{
				"self":         "self",
				"controlplane": "1",
				"worker":       "2",
				"bastion":      "3",
			},
			wantErr: false,
		},
		{
			name: "Invalid rule with bastion in remoteManagedGroups",
			rule: infrav1.SecurityGroupRuleSpec{
				RemoteManagedGroups: []infrav1.ManagedSecurityGroupName{"controlplane", "worker", "bastion"},
			},
			remoteManagedGroups: map[string]string{
				"self":         "self",
				"controlplane": "1",
				"worker":       "2",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRemoteManagedGroups(tt.remoteManagedGroups, tt.rule.RemoteManagedGroups)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAllNodesRule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetRulesFromSpecs(t *testing.T) {
	tests := []struct {
		name                       string
		remoteManagedGroups        map[string]string
		allNodesSecurityGroupRules []infrav1.SecurityGroupRuleSpec
		wantRules                  []resolvedSecurityGroupRuleSpec
		wantErr                    bool
	}{
		{
			name:                       "Empty remoteManagedGroups and allNodesSecurityGroupRules",
			remoteManagedGroups:        map[string]string{},
			allNodesSecurityGroupRules: []infrav1.SecurityGroupRuleSpec{},
			wantRules:                  []resolvedSecurityGroupRuleSpec{},
			wantErr:                    false,
		},
		{
			name: "Valid remoteManagedGroups and allNodesSecurityGroupRules",
			remoteManagedGroups: map[string]string{
				"controlplane": "1",
				"worker":       "2",
			},
			allNodesSecurityGroupRules: []infrav1.SecurityGroupRuleSpec{
				{
					Protocol:     ptr.To("tcp"),
					PortRangeMin: ptr.To(22),
					PortRangeMax: ptr.To(22),
					RemoteManagedGroups: []infrav1.ManagedSecurityGroupName{
						"controlplane",
						"worker",
					},
				},
			},
			wantRules: []resolvedSecurityGroupRuleSpec{
				{
					Protocol:      "tcp",
					PortRangeMin:  22,
					PortRangeMax:  22,
					RemoteGroupID: "1",
				},
				{
					Protocol:      "tcp",
					PortRangeMin:  22,
					PortRangeMax:  22,
					RemoteGroupID: "2",
				},
			},
			wantErr: false,
		},
		{
			name: "Valid remoteManagedGroups in a rule",
			remoteManagedGroups: map[string]string{
				"controlplane": "1",
				"worker":       "2",
			},
			allNodesSecurityGroupRules: []infrav1.SecurityGroupRuleSpec{
				{
					Protocol:            ptr.To("tcp"),
					PortRangeMin:        ptr.To(22),
					PortRangeMax:        ptr.To(22),
					RemoteManagedGroups: []infrav1.ManagedSecurityGroupName{"controlplane"},
				},
			},
			wantRules: []resolvedSecurityGroupRuleSpec{
				{
					Protocol:      "tcp",
					PortRangeMin:  22,
					PortRangeMax:  22,
					RemoteGroupID: "1",
				},
			},
		},
		{
			name: "Valid remoteIPPrefix in a rule",
			remoteManagedGroups: map[string]string{
				"controlplane": "1",
				"worker":       "2",
			},
			allNodesSecurityGroupRules: []infrav1.SecurityGroupRuleSpec{
				{
					Protocol:       ptr.To("tcp"),
					PortRangeMin:   ptr.To(22),
					PortRangeMax:   ptr.To(22),
					RemoteIPPrefix: ptr.To("0.0.0.0/0"),
				},
			},
			wantRules: []resolvedSecurityGroupRuleSpec{
				{
					Protocol:       "tcp",
					PortRangeMin:   22,
					PortRangeMax:   22,
					RemoteIPPrefix: "0.0.0.0/0",
				},
			},
		},
		{
			name: "Valid allNodesSecurityGroupRules with no remote parameter",
			remoteManagedGroups: map[string]string{
				"controlplane": "1",
				"worker":       "2",
			},
			allNodesSecurityGroupRules: []infrav1.SecurityGroupRuleSpec{
				{
					Protocol:     ptr.To("tcp"),
					PortRangeMin: ptr.To(22),
					PortRangeMax: ptr.To(22),
				},
			},
			wantRules: []resolvedSecurityGroupRuleSpec{
				{
					Protocol:     "tcp",
					PortRangeMin: 22,
					PortRangeMax: 22,
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid allNodesSecurityGroupRules with bastion while remoteManagedGroups does not have bastion",
			remoteManagedGroups: map[string]string{
				"controlplane": "1",
				"worker":       "2",
			},
			allNodesSecurityGroupRules: []infrav1.SecurityGroupRuleSpec{
				{
					Protocol:     ptr.To("tcp"),
					PortRangeMin: ptr.To(22),
					PortRangeMax: ptr.To(22),
					RemoteManagedGroups: []infrav1.ManagedSecurityGroupName{
						"bastion",
					},
				},
			},
			wantRules: nil,
			wantErr:   true,
		},
		{
			name: "Invalid allNodesSecurityGroupRules with wrong remoteManagedGroups",
			remoteManagedGroups: map[string]string{
				"controlplane": "1",
				"worker":       "2",
			},
			allNodesSecurityGroupRules: []infrav1.SecurityGroupRuleSpec{
				{
					Protocol:     ptr.To("tcp"),
					PortRangeMin: ptr.To(22),
					PortRangeMax: ptr.To(22),
					RemoteManagedGroups: []infrav1.ManagedSecurityGroupName{
						"controlplanezzz",
						"worker",
					},
				},
			},
			wantRules: nil,
			wantErr:   true,
		},
		{
			name: "Invalid allNodesSecurityGroupRules with bastion while remoteManagedGroups does not have bastion",
			remoteManagedGroups: map[string]string{
				"controlplane": "1",
				"worker":       "2",
			},
			allNodesSecurityGroupRules: []infrav1.SecurityGroupRuleSpec{
				{
					Protocol:     ptr.To("tcp"),
					PortRangeMin: ptr.To(22),
					PortRangeMax: ptr.To(22),
					RemoteManagedGroups: []infrav1.ManagedSecurityGroupName{
						"bastion",
					},
				},
			},
			wantRules: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRules, err := getRulesFromSpecs(tt.remoteManagedGroups, tt.allNodesSecurityGroupRules)
			if (err != nil) != tt.wantErr {
				t.Errorf("getRulesFromSpecs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRules, tt.wantRules) {
				t.Errorf("getRulesFromSpecs() gotRules = %v, want %v", gotRules, tt.wantRules)
			}
		})
	}
}

func TestGenerateDesiredSecGroups(t *testing.T) {
	secGroupNames := map[string]string{
		"controlplane": "k8s-cluster-mycluster-secgroup-controlplane",
		"worker":       "k8s-cluster-mycluster-secgroup-worker",
	}

	observedSecGroupsBySuffix := map[string]*groups.SecGroup{
		"controlplane": {
			ID:   "0",
			Name: "k8s-cluster-mycluster-secgroup-controlplane",
		},
		"worker": {
			ID:   "1",
			Name: "k8s-cluster-mycluster-secgroup-worker",
		},
	}

	tests := []struct {
		name             string
		openStackCluster *infrav1.OpenStackCluster
		// We could also test the exact rules that are returned, but that'll be a lot data to write out.
		// For now we just make sure that the number of rules is correct.
		expectedNumberSecurityGroupRules int
		wantErr                          bool
	}{
		{
			name:                             "Valid openStackCluster with unmanaged securityGroups",
			openStackCluster:                 &infrav1.OpenStackCluster{},
			expectedNumberSecurityGroupRules: 0,
			wantErr:                          false,
		},
		{
			name: "Valid openStackCluster with default securityGroups",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ManagedSecurityGroups: &infrav1.ManagedSecurityGroups{},
				},
			},
			expectedNumberSecurityGroupRules: 14,
			wantErr:                          false,
		},
		{
			name: "Valid openStackCluster with default + additional security groups",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ManagedSecurityGroups: &infrav1.ManagedSecurityGroups{
						// This should add 4 rules (two for the control plane group and two for the worker group)
						AllNodesSecurityGroupRules: []infrav1.SecurityGroupRuleSpec{
							{
								Protocol:            ptr.To("tcp"),
								PortRangeMin:        ptr.To(22),
								PortRangeMax:        ptr.To(22),
								RemoteManagedGroups: []infrav1.ManagedSecurityGroupName{"controlplane", "worker"},
							},
						},
						// This should add one rule
						ControlPlaneNodesSecurityGroupRules: []infrav1.SecurityGroupRuleSpec{
							{
								Protocol:            ptr.To("tcp"),
								PortRangeMin:        ptr.To(9000),
								PortRangeMax:        ptr.To(9000),
								RemoteManagedGroups: []infrav1.ManagedSecurityGroupName{"controlplane"},
							},
						},
						// This should also add one rule
						WorkerNodesSecurityGroupRules: []infrav1.SecurityGroupRuleSpec{
							{
								Protocol:       ptr.To("tcp"),
								Direction:      "ingress",
								EtherType:      ptr.To("IPv4"),
								PortRangeMin:   ptr.To(30000),
								PortRangeMax:   ptr.To(32767),
								RemoteIPPrefix: ptr.To("0.0.0.0/0"),
							},
						},
					},
				},
			},
			expectedNumberSecurityGroupRules: 20,
			wantErr:                          false,
		},
		{
			name: "Valid openStackCluster with invalid allNodesSecurityGroupRules",
			openStackCluster: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{
					ManagedSecurityGroups: &infrav1.ManagedSecurityGroups{
						AllNodesSecurityGroupRules: []infrav1.SecurityGroupRuleSpec{
							{
								Protocol:            ptr.To("tcp"),
								PortRangeMin:        ptr.To(22),
								PortRangeMax:        ptr.To(22),
								RemoteManagedGroups: []infrav1.ManagedSecurityGroupName{"controlplane", "worker", "unknownGroup"},
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			g := NewWithT(t)
			log := testr.New(t)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")

			s, err := NewService(scope.NewWithLogger(mockScopeFactory, log))
			if err != nil {
				t.Fatalf("Failed to create service: %v", err)
			}

			gotSecurityGroups, err := s.generateDesiredSecGroups(tt.openStackCluster, secGroupNames, observedSecGroupsBySuffix)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
			var gotNumberSecurityGroupRules int
			for _, secGroup := range gotSecurityGroups {
				gotNumberSecurityGroupRules += len(secGroup.Rules)
			}
			g.Expect(gotNumberSecurityGroupRules).To(Equal(tt.expectedNumberSecurityGroupRules))
		})
	}
}

func TestReconcileGroupRules(t *testing.T) {
	const (
		sgID           = "6260e813-af79-4592-8d1a-0f42dd26cc42"
		sgRuleID       = "52a532c4-2b44-4582-ba87-b64e62e19b1a"
		sgLegacyRuleID = "a057dcc4-1535-469d-9d28-923cad9d4c56"
		sgName         = "k8s-cluster-mycluster-secgroup-controlplane"
	)

	tests := []struct {
		name          string
		desiredSGSpec securityGroupSpec
		observedSG    groups.SecGroup
		mockExpect    func(m *mock.MockNetworkClientMockRecorder)
		wantSGStatus  infrav1.SecurityGroupStatus
	}{
		{
			name:          "Empty desiredSGSpec and observedSG",
			desiredSGSpec: securityGroupSpec{},
			observedSG: groups.SecGroup{
				ID:    sgID,
				Name:  sgName,
				Rules: []rules.SecGroupRule{},
			},
			mockExpect:   func(*mock.MockNetworkClientMockRecorder) {},
			wantSGStatus: infrav1.SecurityGroupStatus{},
		},
		{
			name: "Same desiredSGSpec and observedSG produces no changes",
			desiredSGSpec: securityGroupSpec{
				Name: sgName,
				Rules: []resolvedSecurityGroupRuleSpec{
					{
						Description:    "Allow SSH",
						Direction:      "ingress",
						EtherType:      "IPv4",
						Protocol:       "tcp",
						PortRangeMin:   22,
						PortRangeMax:   22,
						RemoteGroupID:  "1",
						RemoteIPPrefix: "",
					},
				},
			},
			observedSG: groups.SecGroup{
				ID:   sgID,
				Name: sgName,
				Rules: []rules.SecGroupRule{
					{
						Description:    "Allow SSH",
						Direction:      "ingress",
						EtherType:      "IPv4",
						ID:             "idSGRule",
						Protocol:       "tcp",
						PortRangeMin:   22,
						PortRangeMax:   22,
						RemoteGroupID:  "1",
						RemoteIPPrefix: "",
					},
				},
			},
			mockExpect: func(*mock.MockNetworkClientMockRecorder) {},
		},
		{
			name: "Different desiredSGSpec and observedSG produces changes",
			desiredSGSpec: securityGroupSpec{
				Name: sgName,
				Rules: []resolvedSecurityGroupRuleSpec{
					{
						Description:    "Allow SSH",
						Direction:      "ingress",
						EtherType:      "IPv4",
						Protocol:       "tcp",
						PortRangeMin:   22,
						PortRangeMax:   22,
						RemoteGroupID:  "1",
						RemoteIPPrefix: "",
					},
				},
			},
			observedSG: groups.SecGroup{
				ID:   sgID,
				Name: sgName,
				Rules: []rules.SecGroupRule{
					{
						ID:             sgLegacyRuleID,
						Description:    "Allow SSH legacy",
						Direction:      "ingress",
						EtherType:      "IPv4",
						Protocol:       "tcp",
						PortRangeMin:   222,
						PortRangeMax:   222,
						RemoteGroupID:  "2",
						RemoteIPPrefix: "",
					},
				},
			},
			mockExpect: func(m *mock.MockNetworkClientMockRecorder) {
				m.DeleteSecGroupRule(sgLegacyRuleID).Return(nil)
				m.CreateSecGroupRule(rules.CreateOpts{
					SecGroupID:    sgID,
					Description:   "Allow SSH",
					Direction:     "ingress",
					EtherType:     "IPv4",
					Protocol:      "tcp",
					PortRangeMin:  22,
					PortRangeMax:  22,
					RemoteGroupID: "1",
				}).Return(&rules.SecGroupRule{
					ID:            sgRuleID,
					Description:   "Allow SSH",
					Direction:     "ingress",
					EtherType:     "IPv4",
					Protocol:      "tcp",
					PortRangeMin:  22,
					PortRangeMax:  22,
					RemoteGroupID: "1",
				}, nil)
			},
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			g := NewWithT(t)
			log := testr.New(t)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")

			s, err := NewService(scope.NewWithLogger(mockScopeFactory, log))
			if err != nil {
				t.Fatalf("Failed to create service: %v", err)
			}
			tt.mockExpect(mockScopeFactory.NetworkClient.EXPECT())

			err = s.reconcileGroupRules(&tt.desiredSGSpec, &tt.observedSG)
			g.Expect(err).To(BeNil())
		})
	}
}

func TestService_ReconcileSecurityGroups(t *testing.T) {
	const (
		clusterResourceName = "test-cluster"

		controlPlaneSGName = "k8s-cluster-test-cluster-secgroup-controlplane"
		workerSGName       = "k8s-cluster-test-cluster-secgroup-worker"
		bastionSGName      = "k8s-cluster-test-cluster-secgroup-bastion"
	)

	tests := []struct {
		name                  string
		openStackClusterSpec  infrav1.OpenStackClusterSpec
		expectedClusterStatus infrav1.OpenStackClusterStatus
		expect                func(log logr.Logger, m *mock.MockNetworkClientMockRecorder)
		wantErr               bool
	}{
		{
			name:                  "Do nothing if ManagedSecurityGroups is not enabled",
			openStackClusterSpec:  infrav1.OpenStackClusterSpec{},
			expectedClusterStatus: infrav1.OpenStackClusterStatus{},
		},
		{
			name: "Default control plane and worker security groups",
			openStackClusterSpec: infrav1.OpenStackClusterSpec{
				ManagedSecurityGroups: &infrav1.ManagedSecurityGroups{},
			},
			expect: func(log logr.Logger, m *mock.MockNetworkClientMockRecorder) {
				m.ListSecGroup(groups.ListOpts{Name: controlPlaneSGName}).
					Return([]groups.SecGroup{{ID: "0", Name: controlPlaneSGName}}, nil)
				m.ListSecGroup(groups.ListOpts{Name: workerSGName}).
					Return([]groups.SecGroup{{ID: "1", Name: workerSGName}}, nil)
				m.ListSecGroup(groups.ListOpts{Name: bastionSGName}).Return(nil, nil)

				// We expect a total of 14 rules to be created.
				// Nothing actually looks at the generated
				// rules, but we give them unique IDs anyway
				m.CreateSecGroupRule(gomock.Any()).DoAndReturn(func(opts rules.CreateOpts) (*rules.SecGroupRule, error) {
					log.Info("Created rule", "securityGroup", opts.SecGroupID, "description", opts.Description)
					return &rules.SecGroupRule{ID: uuid.NewString()}, nil
				}).Times(14)
			},
			expectedClusterStatus: infrav1.OpenStackClusterStatus{
				ControlPlaneSecurityGroup: &infrav1.SecurityGroupStatus{
					ID:   "0",
					Name: controlPlaneSGName,
				},
				WorkerSecurityGroup: &infrav1.SecurityGroupStatus{
					ID:   "1",
					Name: workerSGName,
				},
			},
		},
		{
			name: "Default control plane, worker, and bastion security groups",
			openStackClusterSpec: infrav1.OpenStackClusterSpec{
				Bastion: &infrav1.Bastion{
					Enabled: ptr.To(true),
				},
				ManagedSecurityGroups: &infrav1.ManagedSecurityGroups{},
			},
			expect: func(log logr.Logger, m *mock.MockNetworkClientMockRecorder) {
				m.ListSecGroup(groups.ListOpts{Name: controlPlaneSGName}).
					Return([]groups.SecGroup{{ID: "0", Name: controlPlaneSGName}}, nil)
				m.ListSecGroup(groups.ListOpts{Name: workerSGName}).
					Return([]groups.SecGroup{{ID: "1", Name: workerSGName}}, nil)
				m.ListSecGroup(groups.ListOpts{Name: bastionSGName}).
					Return([]groups.SecGroup{{ID: "2", Name: bastionSGName}}, nil)

				// We expect a total of 19 rules to be created.
				// Nothing actually looks at the generated
				// rules, but we give them unique IDs anyway
				m.CreateSecGroupRule(gomock.Any()).DoAndReturn(func(opts rules.CreateOpts) (*rules.SecGroupRule, error) {
					log.Info("Created rule", "securityGroup", opts.SecGroupID, "description", opts.Description)
					return &rules.SecGroupRule{ID: uuid.NewString()}, nil
				}).Times(19)
			},
			expectedClusterStatus: infrav1.OpenStackClusterStatus{
				ControlPlaneSecurityGroup: &infrav1.SecurityGroupStatus{
					ID:   "0",
					Name: controlPlaneSGName,
				},
				WorkerSecurityGroup: &infrav1.SecurityGroupStatus{
					ID:   "1",
					Name: workerSGName,
				},
				BastionSecurityGroup: &infrav1.SecurityGroupStatus{
					ID:   "2",
					Name: bastionSGName,
				},
			},
		},
	}
	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			g := NewWithT(t)
			log := testr.New(t)
			mockScopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			scope := scope.NewWithLogger(mockScopeFactory, log)

			s := &Service{
				scope:  scope,
				client: mockScopeFactory.NetworkClient,
			}
			if tt.expect != nil {
				tt.expect(log, mockScopeFactory.NetworkClient.EXPECT())
			}
			openStackCluster := &infrav1.OpenStackCluster{
				Spec: tt.openStackClusterSpec,
			}
			err := s.ReconcileSecurityGroups(openStackCluster, clusterResourceName)
			if tt.wantErr {
				g.Expect(err).ToNot(BeNil(), "ReconcileSecurityGroups")
			} else {
				g.Expect(err).To(BeNil(), "ReconcileSecurityGroups")
				g.Expect(openStackCluster.Status).To(Equal(tt.expectedClusterStatus), cmp.Diff(openStackCluster.Status, tt.expectedClusterStatus))
			}
		})
	}
}
