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
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
)

const (
	SecGroupPrefix     string = "k8s"
	ControlPlaneSuffix string = "controlplane"
	WorkerSuffix       string = "worker"
	BastionSuffix      string = "bastion"
	NeutronLbaasSuffix string = "lbaas"
	remoteGroupIDSelf  string = "self"
)

var defaultRules = []infrav1.SecurityGroupRule{
	{
		Direction:      "egress",
		Description:    "Full open",
		EtherType:      "IPv4",
		PortRangeMin:   0,
		PortRangeMax:   0,
		Protocol:       "",
		RemoteIPPrefix: "",
	},
	{
		Direction:      "egress",
		Description:    "Full open",
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

	secControlPlaneGroupName := fmt.Sprintf("%s-cluster-%s-secgroup-%s", SecGroupPrefix, clusterName, ControlPlaneSuffix)
	secWorkerGroupName := fmt.Sprintf("%s-cluster-%s-secgroup-%s", SecGroupPrefix, clusterName, WorkerSuffix)
	secGroupNames := map[string]string{
		ControlPlaneSuffix: secControlPlaneGroupName,
		WorkerSuffix:       secWorkerGroupName,
	}

	if openStackCluster.Spec.Bastion != nil && openStackCluster.Spec.Bastion.Enabled {
		secBastionGroupName := fmt.Sprintf("%s-cluster-%s-secgroup-%s", SecGroupPrefix, clusterName, BastionSuffix)
		secGroupNames[BastionSuffix] = secBastionGroupName
	}

	if openStackCluster.Spec.ManagedAPIServerLoadBalancer && !openStackCluster.Spec.UseOctavia {
		secLbaasGroupName := fmt.Sprintf("%s-cluster-%s-secgroup-%s", SecGroupPrefix, clusterName, NeutronLbaasSuffix)
		secGroupNames[NeutronLbaasSuffix] = secLbaasGroupName
	}

	//create security groups first, because desired rules use group ids.
	for _, v := range secGroupNames {
		if err := s.createSecurityGroupIfNotExists(openStackCluster, v); err != nil {
			return err
		}
	}
	// create desired security groups
	desiredSecGroups, err := s.generateDesiredSecGroups(secGroupNames, openStackCluster)
	if err != nil {
		return err
	}

	observedSecGroups := make(map[string]*infrav1.SecurityGroup)
	for k, desiredSecGroup := range desiredSecGroups {

		var err error
		observedSecGroups[k], err = s.getSecurityGroupByName(desiredSecGroup.Name)

		if err != nil {
			return err
		}

		if observedSecGroups[k].ID != "" {
			if matchGroups(desiredSecGroup, *observedSecGroups[k]) {
				s.logger.V(6).Info("Group rules matched, have nothing to do.", "name", desiredSecGroup.Name)
				continue
			}

			s.logger.V(6).Info("Group rules didn't match, reconciling...", "name", desiredSecGroup.Name)
			observedSecGroup, err := s.reconcileGroupRules(desiredSecGroup, *observedSecGroups[k])
			if err != nil {
				return err
			}
			observedSecGroups[k] = &observedSecGroup
			continue
		}
	}

	openStackCluster.Status.ControlPlaneSecurityGroup = observedSecGroups[ControlPlaneSuffix]
	openStackCluster.Status.WorkerSecurityGroup = observedSecGroups[WorkerSuffix]
	openStackCluster.Status.BastionSecurityGroup = observedSecGroups[BastionSuffix]

	return nil
}

func (s *Service) generateDesiredSecGroups(secGroupNames map[string]string, openStackCluster *infrav1.OpenStackCluster) (map[string]infrav1.SecurityGroup, error) {
	desiredSecGroups := make(map[string]infrav1.SecurityGroup)

	var secControlPlaneGroupID string
	var secWorkerGroupID string
	var secBastionGroupID string
	for i, v := range secGroupNames {
		secGroup, err := s.getSecurityGroupByName(v)
		if err != nil {
			return desiredSecGroups, err
		}
		switch i {
		case ControlPlaneSuffix:
			secControlPlaneGroupID = secGroup.ID
		case WorkerSuffix:
			secWorkerGroupID = secGroup.ID
		case BastionSuffix:
			secBastionGroupID = secGroup.ID
		}
	}

	controlPlaneRules := append(
		[]infrav1.SecurityGroupRule{
			{
				Description:    "Kubernetes API",
				Direction:      "ingress",
				EtherType:      "IPv4",
				PortRangeMin:   6443,
				PortRangeMax:   6443,
				Protocol:       "tcp",
				RemoteIPPrefix: "0.0.0.0/0",
			},
			{
				Description:   "Etcd",
				Direction:     "ingress",
				EtherType:     "IPv4",
				PortRangeMin:  2379,
				PortRangeMax:  2380,
				Protocol:      "tcp",
				RemoteGroupID: remoteGroupIDSelf,
			},
			{
				// kubeadm says this is needed
				Description:   "Kubelet API",
				Direction:     "ingress",
				EtherType:     "IPv4",
				PortRangeMin:  10250,
				PortRangeMax:  10250,
				Protocol:      "tcp",
				RemoteGroupID: remoteGroupIDSelf,
			},
			{
				// This is needed to support metrics-server deployments
				Description:   "Kubelet API",
				Direction:     "ingress",
				EtherType:     "IPv4",
				PortRangeMin:  10250,
				PortRangeMax:  10250,
				Protocol:      "tcp",
				RemoteGroupID: secWorkerGroupID,
			},
			{
				Description:   "BGP (calico)",
				Direction:     "ingress",
				EtherType:     "IPv4",
				PortRangeMin:  179,
				PortRangeMax:  179,
				Protocol:      "tcp",
				RemoteGroupID: remoteGroupIDSelf,
			},
			{
				Description:   "BGP (calico)",
				Direction:     "ingress",
				EtherType:     "IPv4",
				PortRangeMin:  179,
				PortRangeMax:  179,
				Protocol:      "tcp",
				RemoteGroupID: secWorkerGroupID,
			},
			{
				Description:   "IP-in-IP (calico)",
				Direction:     "ingress",
				EtherType:     "IPv4",
				Protocol:      "4",
				RemoteGroupID: remoteGroupIDSelf,
			},
			{
				Description:   "IP-in-IP (calico)",
				Direction:     "ingress",
				EtherType:     "IPv4",
				Protocol:      "4",
				RemoteGroupID: secWorkerGroupID,
			},
		},
		defaultRules...,
	)

	workerRules := append(
		[]infrav1.SecurityGroupRule{
			{
				Description:    "Node Port Services",
				Direction:      "ingress",
				EtherType:      "IPv4",
				PortRangeMin:   30000,
				PortRangeMax:   32767,
				Protocol:       "tcp",
				RemoteIPPrefix: "0.0.0.0/0",
			},
			{
				// This is needed to support metrics-server deployments
				Description:   "Kubelet API",
				Direction:     "ingress",
				EtherType:     "IPv4",
				PortRangeMin:  10250,
				PortRangeMax:  10250,
				Protocol:      "tcp",
				RemoteGroupID: remoteGroupIDSelf,
			},
			{
				Description:   "Kubelet API",
				Direction:     "ingress",
				EtherType:     "IPv4",
				PortRangeMin:  10250,
				PortRangeMax:  10250,
				Protocol:      "tcp",
				RemoteGroupID: secControlPlaneGroupID,
			},
			{
				Description:   "BGP (calico)",
				Direction:     "ingress",
				EtherType:     "IPv4",
				PortRangeMin:  179,
				PortRangeMax:  179,
				Protocol:      "tcp",
				RemoteGroupID: remoteGroupIDSelf,
			},
			{
				Description:   "BGP (calico)",
				Direction:     "ingress",
				EtherType:     "IPv4",
				PortRangeMin:  179,
				PortRangeMax:  179,
				Protocol:      "tcp",
				RemoteGroupID: secControlPlaneGroupID,
			},
			{
				Description:   "IP-in-IP (calico)",
				Direction:     "ingress",
				EtherType:     "IPv4",
				Protocol:      "4",
				RemoteGroupID: remoteGroupIDSelf,
			},
			{
				Description:   "IP-in-IP (calico)",
				Direction:     "ingress",
				EtherType:     "IPv4",
				Protocol:      "4",
				RemoteGroupID: secControlPlaneGroupID,
			},
		},
		defaultRules...,
	)

	if openStackCluster.Spec.Bastion != nil && openStackCluster.Spec.Bastion.Enabled {
		controlPlaneRules = append(controlPlaneRules,
			[]infrav1.SecurityGroupRule{
				{
					Description:   "SSH",
					Direction:     "ingress",
					EtherType:     "IPv4",
					PortRangeMin:  22,
					PortRangeMax:  22,
					Protocol:      "tcp",
					RemoteGroupID: secBastionGroupID,
				},
			}...,
		)
		workerRules = append(workerRules,
			[]infrav1.SecurityGroupRule{
				{
					Description:   "SSH",
					Direction:     "ingress",
					EtherType:     "IPv4",
					PortRangeMin:  22,
					PortRangeMax:  22,
					Protocol:      "tcp",
					RemoteGroupID: secBastionGroupID,
				},
			}...,
		)
		desiredSecGroups[BastionSuffix] = infrav1.SecurityGroup{
			Name: secGroupNames[BastionSuffix],
			Rules: append(
				[]infrav1.SecurityGroupRule{
					{
						Description:    "SSH",
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

	if openStackCluster.Spec.ManagedAPIServerLoadBalancer && !openStackCluster.Spec.UseOctavia {
		neutronLbaasRules := append(
			[]infrav1.SecurityGroupRule{
				{
					Description:    "Kubernetes API",
					Direction:      "ingress",
					EtherType:      "IPv4",
					PortRangeMin:   6443,
					PortRangeMax:   6443,
					Protocol:       "tcp",
					RemoteIPPrefix: "0.0.0.0/0",
				},
			},
			defaultRules...,
		)
		if openStackCluster.Spec.APIServerLoadBalancerAdditionalPorts != nil {
			for _, value := range openStackCluster.Spec.APIServerLoadBalancerAdditionalPorts {
				neutronLbaasRules = append(neutronLbaasRules,
					[]infrav1.SecurityGroupRule{
						{
							Description:    "APIServerLoadBalancerAdditionalPorts",
							Direction:      "ingress",
							EtherType:      "IPv4",
							PortRangeMin:   value,
							PortRangeMax:   value,
							Protocol:       "tcp",
							RemoteIPPrefix: "0.0.0.0/0",
						},
					}...,
				)
			}
		}
		desiredSecGroups[NeutronLbaasSuffix] = infrav1.SecurityGroup{
			Name:  secGroupNames[NeutronLbaasSuffix],
			Rules: neutronLbaasRules,
		}
	}

	desiredSecGroups[ControlPlaneSuffix] = infrav1.SecurityGroup{
		Name:  secGroupNames[ControlPlaneSuffix],
		Rules: controlPlaneRules,
	}

	desiredSecGroups[WorkerSuffix] = infrav1.SecurityGroup{
		Name:  secGroupNames[WorkerSuffix],
		Rules: workerRules,
	}

	return desiredSecGroups, nil
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

// reconcileGroupRules reconciles an already existing observed group by essentially emptying out all the rules and
// recreating them.
func (s *Service) reconcileGroupRules(desired, observed infrav1.SecurityGroup) (infrav1.SecurityGroup, error) {
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

func (s *Service) createSecurityGroupIfNotExists(openStackCluster *infrav1.OpenStackCluster, groupName string) error {
	secGroup, err := s.getSecurityGroupByName(groupName)
	if err != nil {
		return err
	}
	if secGroup == nil || secGroup.ID == "" {
		s.logger.V(6).Info("Group doesn't exist, creating it.", "name", groupName)

		createOpts := groups.CreateOpts{
			Name:        groupName,
			Description: "Cluster API managed group",
		}
		s.logger.V(6).Info("Creating group", "name", groupName)
		_, err := groups.Create(s.client, createOpts).Extract()
		if err != nil {
			return err
		}
		record.Eventf(openStackCluster, "SuccessfulCreateSecurityGroup", "Created security group %s with id %s", groupName, secGroup.ID)
		return nil

	}

	sInfo := fmt.Sprintf("Reuse Existing SecurityGroup %s with %s", groupName, secGroup.ID)
	s.logger.V(6).Info(sInfo)

	return nil
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
		Description:    r.Description,
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
		Description:     osSecGroupRule.Description,
		EtherType:       osSecGroupRule.EtherType,
		SecurityGroupID: osSecGroupRule.SecGroupID,
		PortRangeMin:    osSecGroupRule.PortRangeMin,
		PortRangeMax:    osSecGroupRule.PortRangeMax,
		Protocol:        osSecGroupRule.Protocol,
		RemoteGroupID:   osSecGroupRule.RemoteGroupID,
		RemoteIPPrefix:  osSecGroupRule.RemoteIPPrefix,
	}
}
