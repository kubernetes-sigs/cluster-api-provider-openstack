<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Required Configuration](#required-configuration)
  - [Cluster and machines YAML files](#cluster-and-machines-yaml-files)
  - [Private Network](#private-network)
  - [Public Network](#public-network)
  - [Floating IPs](#floating-ips)
  - [Security Group Rules](#security-group-rules)
  - [Security Groups](#security-groups)
  - [Operating System Images](#operating-system-images)
  - [Network Filters](#network-filters)
  - [Multiple Networks](#multiple-networks)
  - [Subnet Filters](#subnet-filters)
  - [Tagging](#tagging)
  - [Metadata](#metadata)
- [Optional Configuration](#optional-configuration)
  - [Boot From Volume](#boot-from-volume)
  - [Timeout settings](#timeout-settings)
  - [Custom pod network CIDR](#custom-pod-network-cidr)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Required Configuration

To successfully run a Kubernetes cluster in OpenStack, you will need to configure a few essential properties. The following configurations are necessary:
  - private network
  - public network
  - floating ip address
  - router connecting private network to public network
  - security group rules
  - at least one of the supported operating system images

## Cluster and machines YAML files

After running `examples/generate.sh` the YAML files will be created in your custom output folder. This files contain configuration on what OpenStack elements to use to create the cluster on, and which cluster components to create. However the template is incomplete and needs to be filled in. The following sections explain some more details about what can be configured.

## Private Network 

Most openstack clusters come with a private network already, but if you would like to create a private network just for Kubernetes, then the following openstack commands will create it, and a subnet for the nodes:

```bash
openstack network create <name of network>
openstack subnet create <name of subnet> --network <name of network> --subnet-range <CIDR ip range>
```

Once you have a network that you want to host the cluster on, you have to configure it as `externalNetworkId` in `cluster.yaml`.

## Public Network

If your openstack cluster does not already have a public network, you should contact your cloud service provider. We will not review how to troubleshoot this here.

## Floating IPs

You have to be able to at least assign floating IPs in your OpenStack. If you don't have rights 
to create floating IPs you have to make sure they already exist before creating the cluster.

There are different places where the floating IP has to be configured:
* single-node control plane:
  * Add a `.spec.floatingIP` field to the `<cluster-name>-controlplane` Machine in`controlplane.yaml`.
* multi-node control plane:
  * Set the floating IP in `.spec.apiServerLoadBalancerFloatingIP` in your `<cluster-name>` Cluster resource in `cluster.yaml`.
* both:
  * Configure floating IP in `.spec.clusterConfiguration.controlPlaneEndpoint` in your `<cluster-name>-controlplane` KubeadmConfig resource in `controlplane.yaml`.

## Security Group Rules

For the installer to work, a few security groups are required to be open. These may be different from the security groups needed to reach a cluster once its running. The following security group rules should be added to the security group of your choosing. For this example, we will suppose you created a security group named ``kubernetes`` that you will use for the cluster.

```bash
openstack security group rule create --ingress --protocol tcp --dst-port 22 kubernetes
openstack security group rule create --ingress --protocol tcp --dst-port 3000:32767 kubernetes
openstack security group rule create --ingress --protocol tcp --dst-port 443 kubernetes
openstack security group rule create --egress kubernetes
```

## Security Groups

In `OpenStackCluster` (`cluster.yaml`) there is a boolean option called `managedSecurityGroups` that, if set to `true`, will create a default set of security groups for the cluster. These are meant for a "standard" setup, and might not be suitable for every environment. Please review the rules below before you use them.

**NOTE**: For now, there is no way to automatically use these rules, which makes them a bit cumbersome to use.

The rules created are:

* A rule for the controlplane machine, that allows access from everywhere to port 22 and 443.
* A rule for all the machines, both the controlplane and the nodes that allow all traffic between members of this group.

In `controlplane.yaml` and `machinedeployment.yaml`, you can specify OpenStack security groups to be applied to each server in the `securityGroups` section of the YAML. You can specify the security group in 3 ways: by ID, by Name, or by filters. When you specify a security group by ID it will always return 1 security group or an error if it fails to find the security group specified. Please note that it is possible to add more than one security group to your machine when using Name or a Filter to specify it. More details about the filter can be found in [SecurityGroupParam](../api/v1alpha3/types.go).

## Operating System Images

We currently depend on an update version of cloud-init otherwise the operating system choice is yours. The kubeadm bootstrap provider we're using also depends on some pre-installed software like a controller-runtime, kubelet, kubeadm, etc.. . For an examples how to build such an image take a look at [image-builder](https://github.com/kubernetes-sigs/image-builder/tree/master/images/capi).

You can reference which operating system image you want to use in the `controlplane.yaml` and `machinedeployment.yaml` files by replacing the `<Image Name>`.

## Network Filters

If you have a complex query that you want to use to lookup a network, then you can do this by using a network filter. More details about the filter can be found in [NetworkParam](../api/v1alpha3/types.go)

By using filters to look up a network, please note that it is possible to get multiple networks as a result. This should not be a problem, however please test your filters with `openstack network list` to be certain that it returns the networks you want. Please refer to the following usage example:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha3
kind: OpenStackMachine
metadata:
  name: <cluster-name>-controlplane
  namespace: <cluster-name>
spec:
  networks:
  - filter:
      name: <network-name>
```

## Multiple Networks

You can specify multiple networks (or subnets) to connect your server to. To do this, simply add another entry in the networks array. The following example connects the server to 3 different networks:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha3
kind: OpenStackMachine
metadata:
  name: <cluster-name>-controlplane
  namespace: <cluster-name>
spec:
  networks:
  - filters:
      name: myNetwork
      tags: myTag
  - uuid: your_network_id
  - subnet_id: your_subnet_id
```

## Subnet Filters

Rather than just using a network, you have the option of specifying a specific subnet to connect your server to. The following is an example of how to specify a specific subnet of a network to use for your server.

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha3
kind: OpenStackMachine
metadata:
  name: <cluster-name>-controlplane
  namespace: <cluster-name>
spec:
  networks:
  - filter:
      name: <network-name>
    subnets:
    - filter:
       name: <subnet-name>
```

## Tagging

By default, all resources will be tagged with the values: `clusterName` and `cluster-api-provider-openstack`. The minimum microversion of the nova api that you need to support server tagging is 2.52. If your cluster does not support this, then disable tagging servers by setting `disableServerTags: true` in `cluster.yaml`. By default, this value is false. If your cluster supports tagging servers, you have the ability to tag all resources created by the cluster in the `cluster.yaml` file. Here is an example how to configure tagging:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha3
kind: OpenStackCluster
metadata:
  name: <cluster-name>
  namespace: <cluster-name>
spec:
  tags:
  - cluster-tag
```

To tag resources specific to a machine, add a value to the tags field in `controlplane.yaml` and `machinedeployment.yaml` like this:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha3
kind: OpenStackMachine
metadata:
  name: <cluster-name>-controlplane
  namespace: <cluster-name>
spec:
  tags:
  - machine-tag
```

## Metadata

Instead of tagging, you also have the option to add metadata to instances. This functionality should be more commonly available than tagging. Here is a usage example:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha3
kind: OpenStackMachine
metadata:
  name: <cluster-name>-controlplane
  namespace: <cluster-name>
spec:
  serverMetadata:
    name: bob
    nickname: bobbert
```

# Optional Configuration

## Boot From Volume

1. For example in `examples/_out/controlplane.yaml` set `spec.rootVolume.diskSize` to something greater than `0` means boot from volume.

   ```yaml
   apiVersion: infrastructure.cluster.x-k8s.io/v1alpha3
   kind: OpenStackMachine
   metadata:
     name: <cluster-name>-controlplane
     namespace: <cluster-name>
   spec:
     rootVolume:
       diskSize: 0
       sourceType: ""
       SourceUUID: ""
   ...
   ```

## Timeout settings

If creating servers in your OpenStack takes a long time, you can increase the timeout, by default it's 5 minutes. You can set it via the `CLUSTER_API_OPENSTACK_INSTANCE_CREATE_TIMEOUT` in your Cluster API Provider OpenStack controller deployment.

## Custom pod network CIDR

If `192.168.0.0/16` is already in use within your network, you must select a different pod network CIDR. You have to replace the CIDR `192.168.0.0/16` with your own in the generated example files: `addons.yaml` and `cluster.yaml`.

