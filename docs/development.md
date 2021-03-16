<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Development Guide](#development-guide)
  - [Using your own capi-openstack controller image for testing cluster creation or deletion](#using-your-own-capi-openstack-controller-image-for-testing-cluster-creation-or-deletion)
    - [Building and upload your own capi-openstack controller image](#building-and-upload-your-own-capi-openstack-controller-image)
    - [Using your own capi-openstack controller image](#using-your-own-capi-openstack-controller-image)
  - [Developing with Tilt](#developing-with-tilt)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Development Guide

This document explains how to develop Cluster API Provider OpenStack.

## Using your own capi-openstack controller image for testing cluster creation or deletion

You need to create your own openstack-capi controller image for testing cluster creation or deletion by your code.
The image is stored in the docker registry. You need to create an account of Docker registry in advance.

### Building and upload your own capi-openstack controller image

Log in to your registry account. Export the following environment variables which will be used by the Makefile.

Variable | Meaning | Mandatory | Example
------------ | ------------- | ------------- | -------------
REGISTRY | The registry name | Yes | docker.io/\<username\>
IMAGE_NAME | The image name (default: capi-openstack-controller) | No | capi-openstack-controller
TAG | The image version (default: dev) | No | latest

Execute the command to build and upload the image to the Docker registry.

```bash
make docker-build docker-push
```

### Using your own capi-openstack controller image

After generating `infrastructure-components.yaml`, replace the `us.gcr.io/k8s-artifacts-prod/capi-openstack/capi-openstack-controller:v0.3.4` with your image.

## Developing with Tilt

We have support for using [Tilt](https://tilt.dev/) for rapid iterative development. Please visit the [Cluster API documentation on Tilt](https://master.cluster-api.sigs.k8s.io/developer/tilt.html) for information on how to set up your development environment. 
