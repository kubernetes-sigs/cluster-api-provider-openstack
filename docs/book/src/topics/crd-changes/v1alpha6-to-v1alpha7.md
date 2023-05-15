<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [v1alpha6 compared to v1alpha7](#v1alpha6-compared-to-v1alpha7)
  - [Migration](#migration)
  - [API Changes](#api-changes)
    - [`OpenStackCluster`](#openstackcluster)
    - [`OpenStackMachine`](#openstackmachine)
      - [Removal of Subnet](#removal-of-subnet)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# v1alpha6 compared to v1alpha7

> ⚠️ v1alpha7 has not been released yet.

## Migration

All users are encouraged to migrate their usage of the CAPO CRDs from older versions to `v1alpha6`. This includes yaml files and source code. As CAPO implements automatic conversions between the CRD versions, this migration can happen after installing the new CAPO release.

## API Changes

This only documents backwards incompatible changes. Fields that were added to v1alpha6 are not listed here.

### `OpenStackMachine`

#### ⚠️ Removal of Networks

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

#### Removal of Subnet

The OpenStackMachine spec previously contained a `subnet` field which could used
to set the `accessIPv4` field on Nova servers. This feature was not widely
used, difficult to use, and could not be extended to support IPv6. It is
removed without replacement.

#### Removal of Port SecurityGroups

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
- uuid: 60ed83f1-8886-41c6-a1c7-fcfbdf3f04c2
- uuid: 0ddd14d9-5c33-4734-b7d0-ac4fdf35c2d9
- uuid: 4a131d3e-9939-4a6b-adea-788a2e89fcd8
```
