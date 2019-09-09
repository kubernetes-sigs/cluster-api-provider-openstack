<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Kubernetes cluster-api-provider-openstack Project](#kubernetes-cluster-api-provider-openstack-project)
  - [Community, discussion, contribution, and support](#community-discussion-contribution-and-support)
    - [Code of conduct](#code-of-conduct)
  - [Compatibility with Cluster API, Kubernetes and OpenStack Versions](#compatibility-with-cluster-api-kubernetes-and-openstack-versions)
  - [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
    - [Cluster Creation](#cluster-creation)
    - [Managed OpenStack Security Groups](#managed-openstack-security-groups)
    - [Interacting with your cluster](#interacting-with-your-cluster)
    - [Cluster Deletion](#cluster-deletion)
    - [Trouble shooting](#trouble-shooting)
  - [Contributing](#contributing)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Kubernetes cluster-api-provider-openstack Project

This repository hosts a concrete implementation of an OpenStack provider for the [cluster-api project](https://github.com/kubernetes-sigs/cluster-api).

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [#cluster-api-openstack on Kubernetes Slack](https://kubernetes.slack.com/messages/cluster-api-openstack)
- [SIG-Cluster-Lifecycle Mailing List](https://groups.google.com/forum/#!forum/kubernetes-sig-cluster-lifecycle)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

------

## Compatibility with Cluster API, Kubernetes and OpenStack Versions

This provider's versions are compatible with the following versions of Cluster API:

||Cluster API v1alpha1 (v0.1)|
|-|-|
|OpenStack Provider v1alpha1 (ea309e7f)|✓|

This provider's versions are able to install and manage the following versions of Kubernetes:

||Kubernetes 1.13.5+|Kubernetes 1.14|Kubernetes 1.15|
|-|-|-|-|
|OpenStack Provider v1alpha1 (ea309e7f)|✓|✓|✓|

Kubernetes control plane and Kubelet versions are defined in `spec.versions.controlPlane` and `spec.versions.kubelet` of `cmd/clusterctl/examples/openstack/machines.yaml.template` respectively.
You can generate `cmd/clusterctl/examples/openstack/out/machines.yaml` by running the `generate-yaml.sh` from the template and change the versions if you want.

**NOTE**: Because the user is able to customize any `user-data`, it is also possible to deploy older versions.
But we won't provide any examples or working templates. See [user-data in the examples](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/tree/master/cmd/clusterctl/examples/openstack/provider-component/user-data).

This provider's versions are able to install kubernetes to the following versions of OpenStack:

||OpenStack Pike|OpenStack Queens|OpenStack Rocky|OpenStack Stein|
|-|-|-|-|-|
|OpenStack Provider v1alpha1 (ea309e7f)|✓|✓|✓|✓|

Each version of Cluster API for OpenStack will attempt to support two Kubernetes versions.

**NOTE:** As the versioning for this project is tied to the versioning of Cluster API, future modifications to this
policy may be made to more closely align with other providers in the Cluster API ecosystem.

------

## Getting Started

### Notice
Currently `cluster-api-provider-openstack` project is evolving into `cluster-api v1alpha2`, please use `release-0.1` branch for `cluster-api v1alpha1` development as it provides function workable code and configurations.

For more information, please refer to [v1alpha2](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/issues/380)

### Prerequisites

1. Install `kustomize` v3.1.0+ (see [here](https://github.com/kubernetes-sigs/kustomize/releases).
2. You can use either VM, container or existing Kubernetes cluster act as bootstrap cluster.
   - If you want to use VM, install [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/), version 0.30.0 or greater.
   - If you want to use container, install [kind](https://github.com/kubernetes-sigs/kind#installation-and-usage).
   - If you want to use existing Kubernetes cluster, prepare your kubeconfig.
3. Install a [driver](https://github.com/kubernetes/minikube/blob/master/docs/drivers.md) **if you are using Minikube**. For Linux, we recommend kvm2. For MacOS, we recommend VirtualBox.
4. An appropriately configured [Go development environment](https://golang.org/doc/install)
5. Build the `clusterctl` tool and make it available in your `PATH`

   ```bash
   git clone https://github.com/kubernetes-sigs/cluster-api ${GOPATH}/src/sigs.k8s.io/cluster-api
   cd ${GOPATH}/src/sigs.k8s.io/cluster-api/
   make clusterctl
   ```

### Cluster Creation

1. Create the YAML files if needed. You can use the `examples/generate.sh` script as documented [here](examples/README.md).

2. Create a cluster:
   - If you are using Minikube:

   ```bash
   clusterctl create cluster \
        --bootstrap-type minikube --bootstrap-flags kubernetes-version=v1.15.0 \
        -c examples/_out/cluster.yaml \
        -m examples/_out/machines.yaml \
        -p examples/_out/provider-components.yaml \
        -a examples/addons.yaml
   ```

   To choose a specific Minikube driver, please use the `--bootstrap-flags vm-driver=xxx` command line parameter. For example to use the kvm2 driver with clusterctl you would add `--bootstrap-flags vm-driver=kvm2`, for linux, if you haven't installed any driver, you can add `--bootstrap-flags vm-driver=none`.

   - If you are using Kind:

   ```bash
   clusterctl create cluster \
           --bootstrap-type kind --bootstrap-cluster-cleanup=false \
           -c examples/_out/cluster.yaml \
           -m examples/_out/machines.yaml \
           -p examples/_out/provider-components.yaml \
           -a examples/addons.yaml
   # Alternatively
   make create-cluster
   ```

   - If you are using an existing Kubernetes cluster:

   ```bash
   clusterctl create cluster \
           --bootstrap-cluster-kubeconfig ~/.kube/config \
           -c examples/_out/cluster.yaml \
           -m examples/_out/machines.yaml \
           -p examples/_out/provider-components.yaml \
           -a examples/addons.yaml
   ```

### Interacting with your cluster

Once you have created a cluster, you can interact with the cluster and machine resources using kubectl:

```bash
kubectl --kubeconfig=kubeconfig get clusters
kubectl --kubeconfig=kubeconfig get machines
kubectl --kubeconfig=kubeconfig get machines -o yaml
```

### Cluster Deletion

Use following command to delete a cluster and all resources it created.
```bash
clusterctl delete cluster --cluster <cluster-name> --bootstrap-type kind --kubeconfig kubeconfig --provider-components examples/_out/provider-components.yaml
```

Or you can manually delete all resources that were created as part of
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

3. (optional) Delete the load balancer in your OpenStack cloud if you created them.

4. Delete the kubeconfig file that were created for your cluster.

   ```bash
   rm kubeconfig
   ```

5. Delete the ssh keypair that were created for your cluster machine.

   ```bash
   rm -rf $HOME/.ssh/openstack_tmp*
   ```

### Troubleshooting

Please refer to [Trouble shooting documentation](docs/trouble_shooting.md) for further info.

## Contributing

Please refer to the [Contribution Guide](CONTRIBUTING.md) and [Development Guide](docs/development.md) for this project.
