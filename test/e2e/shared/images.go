//go:build e2e
// +build e2e

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

package shared

import (
	"context"
	"encoding/base64"
	neturl "net/url"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"

	"sigs.k8s.io/cluster-api-provider-openstack/internal/util/ssa"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

const (
	// The base URL of the CAPO staging artifacts bucket.
	stagingArtifactBase = "https://storage.googleapis.com/artifacts.k8s-staging-capi-openstack.appspot.com/test/"

	// The name of the credentials secret.
	credentialsSecretName = "openstack-credentials" //nolint:gosec // these aren't hard-coded credentials

	// A tag which will be added to all glances images created by these tests.
	E2EImageTag = "capo-e2e"
)

type DownloadImage struct {
	// The name of the ORC Image object
	Name string
	// An absolute URL to download the image from
	URL string
	// A path relative to the staging artifact repo to download the image from
	ArtifactPath string
	// A hash used to verify the downloaded image
	Hash, HashAlgorithm string
}

func CoreImages(e2eCtx *E2EContext) []DownloadImage {
	return []DownloadImage{
		{
			Name:         "cirros",
			ArtifactPath: "cirros/2022-12-05/" + e2eCtx.E2EConfig.GetVariable("OPENSTACK_BASTION_IMAGE_NAME") + ".img",

			// Specifying an image hash means we can't use
			// web-download. This serves as an E2E test of image
			// upload. We only do this with cirros because it's very
			// small and it's going to involve data transfer between
			// the test runner and wherever OpenStack is running
			HashAlgorithm: string(orcv1alpha1.ImageHashAlgorithmMD5),
			// From: https://download.cirros-cloud.net/0.6.1/MD5SUMS
			Hash: "0c839612eb3f2469420f2ccae990827f",
		},
		{
			Name:         "capo-default",
			ArtifactPath: "ubuntu/2024-11-21/" + e2eCtx.E2EConfig.GetVariable("OPENSTACK_IMAGE_NAME") + ".img",
		},
	}
}

func ApplyCoreImagesPlus(ctx context.Context, e2eCtx *E2EContext, additionalImages ...DownloadImage) {
	coreImages := CoreImages(e2eCtx)
	allImages := slices.Concat(coreImages, additionalImages)

	ApplyGlanceImages(ctx, e2eCtx, allImages)
}

func CreateGlanceCredentials(ctx context.Context, e2eCtx *E2EContext) {
	k8sClient := e2eCtx.Environment.BootstrapClusterProxy.GetClient()

	// Generate the credentials secret each image will reference
	credentialsSecret := generateCredentialsSecret(e2eCtx)
	Expect(k8sClient.Create(ctx, credentialsSecret)).To(Succeed(), "create openstack-credentials in default namespace")
}

func imageNames(images []DownloadImage) string {
	names := make([]string, len(images))
	for i := range images {
		names[i] = images[i].Name
	}
	return strings.Join(names, ", ")
}

// ApplyGlanceImages creates ORC Images corresponding to the given set of DownloadImages.
// Note that it does not wait for the images to become available.
func ApplyGlanceImages(ctx context.Context, e2eCtx *E2EContext, images []DownloadImage) {
	By("Applying glances images: " + imageNames(images))

	k8sClient := e2eCtx.Environment.BootstrapClusterProxy.GetClient()

	for _, image := range images {
		// Infer the url if none was specified
		url := image.URL
		if url == "" {
			url = stagingArtifactBase + image.ArtifactPath
		}

		// Infer the name to use for the glance image

		// Use last part of url
		u, err := neturl.Parse(url)
		Expect(err).NotTo(HaveOccurred(), "parsing "+url)
		d := strings.Split(u.Path, "/")
		Expect(len(d)).To(BeNumerically(">", 1), "Not enough path elements in "+url)
		glanceName := d[len(d)-1]

		// Remove the type suffix
		for _, suffix := range []string{".img", ".qcow2"} {
			if strings.HasSuffix(glanceName, suffix) {
				glanceName = glanceName[:len(glanceName)-len(suffix)]
				continue
			}
		}

		var imageHash *orcv1alpha1.ImageHash
		if image.HashAlgorithm != "" && image.Hash != "" {
			imageHash = &orcv1alpha1.ImageHash{
				Algorithm: orcv1alpha1.ImageHashAlgorithm(image.HashAlgorithm),
				Value:     image.Hash,
			}
		}

		// Generate and create the image
		orcImage, applyConfig := generateORCImage(e2eCtx, image.Name, glanceName, url, imageHash)
		Logf("Ensuring glance image " + image.Name)
		Expect(k8sClient.Patch(ctx, orcImage, ssa.ApplyConfigPatch(applyConfig), client.ForceOwnership, client.FieldOwner("capo-e2e"))).To(Succeed(), "ensure image "+image.Name)
	}
}

// WaitForGlanceImagesAvailable waits for the ORC Images described by the given set of DownloadImages to become available.
func WaitForGlanceImagesAvailable(ctx context.Context, e2eCtx *E2EContext, images []DownloadImage) {
	names := imageNames(images)
	By("Waiting for glance images to become available: " + names)

	k8sClient := e2eCtx.Environment.BootstrapClusterProxy.GetClient()

	Eventually(func(ctx context.Context) []string {
		By("Polling images")

		orcImageList := &orcv1alpha1.ImageList{}
		Expect(k8sClient.List(ctx, orcImageList, client.InNamespace("default"))).To(Succeed())

		var available []string
		var notAvailable []string
		for i := range images {
			imageName := images[i].Name
			image := func() *orcv1alpha1.Image {
				for j := range orcImageList.Items {
					item := &orcImageList.Items[j]
					if item.Name == imageName {
						return item
					}
				}
				return nil
			}()

			Expect(image).ToNot(BeNil(), "Did not find "+imageName+" in image list")

			availableCondition := meta.FindStatusCondition(image.Status.Conditions, orcv1alpha1.ConditionAvailable)
			if availableCondition == nil || availableCondition.Status != metav1.ConditionTrue {
				var msg string
				if availableCondition == nil {
					msg = "no status yet"
				} else {
					msg = availableCondition.Message
				}
				notAvailable = append(notAvailable, image.Name+": "+msg)
			} else {
				available = append(available, image.Name)
			}
		}
		Logf("Available: " + strings.Join(available, ", "))
		Logf("Not available: " + strings.Join(notAvailable, ", "))

		return notAvailable
	}, e2eCtx.E2EConfig.GetIntervals("default", "wait-image-create")...).WithContext(ctx).Should(BeEmpty(), "ORC Images are not available")

	Logf("Glance images became available: " + names)
}

func DeleteAllORCImages(ctx context.Context, e2eCtx *E2EContext) {
	k8sClient := e2eCtx.Environment.BootstrapClusterProxy.GetClient()

	By("Deleting glance images")
	Eventually(func() []orcv1alpha1.Image {
		By("Fetching remaining images")

		orcImageList := &orcv1alpha1.ImageList{}
		Expect(k8sClient.List(ctx, orcImageList, client.InNamespace("default"))).To(Succeed())
		images := orcImageList.Items

		for i := range images {
			image := &images[i]
			if image.GetDeletionTimestamp().IsZero() {
				Expect(k8sClient.Delete(ctx, image)).To(Succeed())
				Logf("Deleted ORC Image " + image.Name)
			} else {
				Logf("ORC Image " + image.Name + " is still deleting")
			}
		}

		return images
	}, e2eCtx.E2EConfig.GetIntervals("default", "wait-image-delete")...).Should(BeEmpty(), "ORC Images were not deleted")
}

func generateCredentialsSecret(e2eCtx *E2EContext) *corev1.Secret {
	// We run in Node1BeforeSuite, so OPENSTACK_CLOUD_YAML_B64 is not yet set
	openStackCloudYAMLFile := e2eCtx.E2EConfig.GetVariable(OpenStackCloudYAMLFile)
	caCertB64 := e2eCtx.E2EConfig.GetVariable(OpenStackCloudCACertB64)
	caCert, err := base64.StdEncoding.DecodeString(caCertB64)
	Expect(err).NotTo(HaveOccurred(), "base64 decode CA Cert: "+caCertB64)

	cloudsYAML := getOpenStackCloudYAML(openStackCloudYAMLFile)
	credentialsSecret := corev1.Secret{
		StringData: map[string]string{
			scope.CloudsSecretKey: string(cloudsYAML),
			scope.CASecretKey:     string(caCert),
		},
	}
	credentialsSecret.SetName(credentialsSecretName)
	credentialsSecret.SetNamespace("default")

	return &credentialsSecret
}

func generateORCImage(e2eCtx *E2EContext, name, glanceName, url string, downloadHash *orcv1alpha1.ImageHash) (*orcv1alpha1.Image, *orcapplyconfigv1alpha1.ImageApplyConfiguration) {
	const imageNamespace = "default"

	applyConfig := orcapplyconfigv1alpha1.Image(name, imageNamespace).
		WithSpec(orcapplyconfigv1alpha1.ImageSpec().
			WithResource(orcapplyconfigv1alpha1.ImageResourceSpec().
				WithName(orcv1alpha1.OpenStackName(glanceName)).
				WithTags(E2EImageTag).
				WithContent(orcapplyconfigv1alpha1.ImageContent().
					WithContainerFormat(orcv1alpha1.ImageContainerFormatBare).
					WithDiskFormat(orcv1alpha1.ImageDiskFormatQCOW2).
					WithDownload(orcapplyconfigv1alpha1.ImageContentSourceDownload().
						WithURL(url)))).
			WithCloudCredentialsRef(orcapplyconfigv1alpha1.CloudCredentialsReference().
				WithSecretName(credentialsSecretName).
				WithCloudName(e2eCtx.E2EConfig.GetVariable("OPENSTACK_CLOUD"))))

	if downloadHash != nil {
		applyConfig.Spec.Resource.Content.Download.
			WithHash(orcapplyconfigv1alpha1.ImageHash().
				WithAlgorithm(downloadHash.Algorithm).
				WithValue(downloadHash.Value))
	}

	orcImage := &orcv1alpha1.Image{}
	orcImage.Name = name
	orcImage.Namespace = imageNamespace

	return orcImage, applyConfig
}
