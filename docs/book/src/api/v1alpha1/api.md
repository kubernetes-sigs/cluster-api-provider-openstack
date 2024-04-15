<h2 id="infrastructure.cluster.x-k8s.io/v1alpha1">infrastructure.cluster.x-k8s.io/v1alpha1</h2>
<p>
<p>package v1alpha1 contains API Schema definitions for the infrastructure v1alpha1 API group</p>
</p>
Resource Types:
<ul></ul>
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
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>.
</em></p>
