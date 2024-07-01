# Problem statement

We currently have 2 callers of compute.CreateInstance: the machine controller for OpenStackMachine, and the cluster controller for the bastion[^mapo].

[^mapo]: Incidentally, although it is not a primary concern of this project there is also a third consumer in OpenShift's [Machine API Provider OpenStack](https://github.com/openshift/machine-api-provider-openstack).

Of these, the bastion has been most problematic. The principal current problem with the bastion is that, unlike OpenStackMachine, it is mutable because the bastion is not required to have the same lifecycle as the cluster which contains it. The current API allows the user to modify the bastion, and the controller is expected to delete the current one and create a new one. This is a problem in edge cases where the spec is required to determine if resources were previously created and need to be cleaned up, or not. This is a fundamental design problem, but another less fundamental but still very real problem is that trying to integrate the bastion into both the cluster and (effectively) machine creation and deletion flows creates fragile code and has resulted in numerous bugs.

# Potential solutions

## Do nothing

The current solution has sufficed in practice for quite a long time. Additionally, we now have much better test coverage of bastion operations than we have had in the past, so we should hopefully catch issues sooner.

The primary issue with this is the drag it places on bugfixes and new development, because the code flow of bastion deletion in particular can be hard to reason about.

## Create an OpenStackMachine for the bastion

This is an attractive solution because it better models the independent lifecycle of the bastion. When a new bastion is created we create a new OpenStackMachine. If the bastion spec is updated we delete the old one and create a new one, with the old and new objects having their own, independent state.

The problem with this approach is that OpenStackMachine has other contracts. It cannot be created independently, but requires a CAPI Machine object. Other controllers have expectations of Machine objects, for example that they will eventually have a NodeRef.

Creating a Machine/OpenStackMachine object for the bastion could negatively impact other CAPI users.

## Create a new controller for the bastion

This is essentially the same as using OpenStackMachine, except using a new CRD which can be independent of other CAPI objects.

The obvious disadvantage is that it requires a new CRD and controller.

# Proposal: An OpenStackServer CRD and controller

## Philosophy

We will create a new controller which is capable, in principal, of doing all of the OpenStack resource creation tasks common to bastion and OpenStackMachine creation. That is:

* Resolve OpenStack resource parameters to specific resources (e.g. image, networks, subnets, security groups, server groups)
* Create and delete ports
* Create and delete volumes
* Create and delete the server

It will have no dependency on Machine or OpenStackCluster. So things it will explicitly not do include:

* Attaching the Bastion floating IP
* Creating a loadbalancer member
* Adding a port for the default cluster network if no ports are provided
* Referencing an AvailabilityZone from the Machine object
* Adding default tags from the OpenStackCluster object

## API

Based on OpenStackMachineSpec, with modifications to accomodate the above.


```go
type OpenStackServerSpec struct {
	AvailabilityZone       string
	Flavor                 string
	Image                  ImageParam
	SSHKeyName             string
	Ports                  []PortOpts
	SecurityGroups         []SecurityGroupParam
	Trunk                  bool
	Tags                   []string
	ServerMetadata         []ServerMetadata
	ConfigDrive            *bool
	RootVolume             *RootVolume
	AdditionalBlockDevices []AdditionalBlockDevice
	ServerGroup            *ServerGroupParam
	IdentityRef            *OpenStackIdentityReference
	FloatingIPPoolRef      *corev1.TypedLocalObjectReference
	UserDataRef            *corev1.TypedLocalObjectReference
}

type OpenStackServerStatus struct {
	InstanceID    optional.String
	InstanceState *InstanceState
	Addresses     []corev1.NodeAddress
	Resolved      *ResolvedMachineSpec
	Resources     *MachineResources
	Conditions    clusterv1.Conditions
}
```

As the new API is non-trivial we should initially create it in v1alpha1.

### Upgrading an existing deployment

OpenStackServer would be a new API and does not affect any existing API.

In the same way that OpenStackMachine currently has an 'adoption' phase where it will adopt existing resources, OpenStackServer should adopt matching resources which it would have created. On upgrade to a new version of CAPO which manages the bastion with an OpenStackServer object, I would expect the flow to be:

* Cluster controller creates OpenStackServer and waits for it to report Ready
* OpenStackServer controller looks for existing resources matching its given spec
* OpenStackServer adopts existing resources
* OpenStackServer reports Ready

The end-user should not be required to take any action, and the upgrade process should not involve the deletion and recreation of existing OpenStack resources.

### Notes

* ProviderID is not present: this is a Machine property
* UserDataRef is added. If present it must exist.
* AvailabilityZone is added, and refers explicitly
* It is an error for Ports to be empty. Defaulting the cluster network must be done by the controller creating the object.
* No managed security group will be added automatically. The managed security group must be added explicitly by the controller creating the object.
* A Volume specified with `from: Machine` will use the value of `AvailabilityZone` instead of `Machine.FailureDomain`.

It will set the following `Conditions`:
* `Ready`: `True` when the server is fully provisioned. Once set to `True` will never be subsequently set to `False`.
* `Failed`: `True` means the OpenStackServer controller will not continue to reconcile this object. If set on a server which is not `Ready`, the server will never become `Ready`. If set on a server which is `Ready`, the server has suffered some terminal condition which cannot be resolved automatically. For example it may be in the `ERROR` state, or it may have been deleted.

## Changes to the cluster controller

Firstly the OpenStackCluster will report `Ready` as soon as the cluster infrastructure is created, before creating the bastion. This is not strictly necessary for this change, but it simplifies the bastion creation logic and it makes sense anyway.

The pseudocode of the cluster controller logic for the bastion becomes:
```
if spec.Bastion == nil {
	if OpenStackServer for bastion exists {
		delete OpenStackServer object
		return
	}
} else {
	serverObj := fetch OpenStackServer object
	if serverObj does not exist {
		substitute tags and default network into bastion spec if required
		create OpenStackServer object
		return
	}
	if server spec does not match bastion.Spec {
		deleteOpenStackServer(bastion)
		return
	}
	if server is Ready {
		attach bastion floating IP to server
		populate bastion status in OpenStackCluster
	}
}
```

The OpenStackServer for the bastion is owned by the OpenStackCluster, and is deleted automatically along with it.

The `Enabled` flag for the bastion in OpenStackClusterSpec is deprecated but not removed. Setting `Enabled` explicitly to false will continue to not create a bastion, although there is no longer any need to do this. Validation will be updated to permit changing the bastion spec without requiring the bastion to be deleted first.

`Resolved` and `Resources` from `BastionStatus` would not be removed, but would no longer be populated. This state is now stored in the `OpenStackServer`.

## Optional: Changes to the machine controller

The primary goal of this change is to fix the bastion, so it is not necessary to change the machine controller also. However, once we have implemented the change to the bastion it may make sense to update the machine controller to use the same controller. i.e. The machine controller would:

* Create an OpenStackServer object corresponding to the machine spec plus resolved defaults from the OpenStackCluster.
* Wait for it to be created
* Do any API loadbalancer tasks required
* Copy required CAPI fields from OpenStackServer to OpenStackMachine
* Set ProviderID

## Note (but out of scope of this document)

OpenShift's [Machine API Provider OpenStack](https://github.com/openshift/machine-api-provider-openstack) could create an `OpenStackServer` object instead of calling directly into `compute.CreateInstance`.