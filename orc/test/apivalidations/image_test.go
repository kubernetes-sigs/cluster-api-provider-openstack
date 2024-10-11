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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	corev1 "k8s.io/api/core/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	applyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	imageID = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae12f"
)

var _ = Describe("ORC Image API validations", func() {
	var namespace *corev1.Namespace

	imageStub := func(name string) *orcv1alpha1.Image {
		obj := &orcv1alpha1.Image{}
		obj.Name = name
		obj.Namespace = namespace.Name
		return obj
	}

	testCredentials := func() *applyconfigv1alpha1.CloudCredentialsReferenceApplyConfiguration {
		return applyconfigv1alpha1.CloudCredentialsReference().
			WithSecretName("openstack-credentials").
			WithCloudName("openstack")
	}

	testResource := func() *applyconfigv1alpha1.ImageResourceSpecApplyConfiguration {
		return applyconfigv1alpha1.ImageResourceSpec().
			WithContent(applyconfigv1alpha1.ImageContent().
				WithContainerFormat(orcv1alpha1.ImageContainerFormatBare).
				WithDiskFormat(orcv1alpha1.ImageDiskFormatQCOW2).
				WithSourceType(orcv1alpha1.ImageSourceTypeURL).
				WithSourceURL(applyconfigv1alpha1.ImageContentSourceURL().
					WithURL("https://example.com/example.img")))

	}

	basePatch := func(image *orcv1alpha1.Image) *applyconfigv1alpha1.ImageApplyConfiguration {
		return applyconfigv1alpha1.Image(image.Name, image.Namespace).
			WithSpec(applyconfigv1alpha1.ImageSpec().
				WithCloudCredentialsRef(testCredentials()))
	}

	minimalManagedPatch := func(orcImage *orcv1alpha1.Image) *applyconfigv1alpha1.ImageApplyConfiguration {
		patch := basePatch(orcImage)
		patch.Spec.WithResource(testResource())
		return patch
	}

	testImport := func() *applyconfigv1alpha1.ImageImportApplyConfiguration {
		return applyconfigv1alpha1.ImageImport().WithID(imageID)
	}

	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should allow to create a minimal image", func(ctx context.Context) {
		image := imageStub("image")
		minimalPatch := minimalManagedPatch(image)

		Expect(applyObj(ctx, image, minimalPatch)).To(Succeed())
	})

	It("should default to managementPolicy managed", func(ctx context.Context) {
		image := imageStub("image")
		image.Spec.Resource = &orcv1alpha1.ImageResourceSpec{
			Content: &orcv1alpha1.ImageContent{
				DiskFormat: orcv1alpha1.ImageDiskFormatQCOW2,
				SourceType: orcv1alpha1.ImageSourceTypeURL,
				SourceURL: &orcv1alpha1.ImageContentSourceURL{
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
		image := imageStub("image")
		patch := basePatch(image)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed())

		patch.Spec.WithImport(testImport())
		Expect(applyObj(ctx, image, patch)).To(Succeed())
	})

	It("should not permit unmanaged with resource", func(ctx context.Context) {
		image := imageStub("image")
		patch := basePatch(image)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(testImport()).
			WithResource(testResource())
	})

	It("should not permit empty import", func(ctx context.Context) {
		image := imageStub("image")
		patch := basePatch(image)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.ImageImport())
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed())
	})

	It("should not permit empty import filter", func(ctx context.Context) {
		image := imageStub("image")
		patch := basePatch(image)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.ImageImport().
				WithFilter(applyconfigv1alpha1.ImageFilter()))
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed())
	})

	It("should permit import filter with name", func(ctx context.Context) {
		image := imageStub("image")
		patch := basePatch(image)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.ImageImport().
				WithFilter(applyconfigv1alpha1.ImageFilter().WithName("foo")))
		Expect(applyObj(ctx, image, patch)).To(Succeed())
	})

	It("should require resource for managed", func(ctx context.Context) {
		image := imageStub("image")
		patch := basePatch(image)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed())

		patch.Spec.WithResource(testResource())
		Expect(applyObj(ctx, image, patch)).To(Succeed())
	})

	It("should not permit managed with import", func(ctx context.Context) {
		image := imageStub("image")
		patch := basePatch(image)
		patch.Spec.
			WithImport(testImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged).
			WithResource(testResource())
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed())
	})

	It("should require content when not importing", func(ctx context.Context) {
		image := imageStub("image")
		patch := minimalManagedPatch(image)
		patch.Spec.WithResource(applyconfigv1alpha1.ImageResourceSpec())
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed())
	})

	It("should not permit managedOptions for unmanaged", func(ctx context.Context) {
		image := imageStub("image")
		patch := basePatch(image)
		patch.Spec.
			WithImport(testImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed())
	})

	It("should permit managedOptions for managed", func(ctx context.Context) {
		image := imageStub("image")
		patch := minimalManagedPatch(image)
		patch.Spec.
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, image, patch)).To(Succeed())
	})

	DescribeTable("should permit containerFormat",
		func(ctx context.Context, containerFormat orcv1alpha1.ImageContainerFormat) {
			image := imageStub("image")
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
		image := imageStub("image")
		patch := minimalManagedPatch(image)
		patch.Spec.Resource.Content.WithContainerFormat("foo")
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed(), "create image")
	})

	DescribeTable("should permit diskFormat",
		func(ctx context.Context, diskFormat orcv1alpha1.ImageDiskFormat) {
			image := imageStub("image")
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
		image := imageStub("image")
		patch := minimalManagedPatch(image)
		patch.Spec.Resource.Content.WithDiskFormat("foo")
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed(), "create image")
	})

	DescribeTable("should not permit modifying immutable fields",
		func(ctx context.Context, patchA, patchB func(*applyconfigv1alpha1.ImageSpecApplyConfiguration)) {
			image := imageStub("image")
			patch := minimalManagedPatch(image)
			patchA(patch.Spec)
			Expect(applyObj(ctx, image, patch)).To(Succeed(), format.Object(patch, 2))

			patch = minimalManagedPatch(image)
			patchB(patch.Spec)
			Expect(applyObj(ctx, image, patch)).NotTo(Succeed(), "modify image")
		},
		Entry("imageName", func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.Resource.WithName("foo")
		}, func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.Resource.WithName("bar")
		}),
		Entry("protected", func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.Resource.WithProtected(true)
		}, func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.Resource.WithProtected(false)
		}),
		Entry("tags", func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.Resource.WithTags("foo")
		}, func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.Resource.WithTags("bar")
		}),
		Entry("visibility", func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.Resource.WithVisibility("public")
		}, func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.Resource.WithVisibility("private")
		}),
		Entry("properties", func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.Resource.WithProperties(applyconfigv1alpha1.ImageProperties().WithMinDiskGB(1))
		}, func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.Resource.WithProperties(applyconfigv1alpha1.ImageProperties().WithMinDiskGB(2))
		}),
		Entry("content", func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.Resource.WithContent(applyconfigv1alpha1.ImageContent().
				WithDiskFormat("qcow2").
				WithSourceType(orcv1alpha1.ImageSourceTypeURL).
				WithSourceURL(applyconfigv1alpha1.ImageContentSourceURL().
					WithURL("https://example.com/image1.img")))
		}, func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.Resource.WithContent(applyconfigv1alpha1.ImageContent().
				WithDiskFormat("qcow2").
				WithSourceType(orcv1alpha1.ImageSourceTypeURL).
				WithSourceURL(applyconfigv1alpha1.ImageContentSourceURL().
					WithURL("https://example.com/image2.img")))
		}),
	)
})
