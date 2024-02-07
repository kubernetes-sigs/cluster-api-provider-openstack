<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [v1alpha6 compared to v1alpha7](#v1alpha6-compared-to-v1alpha7)
  - [Migration](#migration)
  - [API Changes](#api-changes)
    - [`OpenStackMachine`](#openstackmachine)
      - [⚠️ Removal of networks](#-removal-of-networks)
      - [Removal of subnet](#removal-of-subnet)
      - [Change to securityGroups](#change-to-securitygroups)
      - [Changes to ports](#changes-to-ports)
        - [Change to securityGroupFilters](#change-to-securitygroupfilters)
        - [Removal of securityGroups](#removal-of-securitygroups)
        - [Removal of tenantId and projectId](#removal-of-tenantid-and-projectid)
        - [Change to profile](#change-to-profile)
      - [Creation of additionalBlockDevices](#creation-of-additionalblockdevices)
    - [`OpenStackCluster`](#openstackcluster)
      - [Change to externalRouterIPs.subnet](#change-to-externalrouteripssubnet)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# v1alpha6 compared to v1alpha7

> ⚠️ v1alpha7 has not been released yet.

## Migration

All users are encouraged to migrate their usage of the CAPO CRDs from older versions to `v1alpha6`. This includes yaml files and source code. As CAPO implements automatic conversions between the CRD versions, this migration can happen after installing the new CAPO release.

## API Changes

This only documents backwards incompatible changes. Fields that were added to v1alpha6 are not listed here.

### `OpenStackMachine`

#### ⚠️ Removal of networks

This is a major breaking change between v1alpha6 and v1alpha7 which in certain circumstances **may require manual action before upgrading to v0.8**.

v1alpha6 allowed network attachments to an OpenStackMachine to be specified as either Networks or Ports. In v1alpha7, Networks are removed. Network attachments may only be specified as ports.

In most cases Networks will be automatically converted to equivalent Ports on upgrade. However, this is not supported in the case where a Network specifies a network or subnet filter which returns more than one OpenStack resource. In this case it is important to rewrite any affected OpenStackMachineTemplates and wait for any consequent rollout to complete prior to updating to version 0.8.

Your configuration is affected if it contains any Network or Subnet filter which returns multiple resources. In a v1alpha6 Network definition this resulted in the creation of multiple ports: one for each returned result. In a Port definition, filters may only return a single resource and throw an error if multiple resources are returned.

For example, take this example OpenStackMachineTemplate specification:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha6
kind: OpenStackMachineTemplate
metadata:
  name: ${CLUSTER_NAME}-md-0
spec:
  template:
    spec:
      ..
      networks:
      - filter:
          tags: tag-matches-multiple-networks
        subnets:
        - filter:
            tags: tag-matches-multiple-subnets
```

In this configuration both the network and subnet filters match multiple resources. In v0.8 this will be automatically converted to:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha7
kind: OpenStackMachineTemplate
metadata:
  name: ${CLUSTER_NAME}-md-0
spec:
  template:
    spec:
      ..
      ports:
      - network:
          tags: tag-matches-multiple-networks
        fixedIPs:
        - subnet:
            tags: tag-matches-multiple-subnets
```

However, this will cause an error when reconciled by the machine controller, because in a port:
* a network filter may only return a single network
* a subnet filter may only return a single subnet

Instead it needs to be rewritten prior to upgrading to version 0.8. It can be rewritten as either ports or networks, as long as each results in the creation of only a single port. For example, rewriting without converting to ports might give:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha6
kind: OpenStackMachineTemplate
metadata:
  name: ${CLUSTER_NAME}-md-0
spec:
  template:
    spec:
      ..
      networks:
      - filter:
          name: network-a
        subnets:
        - filter:
            name: subnet-a
      - filter:
          name: network-b
        subnets:
        - filter:
            name: subnet-b
```

This will be safely converted to use ports when upgrading to version 0.8.

To reiterate: it is not sufficient to leave your templates at v1alpha6 in this case, as it will still result in failure to reconcile in the machine controller. This change must be made prior to updating to version 0.8.

#### Removal of subnet

The OpenStackMachine spec previously contained a `subnet` field which could used
to set the `accessIPv4` field on Nova servers. This feature was not widely
used, difficult to use, and could not be extended to support IPv6. It is
removed without replacement.

#### Change to securityGroups

`securityGroups` has been simplified by the removal of a separate filter parameter. It was previously:
```yaml
securityGroups:
  uuid: ...
  name: ...
  filter:
    description: ...
    tags: ...
    ...
```
It becomes:
```yaml
securityGroups:
  id: ...
  name: ...
  description: ...
  tags: ...
  ...
```

Note that in this change, the `uuid` field has become `id`. So:
```yaml
securityGroups:
- uuid: 4ea83db6-2760-41a9-b25a-e625a1161ed0
```
becomes:
```yaml
securityGroups:
- id: 4ea83db6-2760-41a9-b25a-e625a1161ed0
```

The `limit`, `marker`, `sortKey`, `sortDir`, fields have been removed without replacement. They did not serve any purpose in this context.

The `tenantId` parameter has been removed. Use `projectId` instead.

#### Changes to ports

##### Change to securityGroupFilters

The same change is made to `securityGroupFilters` in `ports` as is [made to `securityGroups` in the machine spec](#change-to-securitygroups).

##### Removal of securityGroups

The Port field of the OpenStackMachine spec previously contained both `securityGroups` and `securityGroupFilters`.
As `securityGroups` can easily be replaced with `securityGroupFilters`, that can express the same and more, `securityGroups` has now been removed.
CAPO can automatically convert `securityGroups` to `securityGroupFilters` when upgrading.

Here is an example of how to use `securityGroupFilters` to replace `securityGroups`:

```yaml
# securityGroups are available in v1alpha6
securityGroups:
- 60ed83f1-8886-41c6-a1c7-fcfbdf3f04c2
- 0ddd14d9-5c33-4734-b7d0-ac4fdf35c2d9
- 4a131d3e-9939-4a6b-adea-788a2e89fcd8
# securityGroupFilters are available in both v1alpha6 and v1alpha7
securityGroupFilters:
- id: 60ed83f1-8886-41c6-a1c7-fcfbdf3f04c2
- id: 0ddd14d9-5c33-4734-b7d0-ac4fdf35c2d9
- id: 4a131d3e-9939-4a6b-adea-788a2e89fcd8
```

##### Removal of tenantId and projectId

These are removed without replacement. They required admin permission to set, which CAPO does not have by default, and served no purpose.

##### Change to profile

We previously allowed to use the Neutron `binding:profile` via `ports.profile` but this API is not declared as stable from the [Neutron API description](https://docs.openstack.org/api-ref/network/v2/index.html?expanded=create-port-detail#create-port).

Instead, we now explicitly support two use cases:
* OVS Hardware Offload
* Trusted Virtual Functions (VF)

Note that the conversion is lossy, we only support the two use cases above so if anyone was relying on anything other than
the supported behaviour, it will be lost.

Here is an example on how to use `ports.profile` for enabling OVS Hardware Offload:

```yaml
profile:
  OVSHWOffload: true
```

Here is an example on how to use `ports.bindingProfile` for enabling "trusted-mode" to the VF:

```yaml
profile:
  TrustedVF: true
```

#### Creation of additionalBlockDevices

We now have the ability for a machine to have additional block devices to be attached.

Here is an example on how to use `additionalBlockDevices` for adding an additional Cinder volume attached
to the server instance:

```yaml
additionalBlockDevices:
- name: database
  sizeGiB: 50
  storage:
    type: Volume
```

Here is an example on how to use `additionalBlockDevices` for adding an additional Cinder volume attached
to the server instance with an availability zone and a cinder type:

```yaml
additionalBlockDevices:
- name: database
  sizeGiB: 50
  storage:
    type: Volume
    volume:
      type: my-volume-type
      availabilityZone: az0
```

Here is an example on how to attach a ephemeral disk to the instance:

```yaml
additionalBlockDevices
- name: disk1
  sizeGiB: 1
  storage:
    type: local
```

Adding more than one ephemeral disk to an instance is possible but you should use it at your own risks, it has been
known to cause some issues in some environments.

### `OpenStackCluster`

#### Change to externalRouterIPs.subnet

The `uuid` field is renamed to `id`, and all fields from `filter` are moved directly into the `subnet`.

```yaml
externalRouterIPs:
- subnet:
    uuid: f23bf9c1-8c66-4383-b474-ada1d1960149
- subnet:
    filter:
      name: my-subnet
```
becomes:
```yaml
externalRouterIPs:
- subnet:
    id: f23bf9c1-8c66-4383-b474-ada1d1960149
- subnet:
    name: my-subnet
```

#### status.router and status.apiServerLoadBalancer moved out of status.network

```yaml
status:
  network:
    id: 756f59c0-2a9b-495e-9bb1-951762523d2d
    name: my-cluster-network
    ...
    router:
      id: dd0b23a7-e785-4061-93c5-464843e8cc39
      name: my-cluster-router
      ...
    apiServerLoadBalancer:
      id: 660d628e-cbcb-4c10-9910-e2e6493643c7
      name: my-api-server-loadbalancer
      ...
```
becomes:
```yaml
status:
  network:
    id: 756f59c0-2a9b-495e-9bb1-951762523d2d
    name: my-cluster-network
    ...
  router:
    id: dd0b23a7-e785-4061-93c5-464843e8cc39
    name: my-cluster-router
    ...
  apiServerLoadBalancer:
    id: 660d628e-cbcb-4c10-9910-e2e6493643c7
    name: my-api-server-loadbalancer
    ...
```

#### status.network.subnet becomes status.network.subnets

```yaml
status:
  network:
    id: 756f59c0-2a9b-495e-9bb1-951762523d2d
    name: my-cluster-network
    subnet:
      id: 0e0c3d69-040a-4b51-a3f5-0f5d010b36f4
      name: my-cluster-subnet
      cidr: 192.168.100.0/24
```
becomes
```yaml
  network:
    id: 756f59c0-2a9b-495e-9bb1-951762523d2d
    name: my-cluster-network
    subnets:
    - id: 0e0c3d69-040a-4b51-a3f5-0f5d010b36f4
      name: my-cluster-subnet
      cidr: 192.168.100.0/24
```

Nothing will currently create more than a single subnet, but there may be multiple subnets in the future. Similarly, code should no longer assume that the CIDR is an IPv4 CIDR, although nothing will currently create anything other than an IPv4 CIDR.
