<h2 id="infrastructure.cluster.x-k8s.io/v1beta1">infrastructure.cluster.x-k8s.io/v1beta1</h2>
<p>
<p>Package v1beta1 contains API Schema definitions for the infrastructure v1beta1 API group.</p>
</p>
Resource Types:
<ul><li>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackCluster">OpenStackCluster</a>
</li><li>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterTemplate">OpenStackClusterTemplate</a>
</li><li>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachine">OpenStackMachine</a>
</li><li>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineTemplate">OpenStackMachineTemplate</a>
</li></ul>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.OpenStackCluster">OpenStackCluster
</h3>
<p>
<p>OpenStackCluster is the Schema for the openstackclusters API.</p>
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
infrastructure.cluster.x-k8s.io/v1beta1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>OpenStackCluster</code></td>
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
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterSpec">
OpenStackClusterSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>managedSubnets</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SubnetSpec">
[]SubnetSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ManagedSubnets describe OpenStack Subnets to be created. Cluster actuator will create a network,
subnets with the defined CIDR, and a router connected to these subnets. Currently only one IPv4
subnet is supported. If you leave this empty, no network will be created.</p>
</td>
</tr>
<tr>
<td>
<code>router</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.RouterParam">
RouterParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Router specifies an existing router to be used if ManagedSubnets are
specified. If specified, no new router will be created.</p>
</td>
</tr>
<tr>
<td>
<code>network</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NetworkParam">
NetworkParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Network specifies an existing network to use if no ManagedSubnets
are specified.</p>
</td>
</tr>
<tr>
<td>
<code>subnets</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SubnetParam">
[]SubnetParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Subnets specifies existing subnets to use if not ManagedSubnets are
specified. All subnets must be in the network specified by Network.
There can be zero, one, or two subnets. If no subnets are specified,
all subnets in Network will be used. If 2 subnets are specified, one
must be IPv4 and the other IPv6.</p>
</td>
</tr>
<tr>
<td>
<code>networkMTU</code><br/>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>NetworkMTU sets the maximum transmission unit (MTU) value to address fragmentation for the private network ID.
This value will be used only if the Cluster actuator creates the network.
If left empty, the network will have the default MTU defined in Openstack network service.
To use this field, the Openstack installation requires the net-mtu neutron API extension.</p>
</td>
</tr>
<tr>
<td>
<code>externalRouterIPs</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ExternalRouterIPParam">
[]ExternalRouterIPParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExternalRouterIPs is an array of externalIPs on the respective subnets.
This is necessary if the router needs a fixed ip in a specific subnet.</p>
</td>
</tr>
<tr>
<td>
<code>externalNetwork</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NetworkParam">
NetworkParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExternalNetwork is the OpenStack Network to be used to get public internet to the VMs.
This option is ignored if DisableExternalNetwork is set to true.</p>
<p>If ExternalNetwork is defined it must refer to exactly one external network.</p>
<p>If ExternalNetwork is not defined or is empty the controller will use any
existing external network as long as there is only one. It is an
error if ExternalNetwork is not defined and there are multiple
external networks unless DisableExternalNetwork is also set.</p>
<p>If ExternalNetwork is not defined and there are no external networks
the controller will proceed as though DisableExternalNetwork was set.</p>
</td>
</tr>
<tr>
<td>
<code>disableExternalNetwork</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisableExternalNetwork specifies whether or not to attempt to connect the cluster
to an external network. This allows for the creation of clusters when connecting
to an external network is not possible or desirable, e.g. if using a provider network.</p>
</td>
</tr>
<tr>
<td>
<code>apiServerLoadBalancer</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.APIServerLoadBalancer">
APIServerLoadBalancer
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>APIServerLoadBalancer configures the optional LoadBalancer for the APIServer.
If not specified, no load balancer will be created for the API server.</p>
</td>
</tr>
<tr>
<td>
<code>disableAPIServerFloatingIP</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisableAPIServerFloatingIP determines whether or not to attempt to attach a floating
IP to the API server. This allows for the creation of clusters when attaching a floating
IP to the API server (and hence, in many cases, exposing the API server to the internet)
is not possible or desirable, e.g. if using a shared VLAN for communication between
management and workload clusters or when the management cluster is inside the
project network.
This option requires that the API server use a VIP on the cluster network so that the
underlying machines can change without changing ControlPlaneEndpoint.Host.
When using a managed load balancer, this VIP will be managed automatically.
If not using a managed load balancer, cluster configuration will fail without additional
configuration to manage the VIP on the control plane machines, which falls outside of
the scope of this controller.</p>
</td>
</tr>
<tr>
<td>
<code>apiServerFloatingIP</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>APIServerFloatingIP is the floatingIP which will be associated with the API server.
The floatingIP will be created if it does not already exist.
If not specified, a new floatingIP is allocated.
This field is not used if DisableAPIServerFloatingIP is set to true.</p>
</td>
</tr>
<tr>
<td>
<code>apiServerFixedIP</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>APIServerFixedIP is the fixed IP which will be associated with the API server.
In the case where the API server has a floating IP but not a managed load balancer,
this field is not used.
If a managed load balancer is used and this field is not specified, a fixed IP will
be dynamically allocated for the load balancer.
If a managed load balancer is not used AND the API server floating IP is disabled,
this field MUST be specified and should correspond to a pre-allocated port that
holds the fixed IP to be used as a VIP.</p>
</td>
</tr>
<tr>
<td>
<code>apiServerPort</code><br/>
<em>
uint16
</em>
</td>
<td>
<em>(Optional)</em>
<p>APIServerPort is the port on which the listener on the APIServer
will be created. If specified, it must be an integer between 0 and 65535.</p>
</td>
</tr>
<tr>
<td>
<code>managedSecurityGroups</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ManagedSecurityGroups">
ManagedSecurityGroups
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ManagedSecurityGroups determines whether OpenStack security groups for the cluster
will be managed by the OpenStack provider or whether pre-existing security groups will
be specified as part of the configuration.
By default, the managed security groups have rules that allow the Kubelet, etcd, and the
Kubernetes API server to function correctly.
It&rsquo;s possible to add additional rules to the managed security groups.
When defined to an empty struct, the managed security groups will be created with the default rules.</p>
</td>
</tr>
<tr>
<td>
<code>disablePortSecurity</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisablePortSecurity disables the port security of the network created for the
Kubernetes cluster, which also disables SecurityGroups</p>
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
<em>(Optional)</em>
<p>Tags to set on all resources in cluster which support tags</p>
</td>
</tr>
<tr>
<td>
<code>controlPlaneEndpoint</code><br/>
<em>
<a href="https://doc.crds.dev/github.com/kubernetes-sigs/cluster-api@v1.6.0">
sigs.k8s.io/cluster-api/api/v1beta1.APIEndpoint
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
It is normally populated automatically by the OpenStackCluster
controller during cluster provisioning. If it is set on creation the
control plane endpoint will use the values set here in preference to
values set elsewhere.
ControlPlaneEndpoint cannot be modified after ControlPlaneEndpoint.Host has been set.</p>
</td>
</tr>
<tr>
<td>
<code>controlPlaneAvailabilityZones</code><br/>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ControlPlaneAvailabilityZones is the set of availability zones which
control plane machines may be deployed to.</p>
</td>
</tr>
<tr>
<td>
<code>controlPlaneOmitAvailabilityZone</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>ControlPlaneOmitAvailabilityZone causes availability zone to be
omitted when creating control plane nodes, allowing the Nova
scheduler to make a decision on which availability zone to use based
on other scheduling constraints</p>
</td>
</tr>
<tr>
<td>
<code>bastion</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.Bastion">
Bastion
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Bastion is the OpenStack instance to login the nodes</p>
<p>As a rolling update is not ideal during a bastion host session, we
prevent changes to a running bastion configuration. To make changes, it&rsquo;s required
to first set <code>enabled: false</code> which will remove the bastion and then changes can be made.</p>
</td>
</tr>
<tr>
<td>
<code>identityRef</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackIdentityReference">
OpenStackIdentityReference
</a>
</em>
</td>
<td>
<p>IdentityRef is a reference to a secret holding OpenStack credentials
to be used when reconciling this cluster. It is also to reconcile
machines unless overridden in the machine spec.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterStatus">
OpenStackClusterStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterTemplate">OpenStackClusterTemplate
</h3>
<p>
<p>OpenStackClusterTemplate is the Schema for the openstackclustertemplates API.</p>
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
infrastructure.cluster.x-k8s.io/v1beta1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>OpenStackClusterTemplate</code></td>
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
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterTemplateSpec">
OpenStackClusterTemplateSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>template</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterTemplateResource">
OpenStackClusterTemplateResource
</a>
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachine">OpenStackMachine
</h3>
<p>
<p>OpenStackMachine is the Schema for the openstackmachines API.</p>
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
infrastructure.cluster.x-k8s.io/v1beta1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>OpenStackMachine</code></td>
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
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineSpec">
OpenStackMachineSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>providerID</code><br/>
<em>
string
</em>
</td>
<td>
<p>ProviderID is the unique identifier as specified by the cloud provider.</p>
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
<p>The flavor reference for the flavor for your server instance.</p>
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
<p>FlavorID allows flavors to be specified by ID.  This field takes precedence
over Flavor.</p>
</td>
</tr>
<tr>
<td>
<code>image</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ImageParam">
ImageParam
</a>
</em>
</td>
<td>
<p>The image to use for your server instance.
If the rootVolume is specified, this will be used when creating the root volume.</p>
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
<p>The ssh key to inject in the instance</p>
</td>
</tr>
<tr>
<td>
<code>ports</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.PortOpts">
[]PortOpts
</a>
</em>
</td>
<td>
<p>Ports to be attached to the server instance. They are created if a port with the given name does not already exist.
If not specified a default port will be added for the default cluster network.</p>
</td>
</tr>
<tr>
<td>
<code>securityGroups</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupParam">
[]SecurityGroupParam
</a>
</em>
</td>
<td>
<p>The names of the security groups to assign to the instance</p>
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
<p>Whether the server instance is created on a trunk port or not.</p>
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
<code>serverMetadata</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ServerMetadata">
[]ServerMetadata
</a>
</em>
</td>
<td>
<p>Metadata mapping. Allows you to create a map of key value pairs to add to the server instance.</p>
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
<p>Config Drive support</p>
</td>
</tr>
<tr>
<td>
<code>rootVolume</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.RootVolume">
RootVolume
</a>
</em>
</td>
<td>
<p>The volume metadata to boot from</p>
</td>
</tr>
<tr>
<td>
<code>additionalBlockDevices</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.AdditionalBlockDevice">
[]AdditionalBlockDevice
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AdditionalBlockDevices is a list of specifications for additional block devices to attach to the server instance</p>
</td>
</tr>
<tr>
<td>
<code>serverGroup</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ServerGroupParam">
ServerGroupParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The server group to assign the machine to.</p>
</td>
</tr>
<tr>
<td>
<code>identityRef</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackIdentityReference">
OpenStackIdentityReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IdentityRef is a reference to a secret holding OpenStack credentials
to be used when reconciling this machine. If not specified, the
credentials specified in the cluster will be used.</p>
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
<p>floatingIPPoolRef is a reference to a IPPool that will be assigned
to an IPAddressClaim. Once the IPAddressClaim is fulfilled, the FloatingIP
will be assigned to the OpenStackMachine.</p>
</td>
</tr>
<tr>
<td>
<code>schedulerHintAdditionalProperties</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SchedulerHintAdditionalProperty">
[]SchedulerHintAdditionalProperty
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
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineStatus">
OpenStackMachineStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineTemplate">OpenStackMachineTemplate
</h3>
<p>
<p>OpenStackMachineTemplate is the Schema for the openstackmachinetemplates API.</p>
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
infrastructure.cluster.x-k8s.io/v1beta1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>OpenStackMachineTemplate</code></td>
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
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineTemplateSpec">
OpenStackMachineTemplateSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>template</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineTemplateResource">
OpenStackMachineTemplateResource
</a>
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.APIServerLoadBalancer">APIServerLoadBalancer
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterSpec">OpenStackClusterSpec</a>)
</p>
<p>
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
<code>enabled</code><br/>
<em>
bool
</em>
</td>
<td>
<p>Enabled defines whether a load balancer should be created. This value
defaults to true if an APIServerLoadBalancer is given.</p>
<p>There is no reason to set this to false. To disable creation of the
API server loadbalancer, omit the APIServerLoadBalancer field in the
cluster spec instead.</p>
</td>
</tr>
<tr>
<td>
<code>additionalPorts</code><br/>
<em>
[]int
</em>
</td>
<td>
<em>(Optional)</em>
<p>AdditionalPorts adds additional tcp ports to the load balancer.</p>
</td>
</tr>
<tr>
<td>
<code>allowedCIDRs</code><br/>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>AllowedCIDRs restrict access to all API-Server listeners to the given address CIDRs.</p>
</td>
</tr>
<tr>
<td>
<code>provider</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Provider specifies name of a specific Octavia provider to use for the
API load balancer. The Octavia default will be used if it is not
specified.</p>
</td>
</tr>
<tr>
<td>
<code>network</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NetworkParam">
NetworkParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Network defines which network should the load balancer be allocated on.</p>
</td>
</tr>
<tr>
<td>
<code>subnets</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SubnetParam">
[]SubnetParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Subnets define which subnets should the load balancer be allocated on.
It is expected that subnets are located on the network specified in this resource.
Only the first element is taken into account.
kubebuilder:validation:MaxLength:=2</p>
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
<p>AvailabilityZone is the failure domain that will be used to create the APIServerLoadBalancer Spec.</p>
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
<p>Flavor is the flavor name that will be used to create the APIServerLoadBalancer Spec.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.AdditionalBlockDevice">AdditionalBlockDevice
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineSpec">OpenStackMachineSpec</a>)
</p>
<p>
<p>AdditionalBlockDevice is a block device to attach to the server.</p>
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
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name of the block device in the context of a machine.
If the block device is a volume, the Cinder volume will be named
as a combination of the machine name and this name.
Also, this name will be used for tagging the block device.
Information about the block device tag can be obtained from the OpenStack
metadata API or the config drive.
Name cannot be &lsquo;root&rsquo;, which is reserved for the root volume.</p>
</td>
</tr>
<tr>
<td>
<code>sizeGiB</code><br/>
<em>
int
</em>
</td>
<td>
<p>SizeGiB is the size of the block device in gibibytes (GiB).</p>
</td>
</tr>
<tr>
<td>
<code>storage</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.BlockDeviceStorage">
BlockDeviceStorage
</a>
</em>
</td>
<td>
<p>Storage specifies the storage type of the block device and
additional storage options.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.AddressPair">AddressPair
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ResolvedPortSpecFields">ResolvedPortSpecFields</a>)
</p>
<p>
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
<code>ipAddress</code><br/>
<em>
string
</em>
</td>
<td>
<p>IPAddress is the IP address of the allowed address pair. Depending on
the configuration of Neutron, it may be supported to specify a CIDR
instead of a specific IP address.</p>
</td>
</tr>
<tr>
<td>
<code>macAddress</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>MACAddress is the MAC address of the allowed address pair. If not
specified, the MAC address will be the MAC address of the port.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.AllocationPool">AllocationPool
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SubnetSpec">SubnetSpec</a>)
</p>
<p>
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
<code>start</code><br/>
<em>
string
</em>
</td>
<td>
<p>Start represents the start of the AllocationPool, that is the lowest IP of the pool.</p>
</td>
</tr>
<tr>
<td>
<code>end</code><br/>
<em>
string
</em>
</td>
<td>
<p>End represents the end of the AlloctionPool, that is the highest IP of the pool.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.Bastion">Bastion
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterSpec">OpenStackClusterSpec</a>)
</p>
<p>
<p>Bastion represents basic information about the bastion node. If you enable bastion, the spec has to be specified.</p>
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
<code>enabled</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Enabled means that bastion is enabled. The bastion is enabled by
default if this field is not specified. Set this field to false to disable the
bastion.</p>
<p>It is not currently possible to remove the bastion from the cluster
spec without first disabling it by setting this field to false and
waiting until the bastion has been deleted.</p>
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineSpec">
OpenStackMachineSpec
</a>
</em>
</td>
<td>
<p>Spec for the bastion itself</p>
<br/>
<br/>
<table>
</table>
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
<p>AvailabilityZone is the failure domain that will be used to create the Bastion Spec.</p>
</td>
</tr>
<tr>
<td>
<code>floatingIP</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>FloatingIP which will be associated to the bastion machine. It&rsquo;s the IP address, not UUID.
The floating IP should already exist and should not be associated with a port. If FIP of this address does not
exist, CAPO will try to create it, but by default only OpenStack administrators have privileges to do so.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.BastionStatus">BastionStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterStatus">OpenStackClusterStatus</a>)
</p>
<p>
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
<code>id</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
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
</td>
</tr>
<tr>
<td>
<code>state</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.InstanceState">
InstanceState
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ip</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>floatingIP</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>resolved</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ResolvedMachineSpec">
ResolvedMachineSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Resolved contains parts of the bastion&rsquo;s machine spec with all
external references fully resolved.</p>
</td>
</tr>
<tr>
<td>
<code>resources</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.MachineResources">
MachineResources
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Resources contains references to OpenStack resources created for the bastion.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.BindingProfile">BindingProfile
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ResolvedPortSpecFields">ResolvedPortSpecFields</a>)
</p>
<p>
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
<code>ovsHWOffload</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>OVSHWOffload enables or disables the OVS hardware offload feature.
This flag is not required on OpenStack clouds since Yoga as Nova will set it automatically when the port is attached.
See: <a href="https://bugs.launchpad.net/nova/+bug/2020813">https://bugs.launchpad.net/nova/+bug/2020813</a></p>
</td>
</tr>
<tr>
<td>
<code>trustedVF</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>TrustedVF enables or disables the “trusted mode” for the VF.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.BlockDeviceStorage">BlockDeviceStorage
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.AdditionalBlockDevice">AdditionalBlockDevice</a>)
</p>
<p>
<p>BlockDeviceStorage is the storage type of a block device to create and
contains additional storage options.</p>
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
<code>type</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.BlockDeviceType">
BlockDeviceType
</a>
</em>
</td>
<td>
<p>Type is the type of block device to create.
This can be either &ldquo;Volume&rdquo; or &ldquo;Local&rdquo;.</p>
</td>
</tr>
<tr>
<td>
<code>volume</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.BlockDeviceVolume">
BlockDeviceVolume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Volume contains additional storage options for a volume block device.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.BlockDeviceType">BlockDeviceType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.BlockDeviceStorage">BlockDeviceStorage</a>)
</p>
<p>
<p>BlockDeviceType defines the type of block device to create.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Local&#34;</p></td>
<td><p>LocalBlockDevice is an ephemeral block device attached to the server.</p>
</td>
</tr><tr><td><p>&#34;Volume&#34;</p></td>
<td><p>VolumeBlockDevice is a volume block device attached to the server.</p>
</td>
</tr></tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.BlockDeviceVolume">BlockDeviceVolume
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.BlockDeviceStorage">BlockDeviceStorage</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.RootVolume">RootVolume</a>)
</p>
<p>
<p>BlockDeviceVolume contains additional storage options for a volume block device.</p>
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
<code>type</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Type is the Cinder volume type of the volume.
If omitted, the default Cinder volume type that is configured in the OpenStack cloud
will be used.</p>
</td>
</tr>
<tr>
<td>
<code>availabilityZone</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.VolumeAvailabilityZone">
VolumeAvailabilityZone
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AvailabilityZone is the volume availability zone to create the volume
in. If not specified, the volume will be created without an explicit
availability zone.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.ExternalRouterIPParam">ExternalRouterIPParam
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterSpec">OpenStackClusterSpec</a>)
</p>
<p>
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
<code>fixedIP</code><br/>
<em>
string
</em>
</td>
<td>
<p>The FixedIP in the corresponding subnet</p>
</td>
</tr>
<tr>
<td>
<code>subnet</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SubnetParam">
SubnetParam
</a>
</em>
</td>
<td>
<p>The subnet in which the FixedIP is used for the Gateway of this router</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.FilterByNeutronTags">FilterByNeutronTags
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NetworkFilter">NetworkFilter</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.RouterFilter">RouterFilter</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupFilter">SecurityGroupFilter</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SubnetFilter">SubnetFilter</a>)
</p>
<p>
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
<code>tags</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NeutronTag">
[]NeutronTag
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Tags is a list of tags to filter by. If specified, the resource must
have all of the tags specified to be included in the result.</p>
</td>
</tr>
<tr>
<td>
<code>tagsAny</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NeutronTag">
[]NeutronTag
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TagsAny is a list of tags to filter by. If specified, the resource
must have at least one of the tags specified to be included in the
result.</p>
</td>
</tr>
<tr>
<td>
<code>notTags</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NeutronTag">
[]NeutronTag
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>NotTags is a list of tags to filter by. If specified, resources which
contain all of the given tags will be excluded from the result.</p>
</td>
</tr>
<tr>
<td>
<code>notTagsAny</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NeutronTag">
[]NeutronTag
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>NotTagsAny is a list of tags to filter by. If specified, resources
which contain any of the given tags will be excluded from the result.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.FixedIP">FixedIP
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.PortOpts">PortOpts</a>)
</p>
<p>
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
<code>subnet</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SubnetParam">
SubnetParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Subnet is an openstack subnet query that will return the id of a subnet to create
the fixed IP of a port in. This query must not return more than one subnet.</p>
</td>
</tr>
<tr>
<td>
<code>ipAddress</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>IPAddress is a specific IP address to assign to the port. If Subnet
is also specified, IPAddress must be a valid IP address in the
subnet. If Subnet is not specified, IPAddress must be a valid IP
address in any subnet of the port&rsquo;s network.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.IdentityRefProvider">IdentityRefProvider
</h3>
<p>
<p>IdentityRefProvider is an interface for obtaining OpenStack credentials from an API object</p>
</p>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.ImageFilter">ImageFilter
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ImageParam">ImageParam</a>)
</p>
<p>
<p>ImageFilter describes a query for an image.</p>
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
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The name of the desired image. If specified, the combination of name and tags must return a single matching image or an error will be raised.</p>
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
<em>(Optional)</em>
<p>The tags associated with the desired image. If specified, the combination of name and tags must return a single matching image or an error will be raised.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.ImageParam">ImageParam
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineSpec">OpenStackMachineSpec</a>)
</p>
<p>
<p>ImageParam describes a glance image. It can be specified by ID, filter, or a
reference to an ORC Image.</p>
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
<code>id</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ID is the uuid of the image. ID will not be validated before use.</p>
</td>
</tr>
<tr>
<td>
<code>filter</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ImageFilter">
ImageFilter
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Filter describes a query for an image. If specified, the combination
of name and tags must return a single matching image or an error will
be raised.</p>
</td>
</tr>
<tr>
<td>
<code>imageRef</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ResourceReference">
ResourceReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImageRef is a reference to an ORC Image in the same namespace as the
referring object.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.InstanceState">InstanceState
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.BastionStatus">BastionStatus</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineStatus">OpenStackMachineStatus</a>)
</p>
<p>
<p>InstanceState describes the state of an OpenStack instance.</p>
</p>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.LoadBalancer">LoadBalancer
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterStatus">OpenStackClusterStatus</a>)
</p>
<p>
<p>LoadBalancer represents basic information about the associated OpenStack LoadBalancer.</p>
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
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>id</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ip</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>internalIP</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>allowedCIDRs</code><br/>
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
<code>tags</code><br/>
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
<code>loadBalancerNetwork</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NetworkStatusWithSubnets">
NetworkStatusWithSubnets
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>LoadBalancerNetwork contains information about network and/or subnets which the
loadbalancer is allocated on.
If subnets are specified within the LoadBalancerNetwork currently only the first
subnet in the list is taken into account.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.MachineResources">MachineResources
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.BastionStatus">BastionStatus</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineStatus">OpenStackMachineStatus</a>)
</p>
<p>
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
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.PortStatus">
[]PortStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Ports is the status of the ports created for the machine.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.ManagedSecurityGroupName">ManagedSecurityGroupName
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupRuleSpec">SecurityGroupRuleSpec</a>)
</p>
<p>
</p>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.ManagedSecurityGroups">ManagedSecurityGroups
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterSpec">OpenStackClusterSpec</a>)
</p>
<p>
<p>ManagedSecurityGroups defines the desired state of security groups and rules for the cluster.</p>
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
<code>allNodesSecurityGroupRules</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupRuleSpec">
[]SecurityGroupRuleSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>allNodesSecurityGroupRules defines the rules that should be applied to all nodes.</p>
</td>
</tr>
<tr>
<td>
<code>controlPlaneNodesSecurityGroupRules</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupRuleSpec">
[]SecurityGroupRuleSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>controlPlaneNodesSecurityGroupRules defines the rules that should be applied to control plane nodes.</p>
</td>
</tr>
<tr>
<td>
<code>workerNodesSecurityGroupRules</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupRuleSpec">
[]SecurityGroupRuleSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>workerNodesSecurityGroupRules defines the rules that should be applied to worker nodes.</p>
</td>
</tr>
<tr>
<td>
<code>allowAllInClusterTraffic</code><br/>
<em>
bool
</em>
</td>
<td>
<p>AllowAllInClusterTraffic allows all ingress and egress traffic between cluster nodes when set to true.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.NetworkFilter">NetworkFilter
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NetworkParam">NetworkParam</a>)
</p>
<p>
<p>NetworkFilter specifies a query to select an OpenStack network. At least one property must be set.</p>
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
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>description</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>projectID</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>FilterByNeutronTags</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.FilterByNeutronTags">
FilterByNeutronTags
</a>
</em>
</td>
<td>
<p>
(Members of <code>FilterByNeutronTags</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.NetworkParam">NetworkParam
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.APIServerLoadBalancer">APIServerLoadBalancer</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterSpec">OpenStackClusterSpec</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.PortOpts">PortOpts</a>)
</p>
<p>
<p>NetworkParam specifies an OpenStack network. It may be specified by either ID or Filter, but not both.</p>
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
<code>id</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ID is the ID of the network to use. If ID is provided, the other filters cannot be provided. Must be in UUID format.</p>
</td>
</tr>
<tr>
<td>
<code>filter</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NetworkFilter">
NetworkFilter
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Filter specifies a filter to select an OpenStack network. If provided, cannot be empty.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.NetworkStatus">NetworkStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NetworkStatusWithSubnets">NetworkStatusWithSubnets</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterStatus">OpenStackClusterStatus</a>)
</p>
<p>
<p>NetworkStatus contains basic information about an existing neutron network.</p>
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
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>id</code><br/>
<em>
string
</em>
</td>
<td>
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
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.NetworkStatusWithSubnets">NetworkStatusWithSubnets
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.LoadBalancer">LoadBalancer</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterStatus">OpenStackClusterStatus</a>)
</p>
<p>
<p>NetworkStatusWithSubnets represents basic information about an existing neutron network and an associated set of subnets.</p>
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
<code>NetworkStatus</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NetworkStatus">
NetworkStatus
</a>
</em>
</td>
<td>
<p>
(Members of <code>NetworkStatus</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>subnets</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.Subnet">
[]Subnet
</a>
</em>
</td>
<td>
<p>Subnets is a list of subnets associated with the default cluster network. Machines which use the default cluster network will get an address from all of these subnets.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.NeutronTag">NeutronTag
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.FilterByNeutronTags">FilterByNeutronTags</a>)
</p>
<p>
<p>NeutronTag represents a tag on a Neutron resource.
It may not be empty and may not contain commas.</p>
</p>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterSpec">OpenStackClusterSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackCluster">OpenStackCluster</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterTemplateResource">OpenStackClusterTemplateResource</a>)
</p>
<p>
<p>OpenStackClusterSpec defines the desired state of OpenStackCluster.</p>
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
<code>managedSubnets</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SubnetSpec">
[]SubnetSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ManagedSubnets describe OpenStack Subnets to be created. Cluster actuator will create a network,
subnets with the defined CIDR, and a router connected to these subnets. Currently only one IPv4
subnet is supported. If you leave this empty, no network will be created.</p>
</td>
</tr>
<tr>
<td>
<code>router</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.RouterParam">
RouterParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Router specifies an existing router to be used if ManagedSubnets are
specified. If specified, no new router will be created.</p>
</td>
</tr>
<tr>
<td>
<code>network</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NetworkParam">
NetworkParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Network specifies an existing network to use if no ManagedSubnets
are specified.</p>
</td>
</tr>
<tr>
<td>
<code>subnets</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SubnetParam">
[]SubnetParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Subnets specifies existing subnets to use if not ManagedSubnets are
specified. All subnets must be in the network specified by Network.
There can be zero, one, or two subnets. If no subnets are specified,
all subnets in Network will be used. If 2 subnets are specified, one
must be IPv4 and the other IPv6.</p>
</td>
</tr>
<tr>
<td>
<code>networkMTU</code><br/>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>NetworkMTU sets the maximum transmission unit (MTU) value to address fragmentation for the private network ID.
This value will be used only if the Cluster actuator creates the network.
If left empty, the network will have the default MTU defined in Openstack network service.
To use this field, the Openstack installation requires the net-mtu neutron API extension.</p>
</td>
</tr>
<tr>
<td>
<code>externalRouterIPs</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ExternalRouterIPParam">
[]ExternalRouterIPParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExternalRouterIPs is an array of externalIPs on the respective subnets.
This is necessary if the router needs a fixed ip in a specific subnet.</p>
</td>
</tr>
<tr>
<td>
<code>externalNetwork</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NetworkParam">
NetworkParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExternalNetwork is the OpenStack Network to be used to get public internet to the VMs.
This option is ignored if DisableExternalNetwork is set to true.</p>
<p>If ExternalNetwork is defined it must refer to exactly one external network.</p>
<p>If ExternalNetwork is not defined or is empty the controller will use any
existing external network as long as there is only one. It is an
error if ExternalNetwork is not defined and there are multiple
external networks unless DisableExternalNetwork is also set.</p>
<p>If ExternalNetwork is not defined and there are no external networks
the controller will proceed as though DisableExternalNetwork was set.</p>
</td>
</tr>
<tr>
<td>
<code>disableExternalNetwork</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisableExternalNetwork specifies whether or not to attempt to connect the cluster
to an external network. This allows for the creation of clusters when connecting
to an external network is not possible or desirable, e.g. if using a provider network.</p>
</td>
</tr>
<tr>
<td>
<code>apiServerLoadBalancer</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.APIServerLoadBalancer">
APIServerLoadBalancer
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>APIServerLoadBalancer configures the optional LoadBalancer for the APIServer.
If not specified, no load balancer will be created for the API server.</p>
</td>
</tr>
<tr>
<td>
<code>disableAPIServerFloatingIP</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisableAPIServerFloatingIP determines whether or not to attempt to attach a floating
IP to the API server. This allows for the creation of clusters when attaching a floating
IP to the API server (and hence, in many cases, exposing the API server to the internet)
is not possible or desirable, e.g. if using a shared VLAN for communication between
management and workload clusters or when the management cluster is inside the
project network.
This option requires that the API server use a VIP on the cluster network so that the
underlying machines can change without changing ControlPlaneEndpoint.Host.
When using a managed load balancer, this VIP will be managed automatically.
If not using a managed load balancer, cluster configuration will fail without additional
configuration to manage the VIP on the control plane machines, which falls outside of
the scope of this controller.</p>
</td>
</tr>
<tr>
<td>
<code>apiServerFloatingIP</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>APIServerFloatingIP is the floatingIP which will be associated with the API server.
The floatingIP will be created if it does not already exist.
If not specified, a new floatingIP is allocated.
This field is not used if DisableAPIServerFloatingIP is set to true.</p>
</td>
</tr>
<tr>
<td>
<code>apiServerFixedIP</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>APIServerFixedIP is the fixed IP which will be associated with the API server.
In the case where the API server has a floating IP but not a managed load balancer,
this field is not used.
If a managed load balancer is used and this field is not specified, a fixed IP will
be dynamically allocated for the load balancer.
If a managed load balancer is not used AND the API server floating IP is disabled,
this field MUST be specified and should correspond to a pre-allocated port that
holds the fixed IP to be used as a VIP.</p>
</td>
</tr>
<tr>
<td>
<code>apiServerPort</code><br/>
<em>
uint16
</em>
</td>
<td>
<em>(Optional)</em>
<p>APIServerPort is the port on which the listener on the APIServer
will be created. If specified, it must be an integer between 0 and 65535.</p>
</td>
</tr>
<tr>
<td>
<code>managedSecurityGroups</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ManagedSecurityGroups">
ManagedSecurityGroups
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ManagedSecurityGroups determines whether OpenStack security groups for the cluster
will be managed by the OpenStack provider or whether pre-existing security groups will
be specified as part of the configuration.
By default, the managed security groups have rules that allow the Kubelet, etcd, and the
Kubernetes API server to function correctly.
It&rsquo;s possible to add additional rules to the managed security groups.
When defined to an empty struct, the managed security groups will be created with the default rules.</p>
</td>
</tr>
<tr>
<td>
<code>disablePortSecurity</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisablePortSecurity disables the port security of the network created for the
Kubernetes cluster, which also disables SecurityGroups</p>
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
<em>(Optional)</em>
<p>Tags to set on all resources in cluster which support tags</p>
</td>
</tr>
<tr>
<td>
<code>controlPlaneEndpoint</code><br/>
<em>
<a href="https://doc.crds.dev/github.com/kubernetes-sigs/cluster-api@v1.6.0">
sigs.k8s.io/cluster-api/api/v1beta1.APIEndpoint
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
It is normally populated automatically by the OpenStackCluster
controller during cluster provisioning. If it is set on creation the
control plane endpoint will use the values set here in preference to
values set elsewhere.
ControlPlaneEndpoint cannot be modified after ControlPlaneEndpoint.Host has been set.</p>
</td>
</tr>
<tr>
<td>
<code>controlPlaneAvailabilityZones</code><br/>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ControlPlaneAvailabilityZones is the set of availability zones which
control plane machines may be deployed to.</p>
</td>
</tr>
<tr>
<td>
<code>controlPlaneOmitAvailabilityZone</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>ControlPlaneOmitAvailabilityZone causes availability zone to be
omitted when creating control plane nodes, allowing the Nova
scheduler to make a decision on which availability zone to use based
on other scheduling constraints</p>
</td>
</tr>
<tr>
<td>
<code>bastion</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.Bastion">
Bastion
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Bastion is the OpenStack instance to login the nodes</p>
<p>As a rolling update is not ideal during a bastion host session, we
prevent changes to a running bastion configuration. To make changes, it&rsquo;s required
to first set <code>enabled: false</code> which will remove the bastion and then changes can be made.</p>
</td>
</tr>
<tr>
<td>
<code>identityRef</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackIdentityReference">
OpenStackIdentityReference
</a>
</em>
</td>
<td>
<p>IdentityRef is a reference to a secret holding OpenStack credentials
to be used when reconciling this cluster. It is also to reconcile
machines unless overridden in the machine spec.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterStatus">OpenStackClusterStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackCluster">OpenStackCluster</a>)
</p>
<p>
<p>OpenStackClusterStatus defines the observed state of OpenStackCluster.</p>
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
<p>Ready is true when the cluster infrastructure is ready.</p>
</td>
</tr>
<tr>
<td>
<code>network</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NetworkStatusWithSubnets">
NetworkStatusWithSubnets
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Network contains information about the created OpenStack Network.</p>
</td>
</tr>
<tr>
<td>
<code>externalNetwork</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NetworkStatus">
NetworkStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExternalNetwork contains information about the external network used for default ingress and egress traffic.</p>
</td>
</tr>
<tr>
<td>
<code>router</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.Router">
Router
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Router describes the default cluster router</p>
</td>
</tr>
<tr>
<td>
<code>apiServerLoadBalancer</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.LoadBalancer">
LoadBalancer
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>APIServerLoadBalancer describes the api server load balancer if one exists</p>
</td>
</tr>
<tr>
<td>
<code>failureDomains</code><br/>
<em>
<a href="https://doc.crds.dev/github.com/kubernetes-sigs/cluster-api@v1.6.0">
sigs.k8s.io/cluster-api/api/v1beta1.FailureDomains
</a>
</em>
</td>
<td>
<p>FailureDomains represent OpenStack availability zones</p>
</td>
</tr>
<tr>
<td>
<code>controlPlaneSecurityGroup</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupStatus">
SecurityGroupStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ControlPlaneSecurityGroup contains the information about the
OpenStack Security Group that needs to be applied to control plane
nodes.</p>
</td>
</tr>
<tr>
<td>
<code>workerSecurityGroup</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupStatus">
SecurityGroupStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>WorkerSecurityGroup contains the information about the OpenStack
Security Group that needs to be applied to worker nodes.</p>
</td>
</tr>
<tr>
<td>
<code>bastionSecurityGroup</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupStatus">
SecurityGroupStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>BastionSecurityGroup contains the information about the OpenStack
Security Group that needs to be applied to worker nodes.</p>
</td>
</tr>
<tr>
<td>
<code>bastion</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.BastionStatus">
BastionStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Bastion contains the information about the deployed bastion host</p>
</td>
</tr>
<tr>
<td>
<code>failureReason</code><br/>
<em>
sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors.DeprecatedCAPIClusterStatusError
</em>
</td>
<td>
<em>(Optional)</em>
<p>FailureReason will be set in the event that there is a terminal problem
reconciling the OpenStackCluster and will contain a succinct value suitable
for machine interpretation.</p>
<p>This field should not be set for transitive errors that a controller
faces that are expected to be fixed automatically over
time (like service outages), but instead indicate that something is
fundamentally wrong with the OpenStackCluster&rsquo;s spec or the configuration of
the controller, and that manual intervention is required. Examples
of terminal errors would be invalid combinations of settings in the
spec, values that are unsupported by the controller, or the
responsible controller itself being critically misconfigured.</p>
<p>Any transient errors that occur during the reconciliation of
OpenStackClusters can be added as events to the OpenStackCluster object
and/or logged in the controller&rsquo;s output.</p>
</td>
</tr>
<tr>
<td>
<code>failureMessage</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>FailureMessage will be set in the event that there is a terminal problem
reconciling the OpenStackCluster and will contain a more verbose string suitable
for logging and human consumption.</p>
<p>This field should not be set for transitive errors that a controller
faces that are expected to be fixed automatically over
time (like service outages), but instead indicate that something is
fundamentally wrong with the OpenStackCluster&rsquo;s spec or the configuration of
the controller, and that manual intervention is required. Examples
of terminal errors would be invalid combinations of settings in the
spec, values that are unsupported by the controller, or the
responsible controller itself being critically misconfigured.</p>
<p>Any transient errors that occur during the reconciliation of
OpenStackClusters can be added as events to the OpenStackCluster object
and/or logged in the controller&rsquo;s output.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterTemplateResource">OpenStackClusterTemplateResource
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterTemplateSpec">OpenStackClusterTemplateSpec</a>)
</p>
<p>
<p>OpenStackClusterTemplateResource describes the data needed to create a OpenStackCluster from a template.</p>
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
<code>spec</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterSpec">
OpenStackClusterSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>managedSubnets</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SubnetSpec">
[]SubnetSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ManagedSubnets describe OpenStack Subnets to be created. Cluster actuator will create a network,
subnets with the defined CIDR, and a router connected to these subnets. Currently only one IPv4
subnet is supported. If you leave this empty, no network will be created.</p>
</td>
</tr>
<tr>
<td>
<code>router</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.RouterParam">
RouterParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Router specifies an existing router to be used if ManagedSubnets are
specified. If specified, no new router will be created.</p>
</td>
</tr>
<tr>
<td>
<code>network</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NetworkParam">
NetworkParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Network specifies an existing network to use if no ManagedSubnets
are specified.</p>
</td>
</tr>
<tr>
<td>
<code>subnets</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SubnetParam">
[]SubnetParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Subnets specifies existing subnets to use if not ManagedSubnets are
specified. All subnets must be in the network specified by Network.
There can be zero, one, or two subnets. If no subnets are specified,
all subnets in Network will be used. If 2 subnets are specified, one
must be IPv4 and the other IPv6.</p>
</td>
</tr>
<tr>
<td>
<code>networkMTU</code><br/>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>NetworkMTU sets the maximum transmission unit (MTU) value to address fragmentation for the private network ID.
This value will be used only if the Cluster actuator creates the network.
If left empty, the network will have the default MTU defined in Openstack network service.
To use this field, the Openstack installation requires the net-mtu neutron API extension.</p>
</td>
</tr>
<tr>
<td>
<code>externalRouterIPs</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ExternalRouterIPParam">
[]ExternalRouterIPParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExternalRouterIPs is an array of externalIPs on the respective subnets.
This is necessary if the router needs a fixed ip in a specific subnet.</p>
</td>
</tr>
<tr>
<td>
<code>externalNetwork</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NetworkParam">
NetworkParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExternalNetwork is the OpenStack Network to be used to get public internet to the VMs.
This option is ignored if DisableExternalNetwork is set to true.</p>
<p>If ExternalNetwork is defined it must refer to exactly one external network.</p>
<p>If ExternalNetwork is not defined or is empty the controller will use any
existing external network as long as there is only one. It is an
error if ExternalNetwork is not defined and there are multiple
external networks unless DisableExternalNetwork is also set.</p>
<p>If ExternalNetwork is not defined and there are no external networks
the controller will proceed as though DisableExternalNetwork was set.</p>
</td>
</tr>
<tr>
<td>
<code>disableExternalNetwork</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisableExternalNetwork specifies whether or not to attempt to connect the cluster
to an external network. This allows for the creation of clusters when connecting
to an external network is not possible or desirable, e.g. if using a provider network.</p>
</td>
</tr>
<tr>
<td>
<code>apiServerLoadBalancer</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.APIServerLoadBalancer">
APIServerLoadBalancer
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>APIServerLoadBalancer configures the optional LoadBalancer for the APIServer.
If not specified, no load balancer will be created for the API server.</p>
</td>
</tr>
<tr>
<td>
<code>disableAPIServerFloatingIP</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisableAPIServerFloatingIP determines whether or not to attempt to attach a floating
IP to the API server. This allows for the creation of clusters when attaching a floating
IP to the API server (and hence, in many cases, exposing the API server to the internet)
is not possible or desirable, e.g. if using a shared VLAN for communication between
management and workload clusters or when the management cluster is inside the
project network.
This option requires that the API server use a VIP on the cluster network so that the
underlying machines can change without changing ControlPlaneEndpoint.Host.
When using a managed load balancer, this VIP will be managed automatically.
If not using a managed load balancer, cluster configuration will fail without additional
configuration to manage the VIP on the control plane machines, which falls outside of
the scope of this controller.</p>
</td>
</tr>
<tr>
<td>
<code>apiServerFloatingIP</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>APIServerFloatingIP is the floatingIP which will be associated with the API server.
The floatingIP will be created if it does not already exist.
If not specified, a new floatingIP is allocated.
This field is not used if DisableAPIServerFloatingIP is set to true.</p>
</td>
</tr>
<tr>
<td>
<code>apiServerFixedIP</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>APIServerFixedIP is the fixed IP which will be associated with the API server.
In the case where the API server has a floating IP but not a managed load balancer,
this field is not used.
If a managed load balancer is used and this field is not specified, a fixed IP will
be dynamically allocated for the load balancer.
If a managed load balancer is not used AND the API server floating IP is disabled,
this field MUST be specified and should correspond to a pre-allocated port that
holds the fixed IP to be used as a VIP.</p>
</td>
</tr>
<tr>
<td>
<code>apiServerPort</code><br/>
<em>
uint16
</em>
</td>
<td>
<em>(Optional)</em>
<p>APIServerPort is the port on which the listener on the APIServer
will be created. If specified, it must be an integer between 0 and 65535.</p>
</td>
</tr>
<tr>
<td>
<code>managedSecurityGroups</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ManagedSecurityGroups">
ManagedSecurityGroups
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ManagedSecurityGroups determines whether OpenStack security groups for the cluster
will be managed by the OpenStack provider or whether pre-existing security groups will
be specified as part of the configuration.
By default, the managed security groups have rules that allow the Kubelet, etcd, and the
Kubernetes API server to function correctly.
It&rsquo;s possible to add additional rules to the managed security groups.
When defined to an empty struct, the managed security groups will be created with the default rules.</p>
</td>
</tr>
<tr>
<td>
<code>disablePortSecurity</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisablePortSecurity disables the port security of the network created for the
Kubernetes cluster, which also disables SecurityGroups</p>
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
<em>(Optional)</em>
<p>Tags to set on all resources in cluster which support tags</p>
</td>
</tr>
<tr>
<td>
<code>controlPlaneEndpoint</code><br/>
<em>
<a href="https://doc.crds.dev/github.com/kubernetes-sigs/cluster-api@v1.6.0">
sigs.k8s.io/cluster-api/api/v1beta1.APIEndpoint
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
It is normally populated automatically by the OpenStackCluster
controller during cluster provisioning. If it is set on creation the
control plane endpoint will use the values set here in preference to
values set elsewhere.
ControlPlaneEndpoint cannot be modified after ControlPlaneEndpoint.Host has been set.</p>
</td>
</tr>
<tr>
<td>
<code>controlPlaneAvailabilityZones</code><br/>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ControlPlaneAvailabilityZones is the set of availability zones which
control plane machines may be deployed to.</p>
</td>
</tr>
<tr>
<td>
<code>controlPlaneOmitAvailabilityZone</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>ControlPlaneOmitAvailabilityZone causes availability zone to be
omitted when creating control plane nodes, allowing the Nova
scheduler to make a decision on which availability zone to use based
on other scheduling constraints</p>
</td>
</tr>
<tr>
<td>
<code>bastion</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.Bastion">
Bastion
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Bastion is the OpenStack instance to login the nodes</p>
<p>As a rolling update is not ideal during a bastion host session, we
prevent changes to a running bastion configuration. To make changes, it&rsquo;s required
to first set <code>enabled: false</code> which will remove the bastion and then changes can be made.</p>
</td>
</tr>
<tr>
<td>
<code>identityRef</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackIdentityReference">
OpenStackIdentityReference
</a>
</em>
</td>
<td>
<p>IdentityRef is a reference to a secret holding OpenStack credentials
to be used when reconciling this cluster. It is also to reconcile
machines unless overridden in the machine spec.</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterTemplateSpec">OpenStackClusterTemplateSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterTemplate">OpenStackClusterTemplate</a>)
</p>
<p>
<p>OpenStackClusterTemplateSpec defines the desired state of OpenStackClusterTemplate.</p>
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
<code>template</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterTemplateResource">
OpenStackClusterTemplateResource
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.OpenStackIdentityReference">OpenStackIdentityReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterSpec">OpenStackClusterSpec</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineSpec">OpenStackMachineSpec</a>)
</p>
<p>
<p>OpenStackIdentityReference is a reference to an infrastructure
provider identity to be used to provision cluster resources.</p>
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
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name is the name of a secret in the same namespace as the resource being provisioned.
The secret must contain a key named <code>clouds.yaml</code> which contains an OpenStack clouds.yaml file.
The secret may optionally contain a key named <code>cacert</code> containing a PEM-encoded CA certificate.</p>
</td>
</tr>
<tr>
<td>
<code>cloudName</code><br/>
<em>
string
</em>
</td>
<td>
<p>CloudName specifies the name of the entry in the clouds.yaml file to use.</p>
</td>
</tr>
<tr>
<td>
<code>region</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Region specifies an OpenStack region to use. If specified, it overrides
any value in clouds.yaml. If specified for an OpenStackMachine, its
value will be included in providerID.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineSpec">OpenStackMachineSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachine">OpenStackMachine</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.Bastion">Bastion</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineTemplateResource">OpenStackMachineTemplateResource</a>)
</p>
<p>
<p>OpenStackMachineSpec defines the desired state of OpenStackMachine.</p>
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
<code>providerID</code><br/>
<em>
string
</em>
</td>
<td>
<p>ProviderID is the unique identifier as specified by the cloud provider.</p>
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
<p>The flavor reference for the flavor for your server instance.</p>
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
<p>FlavorID allows flavors to be specified by ID.  This field takes precedence
over Flavor.</p>
</td>
</tr>
<tr>
<td>
<code>image</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ImageParam">
ImageParam
</a>
</em>
</td>
<td>
<p>The image to use for your server instance.
If the rootVolume is specified, this will be used when creating the root volume.</p>
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
<p>The ssh key to inject in the instance</p>
</td>
</tr>
<tr>
<td>
<code>ports</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.PortOpts">
[]PortOpts
</a>
</em>
</td>
<td>
<p>Ports to be attached to the server instance. They are created if a port with the given name does not already exist.
If not specified a default port will be added for the default cluster network.</p>
</td>
</tr>
<tr>
<td>
<code>securityGroups</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupParam">
[]SecurityGroupParam
</a>
</em>
</td>
<td>
<p>The names of the security groups to assign to the instance</p>
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
<p>Whether the server instance is created on a trunk port or not.</p>
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
<code>serverMetadata</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ServerMetadata">
[]ServerMetadata
</a>
</em>
</td>
<td>
<p>Metadata mapping. Allows you to create a map of key value pairs to add to the server instance.</p>
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
<p>Config Drive support</p>
</td>
</tr>
<tr>
<td>
<code>rootVolume</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.RootVolume">
RootVolume
</a>
</em>
</td>
<td>
<p>The volume metadata to boot from</p>
</td>
</tr>
<tr>
<td>
<code>additionalBlockDevices</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.AdditionalBlockDevice">
[]AdditionalBlockDevice
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AdditionalBlockDevices is a list of specifications for additional block devices to attach to the server instance</p>
</td>
</tr>
<tr>
<td>
<code>serverGroup</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ServerGroupParam">
ServerGroupParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The server group to assign the machine to.</p>
</td>
</tr>
<tr>
<td>
<code>identityRef</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackIdentityReference">
OpenStackIdentityReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IdentityRef is a reference to a secret holding OpenStack credentials
to be used when reconciling this machine. If not specified, the
credentials specified in the cluster will be used.</p>
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
<p>floatingIPPoolRef is a reference to a IPPool that will be assigned
to an IPAddressClaim. Once the IPAddressClaim is fulfilled, the FloatingIP
will be assigned to the OpenStackMachine.</p>
</td>
</tr>
<tr>
<td>
<code>schedulerHintAdditionalProperties</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SchedulerHintAdditionalProperty">
[]SchedulerHintAdditionalProperty
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
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineStatus">OpenStackMachineStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachine">OpenStackMachine</a>)
</p>
<p>
<p>OpenStackMachineStatus defines the observed state of OpenStackMachine.</p>
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
<em>(Optional)</em>
<p>Ready is true when the provider resource is ready.</p>
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
<p>InstanceID is the OpenStack instance ID for this machine.</p>
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
<p>Addresses contains the OpenStack instance associated addresses.</p>
</td>
</tr>
<tr>
<td>
<code>instanceState</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.InstanceState">
InstanceState
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>InstanceState is the state of the OpenStack instance for this machine.</p>
</td>
</tr>
<tr>
<td>
<code>resolved</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ResolvedMachineSpec">
ResolvedMachineSpec
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
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.MachineResources">
MachineResources
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
<code>failureReason</code><br/>
<em>
sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/errors.DeprecatedCAPIMachineStatusError
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>failureMessage</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>FailureMessage will be set in the event that there is a terminal problem
reconciling the Machine and will contain a more verbose string suitable
for logging and human consumption.</p>
<p>This field should not be set for transitive errors that a controller
faces that are expected to be fixed automatically over
time (like service outages), but instead indicate that something is
fundamentally wrong with the Machine&rsquo;s spec or the configuration of
the controller, and that manual intervention is required. Examples
of terminal errors would be invalid combinations of settings in the
spec, values that are unsupported by the controller, or the
responsible controller itself being critically misconfigured.</p>
<p>Any transient errors that occur during the reconciliation of Machines
can be added as events to the Machine object and/or logged in the
controller&rsquo;s output.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code><br/>
<em>
<a href="https://doc.crds.dev/github.com/kubernetes-sigs/cluster-api@v1.6.0">
sigs.k8s.io/cluster-api/api/v1beta1.Conditions
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineTemplateResource">OpenStackMachineTemplateResource
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineTemplateSpec">OpenStackMachineTemplateSpec</a>)
</p>
<p>
<p>OpenStackMachineTemplateResource describes the data needed to create a OpenStackMachine from a template.</p>
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
<code>spec</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineSpec">
OpenStackMachineSpec
</a>
</em>
</td>
<td>
<p>Spec is the specification of the desired behavior of the machine.</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>providerID</code><br/>
<em>
string
</em>
</td>
<td>
<p>ProviderID is the unique identifier as specified by the cloud provider.</p>
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
<p>The flavor reference for the flavor for your server instance.</p>
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
<p>FlavorID allows flavors to be specified by ID.  This field takes precedence
over Flavor.</p>
</td>
</tr>
<tr>
<td>
<code>image</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ImageParam">
ImageParam
</a>
</em>
</td>
<td>
<p>The image to use for your server instance.
If the rootVolume is specified, this will be used when creating the root volume.</p>
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
<p>The ssh key to inject in the instance</p>
</td>
</tr>
<tr>
<td>
<code>ports</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.PortOpts">
[]PortOpts
</a>
</em>
</td>
<td>
<p>Ports to be attached to the server instance. They are created if a port with the given name does not already exist.
If not specified a default port will be added for the default cluster network.</p>
</td>
</tr>
<tr>
<td>
<code>securityGroups</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupParam">
[]SecurityGroupParam
</a>
</em>
</td>
<td>
<p>The names of the security groups to assign to the instance</p>
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
<p>Whether the server instance is created on a trunk port or not.</p>
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
<code>serverMetadata</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ServerMetadata">
[]ServerMetadata
</a>
</em>
</td>
<td>
<p>Metadata mapping. Allows you to create a map of key value pairs to add to the server instance.</p>
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
<p>Config Drive support</p>
</td>
</tr>
<tr>
<td>
<code>rootVolume</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.RootVolume">
RootVolume
</a>
</em>
</td>
<td>
<p>The volume metadata to boot from</p>
</td>
</tr>
<tr>
<td>
<code>additionalBlockDevices</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.AdditionalBlockDevice">
[]AdditionalBlockDevice
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AdditionalBlockDevices is a list of specifications for additional block devices to attach to the server instance</p>
</td>
</tr>
<tr>
<td>
<code>serverGroup</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ServerGroupParam">
ServerGroupParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The server group to assign the machine to.</p>
</td>
</tr>
<tr>
<td>
<code>identityRef</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackIdentityReference">
OpenStackIdentityReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IdentityRef is a reference to a secret holding OpenStack credentials
to be used when reconciling this machine. If not specified, the
credentials specified in the cluster will be used.</p>
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
<p>floatingIPPoolRef is a reference to a IPPool that will be assigned
to an IPAddressClaim. Once the IPAddressClaim is fulfilled, the FloatingIP
will be assigned to the OpenStackMachine.</p>
</td>
</tr>
<tr>
<td>
<code>schedulerHintAdditionalProperties</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SchedulerHintAdditionalProperty">
[]SchedulerHintAdditionalProperty
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
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineTemplateSpec">OpenStackMachineTemplateSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineTemplate">OpenStackMachineTemplate</a>)
</p>
<p>
<p>OpenStackMachineTemplateSpec defines the desired state of OpenStackMachineTemplate.</p>
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
<code>template</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineTemplateResource">
OpenStackMachineTemplateResource
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.PortOpts">PortOpts
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineSpec">OpenStackMachineSpec</a>)
</p>
<p>
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
<code>network</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NetworkParam">
NetworkParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Network is a query for an openstack network that the port will be created or discovered on.
This will fail if the query returns more than one network.</p>
</td>
</tr>
<tr>
<td>
<code>description</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Description is a human-readable description for the port.</p>
</td>
</tr>
<tr>
<td>
<code>nameSuffix</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>NameSuffix will be appended to the name of the port if specified. If unspecified, instead the 0-based index of the port in the list is used.</p>
</td>
</tr>
<tr>
<td>
<code>fixedIPs</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.FixedIP">
[]FixedIP
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>FixedIPs is a list of pairs of subnet and/or IP address to assign to the port. If specified, these must be subnets of the port&rsquo;s network.</p>
</td>
</tr>
<tr>
<td>
<code>securityGroups</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupParam">
[]SecurityGroupParam
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecurityGroups is a list of the names, uuids, filters or any combination these of the security groups to assign to the instance.</p>
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
<em>(Optional)</em>
<p>Tags applied to the port (and corresponding trunk, if a trunk is configured.)
These tags are applied in addition to the instance&rsquo;s tags, which will also be applied to the port.</p>
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
<p>Trunk specifies whether trunking is enabled at the port level. If not
provided the value is inherited from the machine, or false for a
bastion host.</p>
</td>
</tr>
<tr>
<td>
<code>ResolvedPortSpecFields</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ResolvedPortSpecFields">
ResolvedPortSpecFields
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResolvedPortSpecFields</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.PortStatus">PortStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.MachineResources">MachineResources</a>)
</p>
<p>
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
<code>id</code><br/>
<em>
string
</em>
</td>
<td>
<p>ID is the unique identifier of the port.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.ResolvedFixedIP">ResolvedFixedIP
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ResolvedPortSpec">ResolvedPortSpec</a>)
</p>
<p>
<p>ResolvedFixedIP is a FixedIP with the Subnet resolved to an ID.</p>
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
<code>subnet</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>SubnetID is the id of a subnet to create the fixed IP of a port in.</p>
</td>
</tr>
<tr>
<td>
<code>ipAddress</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>IPAddress is a specific IP address to assign to the port. If SubnetID
is also specified, IPAddress must be a valid IP address in the
subnet. If Subnet is not specified, IPAddress must be a valid IP
address in any subnet of the port&rsquo;s network.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.ResolvedMachineSpec">ResolvedMachineSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.BastionStatus">BastionStatus</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineStatus">OpenStackMachineStatus</a>)
</p>
<p>
<p>ResolvedMachineSpec contains resolved references to resources required by the machine.</p>
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
<p>ServerGroupID is the ID of the server group the machine should be added to and is calculated based on ServerGroupFilter.</p>
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
<p>ImageID is the ID of the image to use for the machine and is calculated based on ImageFilter.</p>
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
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ResolvedPortSpec">
[]ResolvedPortSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Ports is the fully resolved list of ports to create for the machine.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.ResolvedPortSpec">ResolvedPortSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ResolvedMachineSpec">ResolvedMachineSpec</a>)
</p>
<p>
<p>ResolvedPortSpec is a PortOpts with all contained references fully resolved.</p>
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
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the port.</p>
</td>
</tr>
<tr>
<td>
<code>description</code><br/>
<em>
string
</em>
</td>
<td>
<p>Description is a human-readable description for the port.</p>
</td>
</tr>
<tr>
<td>
<code>networkID</code><br/>
<em>
string
</em>
</td>
<td>
<p>NetworkID is the ID of the network the port will be created in.</p>
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
<em>(Optional)</em>
<p>Tags applied to the port (and corresponding trunk, if a trunk is configured.)</p>
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
<p>Trunk specifies whether trunking is enabled at the port level.</p>
</td>
</tr>
<tr>
<td>
<code>fixedIPs</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ResolvedFixedIP">
[]ResolvedFixedIP
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>FixedIPs is a list of pairs of subnet and/or IP address to assign to the port. If specified, these must be subnets of the port&rsquo;s network.</p>
</td>
</tr>
<tr>
<td>
<code>securityGroups</code><br/>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecurityGroups is a list of security group IDs to assign to the port.</p>
</td>
</tr>
<tr>
<td>
<code>ResolvedPortSpecFields</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ResolvedPortSpecFields">
ResolvedPortSpecFields
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResolvedPortSpecFields</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.ResolvedPortSpecFields">ResolvedPortSpecFields
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.PortOpts">PortOpts</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ResolvedPortSpec">ResolvedPortSpec</a>)
</p>
<p>
<p>ResolvePortSpecFields is a convenience struct containing all fields of a
PortOpts which don&rsquo;t contain references which need to be resolved, and can
therefore be shared with ResolvedPortSpec.</p>
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
<code>adminStateUp</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>AdminStateUp specifies whether the port should be created in the up (true) or down (false) state. The default is up.</p>
</td>
</tr>
<tr>
<td>
<code>macAddress</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>MACAddress specifies the MAC address of the port. If not specified, the MAC address will be generated.</p>
</td>
</tr>
<tr>
<td>
<code>allowedAddressPairs</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.AddressPair">
[]AddressPair
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AllowedAddressPairs is a list of address pairs which Neutron will
allow the port to send traffic from in addition to the port&rsquo;s
addresses. If not specified, the MAC Address will be the MAC Address
of the port. Depending on the configuration of Neutron, it may be
supported to specify a CIDR instead of a specific IP address.</p>
</td>
</tr>
<tr>
<td>
<code>hostID</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>HostID specifies the ID of the host where the port resides.</p>
</td>
</tr>
<tr>
<td>
<code>vnicType</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>VNICType specifies the type of vNIC which this port should be
attached to. This is used to determine which mechanism driver(s) to
be used to bind the port. The valid values are normal, macvtap,
direct, baremetal, direct-physical, virtio-forwarder, smart-nic and
remote-managed, although these values will not be validated in this
API to ensure compatibility with future neutron changes or custom
implementations. What type of vNIC is actually available depends on
deployments. If not specified, the Neutron default value is used.</p>
</td>
</tr>
<tr>
<td>
<code>profile</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.BindingProfile">
BindingProfile
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Profile is a set of key-value pairs that are used for binding
details. We intentionally don&rsquo;t expose this as a map[string]string
because we only want to enable the users to set the values of the
keys that are known to work in OpenStack Networking API.  See
<a href="https://docs.openstack.org/api-ref/network/v2/index.html?expanded=create-port-detail#create-port">https://docs.openstack.org/api-ref/network/v2/index.html?expanded=create-port-detail#create-port</a>
To set profiles, your tenant needs permissions rule:create_port, and
rule:create_port:binding:profile</p>
</td>
</tr>
<tr>
<td>
<code>disablePortSecurity</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisablePortSecurity enables or disables the port security when set.
When not set, it takes the value of the corresponding field at the network level.</p>
</td>
</tr>
<tr>
<td>
<code>propagateUplinkStatus</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>PropageteUplinkStatus enables or disables the propagate uplink status on the port.</p>
</td>
</tr>
<tr>
<td>
<code>valueSpecs</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ValueSpec">
[]ValueSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Value specs are extra parameters to include in the API request with OpenStack.
This is an extension point for the API, so what they do and if they are supported,
depends on the specific OpenStack implementation.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.ResourceReference">ResourceReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ImageParam">ImageParam</a>)
</p>
<p>
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
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the referenced resource</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.RootVolume">RootVolume
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineSpec">OpenStackMachineSpec</a>)
</p>
<p>
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
<code>sizeGiB</code><br/>
<em>
int
</em>
</td>
<td>
<p>SizeGiB is the size of the block device in gibibytes (GiB).</p>
</td>
</tr>
<tr>
<td>
<code>BlockDeviceVolume</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.BlockDeviceVolume">
BlockDeviceVolume
</a>
</em>
</td>
<td>
<p>
(Members of <code>BlockDeviceVolume</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.Router">Router
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterStatus">OpenStackClusterStatus</a>)
</p>
<p>
<p>Router represents basic information about the associated OpenStack Neutron Router.</p>
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
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>id</code><br/>
<em>
string
</em>
</td>
<td>
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
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>ips</code><br/>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.RouterFilter">RouterFilter
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.RouterParam">RouterParam</a>)
</p>
<p>
<p>RouterFilter specifies a query to select an OpenStack router. At least one property must be set.</p>
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
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>description</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>projectID</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>FilterByNeutronTags</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.FilterByNeutronTags">
FilterByNeutronTags
</a>
</em>
</td>
<td>
<p>
(Members of <code>FilterByNeutronTags</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.RouterParam">RouterParam
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterSpec">OpenStackClusterSpec</a>)
</p>
<p>
<p>RouterParam specifies an OpenStack router to use. It may be specified by either ID or filter, but not both.</p>
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
<code>id</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ID is the ID of the router to use. If ID is provided, the other filters cannot be provided. Must be in UUID format.</p>
</td>
</tr>
<tr>
<td>
<code>filter</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.RouterFilter">
RouterFilter
</a>
</em>
</td>
<td>
<p>Filter specifies a filter to select an OpenStack router. If provided, cannot be empty.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.SchedulerHintAdditionalProperty">SchedulerHintAdditionalProperty
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineSpec">OpenStackMachineSpec</a>)
</p>
<p>
<p>SchedulerHintAdditionalProperty represents a single additional property for a scheduler hint.
It includes a Name to identify the property and a Value that can be of various types.</p>
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
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the scheduler hint property.
It is a unique identifier for the property.</p>
</td>
</tr>
<tr>
<td>
<code>value</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SchedulerHintAdditionalValue">
SchedulerHintAdditionalValue
</a>
</em>
</td>
<td>
<p>Value is the value of the scheduler hint property, which can be of various types
(e.g., bool, string, int). The type is indicated by the Value.Type field.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.SchedulerHintAdditionalValue">SchedulerHintAdditionalValue
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SchedulerHintAdditionalProperty">SchedulerHintAdditionalProperty</a>)
</p>
<p>
<p>SchedulerHintAdditionalValue represents the value of a scheduler hint property.
The value can be of various types: Bool, String, or Number.
The Type field indicates the type of the value being used.</p>
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
<code>type</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SchedulerHintValueType">
SchedulerHintValueType
</a>
</em>
</td>
<td>
<p>Type represents the type of the value.
Valid values are Bool, String, and Number.</p>
</td>
</tr>
<tr>
<td>
<code>bool</code><br/>
<em>
bool
</em>
</td>
<td>
<p>Bool is the boolean value of the scheduler hint, used when Type is &ldquo;Bool&rdquo;.
This field is required if type is &lsquo;Bool&rsquo;, and must not be set otherwise.</p>
</td>
</tr>
<tr>
<td>
<code>number</code><br/>
<em>
int
</em>
</td>
<td>
<p>Number is the integer value of the scheduler hint, used when Type is &ldquo;Number&rdquo;.
This field is required if type is &lsquo;Number&rsquo;, and must not be set otherwise.</p>
</td>
</tr>
<tr>
<td>
<code>string</code><br/>
<em>
string
</em>
</td>
<td>
<p>String is the string value of the scheduler hint, used when Type is &ldquo;String&rdquo;.
This field is required if type is &lsquo;String&rsquo;, and must not be set otherwise.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.SchedulerHintValueType">SchedulerHintValueType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SchedulerHintAdditionalValue">SchedulerHintAdditionalValue</a>)
</p>
<p>
<p>SchedulerHintValueType is the type that represents allowed values for the Type field.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Bool&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Number&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;String&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupFilter">SecurityGroupFilter
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupParam">SecurityGroupParam</a>)
</p>
<p>
<p>SecurityGroupFilter specifies a query to select an OpenStack security group. At least one property must be set.</p>
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
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>description</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>projectID</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>FilterByNeutronTags</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.FilterByNeutronTags">
FilterByNeutronTags
</a>
</em>
</td>
<td>
<p>
(Members of <code>FilterByNeutronTags</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupParam">SecurityGroupParam
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineSpec">OpenStackMachineSpec</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.PortOpts">PortOpts</a>)
</p>
<p>
<p>SecurityGroupParam specifies an OpenStack security group. It may be specified by ID or filter, but not both.</p>
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
<code>id</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ID is the ID of the security group to use. If ID is provided, the other filters cannot be provided. Must be in UUID format.</p>
</td>
</tr>
<tr>
<td>
<code>filter</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupFilter">
SecurityGroupFilter
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Filter specifies a query to select an OpenStack security group. If provided, cannot be empty.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupRuleSpec">SecurityGroupRuleSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ManagedSecurityGroups">ManagedSecurityGroups</a>)
</p>
<p>
<p>SecurityGroupRuleSpec represent the basic information of the associated OpenStack
Security Group Role.
For now this is only used for the allNodesSecurityGroupRules but when we add
other security groups, we&rsquo;ll need to add a validation because
Remote* fields are mutually exclusive.</p>
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
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>name of the security group rule.
It&rsquo;s used to identify the rule so it can be patched and will not be sent to the OpenStack API.</p>
</td>
</tr>
<tr>
<td>
<code>description</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>description of the security group rule.</p>
</td>
</tr>
<tr>
<td>
<code>direction</code><br/>
<em>
string
</em>
</td>
<td>
<p>direction in which the security group rule is applied. The only values
allowed are &ldquo;ingress&rdquo; or &ldquo;egress&rdquo;. For a compute instance, an ingress
security group rule is applied to incoming (ingress) traffic for that
instance. An egress rule is applied to traffic leaving the instance.</p>
</td>
</tr>
<tr>
<td>
<code>etherType</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>etherType must be IPv4 or IPv6, and addresses represented in CIDR must match the
ingress or egress rules.</p>
</td>
</tr>
<tr>
<td>
<code>portRangeMin</code><br/>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>portRangeMin is a number in the range that is matched by the security group
rule. If the protocol is TCP or UDP, this value must be less than or equal
to the value of the portRangeMax attribute.</p>
</td>
</tr>
<tr>
<td>
<code>portRangeMax</code><br/>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>portRangeMax is a number in the range that is matched by the security group
rule. The portRangeMin attribute constrains the portRangeMax attribute.</p>
</td>
</tr>
<tr>
<td>
<code>protocol</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>protocol is the protocol that is matched by the security group rule.</p>
</td>
</tr>
<tr>
<td>
<code>remoteGroupID</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>remoteGroupID is the remote group ID to be associated with this security group rule.
You can specify either remoteGroupID or remoteIPPrefix or remoteManagedGroups.</p>
</td>
</tr>
<tr>
<td>
<code>remoteIPPrefix</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>remoteIPPrefix is the remote IP prefix to be associated with this security group rule.
You can specify either remoteGroupID or remoteIPPrefix or remoteManagedGroups.</p>
</td>
</tr>
<tr>
<td>
<code>remoteManagedGroups</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ManagedSecurityGroupName">
[]ManagedSecurityGroupName
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>remoteManagedGroups is the remote managed groups to be associated with this security group rule.
You can specify either remoteGroupID or remoteIPPrefix or remoteManagedGroups.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.SecurityGroupStatus">SecurityGroupStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterStatus">OpenStackClusterStatus</a>)
</p>
<p>
<p>SecurityGroupStatus represents the basic information of the associated
OpenStack Neutron Security Group.</p>
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
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>name of the security group</p>
</td>
</tr>
<tr>
<td>
<code>id</code><br/>
<em>
string
</em>
</td>
<td>
<p>id of the security group</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.ServerGroupFilter">ServerGroupFilter
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ServerGroupParam">ServerGroupParam</a>)
</p>
<p>
<p>ServerGroupFilter specifies a query to select an OpenStack server group. At least one property must be set.</p>
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
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name is the name of a server group to look for.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.ServerGroupParam">ServerGroupParam
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineSpec">OpenStackMachineSpec</a>)
</p>
<p>
<p>ServerGroupParam specifies an OpenStack server group. It may be specified by ID or filter, but not both.</p>
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
<code>id</code><br/>
<em>
string
</em>
</td>
<td>
<p>ID is the ID of the server group to use.</p>
</td>
</tr>
<tr>
<td>
<code>filter</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ServerGroupFilter">
ServerGroupFilter
</a>
</em>
</td>
<td>
<p>Filter specifies a query to select an OpenStack server group. If provided, it cannot be empty.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.ServerMetadata">ServerMetadata
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackMachineSpec">OpenStackMachineSpec</a>)
</p>
<p>
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
<code>key</code><br/>
<em>
string
</em>
</td>
<td>
<p>Key is the server metadata key</p>
</td>
</tr>
<tr>
<td>
<code>value</code><br/>
<em>
string
</em>
</td>
<td>
<p>Value is the server metadata value</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.Subnet">Subnet
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.NetworkStatusWithSubnets">NetworkStatusWithSubnets</a>)
</p>
<p>
<p>Subnet represents basic information about the associated OpenStack Neutron Subnet.</p>
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
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>id</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>cidr</code><br/>
<em>
string
</em>
</td>
<td>
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
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.SubnetFilter">SubnetFilter
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SubnetParam">SubnetParam</a>)
</p>
<p>
<p>SubnetFilter specifies a filter to select a subnet. At least one parameter must be specified.</p>
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
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>description</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>projectID</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ipVersion</code><br/>
<em>
int
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>gatewayIP</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>cidr</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ipv6AddressMode</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ipv6RAMode</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>FilterByNeutronTags</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.FilterByNeutronTags">
FilterByNeutronTags
</a>
</em>
</td>
<td>
<p>
(Members of <code>FilterByNeutronTags</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.SubnetParam">SubnetParam
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.APIServerLoadBalancer">APIServerLoadBalancer</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ExternalRouterIPParam">ExternalRouterIPParam</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.FixedIP">FixedIP</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterSpec">OpenStackClusterSpec</a>)
</p>
<p>
<p>SubnetParam specifies an OpenStack subnet to use. It may be specified by either ID or filter, but not both.</p>
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
<code>id</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ID is the uuid of the subnet. It will not be validated.</p>
</td>
</tr>
<tr>
<td>
<code>filter</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.SubnetFilter">
SubnetFilter
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Filter specifies a filter to select the subnet. It must match exactly one subnet.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.SubnetSpec">SubnetSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.OpenStackClusterSpec">OpenStackClusterSpec</a>)
</p>
<p>
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
<code>cidr</code><br/>
<em>
string
</em>
</td>
<td>
<p>CIDR is representing the IP address range used to create the subnet, e.g. 10.0.0.0/24.
This field is required when defining a subnet.</p>
</td>
</tr>
<tr>
<td>
<code>dnsNameservers</code><br/>
<em>
[]string
</em>
</td>
<td>
<p>DNSNameservers holds a list of DNS server addresses that will be provided when creating
the subnet. These addresses need to have the same IP version as CIDR.</p>
</td>
</tr>
<tr>
<td>
<code>allocationPools</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.AllocationPool">
[]AllocationPool
</a>
</em>
</td>
<td>
<p>AllocationPools is an array of AllocationPool objects that will be applied to OpenStack Subnet being created.
If set, OpenStack will only allocate these IPs for Machines. It will still be possible to create ports from
outside of these ranges manually.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.ValueSpec">ValueSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.ResolvedPortSpecFields">ResolvedPortSpecFields</a>)
</p>
<p>
<p>ValueSpec represents a single value_spec key-value pair.</p>
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
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the key-value pair.
This is just for identifying the pair and will not be sent to the OpenStack API.</p>
</td>
</tr>
<tr>
<td>
<code>key</code><br/>
<em>
string
</em>
</td>
<td>
<p>Key is the key in the key-value pair.</p>
</td>
</tr>
<tr>
<td>
<code>value</code><br/>
<em>
string
</em>
</td>
<td>
<p>Value is the value in the key-value pair.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.VolumeAZName">VolumeAZName
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.VolumeAvailabilityZone">VolumeAvailabilityZone</a>)
</p>
<p>
<p>VolumeAZName is the name of a volume availability zone. It may not contain spaces.</p>
</p>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.VolumeAZSource">VolumeAZSource
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.VolumeAvailabilityZone">VolumeAvailabilityZone</a>)
</p>
<p>
<p>VolumeAZSource specifies where to obtain the availability zone for a volume.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Machine&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Name&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1beta1.VolumeAvailabilityZone">VolumeAvailabilityZone
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.BlockDeviceVolume">BlockDeviceVolume</a>)
</p>
<p>
<p>VolumeAvailabilityZone specifies the availability zone for a volume.</p>
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
<code>from</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.VolumeAZSource">
VolumeAZSource
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>From specifies where we will obtain the availability zone for the
volume. The options are &ldquo;Name&rdquo; and &ldquo;Machine&rdquo;. If &ldquo;Name&rdquo; is specified
then the Name field must also be specified. If &ldquo;Machine&rdquo; is specified
the volume will use the value of FailureDomain, if any, from the
associated Machine.</p>
</td>
</tr>
<tr>
<td>
<code>name</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1beta1.VolumeAZName">
VolumeAZName
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Name is the name of a volume availability zone to use. It is required
if From is &ldquo;Name&rdquo;. The volume availability zone name may not contain
spaces.</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>.
</em></p>
