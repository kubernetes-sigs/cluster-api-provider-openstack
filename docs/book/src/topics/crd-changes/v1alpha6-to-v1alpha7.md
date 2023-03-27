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

### `OpenStackCluster`

### `OpenStackMachine`

#### Removal of Subnet

The OpenStackMachine spec previously contained a `subnet` field which could used
to set the `accessIPv4` field on Nova servers. This feature was not widely
used, difficult to use, and could not be extended to support IPv6. It is
removed without replacement.
