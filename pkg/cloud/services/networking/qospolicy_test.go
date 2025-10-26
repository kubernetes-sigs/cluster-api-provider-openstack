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

package networking

import (
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/policies"
	. "github.com/onsi/gomega" //nolint:revive
	"go.uber.org/mock/gomock"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/clients/mock"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/filterconvert"
)

func Test_GetQoSPolicyIDByParam(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	g := NewWithT(t)
	mockClient := mock.NewMockNetworkClient(mockCtrl)

	idDirect := "7fd24ceb-788a-441f-ad0a-d8e2f5d31a1d"
	idGold := "d9c88a6d-0b8c-48ff-8f0e-8d85a078c194"

	tests := []struct {
		name    string
		param   *infrav1.QoSPolicyParam
		expect  func()
		wantID  string
		wantErr string
	}{
		{
			name:   "ID short-circuit (no client call)",
			param:  &infrav1.QoSPolicyParam{ID: &idDirect},
			expect: func() {},
			wantID: idDirect,
		},
		{
			name:    "nil ID & nil Filter -> validation error",
			param:   &infrav1.QoSPolicyParam{},
			expect:  func() {},
			wantErr: "invalid qos policy param",
		},
		{
			name: "filter -> single match returns ID",
			param: &infrav1.QoSPolicyParam{
				Filter: &infrav1.QoSPolicyFilter{Name: "gold"},
			},
			expect: func() {
				opts := filterconvert.QoSPolicyFilterToListOpts(&infrav1.QoSPolicyFilter{Name: "gold"})
				mockClient.EXPECT().ListQoSPolicy(opts).
					Return([]policies.Policy{{ID: idGold, Name: "gold"}}, nil)
			},
			wantID: idGold,
		},
		{
			name: "filter -> no matches propagates ErrNoMatches",
			param: &infrav1.QoSPolicyParam{
				Filter: &infrav1.QoSPolicyFilter{Name: "none"},
			},
			expect: func() {
				opts := filterconvert.QoSPolicyFilterToListOpts(&infrav1.QoSPolicyFilter{Name: "none"})
				mockClient.EXPECT().ListQoSPolicy(opts).
					Return([]policies.Policy{}, nil)
			},
			wantErr: capoerrors.ErrNoMatches.Error(),
		},
		{
			name: "filter -> multiple matches propagates ErrMultipleMatches",
			param: &infrav1.QoSPolicyParam{
				Filter: &infrav1.QoSPolicyFilter{Description: "dup"},
			},
			expect: func() {
				opts := filterconvert.QoSPolicyFilterToListOpts(&infrav1.QoSPolicyFilter{Description: "dup"})
				mockClient.EXPECT().ListQoSPolicy(opts).
					Return([]policies.Policy{{ID: "a"}, {ID: "b"}}, nil)
			},
			wantErr: capoerrors.ErrMultipleMatches.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.expect()

			scopeFactory := scope.NewMockScopeFactory(mockCtrl, "")
			log := testr.New(t)
			s := Service{
				client: mockClient,
				scope:  scope.NewWithLogger(scopeFactory, log),
			}
			got, err := s.GetQoSPolicyIDByParam(tt.param)
			if tt.wantErr != "" {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(ContainSubstring(tt.wantErr))
				return
			}
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(got).To(Equal(tt.wantID))
		})
	}
}
