/*
Copyright 2023 The Kubernetes Authors.

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

package v1beta1

import (
	"strings"

	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	securitygroups "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
)

func (securityGroupFilter *SecurityGroupFilter) ToListOpt() securitygroups.ListOpts {
	return securitygroups.ListOpts{
		ID:          securityGroupFilter.ID,
		Name:        securityGroupFilter.Name,
		Description: securityGroupFilter.Description,
		ProjectID:   securityGroupFilter.ProjectID,
		Tags:        joinTags(securityGroupFilter.Tags),
		TagsAny:     joinTags(securityGroupFilter.TagsAny),
		NotTags:     joinTags(securityGroupFilter.NotTags),
		NotTagsAny:  joinTags(securityGroupFilter.NotTagsAny),
	}
}

func (subnetFilter *SubnetFilter) ToListOpt() subnets.ListOpts {
	return subnets.ListOpts{
		Name:            subnetFilter.Name,
		Description:     subnetFilter.Description,
		ProjectID:       subnetFilter.ProjectID,
		IPVersion:       subnetFilter.IPVersion,
		GatewayIP:       subnetFilter.GatewayIP,
		CIDR:            subnetFilter.CIDR,
		IPv6AddressMode: subnetFilter.IPv6AddressMode,
		IPv6RAMode:      subnetFilter.IPv6RAMode,
		ID:              subnetFilter.ID,
		Tags:            joinTags(subnetFilter.Tags),
		TagsAny:         joinTags(subnetFilter.TagsAny),
		NotTags:         joinTags(subnetFilter.NotTags),
		NotTagsAny:      joinTags(subnetFilter.NotTagsAny),
	}
}

func (networkFilter *NetworkFilter) ToListOpt() networks.ListOpts {
	return networks.ListOpts{
		Name:        networkFilter.Name,
		Description: networkFilter.Description,
		ProjectID:   networkFilter.ProjectID,
		ID:          networkFilter.ID,
		Tags:        joinTags(networkFilter.Tags),
		TagsAny:     joinTags(networkFilter.TagsAny),
		NotTags:     joinTags(networkFilter.NotTags),
		NotTagsAny:  joinTags(networkFilter.NotTagsAny),
	}
}

func (networkFilter *NetworkFilter) IsEmpty() bool {
	return networkFilter.Name == "" &&
		networkFilter.Description == "" &&
		networkFilter.ProjectID == "" &&
		networkFilter.ID == "" &&
		len(networkFilter.Tags) == 0 &&
		len(networkFilter.TagsAny) == 0 &&
		len(networkFilter.NotTags) == 0 &&
		len(networkFilter.NotTagsAny) == 0
}

func (routerFilter *RouterFilter) ToListOpt() routers.ListOpts {
	return routers.ListOpts{
		ID:          routerFilter.ID,
		Name:        routerFilter.Name,
		Description: routerFilter.Description,
		ProjectID:   routerFilter.ProjectID,
		Tags:        joinTags(routerFilter.Tags),
		TagsAny:     joinTags(routerFilter.TagsAny),
		NotTags:     joinTags(routerFilter.NotTags),
		NotTagsAny:  joinTags(routerFilter.NotTagsAny),
	}
}

func (imageFilter *ImageFilter) ToListOpt() images.ListOpts {
	listOpts := images.ListOpts{
		Tags: imageFilter.Tags,
	}
	if imageFilter.ID != nil {
		listOpts.ID = *imageFilter.ID
	}
	if imageFilter.Name != nil {
		listOpts.Name = *imageFilter.Name
	}
	return listOpts
}

// splitTags splits a comma separated list of tags into a slice of tags.
// If the input is an empty string, it returns nil representing no list rather
// than an empty list.
func splitTags(tags string) []NeutronTag {
	if tags == "" {
		return nil
	}

	var ret []NeutronTag
	for _, tag := range strings.Split(tags, ",") {
		if tag != "" {
			ret = append(ret, NeutronTag(tag))
		}
	}

	return ret
}

// joinTags joins a slice of tags into a comma separated list of tags.
func joinTags(tags []NeutronTag) string {
	var b strings.Builder
	for i := range tags {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(string(tags[i]))
	}
	return b.String()
}

func ConvertAllTagsTo(tags, tagsAny, notTags, notTagsAny string, neutronTags *FilterByNeutronTags) {
	neutronTags.Tags = splitTags(tags)
	neutronTags.TagsAny = splitTags(tagsAny)
	neutronTags.NotTags = splitTags(notTags)
	neutronTags.NotTagsAny = splitTags(notTagsAny)
}

func ConvertAllTagsFrom(neutronTags *FilterByNeutronTags, tags, tagsAny, notTags, notTagsAny *string) {
	*tags = joinTags(neutronTags.Tags)
	*tagsAny = joinTags(neutronTags.TagsAny)
	*notTags = joinTags(neutronTags.NotTags)
	*notTagsAny = joinTags(neutronTags.NotTagsAny)
}
