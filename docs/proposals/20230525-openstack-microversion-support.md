# Configurable/multi microversion support

OpenStack is diverse.
Multiple versions and configurations exists that can impact what microversions and features that are supported.
The user (of CAPO) can also to some extent choose what features to make use of.
The cross section of this is where CAPO is useful:

## Motivation

A CAPO cluster can be configured in many ways and depending on this configuration, certain features may or may not be needed.
This makes it necessary to know and detect if a feature is used that requires a certain microversion.

It is not enough to check if the user has asked for a specific configuration that would require a certain microversion.
The requirement can come from the OpenStack instance itself.
As an example, we have seen environments where all volumes are multi-attachable.
I.e. it is not possible to create a volume that is not multi-attachable, and thus multi-attach support is a must to be able to do anything in this environment, even if the user doesn't explicitly ask for it.

Currently CAPO has a fixed microversion set.
It is a take it or leave it situation.
Either this version is supported by the OpenStack environment or you cannot use CAPO.

This proposal is to make CAPO more flexible in what it supports.
For example, there could be other versions that doesn't really impact CAPO in any way or there could be cross sections between what the OpenStack environment and CAPO supports, that could be useful to the user if certain features are avoided.
Here is a simple diagram to try to summarize what this proposal aims to achieve:

```text
 ┌────────────────────────────────────┐
 │                                    │
 │                                    │
 │    Versions supported by           │
 │    OpenStack instance              │
 │                                    │
 │    ┌─────────────────────────┐     │
 │    │                         │     │
 │    │  Versions available to  │     │
 │    │  the user               │     │
 └────┼─────────────────────────┼─────┘
      │                         │
      │  Versions supported     │
      │  by CAPO                │
      │                         │
      └─────────────────────────┘
```

"Versions" could be changed to "features" in the diagram also, but a version difference does not always expose a difference in features at the CAPO level.
Therefore "versions" is used in the diagram to highlight the need to support multiple versions even though that may not give any new features.

### Goals

1. Detect user requested features and their version needs.

2. Detect server version support.

### Non-goals

- To allow the user to set the versions freely.

### CAPO microversions history

Initially no microversions were specified (only versions).
In OpenStack, there is support for microversions in Nova, Cinder, Manila and Barbican.
Neutron uses extensions and Keystone uses neither.
So far only Nova (compute) has any microversion specified in CAPO.

The compute client has been bumped from the original v2 version in order to add support for specific features:

- 2.53 is needed for server tags (https://github.com/kubernetes-sigs/cluster-api-provider-openstack/pull/924)
- 2.60 is needed for multi-attach volumes (https://github.com/kubernetes-sigs/cluster-api-provider-openstack/pull/1498)

## Proposal

### User stories

#### Story 1

As a user of CAPO, I would like to create a cluster in an older OpenStack environment that doesn't support `currently hard coded microversion` and I am fine with limiting the features available to me based on this older version.

#### Story 2

As a user of CAPO, I would like to create a cluster in an OpenStack environment that requires a higher microversion than `currently hard coded microversion`.

*Note: The requirement could be because of configuration. For example there could be only multi-attach volume types configured.*

#### Story 3

As developer/maintainer of CAPO, I want to ensure that the code is executed (and tested) with a specific microversion so that the result is deterministic.

#### Story 4

As a developer/maintainer of CAPO, I want to keep the code readable and limit the burden of testing and maintaining support for multiple microversions.

### Implementation details and notes

There is already one part of CAPO that actually has dynamic versioning, thanks to Gophercloud.
The [authentication](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/904381c7f31f8fdda83586f208339cd2aab53e78/pkg/scope/provider.go#L173) flow makes use of [this function](https://github.com/gophercloud/gophercloud/blob/6e1423b668969548d46fd862202a3bf9623b52e0/openstack/client.go#L92-L112) that chooses the version from a list of supported versions, provided by the caller, and the range of versions supported by the server.

The function to choose version could be used to reach goal 2, i.e. take the servers supported versions into account.
This would be enough to support Nova 2.53 and 2.60.
There was no other changes in CAPO for this version bump, so both versions would simply be listed as supported and could be used depending on the server requirements.

A more complex example can be seen in the OpenStack cloud provider repository.
It has [this function](https://github.com/kubernetes/cloud-provider-openstack/blob/35ce5a0c59da23795fc48b292e3736f8cb0bb10f/pkg/util/openstack/loadbalancer.go#L101-L162) for checking if a specified feature is supported by the Octavia instance.

Goal 1 is trickier.
It will require CAPO to match used features with required versions.
The outcome would affect the list of supported versions that can be used when negotiating the version with the server.
Luckily, we only have 2 version changes so far, and one of them did not require any code changes in CAPO.
In other words, we have Nova v2 that can be used without server tags and (2.53 or 2.60) that can be used if server tags are used.

The first possible place to detect usage of a specific feature is in the webhook.
But we do not allow API calls to OpenStack in the webhook, so this is not an option.
Another possibility is to check at time of creation, like we do for [trunk support](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/89b6ead01a90eeca64b89cef4e0039e19c3dd93a/pkg/cloud/services/compute/instance.go#L207-L215).
Just before getting the client and issuing the API call, we know both what we are about to do and what the input is.
This means we can determine the version requirements for each specific call and use this information when creating the client.

With this approach we get fine grained control and an implementation that can be extended with more versions as needed.
However, it can become complex with checks sprinkled through out the code base.
To mitigate this, the proposal is to have unified logic for determining the version requirements (per OpenStack service).
I.e. avoid setting different versions per call just because it is possible.
If we determine that Nova 2.53 or 2.60 is needed for some API call, for example, then use that throughout the current context.

## Alternatives

### Fixed version

This alternative is basically to continue as before.
The (micro)version is hard coded in CAPO so one and only one is supported and used for all API calls.
This means that using CAPO in an environment that requires a different microversion is impossible without making a custom build.
On the other hand, complexity is very low with this approach.

### Version set by the user in clouds.yaml

In this alternative the user would be able to override the version set in CAPO.
The obvious consequence is that the user could very easily break things by using versions that are not tested or supported.
On the other hand, it gives maximum flexibility.
The user can use any version, but the responsibility to ensure it is working also falls on them.

The `clouds.yaml` configuration file already has support for specifying the version so this should be familiar to many users.
However, Gophercloud does not support reading all the versions from this file as of now.
The main reason for avoiding this alternative, though, is the lack of control that CAPO has.
With a well defined list of supported and tested versions, perhaps it could still be considered.
