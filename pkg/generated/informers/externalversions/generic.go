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

// Code generated by informer-gen. DO NOT EDIT.

package externalversions

import (
	"fmt"

	schema "k8s.io/apimachinery/pkg/runtime/schema"
	cache "k8s.io/client-go/tools/cache"
	v1alpha1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha1"
	v1alpha6 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha6"
	v1alpha7 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha7"
	v1beta1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

// GenericInformer is type of SharedIndexInformer which will locate and delegate to other
// sharedInformers based on type
type GenericInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() cache.GenericLister
}

type genericInformer struct {
	informer cache.SharedIndexInformer
	resource schema.GroupResource
}

// Informer returns the SharedIndexInformer.
func (f *genericInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

// Lister returns the GenericLister.
func (f *genericInformer) Lister() cache.GenericLister {
	return cache.NewGenericLister(f.Informer().GetIndexer(), f.resource)
}

// ForResource gives generic access to a shared informer of the matching type
// TODO extend this to unknown resources with a client pool
func (f *sharedInformerFactory) ForResource(resource schema.GroupVersionResource) (GenericInformer, error) {
	switch resource {
	// Group=infrastructure.cluster.x-k8s.io, Version=v1alpha1
	case v1alpha1.SchemeGroupVersion.WithResource("openstackservers"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Infrastructure().V1alpha1().OpenStackServers().Informer()}, nil

		// Group=infrastructure.cluster.x-k8s.io, Version=v1alpha6
	case v1alpha6.SchemeGroupVersion.WithResource("openstackclusters"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Infrastructure().V1alpha6().OpenStackClusters().Informer()}, nil
	case v1alpha6.SchemeGroupVersion.WithResource("openstackclustertemplates"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Infrastructure().V1alpha6().OpenStackClusterTemplates().Informer()}, nil
	case v1alpha6.SchemeGroupVersion.WithResource("openstackmachines"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Infrastructure().V1alpha6().OpenStackMachines().Informer()}, nil
	case v1alpha6.SchemeGroupVersion.WithResource("openstackmachinetemplates"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Infrastructure().V1alpha6().OpenStackMachineTemplates().Informer()}, nil

		// Group=infrastructure.cluster.x-k8s.io, Version=v1alpha7
	case v1alpha7.SchemeGroupVersion.WithResource("openstackclusters"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Infrastructure().V1alpha7().OpenStackClusters().Informer()}, nil
	case v1alpha7.SchemeGroupVersion.WithResource("openstackclustertemplates"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Infrastructure().V1alpha7().OpenStackClusterTemplates().Informer()}, nil
	case v1alpha7.SchemeGroupVersion.WithResource("openstackmachines"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Infrastructure().V1alpha7().OpenStackMachines().Informer()}, nil
	case v1alpha7.SchemeGroupVersion.WithResource("openstackmachinetemplates"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Infrastructure().V1alpha7().OpenStackMachineTemplates().Informer()}, nil

		// Group=infrastructure.cluster.x-k8s.io, Version=v1beta1
	case v1beta1.SchemeGroupVersion.WithResource("openstackclusters"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Infrastructure().V1beta1().OpenStackClusters().Informer()}, nil
	case v1beta1.SchemeGroupVersion.WithResource("openstackclustertemplates"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Infrastructure().V1beta1().OpenStackClusterTemplates().Informer()}, nil
	case v1beta1.SchemeGroupVersion.WithResource("openstackmachines"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Infrastructure().V1beta1().OpenStackMachines().Informer()}, nil
	case v1beta1.SchemeGroupVersion.WithResource("openstackmachinetemplates"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Infrastructure().V1beta1().OpenStackMachineTemplates().Informer()}, nil

	}

	return nil, fmt.Errorf("no informer found for %v", resource)
}