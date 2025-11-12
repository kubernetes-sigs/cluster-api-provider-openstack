# Multi-AZ API Server LoadBalancer for CAPO

## Summary
Add first-class Multi-AZ support for the Kubernetes control plane LoadBalancer in Cluster API Provider OpenStack (CAPO). The feature reconciles one Octavia LoadBalancer per Availability Zone (AZ), places each VIP in the intended subnet for that AZ via an explicit AZ→Subnet mapping, and by default registers control plane nodes only with the LB in the same AZ. Operators expose the control plane endpoint via external DNS multi-value A records that point at the per-AZ LB IPs. This proposal is additive and backward compatible.

## Motivation
- Achieve true multi-AZ resilience for the control plane by avoiding a single VIP dependency.
- Align control plane networking with existing multi-AZ compute placement goals.
- Provide clear, portable primitives across Octavia providers with native AZ hints and an explicit, unambiguous mapping between AZs and VIP subnets.

## Goals
- Create and manage one API server LoadBalancer per configured AZ.
- Support explicit AZ→Subnet mapping only (no positional mapping).
- Default to same-AZ LB membership for control plane nodes; allow opt-in cross-AZ registration.
- Keep the API additive with strong validation, clear events and documentation.
- Preserve user-provided DNS endpoints; DNS record management remains out of scope.

## Non-Goals
- Managing or provisioning DNS records.
- Provider-specific topologies such as ACTIVE_STANDBY across fault domains.
- Service type LoadBalancer for worker Services.

## User Stories
1) As a platform engineer, I want per-AZ LBs so a full AZ outage leaves the cluster reachable via DNS multi-A records that resolve to the remaining AZs.
2) As an operator, I want a safe migration path from single-LB clusters to per-AZ LBs without downtime.
3) As a security-conscious user, I want to restrict VIP access with allowed CIDRs when supported by my Octavia provider.

## Design Overview

### High-level behavior
- When enabled and configured with an explicit mapping, CAPO reconciles one LoadBalancer per Availability Zone (AZ).
- VIP placement is controlled only by an explicit mapping list that binds each AZ to a specific subnet on the LB network.
- Each per-AZ LB is named with an AZ suffix.
- Control plane nodes are registered as LB members only in their AZ by default; opt-in cross-AZ membership is supported.
- Operators expose an external DNS name for the control plane endpoint with one A/AAAA record per AZ LB IP.

### Architecture diagram
```mermaid
flowchart LR
  Clients --> DNS[External DNS zone]
  DNS -->|A record per AZ| LBa[LB az1]
  DNS -->|A record per AZ| LBb[LB az2]
  DNS -->|A record per AZ| LBn[LB azN]
  subgraph OpenStack
    LBa --> LaL[Listeners] --> Pa[Pools] --> CP1[Control plane nodes in az1]
    LBb --> LbL[Listeners] --> Pb[Pools] --> CP2[Control plane nodes in az2]
    LBn --> LnL[Listeners] --> Pn[Pools] --> CPn[Control plane nodes in azN]
  end
```

## Integration with External Global Server Load Balancing (GSLB)

External GSLB systems (e.g., Route 53 health-checked records, Akamai GTM, Cloudflare Load Balancing, NS1, F5 DNS/GTM) pair naturally with this Multi-AZ LB design:

- Clear targets: Each AZ has its own LB with a stable IP (floating IP or provider VIP) and deterministic name. These per-AZ endpoints are ideal GSLB health-check targets.
- Health-aware failover: GSLB continuously probes each per-AZ LB (TCP 6443 or an alternative port configured via additionalPorts) and automatically removes unhealthy AZ endpoints from DNS responses.
- Improved blast-radius isolation: An AZ outage only affects the corresponding AZ LB. GSLB maintains service by answering with remaining healthy AZ LB IPs.
- Policy flexibility: GSLB policies (failover, weighted round-robin, latency/geo) can prefer:
  - Same-region/same-AZ endpoints for lowest latency
  - Spillover to other AZs only on failure
  - Weighted distribution across AZs for capacity utilization

Recommended GSLB patterns
- Record model: Use a single control plane FQDN (the cluster’s spec.controlPlaneEndpoint.Host) and publish multiple A/AAAA records—one per AZ LB IP.
- Health checks:
  - Protocol: TCP on the API port (default 6443). For providers that support L7 checks, TCP is generally sufficient for the Kubernetes API.
  - Source IPs: Ensure GSLB checker IPs are permitted if using allowedCIDRs on listeners.
- TTL guidance:
  - Use low TTL (e.g., 30–60s) to accelerate failover while balancing resolver load.
  - Be aware that some clients cache beyond TTL; plan operationally for a brief grace period during failover.
- IP sourcing:
  - Floating IPs typically simplify routing and are stable across LB re-creation.
  - If using fixed VIPs (no floating), ensure they are routable to your GSLB health-check network and external resolvers that must reach them.
- Automation hooks:
  - Deterministic LB naming (per-AZ suffix) and tags facilitate discovery by GSLB automation to register/update record sets.
  - A controller or out-of-band job can list per-AZ LBs and synchronize GSLB records and health checks.

Failure scenarios and behavior
- Single AZ failure: The corresponding per-AZ LB becomes unhealthy; GSLB health checks fail; DNS answers exclude that AZ until recovery. Existing connections may break depending on client TCP retry behavior; new connections will target healthy AZs.
- Partial AZ degradation (e.g., only some members or monitor thresholds): Octavia monitor status influences LB health; ensure GSLB health thresholds align with Octavia monitor sensitivity to avoid premature removal or flapping.
- Network partitions from health-check vantage points:
  - If GSLB checkers reside outside the cloud, confirm egress paths to per-AZ IPs and allowedCIDRs permit probes from those checkers.
  - Consider diverse checker regions to avoid false positives due to upstream routing issues.

Operational considerations
- Access control: When using allowedCIDRs, include:
  - Management cluster egress IPs (so CAPO can reconcile listeners/pools/monitors)
  - Bastion/router IPs as needed for administration
  - GSLB health-check source IP ranges
- Observability:
  - Track per-AZ LB health and GSLB health check status together to diagnose discrepancies (LB marked healthy, but GSLB marks unhealthy often indicates ACL/routing issues).
- Multi-region future: This proposal focuses on multi-AZ within a region. If multi-region is introduced later, the same per-AZ model composes naturally: per-AZ LBs per region, with GSLB distributing across regions using latency- or geo-based policies and regional failover priorities.

This integration enables operators to achieve health-aware, low-latency, and failure-tolerant access to the Kubernetes API without CAPO managing DNS, while leveraging the explicit per-AZ LB separation for precise GSLB control.

## API Changes (additive)

All changes are confined to the OpenStackCluster API and are backward compatible. Proposed changes in:
- [api/v1beta1/openstackcluster_types.go](api/v1beta1/openstackcluster_types.go)
- [api/v1beta1/types.go](api/v1beta1/types.go)

### Spec additions on APIServerLoadBalancer
- availabilityZoneSubnets []AZSubnetMapping (required to enable multi-AZ)
  - Explicit mapping; each entry includes:
    - availabilityZone string
    - subnet SubnetParam
  - The LB network MUST be specified when using this mapping via spec.apiServerLoadBalancer.network. Each mapped subnet MUST belong to that network.
- allowCrossAZLoadBalancerMembers *bool
  - Default false.
  - When true, register control plane nodes to all per-AZ LBs; otherwise same-AZ only.
- additionalPorts []int
  - Optional extra listener ports besides the Kubernetes API port.
- allowedCIDRs []string
  - Optional VIP ACL list when supported by the Octavia provider.

Notes:
- The existing single-value availabilityZone field (if present) is treated as a legacy single-AZ shorthand; multi-AZ requires availabilityZoneSubnets.

### Status additions
- apiServerLoadBalancers []LoadBalancer
  - A list-map keyed by availabilityZone (kubebuilder listMapKey=availabilityZone).
  - Each entry includes: name, id, ip, internalIP, tags, availabilityZone, loadBalancerNetwork, allowedCIDRs.

### Validation (CRD and controller)
- No duplicate availabilityZone values in availabilityZoneSubnets.
- Each availabilityZoneSubnets.subnet MUST resolve to a subnet that belongs to the specified LB network.
- No duplicate subnets across mappings.
- At least one mapping is required to enable multi-AZ; otherwise behavior is legacy single-LB.

CRD updates in:
- [config/crd/bases/](config/crd/bases/)
- [config/crd/patches/](config/crd/patches/)

## Controller Design

Changes span these components:
- [controllers/openstackcluster_controller.go](controllers/openstackcluster_controller.go)
- [pkg/cloud/services/loadbalancer/](pkg/cloud/services/loadbalancer/)
- [pkg/cloud/services/networking/](pkg/cloud/services/networking/)

### VIP network and subnet resolution
- When spec.apiServerLoadBalancer.network is specified with availabilityZoneSubnets:
  - Resolve each SubnetParam in order; validate that each belongs to the given LB network.
  - Derive the AZ list directly from the mapping entries.
  - Persist the LB network and the ordered subnets into status.apiServerLoadBalancer.loadBalancerNetwork.
- Legacy single-AZ behavior (no mapping provided):
  - If an LB network is specified but no mapping is provided, treat as single-LB and select a subnet per legacy rules (unchanged).
  - If no LB network is specified, default to the cluster network’s subnets (unchanged single-LB behavior).

Initialize or update status.apiServerLoadBalancers entries to carry the LB network reference.

### Per-AZ LoadBalancer reconciliation
For each AZ in availabilityZoneSubnets:
- Determine the VIP subnet from the mapping and create or adopt a LoadBalancer named:
  - k8s-clusterapi-cluster-${NAMESPACE}-${CLUSTER_NAME}-${AZ}-kubeapi
- Set Octavia AvailabilityZone hint when supported by the provider.
- Create or adopt listeners, pools, and monitors for the API port and any additionalPorts.
- If floating IPs are not disabled, allocate and associate a floating IP to the LB VIP port when needed.
- Update or insert the AZ entry in status.apiServerLoadBalancers, including name, id, internalIP, optional ip, tags, allowedCIDRs, and loadBalancerNetwork.

### Legacy adoption and migration
- Discover legacy single-LB resources named:
  - k8s-clusterapi-cluster-${NAMESPACE}-${CLUSTER_NAME}-kubeapi
- When multi-AZ is enabled (availabilityZoneSubnets provided), rename legacy resources to the AZ-specific name for the first configured AZ, or adopt correctly named resources if they already exist.
- Emit clear events and warnings; ensure idempotent operation.

### Member registration behavior
- Determine the machine failure domain (AZ) from the owning control plane machine.
- Default behavior: register the node only with the LoadBalancer whose availabilityZone matches the node’s AZ; if the legacy LB exists without an AZ, include it as a fallback.
- When allowCrossAZLoadBalancerMembers is true: register the node with all per-AZ LBs.
- Reconcile membership across the API port and any additionalPorts.

### Control plane endpoint
- Preserve a user-provided DNS in spec.controlPlaneEndpoint when set and valid.
- Otherwise choose:
  - The LB floating IP if present, else the VIP for an LB.
  - If no LB host is available and floating IPs are allowed, allocate or adopt a floating IP for the cluster endpoint when applicable.
  - If floating IPs are disabled and a fixed IP is provided, use it.
- Operators are expected to configure DNS with one A/AAAA record per AZ LB IP for client-side failover. CAPO does not manage DNS.

### Events and metrics
- Emit events for create/update/delete of LBs, listeners, pools, monitors, and floating IPs.
- Emit warnings when provider features are unavailable or when validations fail.
- Optional metrics (non-breaking) for per-AZ LB counts and reconciliation latency.

## Example configurations

Explicit AZ→Subnet mapping (required for multi-AZ)
```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OpenStackCluster
metadata:
  name: my-cluster
  namespace: default
spec:
  apiServerLoadBalancer:
    enabled: true
    network:
      id: 6c90b532-7ba0-418a-a276-5ae55060b5b0
    availabilityZoneSubnets:
      - availabilityZone: az1
        subnet:
          id: cad5a91a-36de-4388-823b-b0cc82cadfdc
      - availabilityZone: az2
        subnet:
          id: e2407c18-c4e7-4d3d-befa-8eec5d8756f2
    allowCrossAZLoadBalancerMembers: false
```

Allow cross-AZ member registration
```yaml
spec:
  apiServerLoadBalancer:
    enabled: true
    network:
      id: 6c90b532-7ba0-418a-a276-5ae55060b5b0
    availabilityZoneSubnets:
      - availabilityZone: az1
        subnet:
          id: cad5a91a-36de-4388-823b-b0cc82cadfdc
      - availabilityZone: az2
        subnet:
          id: e2407c18-c4e7-4d3d-befa-8eec5d8756f2
    allowCrossAZLoadBalancerMembers: true
```

Restrict access using allowed CIDRs
```yaml
spec:
  apiServerLoadBalancer:
    enabled: true
    network:
      id: 6c90b532-7ba0-418a-a276-5ae55060b5b0
    availabilityZoneSubnets:
      - availabilityZone: az1
        subnet:
          id: cad5a91a-36de-4388-823b-b0cc82cadfdc
      - availabilityZone: az2
        subnet:
          id: e2407c18-c4e7-4d3d-befa-8eec5d8756f2
    allowedCIDRs:
      - 192.0.2.0/24
      - 203.0.113.10
```

## Backward compatibility and migration

- Default behavior remains single-LB when no multi-AZ mapping is provided.
- Enabling multi-AZ:
  - Operators add availabilityZoneSubnets (and optionally additionalPorts, allowedCIDRs, allowCrossAZLoadBalancerMembers) and must specify the LB network.
  - Controller renames or adopts legacy resources into AZ-specific naming.
  - status.apiServerLoadBalancers is populated alongside legacy status until further cleanup.
- Disabling multi-AZ:
  - Remove the mapping; controller maintains single-LB behavior.
  - Per-AZ LBs are not automatically deleted; operators may clean up unused resources.

## Testing strategy

Unit tests
- Validation: duplicate AZs, duplicate subnets in mapping, wrong network-subnet associations.
- LB reconciliation: AZ hint propagation, per-port resource creation and updates.
- Migration/adoption: renaming legacy resources and adopting correctly-named resources.
- Member registration: defaults and cross-AZ opt-in.
- Allowed CIDRs: canonicalization and provider capability handling.

E2E tests
- Multi-AZ suite to verify per-AZ LBs exist with expected names and ports.
- status.apiServerLoadBalancers contains per-AZ entries including LB network and IPs.
- Control plane nodes register to same-AZ LB (or to all LBs when cross-AZ is enabled).
- DNS records remain out of scope for e2e.

Test code locations:
- [pkg/cloud/services/loadbalancer/](pkg/cloud/services/loadbalancer/)
- [controllers/](controllers/)
- [test/e2e/](test/e2e/)

## Risks and mitigations
- Mapping/network mismatches: reject with clear validation messages; enforce via CRD CEL where feasible and in-controller checks.
- Providers ignoring AZ hints: VIP subnet mapping still ensures deterministic placement; document expected variance.
- Increased resource usage: multiple LBs per cluster increase quota consumption; highlight in docs and operations guidance.
- DNS misconfiguration: documented as operator responsibility.

## Rollout plan
1) API and CRD changes:
   - Add new fields and list-map keyed status to OpenStackCluster types in [api/v1beta1/](api/v1beta1/).
   - Update CRDs in [config/crd/bases/](config/crd/bases/) and patches in [config/crd/patches/](config/crd/patches/).
2) Controller implementation:
   - VIP network/subnet resolution and explicit AZ mapping in [controllers/openstackcluster_controller.go](controllers/openstackcluster_controller.go).
   - Per-AZ LB reconciliation, rename/adoption, member selection, and optional floating IPs in [pkg/cloud/services/loadbalancer/](pkg/cloud/services/loadbalancer/).
3) Documentation:
   - Update configuration guide and examples in [docs/book/src/clusteropenstack/configuration.md](docs/book/src/clusteropenstack/configuration.md).
4) Testing:
   - Unit tests across controller and services; e2e suite updates in [test/e2e/](test/e2e/).
5) Optional metrics:
   - Add observability for per-AZ LB counts and reconciliation timings (non-breaking).

## Open questions
- Should we add a future explicit field to declare the endpoint strategy (single VIP vs external DNS multi-A)? Current design preserves user-provided DNS and documents multi-A.