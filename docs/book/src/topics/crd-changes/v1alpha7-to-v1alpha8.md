<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [v1alpha7 compared to v1alpha8](#v1alpha7-compared-to-v1alpha8)
  - [Migration](#migration)
  - [API Changes](#api-changes)
    - [`OpenStackMachine`](#openstackmachine)
      - [⚠️ Change to `serverGroupID`](#️-change-to-servergroupid)
    - [`OpenStackCluster`](#openstackcluster)
      - [Change to externalNetworkID](#change-to-externalnetworkid)
      - [Changes to image](#change-to-image)
      - [Removal of imageUUID](#removal-of-imageuuid)
      - [Change to floatingIP](#change-to-floatingip)
      - [Change to subnet](#change-to-subnet)

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

#### ⚠️ Change to image

The field `image` is now an `ImageFilter` object rather than a string name.
The `ImageFilter` object allows selection of an image by name, by ID or by tags.

```yaml
image: "test-image"
```

becomes

```yaml
image:
  name: "test-image"
```

The image ID will be added to `OpenStackMachine.Status.ReferencedResources.ImageID`. If the image can't be found or filter matches multiple images, an error will be returned.

#### ⚠️ Removal of imageUUID

The fild `imageUUID` has been removed in favor of the `image` field.

```yaml
imageUUID: "72a6a1e6-3e0a-4a8b-9b4c-2d6f9e3e5f0a"
```

becomes

```yaml
image:
  id: "72a6a1e6-3e0a-4a8b-9b4c-2d6f9e3e5f0a"
```

#### Change to floatingIP

The `OpenStackMachineSpec.FloatingIP` field has moved to `OpenStackClusterSpec.Bastion.FloatingIP`.
For example, if you had the following `OpenStackMachineTemplate`:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha6
kind: OpenStackMachineTemplate
metadata:
  name: ${CLUSTER_NAME}-md-0
spec:
  template:
    spec:
      ..
      floatingIP: "1.2.3.4"
```

This will safely converted to use `Bastion.FloatingIP` when upgrading to version 0.8.

To use the new `Bastion.FloatingIP` field, here is an example:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha7
kind: OpenStackCluster
metadata:
  name: ${CLUSTER_NAME}
spec:
  ..
  bastion:
    floatingIP: "1.2.3.4"
```

#### ⚠️ Change to subnet

In v1alpha8, `Subnet` of `OpenStackCluster` is modified to `Subnets` to allow specification of two existent subnets for the dual-stack scenario.

```yaml
  subnet:
    id: a532beb0-c73a-4b5d-af66-3ad05b73d063
```

In v1alpha8, this will be automatically converted to:

```yaml
  subnets:
    - id: a532beb0-c73a-4b5d-af66-3ad05b73d063
```

`Subnets` allows specifications of maximum two `SubnetFilter` one being IPv4 and the other IPv6. Both subnets must be on the same network. Any filtered subnets will be added to `OpenStackCluster.Status.Network.Subnets`.

When subnets are not specified on `OpenStackCluster` and only the network is, the network is used to identify the subnets to use. If more than two subnets exist in the network, the user must specify which ones to use by defining the `OpenStackCluster.Spec.Subnets` field.