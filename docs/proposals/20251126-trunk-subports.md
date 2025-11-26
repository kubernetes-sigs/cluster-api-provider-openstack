# Trunk Subports Support for OpenStackMachine

## Metadata

- **Authors**: @bnallapeta
- **Reviewers**: CAPO maintainers
- **Status**: Proposed
- **Creation Date**: 2025-11-26
- **Last Updated**: 2025-11-26

## Summary

This proposal introduces support for defining trunk subports (e.g., VLANs) directly within the `OpenStackMachine` resource. This allows users to configure complex networking topologies where a single machine interface (trunk) carries traffic for multiple networks using segmentation (VLANs).

## Motivation

### Goals
- Enable users to define subports for trunk ports in `OpenStackMachine`
- Support VLAN segmentation for subports
- Maintain backward compatibility with existing port configurations
- Follow OpenStack Networking (Neutron) trunk API patterns

### Non-Goals
- Support for segmentation types other than VLAN (initially)
- Dynamic modification of subports on running machines (immutable for now)

### User Stories

#### Story 1: NFV Workloads
As a user deploying Network Function Virtualization (NFV) workloads, I need to attach my VM to multiple VLANs through a single trunk interface to separate control plane and data plane traffic without consuming multiple physical interfaces or PCI slots.

#### Story 2: Multi-Tenant Networking
As a platform administrator, I want to provision worker nodes that can connect to multiple tenant networks via VLANs on a single trunk port, allowing efficient network isolation and usage.

### API Design

#### New SubportOpts Struct

We introduce `SubportOpts` to define the properties of a subport.

```go
type SubportOpts struct {
    // SegmentationID is the segmentation ID of the subport. E.g. VLAN ID.
    // +required
    // +kubebuilder:validation:Minimum:=1
    // +kubebuilder:validation:Maximum:=4094
    SegmentationID int `json:"segmentationID"`

    // SegmentationType is the segmentation type of the subport. E.g. "vlan".
    // +required
    // +kubebuilder:validation:Enum=vlan;flat
    SegmentationType string `json:"segmentationType"`

    // Port contains parameters of the port associated with this subport
    CommonPortOpts `json:",inline"`
}
```

#### Updated PortOpts

The existing `PortOpts` struct is updated to include a list of subports.

```go
type PortOpts struct {
    // ... existing fields ...

    // Subports is a list of port specifications that will be created as
    // subports of the trunk.
    // +optional
    // +listType=atomic
    Subports []SubportOpts `json:"subports,omitempty"`
}
```

### Implementation Details

#### Networking Service Updates

The `pkg/cloud/services/networking` package will be updated to handle subport creation and attachment.

1.  **`EnsureTrunkSubPorts` Method**:
    - Iterates through the desired ports.
    - If `trunk: true` is set, it checks for `Subports`.
    - For each subport:
        - Creates a regular Neutron port using `CommonPortOpts`.
        - Adds the port as a subport to the parent trunk using the `AddSubports` API, specifying the `SegmentationID` and `SegmentationType`.

2.  **Lifecycle Management**:
    - Subports are created after the parent port and trunk are created.
    - Deletion of the `OpenStackMachine` will naturally clean up the ports, but explicit cleanup logic for trunks and subports ensures no orphaned resources.

### Backward Compatibility

- **Compatible**: Existing `OpenStackMachine` manifests without `subports` will continue to work as before.
- **Optional**: The `subports` field is optional.

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| **Orphaned Subports** | Ensure deletion logic explicitly handles subport removal if the trunk deletion doesn't cascade (though Neutron usually handles this). |
| **Quota Limits** | Creating many subports consumes Neutron port quotas. Users should be aware of their quota limits. |
| **Complexity** | Trunking adds complexity to network debugging. Clear documentation and examples will be provided. |

### Alternatives Considered

- **Manual Configuration**: Users could manually create trunks and subports using the OpenStack CLI and reference them by ID.
    - *Pros*: No code changes in CAPO.
    - *Cons*: Breaks the "Infrastructure as Code" model; manual steps are error-prone and hard to automate in CAPI workflows.
- **Separate CRD**: Introduce a new CRD for Trunks/Subports.
    - *Pros*: Decouples networking from Machine.
    - *Cons*: Increases API surface area and complexity for common use cases. Embedding in `OpenStackMachine` is more ergonomic for the 90% case.

### Security Considerations

- **Network Isolation**: Subports allow a VM to access multiple networks. Administrators must ensure that the `OpenStackMachine` spec (and the user creating it) has the appropriate permissions to attach to those networks.
- **VLAN Hopping**: Proper Neutron configuration prevents VLAN hopping, but users must ensure they are using valid segmentation IDs authorized for their tenant.

### Testing Strategy

- **Unit Tests**: Verify `EnsureTrunkSubPorts` logic, including correct API calls to OpenStack for subport addition.
- **E2E Tests**: Create an `OpenStackMachine` with trunk and subports, verify connectivity or presence of VLAN interfaces inside the instance (if possible) or verify API state.

## Example Usage

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OpenStackMachineTemplate
spec:
  template:
    spec:
      ports:
        - network:
            filter:
              name: trunk-network
          trunk: true
          subports:
            - segmentationID: 101
              segmentationType: vlan
              network:
                filter:
                  name: vlan-101-network
              fixedIPs:
                - subnet:
                    filter:
                      name: vlan-101-subnet
```
