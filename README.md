# Kubernetes cluster-api-provider-openstack Project

This repository hosts a concrete implementation of an OpenStack provider for the [cluster-api project](https://github.com/kubernetes-sigs/cluster-api).

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [#cluster-api-openstack on Kubernetes Slack](https://kubernetes.slack.com/messages/cluster-api-openstack)
- [SIG-Cluster-Lifecycle Mailing List](https://groups.google.com/forum/#!forum/kubernetes-sig-cluster-lifecycle)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

## Getting Started

### Prerequisites

1. Install `kubectl` (see [here](http://kubernetes.io/docs/user-guide/prereqs/)).
2. If you want to use VM, install [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/), version 0.30.0 or greater. If you want to use container, install [kind](https://github.com/kubernetes-sigs/kind#installation-and-usage).
3. Install a [driver](https://github.com/kubernetes/minikube/blob/master/docs/drivers.md) **if you are using minikube**. For Linux, we recommend kvm2. For MacOS, we recommend VirtualBox.
4. An appropriately configured [Go development environment](https://golang.org/doc/install)
5. Build the `clusterctl` tool

   ```bash
   git clone https://github.com/kubernetes-sigs/cluster-api-provider-openstack $GOPATH/src/sigs.k8s.io/cluster-api-provider-openstack
   cd $GOPATH/src/sigs.k8s.io/cluster-api-provider-openstack/cmd/clusterctl
   go build
   ```

### Cluster Creation

1. Create the `cluster.yaml`, `machines.yaml`, `provider-components.yaml`, and `addons.yaml` files if needed. If you want to use the generate-yaml.sh script, then you will need kustomize version 1.0.11, which can be found at https://github.com/kubernetes-sigs/kustomize/releases/tag/v1.0.11, and the latest go implementation of yq, which can be found at https://github.com/mikefarah/yq. The script has the following usage:

   ```bash
   cd examples/openstack
   ./generate-yaml.sh [options] <path/to/clouds.yaml> <openstack cloud> <provider os>
   cd ../..
   ```
   `<provider os>` specifies the operating system of the virtual machines Kubernetes will run on.
   Supported Operating Systems:
   - `ubuntu`
   - `centos`

   #### Quick notes on clouds.yaml
   We no longer support generating clouds.yaml. You should be able to get a valid clouds.yaml from your openstack cluster. However, make sure that the following fields are included, and correct.

   - `username`
   - `user_domain_name`
   - `project_id`
   - `region_name`
   - `auth_url`
   - `password`

   You **will need** to make changes to the generated files to create a working cluster.
   You can find some guidance on what needs to be edited, and how to create some of the
   required OpenStack resources in the [Configuration documentation](docs/config.md).

   #### Special notes on ssh keys and fetching `admin.conf`

   When running `generate-yaml.sh` the first time, a new ssh keypair is generated and stored as `$HOME/.ssh/openstack_tmp` and `$HOME/.ssh/openstack_tmp.pub`. In order to allow `clusterctl` to fetch Kubernetes' `admin.conf` from the master node, you **must** manually create the key pair in OpenStack. By default the generated `machine.yaml` uses `cluster-api-provider-openstack` to be the `keyName`. However, you are free to change that.

   e.g.
   ```
   openstack keypair create --public-key ~/.ssh/openstack_tmp.pub cluster-api-provider-openstack
   ```

2. Create a cluster:
   - If you are using minikube:

   ```bash
   ./clusterctl create cluster --bootstrap-type minikube --bootstrap-flags kubernetes-version=v1.12.3 \
     --provider openstack -c examples/openstack/out/cluster.yaml \
     -m examples/openstack/out/machines.yaml -p examples/openstack/out/provider-components.yaml
   ```

   To choose a specific minikube driver, please use the `--bootstrap-flags vm-driver=xxx` command line parameter. For example to use the kvm2 driver with clusterctl you woud add `--bootstrap-flags vm-driver=kvm2`, for linux, if you haven't installed any driver, you can add `--bootstrap-flags vm-driver=none`.

   - If you are using kind:

   ```bash
   ./clusterctl create cluster --bootstrap-type kind --provider openstack \
     -c examples/openstack/out/cluster.yaml -m examples/openstack/out/machines.yaml \
     -p examples/openstack/out/provider-components.yaml
   ```

Additional advanced flags can be found via help.

```bash
./clusterctl create cluster --help
```

### Managed OpenStack Security Groups

In `Cluster.spec.ProviderSpec` there is a boolean option called `ManagedSecurityGroups` that, if set to `true`, will create a default set of security groups for the cluster. These are meant for a "standard" setup, and might not be suitable for every environment. Please review the rules below before you use them.

*NOTE*: For now, there is no way to automatically use these rules, which makes them a bit cumbersome to use, this will be possible in the near future.

The rules created are:

* A rule for the controlplane machine, that allows access from everywhere to port 22 and 443.
* A rule for all the machines, both the controlplane and the nodes that allow all traffic between members of this group.

### Interacting with your cluster

If you are using kind, config the `KUBECONFIG` first before using kubectl:

```bash
export KUBECONFIG="$(kind get kubeconfig-path --name="clusterapi")"
```

Once you have created a cluster, you can interact with the cluster and machine
resources using kubectl:

```bash
kubectl --kubeconfig=kubeconfig get clusters
kubectl --kubeconfig=kubeconfig get machines
kubectl --kubeconfig=kubeconfig get machines -o yaml
```

### Cluster Deletion

This guide explains how to delete all resources that were created as part of
your openstack Cluster API Kubernetes cluster.

1. Delete all of the node Machines in the cluster. Make sure to wait for the
  corresponding Nodes to be deleted before moving onto the next step. After this
  step, the master node will be the only remaining node.

   ```bash
   kubectl --kubeconfig=kubeconfig delete machines -l set=node
   kubectl --kubeconfig=kubeconfig get nodes
   ```

2. Delete the master machine.
    ```bash
    kubectl --kubeconfig=kubeconfig delete machines -l set=master
    ```

3. Delete the kubeconfig file that were created for your cluster.

   ```bash
   rm kubeconfig
   ```

4. Delete the ssh keypair that were created for your cluster machine.

   ```bash
   rm -rf $HOME/.ssh/openstack_tmp*
   ```
