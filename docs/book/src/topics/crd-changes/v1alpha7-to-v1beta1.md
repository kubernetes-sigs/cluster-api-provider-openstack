<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [v1alpha7 compared to v1beta1](#v1alpha7-compared-to-v1beta1)
  - [Migration](#migration)
  - [API Changes](#api-changes)
    - [Changes to `identityRef` in both `OpenStackMachine` and `OpenStackCluster`](#changes-to-identityref-in-both-openstackmachine-and-openstackcluster)
      - [Removal of machine identityRef.kind](#removal-of-machine-identityrefkind)
      - [Addition of cloudName](#addition-of-cloudname)
    - [`OpenStackMachine`](#openstackmachine)
      - [Removal of cloudName](#removal-of-cloudname)
      - [Change to serverGroupID](#change-to-servergroupid)
      - [Change to image](#change-to-image)
      - [Removal of imageUUID](#removal-of-imageuuid)
      - [Change to instanceID](#change-to-instanceid)
      - [Changes to ports](#changes-to-ports)
      - [Change to `availabilityZone` for `additionalBlockDevices`](#change-to-availabilityzone-for-additionalblockdevices)
      - [Changes to rootVolume](#changes-to-rootvolume)
      - [Addition of floatingIPPoolRef](#addition-of-floatingippoolref)
    - [`OpenStackCluster`](#openstackcluster)
      - [Removal of cloudName](#removal-of-cloudname-1)
      - [identityRef is now required](#identityref-is-now-required)
      - [Change to externalNetworkID](#change-to-externalnetworkid)
      - [Change to floatingIP](#change-to-floatingip)
      - [Change to subnet](#change-to-subnet)
      - [Change to nodeCidr and dnsNameservers](#change-to-nodecidr-and-dnsnameservers)
      - [Addition of allocationPools](#addition-of-allocationpools)
      - [Change to managedSecurityGroups](#change-to-managedsecuritygroups)
      - [Calico CNI](#calico-cni)
      - [Change to network](#change-to-network)
      - [Change to networkMtu](#change-to-networkmtu)
      - [Changes to apiServerLoadBalancer](#changes-to-apiserverloadbalancer)
      - [Changes to bastion](#changes-to-bastion)
    - [Changes to filters](#changes-to-filters)
      - [Filters are replaced by params with separate id and filter](#filters-are-replaced-by-params-with-separate-id-and-filter)
      - [Changes to filter tags](#changes-to-filter-tags)
      - [Field capitalization consistency](#field-capitalization-consistency)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# v1alpha7 compared to v1beta1

> ⚠️ v1beta1 has not been released yet.

## Migration

All users are encouraged to migrate their usage of the CAPO CRDs from older versions to `v1beta1`. This includes yaml files and source code. As CAPO implements automatic conversions between the CRD versions, this migration can happen after installing the new CAPO release.

## API Changes

This only documents backwards incompatible changes. Fields that were added to v1beta1 are not listed here.

### Changes to `identityRef` in both `OpenStackMachine` and `OpenStackCluster`

#### Removal of machine identityRef.kind

The `identityRef.Kind` field has been removed. It was used to specify the kind of the identity provider to use but was actually ignored. The referenced resource must always be a Secret.

#### Addition of cloudName

The `cloudName` field has been removed from both `OpenStackMachine` and `OpenStackCluster` and added to `identityRef`. It is now a required field when `identityRef` is specified.

### `OpenStackMachine`

#### Removal of cloudName

This has moved to `identityRef.cloudName`.

#### Change to serverGroupID

The field `serverGroupID` has been renamed to `serverGroup` and is now a `ServerGroupParam` object rather than a string ID.

The `ServerGroupParam` object allows selection of a server group by name or by ID.

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
  filter:
    name: "workers"
```

If a server group is provided and found, it will be added to `OpenStackMachine.Status.ReferencedResources.ServerGroupID`. If the server group can't be found or filter matches multiple server groups, an error will be returned.
If no serverGroup is provided, Machine will not be added to any server group and `OpenStackMachine.Status.ReferencedResources.ServerGroupID` will be empty.
It is an error to provide an empty serverGroup.

#### Change to image

The field `image` is now an `ImageParam` object rather than a string name.
The `ImageParam` object allows selection of an image by ID or by filter. Either ID or a filter may be set, but not both.

```yaml
image: "test-image"
```

becomes

```yaml
image:
  filter:
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

#### Change to instanceID

In v1beta1, the field `instanceID` moves from `spec` to `status`.

#### Changes to ports

These changes apply to ports specified in both OpenStackMachines and the Bastion.

The `securityGroupFilters` field is renamed to `securityGroups` to be consistent with other similar fields throughout the API.

When specifying `allowedAddressPairs`, `ipAddress` is now required. Neutron has already required this parameter, so while the API previously allowed it to be unset it would not have resulted in a working VM.

Setting either of the following fields explicitly to the empty string would previously have caused the default behaviour to be used. In v1beta1 it will result in an empty string being used instead. To retain the default behaviour in v1beta1, do not set the value at all. When objects are upgraded automatically by the controller, empty values will become unset values by default, so this applies only to newly created objects.
* nameSuffix
* description

The following fields in `PortOpts` are renamed in order to keep them consistent with K8s API conventions:

* `hostId` becomes `hostID`
* `allowedCidrs` becomes `allowedCIDRs`

#### Change to `availabilityZone` for `additionalBlockDevices`

Prior to v1beta1 it was not possible to specify a volume with no availability zone, allowing Cinder to allocate the volume according to its default policy. Previously, if no availability zone was given CAPO would use the value of `failureDomain` from the associated `Machine`.

In v1beta1 this default changes. In v1beta1, if no `availabilityZone` is specified, CAPO will not specify an availability zone for the created volume.

To support this whilst continuing to support the previous behaviour, `availabilityZone` becomes a struct with 2 fields:

`from` specifies where we get the availability zone from. It can be either `Name`, or `Machine`. It can be omitted, and the default is `Name`, in which case the value of the `name` field will be used. Specifying `Machine` provides the previous default behaviour.

`name` is a specific availability zone to use.

For example, a volume with an explicit AZ in:
* v1alpha7
  ```yaml
  - name: extra
    sizeGiB: 1000
    storage:
      type: volume
      volume:
        type: ceph
        availabilityZone: az1
  ```
* v1beta1
  ```yaml
  - name: extra
    sizeGiB: 1000
    storage:
      type: volume
      volume:
        type: ceph
        availabilityZone:
          name: az1
  ```

A volume which uses the value of `failureDomain` from the `Machine`:
* v1alpha7
  ```yaml
  - name: extra
    sizeGiB: 1000
    storage:
      type: volume
      volume:
        type: ceph
  ```
* v1beta1
  ```yaml
  - name: extra
    sizeGiB: 1000
    storage:
      type: volume
      volume:
        type: ceph
        availabilityZone:
          from: Machine
  ```

A volume which explicitly has no availability zone:
* v1alpha7: Not possible
* v1beta1
  ```yaml
  - name: extra
    sizeGiB: 1000
    storage:
      type: volume
      volume:
        type: ceph
  ```

Volumes upgraded from prior versions will retain their current semantics, so volumes which previously specified no availabilityZone will have an availabilityZone with `from: Machine` added.

#### Changes to rootVolume

The `rootVolume` field has been updated to be consistent with the volume specification for `additionalBlockDevices`. Specifically:

* `rootVolume.diskSize` becomes `rootVolume.sizeGiB`
* `rootVolume.volumeType` becomes `rootVolume.Type`
* `rootVolume.availabilityZone` becomes a struct with the same semantics as `availabilityZone` in `additionalBlockDevices`, described above.

For example:
* v1alpha7
  ```yaml
  rootVolume:
    diskSize: 50
    volumeType: ceph
  ```
* v1beta1
  ```yaml
  rootVolume:
    sizeGiB: 50
    type: ceph
    availabilityZone:
      from: Machine
  ```

#### Addition of floatingIPPoolRef

A new field, FloatingIPPoolRef, has been introduced. It is important to note that this feature requires the existence of an IPPool to operate seamlessly. This new field references an IPPool whice will be used for floating IP allocation.
In additon to this field an IPPool resource called `OpenStackFloatingIPPool` has been added. Please note that this resource is new and still in alpha meaning its more likely to contain bugs and change in the future.

When creating an OpenStackFloatingIPPool resource named MyOpenStackFloatingIPPool, you can reference it within your OpenStackMachine configuration YAML as follows:

```yaml
spec:
  template:
    spec:
      floatingIPPoolRef:
        apiGroup: "infrastructure.cluster.x-k8s.io"
        kind: "OpenStackFloatingIPPool"
        name: "MyOpenStackFloatingIPPool"
```

### `OpenStackCluster`

#### Removal of cloudName

This has moved to `identityRef.cloudName`.

#### identityRef is now required

The API server would previously accept an `OpenStackCluster` without an `identityRef`, although the controller would generate an error. In v1beta1 the API server will no longer accept an `OpenStackCluster` without an `identityRef`.

Note that this is in contrast `identityRef` in `OpenStackMachine`, which remains optional: `OpenStackMachine` will default to the credentials in `OpenStackCluster` if not specified.

#### Change to externalNetworkID

The field `externalNetworkID` has been renamed to `externalNetwork` and is now a `NetworkParam` object rather than a string ID.
The `NetworkParam` object allows selection of a network by id or filter parameters.

```yaml
externalNetworkID: "e60f19e7-cb37-49f9-a2ee-0a1281f6e03e"
```

becomes

```yaml
externalNetwork:
  id: "e60f19e7-cb37-49f9-a2ee-0a1281f6e03e"
```

It is now possible to specify a `NetworkParam` object to select the external network to use for the cluster. The `NetworkParam` object allows to select the network by id or by filter, which can select by name, ID or tags.

```yaml
externalNetwork:
  filter:
    name: "public"
```

If a network is provided, it will be added to `OpenStackCluster.Status.ExternalNetwork`. If the network can't be found, an error will be returned.
If no network is provided, CAPO will try to find network marked "External" and add it to `OpenStackCluster.Status.ExternalNetwork`. If it can't find a network marked "External",
`OpenStackCluster.Status.ExternalNetwork` will be set to nil.
If more than one network is found, an error will be returned.

It is now possible for a user to specify that no external network should be used by setting `DisableExternalNetwork` to `true`:

```yaml
disableExternalNetwork: true
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

The new field has IPv4 validation added.

#### Change to subnet

In v1beta1, `Subnet` of `OpenStackCluster` is modified to `Subnets` to allow specification of two existing subnets for the dual-stack scenario.

```yaml
  subnet:
    id: a532beb0-c73a-4b5d-af66-3ad05b73d063
```

In v1beta1, this will be automatically converted to:

```yaml
  subnets:
    - id: a532beb0-c73a-4b5d-af66-3ad05b73d063
```

`Subnets` allows specifications of maximum two `SubnetParam`s one being IPv4 and the other IPv6. Both subnets must be on the same network. Any filtered subnets will be added to `OpenStackCluster.Status.Network.Subnets`.

When subnets are not specified on `OpenStackCluster` and only the network is, the network is used to identify the subnets to use. If more than two subnets exist in the network, the user must specify which ones to use by defining the `OpenStackCluster.Spec.Subnets` field.

See [Filters are replaced by params with separate id and filter](#filters-are-replaced-by-params-with-separate-id-and-filter) for changes to the `SubnetFilter`.

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
If none of these fields are set, the rule will have a remote IP prefix of `0.0.0.0/0` per Neutron default.

```yaml
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

#### Change to networkMtu

In v1beta1, `OpenStackCluster.spec.NetworkMtu` becomes `OpenStackCluster.spec.NetworkMTU` in order to keep the name consistent with K8s API conventions.

#### Changes to apiServerLoadBalancer

In v1beta1, `OpenStackCluster.spec.apiServerLoadBalancer` becomes optional and can be entirely omitted. Therefore `enabled` can be assumed by the presence of the field. Consequently `enabled` now defaults to `true` if `apiServerLoadBalancer` is specified. The following are now equivalent:

```yaml
spec:
    ...
    apiServerLoadBalancer:
      enabled: true
```

```yaml
spec:
    ...
    apiServerLoadBalancer: {}
```

#### Changes to bastion

In v1beta1 the Bastion is enabled by default if a bastion is specified in `OpenStackCluster.spec.bastion` but `enabled` is not included. However, it is still required to set `enabled` explicitly to `false` before removing the bastion. We intend to remove this requirement in the future at which point the `enabled` field will be redundant.

In v1beta1, `OpenStackCluster.spec.bastion.instance` becomes `OpenStackCluster.spec.bastion.spec`.

```yaml
spec:
    ...
    bastion:
      instance:
        image:
          name: foobar
```

becomes

```yaml
spec:
    ...
    bastion:
      spec:
        image:
          filter:
            name: foobar
```

### Changes to filters

#### Filters are replaced by params with separate id and filter

We previously defined filters for specifying each of the following:
* Images
* Networks
* Subnets
* Security Groups
* Routers

In addition, ServerGroupParam is new in v1beta1, but works in the same way.

Taking Images as an example, in `OpenStackMachineSpec` the `image` parameter accepted the following fields:
* id
* name
* tags

However, there were 2 different behaviours here depending on which fields were specified. If `id` was specified we both ignored all other fields, and made no attempt to validate the id because it cannot be known unless it exists. If `id` was not specified we performed an OpenStack query using the other parameters to determine the id. This behaviour is both difficult to describe and validate, so in v1beta1 we have separated the 2 different behaviours into different fields. The `id` field remains, but all other fields become part of a new `filter` parameter. It is required to specify either `id` or `filter`. Specifying both, neither, or an empty filter, will now result in an admission failure (the API server will reject the operation immediately).

We have also added validation to the id field, which must now parse as a uuid. An id field which does not parse as a uuid will also result in an admission failure.

Specifying a filter object by id remains unchanged.
```yaml
  image:
    id: 02e31c38-d5ba-4c57-addb-00fc71eb5b19
```

Specifying a filter object by a filter parameter must now move filter parameters into the filter field:
```yaml
  image:
    name: my-image
    tags:
    - my-tag
```
becomes:
```yaml
  image:
    filter:
      name: my-image
      tags:
      - my-tag
```

The same principle applies to all the filter types. For example:

To specify `externalNetwork` by id in `OpenStackClusterSpec`:
```yaml
  externalNetwork:
    id: 0a5d4e3d-0a2c-4bed-9172-72544db1f8da
```
or by name:
```yaml
  externalNetwork:
    filter:
      name: my-external-network
```

#### Changes to filter tags

We currently define filters on 4 different Neutron resources which are used throughout the API:
* Networks
* Subnets
* Security Groups
* Routers

Each of these filters provide the following fields which filter by tag:
* tags
* tagsAny
* notTags
* notTagsAny

We previously passed this value to Neutron without change, which treated it as a comma-separated list. In v1beta1 each of the tags fields becomes a list of tags. e.g.:

```yaml
subnet:
  tags: "foo,bar"
```
becomes
```yaml
subnet:
  tags:
  - foo
  - bar
```

Due to the limitations of the encoding of tag queries in Neutron, tags must be non-empty and may not contain commas. Tags will be automatically converted to a list of tags during conversion.

#### Field capitalization consistency

In order to keep field names consistent with K8s API conventions, various fields in the `Filters` are renamed. This includes:

* `SecurityGroupFilter`
  * `projectId` becomes `projectID`
* `NetworkFilter`
  * `projectId` becomes `projectID`
* `SubnetFilter`
  * `projectId` becomes `projectID`
  * `gateway_ip` becomes `gatewayIP`
  * `ipv6RaMode` becomes `ipv6RAMode`
* `RouterFilter`
  * `projectId` becomes `projectID`
