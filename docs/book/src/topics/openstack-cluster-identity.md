# Centralized credentials with OpenStackClusterIdentity

This guide explains how to centralize OpenStack credentials using a cluster-scoped OpenStackClusterIdentity and reference it from clusters and machines.

## Overview
- OpenStackClusterIdentity (cluster-scoped): stores a reference to a Secret that contains `clouds.yaml` (and optional `cacert`), and optionally restricts which namespaces may use it via `namespaceSelector`.
- OpenStackIdentityReference (on OpenStackCluster/OpenStackMachine/OpenStackServer): carries `type`, `name`, and `cloudName`.
  - `type: Secret` (default): `name` is the Secret name in the same namespace.
  - `type: ClusterIdentity`: `name` is the OpenStackClusterIdentity name; Secret location is taken from the identity.
  - For both types, `cloudName` is required and selects the entry in `clouds.yaml`.

## Prerequisites
- A Secret containing OpenStack credentials in `clouds.yaml`:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: openstack-credentials
  namespace: capo-system
stringData:
  clouds.yaml: |
    clouds:
      openstack:
        auth:
          auth_url: https://keystone.example.com/
          application_credential_id: <id>
          application_credential_secret: <secret>
        region_name: RegionOne
        interface: public
        identity_api_version: 3
        auth_type: v3applicationcredential
  # Optional CA certificate
  # cacert: |
  #   -----BEGIN CERTIFICATE-----
  #   ...
  #   -----END CERTIFICATE-----
```

## Create an OpenStackClusterIdentity
- Optionally restrict which namespaces can use it with `namespaceSelector`.
```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
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
```

## Reference the identity from OpenStackCluster
- Use `type: ClusterIdentity`, specify the identity `name`, and the `cloudName` to select the clouds.yaml entry.
```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OpenStackCluster
metadata:
  name: cluster-a
  namespace: team-a
spec:
  identityRef:
    type: ClusterIdentity
    name: production-openstack
    cloudName: openstack
```

## Using a Secret directly (default)
- If you don’t need cross-namespace identities, use a Secret in the same namespace.
```yaml
spec:
  identityRef:
    # type defaults to Secret
    name: my-secret
    cloudName: openstack
```

## Access control behavior
- If `namespaceSelector` is not set: all namespaces may use the identity.
- If set: only namespaces matching the selector may use the identity.
- If access is denied, the controller returns an error and reconciliation fails. A Warning event can be emitted by the controller with a reason such as `IdentityAccessDenied`.

## RBAC requirements
Ensure the controller has permissions to read: (included in role.yaml by default)
```yaml
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get"]
- apiGroups: ["infrastructure.cluster.x-k8s.io"]
  resources: ["openstackclusteridentities"]
  verbs: ["get","list","watch"]
```

## Notes
- `cloudName` is required on `identityRef` for both `type: Secret` and `type: ClusterIdentity`.
- The Secret must contain a `clouds.yaml` key, and may optionally contain `cacert`.
