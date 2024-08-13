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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	applyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

var _ = Describe("ORC Image API validations", func() {
	var namespace *corev1.Namespace

	imageStub := func(name string) *orcv1alpha1.Image {
		obj := &orcv1alpha1.Image{}
		obj.Name = name
		obj.Namespace = namespace.Name
		return obj
	}
	minimalPatch := func(orcImage *orcv1alpha1.Image) *applyconfigv1alpha1.ImageApplyConfiguration {
		return applyconfigv1alpha1.Image(orcImage.Name, orcImage.Namespace).
			WithSpec(applyconfigv1alpha1.ImageSpec().
				WithContent(applyconfigv1alpha1.ImageContent().
					WithContainerFormat(orcv1alpha1.ImageContainerFormatBare).
					WithDiskFormat(orcv1alpha1.ImageDiskFormatQCOW2).
					WithSourceType(orcv1alpha1.ImageSourceTypeURL).
					WithSourceURL(applyconfigv1alpha1.ImageContentSourceURL().
						WithURL("https://example.com/example.img"))).
				WithCloudCredentialsRef(applyconfigv1alpha1.CloudCredentialsReference().
					WithName("openstack-credentials").
					WithCloudName("openstack")))
	}

	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should allow to create a minimal image", func(ctx context.Context) {
		image := imageStub("image")
		minimalPatch := minimalPatch(image)

		Expect(applyObj(ctx, image, minimalPatch)).To(Succeed())
	})

	DescribeTable("should not allow invalid names",
		func(ctx context.Context, name string) {
			image := imageStub(name)
			minimalPatch := minimalPatch(image)

			Expect(applyObj(ctx, image, minimalPatch)).NotTo(Succeed())
		},
		Entry("longer than 63 characters", strings.Repeat("a", 64)),
		Entry("contains /", "a/a"),
	)

	It("should require contentSource when controllerOptions are defined and empty", func(ctx context.Context) {
		image := imageStub("image")
		patch := minimalPatch(image)
		patch.Spec.
			WithControllerOptions(applyconfigv1alpha1.ControllerOptions())
		Expect(applyObj(ctx, image, patch)).To(Succeed(), "with contentSource")

		patch.Spec.
			WithContent(nil)
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed(), "without contentSource")
	})

	It("should require contentSource when controllerOptions.onCreate is AdoptOrCreate", func(ctx context.Context) {
		image := imageStub("image-with")
		patch := minimalPatch(image)
		patch.Spec.
			WithControllerOptions(applyconfigv1alpha1.ControllerOptions().
				WithOnCreate(orcv1alpha1.ControllerOptionsOnCreateAdoptOrCreate))
		Expect(applyObj(ctx, image, patch)).To(Succeed(), "with contentSource")

		image = imageStub("image-without")
		patch.Spec.
			WithControllerOptions(applyconfigv1alpha1.ControllerOptions().
				WithOnCreate(orcv1alpha1.ControllerOptionsOnCreateAdoptOrCreate)).
			WithContent(nil)
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed(), "without contentSource")
	})

	It("should require contentSource is not set when controllerOptions.onCreate is Adopt", func(ctx context.Context) {
		image := imageStub("image-without")
		patch := minimalPatch(image)
		patch.Spec.
			WithControllerOptions(applyconfigv1alpha1.ControllerOptions().
				WithOnCreate(orcv1alpha1.ControllerOptionsOnCreateAdopt)).
			WithContent(nil)
		Expect(applyObj(ctx, image, patch)).To(Succeed(), "without contentSource")

		image = imageStub("image-with")
		patch = minimalPatch(image)
		patch.Spec.
			WithControllerOptions(applyconfigv1alpha1.ControllerOptions().
				WithOnCreate(orcv1alpha1.ControllerOptionsOnCreateAdopt))
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed(), "with contentSource")
	})

	DescribeTable("should permit containerFormat",
		func(ctx context.Context, containerFormat orcv1alpha1.ImageContainerFormat) {
			image := imageStub("image")
			patch := minimalPatch(image)
			patch.Spec.Content.WithContainerFormat(containerFormat)
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
		patch := minimalPatch(image)
		patch.Spec.Content.WithContainerFormat("foo")
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed(), "create image")
	})

	DescribeTable("should permit diskFormat",
		func(ctx context.Context, diskFormat orcv1alpha1.ImageDiskFormat) {
			image := imageStub("image")
			patch := minimalPatch(image)
			patch.Spec.Content.WithDiskFormat(diskFormat)
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
		patch := minimalPatch(image)
		patch.Spec.Content.WithDiskFormat("foo")
		Expect(applyObj(ctx, image, patch)).NotTo(Succeed(), "create image")
	})

	DescribeTable("should not permit modifying immutable fields",
		func(ctx context.Context, patchA, patchB func(*applyconfigv1alpha1.ImageSpecApplyConfiguration)) {
			image := imageStub("image")
			patch := minimalPatch(image)
			patchA(patch.Spec)
			Expect(applyObj(ctx, image, patch)).To(Succeed(), "create image")

			patch = minimalPatch(image)
			patchB(patch.Spec)
			Expect(applyObj(ctx, image, patch)).NotTo(Succeed(), "modify image")
		},
		Entry("imageName", func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.WithImageName("foo")
		}, func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.WithImageName("bar")
		}),
		Entry("protected", func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.WithProtected(true)
		}, func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.WithProtected(false)
		}),
		Entry("tags", func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.WithTags("foo")
		}, func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.WithTags("bar")
		}),
		Entry("visibility", func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.WithVisibility("public")
		}, func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.WithVisibility("private")
		}),
		Entry("properties", func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.WithProperties(applyconfigv1alpha1.ImageProperties().WithMinDiskGB(1))
		}, func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.WithProperties(applyconfigv1alpha1.ImageProperties().WithMinDiskGB(2))
		}),
		Entry("content", func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.WithContent(applyconfigv1alpha1.ImageContent().
				WithDiskFormat("qcow2").
				WithSourceType(orcv1alpha1.ImageSourceTypeURL).
				WithSourceURL(applyconfigv1alpha1.ImageContentSourceURL().
					WithURL("https://example.com/image1.img")))
		}, func(spec *applyconfigv1alpha1.ImageSpecApplyConfiguration) {
			spec.WithContent(applyconfigv1alpha1.ImageContent().
				WithDiskFormat("qcow2").
				WithSourceType(orcv1alpha1.ImageSourceTypeURL).
				WithSourceURL(applyconfigv1alpha1.ImageContentSourceURL().
					WithURL("https://example.com/image2.img")))
		}),
	)
})
