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
2. Install `kustomize` v3.1.0+ (see [here](https://github.com/kubernetes-sigs/kustomize/releases)).

## Generation

For convenience, a script is provided which generates example YAMLs. The generated YAMLs then have to be customized manually.

1. Run the generation script.

```
cd examples
export CLUSTER_NAME=<cluster-name>
./generate.sh [options] <path/to/clouds.yaml> <openstack-cloud> [output folder]
```

   `<clouds.yaml>` is a YAML configuration file for Openstack, for more details refer to [OpenStack client configuration files](https://docs.openstack.org/python-openstackclient/latest/configuration/index.html#configuration-files).

   `<openstack-cloud>` is the cloud you are going to use, e.g. multiple clouds might be defined in `clouds.yaml`.
   This will be the cloud on which the new workload Kubernetes cluster will be created. For example, assume you have 
   multiple clouds defined in the `clouds.yaml` like shown below. You have to decide if you want to create your cluster 
   in `openstack1` or `ds-admin`.

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
   In case your OpenStack cluster endpoint is using SSL and the cert is signed by an unknown CA, a specific CA certificate
   can be provided via the cacert field.

   `[output folder]` is where the YAML files will be stored, by default it's `_out`.

## Manual Modification

You **will need** to make changes to the generated files to create a working cluster.
You can find some guidance on what needs to be edited, and how to create some of the
required OpenStack resources in the [Configuration documentation](../docs/config.md).


#### Quick notes on clouds.yaml

We no longer support generating clouds.yaml. You should be able to get a valid clouds.yaml from your openstack cluster. 
However, make sure that the following fields are included, and correct.

- `username`
- `user_domain_name`
- `project_id`
- `region_name`
- `auth_url`
- `password`

#### Notes on ssh keys for debug purpose.

When running `generate.sh` the first time, a new ssh keypair is generated and stored in `$HOME/.ssh/openstack_tmp` and 
`$HOME/.ssh/openstack_tmp.pub`. The key is for debugging purposes only. You have to create the ssh key manually in OpenStack 
before creating the cluster, e.g.:

```bash
openstack keypair create --public-key ~/.ssh/openstack_tmp.pub cluster-api-provider-openstack
```
