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

package apivalidations

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1beta1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
)

var _ = Describe("Conversion data annotation", func() {
	var (
		namespace *corev1.Namespace
		cluster   *infrav1.OpenStackCluster
		user      client.Client
	)

	ownStatus := infrav1.OpenStackClusterStatus{
		Network: &infrav1.NetworkStatusWithSubnets{NetworkStatus: infrav1.NetworkStatus{ID: "own-network"}},
		Router:  &infrav1.Router{ID: "own-router"},
		Bastion: &infrav1.BastionStatus{ID: "own-bastion", FloatingIP: "192.0.2.10"},
	}

	BeforeEach(func() {
		namespace = createNamespace()

		cluster = &infrav1.OpenStackCluster{}
		cluster.Namespace = namespace.Name
		cluster.GenerateName = clusterNamePrefix
		cluster.Spec.IdentityRef = infrav1.OpenStackIdentityReference{Name: "creds", CloudName: "openstack"}
		Expect(createObj(cluster)).To(Succeed(), "OpenStackCluster creation should succeed")

		cluster.Status = ownStatus
		Expect(k8sClient.Status().Update(ctx, cluster)).To(Succeed(), "OpenStackCluster status update should succeed")

		By("Creating a user who can update openstackclusters but not openstackclusters/status")
		role := &rbacv1.Role{}
		role.Namespace = namespace.Name
		role.Name = "openstackcluster-editor"
		role.Rules = []rbacv1.PolicyRule{{
			APIGroups: []string{infrav1.GroupName},
			Resources: []string{"openstackclusters"},
			Verbs:     []string{"get", "list", "update", "patch"},
		}}
		Expect(k8sClient.Create(ctx, role)).To(Succeed())

		binding := &rbacv1.RoleBinding{}
		binding.Namespace = namespace.Name
		binding.Name = "openstackcluster-editor"
		binding.Subjects = []rbacv1.Subject{{Kind: rbacv1.UserKind, Name: "openstackcluster-editor", APIGroup: rbacv1.GroupName}}
		binding.RoleRef = rbacv1.RoleRef{Kind: "Role", Name: role.Name, APIGroup: rbacv1.GroupName}
		Expect(k8sClient.Create(ctx, binding)).To(Succeed())

		userCfg := rest.CopyConfig(cfg)
		userCfg.Impersonate = rest.ImpersonationConfig{UserName: "openstackcluster-editor"}
		var err error
		user, err = client.New(userCfg, client.Options{Scheme: testScheme})
		Expect(err).NotTo(HaveOccurred())

		statusPatch := client.RawPatch(types.MergePatchType, []byte(`{"status":{"router":{"id":"victim-router"}}}`))
		Expect(user.Status().Patch(ctx, cluster.DeepCopy(), statusPatch)).To(Satisfy(apierrors.IsForbidden),
			"Patching status without openstackclusters/status permission should be forbidden")
	})

	// updateWithV1beta1 makes a label-only update through the v1beta1 API and
	// returns the stored status afterwards.
	updateWithV1beta1 := func() infrav1.OpenStackClusterStatus {
		v1beta1Cluster := &infrav1beta1.OpenStackCluster{}
		Expect(user.Get(ctx, client.ObjectKeyFromObject(cluster), v1beta1Cluster)).To(Succeed())
		v1beta1Cluster.Labels = map[string]string{"touched": "yes"}
		Expect(user.Update(ctx, v1beta1Cluster)).To(Succeed(), "v1beta1 update should succeed")

		stored := &infrav1.OpenStackCluster{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cluster), stored)).To(Succeed())
		Expect(stored.Labels).To(HaveKeyWithValue("touched", "yes"))
		return stored.Status
	}

	It("should keep status on a v1beta1 update without a conversion data annotation", func() {
		Expect(updateWithV1beta1()).To(Equal(ownStatus))
	})

	It("should not restore status from a conversion data annotation", func() {
		By("Setting a conversion data annotation with a forged status")
		forged, err := json.Marshal(map[string]any{
			"status": map[string]any{
				"ready":   true,
				"network": map[string]any{"id": "victim-network"},
				"router":  map[string]any{"id": "victim-router"},
				"bastion": map[string]any{"id": "victim-bastion", "floatingIP": "203.0.113.50"},
			},
		})
		Expect(err).NotTo(HaveOccurred())
		annotationPatch, err := json.Marshal(map[string]any{
			"metadata": map[string]any{
				"annotations": map[string]string{utilconversion.DataAnnotation: string(forged)},
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(user.Patch(ctx, cluster.DeepCopy(), client.RawPatch(types.MergePatchType, annotationPatch))).To(Succeed(),
			"Setting an annotation only needs update permission")

		Expect(updateWithV1beta1()).To(Equal(ownStatus))
	})
})
