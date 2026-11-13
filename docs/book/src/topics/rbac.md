# RBAC aggregation

CAPO ships ClusterRoles that aggregate its resources into the builtin Kubernetes `view`, `edit` and `admin` ClusterRoles. Existing `view`, `edit` and `admin` bindings in the management cluster gain these permissions automatically when CAPO is upgraded.

| Builtin role | Access to CAPO resources |
|--------------|--------------------------|
| `view` | `get`, `list` and `watch` on `OpenStackCluster`, `OpenStackClusterTemplate`, `OpenStackMachine`, `OpenStackMachineTemplate`, `OpenStackFloatingIPPool`, `OpenStackServer` and `OpenStackClusterIdentity`, plus `get` on the `status` subresources |
| `edit` | the same read access, plus `create`, `delete`, `deletecollection`, `patch` and `update` on `OpenStackCluster`, `OpenStackClusterTemplate`, `OpenStackFloatingIPPool`, `OpenStackMachineTemplate` and `OpenStackClusterIdentity` |
| `admin` | everything `edit` has, plus `create`, `delete`, `deletecollection`, `patch` and `update` on `OpenStackMachine` and `OpenStackServer` |

`OpenStackMachine` and `OpenStackServer` are managed by the controllers, and deleting them can leave behind OpenStack resources the controller cannot recover, so write access is limited to `admin`.

None of these resources hold credentials, they only reference Secrets by name.

`OpenStackClusterIdentity` is cluster-scoped, so access to it requires a `ClusterRoleBinding` to the builtin role. Be aware that it decides which namespaces may use which OpenStack credentials.

The roles are defined in `config/rbac/aggregate_roles.yaml` and are part of the default CAPO manifests.
