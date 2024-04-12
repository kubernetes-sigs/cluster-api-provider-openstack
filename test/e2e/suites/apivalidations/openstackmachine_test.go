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

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

var _ = Describe("OpenStackMachine API validations", func() {
	var namespace *corev1.Namespace

	defaultMachine := func() *infrav1.OpenStackMachine {
		// Initialise a basic machine object in the correct namespace
		machine := &infrav1.OpenStackMachine{
			Spec: infrav1.OpenStackMachineSpec{
				Image: infrav1.ImageParam{Filter: &infrav1.ImageFilter{Name: ptr.To("test-image")}},
			},
		}
		machine.Namespace = namespace.Name
		machine.GenerateName = "machine-"
		return machine
	}

	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should allow the smallest permissible machine spec", func() {
		Expect(k8sClient.Create(ctx, defaultMachine())).To(Succeed(), "OpenStackMachine creation should succeed")
	})

	It("should only allow the providerID to be set once", func() {
		machine := defaultMachine()

		By("Creating a bare machine")
		Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "OpenStackMachine creation should succeed")

		By("Setting the providerID")
		machine.Spec.ProviderID = ptr.To("foo")
		Expect(k8sClient.Update(ctx, machine)).To(Succeed(), "Setting providerID should succeed")

		By("Modifying the providerID")
		machine.Spec.ProviderID = ptr.To("bar")
		Expect(k8sClient.Update(ctx, machine)).NotTo(Succeed(), "Updating providerID should fail")
	})

	It("should not allow server metadata to exceed 255 characters", func() {
		machine := defaultMachine()

		By("Creating a machine with a metadata key that is too long")
		machine.Spec.ServerMetadata = []infrav1.ServerMetadata{
			{
				Key:   strings.Repeat("a", 256),
				Value: "value",
			},
		}
		Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with a long metadata key should fail")

		By("Creating a machine with a metadata value that is too long")
		machine.Spec.ServerMetadata = []infrav1.ServerMetadata{
			{
				Key:   "key",
				Value: strings.Repeat("a", 256),
			},
		}
		Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with a long metadata value should fail")

		By("Creating a machine with a metadata key and value of 255 characters should succeed")
		machine.Spec.ServerMetadata = []infrav1.ServerMetadata{
			{
				Key:   strings.Repeat("a", 255),
				Value: strings.Repeat("b", 255),
			},
		}
		Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "Creating a machine with max metadata key and value should succeed")
	})

	Context("volumes", func() {
		It("should not allow volume with zero size", func() {
			machine := defaultMachine()
			machine.Spec.RootVolume = &infrav1.RootVolume{
				SizeGiB: 0,
			}
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with a zero size root volume should fail")

			machine = defaultMachine()
			machine.Spec.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
				{
					Name:    "test-volume",
					SizeGiB: 0,
				},
			}
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with a zero size additional block device should fail")
		})

		/* FIXME: These tests are failing
		It("should not allow additional volume with empty name", func() {
			machine.Spec.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
				{
					Name:    "",
					SizeGiB: 1,
				},
			}
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with an empty name additional block device should fail")
		})

		It("should not allow additional volume with name root", func() {
			machine.Spec.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
				{
					Name:    "root",
					SizeGiB: 1,
				},
			}
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with a root named additional block device should fail")
		})
		*/

		It("should not allow additional volume with duplicate name", func() {
			machine := defaultMachine()
			machine.Spec.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
				{
					Name:    "test-volume",
					SizeGiB: 1,
				},
				{
					Name:    "test-volume",
					SizeGiB: 2,
				},
			}
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with duplicate named additional block device should fail")
		})

		defaultMachineWithRootVolumeAZ := func(az *infrav1.VolumeAvailabilityZone) *infrav1.OpenStackMachine {
			machine := defaultMachine()
			machine.Spec.RootVolume = &infrav1.RootVolume{
				SizeGiB: 1,
			}
			machine.Spec.RootVolume.AvailabilityZone = az
			return machine
		}

		defaultMachineWithAdditionBlockDeviceAZ := func(az *infrav1.VolumeAvailabilityZone) *infrav1.OpenStackMachine {
			machine := defaultMachine()
			machine.Spec.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
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
			return machine
		}

		It("should allow volume with defaulted AZ from", func() {
			azName := infrav1.VolumeAZName("test-az")
			az := infrav1.VolumeAvailabilityZone{
				Name: &azName,
			}

			machine := defaultMachineWithRootVolumeAZ(&az)
			Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "Creating a machine with a root volume with an availability zone should succeed")

			machine = defaultMachineWithAdditionBlockDeviceAZ(&az)
			Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "Creating a machine with an additional block device with an availability zone should succeed")
		})

		It("should allow volume with AZ from Name", func() {
			azName := infrav1.VolumeAZName("test-az")
			az := infrav1.VolumeAvailabilityZone{
				From: infrav1.VolumeAZFromName,
				Name: &azName,
			}

			machine := defaultMachineWithRootVolumeAZ(&az)
			Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "Creating a machine with a root volume with an availability zone should succeed")

			machine = defaultMachineWithAdditionBlockDeviceAZ(&az)
			Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "Creating a machine with an additional block device with an availability zone should succeed")
		})

		It("should allow volume AZ from Machine", func() {
			az := infrav1.VolumeAvailabilityZone{
				From: infrav1.VolumeAZFromMachine,
			}

			machine := defaultMachineWithRootVolumeAZ(&az)
			Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "Creating a machine with a root volume with an availability zone should succeed")

			machine = defaultMachineWithAdditionBlockDeviceAZ(&az)
			Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "Creating a machine with an additional block device with an availability zone should succeed")
		})

		It("should not allow volume AZ with invalid from", func() {
			az := infrav1.VolumeAvailabilityZone{
				From: "invalid",
			}

			machine := defaultMachineWithRootVolumeAZ(&az)
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with a root volume with an invalid availability zone should fail")

			machine = defaultMachineWithAdditionBlockDeviceAZ(&az)
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with an additional block device with an invalid availability zone should fail")
		})

		It("should not allow empty volume AZ", func() {
			az := infrav1.VolumeAvailabilityZone{}

			machine := defaultMachineWithRootVolumeAZ(&az)
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with a root volume with an empty availability zone should fail")

			machine = defaultMachineWithAdditionBlockDeviceAZ(&az)
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with an additional block device with an empty availability zone should fail")
		})

		It("should not allow volume AZ from Name with missing name", func() {
			az := infrav1.VolumeAvailabilityZone{
				From: infrav1.VolumeAZFromName,
			}

			machine := defaultMachineWithRootVolumeAZ(&az)
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with a root volume with a missing name availability zone should fail")

			machine = defaultMachineWithAdditionBlockDeviceAZ(&az)
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with an additional block device with a missing name availability zone should fail")
		})

		It("should not allow volume AZ from Machine with name", func() {
			azName := infrav1.VolumeAZName("test-az")
			az := infrav1.VolumeAvailabilityZone{
				From: infrav1.VolumeAZFromMachine,
				Name: &azName,
			}

			machine := defaultMachineWithRootVolumeAZ(&az)
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with a root volume with a name availability zone should fail")

			machine = defaultMachineWithAdditionBlockDeviceAZ(&az)
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with an additional block device with a name availability zone should fail")
		})

		It("should not allow volume AZ from Name with empty name", func() {
			azName := infrav1.VolumeAZName("")
			az := infrav1.VolumeAvailabilityZone{
				From: infrav1.VolumeAZFromName,
				Name: &azName,
			}

			machine := defaultMachineWithRootVolumeAZ(&az)
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with a root volume with an empty name availability zone should fail")

			machine = defaultMachineWithAdditionBlockDeviceAZ(&az)
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with an additional block device with an empty name availability zone should fail")
		})
	})
})
