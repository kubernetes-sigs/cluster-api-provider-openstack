# Kubernetes cluster-api-provider-openstack Project

This repository hosts a concrete implementation of an OpenStack provider for the [cluster-api project](https://github.com/kubernetes-sigs/cluster-api).

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [#cluster-api on Kubernetes Slack](http://kubernetes.slack.com/messages/cluster-api)
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
   ./generate-yaml.sh --provider-os [os name]
   cd ../..
   ```
   [os name] is the operating system of your provider environment. 
   Supported Operating Systems: 
   - `ubuntu` 
   - `centos`

   #### Interactively submit provider information
   By default, the generater script will give you a series of command line prompts, asking the following information about your cloud provider:

   - user-name
   - domain-name
   - project-id
   - region-name
   - auth-url
   - password

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

2. Create a cluster:

   ```bash
   ./clusterctl create cluster --minikube kubernetes-version=v1.12.3 --provider openstack -c examples/openstack/[os name]/out/cluster.yaml -m examples/openstack/[os name]/out/machines.yaml -p examples/openstack/[os name]/out/provider-components.yaml
   ```

To choose a specific minikube driver, please use the `--vm-driver` command line parameter. For example to use the kvm2 driver with clusterctl you woud add `--vm-driver kvm2`, for linux, if you haven't installed any driver, you can add `--vm-driver none`.

Additional advanced flags can be found via help.

```bash
./clusterctl create cluster --help
```

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

3. Delete the ssh keypair that were created for your cluster machine.

   ```bash
   rm -rf $HOME/.ssh/openstack_tmp*
   ```
