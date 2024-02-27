<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [v1alpha7 compared to v1beta1](#v1alpha7-compared-to-v1beta1)
  - [Migration](#migration)
  - [API Changes](#api-changes)
    - [`OpenStackMachine`](#openstackmachine)
      - [Removal of machine identityRef.kind](#removal-of-machine-identityrefkind)
      - [Change to serverGroupID](#change-to-servergroupid)
    - [`OpenStackCluster`](#openstackcluster)
      - [Removal of cluster identityRef.kind](#removal-of-cluster-identityrefkind)
      - [Change to externalNetworkID](#change-to-externalnetworkid)
      - [Changes to image](#change-to-image)
      - [Removal of imageUUID](#removal-of-imageuuid)
      - [Change to floatingIP](#change-to-floatingip)
      - [Change to subnet](#change-to-subnet)
      - [Change to nodeCidr and dnsNameservers](#change-to-nodecidr-and-dnsnameservers)
      - [Change to managedSecurityGroups](#change-to-managedsecuritygroups)
      - [Calico CNI](#calico-cni)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# v1alpha7 compared to v1beta1

> ⚠️ v1beta1 has not been released yet.

## Migration

All users are encouraged to migrate their usage of the CAPO CRDs from older versions to `v1beta1`. This includes yaml files and source code. As CAPO implements automatic conversions between the CRD versions, this migration can happen after installing the new CAPO release.

## API Changes

This only documents backwards incompatible changes. Fields that were added to v1beta1 are not listed here.

### `OpenStackMachine`

#### Removal of machine identityRef.kind

The `identityRef.Kind` field has been removed. It was used to specify the kind of the identity provider to use but was actually ignored.

#### Change to serverGroupID

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

#### Removal of cluster identityRef.kind

The `identityRef.Kind` field has been removed. It was used to specify the kind of the identity provider to use but was actually ignored.

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

#### Change to image

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

#### Removal of imageUUID

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

#### Change to subnet

In v1beta1, `Subnet` of `OpenStackCluster` is modified to `Subnets` to allow specification of two existent subnets for the dual-stack scenario.

```yaml
  subnet:
    id: a532beb0-c73a-4b5d-af66-3ad05b73d063
```

In v1beta1, this will be automatically converted to:

```yaml
  subnets:
    - id: a532beb0-c73a-4b5d-af66-3ad05b73d063
```

`Subnets` allows specifications of maximum two `SubnetFilter` one being IPv4 and the other IPv6. Both subnets must be on the same network. Any filtered subnets will be added to `OpenStackCluster.Status.Network.Subnets`.

When subnets are not specified on `OpenStackCluster` and only the network is, the network is used to identify the subnets to use. If more than two subnets exist in the network, the user must specify which ones to use by defining the `OpenStackCluster.Spec.Subnets` field.

#### Change to nodeCidr and dnsNameservers

In v1beta1, `OpenStackCluster.Spec.ManagedSubnets` array field is introduced. The `NodeCIDR` and `DNSNameservers` fields of `OpenStackCluster.Spec` are moved into that structure (renaming `NodeCIDR` to `CIDR`). For example:

```yaml
  nodeCidr: "10.0.0.0/24"
  dnsNameservers: "10.0.0.123"
```

In v1beta1, this will be automatically converted to:

```yaml
  managedSubnets:
  - cidr: "10.0.0.0/24"
    dnsNameservers: "10.0.0.123"
```

Please note that currently `managedSubnets` can only hold one element.

#### Addition of allocationPools

In v1beta1, an `AllocationPools` property is introduced to `OpenStackCluster.Spec.ManagedSubnets`. When specified, OpenStack subnet created by CAPO will have the given values set as the `allocation_pools` property. This allows users to make sure OpenStack will not allocate some IP ranges in the subnet automatically. If the subnet is precreated and configured, CAPO will ignore `AllocationPools` property.

#### Change to managedSecurityGroups

The field `managedSecurityGroups` is now a pointer to a `ManagedSecurityGroups` object rather than a boolean.

Also, we can now add security group rules that authorize traffic from all nodes via `allNodesSecurityGroupRules`.
It takes a list of security groups rules that should be applied to selected nodes.
The following rule fields are mutually exclusive: `remoteManagedGroups`, `remoteGroupID` and `remoteIPPrefix`.
Valid values for `remoteManagedGroups` are `controlplane`, `worker` and `bastion`.

Also, `OpenStackCluster.Spec.AllowAllInClusterTraffic` moved under `ManagedSecurityGroups`.

```yaml
managedSecurityGroups: true
```

becomes

```yaml
managedSecurityGroups: {}
```

and

```yaml
allowAllInClusterTraffic: true
managedSecurityGroups: true
```

becomes

```yaml
managedSecurityGroups:
  allowAllInClusterTraffic: true
```

To apply a security group rule that will allow BGP between the control plane and workers, you can follow this example:

```yaml
managedSecurityGroups:
  allNodesSecurityGroupRules:
  - remoteManagedGroups:
    - controlplane
    - worker
    direction: ingress
    etherType: IPv4
    name: BGP (Calico)
    portRangeMin: 179
    portRangeMax: 179
    protocol: tcp
    description: "Allow BGP between control plane and workers"
```

#### Calico CNI

Historically we used to create the necessary security group rules for Calico CNI to work. This is no longer the case.
Now the user needs to request creation of the security group rules by using the `managedSecurityGroups.allNodesSecurityGroupRules` feature.

Note that when upgrading from a previous version, the Calico CNI security group rules will be added automatically to
allow backwards compatibility if `allowAllInClusterTraffic` is set to false.

#### Change to network

In v1beta1, when the `OpenStackCluster.Spec.Network` is not defined, the `Subnets` are now used to identify the `Network`.
