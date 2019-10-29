# Getting started

## Prerequisites

1. Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
1. Install `kustomize` `v3.1.0+` (see [kustomize/releases](https://github.com/kubernetes-sigs/kustomize/releases))
1. Download the latest `v0.2.x` release of `clusterctl` from [cluster-api/releases](https://github.com/kubernetes-sigs/cluster-api/releases)
1. You can use either a VM, container or existing Kubernetes cluster as bootstrap cluster.
   - If you want to use VM, install [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/), version 0.30.0 or greater. Also install a [driver](https://github.com/kubernetes/minikube/blob/master/docs/drivers.md). For Linux, we recommend `kvm2`. For MacOS, we recommend `VirtualBox`.
   - If you want to use a container, install [Kind](https://github.com/kubernetes-sigs/kind#installation-and-usage).
   - If you want to use an existing Kubernetes cluster, prepare a kubeconfig which for this cluster.
1. The CAPO provider requires an OS image (available in OpenStack), which is build like the ones in [image-builder](https://github.com/kubernetes-sigs/image-builder/tree/master/images/capi) 

## Cluster Creation

1. Generate example YAML files if needed. You can use the `examples/generate.sh` script as documented [here](./../examples/README.md).

2. Create a cluster:
   - If you are using Minikube:

   ```bash
   clusterctl create cluster \
        --bootstrap-type minikube --bootstrap-flags kubernetes-version=v1.15.0 \
        -c examples/_out/cluster.yaml \
        -m examples/_out/controlplane.yaml \
        -p examples/_out/provider-components.yaml \
        -a examples/addons.yaml
   ```

   To choose a specific Minikube driver, please use the `--bootstrap-flags vm-driver=xxx` command line parameter. For example to use the `kvm2` driver with clusterctl you would add `--bootstrap-flags vm-driver=kvm2`, for linux, if you haven't installed any driver, you can add `--bootstrap-flags vm-driver=none`.

   - If you are using `Kind`:

   ```bash
   clusterctl create cluster \
           --bootstrap-type kind --bootstrap-cluster-cleanup=false \
           -c examples/_out/cluster.yaml \
           -m examples/_out/controlplane.yaml \
           -p examples/_out/provider-components.yaml \
           -a examples/addons.yaml
   ```

   - If you are using an existing Kubernetes cluster:

   ```bash
   clusterctl create cluster \
           --bootstrap-cluster-kubeconfig ~/.kube/config \
           -c examples/_out/cluster.yaml \
           -m examples/_out/controlplane.yaml \
           -p examples/_out/provider-components.yaml \
           -a examples/addons.yaml
   ```

### Interacting with your cluster

Once you have created a cluster, you can interact with the cluster and machine resources via `kubectl`:

```bash
export KUBECONFIG=./kubeconfig
kubectl get clusters
kubectl get machines
```

### Deploying additional machines

You can deploy additional machines via the `examples/_out/machinedeployment.yaml`. You probably have to customize the 
configuration of these machines.

## Cluster Deletion

Use the following command to delete a cluster and all resources it created.

```bash
clusterctl delete cluster --cluster <cluster-name> --bootstrap-type kind --kubeconfig kubeconfig --provider-components examples/_out/provider-components.yaml
```

Or you can manually delete all resources that were created as part of
your openstack Cluster API Kubernetes cluster.

1. Delete all of the worker machines in the cluster. Make sure to wait for the
  corresponding machines to be deleted before moving onto the next step. After this
  step, only the control plane node(s) will remain.

   ```bash
   kubectl --kubeconfig=kubeconfig delete machines -l set=node
   kubectl --kubeconfig=kubeconfig get nodes
   ```

2. Delete the control plane machines.
    ```bash
    kubectl --kubeconfig=kubeconfig delete machines -l set=master
    ```

3. (optional) Delete the load balancer in your OpenStack cloud if one has been created.

4. Delete the `kubeconfig` file that were created for your cluster.

   ```bash
   rm kubeconfig
   ```

5. Delete the ssh keypair that was created for your machines.

   ```bash
   rm -rf $HOME/.ssh/openstack_tmp*
   ```

## Troubleshooting

Please refer to [Trouble shooting documentation](./troubleshooting.md) for further info.
