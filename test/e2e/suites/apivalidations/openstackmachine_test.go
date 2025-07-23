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
				Image:  infrav1.ImageParam{Filter: &infrav1.ImageFilter{Name: ptr.To("test-image")}},
				Flavor: ptr.To("flavor-name"),
			},
		}
		machine.Namespace = namespace.Name
		machine.GenerateName = "machine-"
		return machine
	}

	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should allow to create a machine with correct spec", func() {
		machine := defaultMachine()

		By("Creating the smallest permissible machine spec")
		Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "OpenStackMachine creation should succeed")

		machine = defaultMachine()
		machine.Spec.IdentityRef = &infrav1.OpenStackIdentityReference{Name: "foobar", CloudName: "staging"}

		By("Creating a machine with spec.identityRef")
		Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "OpenStackMachine creation with spec.identityRef should succeed")
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

	It("should allow the identityRef to be set several times", func() {
		machine := defaultMachine()

		By("Creating a bare machine")
		Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "OpenStackMachine creation should succeed")

		By("Setting the identityRef")
		machine.Spec.IdentityRef = ptr.To(infrav1.OpenStackIdentityReference{Name: "foo", CloudName: "staging"})
		Expect(k8sClient.Update(ctx, machine)).To(Succeed(), "Setting the identityRef should succeed")

		By("Updating the identityRef.Name")
		machine.Spec.IdentityRef = ptr.To(infrav1.OpenStackIdentityReference{Name: "bar", CloudName: "staging"})
		Expect(k8sClient.Update(ctx, machine)).To(Succeed(), "Updating the identityRef.Name should succeed")

		By("Updating the identityRef.CloudName")
		machine.Spec.IdentityRef = ptr.To(infrav1.OpenStackIdentityReference{Name: "bar", CloudName: "production"})
		Expect(k8sClient.Update(ctx, machine)).To(Succeed(), "Updating the identityRef.CloudName should succeed")

		By("Clearing the identityRef")
		machine.Spec.IdentityRef = nil
		Expect(k8sClient.Update(ctx, machine)).To(Succeed(), "Clearing the identityRef should succeed")
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

	It("should allow server identityRef Region field or unset on creation", func() {
		machine := defaultMachine()

		By("Creating a machine with identityRef Region field set on creation")
		machine.Spec.IdentityRef = &infrav1.OpenStackIdentityReference{
			Name:      "secretName",
			CloudName: "cloudName",
			Region:    "regionName",
		}
		Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "Creating a machine with a spec.identityRef.Region field should be allowed")

		By("Updating the identityRef Region field")
		machine.Spec.IdentityRef.Region = "anotherRegionName"
		Expect(k8sClient.Update(ctx, machine)).NotTo(Succeed(), "Updating spec.identityRef.Region field should fail")

		machine = defaultMachine()
		By("Creating a machine with identityRef Region field not set on creation")
		machine.Spec.IdentityRef = &infrav1.OpenStackIdentityReference{
			Name:      "secretName",
			CloudName: "cloudName",
		}
		Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "Creating a machine without a spec.identityRef.Region field should be allowed")

		By("Setting the identityRef Region field")
		machine.Spec.IdentityRef.Region = "regionName"
		Expect(k8sClient.Update(ctx, machine)).NotTo(Succeed(), "Setting spec.identityRef.Region field should fail")
	})

	Context("flavors", func() {
		It("should require either a flavor or flavorID", func() {
			machine := defaultMachine()

			By("Creating a machine with no flavor or flavor id")
			machine.Spec.Flavor = nil
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with no flavor name or id should fail")

			By("Creating a machine with a flavor id")
			machine.Spec.FlavorID = ptr.To("6aa02f56-c595-4d2f-9f8e-3c6296a4bed9")
			Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "Creating a machine with a flavor id should succeed")
		})
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

		It("should allow to create machine with spec.RootVolume and non-root device name in spec.AdditionalBlockDevices", func() {
			machine := defaultMachine()
			machine.Spec.RootVolume = &infrav1.RootVolume{SizeGiB: 50, BlockDeviceVolume: infrav1.BlockDeviceVolume{}}
			machine.Spec.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
				{Name: "user", SizeGiB: 30, Storage: infrav1.BlockDeviceStorage{}},
			}

			By("Creating a machine with spec.RootVolume and non-root device name in spec.AdditionalBlockDevices")
			Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "OpenStackMachine creation with non-root device name in spec.AdditionalBlockDevices should succeed")
		})

		It("should not allow to create machine with spec.RootVolume and root device name in spec.AdditionalBlockDevices", func() {
			machine := defaultMachine()
			machine.Spec.RootVolume = &infrav1.RootVolume{SizeGiB: 50, BlockDeviceVolume: infrav1.BlockDeviceVolume{}}
			machine.Spec.AdditionalBlockDevices = []infrav1.AdditionalBlockDevice{
				{Name: "root", SizeGiB: 30, Storage: infrav1.BlockDeviceStorage{}},
			}

			By("Creating a machine with spec.RootVolume and root device name in spec.AdditionalBlockDevices")
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation with root device name in spec.AdditionalBlockDevices should not succeed")
		})

		It("should not allow to create machine with both SecurityGroups and DisablePortSecurity", func() {
			machine := defaultMachine()
			machine.Spec.Ports = []infrav1.PortOpts{
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

			By("Creating a machine with both SecurityGroups and DisablePortSecurity")
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation with both SecurityGroups and DisablePortSecurity should not succeed")
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

	Context("schedulerHints", func() {
		It("should allow empty schedulerHints", func() {
			machine := defaultMachine()
			machine.Spec.SchedulerHintAdditionalProperties = []infrav1.SchedulerHintAdditionalProperty{}
			Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "Creating a machine with an empty SchedulerHintAdditionalProperties should succeed.")
		})

		It("should not allow item with empty name", func() {
			machine := defaultMachine()
			machine.Spec.SchedulerHintAdditionalProperties = []infrav1.SchedulerHintAdditionalProperty{
				{
					Name: "",
					Value: infrav1.SchedulerHintAdditionalValue{
						Type: infrav1.SchedulerHintTypeBool,
						Bool: ptr.To(false),
					},
				},
			}
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with SchedulerHintAdditionalProperties including an item with empty name should fail.")
		})

		It("should not allow item with empty value", func() {
			machine := defaultMachine()
			machine.Spec.SchedulerHintAdditionalProperties = []infrav1.SchedulerHintAdditionalProperty{
				{
					Name: "test-hints",
				},
			}
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "Creating a machine with SchedulerHintAdditionalProperties including an item with empty value should fail.")
		})

		It("should allow correct SchedulerHintAdditionalProperties", func() {
			machineB := defaultMachine()
			machineB.Spec.SchedulerHintAdditionalProperties = []infrav1.SchedulerHintAdditionalProperty{
				{
					Name: "test-hints",
					Value: infrav1.SchedulerHintAdditionalValue{
						Type: infrav1.SchedulerHintTypeBool,
						Bool: ptr.To(true),
					},
				},
			}
			By("Creating SchedulerHint with bool type")
			Expect(k8sClient.Create(ctx, machineB)).To(Succeed(), "Creating a machine with bool type scheduler hint property should succeed.")
			machineN := defaultMachine()
			machineN.Spec.SchedulerHintAdditionalProperties = []infrav1.SchedulerHintAdditionalProperty{
				{
					Name: "test-hints",
					Value: infrav1.SchedulerHintAdditionalValue{
						Type:   infrav1.SchedulerHintTypeNumber,
						Number: ptr.To(1),
					},
				},
			}
			By("Creating SchedulerHint with number type")
			Expect(k8sClient.Create(ctx, machineN)).To(Succeed(), "Creating a machine with number type scheduler hint property should succeed.")
			machineS := defaultMachine()
			machineS.Spec.SchedulerHintAdditionalProperties = []infrav1.SchedulerHintAdditionalProperty{
				{
					Name: "test-hints",
					Value: infrav1.SchedulerHintAdditionalValue{
						Type:   infrav1.SchedulerHintTypeString,
						String: ptr.To("test-hint"),
					},
				},
			}
			By("Creating SchedulerHint with string type")
			Expect(k8sClient.Create(ctx, machineS)).To(Succeed(), "Creating a machine with string type scheduler hint property should succeed.")
		})

		It("should not allow incorrect SchedulerHintAdditionalProperties with bool type", func() {
			machineBN := defaultMachine()
			machineBN.Spec.SchedulerHintAdditionalProperties = []infrav1.SchedulerHintAdditionalProperty{
				{
					Name: "test-hints",
					Value: infrav1.SchedulerHintAdditionalValue{
						Type:   infrav1.SchedulerHintTypeBool,
						Number: ptr.To(1),
					},
				},
			}
			By("Creating SchedulerHint with bool type and number value")
			Expect(k8sClient.Create(ctx, machineBN)).NotTo(Succeed(), "Creating a machine with bool type but number value scheduler hint property should fail.")
			machineBS := defaultMachine()
			machineBS.Spec.SchedulerHintAdditionalProperties = []infrav1.SchedulerHintAdditionalProperty{
				{
					Name: "test-hints",
					Value: infrav1.SchedulerHintAdditionalValue{
						Type:   infrav1.SchedulerHintTypeBool,
						String: ptr.To("test-hint"),
					},
				},
			}
			By("Creating SchedulerHint with bool type and string value")
			Expect(k8sClient.Create(ctx, machineBS)).NotTo(Succeed(), "Creating a machine with bool type but string value scheduler hint property should fail.")
		})

		It("should not allow incorrect SchedulerHintAdditionalProperties with number type", func() {
			machineNB := defaultMachine()
			machineNB.Spec.SchedulerHintAdditionalProperties = []infrav1.SchedulerHintAdditionalProperty{
				{
					Name: "test-hints",
					Value: infrav1.SchedulerHintAdditionalValue{
						Type: infrav1.SchedulerHintTypeNumber,
						Bool: ptr.To(true),
					},
				},
			}
			By("Creating SchedulerHint with number type and bool value")
			Expect(k8sClient.Create(ctx, machineNB)).NotTo(Succeed(), "Creating a machine with number type but bool value scheduler hint property should fail.")
			machineNS := defaultMachine()
			machineNS.Spec.SchedulerHintAdditionalProperties = []infrav1.SchedulerHintAdditionalProperty{
				{
					Name: "test-hints",
					Value: infrav1.SchedulerHintAdditionalValue{
						Type:   infrav1.SchedulerHintTypeNumber,
						String: ptr.To("test-hint"),
					},
				},
			}
			By("Creating SchedulerHint with number type and string value")
			Expect(k8sClient.Create(ctx, machineNS)).NotTo(Succeed(), "Creating a machine with number type but string value scheduler hint property should fail.")
		})

		It("should not allow incorrect SchedulerHintAdditionalProperties with string type", func() {
			machineSB := defaultMachine()
			machineSB.Spec.SchedulerHintAdditionalProperties = []infrav1.SchedulerHintAdditionalProperty{
				{
					Name: "test-hints",
					Value: infrav1.SchedulerHintAdditionalValue{
						Type: infrav1.SchedulerHintTypeString,
						Bool: ptr.To(true),
					},
				},
			}
			By("Creating SchedulerHint with string type and bool value")
			Expect(k8sClient.Create(ctx, machineSB)).NotTo(Succeed(), "Creating a machine with string type but bool value scheduler hint property should fail.")
			machineSN := defaultMachine()
			machineSN.Spec.SchedulerHintAdditionalProperties = []infrav1.SchedulerHintAdditionalProperty{
				{
					Name: "test-hints",
					Value: infrav1.SchedulerHintAdditionalValue{
						Type:   infrav1.SchedulerHintTypeString,
						Number: ptr.To(1),
					},
				},
			}
			By("Creating SchedulerHint with string type and number value")
			Expect(k8sClient.Create(ctx, machineSN)).NotTo(Succeed(), "Creating a machine with string type but number value scheduler hint property should fail.")
		})
	})
})
