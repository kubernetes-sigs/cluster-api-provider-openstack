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

package clients

import (
	"fmt"

	"k8s.io/klog/v2"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"

	openstackconfigv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
)

const (
	secGroupPrefix     string = "k8s"
	controlPlaneSuffix string = "controlplane"
	globalSuffix       string = "all"
)

var defaultRules = []openstackconfigv1.SecurityGroupRule{
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

// SecGroupService interfaces with the OpenStack Networking API.
// It will create security groups if they're managed.
type SecGroupService struct {
	client *gophercloud.ServiceClient
}

// NewSecGroupService returns an initialised instance of SecGroupService.
func NewSecGroupService(client *gophercloud.ServiceClient) (*SecGroupService, error) {
	return &SecGroupService{
		client: client,
	}, nil
}

// Reconcile the security groups.
func (s *SecGroupService) Reconcile(clusterName string, desired openstackconfigv1.OpenstackClusterProviderSpec, status *openstackconfigv1.OpenstackClusterProviderStatus) error {
	klog.Infof("Reconciling security groups for cluster %s", clusterName)
	if !desired.ManagedSecurityGroups {
		klog.V(4).Infof("No need to reconcile security groups for cluster %s", clusterName)
		return nil
	}
	desiredSecGroups := map[string]openstackconfigv1.SecurityGroup{
		"controlplane": s.generateControlPlaneGroup(clusterName),
		"global":       s.generateGlobalGroup(clusterName),
	}
	observedSecGroups := make(map[string]*openstackconfigv1.SecurityGroup)

	for k, desiredSecGroup := range desiredSecGroups {
		klog.Infof("Reconciling security group %s", desiredSecGroup.Name)

		var err error
		observedSecGroups[k], err = s.getSecurityGroupByName(desiredSecGroup.Name)

		if err != nil {
			return err
		}

		if observedSecGroups[k].ID != "" {
			if s.matchGroups(&desiredSecGroup, observedSecGroups[k]) {
				klog.V(6).Infof("Group %s matched, have nothing to do.", desiredSecGroup.Name)
				continue
			}

			klog.V(6).Infof("Group %s didn't match, reconciling...", desiredSecGroup.Name)
			observedSecGroups[k], err = s.reconcileGroup(&desiredSecGroup, observedSecGroups[k])
			if err != nil {
				return err
			}
			continue
		}

		klog.V(6).Infof("Group %s doesn't exist, creating it.", desiredSecGroup.Name)
		observedSecGroups[k], err = s.createSecGroup(desiredSecGroup)
	}

	status.ControlPlaneSecurityGroup = observedSecGroups["controlplane"]
	status.GlobalSecurityGroup = observedSecGroups["global"]

	return nil
}

func (s *SecGroupService) Delete(group *openstackconfigv1.SecurityGroup) error {
	exists, err := s.exists(group.ID)
	if err != nil {
		return err
	}
	if exists {
		return groups.Delete(s.client, group.ID).ExtractErr()
	}
	return nil
}

func (s *SecGroupService) exists(groupID string) (bool, error) {
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

func (s *SecGroupService) generateControlPlaneGroup(clusterName string) openstackconfigv1.SecurityGroup {
	secGroupName := fmt.Sprintf("%s-cluster-%s-secgroup-%s", secGroupPrefix, clusterName, controlPlaneSuffix)

	// Hardcoded rules for now, we might want to make this definable in the Spec but it's more
	// likely that the infrastructure plan in cluster-api will have taken form by then.
	return openstackconfigv1.SecurityGroup{
		Name: secGroupName,
		Rules: append(
			[]openstackconfigv1.SecurityGroupRule{
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

func (s *SecGroupService) generateGlobalGroup(clusterName string) openstackconfigv1.SecurityGroup {
	secGroupName := fmt.Sprintf("%s-cluster-%s-secgroup-%s", secGroupPrefix, clusterName, globalSuffix)

	// As above, hardcoded rules.
	return openstackconfigv1.SecurityGroup{
		Name: secGroupName,
		Rules: append(
			[]openstackconfigv1.SecurityGroupRule{
				{
					Direction:     "ingress",
					EtherType:     "IPv4",
					PortRangeMin:  1,
					PortRangeMax:  65535,
					Protocol:      "tcp",
					RemoteGroupID: "self",
				},
				{
					Direction:     "ingress",
					EtherType:     "IPv4",
					PortRangeMin:  1,
					PortRangeMax:  65535,
					Protocol:      "udp",
					RemoteGroupID: "self",
				},
				{
					Direction:     "ingress",
					EtherType:     "IPv4",
					PortRangeMin:  0,
					PortRangeMax:  0,
					Protocol:      "icmp",
					RemoteGroupID: "self",
				},
			},
			defaultRules...,
		),
	}
}

// matchGroups will check if security groups match.
func (s *SecGroupService) matchGroups(desired, observed *openstackconfigv1.SecurityGroup) bool {
	// If they have differing amount of rules they obviously don't match.
	if len(desired.Rules) != len(observed.Rules) {
		return false
	}

	// Rules aren't in any order, so we're doing this the hard way.
	for _, desiredRule := range desired.Rules {
		r := desiredRule
		if r.RemoteGroupID == "self" {
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
func (s *SecGroupService) reconcileGroup(desired, observed *openstackconfigv1.SecurityGroup) (*openstackconfigv1.SecurityGroup, error) {
	klog.V(6).Infof("Deleting all rules for group %s", observed.Name)
	for _, rule := range observed.Rules {
		klog.V(6).Infof("Deleting rule %s from group %s", rule.ID, observed.Name)
		err := rules.Delete(s.client, rule.ID).ExtractErr()
		if err != nil {
			return &openstackconfigv1.SecurityGroup{}, err
		}
	}
	recreatedRules := make([]openstackconfigv1.SecurityGroupRule, 0, len(desired.Rules))
	klog.V(6).Infof("Recreating all rules for group %s", observed.Name)
	for _, rule := range desired.Rules {
		r := rule
		r.SecurityGroupID = observed.ID
		if r.RemoteGroupID == "self" {
			r.RemoteGroupID = observed.ID
		}
		newRule, err := s.createRule(r)
		if err != nil {
			return &openstackconfigv1.SecurityGroup{}, err
		}
		recreatedRules = append(recreatedRules, newRule)
	}
	observed.Rules = recreatedRules
	return observed, nil
}

func (s *SecGroupService) createSecGroup(group openstackconfigv1.SecurityGroup) (*openstackconfigv1.SecurityGroup, error) {
	createOpts := groups.CreateOpts{
		Name:        group.Name,
		Description: "Cluster API managed group",
	}
	klog.V(6).Infof("Creating group %+v", createOpts)
	g, err := groups.Create(s.client, createOpts).Extract()
	if err != nil {
		return &openstackconfigv1.SecurityGroup{}, err
	}

	newGroup := s.convertOSSecGroupToConfigSecGroup(*g)
	rules := make([]openstackconfigv1.SecurityGroupRule, 0, len(group.Rules))
	klog.V(6).Infof("Creating rules for group %s", group.Name)
	for _, rule := range group.Rules {
		r := rule
		r.SecurityGroupID = newGroup.ID
		if r.RemoteGroupID == "self" {
			r.RemoteGroupID = newGroup.ID
		}
		newRule, err := s.createRule(r)
		if err != nil {
			return &openstackconfigv1.SecurityGroup{}, err
		}
		rules = append(rules, newRule)
	}
	newGroup.Rules = rules

	return newGroup, nil
}

func (s *SecGroupService) getSecurityGroupByName(name string) (*openstackconfigv1.SecurityGroup, error) {
	opts := groups.ListOpts{
		Name: name,
	}

	klog.V(6).Infof("Attempting to fetch security group with name %s", name)
	allPages, err := groups.List(s.client, opts).AllPages()
	if err != nil {
		return &openstackconfigv1.SecurityGroup{}, err
	}

	allGroups, err := groups.ExtractGroups(allPages)
	if err != nil {
		return &openstackconfigv1.SecurityGroup{}, err
	}

	switch len(allGroups) {
	case 0:
		return &openstackconfigv1.SecurityGroup{}, nil
	case 1:
		return s.convertOSSecGroupToConfigSecGroup(allGroups[0]), nil
	}

	return &openstackconfigv1.SecurityGroup{}, fmt.Errorf("More than one security group found named: %s", name)
}

func (s *SecGroupService) createRule(r openstackconfigv1.SecurityGroupRule) (openstackconfigv1.SecurityGroupRule, error) {
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
	klog.V(6).Infof("Creating rule %+v", createOpts)
	rule, err := rules.Create(s.client, createOpts).Extract()
	if err != nil {
		return openstackconfigv1.SecurityGroupRule{}, err
	}
	return s.convertOSSecGroupRuleToConfigSecGroupRule(*rule), nil
}

func (s *SecGroupService) convertOSSecGroupToConfigSecGroup(osSecGroup groups.SecGroup) *openstackconfigv1.SecurityGroup {
	rules := make([]openstackconfigv1.SecurityGroupRule, len(osSecGroup.Rules))
	for i, rule := range osSecGroup.Rules {
		rules[i] = s.convertOSSecGroupRuleToConfigSecGroupRule(rule)
	}
	return &openstackconfigv1.SecurityGroup{
		ID:    osSecGroup.ID,
		Name:  osSecGroup.Name,
		Rules: rules,
	}

}

func (s *SecGroupService) convertOSSecGroupRuleToConfigSecGroupRule(osSecGroupRule rules.SecGroupRule) openstackconfigv1.SecurityGroupRule {
	return openstackconfigv1.SecurityGroupRule{
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
