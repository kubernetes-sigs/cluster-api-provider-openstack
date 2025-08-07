<h2 id="infrastructure.cluster.x-k8s.io/v1alpha1">infrastructure.cluster.x-k8s.io/v1alpha1</h2>
<p>
<p>package v1alpha1 contains API Schema definitions for the infrastructure v1alpha1 API group</p>
</p>
Resource Types:
<ul><li>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackServer">OpenStackServer</a>
</li></ul>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackServer">OpenStackServer
</h3>
<p>
<p>OpenStackServer is the Schema for the openstackservers API.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
infrastructure.cluster.x-k8s.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>OpenStackServer</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
Kubernetes meta/v1.ObjectMeta
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackServerSpec">
OpenStackServerSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>additionalBlockDevices</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.AdditionalBlockDevice">
[]sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.AdditionalBlockDevice
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AdditionalBlockDevices is a list of specifications for additional block devices to attach to the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>availabilityZone</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>AvailabilityZone is the availability zone in which to create the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>configDrive</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>ConfigDrive is a flag to enable config drive for the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>flavor</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The flavor reference for the flavor for the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>flavorID</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>FlavorID allows flavors to be specified by ID.  This field takes precedence
over Flavor.</p>
</td>
</tr>
<tr>
<td>
<code>floatingIPPoolRef</code><br/>
<em>
Kubernetes core/v1.TypedLocalObjectReference
</em>
</td>
<td>
<em>(Optional)</em>
<p>FloatingIPPoolRef is a reference to a FloatingIPPool to allocate a floating IP from.</p>
</td>
</tr>
<tr>
<td>
<code>identityRef</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackIdentityReference">
sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.OpenStackIdentityReference
</a>
</em>
</td>
<td>
<p>IdentityRef is a reference to a secret holding OpenStack credentials.</p>
</td>
</tr>
<tr>
<td>
<code>image</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.ImageParam">
sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.ImageParam
</a>
</em>
</td>
<td>
<p>The image to use for the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>ports</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.PortOpts">
[]sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.PortOpts
</a>
</em>
</td>
<td>
<p>Ports to be attached to the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>rootVolume</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.RootVolume">
sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.RootVolume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>RootVolume is the specification for the root volume of the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>sshKeyName</code><br/>
<em>
string
</em>
</td>
<td>
<p>SSHKeyName is the name of the SSH key to inject in the instance.</p>
</td>
</tr>
<tr>
<td>
<code>securityGroups</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupParam">
[]sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.SecurityGroupParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecurityGroups is a list of security groups names to assign to the instance.</p>
</td>
</tr>
<tr>
<td>
<code>serverGroup</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.ServerGroupParam">
sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.ServerGroupParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServerGroup is the server group to which the server instance belongs.</p>
</td>
</tr>
<tr>
<td>
<code>serverMetadata</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.ServerMetadata">
[]sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.ServerMetadata
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServerMetadata is a map of key value pairs to add to the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>tags</code><br/>
<em>
[]string
</em>
</td>
<td>
<p>Tags which will be added to the machine and all dependent resources
which support them. These are in addition to Tags defined on the
cluster.
Requires Nova api 2.52 minimum!</p>
</td>
</tr>
<tr>
<td>
<code>trunk</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Trunk is a flag to indicate if the server instance is created on a trunk port or not.</p>
</td>
</tr>
<tr>
<td>
<code>userDataRef</code><br/>
<em>
Kubernetes core/v1.LocalObjectReference
</em>
</td>
<td>
<em>(Optional)</em>
<p>UserDataRef is a reference to a secret containing the user data to
be injected into the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>schedulerHintAdditionalProperties</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.SchedulerHintAdditionalProperty">
[]sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.SchedulerHintAdditionalProperty
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SchedulerHintAdditionalProperties are arbitrary key/value pairs that provide additional hints
to the OpenStack scheduler. These hints can influence how instances are placed on the infrastructure,
such as specifying certain host aggregates or availability zones.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackServerStatus">
OpenStackServerStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackFloatingIPPool">OpenStackFloatingIPPool
</h3>
<p>
<p>OpenStackFloatingIPPool is the Schema for the openstackfloatingippools API.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code><br/>
<em>
Kubernetes meta/v1.ObjectMeta
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackFloatingIPPoolSpec">
OpenStackFloatingIPPoolSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>preAllocatedFloatingIPs</code><br/>
<em>
[]string
</em>
</td>
<td>
<p>PreAllocatedFloatingIPs is a list of floating IPs precreated in OpenStack that should be used by this pool.
These are used before allocating new ones and are not deleted from OpenStack when the pool is deleted.</p>
</td>
</tr>
<tr>
<td>
<code>maxIPs</code><br/>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>MaxIPs is the maximum number of floating ips that can be allocated from this pool, if nil there is no limit.
If set, the pool will stop allocating floating ips when it reaches this number of ClaimedIPs.</p>
</td>
</tr>
<tr>
<td>
<code>identityRef</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackIdentityReference">
sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.OpenStackIdentityReference
</a>
</em>
</td>
<td>
<p>IdentityRef is a reference to a identity to be used when reconciling this pool.</p>
</td>
</tr>
<tr>
<td>
<code>floatingIPNetwork</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.NetworkParam">
sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.NetworkParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>FloatingIPNetwork is the external network to use for floating ips, if there&rsquo;s only one external network it will be used by default</p>
</td>
</tr>
<tr>
<td>
<code>reclaimPolicy</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ReclaimPolicy">
ReclaimPolicy
</a>
</em>
</td>
<td>
<p>The stratergy to use for reclaiming floating ips when they are released from a machine</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackFloatingIPPoolStatus">
OpenStackFloatingIPPoolStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackFloatingIPPoolSpec">OpenStackFloatingIPPoolSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackFloatingIPPool">OpenStackFloatingIPPool</a>)
</p>
<p>
<p>OpenStackFloatingIPPoolSpec defines the desired state of OpenStackFloatingIPPool.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>preAllocatedFloatingIPs</code><br/>
<em>
[]string
</em>
</td>
<td>
<p>PreAllocatedFloatingIPs is a list of floating IPs precreated in OpenStack that should be used by this pool.
These are used before allocating new ones and are not deleted from OpenStack when the pool is deleted.</p>
</td>
</tr>
<tr>
<td>
<code>maxIPs</code><br/>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>MaxIPs is the maximum number of floating ips that can be allocated from this pool, if nil there is no limit.
If set, the pool will stop allocating floating ips when it reaches this number of ClaimedIPs.</p>
</td>
</tr>
<tr>
<td>
<code>identityRef</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackIdentityReference">
sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.OpenStackIdentityReference
</a>
</em>
</td>
<td>
<p>IdentityRef is a reference to a identity to be used when reconciling this pool.</p>
</td>
</tr>
<tr>
<td>
<code>floatingIPNetwork</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.NetworkParam">
sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.NetworkParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>FloatingIPNetwork is the external network to use for floating ips, if there&rsquo;s only one external network it will be used by default</p>
</td>
</tr>
<tr>
<td>
<code>reclaimPolicy</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ReclaimPolicy">
ReclaimPolicy
</a>
</em>
</td>
<td>
<p>The stratergy to use for reclaiming floating ips when they are released from a machine</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackFloatingIPPoolStatus">OpenStackFloatingIPPoolStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackFloatingIPPool">OpenStackFloatingIPPool</a>)
</p>
<p>
<p>OpenStackFloatingIPPoolStatus defines the observed state of OpenStackFloatingIPPool.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>claimedIPs</code><br/>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>availableIPs</code><br/>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>failedIPs</code><br/>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>FailedIPs contains a list of floating ips that failed to be allocated</p>
</td>
</tr>
<tr>
<td>
<code>floatingIPNetwork</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.NetworkStatus">
sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.NetworkStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>floatingIPNetwork contains information about the network used for floating ips</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code><br/>
<em>
sigs.k8s.io/cluster-api/api/core/v1beta1.Conditions
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackServerSpec">OpenStackServerSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackServer">OpenStackServer</a>)
</p>
<p>
<p>OpenStackServerSpec defines the desired state of OpenStackServer.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>additionalBlockDevices</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.AdditionalBlockDevice">
[]sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.AdditionalBlockDevice
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AdditionalBlockDevices is a list of specifications for additional block devices to attach to the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>availabilityZone</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>AvailabilityZone is the availability zone in which to create the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>configDrive</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>ConfigDrive is a flag to enable config drive for the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>flavor</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The flavor reference for the flavor for the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>flavorID</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>FlavorID allows flavors to be specified by ID.  This field takes precedence
over Flavor.</p>
</td>
</tr>
<tr>
<td>
<code>floatingIPPoolRef</code><br/>
<em>
Kubernetes core/v1.TypedLocalObjectReference
</em>
</td>
<td>
<em>(Optional)</em>
<p>FloatingIPPoolRef is a reference to a FloatingIPPool to allocate a floating IP from.</p>
</td>
</tr>
<tr>
<td>
<code>identityRef</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackIdentityReference">
sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.OpenStackIdentityReference
</a>
</em>
</td>
<td>
<p>IdentityRef is a reference to a secret holding OpenStack credentials.</p>
</td>
</tr>
<tr>
<td>
<code>image</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.ImageParam">
sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.ImageParam
</a>
</em>
</td>
<td>
<p>The image to use for the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>ports</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.PortOpts">
[]sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.PortOpts
</a>
</em>
</td>
<td>
<p>Ports to be attached to the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>rootVolume</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.RootVolume">
sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.RootVolume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>RootVolume is the specification for the root volume of the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>sshKeyName</code><br/>
<em>
string
</em>
</td>
<td>
<p>SSHKeyName is the name of the SSH key to inject in the instance.</p>
</td>
</tr>
<tr>
<td>
<code>securityGroups</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupParam">
[]sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.SecurityGroupParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecurityGroups is a list of security groups names to assign to the instance.</p>
</td>
</tr>
<tr>
<td>
<code>serverGroup</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.ServerGroupParam">
sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.ServerGroupParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServerGroup is the server group to which the server instance belongs.</p>
</td>
</tr>
<tr>
<td>
<code>serverMetadata</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.ServerMetadata">
[]sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.ServerMetadata
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServerMetadata is a map of key value pairs to add to the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>tags</code><br/>
<em>
[]string
</em>
</td>
<td>
<p>Tags which will be added to the machine and all dependent resources
which support them. These are in addition to Tags defined on the
cluster.
Requires Nova api 2.52 minimum!</p>
</td>
</tr>
<tr>
<td>
<code>trunk</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Trunk is a flag to indicate if the server instance is created on a trunk port or not.</p>
</td>
</tr>
<tr>
<td>
<code>userDataRef</code><br/>
<em>
Kubernetes core/v1.LocalObjectReference
</em>
</td>
<td>
<em>(Optional)</em>
<p>UserDataRef is a reference to a secret containing the user data to
be injected into the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>schedulerHintAdditionalProperties</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.SchedulerHintAdditionalProperty">
[]sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.SchedulerHintAdditionalProperty
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SchedulerHintAdditionalProperties are arbitrary key/value pairs that provide additional hints
to the OpenStack scheduler. These hints can influence how instances are placed on the infrastructure,
such as specifying certain host aggregates or availability zones.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackServerStatus">OpenStackServerStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackServer">OpenStackServer</a>)
</p>
<p>
<p>OpenStackServerStatus defines the observed state of OpenStackServer.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ready</code><br/>
<em>
bool
</em>
</td>
<td>
<p>Ready is true when the OpenStack server is ready.</p>
</td>
</tr>
<tr>
<td>
<code>instanceID</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>InstanceID is the ID of the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>instanceState</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.InstanceState">
sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.InstanceState
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>InstanceState is the state of the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>addresses</code><br/>
<em>
[]Kubernetes core/v1.NodeAddress
</em>
</td>
<td>
<em>(Optional)</em>
<p>Addresses is the list of addresses of the server instance.</p>
</td>
</tr>
<tr>
<td>
<code>resolved</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ResolvedServerSpec">
ResolvedServerSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Resolved contains parts of the machine spec with all external
references fully resolved.</p>
</td>
</tr>
<tr>
<td>
<code>resources</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ServerResources">
ServerResources
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Resources contains references to OpenStack resources created for the machine.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code><br/>
<em>
sigs.k8s.io/cluster-api/api/core/v1beta1.Conditions
</em>
</td>
<td>
<em>(Optional)</em>
<p>Conditions defines current service state of the OpenStackServer.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.ReclaimPolicy">ReclaimPolicy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackFloatingIPPoolSpec">OpenStackFloatingIPPoolSpec</a>)
</p>
<p>
<p>ReclaimPolicy is a string type alias to represent reclaim policies for floating ips.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Delete&#34;</p></td>
<td><p>ReclaimDelete is the reclaim policy for floating ips.</p>
</td>
</tr><tr><td><p>&#34;Retain&#34;</p></td>
<td><p>ReclaimRetain is the reclaim policy for floating ips.</p>
</td>
</tr></tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.ResolvedServerSpec">ResolvedServerSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackServerStatus">OpenStackServerStatus</a>)
</p>
<p>
<p>ResolvedServerSpec contains resolved references to resources required by the server.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>serverGroupID</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServerGroupID is the ID of the server group the server should be added to and is calculated based on ServerGroupFilter.</p>
</td>
</tr>
<tr>
<td>
<code>imageID</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImageID is the ID of the image to use for the server and is calculated based on ImageFilter.</p>
</td>
</tr>
<tr>
<td>
<code>flavorID</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>FlavorID is the ID of the flavor to use.</p>
</td>
</tr>
<tr>
<td>
<code>ports</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.ResolvedPortSpec">
[]sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.ResolvedPortSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Ports is the fully resolved list of ports to create for the server.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.ServerResources">ServerResources
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackServerStatus">OpenStackServerStatus</a>)
</p>
<p>
<p>ServerResources contains references to OpenStack resources created for the server.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ports</code><br/>
<em>
<a href="https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api#infrastructure.cluster.x-k8s.io/v1beta1.PortStatus">
[]sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1.PortStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Ports is the status of the ports created for the server.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.ServerStatusError">ServerStatusError
(<code>string</code> alias)</p></h3>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;CreateError&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>.
</em></p>
