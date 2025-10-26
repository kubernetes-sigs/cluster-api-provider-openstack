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
	"errors"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/policies"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	capoerrors "sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/filterconvert"
)

// GetQoSPolicyIDByParam returns a qos policy ID based on the params passed by
// the user.
func (s *Service) GetQoSPolicyIDByParam(policyParam *infrav1.QoSPolicyParam) (string, error) {
	if policyParam.ID != nil {
		return *policyParam.ID, nil
	}

	if policyParam.Filter == nil {
		// Should have been caught by validation
		return "", errors.New("invalid qos policy param, either ID or Filter must be set")
	}

	listOpts := filterconvert.QoSPolicyFilterToListOpts(policyParam.Filter)
	policy, err := s.getQoSPolicyByFilter(listOpts)
	if err != nil {
		return "", err
	}
	return policy.ID, nil
}

func (s *Service) getQoSPolicyByFilter(opts policies.ListOpts) (*policies.Policy, error) {
	policyList, err := s.client.ListQoSPolicy(opts)
	if err != nil {
		return nil, err
	}

	switch len(policyList) {
	case 0:
		return nil, capoerrors.ErrNoMatches
	case 1:
		return &policyList[0], nil
	}
	return nil, capoerrors.ErrMultipleMatches
}
