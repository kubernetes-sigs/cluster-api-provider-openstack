# OpenStackClusterIdentity for Centralized Credential Management

## Metadata

- **Authors**: @bnallapeta
- **Reviewers**: CAPO maintainers
- **Status**: Proposed
- **Creation Date**: 2025-07-22
- **Last Updated**: 2025-07-22

## Summary

This proposal introduces `OpenStackClusterIdentity`, a cluster-scoped resource for centralized OpenStack credential management in CAPO. This enables multi-tenant environments to share credentials across namespaces while maintaining proper access controls, following patterns from AWS and Azure Cluster API providers.

## Motivation

### Goals
- Enable centralized storage of OpenStack credentials in cluster-scoped resources
- Provide fine-grained namespace access controls for credential usage
- Maintain 100% backward compatibility with existing OpenStackCluster resources
- Support manual, gradual migration without breaking existing deployments
- Follow established patterns from other Cluster API providers (AWS, Azure)

### Non-Goals
- Automatic migration of existing deployments
- Integration with external secret management systems
- Breaking changes to existing API or functionality

### User Stories

#### Story 1: Platform Administrator
As a platform administrator managing multiple tenant namespaces, I want to store OpenStack credentials centrally in a secure namespace (e.g., `capo-system`), control which tenant namespaces can use specific credentials, and rotate credentials in one place without updating every namespace.

#### Story 2: Tenant User  
As a tenant user in namespace `team-a`, I want to create OpenStack clusters using centrally managed credentials without managing OpenStack secrets in my namespace, with clear error messages if I don't have permission to use specific credentials.

#### Story 3: Multi-Region Setup
As an administrator managing clusters across multiple OpenStack regions, I want to create region-specific cluster identities with appropriate credentials and allow tenants to use different regional credentials based on their needs.

### API Design

#### New OpenStackClusterIdentity Resource (Cluster-scoped)

```go
type OpenStackClusterIdentity struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec   OpenStackClusterIdentitySpec   `json:"spec,omitempty"`
    Status OpenStackClusterIdentityStatus `json:"status,omitempty"`
}

type OpenStackClusterIdentitySpec struct {
    // SecretRef references the secret containing OpenStack credentials
    SecretRef OpenStackCredentialSecretReference `json:"secretRef"`

    // AllowedNamespaces defines which namespaces can use this identity
    // +optional
    AllowedNamespaces []string `json:"allowedNamespaces,omitempty"`

    // NamespaceSelector selects allowed namespaces via labels
    // +optional
    NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`
}

type OpenStackCredentialSecretReference struct {
    Name      string `json:"name"`
    Namespace string `json:"namespace"`
}

type OpenStackClusterIdentityStatus struct {
    Ready bool `json:"ready"`
    Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

type OpenStackClusterIdentityReference struct {
    Name string `json:"name"`
}
```

**Enhanced OpenStackCluster:**
```go
type OpenStackClusterSpec struct {
    // ... existing fields unchanged ...

    // ClusterIdentityRef references a cluster-scoped identity (NEW)
    // Takes precedence over IdentityRef if specified
    // +optional
    ClusterIdentityRef *OpenStackClusterIdentityReference `json:"clusterIdentityRef,omitempty"`

    // IdentityRef references namespace-local secret (EXISTING - unchanged)
    // +kubebuilder:validation:Required  
    IdentityRef OpenStackIdentityReference `json:"identityRef"`
}
```

### Implementation Details

#### Credential Resolution Logic
The scope factory will implement dual-path credential resolution:

```go
func (f *providerScopeFactory) resolveCredentials(obj infrav1.IdentityRefProvider) {
    // Priority 1: Check for cluster identity reference
    if clusterRef := obj.GetClusterIdentityRef(); clusterRef != nil {
        return f.newScopeFromClusterIdentity(clusterRef)
    }
    
    // Priority 2: Fall back to existing namespace identity behavior
    return f.newScopeFromNamespaceIdentity(obj.GetIdentityRef())
}
```

**Key Edge Cases:**
- Both references specified: Use `clusterIdentityRef`, log warning
- Identity deletion: Clusters show degraded status, don't fail
- Permission changes: Dynamic re-validation during reconciliation
- Invalid access: Clear error messages with namespace authorization checks

**RBAC Requirements:**
```yaml
rules:
- apiGroups: [""]
  resources: ["secrets", "namespaces"]
  verbs: ["get"]
- apiGroups: ["infrastructure.cluster.x-k8s.io"]
  resources: ["openstackclusteridentities"]
  verbs: ["get", "list", "watch"]
```

### Backward Compatibility

- Existing `identityRef` field remains required and fully functional
- Clusters without `clusterIdentityRef` work exactly as before
- `clusterIdentityRef` takes precedence when both are specified

#### Migration Strategy
- **No forced migration**: Existing deployments continue working indefinitely
- **Manual opt-in**: Users choose when to adopt cluster identities
- **Gradual adoption**: Mix of old and new approaches supported
- **Clear documentation**: Step-by-step migration guides and examples

### Testing Strategy

**Unit Tests**: API validation, dual-path credential resolution, RBAC permission checking, edge cases
**Integration Tests**: End-to-end credential resolution, cross-namespace access validation, permission enforcement
**E2E Tests**: Full cluster lifecycle with cluster identity, mixed deployments, runtime permission changes
**Security Tests**: Unauthorized access attempts, RBAC boundary enforcement, audit trail verification

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Cross-namespace RBAC complexity | Extensive testing, clear documentation, validation webhooks |
| Security boundary violations | Strict validation, audit logging, security review |
| Migration complexity | Clear documentation, examples, optional migration |
| Performance impact | Caching strategy, minimal additional overhead |

## Alternatives

**External Secret Operator**: More complex, adds external dependency
**OpenStack Application Credentials**: Not universally supported across deployments
**Namespace-scoped Identity**: Doesn't solve centralized management
**ConfigMap-based References**: No validation, security concerns

## Example Usage

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OpenStackClusterIdentity
metadata:
  name: production-openstack
spec:
  secretRef:
    name: openstack-credentials
    namespace: capo-system
  allowedNamespaces: [team-a, team-b]

---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OpenStackClusterIdentity
metadata:
  name: development-openstack
spec:
  secretRef:
    name: dev-openstack-credentials
    namespace: capo-system
  namespaceSelector:
    matchLabels:
      environment: "development"
```

### Use in OpenStackCluster
```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OpenStackCluster
metadata:
  name: my-cluster
  namespace: team-a
spec:
  clusterIdentityRef:
    name: production-openstack
  identityRef:  # Fallback for backward compatibility
    name: fallback-secret
    cloudName: openstack
```

## Implementation Notes

**Controller Changes**: Modify `pkg/scope/provider.go` for dual-path resolution, update cluster controller for identity validation
**API Generation**: Update CRD generation, generate deepcopy/clients/informers, update webhooks
**Documentation**: API reference, migration guides, security best practices

This proposal provides centralized credential management while maintaining full backward compatibility and following established Kubernetes patterns.