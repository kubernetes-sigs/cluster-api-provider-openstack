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

package webhooks

import (
	"context"
	"testing"

	. "github.com/onsi/gomega" //nolint:revive
	"k8s.io/utils/ptr"

	infrav1beta2 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
)

func TestOpenStackClusterTemplate_ValidateUpdate(t *testing.T) {
	tests := []struct {
		name        string
		oldTemplate *infrav1beta2.OpenStackClusterTemplate
		newTemplate *infrav1beta2.OpenStackClusterTemplate
		wantErr     bool
	}{
		{
			name: "Changing spec.template.spec is not allowed",
			oldTemplate: &infrav1beta2.OpenStackClusterTemplate{
				Spec: infrav1beta2.OpenStackClusterTemplateSpec{
					Template: infrav1beta2.OpenStackClusterTemplateResource{
						Spec: infrav1beta2.OpenStackClusterSpec{
							IdentityRef: infrav1beta2.OpenStackIdentityReference{
								Name:      "foobar",
								CloudName: "foobar",
							},
						},
					},
				},
			},
			newTemplate: &infrav1beta2.OpenStackClusterTemplate{
				Spec: infrav1beta2.OpenStackClusterTemplateSpec{
					Template: infrav1beta2.OpenStackClusterTemplateResource{
						Spec: infrav1beta2.OpenStackClusterSpec{
							IdentityRef: infrav1beta2.OpenStackIdentityReference{
								Name:      "changed",
								CloudName: "foobar",
							},
							DisableAPIServerFloatingIP: ptr.To(true),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "No change to spec.template.spec is allowed",
			oldTemplate: &infrav1beta2.OpenStackClusterTemplate{
				Spec: infrav1beta2.OpenStackClusterTemplateSpec{
					Template: infrav1beta2.OpenStackClusterTemplateResource{
						Spec: infrav1beta2.OpenStackClusterSpec{
							IdentityRef: infrav1beta2.OpenStackIdentityReference{
								Name:      "foobar",
								CloudName: "foobar",
							},
						},
					},
				},
			},
			newTemplate: &infrav1beta2.OpenStackClusterTemplate{
				Spec: infrav1beta2.OpenStackClusterTemplateSpec{
					Template: infrav1beta2.OpenStackClusterTemplateResource{
						Spec: infrav1beta2.OpenStackClusterSpec{
							IdentityRef: infrav1beta2.OpenStackIdentityReference{
								Name:      "foobar",
								CloudName: "foobar",
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			ctx := context.TODO()

			webhook := &openStackClusterTemplateWebhook{}
			warn, err := webhook.ValidateUpdate(ctx, tt.oldTemplate, tt.newTemplate)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
			g.Expect(warn).To(BeEmpty())
		})
	}
}
