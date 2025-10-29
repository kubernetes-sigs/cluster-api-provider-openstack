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

package apivalidations

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	infrav1alpha1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

var _ = Describe("OpenStackServer API validations", func() {
	var namespace *corev1.Namespace

	defaultServer := func() *infrav1alpha1.OpenStackServer {
		// Initialise a basic server object in the correct namespace
		server := &infrav1alpha1.OpenStackServer{
			Spec: infrav1alpha1.OpenStackServerSpec{
				Flavor: ptr.To("test-flavor"),
				IdentityRef: infrav1.OpenStackIdentityReference{
					Name:      "test-identity",
					CloudName: "test-cloud",
				},
				Image: infrav1.ImageParam{Filter: &infrav1.ImageFilter{Name: ptr.To("test-image")}},
				Ports: []infrav1.PortOpts{
					{
						CommonPortOpts: infrav1.CommonPortOpts{
							Network: &infrav1.NetworkParam{
								Filter: &infrav1.NetworkFilter{
									Name: "test-network",
								},
							},
						},
					},
				},
			},
		}
		server.Namespace = namespace.Name
		server.GenerateName = "openstackserver-"
		return server
	}

	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should allow to create a server with correct spec", func() {
		server := defaultServer()

		By("Creating the smallest permissible server spec")
		Expect(k8sClient.Create(ctx, server)).To(Succeed(), "OpenStackServer creation should succeed")
	})

	It("should not allow the identityRef to be set several times", func() {
		server := defaultServer()

		By("Creating a bare server")
		Expect(k8sClient.Create(ctx, server)).To(Succeed(), "OpenStackserver creation should succeed")

		By("Setting the identityRef")
		server.Spec.IdentityRef = infrav1.OpenStackIdentityReference{Name: "foo", CloudName: "staging"}
		Expect(k8sClient.Update(ctx, server)).NotTo(Succeed(), "OpenStackserver update should fail")
	})

	It("should not allow server metadata to exceed 255 characters", func() {
		server := defaultServer()

		By("Creating a server with a metadata key that is too long")
		server.Spec.ServerMetadata = []infrav1.ServerMetadata{
			{
				Key:   strings.Repeat("a", 256),
				Value: "value",
			},
		}
		Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "Creating a server with a long metadata key should fail")

		By("Creating a server with a metadata value that is too long")
		server.Spec.ServerMetadata = []infrav1.ServerMetadata{
			{
				Key:   "key",
				Value: strings.Repeat("a", 256),
			},
		}
		Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "Creating a server with a long metadata value should fail")

		By("Creating a server with a metadata key and value of 255 characters should succeed")
		server.Spec.ServerMetadata = []infrav1.ServerMetadata{
			{
				Key:   strings.Repeat("a", 255),
				Value: strings.Repeat("b", 255),
			},
		}
		Expect(k8sClient.Create(ctx, server)).To(Succeed(), "Creating a server with max metadata key and value should succeed")
	})

	Context("flavors", func() {
		It("should require either a flavor or flavorID", func() {
			server := defaultServer()

			By("Creating a server with no flavor or flavor id")
			server.Spec.Flavor = nil
			Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "Creating a server with no flavor name or id should fail")

			By("Creating a server with a flavor id")
			server.Spec.FlavorID = ptr.To("6aa02f56-c595-4d2f-9f8e-3c6296a4bed9")
			Expect(k8sClient.Create(ctx, server)).To(Succeed(), "Creating a server with a flavor id should succeed")
		})
	})

	Context("volumes", func() {
		It("should not allow volume with zero size", func() {
			server := defaultServer()
			server.Spec.RootVolume = &infrav1.RootVolume{
				SizeGiB: 0,
			}
			Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "Creating a server with a zero size root volume should fail")

			server = defaultServer()
			server.Spec.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
				{
					Name:    "test-volume",
					SizeGiB: 0,
				},
			}
			Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "Creating a server with a zero size additional block device should fail")
		})

		It("should allow to create server with spec.RootVolume and non-root device name in spec.AdditionalBlockDevices", func() {
			server := defaultServer()
			server.Spec.RootVolume = &infrav1.RootVolume{SizeGiB: 50, BlockDeviceVolume: infrav1.BlockDeviceVolume{}}
			server.Spec.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
				{Name: "user", SizeGiB: 30, Storage: infrav1.BlockDeviceStorage{}},
			}

			By("Creating a server with spec.RootVolume and non-root device name in spec.AdditionalBlockDevices")
			Expect(k8sClient.Create(ctx, server)).To(Succeed(), "OpenStackserver creation with non-root device name in spec.AdditionalBlockDevices should succeed")
		})

		It("should not allow to create server with spec.RootVolume and root device name in spec.AdditionalBlockDevices", func() {
			server := defaultServer()
			server.Spec.RootVolume = &infrav1.RootVolume{SizeGiB: 50, BlockDeviceVolume: infrav1.BlockDeviceVolume{}}
			server.Spec.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
				{Name: "root", SizeGiB: 30, Storage: infrav1.BlockDeviceStorage{}},
			}

			By("Creating a server with spec.RootVolume and root device name in spec.AdditionalBlockDevices")
			Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "OpenStackserver creation with root device name in spec.AdditionalBlockDevices should not succeed")
		})

		It("should not allow to create server with both SecurityGroups and DisablePortSecurity", func() {
			server := defaultServer()
			server.Spec.Ports = []infrav1.PortOpts{
				{
					CommonPortOpts: infrav1.CommonPortOpts{
						SecurityGroups: []infrav1.SecurityGroupParam{{
							Filter: &infrav1.SecurityGroupFilter{Name: "test-security-group"},
						}},
						ResolvedPortSpecFields: infrav1.ResolvedPortSpecFields{
							DisablePortSecurity: ptr.To(true),
						},
					},
				},
			}

			By("Creating a server with both SecurityGroups and DisablePortSecurity")
			Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "OpenStackServer creation with both SecurityGroups and DisablePortSecurity should not succeed")
		})

		/* FIXME: These tests are failing
		It("should not allow additional volume with empty name", func() {
			server.Spec.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
				{
					Name:    "",
					SizeGiB: 1,
				},
			}
			Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "Creating a server with an empty name additional block device should fail")
		})

		It("should not allow additional volume with name root", func() {
			server.Spec.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
				{
					Name:    "root",
					SizeGiB: 1,
				},
			}
			Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "Creating a server with a root named additional block device should fail")
		})
		*/

		It("should not allow additional volume with duplicate name", func() {
			server := defaultServer()
			server.Spec.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
				{
					Name:    "test-volume",
					SizeGiB: 1,
				},
				{
					Name:    "test-volume",
					SizeGiB: 2,
				},
			}
			Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "Creating a server with duplicate named additional block device should fail")
		})

		defaultserverWithRootVolumeAZ := func(az *infrav1.VolumeAvailabilityZone) *infrav1alpha1.OpenStackServer {
			server := defaultServer()
			server.Spec.RootVolume = &infrav1.RootVolume{
				SizeGiB: 1,
			}
			server.Spec.RootVolume.AvailabilityZone = az
			return server
		}

		defaultserverWithAdditionBlockDeviceAZ := func(az *infrav1.VolumeAvailabilityZone) *infrav1alpha1.OpenStackServer {
			server := defaultServer()
			server.Spec.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
				{
					Name:    "test-volume",
					SizeGiB: 1,
					Storage: infrav1.BlockDeviceStorage{
						Type: infrav1.VolumeBlockDevice,
						Volume: &infrav1.BlockDeviceVolume{
							AvailabilityZone: az,
						},
					},
				},
			}
			return server
		}

		It("should allow volume with defaulted AZ from", func() {
			azName := infrav1.VolumeAZName("test-az")
			az := infrav1.VolumeAvailabilityZone{
				Name: &azName,
			}

			server := defaultserverWithRootVolumeAZ(&az)
			Expect(k8sClient.Create(ctx, server)).To(Succeed(), "Creating a server with a root volume with an availability zone should succeed")

			server = defaultserverWithAdditionBlockDeviceAZ(&az)
			Expect(k8sClient.Create(ctx, server)).To(Succeed(), "Creating a server with an additional block device with an availability zone should succeed")
		})

		It("should allow volume with AZ from Name", func() {
			azName := infrav1.VolumeAZName("test-az")
			az := infrav1.VolumeAvailabilityZone{
				From: infrav1.VolumeAZFromName,
				Name: &azName,
			}

			server := defaultserverWithRootVolumeAZ(&az)
			Expect(k8sClient.Create(ctx, server)).To(Succeed(), "Creating a server with a root volume with an availability zone should succeed")

			server = defaultserverWithAdditionBlockDeviceAZ(&az)
			Expect(k8sClient.Create(ctx, server)).To(Succeed(), "Creating a server with an additional block device with an availability zone should succeed")
		})

		It("should allow volume AZ from server", func() {
			az := infrav1.VolumeAvailabilityZone{
				From: infrav1.VolumeAZFromMachine,
			}

			server := defaultserverWithRootVolumeAZ(&az)
			Expect(k8sClient.Create(ctx, server)).To(Succeed(), "Creating a server with a root volume with an availability zone should succeed")

			server = defaultserverWithAdditionBlockDeviceAZ(&az)
			Expect(k8sClient.Create(ctx, server)).To(Succeed(), "Creating a server with an additional block device with an availability zone should succeed")
		})

		It("should not allow volume AZ with invalid from", func() {
			az := infrav1.VolumeAvailabilityZone{
				From: "invalid",
			}

			server := defaultserverWithRootVolumeAZ(&az)
			Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "Creating a server with a root volume with an invalid availability zone should fail")

			server = defaultserverWithAdditionBlockDeviceAZ(&az)
			Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "Creating a server with an additional block device with an invalid availability zone should fail")
		})

		It("should not allow empty volume AZ", func() {
			az := infrav1.VolumeAvailabilityZone{}

			server := defaultserverWithRootVolumeAZ(&az)
			Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "Creating a server with a root volume with an empty availability zone should fail")

			server = defaultserverWithAdditionBlockDeviceAZ(&az)
			Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "Creating a server with an additional block device with an empty availability zone should fail")
		})

		It("should not allow volume AZ from Name with missing name", func() {
			az := infrav1.VolumeAvailabilityZone{
				From: infrav1.VolumeAZFromName,
			}

			server := defaultserverWithRootVolumeAZ(&az)
			Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "Creating a server with a root volume with a missing name availability zone should fail")

			server = defaultserverWithAdditionBlockDeviceAZ(&az)
			Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "Creating a server with an additional block device with a missing name availability zone should fail")
		})

		It("should not allow volume AZ from server with name", func() {
			azName := infrav1.VolumeAZName("test-az")
			az := infrav1.VolumeAvailabilityZone{
				From: infrav1.VolumeAZFromMachine,
				Name: &azName,
			}

			server := defaultserverWithRootVolumeAZ(&az)
			Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "Creating a server with a root volume with a name availability zone should fail")

			server = defaultserverWithAdditionBlockDeviceAZ(&az)
			Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "Creating a server with an additional block device with a name availability zone should fail")
		})

		It("should not allow volume AZ from Name with empty name", func() {
			azName := infrav1.VolumeAZName("")
			az := infrav1.VolumeAvailabilityZone{
				From: infrav1.VolumeAZFromName,
				Name: &azName,
			}

			server := defaultserverWithRootVolumeAZ(&az)
			Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "Creating a server with a root volume with an empty name availability zone should fail")

			server = defaultserverWithAdditionBlockDeviceAZ(&az)
			Expect(k8sClient.Create(ctx, server)).NotTo(Succeed(), "Creating a server with an additional block device with an empty name availability zone should fail")
		})
	})
})
