# Development Guide

This document explains how to develop cluster-api-provider-openstack.

## Using your own openstack-cluster-api-controller image for testing cluster creation or deletion

You need to create your own openstack-cluster-api-controller image for testing cluster creation or deletion by your code.
The image is stored in the docker registry. You need to create an account of Docker registry in advance.

1. Building your own openstack-cluster-api-controller image
1. Using your own openstack-cluster-api-controller image

### Building and upload your own openstack-cluster-api-controller image

Set environment variables which is used in Makefile.

Variable | Meaning | Mandatory | Example
------------ | ------------- | ------------- | -------------
REGISTRY | The registy name | Yes | alice
VERSION | The image version | No | 3
DOCKER_USERNAME | The username for logging in to the Docker registry | Yes | alice
DOCKER_PASSWORD | The password for logging in to the Docker registry | Yes | Passw0rd

Execute the command to build and upload the image to the Docker registry.

```bash
$ make upload-images

```

### Using your own openstack-cluster-api-controller image

After generating `provider-components.yaml`, update `spec.template.spec.containers[].image` in the file.
Replace `k8scloudprovider` with REGISTRY and `latest` with VERSION respectively.
