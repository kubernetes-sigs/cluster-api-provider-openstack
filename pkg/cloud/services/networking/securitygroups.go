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
	"fmt"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha3"
)

const (
	secGroupPrefix     string = "k8s"
	controlPlaneSuffix string = "controlplane"
	globalSuffix       string = "all"
	remoteGroupIDSelf  string = "self"
)

var defaultRules = []infrav1.SecurityGroupRule{
	{
		Direction:      "egress",
		EtherType:      "IPv4",
		PortRangeMin:   0,
		PortRangeMax:   0,
		Protocol:       "",
		RemoteIPPrefix: "",
	},
	{
		Direction:      "egress",
		EtherType:      "IPv6",
		PortRangeMin:   0,
		PortRangeMax:   0,
		Protocol:       "",
		RemoteIPPrefix: "",
	},
}

// Reconcile the security groups.
func (s *Service) ReconcileSecurityGroups(clusterName string, openStackCluster *infrav1.OpenStackCluster) error {
	s.logger.Info("Reconciling security groups", "cluster", clusterName)
	if !openStackCluster.Spec.ManagedSecurityGroups {
		s.logger.V(4).Info("No need to reconcile security groups", "cluster", clusterName)
		return nil
	}
	desiredSecGroups := map[string]infrav1.SecurityGroup{
		"controlplane": generateControlPlaneGroup(clusterName),
		"global":       generateGlobalGroup(clusterName),
	}
	observedSecGroups := make(map[string]*infrav1.SecurityGroup)

	for k, desiredSecGroup := range desiredSecGroups {
		s.logger.Info("Reconciling security group", "name", desiredSecGroup.Name)

		var err error
		observedSecGroups[k], err = s.getSecurityGroupByName(desiredSecGroup.Name)

		if err != nil {
			return err
		}

		if observedSecGroups[k].ID != "" {
			if matchGroups(desiredSecGroup, *observedSecGroups[k]) {
				s.logger.V(6).Info("Group matched, have nothing to do.", "name", desiredSecGroup.Name)
				continue
			}

			s.logger.V(6).Info("Group didn't match, reconciling...", "name", desiredSecGroup.Name)
			observedSecGroup, err := s.reconcileGroup(desiredSecGroup, *observedSecGroups[k])
			if err != nil {
				return err
			}
			observedSecGroups[k] = &observedSecGroup
			continue
		}

		s.logger.V(6).Info("Group doesn't exist, creating it.", "name", desiredSecGroup.Name)
		observedSecGroups[k], err = s.createSecGroup(desiredSecGroup)
		if err != nil {
			return err
		}
	}

	openStackCluster.Status.ControlPlaneSecurityGroup = observedSecGroups["controlplane"]
	openStackCluster.Status.GlobalSecurityGroup = observedSecGroups["global"]

	return nil
}

func (s *Service) DeleteSecurityGroups(group *infrav1.SecurityGroup) error {
	exists, err := s.exists(group.ID)
	if err != nil {
		return err
	}
	if exists {
		return groups.Delete(s.client, group.ID).ExtractErr()
	}
	return nil
}

func (s *Service) exists(groupID string) (bool, error) {
	opts := groups.ListOpts{
		ID: groupID,
	}
	allPages, err := groups.List(s.client, opts).AllPages()
	if err != nil {
		return false, err
	}
	allGroups, err := groups.ExtractGroups(allPages)
	if err != nil {
		return false, err
	}
	if len(allGroups) == 0 {
		return false, nil
	}
	return true, nil
}

func generateControlPlaneGroup(clusterName string) infrav1.SecurityGroup {
	secGroupName := fmt.Sprintf("%s-cluster-%s-secgroup-%s", secGroupPrefix, clusterName, controlPlaneSuffix)

	// Hardcoded rules for now, we might want to make this definable in the Spec but it's more
	// likely that the infrastructure plan in cluster-api will have taken form by then.
	return infrav1.SecurityGroup{
		Name: secGroupName,
		Rules: append(
			[]infrav1.SecurityGroupRule{
				{
					Direction:      "ingress",
					EtherType:      "IPv4",
					PortRangeMin:   443,
					PortRangeMax:   443,
					Protocol:       "tcp",
					RemoteIPPrefix: "0.0.0.0/0",
				},
				{
					Direction:      "ingress",
					EtherType:      "IPv4",
					PortRangeMin:   22,
					PortRangeMax:   22,
					Protocol:       "tcp",
					RemoteIPPrefix: "0.0.0.0/0",
				},
			},
			defaultRules...,
		),
	}
}

func generateGlobalGroup(clusterName string) infrav1.SecurityGroup {
	secGroupName := fmt.Sprintf("%s-cluster-%s-secgroup-%s", secGroupPrefix, clusterName, globalSuffix)

	// As above, hardcoded rules.

	return infrav1.SecurityGroup{
		Name: secGroupName,
		Rules: append(
			[]infrav1.SecurityGroupRule{
				{
					Direction:     "ingress",
					EtherType:     "IPv4",
					PortRangeMin:  1,
					PortRangeMax:  65535,
					Protocol:      "tcp",
					RemoteGroupID: remoteGroupIDSelf,
				},
				{
					Direction:     "ingress",
					EtherType:     "IPv4",
					PortRangeMin:  1,
					PortRangeMax:  65535,
					Protocol:      "udp",
					RemoteGroupID: remoteGroupIDSelf,
				},
				{
					Direction:     "ingress",
					EtherType:     "IPv4",
					PortRangeMin:  0,
					PortRangeMax:  0,
					Protocol:      "icmp",
					RemoteGroupID: remoteGroupIDSelf,
				},
			},
			defaultRules...,
		),
	}
}

// matchGroups will check if security groups match.
func matchGroups(desired, observed infrav1.SecurityGroup) bool {
	// If they have differing amount of rules they obviously don't match.
	if len(desired.Rules) != len(observed.Rules) {
		return false
	}

	// Rules aren't in any order, so we're doing this the hard way.
	for _, desiredRule := range desired.Rules {
		r := desiredRule
		if r.RemoteGroupID == remoteGroupIDSelf {
			r.RemoteGroupID = observed.ID
		}
		ruleMatched := false
		for _, observedRule := range observed.Rules {
			if observedRule.Equal(r) {
				ruleMatched = true
				break
			}
		}

		if !ruleMatched {
			return false
		}
	}
	return true
}

// reconcileGroup reconciles an already existing observed group by essentially emptying out all the rules and
// recreating them.
func (s *Service) reconcileGroup(desired, observed infrav1.SecurityGroup) (infrav1.SecurityGroup, error) {
	s.logger.V(6).Info("Deleting all rules for group", "name", observed.Name)
	for _, rule := range observed.Rules {
		s.logger.V(6).Info("Deleting rule", "ruleID", rule.ID, "groupName", observed.Name)
		err := rules.Delete(s.client, rule.ID).ExtractErr()
		if err != nil {
			return infrav1.SecurityGroup{}, err
		}
	}
	recreatedRules := make([]infrav1.SecurityGroupRule, 0, len(desired.Rules))
	s.logger.V(6).Info("Recreating all rules for group", "name", observed.Name)
	for _, rule := range desired.Rules {
		r := rule
		r.SecurityGroupID = observed.ID
		if r.RemoteGroupID == remoteGroupIDSelf {
			r.RemoteGroupID = observed.ID
		}
		newRule, err := s.createRule(r)
		if err != nil {
			return infrav1.SecurityGroup{}, err
		}
		recreatedRules = append(recreatedRules, newRule)
	}
	observed.Rules = recreatedRules
	return observed, nil
}

func (s *Service) createSecGroup(group infrav1.SecurityGroup) (*infrav1.SecurityGroup, error) {
	createOpts := groups.CreateOpts{
		Name:        group.Name,
		Description: "Cluster API managed group",
	}
	s.logger.V(6).Info("Creating group", "name", group.Name)
	g, err := groups.Create(s.client, createOpts).Extract()
	if err != nil {
		return &infrav1.SecurityGroup{}, err
	}

	newGroup := convertOSSecGroupToConfigSecGroup(*g)
	securityGroupRules := make([]infrav1.SecurityGroupRule, 0, len(group.Rules))
	s.logger.V(6).Info("Creating rules for group", "name", group.Name)
	for _, rule := range group.Rules {
		r := rule
		r.SecurityGroupID = newGroup.ID
		if r.RemoteGroupID == remoteGroupIDSelf {
			r.RemoteGroupID = newGroup.ID
		}
		newRule, err := s.createRule(r)
		if err != nil {
			return &infrav1.SecurityGroup{}, err
		}
		securityGroupRules = append(securityGroupRules, newRule)
	}
	newGroup.Rules = securityGroupRules

	return newGroup, nil
}

func (s *Service) getSecurityGroupByName(name string) (*infrav1.SecurityGroup, error) {
	opts := groups.ListOpts{
		Name: name,
	}

	s.logger.V(6).Info("Attempting to fetch security group with", "name", name)
	allPages, err := groups.List(s.client, opts).AllPages()
	if err != nil {
		return &infrav1.SecurityGroup{}, err
	}

	allGroups, err := groups.ExtractGroups(allPages)
	if err != nil {
		return &infrav1.SecurityGroup{}, err
	}

	switch len(allGroups) {
	case 0:
		return &infrav1.SecurityGroup{}, nil
	case 1:
		return convertOSSecGroupToConfigSecGroup(allGroups[0]), nil
	}

	return &infrav1.SecurityGroup{}, fmt.Errorf("more than one security group found named: %s", name)
}

func (s *Service) createRule(r infrav1.SecurityGroupRule) (infrav1.SecurityGroupRule, error) {
	dir := rules.RuleDirection(r.Direction)
	proto := rules.RuleProtocol(r.Protocol)
	etherType := rules.RuleEtherType(r.EtherType)

	createOpts := rules.CreateOpts{
		Direction:      dir,
		PortRangeMin:   r.PortRangeMin,
		PortRangeMax:   r.PortRangeMax,
		Protocol:       proto,
		EtherType:      etherType,
		RemoteGroupID:  r.RemoteGroupID,
		RemoteIPPrefix: r.RemoteIPPrefix,
		SecGroupID:     r.SecurityGroupID,
	}
	s.logger.V(6).Info("Creating rule")
	rule, err := rules.Create(s.client, createOpts).Extract()
	if err != nil {
		return infrav1.SecurityGroupRule{}, err
	}
	return convertOSSecGroupRuleToConfigSecGroupRule(*rule), nil
}

func convertOSSecGroupToConfigSecGroup(osSecGroup groups.SecGroup) *infrav1.SecurityGroup {
	securityGroupRules := make([]infrav1.SecurityGroupRule, len(osSecGroup.Rules))
	for i, rule := range osSecGroup.Rules {
		securityGroupRules[i] = convertOSSecGroupRuleToConfigSecGroupRule(rule)
	}
	return &infrav1.SecurityGroup{
		ID:    osSecGroup.ID,
		Name:  osSecGroup.Name,
		Rules: securityGroupRules,
	}
}

func convertOSSecGroupRuleToConfigSecGroupRule(osSecGroupRule rules.SecGroupRule) infrav1.SecurityGroupRule {
	return infrav1.SecurityGroupRule{
		ID:              osSecGroupRule.ID,
		Direction:       osSecGroupRule.Direction,
		EtherType:       osSecGroupRule.EtherType,
		SecurityGroupID: osSecGroupRule.SecGroupID,
		PortRangeMin:    osSecGroupRule.PortRangeMin,
		PortRangeMax:    osSecGroupRule.PortRangeMax,
		Protocol:        osSecGroupRule.Protocol,
		RemoteGroupID:   osSecGroupRule.RemoteGroupID,
		RemoteIPPrefix:  osSecGroupRule.RemoteIPPrefix,
	}
}
