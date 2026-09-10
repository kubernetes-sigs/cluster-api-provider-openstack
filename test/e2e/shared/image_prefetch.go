//go:build e2e

/*
Copyright 2026 The Kubernetes Authors.

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

package e2eshared

import (
	"context"
	"encoding/base64"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
)

const (
	// imagePrefetchNamespace is the namespace used to prefetch glance images
	// before any spec runs. It doesn't matter which namespace is used: ORC
	// identifies a Glance image by name, so any Image object anywhere that
	// references the same name will adopt the image downloaded here instead
	// of downloading it again.
	imagePrefetchNamespace = "default"

	// imagePrefetchSecretName is the name of the cloud-config secret created
	// solely to support prefetching images.
	imagePrefetchSecretName = "e2e-image-prefetch-cloud-config" //nolint:gosec // Not a credential, just an object name.
)

// imagePrefetchSpec describes a glance image used somewhere in the e2e suite
// which should be downloaded up front, together with the e2e config
// variables which describe it.
type imagePrefetchSpec struct {
	// name is used to build the ORC Image object's name. It doesn't need to
	// match the Glance image name.
	name string

	nameVariable          string
	urlVariable           string
	hashVariable          string
	hashAlgorithmVariable string
}

// imagesToPrefetch lists every distinct glance image used anywhere in the
// e2e test suite. Keep this in sync with the Image resources defined under
// test/e2e/data/kustomize/components/images, .../upgrade-from-images, and
// the flatcar/flatcar-sysext flavor patches.
var imagesToPrefetch = []imagePrefetchSpec{
	{
		name:         "node-image",
		nameVariable: OpenStackImageName,
		urlVariable:  "OPENSTACK_IMAGE_URL",
	},
	{
		name:                  "bastion-image",
		nameVariable:          "OPENSTACK_BASTION_IMAGE_NAME",
		urlVariable:           "OPENSTACK_BASTION_IMAGE_URL",
		hashVariable:          "OPENSTACK_BASTION_IMAGE_HASH",
		hashAlgorithmVariable: "OPENSTACK_BASTION_IMAGE_HASH_ALGORITHM",
	},
	{
		name:         "node-image-upgrade-from",
		nameVariable: "OPENSTACK_IMAGE_NAME_UPGRADE_FROM",
		urlVariable:  "OPENSTACK_IMAGE_URL_UPGRADE_FROM",
	},
	{
		name:         "flatcar-image",
		nameVariable: "OPENSTACK_FLATCAR_IMAGE_NAME",
		urlVariable:  "OPENSTACK_FLATCAR_IMAGE_URL",
	},
	{
		name:         "flatcar-sysext-image",
		nameVariable: "FLATCAR_IMAGE_NAME",
		urlVariable:  "FLATCAR_IMAGE_URL",
	},
}

// PrefetchNodeImages creates an ORC Image object for every glance image used
// anywhere in the e2e test suite, and waits for all of them to become
// Available, before any spec creates a workload cluster.
func PrefetchNodeImages(ctx context.Context, e2eCtx *E2EContext) {
	By("Prefetching glance images used by the e2e suite")
	defer By("Finished prefetching glance images used by the e2e suite")

	k8sClient := e2eCtx.Environment.BootstrapClusterProxy.GetClient()

	cloudYAMLFile := e2eCtx.E2EConfig.MustGetVariable(OpenStackCloudYAMLFile)
	cloudName := e2eCtx.E2EConfig.MustGetVariable(OpenStackCloud)

	cacert, err := base64.StdEncoding.DecodeString(e2eCtx.E2EConfig.MustGetVariable(OpenStackCloudCACertB64))
	Expect(err).NotTo(HaveOccurred())

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      imagePrefetchSecretName,
			Namespace: imagePrefetchNamespace,
		},
		Data: map[string][]byte{
			orcv1alpha1.CloudCredentialsConfigSecretKey: getOpenStackCloudYAML(cloudYAMLFile),
			orcv1alpha1.CloudCredencialsCASecretKey:     cacert,
		},
	}
	Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, secret))).To(Succeed())

	var prefetched []orcv1alpha1.Image
	for _, spec := range imagesToPrefetch {
		if !e2eCtx.E2EConfig.HasVariable(spec.nameVariable) || !e2eCtx.E2EConfig.HasVariable(spec.urlVariable) {
			Logf("Skipping image prefetch for %q: variables %q/%q are not set", spec.name, spec.nameVariable, spec.urlVariable)
			continue
		}

		download := &orcv1alpha1.ImageContentSourceDownload{
			URL: e2eCtx.E2EConfig.MustGetVariable(spec.urlVariable),
		}
		if spec.hashVariable != "" && e2eCtx.E2EConfig.HasVariable(spec.hashVariable) {
			download.Hash = &orcv1alpha1.ImageHash{
				Algorithm: orcv1alpha1.ImageHashAlgorithm(e2eCtx.E2EConfig.MustGetVariable(spec.hashAlgorithmVariable)),
				Value:     e2eCtx.E2EConfig.MustGetVariable(spec.hashVariable),
			}
		}

		image := &orcv1alpha1.Image{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "prefetch-" + spec.name,
				Namespace: imagePrefetchNamespace,
			},
			Spec: orcv1alpha1.ImageSpec{
				ManagementPolicy: orcv1alpha1.ManagementPolicyManaged,
				ManagedOptions: &orcv1alpha1.ManagedOptions{
					OnDelete: orcv1alpha1.OnDeleteDetach,
				},
				CloudCredentialsRef: orcv1alpha1.CloudCredentialsReference{
					SecretName: imagePrefetchSecretName,
					CloudName:  cloudName,
				},
				Resource: &orcv1alpha1.ImageResourceSpec{
					Name: ptr.To(orcv1alpha1.OpenStackName(e2eCtx.E2EConfig.MustGetVariable(spec.nameVariable))),
					Content: &orcv1alpha1.ImageContent{
						ContainerFormat: orcv1alpha1.ImageContainerFormatBare,
						DiskFormat:      orcv1alpha1.ImageDiskFormatQCOW2,
						Download:        download,
					},
				},
			},
		}

		Logf("Prefetching image %q (%s)", spec.name, e2eCtx.E2EConfig.MustGetVariable(spec.nameVariable))
		err := k8sClient.Create(ctx, image)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}
		prefetched = append(prefetched, *image)
	}

	if len(prefetched) == 0 {
		return
	}

	Eventually(func() error {
		for i := range prefetched {
			img := &orcv1alpha1.Image{}
			if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&prefetched[i]), img); err != nil {
				return err
			}
			if !meta.IsStatusConditionTrue(img.Status.Conditions, orcv1alpha1.ConditionAvailable) {
				return fmt.Errorf("image %s/%s is not yet Available", img.Namespace, img.Name)
			}
		}
		return nil
	}, e2eCtx.E2EConfig.GetIntervals(imagePrefetchNamespace, "wait-image-create")...).Should(Succeed(), "Prefetched images did not become available in time")
}
