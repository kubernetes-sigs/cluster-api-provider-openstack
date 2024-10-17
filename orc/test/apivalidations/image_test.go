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
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	applyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	imageID = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae12f"
)

func imageStub(name string, namespace *corev1.Namespace) *orcv1alpha1.Image {
	obj := &orcv1alpha1.Image{}
	obj.Name = name
	obj.Namespace = namespace.Name
	return obj
}

func testCredentials() *applyconfigv1alpha1.CloudCredentialsReferenceApplyConfiguration {
	return applyconfigv1alpha1.CloudCredentialsReference().
		WithSecretName("openstack-credentials").
		WithCloudName("openstack")
}

func testResource() *applyconfigv1alpha1.ImageResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.ImageResourceSpec().
		WithContent(applyconfigv1alpha1.ImageContent().
			WithContainerFormat(orcv1alpha1.ImageContainerFormatBare).
			WithDiskFormat(orcv1alpha1.ImageDiskFormatQCOW2).
			WithDownload(applyconfigv1alpha1.ImageContentSourceDownload().
				WithURL("https://example.com/example.img")))

}

func basePatch(image client.Object) *applyconfigv1alpha1.ImageApplyConfiguration {
	return applyconfigv1alpha1.Image(image.GetName(), image.GetNamespace()).
		WithSpec(applyconfigv1alpha1.ImageSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func minimalManagedPatch(orcImage client.Object) *applyconfigv1alpha1.ImageApplyConfiguration {
	patch := basePatch(orcImage)
	patch.Spec.WithResource(testResource())
	return patch
}

func testImport() *applyconfigv1alpha1.ImageImportApplyConfiguration {
	return applyconfigv1alpha1.ImageImport().WithID(imageID)
}

type getWithFn[argType, returnType any] func(*applyconfigv1alpha1.ImageApplyConfiguration) func(argType) returnType

func testMutability[argType, returnType any](ctx context.Context, namespace *corev1.Namespace, getFn getWithFn[argType, returnType], valueA, valueB argType, allowsUnset bool, initFns ...func(*applyconfigv1alpha1.ImageApplyConfiguration)) {
	setup := func(name string) (client.Object, *applyconfigv1alpha1.ImageApplyConfiguration, func(argType) returnType) {
		obj := imageStub(name, namespace)
		patch := minimalManagedPatch(obj)
		for _, initFn := range initFns {
			initFn(patch)
		}
		withFn := getFn(patch)

		return obj, patch, withFn
	}

	if allowsUnset {
		obj, patch, withFn := setup("unset")

		Expect(applyObj(ctx, obj, patch)).To(Succeed(), fmt.Sprintf("create with value unset: %s", format.Object(patch, 2)))

		withFn(valueA)
		Expect(applyObj(ctx, obj, patch)).NotTo(Succeed(), fmt.Sprintf("update with value set: %s", format.Object(patch, 2)))
	}

	obj, patch, withFn := setup("modify")

	withFn(valueA)
	Expect(applyObj(ctx, obj, patch)).To(Succeed(), fmt.Sprintf("create with value '%v': %s", valueA, format.Object(patch, 2)))

	withFn(valueB)
	Expect(applyObj(ctx, obj, patch)).NotTo(Succeed(), fmt.Sprintf("update with value '%v': %s", valueB, format.Object(patch, 2)))
}

var _ = Describe("ORC Image API validations", func() {
	var namespace *corev1.Namespace

	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should allow to create a minimal image", func(ctx context.Context) {
		image := imageStub("image", namespace)
		minimalPatch := minimalManagedPatch(image)

		Expect(applyObj(ctx, image, minimalPatch)).To(Succeed())
	})

	It("should default to managementPolicy managed", func(ctx context.Context) {
		image := imageStub("image", namespace)
		image.Spec.Resource = &orcv1alpha1.ImageResourceSpec{
			Content: &orcv1alpha1.ImageContent{
				DiskFormat: orcv1alpha1.ImageDiskFormatQCOW2,
				Download: &orcv1alpha1.ImageContentSourceDownload{
					URL: "https://example.com/example.img",
				},
			},
		}
		image.Spec.CloudCredentialsRef = orcv1alpha1.CloudCredentialsReference{
			SecretName: "my-secret",
			CloudName:  "my-cloud",
		}

		Expect(k8sClient.Create(ctx, image)).To(Succeed())
		Expect(image.Spec.ManagementPolicy).To(Equal(orcv1alpha1.ManagementPolicyManaged))
	})

	It("should require import for unmanaged", func(ctx context.Context) {
		image := imageStub("image", namespace)
		patch := basePatch(image)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed())

		patch.Spec.WithImport(testImport())
		Expect(applyObj(ctx, image, patch)).To(Succeed())
	})

	It("should not permit unmanaged with resource", func(ctx context.Context) {
		image := imageStub("image", namespace)
		patch := basePatch(image)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(testImport()).
			WithResource(testResource())
	})

	It("should not permit empty import", func(ctx context.Context) {
		image := imageStub("image", namespace)
		patch := basePatch(image)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.ImageImport())
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed())
	})

	It("should not permit empty import filter", func(ctx context.Context) {
		image := imageStub("image", namespace)
		patch := basePatch(image)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.ImageImport().
				WithFilter(applyconfigv1alpha1.ImageFilter()))
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed())
	})

	It("should permit import filter with name", func(ctx context.Context) {
		image := imageStub("image", namespace)
		patch := basePatch(image)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.ImageImport().
				WithFilter(applyconfigv1alpha1.ImageFilter().WithName("foo")))
		Expect(applyObj(ctx, image, patch)).To(Succeed())
	})

	It("should require resource for managed", func(ctx context.Context) {
		image := imageStub("image", namespace)
		patch := basePatch(image)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed())

		patch.Spec.WithResource(testResource())
		Expect(applyObj(ctx, image, patch)).To(Succeed())
	})

	It("should not permit managed with import", func(ctx context.Context) {
		image := imageStub("image", namespace)
		patch := basePatch(image)
		patch.Spec.
			WithImport(testImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged).
			WithResource(testResource())
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed())
	})

	It("should require content when not importing", func(ctx context.Context) {
		image := imageStub("image", namespace)
		patch := minimalManagedPatch(image)
		patch.Spec.WithResource(applyconfigv1alpha1.ImageResourceSpec())
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed())
	})

	It("should not permit managedOptions for unmanaged", func(ctx context.Context) {
		image := imageStub("image", namespace)
		patch := basePatch(image)
		patch.Spec.
			WithImport(testImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed())
	})

	It("should permit managedOptions for managed", func(ctx context.Context) {
		image := imageStub("image", namespace)
		patch := minimalManagedPatch(image)
		patch.Spec.
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, image, patch)).To(Succeed())
	})

	DescribeTable("should permit containerFormat",
		func(ctx context.Context, containerFormat orcv1alpha1.ImageContainerFormat) {
			image := imageStub("image", namespace)
			patch := minimalManagedPatch(image)
			patch.Spec.Resource.Content.WithContainerFormat(containerFormat)
			Expect(applyObj(ctx, image, patch)).To(Succeed(), "create image")
		},
		Entry(string(orcv1alpha1.ImageContainerFormatAKI), orcv1alpha1.ImageContainerFormatAKI),
		Entry(string(orcv1alpha1.ImageContainerFormatAMI), orcv1alpha1.ImageContainerFormatAMI),
		Entry(string(orcv1alpha1.ImageContainerFormatARI), orcv1alpha1.ImageContainerFormatARI),
		Entry(string(orcv1alpha1.ImageContainerFormatBare), orcv1alpha1.ImageContainerFormatBare),
		Entry(string(orcv1alpha1.ImageContainerFormatDocker), orcv1alpha1.ImageContainerFormatDocker),
		Entry(string(orcv1alpha1.ImageContainerFormatOVA), orcv1alpha1.ImageContainerFormatOVA),
		Entry(string(orcv1alpha1.ImageContainerFormatOVF), orcv1alpha1.ImageContainerFormatOVF),
	)

	It("should not permit invalid containerFormat", func(ctx context.Context) {
		image := imageStub("image", namespace)
		patch := minimalManagedPatch(image)
		patch.Spec.Resource.Content.WithContainerFormat("foo")
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed(), "create image")
	})

	DescribeTable("should permit diskFormat",
		func(ctx context.Context, diskFormat orcv1alpha1.ImageDiskFormat) {
			image := imageStub("image", namespace)
			patch := minimalManagedPatch(image)
			patch.Spec.Resource.Content.WithDiskFormat(diskFormat)
			Expect(applyObj(ctx, image, patch)).To(Succeed(), "create image")
		},
		Entry(string(orcv1alpha1.ImageDiskFormatAMI), orcv1alpha1.ImageDiskFormatAMI),
		Entry(string(orcv1alpha1.ImageDiskFormatARI), orcv1alpha1.ImageDiskFormatARI),
		Entry(string(orcv1alpha1.ImageDiskFormatAKI), orcv1alpha1.ImageDiskFormatAKI),
		Entry(string(orcv1alpha1.ImageDiskFormatVHD), orcv1alpha1.ImageDiskFormatVHD),
		Entry(string(orcv1alpha1.ImageDiskFormatVHDX), orcv1alpha1.ImageDiskFormatVHDX),
		Entry(string(orcv1alpha1.ImageDiskFormatVMDK), orcv1alpha1.ImageDiskFormatVMDK),
		Entry(string(orcv1alpha1.ImageDiskFormatRaw), orcv1alpha1.ImageDiskFormatRaw),
		Entry(string(orcv1alpha1.ImageDiskFormatQCOW2), orcv1alpha1.ImageDiskFormatQCOW2),
		Entry(string(orcv1alpha1.ImageDiskFormatVDI), orcv1alpha1.ImageDiskFormatVDI),
		Entry(string(orcv1alpha1.ImageDiskFormatPLoop), orcv1alpha1.ImageDiskFormatPLoop),
		Entry(string(orcv1alpha1.ImageDiskFormatISO), orcv1alpha1.ImageDiskFormatISO),
	)

	It("should not permit invalid diskFormat", func(ctx context.Context) {
		image := imageStub("image", namespace)
		patch := minimalManagedPatch(image)
		patch.Spec.Resource.Content.WithDiskFormat("foo")
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed(), "create image")
	})

	It("should not permit modifying resource.name", func(ctx context.Context) {
		testMutability(ctx, namespace,
			func(applyConfig *applyconfigv1alpha1.ImageApplyConfiguration) func(string) *applyconfigv1alpha1.ImageResourceSpecApplyConfiguration {
				return applyConfig.Spec.Resource.WithName
			},
			"foo", "bar", true,
		)
	})

	It("should not permit modifying resource.protected", func(ctx context.Context) {
		testMutability(ctx, namespace,
			func(applyConfig *applyconfigv1alpha1.ImageApplyConfiguration) func(bool) *applyconfigv1alpha1.ImageResourceSpecApplyConfiguration {
				return applyConfig.Spec.Resource.WithProtected
			}, true, false, true,
		)
	})

	It("should not permit modifying resource.tags", func(ctx context.Context) {
		testMutability(ctx, namespace,
			func(applyConfig *applyconfigv1alpha1.ImageApplyConfiguration) func(string) *applyconfigv1alpha1.ImageResourceSpecApplyConfiguration {
				return func(tag string) *applyconfigv1alpha1.ImageResourceSpecApplyConfiguration {
					return applyConfig.Spec.Resource.WithTags(orcv1alpha1.ImageTag(tag))
				}
			}, "foo", "bar", true,
		)
	})

	It("should not permit modifying resource.visibility", func(ctx context.Context) {
		testMutability(ctx, namespace,
			func(applyConfig *applyconfigv1alpha1.ImageApplyConfiguration) func(orcv1alpha1.ImageVisibility) *applyconfigv1alpha1.ImageResourceSpecApplyConfiguration {
				return applyConfig.Spec.Resource.WithVisibility
			}, orcv1alpha1.ImageVisibilityPublic, orcv1alpha1.ImageVisibilityPrivate, true,
		)
	})

	It("should not permit modifying resource.properties", func(ctx context.Context) {
		valueA := applyconfigv1alpha1.ImageProperties().WithMinDiskGB(1)
		valueB := applyconfigv1alpha1.ImageProperties().WithMinDiskGB(2)

		testMutability(ctx, namespace,
			func(applyConfig *applyconfigv1alpha1.ImageApplyConfiguration) func(*applyconfigv1alpha1.ImagePropertiesApplyConfiguration) *applyconfigv1alpha1.ImageResourceSpecApplyConfiguration {
				return applyConfig.Spec.Resource.WithProperties
			}, valueA, valueB, true,
		)
	})

	It("should not permit modifying resource.properties.hardware", func(ctx context.Context) {
		valueA := applyconfigv1alpha1.ImagePropertiesHardware().WithCPUCores(1)
		valueB := applyconfigv1alpha1.ImagePropertiesHardware().WithCPUCores(2)

		testMutability(ctx, namespace,
			func(applyConfig *applyconfigv1alpha1.ImageApplyConfiguration) func(*applyconfigv1alpha1.ImagePropertiesHardwareApplyConfiguration) *applyconfigv1alpha1.ImagePropertiesApplyConfiguration {
				return applyConfig.Spec.Resource.Properties.WithHardware
			}, valueA, valueB, true,
			func(patch *applyconfigv1alpha1.ImageApplyConfiguration) {
				patch.Spec.Resource.WithProperties(applyconfigv1alpha1.ImageProperties())
			},
		)
	})

	It("should not permit modifying resource.content.containerFormat", func(ctx context.Context) {
		testMutability(ctx, namespace,
			func(applyConfig *applyconfigv1alpha1.ImageApplyConfiguration) func(orcv1alpha1.ImageContainerFormat) *applyconfigv1alpha1.ImageContentApplyConfiguration {
				return func(fmt orcv1alpha1.ImageContainerFormat) *applyconfigv1alpha1.ImageContentApplyConfiguration {
					content := applyConfig.Spec.Resource.Content
					if content == nil {
						content = applyconfigv1alpha1.ImageContent().
							WithDiskFormat(orcv1alpha1.ImageDiskFormatQCOW2).
							WithDownload(applyconfigv1alpha1.ImageContentSourceDownload().
								WithURL("https://example.com/image.qcow2"))
						applyConfig.Spec.Resource.Content = content
					}
					return content.WithContainerFormat(fmt)
				}
			}, orcv1alpha1.ImageContainerFormatAKI, orcv1alpha1.ImageContainerFormatAMI, false,
			func(patch *applyconfigv1alpha1.ImageApplyConfiguration) {
				patch.Spec.Resource.Content = nil
			},
		)
	})

	It("should not permit modifying resource.content.download", func(ctx context.Context) {
		testMutability(ctx, namespace,
			func(applyConfig *applyconfigv1alpha1.ImageApplyConfiguration) func(string) *applyconfigv1alpha1.ImageContentSourceDownloadApplyConfiguration {
				return applyConfig.Spec.Resource.Content.Download.WithURL
			}, "https://example.com/image1.qcow2", "https://example.com/image2.qcow2", false,
		)
	})
})
