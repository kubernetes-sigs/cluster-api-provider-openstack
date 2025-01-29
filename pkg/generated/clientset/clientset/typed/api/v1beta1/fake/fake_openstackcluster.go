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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	gentype "k8s.io/client-go/gentype"
	v1beta1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	apiv1beta1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/generated/applyconfiguration/api/v1beta1"
	typedapiv1beta1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/generated/clientset/clientset/typed/api/v1beta1"
)

// fakeOpenStackClusters implements OpenStackClusterInterface
type fakeOpenStackClusters struct {
	*gentype.FakeClientWithListAndApply[*v1beta1.OpenStackCluster, *v1beta1.OpenStackClusterList, *apiv1beta1.OpenStackClusterApplyConfiguration]
	Fake *FakeInfrastructureV1beta1
}

func newFakeOpenStackClusters(fake *FakeInfrastructureV1beta1, namespace string) typedapiv1beta1.OpenStackClusterInterface {
	return &fakeOpenStackClusters{
		gentype.NewFakeClientWithListAndApply[*v1beta1.OpenStackCluster, *v1beta1.OpenStackClusterList, *apiv1beta1.OpenStackClusterApplyConfiguration](
			fake.Fake,
			namespace,
			v1beta1.SchemeGroupVersion.WithResource("openstackclusters"),
			v1beta1.SchemeGroupVersion.WithKind("OpenStackCluster"),
			func() *v1beta1.OpenStackCluster { return &v1beta1.OpenStackCluster{} },
			func() *v1beta1.OpenStackClusterList { return &v1beta1.OpenStackClusterList{} },
			func(dst, src *v1beta1.OpenStackClusterList) { dst.ListMeta = src.ListMeta },
			func(list *v1beta1.OpenStackClusterList) []*v1beta1.OpenStackCluster {
				return gentype.ToPointerSlice(list.Items)
			},
			func(list *v1beta1.OpenStackClusterList, items []*v1beta1.OpenStackCluster) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
