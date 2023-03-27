<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Hacking CI for the E2E tests](#hacking-ci-for-the-e2e-tests)
  - [Prow](#prow)
  - [DevStack](#devstack)
    - [DevStack OS](#devstack-os)
    - [Configuration](#configuration)
    - [Build order](#build-order)
    - [Networking](#networking)
    - [Availability zones](#availability-zones)
  - [Connecting to DevStack](#connecting-to-devstack)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Hacking CI for the E2E tests

## Prow

CAPO tests are executed by Prow. They are defined in the [Kubernetes test-infra repository](https://github.com/kubernetes/test-infra/tree/master/config/jobs/kubernetes-sigs/cluster-api-provider-openstack). The E2E tests run as a presubmit. They run in a docker container in Prow infrastructure which contains a checkout of the CAPO tree under test. The entry point for tests is `scripts/ci-e2e.sh`, which is defined in the job in Prow.

## DevStack

The E2E tests require an OpenStack cloud to run against, which we provision during the test with DevStack. The project has access to capacity on GCP, so we provision DevStack on 2 GCP instances.

The entry point for the creation of the test DevStack is `hack/ci/create_devstack.sh`, which is executed by `scripts/ci-e2e.sh`. We create 2 instances: `controller` and `worker`. Each will provision itself via cloud-init using config defined in `hack/ci/cloud-init`.

### DevStack OS

In GCE, DevStack is installed on a community-maintained Ubuntu 20.04 LTS cloud image. The cloud-init config is also intended to work on CentOS 8, and this is known to work as of 2021-01-12. However, note that this is not regularly tested. See the comment in `hack/ci/gce-project.sh` for how to deploy on CentOS.

It is convenient to the project to have a viable second OS option as it gives us an option to work around issues which only affect one or the other. This is most likely when enabling new DevStack features, but may also include infrastructure issues. Consequently, when making changes to cloud-init, try not to use features specific to Ubuntu or CentOS. DevStack already supports both operating systems, so we just need to be careful in our peripheral configuration, for example by using cloud-init's `packages` module rather than manually invoking `apt-get` or `yum`. Fortunately package names tend to be consistent across the two distributions.

### Configuration

We configure a 2 node DevStack. `controller` is running:

* All control plane services
* Nova: all services, including compute
* Glance: all services
* Octavia: all services
* Neutron: all services with ML2/OVS, including L3 agent
* Cinder: all services, including volume with default LVM/iSCSI backend

`worker` is running:

* Nova: compute only
* Neutron: agent only (not L3 agent)
* Cinder: volume only with default LVM/iSCSI backend

`controller` is using the `n2-standard-16` machine type with 16 vCPUs and 64 GB RAM. `worker` is using the `n2-standard-8` machine type with 8 vCPUs and 32 GB RAM. Each job has a quota limit of 24 vCPUs.

### Build order

We build `controller` first, and then `worker`. We let `worker` build asynchronously because tests which don't require a second AZ can run without it while it builds. A systemd job defined in the cloud-init of `controller` polls for `worker` coming up and automatically configures it.

### Networking

Both instances share a common network which uses the CIDR defined in `PRIVATE_NETORK_CIDR` in `hack/ci/create_devstack.sh`. Each instance has a single IP on this network:

* `controller`: `10.0.3.15`
* `worker`: `10.0.3.16`

In addition, DevStack will create a floating IP network using CIDR defined in `FLOATING_RANGE` in `hack/ci/create_devstack.sh`. As the neutron L3 agent is only running on the controller, all of this traffic is handled on the controller, even if the source is an instance running on the worker. The controller creates `iptables` rules to NAT this traffic.

The effect of this is that instances created on either `controller` or `worker` can get a floating ip from the `public` network. Traffic using this floating IP will be routed via `controller` and externally via NAT.

### Availability zones

We are running `nova compute` and `cinder volume` on each of `controller` and `worker`. Each `nova compute` and `cinder volume` are configured to be in their own availability zone. The names of the availability zones are defined in `OPENSTACK_FAILURE_DOMAIN` and `OPENSTACK_FAILURE_DOMAIN_ALT` in `test/e2e/data/e2e_conf.yaml`, with the services running on `controller` being in `OPENSTACK_FAILURE_DOMAIN` and the services running on `worker` being in `OPENSTACK_FAILURE_DOMAIN_ALT`.

This configuration is intended only to allow the testing of functionality related to availability zones, and does not imply any robustness to failure.

Nova is configured (via `[DEFAULT]/default_schedule_zone`) to place all workloads on the controller unless they have an explicit availability zone. The intention is that `controller` should have the capacity to run all tests which are agnostic to availability zones. This means that the explicitly multi-az tests do not risk failure due to capacity issues.

However, this is not sufficient because by default [CAPI explicitly schedules the control plane across all discovered availability zones](https://github.com/kubernetes-sigs/cluster-api/blob/e7769d7a6b3a4eb32292938eed8c470b7018a8b3/controlplane/kubeadm/controllers/scale.go#L77-L82). Consequently we explicitly confine all clusters to `OPENSTACK_FAILURE_DOMAIN` (`controller`) in the test cluster definitions in `test/e2e/data/infrastructure-openstack`.

## Connecting to DevStack

The E2E tests running in Prow create a kind cluster. This also running in Prow using Docker in Docker. The E2E tests configure this cluster with clusterctl, which is where CAPO executes.

`create_devstack.sh` wrote a `clouds.yaml` to the working directory, which is passed to CAPO via the cluster definitions in `test/e2e/data/infrastructure-openstack`. This `clouds.yaml` references the public, routable IP of `controller`. However, DevStack created all the service endpoints using `controller`'s private IP, which is not publicly routable. In addition, the tests need to be able to SSH to the floating IP of the Bastion. This floating IP is also allocated from a range which is not publicly routable.

To allow this access we run `sshuttle` from `create_devstack.sh`. This creates an SSH tunnel and routes traffic for `PRIVATE_NETWORK_CIDR` and `FLOATING_RANGE` over it.

Note that the semantics of a `sshuttle` tunnel are problematic. While they happen to work currently for DinD, Podman runs the kind cluster in a separate network namespace. This means that kind running in podman cannot route over `sshuttle` running outside the kind cluster. This may also break in future versions of Docker.
