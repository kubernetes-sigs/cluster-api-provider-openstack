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

package convert

import (
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"

	infrav1alpha7 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha7"
)

// NetworkFilterToListOpt converts a v1alpha7.NetworkFilter to a networks.ListOpts
// Still used by the Floating IP IPAM controller until we bump it to v1beta1.
func NetworkFilterToListOpt(networkFilter *infrav1alpha7.NetworkFilter) networks.ListOpts {
	return networks.ListOpts{
		Name:        networkFilter.Name,
		Description: networkFilter.Description,
		ProjectID:   networkFilter.ProjectID,
		ID:          networkFilter.ID,
		Tags:        networkFilter.Tags,
		TagsAny:     networkFilter.TagsAny,
		NotTags:     networkFilter.NotTags,
		NotTagsAny:  networkFilter.NotTagsAny,
	}
}
