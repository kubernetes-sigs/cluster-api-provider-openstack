# AutoScale From Zero

> **Note:** AutoScaleFromZero is available in >= 0.14.

The `AutoScaleFromZero` feature flag enables the usage of [cluster-autoscaler](https://github.com/kubernetes/autoscaler/tree/bc3f44c85df17bccc940adb7c885b192cf6135d7/cluster-autoscaler/cloudprovider/clusterapi#cluster-autoscaler-on-cluster-api) to scale from/to zero without the need of annotations. More information on how to use the cluster-autoscaler can be found [here](https://github.com/kubernetes/autoscaler/tree/bc3f44c85df17bccc940adb7c885b192cf6135d7/cluster-autoscaler/cloudprovider/clusterapi#scale-from-zero-support).

## Enabling AutoScaleFromZero

You can enable `AutoScaleFromZero` using the following.

- Environment variable: `EXP_CAPO_AUTOSCALE_FROM_ZERO=true`
- clusterctl.yaml variable: `EXP_CAPO_AUTOSCALE_FROM_ZERO: true`
- --feature-gates argument: `AutoScaleFromZero=true`

## Automatically Populated Status Fields

> **Note**: Unsupported fields may be provided via annotations or incorporated into the controller by extending its functionality.

The controller automatically fills two sections of `OpenStackMachineTemplate.Status`:  
- **capacity** (resource quantities)  
- **nodeInfo** (OS metadata)  

The following mappings describe exactly where each value originates.

### Capacity (`Status.Capacity`)
- **CPU**: From the `VCPUs` property of the resolved OpenStack flavor

- **Memory**: From the `RAM` property of the resolved OpenStack flavor

- **Ephemeral Storage**: From the `Ephemeral` property of the resolved OpenStack flavor

- **Root Storage**: Determined based on the boot method:  
  - If **booting from volume** taken from `OpenStackMachineTemplate.Spec.Template.Spec.RootVolume.SizeGiB`  
  - If **booting from image** taken from the `Disk` property of the resolved OpenStack flavor

### Node Information (`Status.NodeInfo`)
- **Operating System**: From the `os_type` property of the resolved OpenStack image.
