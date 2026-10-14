# RBAC aggregation

CAPO ships ClusterRoles that aggregate its resources into the builtin Kubernetes `view`, `edit` and `admin` ClusterRoles:

| Builtin role | Access to CAPO resources |
|--------------|--------------------------|
| `view` | `get`, `list` and `watch` on `OpenStackCluster`, `OpenStackClusterTemplate`, `OpenStackMachine`, `OpenStackMachineTemplate`, `OpenStackFloatingIPPool`, `OpenStackServer` and `OpenStackClusterIdentity`, plus `get` on the `status` subresources |
| `edit`, `admin` | the same read access, plus `create`, `delete`, `deletecollection`, `patch` and `update` |

None of these resources hold credentials, they only reference Secrets by name.

`OpenStackClusterIdentity` is cluster-scoped, so access to it requires a `ClusterRoleBinding` to the builtin role. Be aware that it decides which namespaces may use which OpenStack credentials.

The roles are defined in `config/rbac/aggregate_roles.yaml` and are part of the default CAPO manifests.
