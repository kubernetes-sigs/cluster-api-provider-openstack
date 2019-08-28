<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Openstack Example Files](#openstack-example-files)
  - [Contents](#contents)
  - [Prerequisites](#prerequisites)
  - [Generation](#generation)
  - [Manual Modification](#manual-modification)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Openstack Samples

## Prerequisites

1. Install `yq` (see [here](https://github.com/mikefarah/yq)).
1. Install `envsubst` (see [here](http://linuxcommandlibrary.com/man/envsubst.html)).
2. Install `kustomize` v3.1.0+ (see [here](https://github.com/kubernetes-sigs/kustomize/releases)

## Generation
For convenience, a generation script which populates templates based on openstack cloud provider
configuration is provided.

1. Run the generation script.
```
cd examples
export CLUSTER_NAME=<cluster-name>
./generate.sh [options] <path/to/clouds.yaml> <openstack cloud> [output folder]
```

   `<clouds.yaml>` is a YAML configuration file for Openstack, refer to [clouds.yaml](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/master/pkg/cloud/openstack/clients/clouds.yaml), and [OpenStack client configuration files](https://docs.openstack.org/python-openstackclient/latest/configuration/index.html#configuration-files).

   `<openstack cloud>` is the cloud you are going to use, e.g. multiple cloud might be defined in `clouds.yaml`
   and this will be cloud on which the new workload Kubernetes cluster will be created.
   For example, assume you have 2 clouds defined below as `clouds.yaml` and specify `openstack1` will use all definition in it.

   ```
   clouds:
     openstack1:
       auth:
         auth_url: http://192.168.122.10:35357/
       region_name: RegionOne
     ds-admin:
       auth:
         auth_url: http://192.168.122.10:35357/
       region_name: RegionOne
   ```
   In case your OpenStack cluster endpoint is using SSL and the cert is signed by an unknown CA, a specific cacert can be provided via cacert.

   `[output folder]` is where to put generated yaml files, by default it's `out`.

## Manual Modification

You **will need** to make changes to the generated files to create a working cluster.
You can find some guidance on what needs to be edited, and how to create some of the
required OpenStack resources in the [Configuration documentation](../docs/config.md).

Note that to set the desired security groups the UUIDs must be used.
Using security groups names is not supported.

#### Quick notes on clouds.yaml

We no longer support generating clouds.yaml. You should be able to get a valid clouds.yaml from your openstack cluster. However, make sure that the following fields are included, and correct.

- `username`
- `user_domain_name`
- `project_id`
- `region_name`
- `auth_url`
- `password`

#### Notes on ssh keys for debug purpose.

When running `generate.sh` the first time, a new ssh keypair is generated and stored as `$HOME/.ssh/openstack_tmp` and `$HOME/.ssh/openstack_tmp.pub`. The key is for debug purpose only now. You have to create the ssh key manually in OpenStack before creating the cluster, e.g.:

```
openstack keypair create --public-key ~/.ssh/openstack_tmp.pub cluster-api-provider-openstack
```
