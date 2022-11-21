/*
Copyright 2022 The Kubernetes Authors.

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

package simulator

import (
	"fmt"

	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
)

type (
	ListImagesPreHook  func(listOpts images.ListOptsBuilder) (bool, []images.Image, error)
	ListImagesPostHook func(images.ListOptsBuilder, []images.Image, error)
)

type ImageSimulator struct {
	Simulator *OpenStackSimulator

	Images []images.Image

	ListImagesPreHook  ListImagesPreHook
	ListImagesPostHook ListImagesPostHook
}

func NewImageSimulator(p *OpenStackSimulator) *ImageSimulator {
	return &ImageSimulator{Simulator: p}
}

/*
 * Simulator implementation methods
 */

func (c *ImageSimulator) ImplListImages(listOpts images.ListOptsBuilder) ([]images.Image, error) {
	query, err := listOpts.ToImageListQuery()
	if err != nil {
		return nil, fmt.Errorf("creating image list query: %w", err)
	}
	name, err := getNameFromQuery(query)
	if err != nil {
		return nil, fmt.Errorf("ListImages: %w", err)
	}

	images := []images.Image{}
	for _, image := range c.Images {
		if image.Name == name {
			images = append(images, image)
		}
	}

	return images, nil
}

/*
 * Callback handler stubs
 */

func (c *ImageSimulator) ListImages(listOpts images.ListOptsBuilder) ([]images.Image, error) {
	if c.ListImagesPreHook != nil {
		handled, images, err := c.ListImagesPreHook(listOpts)
		if handled {
			return images, err
		}
	}

	images, err := c.ImplListImages(listOpts)

	if c.ListImagesPostHook != nil {
		c.ListImagesPostHook(listOpts, images, err)
	}

	return images, err
}

/*
 * Simulator state helpers
 */

func (c *ImageSimulator) SimAddImage(name, id string) {
	c.Images = append(c.Images, images.Image{
		Name: name,
		ID:   id,
	})
}
