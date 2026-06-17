# Priority Queue

> **Note:** PriorityQueue is available as an alpha feature in 0.14 (disabled by
> default) and graduated to beta in 0.15 (enabled by default).

The `PriorityQueue` feature flag enables the usage of the controller-runtime PriorityQueue.

This feature deprioritizes reconciliation of objects that were not edge-triggered (i.e. due to an create/update etc.) and makes the controller more responsive during full resyncs and controller startups.

More information on controller-runtime PriorityQueue:
- [release-notes](https://github.com/kubernetes-sigs/controller-runtime/releases/tag/v0.20.0)
- [design docs](https://github.com/kubernetes-sigs/controller-runtime/pull/3013)
- [tracking issue](https://github.com/kubernetes-sigs/controller-runtime/issues/2374)

## Feature gate maturity

| Version | Stage | Default  |
|---------|-------|----------|
| 0.14    | Alpha | Disabled |
| 0.15    | Beta  | Enabled  |

## Enabling/Disabling Priority Queue

To enable:
- Environment variable: `EXP_CAPO_PRIORITY_QUEUE=true`
- clusterctl.yaml variable: `EXP_CAPO_PRIORITY_QUEUE: true`
- --feature-gates argument: `PriorityQueue=true`

To disable:
- Environment variable: `EXP_CAPO_PRIORITY_QUEUE=false`
- clusterctl.yaml variable: `EXP_CAPO_PRIORITY_QUEUE: false`
- `--feature-gates` argument: `PriorityQueue=false`
