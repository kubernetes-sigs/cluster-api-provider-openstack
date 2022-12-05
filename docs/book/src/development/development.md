<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Development Guide](#development-guide)
  - [Using your own capi-openstack controller image for testing cluster creation or deletion](#using-your-own-capi-openstack-controller-image-for-testing-cluster-creation-or-deletion)
    - [Building and upload your own capi-openstack controller image](#building-and-upload-your-own-capi-openstack-controller-image)
    - [Using your own capi-openstack controller image](#using-your-own-capi-openstack-controller-image)
  - [Developing with Tilt](#developing-with-tilt)
  - [Running E2E tests locally](#running-e2e-tests-locally)
    - [Support for clouds using SSL](#support-for-clouds-using-ssl)
    - [Support for clouds with multiple external networks](#support-for-clouds-with-multiple-external-networks)
    - [OpenStack prerequisites](#openstack-prerequisites)
  - [Running E2E tests using rootless podman](#running-e2e-tests-using-rootless-podman)
    - [Host configuration](#host-configuration)
    - [Running podman system service to emulate docker daemon](#running-podman-system-service-to-emulate-docker-daemon)

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

We have support for using [Tilt](https://tilt.dev/) for rapid iterative development. Please visit the [Cluster API documentation on Tilt](https://cluster-api.sigs.k8s.io/developer/tilt.html) for information on how to set up your development environment. 

## Running E2E tests locally

You can run the E2E tests locally with:

```bash
make test-e2e OPENSTACK_CLOUD_YAML_FILE=/path/to/clouds.yaml OPENSTACK_CLOUD=mycloud
```

where `mycloud` is an entry in `clouds.yaml`.

The E2E tests:
* Build a CAPO image from the local working directory
* Create a kind cluster locally
* Deploy downloaded CAPI, and locally-build CAPO to kind
* Create an e2e namespace per-test on the kind cluster
* Deploy cluster templates to the test namespace
* Create test clusters on the target OpenStack

### Support for clouds using SSL

If your cloud requires a cacert you must also pass this to make via `OPENSTACK_CLOUD_CACERT_B64`, i.e.:

```bash
make test-e2e OPENSTACK_CLOUD_YAML_FILE=/path/to/clouds.yaml OPENSTACK_CLOUD=my_cloud \
              OPENSTACK_CLOUD_CACERT_B64=$(base64 -w0 /path/to/mycloud-ca.crt)
```

CAPO deployed in the local kind cluster will automatically pick up a `cacert` defined in your `clouds.yaml` so you will see servers created in OpenStack without specifying `OPENSTACK_CLOUD_CACERT_B64`. However, the cacert won't be deployed to those servers, so kubelet will fail to start.

### Support for clouds with multiple external networks

If your cloud contains only a single external network CAPO will automatically select that network for use by a deployed cluster. However, if there are multiple external networks CAPO will log an error and fail to create any machines. In this case you must pass the id of an external network to use explicitly with `OPENSTACK_EXTERNAL_NETWORK_ID`, i.e.:

```bash
make test-e2e OPENSTACK_CLOUD_YAML_FILE=/path/to/clouds.yaml OPENSTACK_CLOUD=my_cloud \
              OPENSTACK_EXTERNAL_NETWORK_ID=27635f93-583d-454e-9c6d-3d305e7f8a22
```

`OPENSTACK_EXTERNAL_NETWORK_ID` must be specified as a uuid. Specifying by name is not supported.

You can list available external networks with:

```bash
$ openstack network list --external
+--------------------------------------+----------+--------------------------------------+
| ID                                   | Name     | Subnets                              |
+--------------------------------------+----------+--------------------------------------+
| 27635f93-583d-454e-9c6d-3d305e7f8a22 | external | be64cd07-f8b7-4705-8446-26b19eab3914 |
| cf2e83dc-545d-490f-9f9c-4e90927546f2 | hostonly | ec95befe-72f4-4af6-a263-2aec081f47d3 |
+--------------------------------------+----------+--------------------------------------+
```

### OpenStack prerequisites

The file [`test/e2e/data/e2e_conf.yaml`](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/main/test/e2e/data/e2e_conf.yaml) and the test templates under [`test/e2e/data/infrastructure-openstack`](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/tree/main/test/e2e/data/infrastructure-openstack) reference several OpenStack resources which must exist before running the test:

* System requirements
  * Multiple nodes
  * `controller`: 16 CPUs / 64 GB RAM
  * `worker`: 8 CPUs / 32 GB RAM
* Availability zones (for multi-AZ tests)
  * `testaz1`: used by all test cases
  * `testaz2`: used by multi-az test case
* Services (Additional services to be enabled)
  * Octavia
  * Network trunking (neutron-trunk)
  * see [Configration](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/main/docs/book/src/development/ci.md#configuration) for more details.
* Glance images
  * `cirros-0.6.1-x86_64-disk`
    * Download from https://docs.openstack.org/image-guide/obtain-images.html
  * `ubuntu-2004-kube-v1.23.10`
    * Download from https://storage.googleapis.com/artifacts.k8s-staging-capi-openstack.appspot.com/test/ubuntu/2022-12-05/ubuntu-2004-kube-v1.23.10.qcow2
    * Or generate using the `images/capi` directory from https://github.com/kubernetes-sigs/image-builder
      * Boot volume size must be less than 15GB
* Flavors
  * `m1.medium`: used by control plane
  * `m1.small`: used by workers
  * `m1.tiny`: used by bastion
* clouds.yaml
  * `capo-e2e`: for general user authorization
  * `capo-e2e-admin`: for administrator user authorization
  * i.e.:
    ``` yaml
    clouds:
      capo-e2e:
        auth:
          auth_url: http://Node-Address/identity
          project_name: demo
          project_domain_name: Default
          user_domain_name: Default
          username: demo
          password: secret
        region_name: RegionOne

      capo-e2e-admin:
        auth:
          auth_url: http://Node-Address/identity
          project_name: demo
          project_domain_name: Default
          user_domain_name: Default
          username: admin
          password: secret
        region_name: RegionOne
    ```

You can also use [Hacking CI scripts](https://cluster-api-openstack.sigs.k8s.io/development/ci.html#devstack) to automatically create OpenStack environment.


## Running E2E tests using rootless podman

You can use unprivileged podman to:
* Build the CAPO image
* Deploy the kind cluster

To do this you need to configure the host appropriately and pass `PODMAN=1` to make, i.e.:

```bash
make test-e2e OPENSTACK_CLOUD_YAML_FILE=/path/to/clouds.yaml OPENSTACK_CLOUD=my_cloud \
              PODMAN=1
```

### Host configuration

Firstly, you must be using kernel >=5.11. If you are using Fedora, this means Fedora >= 34.

You must configure systemd and iptables as described in https://kind.sigs.k8s.io/docs/user/rootless/. There is no need to configure cgroups v2 on Fedora, as it uses this by default.

You must install the `podman-docker` package to emulate the docker cli tool. However, this is not sufficient on its own as described below.

### Running podman system service to emulate docker daemon

While kind itself supports podman, the cluster-api test framework does not. This framework is used by the CAPO tests to push test images into the kind cluster. Unfortunately the cluster-api test framework explicitly connects to a running docker daemon, so cli emulation is not sufficient for compatibility. This issue is tracked in https://github.com/kubernetes-sigs/cluster-api/issues/5146, and the following workaround can be ignored when this is resolved.

podman includes a 'system service' which emulates docker. For the tests to work, this service must be running and listening on a unix socket at `/var/run/docker.sock`. You can achieve this with:

```bash
$ podman system service -t 0 &
$ sudo rm /var/run/docker.sock
$ sudo ln -s /run/user/$(id -u)/podman/podman.sock /var/run/docker.sock
```
