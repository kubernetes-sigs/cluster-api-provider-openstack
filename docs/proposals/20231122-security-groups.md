---
title: Flexible managed security sroups
authors:
  - "@emilienm"
reviewers:
  - "@jichenjc"
  - "@lentzi90"
  - "@mdbooth"
creation-date: 2023-11-22
last-updated: 2024-02-21

status: provisional
---

## Title

Flexible managed security groups

## Table of Contents

- [Flexible managed security groups](#title)
  - [Table of Contents](#table-of-contents)
  - [Glossary](#glossary)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals/Future Work](#non-goalsfuture-work)
  - [Proposal](#proposal)
    - [User Stories](#user-stories)
    - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Alternatives](#alternatives)
  - [Upgrade Strategy](#upgrade-strategy)
  - [Additional Details](#additional-details)
    - [Test Plan [optional]](#test-plan-optional)
    - [Graduation Criteria [optional]](#graduation-criteria-optional)
    - [Version Skew Strategy [optional]](#version-skew-strategy-optional)
  - [Implementation History](#implementation-history)

## Glossary

- Egress - Egress traffic is traffic that is leaving the cluster.
- Ingress - Ingress traffic is traffic that is coming into the cluster.
- Security Group - A security group is a named set of rules that control traffic to and from virtual machine instances.
  The rules can specify which ports and protocols to allow, and which IP address ranges to allow traffic to or from.
  Security groups can be associated with multiple instances, and multiple security groups can be associated with a single instance.
  Security groups act as a firewall for associated instances, controlling both inbound and outbound traffic at the instance level.

## Summary

This proposal introduces a new API to manage more security groups and rules.

## Motivation

We want users to be able to not think about the security groups that are needed for their Control Plane and Worker Nodes. However, we want to be able to provide a way for users to have more control over the managed security groups and not have to worry about rules that are not needed by their cluster. They will be able to add additional security groups and rules to the cluster.

### Goals

1. Create a new API to manage more security groups and rules:
   1. Default rules provided by `OpenStackCluster.spec.managedSecurityGroups` will have a new API used to manage additional security groups and rules.
   1. Successfully be able to new manage security groups rules for the Bastion, the Control Plane and the Worker Nodes.
   1. Successfully be able to manage the lifecycle of additional security groups that later can be used in the Machine Template.

### Non-Goals/Future Work

1. Removing pre-existing security groups support in `OpenStackMachineTemplate`, via `OpenStackMachineSpec` and for the Bastion in `OpenStackCluster`.
1. Deprecate `OpenStackCluster.spec.managedSecurityGroups.llowAllInClusterTraffic`, as we'll still need it.

## Proposal

### User Stories

- When creating a cluster as an operator, I don't want to worry about the security groups that are needed for my Control Plane and Worker Nodes to be functional and
  I want these security groups to be managed automatically.

- As an operator, I can already pre-create security groups and provide them to the Machine Templates. However, to remove that burden from the user,
  I now want to let the controller manage the lifecycle of these security groups.
  and have a new API to provide the additional security groups and their rules.

- As an operator, I want to be able to manage additional security group rules for the Bastion, the Control Plane and the Worker Nodes in order to provide more
  rules than the defaults, so I have more flexibility in my cluster.

- As an operator, I want to be able to manage additional security group rules for the CNI so I have more options than Calico. I need to provide my CNI's specific security group rules
  that I need applied to all the machines, so I have more flexibility in my cluster.

### Implementation Details/Notes/Constraints

The plan is to create a new controller and migrate the management of security groups and their rules in there when the `OpenStackCluster.Spec.ManagedSecurityGroups` is set to `True`.

A cluster operator will be able to add additional security groups and rules to the cluster by adding them to the `OpenStackCluster` spec.
The controller will then reconcile the security groups and their rules to the cluster between what's in the spec and what's in the status.
Special care will be taken to not remove security groups that are still in use by the machines.
Also, during an upgrade, the controller will make sure to add the new security groups and their rules to the machines before reducing the default rules
which won't have the Calico rules anymore.

#### Data model changes

The type for `OpenStackCluster.Spec.ManagedSecurityGroups` was a boolean but will become a structure with new fields.

Example `ManagedSecurityGroups` yaml:

```yaml
managedSecurityGroups:
  # Enable the management of security groups with the default rules (kubelet, etcd, etc.)
  # The default stays false
  enabled: true
  # Allow to extend the default security group rules for the Bastion,
  # the Control Plane and the Worker security groups
  additionalBastionSecurityGroupRules:
    - direction: ingress
      ethertype: IPv4
      portRangeMin: 1234
      portRangeMax: 1234
      protocol: tcp
  additionalControlPlaneSecurityGroupRules:
    - direction: ingress
      ethertype: IPv4
      portRangeMin: 1234
      portRangeMax: 1234
      protocol: tcp
  additionalWorkerSecurityGroupRules:
    - direction: ingress
      ethertype: IPv4
      portRangeMin: 1234
      portRangeMax:1234
      protocol: tcp
  # Allow to provide rules that will be applied to all nodes
  allNodesSecurityGroupRules:
    - direction: ingress
      ethertype: IPv4
      portRangeMin: 1234
      portRangeMax: 1234
      protocol: tcp
  # Allow to provide additional security groups and rules that can be used in the Machine Template
  additionalSecurityGroups:
    - name: my-security-group
      description: My security group
      rules:
        - direction: ingress
          ethertype: IPv4
          portRangeMin: 1234
          portRangeMax: 1234
          protocol: tcp
  # When set to `True`, the controller will add the Calico rules to the All nodes security group and update
  # the machines before the default rules are removed from the managed security groups
  useLegacyCalicoRules: false
```

#### Enable or disable the management of security groups

`OpenStackCluster.Spec.ManagedSecurityGroups.enabled` will be used to enable or disable the management of security groups in general.
It was a boolean and will now be a structure with new fields.

Conversion will be done automatically for the user.

#### Additional security rules

`OpenStackCluster.Spec.ManagedSecurityGroups.additionalBastionSecurityGroupRules` will be used to add additional security rules to the Bastion security group.
`OpenStackCluster.Spec.ManagedSecurityGroups.additionalControlPlaneSecurityGroupRules` will be used to add additional security rules to the Control Plane security group.
`OpenStackCluster.Spec.ManagedSecurityGroups.additionalWorkerSecurityGroupRules` will be used to add additional security rules to the Worker security group.

Theses rules will be managed by the new controller and added to respectively the Bastion, Control Plane and Worker security groups status.
A user will be able to add or remove rules from the spec and the controller will reconcile the status accordingly and update the security groups.

#### Additional security groups

`OpenStackCluster.Spec.ManagedSecurityGroups.additionalSecurityGroups` will be used to add additional security groups and their rules.
The security groups will be managed by the new controller and added to the status.
A user will be able to add or remove security groups from the spec and the controller will reconcile the status accordingly and update the security groups.

The controller will also ensure that the security groups are not removed if they are still in use by the machines and add them to the machines if they are not already there.

#### All nodes security group rules

`OpenStackCluster.Spec.ManagedSecurityGroups.allNodesSecurityGroupRules` will be used to add additional security rules to the a new security group applied to all nodes.
The rules will be managed by the new controller and added to the All ndoes security group status.
A user will be able to add or remove rules from the spec and the controller will reconcile the status accordingly and update the All nodes security group.

#### Migration path for Calico users

`OpenStackCluster.Spec.ManagedSecurityGroups.useLegacyCalicoRules` will be used to add the Calico rules to the All nodes security group and update the
machines before the default rules are removed from the managed security groups. This parameter might be removed in the future.

### Risks and Mitigations

Example risks:

- The new controller might not be able to manage the security groups and their rules correctly during an upgrade.

## Alternatives

No currently known alternatives exist which are public and have been implemented for CAPO.

## Upgrade Strategy

The upgrade strategy will be to add the new controller and migrate the management of security groups and their default rules in there when the `managedSecurityGroups.enabled` is set to `True`.
For Calico users, the `useLegacyCalicoRules` will be used to add the Calico rules to the All nodes security group and update the machines before the default rules are removed from the managed security groups.

Since we are changing the type of `OpenStackCluster.Spec.ManagedSecurityGroups` from a boolean to a structure, we will have to add a conversion function to convert the boolean to the structure.

## Additional Details

### Test Plan

- The new controller will be tested with unit tests.
- Unit tests will be added to the security group reconciler to make sure that the security groups and their rules are managed correctly.
- The exising e2e tests will help to make sure that the new controller is working as expected and no regression is introduced.

### Version Skew Strategy

The feature itself should not depend significantly on the version of CAPI and will be backwards compatible with old versions of CAPO since it will be adding new options. If there is a drift in CAPI and CAPO versions, the functionality should stay the same without breaking anything.

## Implementation History

- [ ] 11/21/2023: Open WIP PR [ https://github.com/kubernetes-sigs/cluster-api-provider-openstack/pull/1751 ]
- [ ] 11/22/2023: Open this KEP PR
