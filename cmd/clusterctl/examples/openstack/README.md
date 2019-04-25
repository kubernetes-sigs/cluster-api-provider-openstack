<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Openstack Example Files](#openstack-example-files)
  - [Contents](#contents)
  - [Prerequisites](#prerequisites)
  - [Generation](#generation)
  - [Manual Modification](#manual-modification)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Openstack Example Files
## Contents
- `*.yaml` - concrete example files that can be used as is.
- `*.yaml.template` - template example files that need values filled in before use.

## Prerequisites

1. Install `yq` (see [here](https://github.com/mikefarah/yq)).
2. Install `kustomize` v1.0.11 which can be found [here](https://github.com/kubernetes-sigs/kustomize/releases/tag/v1.0.11).

## Generation
For convenience, a generation script which populates templates based on openstack cloud provider
configuration is provided.

1. Run the generation script.
```
./generate-yaml.sh [options] <path/to/clouds.yaml> <openstack cloud> <provider os: [centos,ubuntu,coreos]> [output folder]
```

   `<clouds.yaml>` is a yaml file to record how to interact with Openstack Cloud, refer [clouds.yaml](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/master/pkg/cloud/openstack/clients/clouds.yaml), and [openclient configuration files](https://docs.openstack.org/python-openstackclient/latest/configuration/index.html#configuration-files) has additional information.

   `<openstack cloud>` is the cloud you are going to use, e.g. multiple cloud might be defined in `clouds.yaml`
   and this will be cloud to be used for the new kubernetes to interact with.
   for example, assume you have 2 clouds defined below as `clouds.yaml` and specify `openstack1` will use all definition in it.

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

   `<provider os>` is the operating system of your provider environment.

   Supported Operating Systems:
   - `centos` 
   - `ubuntu`
   - `coreos`

   `[output folder]` is where to put generated yaml files, by default it's `out`.

If yaml file already exists, you will see an error like the one below:

```
$ ./generate-yaml.sh [options] <path/to/clouds.yaml> openstack <provider os: [centos,ubuntu,coreos]> [output folder]

File provider-components.yaml already exists. Delete it manually before running this script.
```

## Manual Modification
You may always manually curate files based on the examples provided.

Note that to set the desired security groups the UUIDs must be used.
Using security groups names is not supported.
