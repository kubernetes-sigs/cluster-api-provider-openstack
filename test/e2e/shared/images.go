//go:build e2e

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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
)

const (
	// A tag which will be added to all glances images created by these tests.
	E2EImageTag = "capo-e2e"
)

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
