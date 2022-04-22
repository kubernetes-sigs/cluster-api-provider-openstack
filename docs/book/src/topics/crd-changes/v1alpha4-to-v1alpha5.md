<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [v1alpha4 compared to v1alpha5](#v1alpha4-compared-to-v1alpha5)
  - [Conversion from v1alpha4 to v1alpha5](#conversion-from-v1alpha4-to-v1alpha5)
  - [API Changes](#api-changes)
    - [`OpenStackCluster`](#openstackcluster)
      - [Managed API LoadBalancer](#managed-api-loadbalancer)
      - [Major Changes to Ports and Networks](#major-changes-to-ports-and-networks)
    - [`OpenStackMachine`](#openstackmachine)
      - [Rename of `status.error{Reason,Message}` to `status.failure{Reason,Message}`](#rename-of-statuserrorreasonmessage-to-statusfailurereasonmessage)
      - [Changes to `rootVolume`](#changes-to-rootvolume)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# v1alpha4 compared to v1alpha5

## Migration


All users are encouraged to migrate their usage of the CAPO CRDs from older versions to `v1alpha5`. This includes yaml files and source code. As CAPO implements automatic conversions between the CRD versions, this migration can happen after installing the new CAPO release.

## API Changes

This only documents backwards incompatible changes. Fields that were added to v1alpha5 are not listed here.

### `OpenStackCluster`

#### Managed API LoadBalancer

The fields related to the managed API LoadBalancer were moved into a seperate object:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha4
kind: OpenStackCluster
spec:
  managedAPIServerLoadBalancer: true
  apiServerLoadBalancerAdditionalPorts: [443]
```

becomes:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha5
kind: OpenStackCluster
spec:
  apiServerLoadBalancer:
    enabled: true
    additionalPorts: [443]
```

### `OpenStackMachine`

#### Major Changes to Ports and Networks

When using Ports it is now possible to specify network and subnet by filter instead of just ID. As a consequence, the relevant ID fields are now moved into the new filter specifications:

```yaml
ports:
  - networkId: d-e-a-d
    fixedIPs:
      - subnetId: b-e-e-f
```

becomes:

```yaml
ports:
  - network:
      id: d-e-a-d
    fixedIPs:
      subnet:
        id: b-e-e-f
```

Networks are now deprecated. With one exception, all functionality of Networks is now available for Ports. Consequently, Networks will be removed from the API in a future release.

The ability of a Network to add multiple ports with a single directive will not be added to Ports. When moving to Ports, all ports must be added explicitly. Specifically, when evaluating the network or subnet filter of a Network, if there are multiple matches we will add all of these to the server. By contrast we raise an error if the network or subnet filter of a Port does not return exactly one result.

`tenantId` was previously a synonym for `projectId` in both network and subnet filters. This has now been removed. Use `projectId` instead.

The following fields are removed from network and subnet filters without replacement:

- status
- adminStateUp
- shared
- marker
- limit
- sortKey
- sortDir
- subnetPoolId

#### Rename of `status.error{Reason,Message}` to `status.failure{Reason,Message}`

The actual fields were previously already renamed, but we still used the `error` prefix in JSON. This was done to align with CAPI, where these fields were [renamed in v1alpha3](https://cluster-api.sigs.k8s.io/developer/providers/v1alpha2-to-v1alpha3.html#external-objects-will-need-to-rename-statuserrorreason-and-statuserrormessage).

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha4
kind: OpenStackMachine
status:
  errorReason: UpdateError
  errorMessage: Something when wrong
```

becomes:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha5
kind: OpenStackMachine
status:
  failureReason: UpdateError
  failureMessage: Something when wrong
```

#### Changes to `rootVolume`

The following fields were removed without replacement:

- `rootVolume.deviceType`
- `rootVolume.sourceType`

Additionally, `rootVolume.sourceUUID` has been replaced by using `ImageUUID` or `Image` from the OpenStackMachine as appropriate.