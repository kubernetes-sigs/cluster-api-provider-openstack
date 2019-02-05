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
2. Install [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/), version 0.30.0 or greater.
3. Install a [driver](https://github.com/kubernetes/minikube/blob/master/docs/drivers.md) for minikube. For Linux, we recommend kvm2. For MacOS, we recommend VirtualBox.
4. An appropriately configured [Go development environment](https://golang.org/doc/install)
5. Build the `clusterctl` tool

   ```bash
   git clone https://github.com/kubernetes-sigs/cluster-api-provider-openstack $GOPATH/src/sigs.k8s.io/cluster-api-provider-openstack
   cd $GOPATH/src/sigs.k8s.io/cluster-api-provider-openstack/cmd/clusterctl
   go build
   ```

### Cluster Creation

1. Create the `cluster.yaml`, `machines.yaml`, `provider-components.yaml`, and `addons.yaml` files if needed:

   ```bash
   cd examples/openstack
   ./generate-yaml.sh --provider-os [os name] [options]
   cd ../..
   ```
   [os name] is the operating system of your provider environment. 
   Supported Operating Systems: 
   - `ubuntu` 
   - `centos`

   #### Interactively submit provider information
   By default, the generater script will give you a series of command line prompts, asking the following information about your cloud provider:

   - `user-name`
   - `domain-name`
   - `project-id`
   - `region-name`
   - `auth-url`
   - `password`

   #### Use clouds.yaml to submit provider information
   If you want to generate scripts without being prompted interactively, you can pass generate-yaml a clouds.yaml file. After downloading your clouds.yaml from your provider, make sure that it has the information listed above filled out. It is very likely that it will at lest be missing the password field. Also, note that domain-name is the same as project-name. You may reference the following sample clouds.yaml to see what yours should look like.

   ```yaml
   clouds:
     openstack:
       auth:
         auth_url: https://yourauthurl:5000/v3
         username: foo
         password: bar
         project_id: foobar123
         project_name: foobar
         user_domain_name: "Default"
       region_name: "Region_1"
       interface: "public"
       identity_api_version: 3
   ```

   To specify which cloud to use, set the OS_CLOUD environment variable with its name. By default, the generator will use the cloud "openstack". Based on the example above, the following command sets the correct cloud:

   ```bash
   export OS_CLOUD=openstack
   ```

   To pass a clouds.yaml file to generate-yaml, set the **-c** or **--clouds** options, followed by the path to a clouds.yaml file. Here are some examples of this syntax:

   ```bash
   ./generate-yaml.sh --provider-os [os name] -c clouds.yaml
   ./generate-yaml.sh --provider-os [os name] --clouds clouds.yaml
   ```

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

   ```bash
   ./clusterctl create cluster --bootstrap-type minikube --bootstrap-flags kubernetes-version=v1.12.3 --provider openstack -c examples/openstack/[os name]/out/cluster.yaml -m examples/openstack/[os name]/out/machines.yaml -p examples/openstack/[os name]/out/provider-components.yaml
   ```

To choose a specific minikube driver, please use the `--bootstrap-flags vm-driver=xxx` command line parameter. For example to use the kvm2 driver with clusterctl you woud add `--bootstrap-flags vm-driver=kvm2`, for linux, if you haven't installed any driver, you can add `--bootstrap-flags vm-driver=none`.

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
