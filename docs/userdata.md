# Passing in user-data

`user-data` is a way to ship config files and scripts into OpenStack VMs. Usually VMs come up and call `cloud-init` to get `user-data` from the OpenStack Metadata Service and execute it.

For further documentations have a look at:

* [OpenStack Metadata Service](https://docs.openstack.org/nova/latest/user/metadata-service.html)
* [cloud-init](https://cloudinit.readthedocs.io/en/latest/)

## Creating a machine

We support some distributions out of the box. So there is no need to pass in custom `user-data`.

Supported Distributions:

* Ubuntu (16.04 and 18.04) (`distributionType: ubuntu`)
* CentOS 7.3 (`distributionType: centos`)

### Example with the builtin `user-data`

An example machine specification looks like this:

```yaml
apiVersion: cluster.k8s.io/v1alpha1
kind: Machine
metadata:
  labels:
    set: node
  name: openstack-node-abcde
spec:
  providerSpec:
    value:
      apiVersion: openstackproviderconfig/v1alpha1
      availabilityZone: ix1
      cloudName: openstack
      flavor: m1.small
      image: CentOS 7 - latest
      distributionType: centos
      keyName: cluster-api-provider-openstack
      kind: OpenstackProviderSpec
      networks:
      - uuid: 864dc69d-3aef-473e-a500-18abc5b9b76f
      securityGroups:
      - default
      - kube-group
      sshUserName: centos
  versions:
    kubelet: 1.12.5
```

### Example with custom `user-data`

```yaml
apiVersion: cluster.k8s.io/v1alpha1
kind: Machine
metadata:
  labels:
    set: node
  name: openstack-node-wq8cx
spec:
  providerSpec:
    value:
      apiVersion: openstackproviderconfig/v1alpha1
      availabilityZone: ix1
      cloudName: openstack
      flavor: m1.small
      image: CentOS 7 - cg
      keyName: cluster-api-provider-openstack
      kind: OpenstackProviderSpec
      networks:
      - uuid: 964dc69d-3aef-473e-a500-18abc5b9b76f
      securityGroups:
      - default
      - kube-group
      sshUserName: centos
      userDataSecret:
        name: worker-user-data
        namespace: openstack-provider-system
  versions:
    kubelet: 1.12.5
```

So, what you need to do is passing in a `userDataSecret`:

```yaml
      userDataSecret:
        name: worker-user-data
        namespace: openstack-provider-system
```

This secret in fact is a Go Template. See [`machineScript.go`](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/master/pkg/cloud/openstack/machine/machineScript.go) for parameters you could use. Because this is a usual OpenStack Metadata, you can pass in whatever `cloud-init` supports. It could be a script, `cloud-config`, even multipart.