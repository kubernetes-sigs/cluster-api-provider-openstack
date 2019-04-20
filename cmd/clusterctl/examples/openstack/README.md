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

## Generation
For convenience, a generation script which populates templates based on openstack cloud provider
configuration is provided.

1. Run the generation script.
```
./generate-yaml.sh --provider-os [os name]
```

   [os name] is the operating system of your provider environment. 

   Supported Operating Systems: 
   - `ubuntu` 
   - `centos`

If yaml file already exists, you will see an error like the one below:

```
$ ./generate-yaml.sh --provider-os [os name]
File provider-components.yaml already exists. Delete it manually before running this script.
```

## Manual Modification
You may always manually curate files based on the examples provided.

Note that to set the desired security groups the UUIDs must be used.
Using security groups names is not supported.