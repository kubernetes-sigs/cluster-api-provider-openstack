# Openstack Example Files
## Contents
*.yaml files - concrete example files that can be used as is.
*.yaml.template files - template example files that need values filled in before use.

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

