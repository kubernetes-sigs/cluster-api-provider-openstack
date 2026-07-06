# Configurable default outbound traffic for managed security groups

## Metadata

* Authors: @alejandro-ripoll
* Reviewers: CAPO maintainers
* Status: Proposed
* Creation Date: 2026-07-06
* Issue: kubernetes-sigs/cluster-api-provider-openstack#3209

## Summary

CAPO managed security groups currently include default egress rules that allow all outbound IPv4 and IPv6 traffic. This proposal adds a small opt-out field to `ManagedSecurityGroups` so users can keep CAPO-managed security groups while removing only the unrestricted outbound rules.

The proposed API is additive and preserves the current behavior by default.

## Motivation

Some environments do not allow unrestricted outbound traffic from cluster nodes. Today, users who need restricted egress cannot keep the managed security group lifecycle while removing the default allow-all egress rules.

This is especially relevant for environments with strict network segmentation, compliance requirements, or zero-trust policies. These users still benefit from CAPO creating and reconciling the security groups and the required Kubernetes rules, but they need control over outbound destinations.

CAPO managed security group rules are not all the same kind of rule:

* Infrastructure rules required for normal cluster operation, such as etcd, kubelet, API server, NodePort, and bastion SSH.
* Permissive default outbound rules that allow all IPv4 and IPv6 egress.

This proposal targets only the second category. Required infrastructure rules remain managed by CAPO.

### Goals

* Preserve the current default behavior.
* Allow users to disable only the default allow-all outbound rules.
* Keep required infrastructure rules managed by CAPO.
* Reuse the existing `SecurityGroupRuleSpec` API for explicit egress rules.
* Keep the change limited to `ManagedSecurityGroups`.
* Avoid requiring users to disable `managedSecurityGroups`.

### Non-Goals

* Define a higher-level egress policy API.
* Manage external firewall rules, routers, NAT, proxies, or DNS.
* Validate whether the configured egress rules are sufficient for a working Kubernetes cluster.
* Change existing required infrastructure rules.
* Change behavior when `managedSecurityGroups` is not used.
* Add a generic flag to disable all predefined rules.

## User Stories

### Story 1: Restricted egress

As a platform engineer, I want CAPO to manage security groups for a cluster, but I want to restrict node egress to a known set of CIDRs instead of allowing `0.0.0.0/0` and `::/0`.

### Story 2: Existing managed security group behavior

As an existing CAPO user, I want current clusters and manifests to keep the same behavior unless I explicitly opt out of the default egress rules.

### Story 3: Gradual adoption

As an operator, I want to first disable the default egress rules in non-production clusters and add only the egress rules my environment requires.

## Proposal

Add one optional field to `ManagedSecurityGroups`:

```go
type ManagedSecurityGroups struct {
  // AllowAllOutboundTraffic controls whether CAPO adds the default
  // allow-all outbound rules to managed security groups.
  //
  // When nil or true, CAPO keeps the current behavior and creates default
  // IPv4 and IPv6 allow-all egress rules.
  //
  // When false, CAPO does not create those default allow-all egress rules.
  // Other CAPO-managed infrastructure rules are still created.
  //
  // +kubebuilder:default=true
  // +optional
  AllowAllOutboundTraffic *bool `json:"allowAllOutboundTraffic,omitempty"`
}
```

Behavior:

* If `managedSecurityGroups` is omitted, behavior is unchanged.
* If `managedSecurityGroups` is set to an empty object, behavior is unchanged.
* If `allowAllOutboundTraffic` is unset or `true`, CAPO creates the current default allow-all IPv4 and IPv6 egress rules.
* If `allowAllOutboundTraffic` is `false`, CAPO skips only the default allow-all egress rules.
* Required infrastructure rules remain unchanged.
* User-defined egress rules continue to be reconciled normally.
* The same allow-all outbound behavior is applied consistently to the node security groups and the bastion security group.

## API Changes

The change is additive:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OpenStackCluster
metadata:
  name: restricted-egress
spec:
  managedSecurityGroups:
    allowAllOutboundTraffic: false
    allNodesSecurityGroupRules:
    - name: allow-dns
      direction: egress
      etherType: IPv4
      protocol: udp
      portRangeMin: 53
      portRangeMax: 53
      remoteIPPrefix: 10.0.0.53/32
    - name: allow-https-to-proxy
      direction: egress
      etherType: IPv4
      protocol: tcp
      portRangeMin: 443
      portRangeMax: 443
      remoteIPPrefix: 10.0.10.0/24
```

No new status fields are required.

## Controller Design

CAPO already builds managed security group rules before merging user-provided rules. This logic should keep required infrastructure rules separate from permissive allow-all outbound rules.

The controller should:

1. Keep required infrastructure rules enabled whenever managed security groups are enabled.
2. Keep the existing allow-all outbound rules for backward compatibility when the new field is unset or true.
3. Add a helper that decides whether the permissive outbound rules should be included:

```go
func allowAllOutboundTraffic(managedSecurityGroups *infrav1.ManagedSecurityGroups) bool {
    return managedSecurityGroups == nil ||
        managedSecurityGroups.AllowAllOutboundTraffic == nil ||
        *managedSecurityGroups.AllowAllOutboundTraffic
}
```

4. When building managed security group rules, include the current default allow-all egress rules only if `allowAllOutboundTraffic(...)` returns `true`.
5. Continue applying all user-provided `SecurityGroupRuleSpec` entries unchanged.
6. Apply the same behavior to the bastion security group so disabling allow-all outbound traffic does not leave a separate unrestricted egress path there.

The implementation should avoid a generic `enablePredefinedRules` style flag. Such a flag would also disable required infrastructure rules and force users to recreate them by hand, which is not the goal of this change.

## Backward Compatibility

This proposal is backward compatible.

Existing manifests do not need to change. Existing users keep the current unrestricted egress behavior because `allowAllOutboundTraffic` defaults to enabled when omitted.

The new behavior is only used when users explicitly set:

```yaml
managedSecurityGroups:
  allowAllOutboundTraffic: false
```

## Validation

No special validation is required for `allowAllOutboundTraffic`.

Existing validation for `SecurityGroupRuleSpec` continues to apply to explicit egress rules. In particular, users are responsible for providing valid `direction`, `etherType`, protocol, port range, and remote fields.

## Testing

Add unit tests for managed security group rule generation:

* `allowAllOutboundTraffic` unset keeps default IPv4 and IPv6 allow-all egress rules.
* `allowAllOutboundTraffic: true` keeps default IPv4 and IPv6 allow-all egress rules.
* `allowAllOutboundTraffic: false` removes only default IPv4 and IPv6 allow-all egress rules.
* User-defined egress rules are still included when `allowAllOutboundTraffic: false`.
* Existing required infrastructure rules are not changed.
* Bastion security group behavior follows the same setting.

## Documentation

Update the managed security groups documentation with:

* The default behavior.
* The new `allowAllOutboundTraffic` field.
* A minimal restricted-egress example.
* A note that users must provide any egress required by their environment, such as DNS, image registry, package mirror, metadata service, or API endpoints.
* A note that CAPO still manages required infrastructure rules when default outbound traffic is disabled.

## Risks and Mitigations

Risk: Users may disable default outbound traffic without adding the outbound rules required for node bootstrap or normal cluster operation.

Mitigation: Document the behavior clearly and keep it opt-in. CAPO still manages the infrastructure rules it already owns.

Risk: Naming could be interpreted as a full egress policy feature.

Mitigation: Keep the field scoped to the existing default allow-all outbound rules. Explicit custom rules remain modeled through `SecurityGroupRuleSpec`.

## Alternatives

### Add explicit egress CIDR fields

CAPO could add fields such as `egressCIDRs` or `defaultEgressCIDRs`.

This is not proposed because `SecurityGroupRuleSpec` already supports explicit egress rules, including `remoteIPPrefix`, protocol, and ports. Adding a second egress-specific API would duplicate existing functionality.

### Disable all default rules

CAPO could add a field to disable every default managed security group rule.

This is broader than needed. The requested use case is specifically about the default unrestricted egress rules. Disabling all predefined rules would also remove required infrastructure rules such as etcd, kubelet, API server, NodePort, and bastion SSH. That would force users to re-author rules that CAPO already knows how to manage.

### Require unmanaged security groups

Users can avoid CAPO defaults by not using `managedSecurityGroups` and managing security groups themselves.

This works, but it removes the managed lifecycle that users still want. It also increases the amount of OpenStack-specific setup required outside CAPO.
