# Port Garbage Collection

> **Note:** PortGarbageCollection is available as an alpha feature in >= 0.16 (disabled by default).

CAPO creates the Neutron ports of every server itself and passes them to Nova by ID. Nova treats
ports it did not create as owned by the caller: deleting the server only detaches them, and Neutron
never deletes an unattached port on its own. CAPO is therefore the only component that can clean
these ports up, and it does so for every port recorded in `OpenStackServer.status.resources.ports`.

A port CAPO created but never recorded is not cleaned up by anything. This can happen when the
controller creates a port and then fails to persist its ID, for example because it restarted or the
status update failed, or because of bugs such as
[#3427](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/issues/3427). These ports
stay `DOWN` in Neutron, hold an IP address, and count against the project's port quota.

The `PortGarbageCollection` feature flag makes the OpenStackServer controller delete such ports:

- after it ensures the server's ports during a normal reconcile, and
- after it deletes the server and its recorded ports.

## Which ports are deleted

A port is only deleted when all of the following hold:

- its name, network ID and description equal those of one of the server's resolved ports,
- its ID is not recorded in `OpenStackServer.status.resources.ports`, and
- it is not attached to anything: both `device_id` and `device_owner` are empty.

Ports created by Nova are always attached to their server and are never deleted. If a matching port
has a trunk and the trunk extension is available, the trunk is deleted with it.

Like port adoption, this relies on server names being unique within an OpenStack project and
network. If two `OpenStackServer` objects with the same name share a project, network and port
description, for example clusters with the same name in different namespaces, one server can delete
an unattached port that belongs to the other. Do not enable the feature in that setup.

When the feature is enabled, every reconcile of an `OpenStackServer` lists Neutron ports once per
desired port, filtered by name, network and description.

## Enabling PortGarbageCollection

You can enable `PortGarbageCollection` using the following.

- Environment variable: `EXP_CAPO_PORT_GARBAGE_COLLECTION=true`
- clusterctl.yaml variable: `EXP_CAPO_PORT_GARBAGE_COLLECTION: true`
- --feature-gates argument: `PortGarbageCollection=true`
