# Experimental Features

CAPO now ships with experimental features the users can enable.

Currently CAPO has the following experimental features:
* `PriorityQueue` (env var: `EXP_CAPO_PRIORITY_QUEUE`): [PriorityQueue](./priority-queue.md)

## Enabling Experimental Features for Management Clusters Started with clusterctl

Users can enable/disable features by setting OS environment variables before running `clusterctl init`, e.g.:

```yaml
export EXP_SOME_FEATURE_NAME=true

clusterctl init --infrastructure openstack
```

As an alternative to environment variables, it is also possible to set variables in the clusterctl config file located at `$XDG_CONFIG_HOME/cluster-api/clusterctl.yaml`, e.g.:
```yaml
# Values for environment variable substitution
EXP_SOME_FEATURE_NAME: "true"
```
In case a variable is defined in both the config file and as an OS environment variable, the environment variable takes precedence.
For more information on how to set variables for clusterctl, see [clusterctl Configuration File](https://cluster-api.sigs.k8s.io/clusterctl/configuration)


## Enabling Experimental Features on Existing Management Clusters

To enable/disable features on existing management clusters, users can edit the controller manager
deployments, which will then trigger a restart with the requested features. E.g:

```
kubectl edit -n capo-system deployment.apps/capo-controller-manager
```
```
// Enable/disable available features by modifying Args below.
spec:
  template:
    spec:
      containers:
      - args:
        - --leader-elect
        - --feature-gates=SomeFeature=true,OtherFeature=false
```

Similarly, to **validate** if a particular feature is enabled, see the arguments by issuing:

```bash
kubectl describe -n capo-system deployment.apps/capo-controller-manager
```

## Enabling Experimental Features for e2e Tests

Features can be enabled by setting them as environmental variables before running e2e tests.

For `ci` this can also be done through updating `./test/e2e/data/e2e_conf.yaml`.

## Enabling Experimental Features on Tilt

On development environments started with `Tilt`, features can be enabled by setting the feature variables in `kustomize_substitutions`, e.g.:

```yaml
kustomize_substitutions:
  EXP_CAPO_PRIORITY_QUEUE: 'true'
```

For more details on setting up a development environment with `tilt`, see [Developing with Tilt](../development/development.md#developing-with-tilt)

