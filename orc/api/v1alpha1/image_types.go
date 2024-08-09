/*
Copyright 2024 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ImageControllerFinalizer  = "openstack.k-orc.cloud/image"
	ImageControllerFieldOwner = "openstack.k-orc.cloud/imagecontroller"
)

// GlanceTag is the name of the go field tag in properties structs used to specify the glance property name.
const GlanceTag = "glance"

// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=255
type ImageTag string

// +kubebuilder:validation:Enum:=ami;ari;aki;bare;ovf;ova;docker
type ImageContainerFormat string

const (
	ImageContainerFormatAKI    ImageContainerFormat = "aki"
	ImageContainerFormatAMI    ImageContainerFormat = "ami"
	ImageContainerFormatARI    ImageContainerFormat = "ari"
	ImageContainerFormatBare   ImageContainerFormat = "bare"
	ImageContainerFormatDocker ImageContainerFormat = "docker"
	ImageContainerFormatOVA    ImageContainerFormat = "ova"
	ImageContainerFormatOVF    ImageContainerFormat = "ovf"
)

// +kubebuilder:validation:Enum:=ami;ari;aki;vhd;vhdx;vmdk;raw;qcow2;vdi;ploop;iso
type ImageDiskFormat string

const (
	ImageDiskFormatAMI   ImageDiskFormat = "ami"
	ImageDiskFormatARI   ImageDiskFormat = "ari"
	ImageDiskFormatAKI   ImageDiskFormat = "aki"
	ImageDiskFormatVHD   ImageDiskFormat = "vhd"
	ImageDiskFormatVHDX  ImageDiskFormat = "vhdx"
	ImageDiskFormatVMDK  ImageDiskFormat = "vmdk"
	ImageDiskFormatRaw   ImageDiskFormat = "raw"
	ImageDiskFormatQCOW2 ImageDiskFormat = "qcow2"
	ImageDiskFormatVDI   ImageDiskFormat = "vdi"
	ImageDiskFormatPLoop ImageDiskFormat = "ploop"
	ImageDiskFormatISO   ImageDiskFormat = "iso"
)

// +kubebuilder:validation:Enum:=public;private;shared;community
type ImageVisibility string

const (
	ImageVisibilityPublic    ImageVisibility = "public"
	ImageVisibilityPrivate   ImageVisibility = "private"
	ImageVisibilityShared    ImageVisibility = "shared"
	ImageVisibilityCommunity ImageVisibility = "community"
)

// +kubebuilder:validation:Enum:=md5;sha1;sha256;sha512
type ImageHashAlgorithm string

const (
	ImageHashAlgorithmMD5    ImageHashAlgorithm = "md5"
	ImageHashAlgorithmSHA1   ImageHashAlgorithm = "sha1"
	ImageHashAlgorithmSHA256 ImageHashAlgorithm = "sha256"
	ImageHashAlgorithmSHA512 ImageHashAlgorithm = "sha512"
)

// See https://docs.openstack.org/glance/latest/admin/useful-image-properties.html
// for a list of 'well known' image properties we might consider supporting explicitly.
//
// The set of supported properties is currently arbitrarily selective. We should
// add supported options here freely.

// ImageHWBus is a type of hardware bus.
//
// Permitted values are scsi, virtio, uml, xen, ide, usb, and lxc.
// +kubebuilder:validation:Enum:=scsi;virtio;uml;xen;ide;usb;lxc
type ImageHWBus string

type ImagePropertiesHardware struct {
	// CPUSockets is the preferred number of sockets to expose to the guest
	// +optional
	CPUSockets *int `json:"cpuSockets,omitempty" glance:"hw_cpu_sockets"`

	// CPUCores is the preferred number of cores to expose to the guest
	// +optional
	CPUCores *int `json:"cpuCores,omitempty" glance:"hw_cpu_cores"`

	// CPUThreads is the preferred number of threads to expose to the guest
	// +optional
	CPUThreads *int `json:"cpuThreads,omitempty" glance:"hw_cpu_threads"`

	// CPUPolicy is used to pin the virtual CPUs (vCPUs) of instances to the
	// host's physical CPU cores (pCPUs). Host aggregates should be used to
	// separate these pinned instances from unpinned instances as the latter
	// will not respect the resourcing requirements of the former.
	//
	// Permitted values are shared (the default), and dedicated.
	//
	// shared: The guest vCPUs will be allowed to freely float across host
	// pCPUs, albeit potentially constrained by NUMA policy.
	//
	// dedicated: The guest vCPUs will be strictly pinned to a set of host
	// pCPUs. In the absence of an explicit vCPU topology request, the
	// drivers typically expose all vCPUs as sockets with one core and one
	// thread. When strict CPU pinning is in effect the guest CPU topology
	// will be setup to match the topology of the CPUs to which it is
	// pinned. This option implies an overcommit ratio of 1.0. For example,
	// if a two vCPU guest is pinned to a single host core with two threads,
	// then the guest will get a topology of one socket, one core, two
	// threads.
	// +kubebuilder:validation:Enum:=shared;dedicated
	// +optional
	CPUPolicy *string `json:"cpuPolicy,omitempty" glance:"hw_cpu_policy"`

	// CPUThreadPolicy further refines a CPUPolicy of 'dedicated' by stating
	// how hardware CPU threads in a simultaneous multithreading-based (SMT)
	// architecture be used. SMT-based architectures include Intel
	// processors with Hyper-Threading technology. In these architectures,
	// processor cores share a number of components with one or more other
	// cores. Cores in such architectures are commonly referred to as
	// hardware threads, while the cores that a given core share components
	// with are known as thread siblings.
	//
	// Permitted values are prefer (the default), isolate, and require.
	//
	// prefer: The host may or may not have an SMT architecture. Where an
	// SMT architecture is present, thread siblings are preferred.
	//
	// isolate: The host must not have an SMT architecture or must emulate a
	// non-SMT architecture. If the host does not have an SMT architecture,
	// each vCPU is placed on a different core as expected. If the host does
	// have an SMT architecture - that is, one or more cores have thread
	// siblings - then each vCPU is placed on a different physical core. No
	// vCPUs from other guests are placed on the same core. All but one
	// thread sibling on each utilized core is therefore guaranteed to be
	// unusable.
	//
	// require: The host must have an SMT architecture. Each vCPU is
	// allocated on thread siblings. If the host does not have an SMT
	// architecture, then it is not used. If the host has an SMT
	// architecture, but not enough cores with free thread siblings are
	// available, then scheduling fails.
	// +kubebuilder:validation:Enum:=prefer;isolate;require
	// +optional
	CPUThreadPolicy *string `json:"cpuThreadPolicy,omitempty" glance:"hw_cpu_thread_policy"`

	// CDROMBus specifies the type of disk controller to attach CD-ROM devices to.
	// +optional
	CDROMBus *ImageHWBus `json:"cdromBus,omitempty" glance:"hw_cdrom_bus"`

	// DiskBus specifies the type of disk controller to attach disk devices to.
	// +optional
	DiskBus *ImageHWBus `json:"diskBus,omitempty" glance:"hw_disk_bus"`

	// TODO: hw_machine_type seems important to support early, but how to
	// select a supported set?

	// SCSIModel enables the use of VirtIO SCSI (virtio-scsi) to provide
	// block device access for compute instances; by default, instances use
	// VirtIO Block (virtio-blk). VirtIO SCSI is a para-virtualized SCSI
	// controller device that provides improved scalability and performance,
	// and supports advanced SCSI hardware.
	//
	// The only permitted value is virtio-scsi.
	// +kubebuilder:validation:Enum:=virtio-scsi
	// +optional
	SCSIModel *string `json:"scsiModel,omitempty" glance:"hw_scsi_model"`

	// VIFModel specifies the model of virtual network interface device to use.
	//
	// Permitted values are e1000, e1000e, ne2k_pci, pcnet, rtl8139, virtio,
	// and vmxnet3.
	// +kubebuilder:validation:Enum:=e1000;e1000e;ne2k_pci;pcnet;rtl8139;virtio;vmxnet3
	// +optional
	VIFModel *string `json:"vifModel,omitempty" glance:"hw_vif_model"`
}

type ImageProperties struct {
	// MinDisk is the minimum amount of disk space in GB that is required to boot the image
	// +kubebuilder:validation:Minimum:=1
	// +optional
	MinDiskGB *int `json:"minDiskGB,omitempty"`

	// MinMemoryMB is the minimum amount of RAM in MB that is required to boot the image.
	// +kubebuilder:validation:Minimum:=1
	// +optional
	MinMemoryMB *int `json:"minMemoryMB,omitempty"`

	// Hardware is a set of properties which control the virtual hardware
	// created by Nova.
	// +optional
	Hardware *ImagePropertiesHardware `json:"hardware,omitempty"`
}

// +kubebuilder:validation:Enum:=xz;gz;bz2
type ImageCompression string

const (
	ImageCompressionXZ  ImageCompression = "xz"
	ImageCompressionGZ  ImageCompression = "gz"
	ImageCompressionBZ2 ImageCompression = "bz2"
)

// +kubebuilder:validation:Enum:=URL
type ImageContentSourceType string

const (
	ImageSourceTypeURL ImageContentSourceType = "URL"
)

// ImageContent specifies the source of image data
// +kubebuilder:validation:XValidation:rule="has(self.sourceType) && self.sourceType == 'URL' ?  has(self.sourceURL) : !has(self.sourceURL)",message="sourceURL is required when sourceType is URL, and forbidden otherwise"
// +union.
type ImageContent struct {
	// ContainerFormat is the format of the image container.
	// qcow2 and raw images do not usually have a container. This is specified as "bare", which is also the default.
	// Permitted values are ami, ari, aki, bare, ovf, ova, and docker.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="containerFormat is immutable"
	// +kubebuilder:validation:Required
	// +kubebuilder:default:=bare
	ContainerFormat ImageContainerFormat `json:"containerFormat"`

	// DiskFormat is the format of the disk image.
	// Normal values are "qcow2", or "raw". Glance may be configured to support others.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="diskFormat is immutable"
	// +kubebuilder:validation:Required
	DiskFormat ImageDiskFormat `json:"diskFormat"`

	// SourceType is the type of the image content source
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="sourceType is immutable"
	// +kubebuilder:validation:Required
	// +unionDiscriminator
	SourceType ImageContentSourceType `json:"sourceType"`

	// SourceURL describes how to obtain image data by downloading it from a SourceURL. Must be set if Type is 'url'
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="sourceURL is immutable"
	// +unionMember
	SourceURL *ImageContentSourceURL `json:"sourceURL,omitempty"`
}

type ImageContentSourceURL struct {
	// URL containing image data
	// +kubebuilder:validation:Format=uri
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// Decompress specifies that the source data must be decompressed with the
	// given compression algorithm before being stored. Specifying Decompress
	// will disable the use of Glance's web-download, as web-download cannot
	// currently deterministically decompress downloaded content.
	// +optional
	Decompress *ImageCompression `json:"decompress,omitempty"`

	// DownloadHash is a hash which will be used to verify downloaded data, i.e.
	// before any decompression. If not specified, no hash verification will be
	// performed. Specifying a DownloadHash will disable the use of Glance's
	// web-download, as web-download cannot currently deterministically verify
	// the hash of downloaded content.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="downloadHash is immutable"
	// +optional
	DownloadHash *ImageHash `json:"downloadHash,omitempty"`
}

type ImageHash struct {
	// Algorithm is the hash algorithm used to generate value.
	// +kubebuilder:validation:Required
	Algorithm ImageHashAlgorithm `json:"algorithm"`

	// Value is the hash of the image data using Algorithm. It must be hex encoded using lowercase letters.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=1024
	// +kubebuilder:validation:Pattern:=`^[0-9a-f]+$`
	// +kubebuilder:validation:Required
	Value string `json:"value"`
}

// ImageSpec defines the desired state of an Image.
// +kubebuilder:validation:XValidation:rule="(!has(self.controllerOptions) || !has(self.controllerOptions.onCreate) || self.controllerOptions.onCreate == 'AdoptOrCreate') ? has(self.content) : !has(self.content)",message="content must be set if controllerOptions.onCreate is AdoptOrCreate, not otherwise"
type ImageSpec struct {
	// ImageName will be the name of the created Glance image.
	// If not specified, the name of the Image object will be used.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="imageName is immutable"
	// +optional
	ImageName *string `json:"imageName,omitempty"`

	// Protected specifies that the image is protected from deletion.
	// If not specified, the default is false.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="protected is immutable"
	// +optional
	Protected *bool `json:"protected,omitempty"`

	// Tags is a list of tags which will be applied to the image. A tag has a maximum length of 255 characters.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="tags is immutable"
	// +listType=set
	// +optional
	Tags []ImageTag `json:"tags,omitempty"`

	// Visibility of the image
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="visibility is immutable"
	// +optional
	Visibility *ImageVisibility `json:"visibility,omitempty"`

	// Properties is metadata available to consumers of the image
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="properties is immutable"
	// +optional
	Properties *ImageProperties `json:"properties,omitempty"`

	// Content specifies how to obtain the image content. Must be set if
	// ControllerOptions.OnCreate is AdoptOrCreate. Must not be set if
	// ControllerOptions.OnCreate is Adopt.
	// +optional
	Content *ImageContent `json:"content,omitempty"`

	// ControllerOptions are options which control the behaviour of the resource controller.
	// +optional
	ControllerOptions *ControllerOptions `json:"controllerOptions,omitempty"`

	// CloudCredentialsRef points to a secret containing OpenStack credentials
	// +kubebuilder:validation:Required
	CloudCredentialsRef CloudCredentialsReference `json:"cloudCredentialsRef"`
}

var _ ObjectWithControllerOptions = &Image{}

func (i *Image) GetControllerOptions() *ControllerOptions {
	return i.Spec.ControllerOptions
}

// ImageStatus defines the observed state of an Image.
type ImageStatus struct {
	// Conditions represents the observed status of the object.
	// Known .status.conditions.type are: "Available", "Progressing", "Failed"
	//
	// Available represents the availability of the glance image. If it is
	// true then the image is ready for use in glance, and its hash has been
	// verified.
	//
	// Progressing indicates that reconciliation is in progress. It may not
	// be in progress either because reconciliation was completed
	// successfully, or because an error was encountered that cannot be
	// retried.
	//
	// Failed indicates that reconciliation was not successful, and further
	// reconciliation is not possible.
	//
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// DownloadAttempts is the number of times the controller has attempted to download the image contents
	// +optional
	DownloadAttempts *int `json:"downloadAttempts,omitempty"`

	// ImageID is the UUID of the glance image
	// +optional
	ImageID *string `json:"imageID,omitempty"`

	// Status is the image status as reported by glance
	// +optional
	Status *string `json:"status,omitempty"`

	// Hash is the hash of the image data published by glance
	// +optional
	Hash *ImageHash `json:"hash,omitempty"`

	// SizeB is the size of the image data, in bytes
	// +optional
	SizeB *int64 `json:"sizeB,omitempty"`

	// VirtualSizeB is the size of the disk the image data represents, in bytes
	// +optional
	VirtualSizeB *int64 `json:"virtualSizeB,omitempty"`
}

var _ ObjectWithConditions = &Image{}

func (i *Image) GetConditions() []metav1.Condition {
	return i.Status.Conditions
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ImageID",type="string",JSONPath=".status.imageID",description="Glance image ID"
// +kubebuilder:printcolumn:name="Available",type="string",JSONPath=".status.conditions[?(@.type=='Available')].status",description="Availability status of image"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[?(@.type=='Available')].message",description="Message describing current availability status"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation"

// Image is the Schema for the ORC images API.
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 63",message="name may not be longer than 63 characters"
type Image struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageSpec   `json:"spec,omitempty"`
	Status ImageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ImageList contains a list of Image.
type ImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Image `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Image{}, &ImageList{})
}

func (i *Image) GetCloudCredentialsRef() (*string, *CloudCredentialsReference) {
	if i == nil {
		return nil, nil
	}

	return &i.Namespace, &i.Spec.CloudCredentialsRef
}

var _ CloudCredentialsRefProvider = &Image{}
