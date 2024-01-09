<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [v1alpha7 compared to v1alpha8](#v1alpha7-compared-to-v1alpha8)
  - [Migration](#migration)
  - [API Changes](#api-changes)
    - [`OpenStackMachine`](#openstackmachine)
      - [Change to `serverGroupID`](#change-to-servergroupid)
    - [`OpenStackCluster`](#openstackcluster)
      - [Change to externalNetworkID](#change-to-externalnetworkid)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# v1alpha7 compared to v1alpha8

> ⚠️ v1alpha8 has not been released yet.
## Migration

All users are encouraged to migrate their usage of the CAPO CRDs from older versions to `v1alpha8`. This includes yaml files and source code. As CAPO implements automatic conversions between the CRD versions, this migration can happen after installing the new CAPO release.

## API Changes

This only documents backwards incompatible changes. Fields that were added to v1alpha8 are not listed here.

### `OpenStackMachine`

#### ⚠️ Change to `serverGroupID`

The field `serverGroupID` has been renamed to `serverGroup` and is now a `ServerGroupFilter` object rather than a string ID.

The `ServerGroupFilter` object allows selection of a server group by name or by ID.

```yaml
serverGroupID: "e60f19e7-cb37-49f9-a2ee-0a1281f6e03e"
```

becomes

```yaml
serverGroup:
  id: "e60f19e7-cb37-49f9-a2ee-0a1281f6e03e"
```

To select a server group by name instead of ID:

```yaml
serverGroup:
  name: "workers"
```

If a server group is provided and found, it'll be added to `OpenStackMachine.Status.ReferencedResources.ServerGroupID`. If the server group can't be found or filter matches multiple server groups, an error will be returned.
If empty object or null is provided, Machine will not be added to any server group and `OpenStackMachine.Status.ReferencedResources.ServerGroupID` will be empty.

### `OpenStackCluster`

#### Change to externalNetworkID


The field `externalNetworkID` has been renamed to `externalNetwork` and is now a `NetworkFilter` object rather than a string ID.
The `NetworkFilter` object allows selection of a network by name, by ID or by tags.

```yaml
externalNetworkID: "e60f19e7-cb37-49f9-a2ee-0a1281f6e03e"
```

becomes

```yaml
externalNetwork:
  id: "e60f19e7-cb37-49f9-a2ee-0a1281f6e03e"
```

It is now possible to specify a `NetworkFilter` object to select the external network to use for the cluster. The `NetworkFilter` object allows to select the network by name, by ID or by tags.

```yaml
externalNetwork:
  name: "public"
```

If a network is provided, it'll be added to `OpenStackCluster.Status.ExternalNetwork`. If the network can't be found, an error will be returned.
If no network is provided, CAPO will try to find network marked "External" and add it to `OpenStackCluster.Status.ExternalNetwork`. If it can't find a network marked "External",
`OpenStackCluster.Status.ExternalNetwork` will be set to nil.
If more than one network is found, an error will be returned.

It is now possible for a user to specify that no external network should be used by setting `DisableExternalNetwork` to `true`:

```yaml
disableExternalNetwork: true
```
