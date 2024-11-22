<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Development Guide](#development-guide)
  - [Using your own capi-openstack controller image for testing cluster creation or deletion](#using-your-own-capi-openstack-controller-image-for-testing-cluster-creation-or-deletion)
    - [Building and upload your own capi-openstack controller image](#building-and-upload-your-own-capi-openstack-controller-image)
    - [Using your own capi-openstack controller image](#using-your-own-capi-openstack-controller-image)
  - [Testing Cluster Creation using the 'dev-test' ClusterClass with Tilt](#testing-cluster-creation-using-the-dev-test-clusterclass-with-tilt)
    - [Developing with Tilt](#developing-with-tilt)
    - [Apply ClusterClass and create Cluster](#apply-clusterclass-and-create-cluster)
    - [Automatically applying kustomizations with Tilt](#automatically-applying-kustomizations-with-tilt)
  - [Using the 'dev-test' ClusterClass without Tilt](#using-the-dev-test-clusterclass-without-tilt)
    - [Creating a Kind Cluster](#creating-a-kind-cluster)
    - [Secret Configuration](#secret-configuration)
    - [Apply the ClusterClass and create Clusters](#apply-the-clusterclass-and-create-clusters)
  - [Running E2E tests locally](#running-e2e-tests-locally)
    - [Support for clouds using SSL](#support-for-clouds-using-ssl)
    - [Support for clouds with multiple external networks](#support-for-clouds-with-multiple-external-networks)
    - [E2E test environment](#e2e-test-environment)
      - [Requirements](#requirements)
      - [Create E2E test environment](#create-e2e-test-environment)
        - [OpenStack](#openstack)
        - [DevStack](#devstack)
          - [Server side](#server-side)
          - [CAPO side](#capo-side)
  - [Running E2E tests using rootless podman](#running-e2e-tests-using-rootless-podman)
    - [Host configuration](#host-configuration)
    - [Running podman system service to emulate docker daemon](#running-podman-system-service-to-emulate-docker-daemon)
  - [API concepts](#api-concepts)
    - [`referencedResources`](#referencedresources)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Development Guide

This document explains how to develop Cluster API Provider OpenStack (CAPO).
Note that CAPO depends on ORC. No matter how you choose to work, you will need to deploy ORC in order to make CAPO functional:

```bash
kubectl apply -f https://github.com/k-orc/openstack-resource-controller/releases/download/v1.0.0/install.yaml
```

TL;DR: Here is a short version for how to develop with Tilt:

1. Create a devstack
2. Run tilt, configured through the Cluster API repository
3. Install ORC
4. Create a secret with the clouds.yaml to match the devstack
5. Add images to the devstack

```bash
# Create Devstack
export RESOURCE_TYPE=...
./hack/ci/create_devstack.sh
# Start tilt (separate terminal in cluster-api repo)
# NOTE: Configure tilt-settings.yaml first! See below.
make tilt-up
# Back in CAPO repo
# Install ORC
kubectl apply -f https://github.com/k-orc/openstack-resource-controller/releases/download/v1.0.0/install.yaml
# Create secret with clouds.yaml (the file is created by create_devstack.sh)
kubectl create secret generic dev-test-cloud-config --from-file=clouds.yaml
# Add images to use in the tests
clusterctl generate yaml --from templates/images-template.yaml | kubectl apply -f -
```

At this point, you should be able to apply the `dev-test` ClusterClass and start creating/deleting `development` clusters through the Tilt UI.

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

## Automatically Adding Images to OpenStack

Before you can create a Cluster, you will need a suitable image in OpenStack.
There is a convenient template available in `templates/images-template.yaml` for this purpose.

```bash
clusterctl generate yaml --from templates/images-template.yaml | kubectl apply -f -
```

## Testing Cluster Creation using the 'dev-test' ClusterClass with Tilt

This guide demonstrates how to create a Kubernetes cluster using a ClusterClass, specifically designed for a development environment. It includes configuring secrets, applying the ClusterClass, and creating a cluster with Tilt.

The `dev-test` ClusterClass is designed for development.
This means that it is using the latest (potentially unstable) API version.
The defaults are also aligned with the devstack setup (documented below) to make it as easy as possible to use in a development flow.
However, this also means that it may *not* be well suited for general usage.

### Developing with Tilt

We have support for using [Tilt](https://tilt.dev/) for rapid iterative development. Please visit the [Cluster API documentation on Tilt](https://cluster-api.sigs.k8s.io/developer/tilt.html) for information on how to set up your development environment.

The `Tiltfile` in the Cluster API repository can be used as is with CAPO, but we need to add some configuration.
For using Tilt with ClusterClass, update your `tilt-settings.yaml` file (located in the root of the CAPI repository) as described:

```yaml
template_dirs:
  openstack:
  # Make Tilt aware of the CAPO templates
  - ../cluster-api-provider-openstack/templates

kustomize_substitutions:
  CLUSTER_TOPOLOGY: "true"
  # [Optional] SSH Keypair Name for Instances in OpenStack (Default: "")
  OPENSTACK_SSH_KEY_NAME: "<openstack_keypair_name>"
  # [Optional] Control Plane Machine Flavor (Default: m1.medium)
  OPENSTACK_CONTROL_PLANE_MACHINE_FLAVOR: "<openstack_control_plane_machine_flavor>"
  # [Optional] Node Machine Flavor (Default: m1.small)
  OPENSTACK_NODE_MACHINE_FLAVOR: "<openstack_node_machine_flavor>"
  # [Optional] OpenStack Cloud Environment (Default: capo-e2e)
  OPENSTACK_CLOUD: "<openstack_cloud>"
  # See templates/cluster-template-development.yaml and
  # templates/clusterclass-dev-test.yaml for more variables.

# [Optional] Automatically apply a kustomization, e.g. for adding the clouds.yaml secret
additional_kustomizations:
  secret_kustomization: /path/to/kustomize/secret/configuration
```

### Apply ClusterClass and create Cluster

When you are happy with the configuration, start the environment as explained in the CAPI documentation.
Open the Tilt dashboard in your browser.
After a while, you should be able to find resources called `CAPO.clusterclasses` and `CAPO.templates`.
These shoud correspond to what exists in the `templates` folder and you should see widgets for applying and deleting them.

**Note:** When you apply a cluster template, there will be a `KUBERNETES_VERSION` variable.
Make sure that the image or sysext overlay you use exists for and works with that version!

**Note:** All clusters created from the dev-test ClusterClass will require a secret named `dev-test-cloud-config` with the `clouds.yaml` to be used by CAPO for interacting with OpenStack.
You can create it manually or see below how to make Tilt automate it.

### Automatically applying kustomizations with Tilt

This explains how to automatically create the secret containing `clouds.yaml`.
The same procedure can be used for any other things you want to create in the cluster.

Ensure the specified path (`/path/to/kustomize/secret/configuration`) contains both the `clouds.yaml` file and a `kustomization.yaml` file. The `kustomization.yaml` should define the necessary resources, such as a Kubernetes secret, using the `clouds.yaml` file.

For example, if you want to automatically create a secret named `dev-test-cloud-config` with the content of your `clouds.yaml` every time you do `tilt up`, you could do the following.

Create a folder to hold the kustomization.
We will use `/tmp/capo-dev` as example here.

Add the `clouds.yaml` file that you want to use to the folder.
It could look something like this:

```yaml
clouds:
  capo-e2e:
    auth:
      username: demo
      password: secretadmin
      # If using application credentials you would have something like this instead:
      # auth_type: v3applicationcredential
      # application_credential_id: abc123
      # application_credential_secret: 456def
      user_domain_id: default
      auth_url: https://example.com/identity
      domain_id: default
      project_name: demo
    verify: false
    region_name: RegionOne
```

Create a kustomization file named `kustomization.yaml` in the same folder:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
# Do not add random hash to the end of the secret name
generatorOptions:
  disableNameSuffixHash: true
secretGenerator:
- files:
  - clouds.yaml
  name: dev-test-cloud-config
  type: Opaque
```

If you now add `/tmp/capo-dev` to the `additional_kustomizations`, tilt will automatically apply
the secret.

To check that the kustomization produces the desired output, do `kustomize build /tmp/capo-dev`.

## Using the 'dev-test' ClusterClass without Tilt

If you want to use the ClusterClass without Tilt, you will need to follow these steps instead of the above.

### Creating a Kind Cluster

Create a Kind cluster and deploy CAPO.

**Note:** As the dev-test ClusterClass is made for development, it may be using a newer API version than what is in the latest release.
You *will need* to [use local artifacts](https://cluster-api.sigs.k8s.io/clusterctl/developers#use-local-artifacts) for this to work in most cases!

```bash
kind create cluster
export CLUSTER_TOPOLOGY=true
clusterctl init --infrastructure openstack
```

### Secret Configuration

CAPO needs a clouds.yaml file in order to manage the OpenStack resources needed for the Cluster. This should be supplied as a secret named `dev-test-cloud-config`. You can create this secret for example with:

```bash
kubectl create secret generic dev-test-cloud-config --from-file=clouds.yaml
```

### Apply the ClusterClass and create Clusters

You can use `clusterctl` to render the ClusterClass:

```bash
clusterctl generate yaml  --from templates/clusterclass-dev-test.yaml | kubectl apply -f -
```

Create a cluster using the development template, that makes use of the ClusterClass:

```bash
clusterctl generate cluster my-cluster --kubernetes-version=v1.29.0 --from templates/cluster-template-development.yaml > my-cluster.yaml
kubectl apply -f my-cluster.yaml
```

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

### E2E test environment

The test suite is executed in an existing OpenStack environment. You can create and manage this environment yourself or use the [hacking CI scripts][hacking-ci-scripts] to provision an environment with DevStack similar to the one used for continuous integration.

#### Requirements

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

#### Create E2E test environment

You can easily create a test environment similar to the one used during continuous integration on OpenStack, AWS or GCE with the [hacking CI scripts][hacking-ci-scripts].

The entry point for the creation of the DevStack environment is the [create_devstack.sh][hack-ci-create-devstack] script, which executes specific scripts to create infrastructure on different clouds:

- AWS: [aws-project.sh][hack-ci-aws-project]
- GCE: [gce-project.sh][hack-ci-gce-project]
- OpenStack: [openstack.sh][hack-ci-openstack]
- Libvirt: [libvirt.sh][hack-ci-libvirt]

You can switch between these cloud providers, by setting the `RESOURCE_TYPE` environment variable to `aws-project`, `gce-project`, `openstack` or `libvirt` respectively.

##### OpenStack

Configure the following environment variables for OpenStack:

```bash
export RESOURCE_TYPE="openstack"
export OS_CLOUD=<your cloud>
export OPENSTACK_FLAVOR_controller=<flavor with >= 16 cores, 64GB RAM and 50GB storage>
export OPENSTACK_FLAVOR_worker=<flavor with >= 8 cores, 32GB RAM and 50GB storage>
export OPENSTACK_PUBLIC_NETWORK=<name of the external network>
export OPENSTACK_SSH_KEY_NAME=<your ssh key-pair name>
export SSH_PUBLIC_KEY_FILE=/home/user/.ssh/id_ed25519.pub
export SSH_PRIVATE_KEY_FILE=/home/user/.ssh/id_ed25519
```

and create the environment by running:

```bash
./hack/ci/create_devstack.sh
```

##### DevStack

Here's a few notes to setup a DevStack environment and debug ressources (tested on `m3.small` from Equinix Metal: https://deploy.equinix.com/product/servers/m3-small/)

###### Server side

As a root user, install and configure DevStack:

```
# useradd -s /bin/bash -d /opt/stack -m stack
# chmod +x /opt/stack
# echo "stack ALL=(ALL) NOPASSWD: ALL" | tee /etc/sudoers.d/stack
# sudo -u stack -i
$ git clone https://opendev.org/openstack/devstack
$ cd devstack
$ cat > local.conf <<EOF
[[local|localrc]]
ADMIN_PASSWORD=!!! CHANGE ME !!!
DATABASE_PASSWORD=\$ADMIN_PASSWORD
RABBIT_PASSWORD=\$ADMIN_PASSWORD
SERVICE_PASSWORD=\$ADMIN_PASSWORD

GIT_BASE=https://opendev.org
# Enable Logging
LOGFILE=$DEST/logs/stack.sh.log
VERBOSE=True
LOG_COLOR=True
enable_service rabbit
enable_plugin neutron $GIT_BASE/openstack/neutron
# Octavia supports using QoS policies on the VIP port:
enable_service q-qos
enable_service placement-api placement-client
# Octavia services
enable_plugin octavia $GIT_BASE/openstack/octavia master
enable_plugin octavia-dashboard $GIT_BASE/openstack/octavia-dashboard
enable_plugin ovn-octavia-provider $GIT_BASE/openstack/ovn-octavia-provider
enable_plugin octavia-tempest-plugin $GIT_BASE/openstack/octavia-tempest-plugin
enable_service octavia o-api o-cw o-hm o-hk o-da
# Cinder
enable_service c-api c-vol c-sch
EOF
$ ./stack.sh
```

If you want to enable web-download (i.e import images from URL):
```
# /etc/glance/glance-api.conf
show_multiple_locations = True

# ./horizon/openstack_dashboard/defaults.py
IMAGE_ALLOW_LOCATIONS = True

# /etc/glance/glance-image-import.conf
[image_import_opts]
image_import_plugins = ['image_decompression']

$ sudo systemctl restart devstack@g-api.service apache2
```

With this dev setup, it might be useful to enable DHCP for the public subnet:
Admin > Network > Networks > `public` > Subnets > `public-subnet` > Edit Subnet > Subnet Details > :ballot_box_with_check: Enable DHCP + Add DNS

###### CAPO side

To work with this setup, it takes an update of the `test/e2e/data/e2e_conf.yaml` file. (NOTE: You can decide to update the m1.small flavor to avoid changing it)

```diff
diff --git a/test/e2e/data/e2e_conf.yaml b/test/e2e/data/e2e_conf.yaml
index 0d66e1f2..a3b2bd78 100644
--- a/test/e2e/data/e2e_conf.yaml
+++ b/test/e2e/data/e2e_conf.yaml
@@ -136,7 +136,7 @@ variables:
   CNI: "../../data/cni/calico.yaml"
   CCM: "../../data/ccm/cloud-controller-manager.yaml"
   EXP_CLUSTER_RESOURCE_SET: "true"
-  OPENSTACK_BASTION_IMAGE_NAME: "cirros-0.6.1-x86_64-disk"
+  OPENSTACK_BASTION_IMAGE_NAME: "cirros-0.5.2-x86_64-disk"
   OPENSTACK_BASTION_MACHINE_FLAVOR: "m1.tiny"
   OPENSTACK_CLOUD: "capo-e2e"
   OPENSTACK_CLOUD_ADMIN: "capo-e2e-admin"
@@ -144,10 +144,10 @@ variables:
   OPENSTACK_CLOUD_YAML_FILE: '../../../../clouds.yaml'
   OPENSTACK_CONTROL_PLANE_MACHINE_FLAVOR: "m1.medium"
   OPENSTACK_DNS_NAMESERVERS: "8.8.8.8"
-  OPENSTACK_FAILURE_DOMAIN: "testaz1"
-  OPENSTACK_FAILURE_DOMAIN_ALT: "testaz2"
+  OPENSTACK_FAILURE_DOMAIN: "nova"
+  OPENSTACK_FAILURE_DOMAIN_ALT: "nova"
   OPENSTACK_IMAGE_NAME: "focal-server-cloudimg-amd64"
-  OPENSTACK_NODE_MACHINE_FLAVOR: "m1.small"
+  OPENSTACK_NODE_MACHINE_FLAVOR: "m1.medium"
```

Before running a test:
* start `sshuttle` (https://github.com/sshuttle/sshuttle) to setup the network between the host and the devstack instance correctly.
```
sshuttle -r stack@<devstack-server-ip> 172.24.4.0/24 -l 0.0.0.0
```
* import the tested image in DevStack by matching the name defined in `e2e_conf.yaml` (`OPENSTACK_FLATCAR_IMAGE_NAME` or `OPENSTACK_IMAGE_NAME`)

To run a specific test, it's possible to fill this variable `E2E_GINKGO_FOCUS`, if you want to SSH into an instance to debug it, it's possible to proxy jump via the bastion and to use the SSH key generated by Nova, for example with Flatcar:
```
ssh -J cirros@172.24.4.229 -i ./_artifacts/ssh/cluster-api-provider-openstack-sigs-k8s-io core@10.6.0.145
```

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

<!-- References -->

[hacking-ci-scripts]: https://cluster-api-openstack.sigs.k8s.io/development/ci.html#devstack
[hack-ci-aws-project]: https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/main/hack/ci/aws-project.sh
[hack-ci-create-devstack]: https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/main/hack/ci/create_devstack.sh
[hack-ci-gce-project]: https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/main/hack/ci/gce-project.sh
[hack-ci-openstack]: https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/main/hack/ci/openstack.sh
[hack-ci-libvirt]: https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/main/hack/ci/libvirt.sh

## API concepts

This sections goal is to gather various insights into the API design that can serve as a reference to explain various choices made without need to analyze discussions in individual PRs.

### `referencedResources`

Starting from v1beta1 both `OpenStackMachineStatus` and `BastionsStatus` feature a field named `referencedResources` which aims to include fields that list individual IDs of the resources associated with the machine or bastion. These IDs are calculated on machine or bastion creation and are not intended to be changed during the object lifecycle.

Having all the IDs of related resources saved in the statuses allows CAPO to make easy decisions about deleting the related resources when deleting the VM corresponding to the machine or bastion.

### `OpenStackServer`

`OpenStackServer` is a new resource introduced as v1alpha1. It is a representation of a server in OpenStack. It is used to manage the lifecycle of the server and to store the server's status.
Both the bastion and the machine are represented by an `OpenStackServer` object.
Even if it's technically possible for an user to create an `OpenStackServer` object directly, it is not supported for now as there is no use case for it.

To get the list of `OpenStackServer` objects, the user can use the following command:

```shell
kubectl get openstackservers
```

This object is immutable and is created by the controller when a machine or a bastion is created. The `OpenStackServer` object is deleted when the machine or the bastion is deleted.
