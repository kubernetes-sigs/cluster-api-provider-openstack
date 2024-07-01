<h2 id="infrastructure.cluster.x-k8s.io/v1alpha1">infrastructure.cluster.x-k8s.io/v1alpha1</h2>
<p>
<p>package v1alpha1 contains API Schema definitions for the infrastructure v1alpha1 API group</p>
</p>
Resource Types:
<ul></ul>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.ImageAdditionalProperty">ImageAdditionalProperty
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackImageSpec">OpenStackImageSpec</a>)
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageAdditionalPropertyName">
ImageAdditionalPropertyName
</a>
</em>
</td>
<td>
<p>Name is the name of the glance property. It is an error for it to conflict with any explicitly defined property.</p>
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
<p>Value is the value of the glance property</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.ImageAdditionalPropertyName">ImageAdditionalPropertyName
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageAdditionalProperty">ImageAdditionalProperty</a>)
</p>
<p>
</p>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.ImageCompression">ImageCompression
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageSourceURL">ImageSourceURL</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;gz&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;xz&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.ImageContainerFormat">ImageContainerFormat
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackImageSpec">OpenStackImageSpec</a>)
</p>
<p>
</p>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.ImageContent">ImageContent
</h3>
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
<code>hash</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageHash">
ImageHash
</a>
</em>
</td>
<td>
<p>Hash is a hash which can be used to verify the downloaded image data</p>
</td>
</tr>
<tr>
<td>
<code>glanceHashAlgorithm</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageHashAlgorithm">
ImageHashAlgorithm
</a>
</em>
</td>
<td>
<p>GlanceHashAlgorithm is the algorithm of the hash glance will publish.
If not set it will default to sha512, which is Glance&rsquo;s default. It
MUST be set to the value configured in Glance.</p>
</td>
</tr>
<tr>
<td>
<code>source</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageSource">
ImageSource
</a>
</em>
</td>
<td>
<p>Source specifies how to obtain the image data</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.ImageDiskFormat">ImageDiskFormat
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackImageSpec">OpenStackImageSpec</a>)
</p>
<p>
</p>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.ImageHash">ImageHash
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageContent">ImageContent</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackImageStatus">OpenStackImageStatus</a>)
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
<code>algorithm</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageHashAlgorithm">
ImageHashAlgorithm
</a>
</em>
</td>
<td>
<p>Algorithm is the hash algorithm used to generate value.</p>
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
<p>Value is the hash of the image data using Algorithm.
kubebuilder:validation:MinLength:=1</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.ImageHashAlgorithm">ImageHashAlgorithm
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageContent">ImageContent</a>, 
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageHash">ImageHash</a>)
</p>
<p>
<p>kubebuilder:validation:Enum:=md5;sha1;sha256;sha512</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;md5&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;sha1&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;sha256&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;sha512&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.ImageProperties">ImageProperties
</h3>
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
<code>minDiskGB</code><br/>
<em>
int
</em>
</td>
<td>
<p>MinDisk is the minimum amount of disk space in GB that is required to boot the image</p>
</td>
</tr>
<tr>
<td>
<code>minRAMMB</code><br/>
<em>
int
</em>
</td>
<td>
<p>MinRAMMB is the minimum amount of RAM in MB that is required to boot the image.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.ImageSource">ImageSource
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageContent">ImageContent</a>)
</p>
<p>
<p>ImageSource specifies the source of image data</p>
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageSourceType">
ImageSourceType
</a>
</em>
</td>
<td>
<p>Type is the type of the image source</p>
</td>
</tr>
<tr>
<td>
<code>url</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageSourceURL">
ImageSourceURL
</a>
</em>
</td>
<td>
<p>URL describes how to obtain image data by downloading it from a URL. Must be set if Type is &lsquo;url&rsquo;</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.ImageSourceType">ImageSourceType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageSource">ImageSource</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;url&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.ImageSourceURL">ImageSourceURL
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageSource">ImageSource</a>)
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
<code>url</code><br/>
<em>
string
</em>
</td>
<td>
<p>URL containing image data</p>
</td>
</tr>
<tr>
<td>
<code>decompress</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageCompression">
ImageCompression
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Decompress specifies that the source data must be decompressed with the given compression algorithm before being stored</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.ImageTag">ImageTag
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackImageSpec">OpenStackImageSpec</a>)
</p>
<p>
</p>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.ImageVisibility">ImageVisibility
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackImageSpec">OpenStackImageSpec</a>)
</p>
<p>
<p>kubebuilder:validation:Enum:=public;private;shared;community</p>
</p>
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
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackImage">OpenStackImage
</h3>
<p>
<p>OpenStackImage is the Schema for the openstackimages API.</p>
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
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackImageSpec">
OpenStackImageSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>containerFormat</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageContainerFormat">
ImageContainerFormat
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ContainerFormat is the format of the image container.
qcow2 and raw images do not usually have a container, and this can be omitted.</p>
</td>
</tr>
<tr>
<td>
<code>diskFormat</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageDiskFormat">
ImageDiskFormat
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DiskFormat is the format of the disk image.
Normal values are &ldquo;qcow2&rdquo;, or &ldquo;raw&rdquo;. Glance may be configured to support others.</p>
</td>
</tr>
<tr>
<td>
<code>protected</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Protected specifies that the image is protected from deletion.
If not specified, the default is false.</p>
</td>
</tr>
<tr>
<td>
<code>tags</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageTag">
[]ImageTag
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Tags is a list of tags which will be applied to the image. A tag has a maximum length of 255 characters.</p>
</td>
</tr>
<tr>
<td>
<code>visibility</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageVisibility">
ImageVisibility
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Visibility of the image</p>
</td>
</tr>
<tr>
<td>
<code>additionalProperties</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageAdditionalProperty">
[]ImageAdditionalProperty
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AdditionalProperties allows arbitrary glance properties to be set. These will be merged</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackImageStatus">
OpenStackImageStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackImageSpec">OpenStackImageSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackImage">OpenStackImage</a>)
</p>
<p>
<p>OpenStackImageSpec defines the desired state of OpenStackImage.</p>
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
<code>containerFormat</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageContainerFormat">
ImageContainerFormat
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ContainerFormat is the format of the image container.
qcow2 and raw images do not usually have a container, and this can be omitted.</p>
</td>
</tr>
<tr>
<td>
<code>diskFormat</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageDiskFormat">
ImageDiskFormat
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DiskFormat is the format of the disk image.
Normal values are &ldquo;qcow2&rdquo;, or &ldquo;raw&rdquo;. Glance may be configured to support others.</p>
</td>
</tr>
<tr>
<td>
<code>protected</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Protected specifies that the image is protected from deletion.
If not specified, the default is false.</p>
</td>
</tr>
<tr>
<td>
<code>tags</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageTag">
[]ImageTag
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Tags is a list of tags which will be applied to the image. A tag has a maximum length of 255 characters.</p>
</td>
</tr>
<tr>
<td>
<code>visibility</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageVisibility">
ImageVisibility
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Visibility of the image</p>
</td>
</tr>
<tr>
<td>
<code>additionalProperties</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageAdditionalProperty">
[]ImageAdditionalProperty
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AdditionalProperties allows arbitrary glance properties to be set. These will be merged</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackImageStatus">OpenStackImageStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.OpenStackImage">OpenStackImage</a>)
</p>
<p>
<p>OpenStackImageStatus defines the observed state of OpenStackImage.</p>
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
<p>ID is the UUID of the glance image</p>
</td>
</tr>
<tr>
<td>
<code>hash</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.ImageHash">
ImageHash
</a>
</em>
</td>
<td>
<p>Hash is the hash of the image data calculated by glance</p>
</td>
</tr>
<tr>
<td>
<code>sizeB</code><br/>
<em>
int64
</em>
</td>
<td>
<p>SizeB is the size of the image data, in bytes</p>
</td>
</tr>
<tr>
<td>
<code>virtualSizeB</code><br/>
<em>
int64
</em>
</td>
<td>
<p>VirtualSizeB is the size of the disk the image data represents, in bytes</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code><br/>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Condition">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<p>Conditions represents the
Known .status.conditions.type are: &ldquo;Available&rdquo;, &ldquo;Progressing&rdquo;, and &ldquo;Status&rdquo;</p>
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
