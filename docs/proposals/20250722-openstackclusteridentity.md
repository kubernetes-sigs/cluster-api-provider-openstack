# OpenStackClusterIdentity for Centralized Credential Management

## Metadata

- **Authors**: @bnallapeta
- **Reviewers**: CAPO maintainers (@mdbooth)
- **Status**: Proposed
- **Creation Date**: 2025-07-22
- **Last Updated**: 2025-07-29

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
    metav1.TypeMeta `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec OpenStackClusterIdentitySpec `json:"spec,omitempty"`
}

type OpenStackClusterIdentitySpec struct {
    // SecretRef references the secret containing OpenStack credentials
    SecretRef OpenStackCredentialSecretReference `json:"secretRef"`

    // NamespaceSelector selects allowed namespaces via labels
    // All namespaces have a kubernetes.io/metadata.name label containing their name
    // +optional
    NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`
}

type OpenStackCredentialSecretReference struct {
    Name      string `json:"name"`
    Namespace string `json:"namespace"`
}
```

#### Enhanced OpenStackIdentityReference (Discriminated Union)

We extend the existing `identityRef` to support multiple types using a discriminated union pattern:

```go
type OpenStackIdentityReference struct {
    // Type specifies the identity reference type
    // +kubebuilder:validation:Enum=Secret;ClusterIdentity
    // +kubebuilder:default=Secret
    // +kubebuilder:validation:XValidation:rule="self == 'Secret' ? has(self.cloudName) : !has(self.cloudName)",message="cloudName required for Secret type, forbidden for ClusterIdentity type"
    // +kubebuilder:validation:XValidation:rule="has(self.name)",message="name is required"
    // +optional
    Type string `json:"type,omitempty"`

    // Name of the secret (type=Secret) or cluster identity (type=ClusterIdentity)
    // +optional
    Name string `json:"name,omitempty"`
    
    // CloudName required for Secret type, forbidden for ClusterIdentity type
    // +optional
    CloudName string `json:"cloudName,omitempty"`
    
    // Region applies to both types
    // +optional
    Region string `json:"region,omitempty"`
}
```

### Implementation Details

#### Credential Resolution Logic
The scope factory will implement type-based credential resolution:

```go
func (f *providerScopeFactory) resolveCredentials(identityRef *infrav1.OpenStackIdentityReference) {
    switch identityRef.Type {
    case "ClusterIdentity":
        return f.newScopeFromClusterIdentity(identityRef.Name)
    case "Secret", "": // Default to Secret for backward compatibility
        return f.newScopeFromSecretIdentity(identityRef)
    default:
        return fmt.Errorf("unsupported identity type: %s", identityRef.Type)
    }
}
```

#### Permission and Access Control

This feature involves two distinct types of permissions:

**1. Controller RBAC (Kubernetes-level permissions)**
The CAPO controller already has cluster-wide secret access. We only need to add:

```yaml
# ADD to existing config/rbac/role.yaml
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get"]  # Read namespace metadata for validation
- apiGroups: ["infrastructure.cluster.x-k8s.io"]
  resources: ["openstackclusteridentities"]
  verbs: ["get", "list", "watch"]  # Manage cluster identities
```

**2. Namespace Access Control (Application-level permissions)**
The cluster identity defines which namespaces can use it via namespace selectors:

```go
func (r *OpenStackClusterReconciler) validateNamespaceAccess(identity *OpenStackClusterIdentity, namespace string) error {
    // If no selector specified, allow all namespaces
    if identity.Spec.NamespaceSelector == nil {
        return nil
    }
    
    // Get the namespace object
    ns := &corev1.Namespace{}
    err := r.Client.Get(ctx, types.NamespacedName{Name: namespace}, ns)
    if err != nil {
        return fmt.Errorf("failed to get namespace %s: %w", namespace, err)
    }
    
    // Check if namespace matches the selector
    selector, err := metav1.LabelSelectorAsSelector(identity.Spec.NamespaceSelector)
    if err != nil {
        return fmt.Errorf("invalid namespace selector: %w", err)
    }
    
    if !selector.Matches(labels.Set(ns.Labels)) {
        return fmt.Errorf("namespace %s not allowed to use cluster identity %s", 
            namespace, identity.Name)
    }
    
    return nil
}
```

**Key Edge Cases:**
- Missing type field: Defaults to `Secret` behavior (100% backward compatible)
- Invalid type: Clear error message
- Invalid field combinations: CEL validation prevents misconfigurations
- Namespace access denied: Clear error message with identity name and namespace

#### CEL Validation

We use CEL (Common Expression Language) for API validation, following existing CAPO patterns:

```go
type OpenStackIdentityReference struct {
    // Type specifies the identity reference type
    // +kubebuilder:validation:Enum=Secret;ClusterIdentity
    // +kubebuilder:default=Secret
    // +kubebuilder:validation:XValidation:rule="self == 'Secret' ? has(self.cloudName) : !has(self.cloudName)",message="cloudName required for Secret type, forbidden for ClusterIdentity type"
    // +kubebuilder:validation:XValidation:rule="has(self.name)",message="name is required"
    // +optional
    Type string `json:"type,omitempty"`

    // Name of the secret (type=Secret) or cluster identity (type=ClusterIdentity)
    // +optional
    Name string `json:"name,omitempty"`
    
    // CloudName required for Secret type, forbidden for ClusterIdentity type
    // +optional
    CloudName string `json:"cloudName,omitempty"`
    
    // Region applies to both types
    // +optional
    Region string `json:"region,omitempty"`
}
```

**CEL Validation Rules:**
1. **Name Required**: `name` field is always required for both types
2. **CloudName Logic**: Required for Secret type, forbidden for ClusterIdentity type
3. **Type Safety**: Enum validation ensures only valid types are accepted

### Backward Compatibility

- **Perfect backward compatibility**: Existing `identityRef` configurations work unchanged
- **Default behavior**: Missing `type` field defaults to `Secret` behavior
- **No migration required**: Existing clusters continue working
- **Gradual adoption**: Users can adopt `type: ClusterIdentity` when required

#### Migration Strategy
- **No forced migration**: Existing deployments continue working indefinitely
- **Manual opt-in**: Users add `type: ClusterIdentity` when ready
- **Clear validation**: Type-specific field validation prevents misconfigurations

### Testing Strategy

**Unit Tests**: API validation for both identity types, credential resolution logic, namespace access validation
**Integration Tests**: End-to-end credential resolution for both types, cross-namespace access validation
**E2E Tests**: Full cluster lifecycle with both identity types, mixed deployments
**Security Tests**: Unauthorized access attempts, namespace boundary enforcement

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Complex validation logic | Type-specific validation, comprehensive testing |
| Field confusion | Clear documentation, validation webhooks |
| Migration issues | Extensive backward compatibility testing |
| Cross-namespace security | Strict namespace selector validation, audit logging |

## Alternatives

**Separate clusterIdentityRef field**: Requires dual fields, harder to maintain long-term
**External Secret Operator**: More complex, adds external dependency
**ConfigMap-based References**: No validation, security concerns

## Example Usage

### Create Cluster Identity
```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OpenStackClusterIdentity
metadata:
  name: production-openstack
spec:
  secretRef:
    name: openstack-credentials
    namespace: capo-system
  namespaceSelector:
    matchExpressions:
    - key: kubernetes.io/metadata.name
      operator: In
      values: [team-a, team-b]
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
      environment: development
```

### Current Secret-based Identity (unchanged)
```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OpenStackCluster
metadata:
  name: cluster-a
  namespace: team-a
spec:
  identityRef:
    # No type field = defaults to Secret behavior
    name: my-secret
    cloudName: openstack
```

### New ClusterIdentity-based (new usage)
```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OpenStackCluster
metadata:
  name: cluster-a
  namespace: team-a
spec:
  identityRef:
    type: ClusterIdentity
    name: prod-openstack
```

### Explicit Secret Type (optional)
```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OpenStackCluster
metadata:
  name: cluster-a
  namespace: team-a
spec:
  identityRef:
    type: Secret
    name: my-secret
    cloudName: openstack
```

## Implementation Notes

**API Changes**: 
- Extend `OpenStackIdentityReference` with `type` field with `Secret` and `ClusterIdentity` as the only supported values
- Add CEL validation rules for type-specific field combinations
- Update CRD generation for new field

**RBAC Changes**:
- Add namespace `get` permission to existing `config/rbac/role.yaml`
- Add `openstackclusteridentities` resource permissions

**Controller Changes**: 
- Modify `pkg/scope/provider.go` for type-based credential resolution
- Add namespace access validation logic in controllers
- Add cluster identity lookup and permission checking
- Implement comprehensive error handling with clear messages

**Backward Compatibility**: 
- Default `type` to `Secret` when not specified
- CEL validation maintains existing field requirements for secret-based identities
- Ensure all existing configurations continue working

**Security Considerations**:
- Validate namespace access on every cluster reconciliation
- Log access attempts for audit purposes
- Clear error messages for permission denials
- Fail securely when cluster identity is not accessible

**Broader Impact**: 
This change automatically enables cluster identity support for:
- `OpenStackCluster` resources
- `OpenStackMachine` resources  
- `OpenStackServer` resources

This proposal provides centralized credential management while maintaining full backward compatibility and following established Kubernetes patterns (discriminated union).
