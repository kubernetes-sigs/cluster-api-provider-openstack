<h2 id="infrastructure.cluster.x-k8s.io/v1alpha6">infrastructure.cluster.x-k8s.io/v1alpha6</h2>
<p>
<p>Package v1alpha6 contains API Schema definitions for the infrastructure v1alpha6 API group.</p>
</p>
Resource Types:
<ul><li>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackCluster">OpenStackCluster</a>
</li><li>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterTemplate">OpenStackClusterTemplate</a>
</li><li>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachine">OpenStackMachine</a>
</li><li>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineTemplate">OpenStackMachineTemplate</a>
</li></ul>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackCluster">OpenStackCluster
</h3>
<p>
<p>OpenStackCluster is the Schema for the openstackclusters API.</p>
<p>Deprecated: This type will be removed in one of the next releases.</p>
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
infrastructure.cluster.x-k8s.io/v1alpha6
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterSpec">
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
<code>cloudName</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The name of the cloud to use from the clouds secret</p>
</td>
</tr>
<tr>
<td>
<code>nodeCidr</code><br/>
<em>
string
</em>
</td>
<td>
<p>NodeCIDR is the OpenStack Subnet to be created. Cluster actuator will create a
network, a subnet with NodeCIDR, and a router connected to this subnet.
If you leave this empty, no network will be created.</p>
</td>
</tr>
<tr>
<td>
<code>network</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.NetworkFilter">
NetworkFilter
</a>
</em>
</td>
<td>
<p>If NodeCIDR cannot be set this can be used to detect an existing network.</p>
</td>
</tr>
<tr>
<td>
<code>subnet</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.SubnetFilter">
SubnetFilter
</a>
</em>
</td>
<td>
<p>If NodeCIDR cannot be set this can be used to detect an existing subnet.</p>
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
<p>DNSNameservers is the list of nameservers for OpenStack Subnet being created.
Set this value when you need create a new network/subnet while the access
through DNS is required.</p>
</td>
</tr>
<tr>
<td>
<code>externalRouterIPs</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.ExternalRouterIPParam">
[]ExternalRouterIPParam
</a>
</em>
</td>
<td>
<p>ExternalRouterIPs is an array of externalIPs on the respective subnets.
This is necessary if the router needs a fixed ip in a specific subnet.</p>
</td>
</tr>
<tr>
<td>
<code>externalNetworkId</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExternalNetworkID is the ID of an external OpenStack Network. This is necessary
to get public internet to the VMs.</p>
</td>
</tr>
<tr>
<td>
<code>apiServerLoadBalancer</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.APIServerLoadBalancer">
APIServerLoadBalancer
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>APIServerLoadBalancer configures the optional LoadBalancer for the APIServer.
It must be activated by setting <code>enabled: true</code>.</p>
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
int
</em>
</td>
<td>
<p>APIServerPort is the port on which the listener on the APIServer
will be created</p>
</td>
</tr>
<tr>
<td>
<code>managedSecurityGroups</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>ManagedSecurityGroups determines whether OpenStack security groups for the cluster
will be managed by the OpenStack provider or whether pre-existing security groups will
be specified as part of the configuration.
By default, the managed security groups have rules that allow the Kubelet, etcd, the
Kubernetes API server and the Calico CNI plugin to function correctly.</p>
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
<em>(Optional)</em>
<p>AllowAllInClusterTraffic is only used when managed security groups are in use.
If set to true, the rules for the managed security groups are configured so that all
ingress and egress between cluster nodes is permitted, allowing CNIs other than
Calico to be used.</p>
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
<p>Tags for all resources in cluster</p>
</td>
</tr>
<tr>
<td>
<code>controlPlaneEndpoint</code><br/>
<em>
<a href="https://doc.crds.dev/github.com/kubernetes-sigs/cluster-api@v1.5.1">
sigs.k8s.io/cluster-api/api/v1beta1.APIEndpoint
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.</p>
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
<p>ControlPlaneAvailabilityZones is the az to deploy control plane to</p>
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
<p>Indicates whether to omit the az for control plane nodes, allowing the Nova scheduler
to make a decision on which az to use based on other scheduling constraints</p>
</td>
</tr>
<tr>
<td>
<code>bastion</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.Bastion">
Bastion
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Bastion is the OpenStack instance to login the nodes</p>
<p>As a rolling update is not ideal during a bastion host session, we
prevent changes to a running bastion configuration. Set <code>enabled: false</code> to
make changes.</p>
</td>
</tr>
<tr>
<td>
<code>identityRef</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackIdentityReference">
OpenStackIdentityReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IdentityRef is a reference to a identity to be used when reconciling this cluster</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterStatus">
OpenStackClusterStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterTemplate">OpenStackClusterTemplate
</h3>
<p>
<p>OpenStackClusterTemplate is the Schema for the openstackclustertemplates API.</p>
<p>Deprecated: This type will be removed in one of the next releases.</p>
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
infrastructure.cluster.x-k8s.io/v1alpha6
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterTemplateSpec">
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterTemplateResource">
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
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachine">OpenStackMachine
</h3>
<p>
<p>OpenStackMachine is the Schema for the openstackmachines API.</p>
<p>Deprecated: This type will be removed in one of the next releases.</p>
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
infrastructure.cluster.x-k8s.io/v1alpha6
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineSpec">
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
<code>instanceID</code><br/>
<em>
string
</em>
</td>
<td>
<p>InstanceID is the OpenStack instance ID for this machine.</p>
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
<em>(Optional)</em>
<p>The name of the cloud to use from the clouds secret</p>
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
<code>image</code><br/>
<em>
string
</em>
</td>
<td>
<p>The name of the image to use for your server instance.
If the RootVolume is specified, this will be ignored and use rootVolume directly.</p>
</td>
</tr>
<tr>
<td>
<code>imageUUID</code><br/>
<em>
string
</em>
</td>
<td>
<p>The uuid of the image to use for your server instance.
if it&rsquo;s empty, Image name will be used</p>
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
<code>networks</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.NetworkParam">
[]NetworkParam
</a>
</em>
</td>
<td>
<p>A networks object. Required parameter when there are multiple networks defined for the tenant.
When you do not specify both networks and ports parameters, the server attaches to the only network created for the current tenant.</p>
</td>
</tr>
<tr>
<td>
<code>ports</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.PortOpts">
[]PortOpts
</a>
</em>
</td>
<td>
<p>Ports to be attached to the server instance. They are created if a port with the given name does not already exist.
When you do not specify both networks and ports parameters, the server attaches to the only network created for the current tenant.</p>
</td>
</tr>
<tr>
<td>
<code>subnet</code><br/>
<em>
string
</em>
</td>
<td>
<p>UUID, IP address of a port from this subnet will be marked as AccessIPv4 on the created compute instance</p>
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
<p>The floatingIP which will be associated to the machine, only used for master.
The floatingIP should have been created and haven&rsquo;t been associated.</p>
</td>
</tr>
<tr>
<td>
<code>securityGroups</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.SecurityGroupParam">
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
<p>Machine tags
Requires Nova api 2.52 minimum!</p>
</td>
</tr>
<tr>
<td>
<code>serverMetadata</code><br/>
<em>
map[string]string
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.RootVolume">
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
<code>serverGroupID</code><br/>
<em>
string
</em>
</td>
<td>
<p>The server group to assign the machine to</p>
</td>
</tr>
<tr>
<td>
<code>identityRef</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackIdentityReference">
OpenStackIdentityReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IdentityRef is a reference to a identity to be used when reconciling this cluster</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineStatus">
OpenStackMachineStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineTemplate">OpenStackMachineTemplate
</h3>
<p>
<p>OpenStackMachineTemplate is the Schema for the openstackmachinetemplates API.</p>
<p>Deprecated: This type will be removed in one of the next releases.</p>
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
infrastructure.cluster.x-k8s.io/v1alpha6
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineTemplateSpec">
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineTemplateResource">
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
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.APIServerLoadBalancer">APIServerLoadBalancer
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterSpec">OpenStackClusterSpec</a>)
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
<p>Enabled defines whether a load balancer should be created.</p>
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
<p>AdditionalPorts adds additional tcp ports to the load balancer.</p>
</td>
</tr>
<tr>
<td>
<code>allowedCidrs</code><br/>
<em>
[]string
</em>
</td>
<td>
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
<p>Octavia Provider Used to create load balancer</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.AddressPair">AddressPair
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.PortOpts">PortOpts</a>)
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
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.Bastion">Bastion
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterSpec">OpenStackClusterSpec</a>)
</p>
<p>
<p>Bastion represents basic information about the bastion node.</p>
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
</td>
</tr>
<tr>
<td>
<code>instance</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineSpec">
OpenStackMachineSpec
</a>
</em>
</td>
<td>
<p>Instance for the bastion itself</p>
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
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.ExternalRouterIPParam">ExternalRouterIPParam
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterSpec">OpenStackClusterSpec</a>)
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.SubnetParam">
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
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.FixedIP">FixedIP
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.PortOpts">PortOpts</a>)
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.SubnetFilter">
SubnetFilter
</a>
</em>
</td>
<td>
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
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.Instance">Instance
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterStatus">OpenStackClusterStatus</a>)
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
<code>trunk</code><br/>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>failureDomain</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>securigyGroups</code><br/>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>networks</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.Network">
[]Network
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>subnet</code><br/>
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
</td>
</tr>
<tr>
<td>
<code>image</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>imageUUID</code><br/>
<em>
string
</em>
</td>
<td>
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
<code>userData</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
map[string]string
</em>
</td>
<td>
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
</td>
</tr>
<tr>
<td>
<code>rootVolume</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.RootVolume">
RootVolume
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>serverGroupID</code><br/>
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.InstanceState">
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
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.InstanceState">InstanceState
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.Instance">Instance</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineStatus">OpenStackMachineStatus</a>)
</p>
<p>
<p>InstanceState describes the state of an OpenStack instance.</p>
</p>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.LoadBalancer">LoadBalancer
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.Network">Network</a>)
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
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.Network">Network
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.Instance">Instance</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterStatus">OpenStackClusterStatus</a>)
</p>
<p>
<p>Network represents basic information about an OpenStack Neutron Network associated with an instance&rsquo;s port.</p>
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
<code>subnet</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.Subnet">
Subnet
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>port</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.PortOpts">
PortOpts
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>router</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.Router">
Router
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>apiServerLoadBalancer</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.LoadBalancer">
LoadBalancer
</a>
</em>
</td>
<td>
<p>Be careful when using APIServerLoadBalancer, because this field is optional and therefore not
set in all cases</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.NetworkFilter">NetworkFilter
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.NetworkParam">NetworkParam</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterSpec">OpenStackClusterSpec</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.PortOpts">PortOpts</a>)
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
<code>projectId</code><br/>
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
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>tagsAny</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>notTags</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>notTagsAny</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.NetworkParam">NetworkParam
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineSpec">OpenStackMachineSpec</a>)
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
<code>uuid</code><br/>
<em>
string
</em>
</td>
<td>
<p>Optional UUID of the network.
If specified this will not be validated prior to server creation.
Required if <code>Subnets</code> specifies a subnet by UUID.</p>
</td>
</tr>
<tr>
<td>
<code>fixedIP</code><br/>
<em>
string
</em>
</td>
<td>
<p>A fixed IPv4 address for the NIC.</p>
</td>
</tr>
<tr>
<td>
<code>filter</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.NetworkFilter">
NetworkFilter
</a>
</em>
</td>
<td>
<p>Filters for optional network query</p>
</td>
</tr>
<tr>
<td>
<code>subnets</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.SubnetParam">
[]SubnetParam
</a>
</em>
</td>
<td>
<p>Subnet within a network to use</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterSpec">OpenStackClusterSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackCluster">OpenStackCluster</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterTemplateResource">OpenStackClusterTemplateResource</a>)
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
<code>cloudName</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The name of the cloud to use from the clouds secret</p>
</td>
</tr>
<tr>
<td>
<code>nodeCidr</code><br/>
<em>
string
</em>
</td>
<td>
<p>NodeCIDR is the OpenStack Subnet to be created. Cluster actuator will create a
network, a subnet with NodeCIDR, and a router connected to this subnet.
If you leave this empty, no network will be created.</p>
</td>
</tr>
<tr>
<td>
<code>network</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.NetworkFilter">
NetworkFilter
</a>
</em>
</td>
<td>
<p>If NodeCIDR cannot be set this can be used to detect an existing network.</p>
</td>
</tr>
<tr>
<td>
<code>subnet</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.SubnetFilter">
SubnetFilter
</a>
</em>
</td>
<td>
<p>If NodeCIDR cannot be set this can be used to detect an existing subnet.</p>
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
<p>DNSNameservers is the list of nameservers for OpenStack Subnet being created.
Set this value when you need create a new network/subnet while the access
through DNS is required.</p>
</td>
</tr>
<tr>
<td>
<code>externalRouterIPs</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.ExternalRouterIPParam">
[]ExternalRouterIPParam
</a>
</em>
</td>
<td>
<p>ExternalRouterIPs is an array of externalIPs on the respective subnets.
This is necessary if the router needs a fixed ip in a specific subnet.</p>
</td>
</tr>
<tr>
<td>
<code>externalNetworkId</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExternalNetworkID is the ID of an external OpenStack Network. This is necessary
to get public internet to the VMs.</p>
</td>
</tr>
<tr>
<td>
<code>apiServerLoadBalancer</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.APIServerLoadBalancer">
APIServerLoadBalancer
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>APIServerLoadBalancer configures the optional LoadBalancer for the APIServer.
It must be activated by setting <code>enabled: true</code>.</p>
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
int
</em>
</td>
<td>
<p>APIServerPort is the port on which the listener on the APIServer
will be created</p>
</td>
</tr>
<tr>
<td>
<code>managedSecurityGroups</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>ManagedSecurityGroups determines whether OpenStack security groups for the cluster
will be managed by the OpenStack provider or whether pre-existing security groups will
be specified as part of the configuration.
By default, the managed security groups have rules that allow the Kubelet, etcd, the
Kubernetes API server and the Calico CNI plugin to function correctly.</p>
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
<em>(Optional)</em>
<p>AllowAllInClusterTraffic is only used when managed security groups are in use.
If set to true, the rules for the managed security groups are configured so that all
ingress and egress between cluster nodes is permitted, allowing CNIs other than
Calico to be used.</p>
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
<p>Tags for all resources in cluster</p>
</td>
</tr>
<tr>
<td>
<code>controlPlaneEndpoint</code><br/>
<em>
<a href="https://doc.crds.dev/github.com/kubernetes-sigs/cluster-api@v1.5.1">
sigs.k8s.io/cluster-api/api/v1beta1.APIEndpoint
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.</p>
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
<p>ControlPlaneAvailabilityZones is the az to deploy control plane to</p>
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
<p>Indicates whether to omit the az for control plane nodes, allowing the Nova scheduler
to make a decision on which az to use based on other scheduling constraints</p>
</td>
</tr>
<tr>
<td>
<code>bastion</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.Bastion">
Bastion
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Bastion is the OpenStack instance to login the nodes</p>
<p>As a rolling update is not ideal during a bastion host session, we
prevent changes to a running bastion configuration. Set <code>enabled: false</code> to
make changes.</p>
</td>
</tr>
<tr>
<td>
<code>identityRef</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackIdentityReference">
OpenStackIdentityReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IdentityRef is a reference to a identity to be used when reconciling this cluster</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterStatus">OpenStackClusterStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackCluster">OpenStackCluster</a>)
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
</td>
</tr>
<tr>
<td>
<code>network</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.Network">
Network
</a>
</em>
</td>
<td>
<p>Network contains all information about the created OpenStack Network.
It includes Subnets and Router.</p>
</td>
</tr>
<tr>
<td>
<code>externalNetwork</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.Network">
Network
</a>
</em>
</td>
<td>
<p>External Network contains information about the created OpenStack external network.</p>
</td>
</tr>
<tr>
<td>
<code>failureDomains</code><br/>
<em>
<a href="https://doc.crds.dev/github.com/kubernetes-sigs/cluster-api@v1.5.1">
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.SecurityGroup">
SecurityGroup
</a>
</em>
</td>
<td>
<p>ControlPlaneSecurityGroups contains all the information about the OpenStack
Security Group that needs to be applied to control plane nodes.
TODO: Maybe instead of two properties, we add a property to the group?</p>
</td>
</tr>
<tr>
<td>
<code>workerSecurityGroup</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.SecurityGroup">
SecurityGroup
</a>
</em>
</td>
<td>
<p>WorkerSecurityGroup contains all the information about the OpenStack Security
Group that needs to be applied to worker nodes.</p>
</td>
</tr>
<tr>
<td>
<code>bastionSecurityGroup</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.SecurityGroup">
SecurityGroup
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>bastion</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.Instance">
Instance
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>failureReason</code><br/>
<em>
<a href="https://pkg.go.dev/sigs.k8s.io/cluster-api@v1.5.1/errors#ClusterStatusError">
sigs.k8s.io/cluster-api/errors.ClusterStatusError
</a>
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
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterTemplateResource">OpenStackClusterTemplateResource
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterTemplateSpec">OpenStackClusterTemplateSpec</a>)
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterSpec">
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
<code>cloudName</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The name of the cloud to use from the clouds secret</p>
</td>
</tr>
<tr>
<td>
<code>nodeCidr</code><br/>
<em>
string
</em>
</td>
<td>
<p>NodeCIDR is the OpenStack Subnet to be created. Cluster actuator will create a
network, a subnet with NodeCIDR, and a router connected to this subnet.
If you leave this empty, no network will be created.</p>
</td>
</tr>
<tr>
<td>
<code>network</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.NetworkFilter">
NetworkFilter
</a>
</em>
</td>
<td>
<p>If NodeCIDR cannot be set this can be used to detect an existing network.</p>
</td>
</tr>
<tr>
<td>
<code>subnet</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.SubnetFilter">
SubnetFilter
</a>
</em>
</td>
<td>
<p>If NodeCIDR cannot be set this can be used to detect an existing subnet.</p>
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
<p>DNSNameservers is the list of nameservers for OpenStack Subnet being created.
Set this value when you need create a new network/subnet while the access
through DNS is required.</p>
</td>
</tr>
<tr>
<td>
<code>externalRouterIPs</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.ExternalRouterIPParam">
[]ExternalRouterIPParam
</a>
</em>
</td>
<td>
<p>ExternalRouterIPs is an array of externalIPs on the respective subnets.
This is necessary if the router needs a fixed ip in a specific subnet.</p>
</td>
</tr>
<tr>
<td>
<code>externalNetworkId</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExternalNetworkID is the ID of an external OpenStack Network. This is necessary
to get public internet to the VMs.</p>
</td>
</tr>
<tr>
<td>
<code>apiServerLoadBalancer</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.APIServerLoadBalancer">
APIServerLoadBalancer
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>APIServerLoadBalancer configures the optional LoadBalancer for the APIServer.
It must be activated by setting <code>enabled: true</code>.</p>
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
int
</em>
</td>
<td>
<p>APIServerPort is the port on which the listener on the APIServer
will be created</p>
</td>
</tr>
<tr>
<td>
<code>managedSecurityGroups</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>ManagedSecurityGroups determines whether OpenStack security groups for the cluster
will be managed by the OpenStack provider or whether pre-existing security groups will
be specified as part of the configuration.
By default, the managed security groups have rules that allow the Kubelet, etcd, the
Kubernetes API server and the Calico CNI plugin to function correctly.</p>
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
<em>(Optional)</em>
<p>AllowAllInClusterTraffic is only used when managed security groups are in use.
If set to true, the rules for the managed security groups are configured so that all
ingress and egress between cluster nodes is permitted, allowing CNIs other than
Calico to be used.</p>
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
<p>Tags for all resources in cluster</p>
</td>
</tr>
<tr>
<td>
<code>controlPlaneEndpoint</code><br/>
<em>
<a href="https://doc.crds.dev/github.com/kubernetes-sigs/cluster-api@v1.5.1">
sigs.k8s.io/cluster-api/api/v1beta1.APIEndpoint
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.</p>
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
<p>ControlPlaneAvailabilityZones is the az to deploy control plane to</p>
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
<p>Indicates whether to omit the az for control plane nodes, allowing the Nova scheduler
to make a decision on which az to use based on other scheduling constraints</p>
</td>
</tr>
<tr>
<td>
<code>bastion</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.Bastion">
Bastion
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Bastion is the OpenStack instance to login the nodes</p>
<p>As a rolling update is not ideal during a bastion host session, we
prevent changes to a running bastion configuration. Set <code>enabled: false</code> to
make changes.</p>
</td>
</tr>
<tr>
<td>
<code>identityRef</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackIdentityReference">
OpenStackIdentityReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IdentityRef is a reference to a identity to be used when reconciling this cluster</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterTemplateSpec">OpenStackClusterTemplateSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterTemplate">OpenStackClusterTemplate</a>)
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterTemplateResource">
OpenStackClusterTemplateResource
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackIdentityReference">OpenStackIdentityReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterSpec">OpenStackClusterSpec</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineSpec">OpenStackMachineSpec</a>)
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
<code>kind</code><br/>
<em>
string
</em>
</td>
<td>
<p>Kind of the identity. Must be supported by the infrastructure
provider and may be either cluster or namespace-scoped.</p>
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
<p>Name of the infrastructure identity to be used.
Must be either a cluster-scoped resource, or namespaced-scoped
resource the same namespace as the resource(s) being provisioned.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineSpec">OpenStackMachineSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachine">OpenStackMachine</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.Bastion">Bastion</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineTemplateResource">OpenStackMachineTemplateResource</a>)
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
<code>instanceID</code><br/>
<em>
string
</em>
</td>
<td>
<p>InstanceID is the OpenStack instance ID for this machine.</p>
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
<em>(Optional)</em>
<p>The name of the cloud to use from the clouds secret</p>
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
<code>image</code><br/>
<em>
string
</em>
</td>
<td>
<p>The name of the image to use for your server instance.
If the RootVolume is specified, this will be ignored and use rootVolume directly.</p>
</td>
</tr>
<tr>
<td>
<code>imageUUID</code><br/>
<em>
string
</em>
</td>
<td>
<p>The uuid of the image to use for your server instance.
if it&rsquo;s empty, Image name will be used</p>
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
<code>networks</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.NetworkParam">
[]NetworkParam
</a>
</em>
</td>
<td>
<p>A networks object. Required parameter when there are multiple networks defined for the tenant.
When you do not specify both networks and ports parameters, the server attaches to the only network created for the current tenant.</p>
</td>
</tr>
<tr>
<td>
<code>ports</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.PortOpts">
[]PortOpts
</a>
</em>
</td>
<td>
<p>Ports to be attached to the server instance. They are created if a port with the given name does not already exist.
When you do not specify both networks and ports parameters, the server attaches to the only network created for the current tenant.</p>
</td>
</tr>
<tr>
<td>
<code>subnet</code><br/>
<em>
string
</em>
</td>
<td>
<p>UUID, IP address of a port from this subnet will be marked as AccessIPv4 on the created compute instance</p>
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
<p>The floatingIP which will be associated to the machine, only used for master.
The floatingIP should have been created and haven&rsquo;t been associated.</p>
</td>
</tr>
<tr>
<td>
<code>securityGroups</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.SecurityGroupParam">
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
<p>Machine tags
Requires Nova api 2.52 minimum!</p>
</td>
</tr>
<tr>
<td>
<code>serverMetadata</code><br/>
<em>
map[string]string
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.RootVolume">
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
<code>serverGroupID</code><br/>
<em>
string
</em>
</td>
<td>
<p>The server group to assign the machine to</p>
</td>
</tr>
<tr>
<td>
<code>identityRef</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackIdentityReference">
OpenStackIdentityReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IdentityRef is a reference to a identity to be used when reconciling this cluster</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineStatus">OpenStackMachineStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachine">OpenStackMachine</a>)
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.InstanceState">
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
<code>failureReason</code><br/>
<em>
<a href="https://pkg.go.dev/sigs.k8s.io/cluster-api@v1.5.1/errors#MachineStatusError">
sigs.k8s.io/cluster-api/errors.MachineStatusError
</a>
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
<a href="https://doc.crds.dev/github.com/kubernetes-sigs/cluster-api@v1.5.1">
sigs.k8s.io/cluster-api/api/v1beta1.Conditions
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineTemplateResource">OpenStackMachineTemplateResource
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineTemplateSpec">OpenStackMachineTemplateSpec</a>)
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineSpec">
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
<code>instanceID</code><br/>
<em>
string
</em>
</td>
<td>
<p>InstanceID is the OpenStack instance ID for this machine.</p>
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
<em>(Optional)</em>
<p>The name of the cloud to use from the clouds secret</p>
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
<code>image</code><br/>
<em>
string
</em>
</td>
<td>
<p>The name of the image to use for your server instance.
If the RootVolume is specified, this will be ignored and use rootVolume directly.</p>
</td>
</tr>
<tr>
<td>
<code>imageUUID</code><br/>
<em>
string
</em>
</td>
<td>
<p>The uuid of the image to use for your server instance.
if it&rsquo;s empty, Image name will be used</p>
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
<code>networks</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.NetworkParam">
[]NetworkParam
</a>
</em>
</td>
<td>
<p>A networks object. Required parameter when there are multiple networks defined for the tenant.
When you do not specify both networks and ports parameters, the server attaches to the only network created for the current tenant.</p>
</td>
</tr>
<tr>
<td>
<code>ports</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.PortOpts">
[]PortOpts
</a>
</em>
</td>
<td>
<p>Ports to be attached to the server instance. They are created if a port with the given name does not already exist.
When you do not specify both networks and ports parameters, the server attaches to the only network created for the current tenant.</p>
</td>
</tr>
<tr>
<td>
<code>subnet</code><br/>
<em>
string
</em>
</td>
<td>
<p>UUID, IP address of a port from this subnet will be marked as AccessIPv4 on the created compute instance</p>
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
<p>The floatingIP which will be associated to the machine, only used for master.
The floatingIP should have been created and haven&rsquo;t been associated.</p>
</td>
</tr>
<tr>
<td>
<code>securityGroups</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.SecurityGroupParam">
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
<p>Machine tags
Requires Nova api 2.52 minimum!</p>
</td>
</tr>
<tr>
<td>
<code>serverMetadata</code><br/>
<em>
map[string]string
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.RootVolume">
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
<code>serverGroupID</code><br/>
<em>
string
</em>
</td>
<td>
<p>The server group to assign the machine to</p>
</td>
</tr>
<tr>
<td>
<code>identityRef</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackIdentityReference">
OpenStackIdentityReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IdentityRef is a reference to a identity to be used when reconciling this cluster</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineTemplateSpec">OpenStackMachineTemplateSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineTemplate">OpenStackMachineTemplate</a>)
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineTemplateResource">
OpenStackMachineTemplateResource
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.PortOpts">PortOpts
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.Network">Network</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineSpec">OpenStackMachineSpec</a>)
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.NetworkFilter">
NetworkFilter
</a>
</em>
</td>
<td>
<p>Network is a query for an openstack network that the port will be created or discovered on.
This will fail if the query returns more than one network.</p>
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
<p>Used to make the name of the port unique. If unspecified, instead the 0-based index of the port in the list is used.</p>
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
<code>adminStateUp</code><br/>
<em>
bool
</em>
</td>
<td>
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
</td>
</tr>
<tr>
<td>
<code>fixedIPs</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.FixedIP">
[]FixedIP
</a>
</em>
</td>
<td>
<p>Specify pairs of subnet and/or IP address. These should be subnets of the network with the given NetworkID.</p>
</td>
</tr>
<tr>
<td>
<code>tenantId</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>projectId</code><br/>
<em>
string
</em>
</td>
<td>
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
<p>The uuids of the security groups to assign to the instance</p>
</td>
</tr>
<tr>
<td>
<code>securityGroupFilters</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.SecurityGroupParam">
[]SecurityGroupParam
</a>
</em>
</td>
<td>
<p>The names, uuids, filters or any combination these of the security groups to assign to the instance</p>
</td>
</tr>
<tr>
<td>
<code>allowedAddressPairs</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.AddressPair">
[]AddressPair
</a>
</em>
</td>
<td>
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
<p>Enables and disables trunk at port level. If not provided, openStackMachine.Spec.Trunk is inherited.</p>
</td>
</tr>
<tr>
<td>
<code>hostId</code><br/>
<em>
string
</em>
</td>
<td>
<p>The ID of the host where the port is allocated</p>
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
<p>The virtual network interface card (vNIC) type that is bound to the neutron port.</p>
</td>
</tr>
<tr>
<td>
<code>profile</code><br/>
<em>
map[string]string
</em>
</td>
<td>
<p>A dictionary that enables the application running on the specified
host to pass and receive virtual network interface (VIF) port-specific
information to the plug-in.</p>
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
<p>DisablePortSecurity enables or disables the port security when set.
When not set, it takes the value of the corresponding field at the network level.</p>
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
<p>Tags applied to the port (and corresponding trunk, if a trunk is configured.)
These tags are applied in addition to the instance&rsquo;s tags, which will also be applied to the port.</p>
</td>
</tr>
<tr>
<td>
<code>valueSpecs</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.ValueSpec">
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
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.RootVolume">RootVolume
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.Instance">Instance</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineSpec">OpenStackMachineSpec</a>)
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
<code>diskSize</code><br/>
<em>
int
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>volumeType</code><br/>
<em>
string
</em>
</td>
<td>
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
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.Router">Router
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.Network">Network</a>)
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
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.SecurityGroup">SecurityGroup
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterStatus">OpenStackClusterStatus</a>)
</p>
<p>
<p>SecurityGroup represents the basic information of the associated
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
<code>rules</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.SecurityGroupRule">
[]SecurityGroupRule
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.SecurityGroupFilter">SecurityGroupFilter
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.SecurityGroupParam">SecurityGroupParam</a>)
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
<code>tenantId</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>projectId</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>limit</code><br/>
<em>
int
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>marker</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>sortKey</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>sortDir</code><br/>
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
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>tagsAny</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>notTags</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>notTagsAny</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.SecurityGroupParam">SecurityGroupParam
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackMachineSpec">OpenStackMachineSpec</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.PortOpts">PortOpts</a>)
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
<code>uuid</code><br/>
<em>
string
</em>
</td>
<td>
<p>Security Group UID</p>
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
<p>Security Group name</p>
</td>
</tr>
<tr>
<td>
<code>filter</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.SecurityGroupFilter">
SecurityGroupFilter
</a>
</em>
</td>
<td>
<p>Filters used to query security groups in openstack</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.SecurityGroupRule">SecurityGroupRule
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.SecurityGroup">SecurityGroup</a>)
</p>
<p>
<p>SecurityGroupRule represent the basic information of the associated OpenStack
Security Group Role.</p>
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
<code>direction</code><br/>
<em>
string
</em>
</td>
<td>
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
</td>
</tr>
<tr>
<td>
<code>securityGroupID</code><br/>
<em>
string
</em>
</td>
<td>
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
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.Subnet">Subnet
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.Network">Network</a>)
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
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.SubnetFilter">SubnetFilter
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.FixedIP">FixedIP</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.OpenStackClusterSpec">OpenStackClusterSpec</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.SubnetParam">SubnetParam</a>)
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
<code>projectId</code><br/>
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
<code>gateway_ip</code><br/>
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
<code>ipv6RaMode</code><br/>
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
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>tagsAny</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>notTags</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>notTagsAny</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.SubnetParam">SubnetParam
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.ExternalRouterIPParam">ExternalRouterIPParam</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.NetworkParam">NetworkParam</a>)
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
<code>uuid</code><br/>
<em>
string
</em>
</td>
<td>
<p>Optional UUID of the subnet.
If specified this will not be validated prior to server creation.
If specified, the enclosing <code>NetworkParam</code> must also be specified by UUID.</p>
</td>
</tr>
<tr>
<td>
<code>filter</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.SubnetFilter">
SubnetFilter
</a>
</em>
</td>
<td>
<p>Filters for optional subnet query</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha6.ValueSpec">ValueSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha6.PortOpts">PortOpts</a>)
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
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>.
</em></p>
