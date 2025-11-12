# Priority Queue

> **Note:** PriorityQueue is available in >= 0.14

The `PriorityQueue` feature flag enables the usage of the controller-runtime PriorityQueue.

This feature deprioritizes reconciliation of objects that were not edge-triggered (i.e. due to an create/update etc.) and makes the controller more responsive during full resyncs and controller startups.

More information on controller-runtime PriorityQueue:
- [release-notes](https://github.com/kubernetes-sigs/controller-runtime/releases/tag/v0.20.0)
- [design docs](https://github.com/kubernetes-sigs/controller-runtime/pull/3013)
- [tracking issue](https://github.com/kubernetes-sigs/controller-runtime/issues/2374)

## Enabling Priority Queue

You can enable `PriorityQueue` using the following.

- Environment variable: `EXP_CAPO_PRIORITY_QUEUE=true`
- clusterctl.yaml variable: `EXP_CAPO_PRIORITY_QUEUE: true`
- --feature-gates argument: `PriorityQueue=true`
