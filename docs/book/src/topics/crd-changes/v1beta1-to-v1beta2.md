<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [v1beta1 compared to v1beta2](#v1beta1-compared-to-v1beta2)
  - [Migration](#migration)
  - [API Changes](#api-changes)
    - [Conditions format change](#conditions-format-change)
    - [Removal of deprecated status fields](#removal-of-deprecated-status-fields)
    - [FailureDomains representation change](#failuredomains-representation-change)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# v1beta1 compared to v1beta2

## Migration

All users are encouraged to migrate their usage of the CAPO CRDs from `v1beta1` to `v1beta2`. This includes yaml files and source code. As CAPO implements automatic conversion webhooks between the CRD versions, this migration can happen after installing the new CAPO release.

**For most users, no action is required.** The conversion webhooks handle all translation between v1beta1 and v1beta2 automatically. The changes below are relevant primarily for developers writing controllers or tooling that reads CAPO objects directly.

There are **no breaking changes to spec fields**. All spec fields from v1beta1 are present and unchanged in v1beta2. The breaking changes are limited to status fields and internal representations.

## API Changes

This only documents backwards incompatible changes. Fields that were added to v1beta2 are not listed here.

### Conditions format change

Conditions have changed from CAPI v1beta1 `Conditions` type to standard Kubernetes `[]metav1.Condition`.

In v1beta1:
```yaml
status:
  conditions:
  - type: Ready
    status: "True"
    severity: Info
    lastTransitionTime: "2024-01-01T00:00:00Z"
    reason: AllComponentsReady
    message: "All components are ready"
```

In v1beta2:
```yaml
status:
  conditions:
  - type: Ready
    status: "True"
    observedGeneration: 3
    lastTransitionTime: "2024-01-01T00:00:00Z"
    reason: AllComponentsReady
    message: "All components are ready"
```

Key differences:
- The `severity` field is removed (not present in `metav1.Condition`).
- The `observedGeneration` field is added.
- The `status` field uses `metav1.ConditionStatus` (`"True"`, `"False"`, `"Unknown"`) instead of `corev1.ConditionStatus`. The string values are identical, but the Go types differ.

This affects `OpenStackCluster`, `OpenStackMachine`, `OpenStackServer`, and `OpenStackFloatingIPPool`.

### Removal of deprecated status fields

The following deprecated status fields have been removed from v1beta2:

**`OpenStackCluster`:**
- `status.ready` — now derived from the `Ready` condition.
- `status.failureReason` — replaced by condition `Reason` fields.
- `status.failureMessage` — replaced by condition `Message` fields.

**`OpenStackMachine`:**
- `status.ready` — now derived from the `Ready` condition.
- `status.failureReason` — replaced by condition `Reason` fields.
- `status.failureMessage` — replaced by condition `Message` fields.

If your code reads `status.ready`, use the `Ready` condition instead:

```go
// v1beta1
if cluster.Status.Ready {
    // ...
}

// v1beta2
import "k8s.io/apimachinery/pkg/api/meta"

readyCondition := meta.FindStatusCondition(cluster.Status.Conditions, "Ready")
if readyCondition != nil && readyCondition.Status == metav1.ConditionTrue {
    // ...
}
```

### FailureDomains representation change

`FailureDomains` in `OpenStackCluster` status changed from a map to a slice.

In v1beta1:
```yaml
status:
  failureDomains:
    az-1:
      controlPlane: true
      attributes:
        region: us-east-1
    az-2:
      controlPlane: false
```

In v1beta2:
```yaml
status:
  failureDomains:
  - name: az-1
    controlPlane: true
    attributes:
      region: us-east-1
  - name: az-2
    controlPlane: false
```

The conversion webhook handles this automatically. The slice is sorted by name for deterministic ordering.
