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
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

func TestOpenStackMachineTemplate_ValidateUpdate(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name        string
		oldTemplate *infrav1.OpenStackMachineTemplate
		newTemplate *infrav1.OpenStackMachineTemplate
		req         *admission.Request
		wantErr     bool
	}{
		{
			name: "OpenStackMachineTemplate with immutable spec",
			oldTemplate: &infrav1.OpenStackMachineTemplate{
				Spec: infrav1.OpenStackMachineTemplateSpec{
					Template: infrav1.OpenStackMachineTemplateResource{
						Spec: infrav1.OpenStackMachineSpec{
							Flavor: ptr.To("foo"),
							Image: infrav1.ImageParam{
								Filter: &infrav1.ImageFilter{
									Name: ptr.To("bar"),
								},
							},
						},
					},
				},
			},
			newTemplate: &infrav1.OpenStackMachineTemplate{
				Spec: infrav1.OpenStackMachineTemplateSpec{
					Template: infrav1.OpenStackMachineTemplateResource{
						Spec: infrav1.OpenStackMachineSpec{
							Flavor: ptr.To("foo"),
							Image: infrav1.ImageParam{
								Filter: &infrav1.ImageFilter{
									Name: ptr.To("NewImage"),
								},
							},
						},
					},
				},
			},
			req:     &admission.Request{},
			wantErr: true,
		},
		{
			name: "OpenStackMachineTemplate with mutable metadata",
			oldTemplate: &infrav1.OpenStackMachineTemplate{
				Spec: infrav1.OpenStackMachineTemplateSpec{
					Template: infrav1.OpenStackMachineTemplateResource{
						Spec: infrav1.OpenStackMachineSpec{
							Flavor: ptr.To("foo"),
							Image: infrav1.ImageParam{
								Filter: &infrav1.ImageFilter{
									Name: ptr.To("bar"),
								},
							},
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			},
			newTemplate: &infrav1.OpenStackMachineTemplate{
				Spec: infrav1.OpenStackMachineTemplateSpec{
					Template: infrav1.OpenStackMachineTemplateResource{
						Spec: infrav1.OpenStackMachineSpec{
							Flavor: ptr.To("foo"),
							Image: infrav1.ImageParam{
								Filter: &infrav1.ImageFilter{
									Name: ptr.To("bar"),
								},
							},
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "bar",
				},
			},
			req: &admission.Request{},
		},
		{
			name: "don't allow modification, dry run, no skip immutability annotation set",
			oldTemplate: &infrav1.OpenStackMachineTemplate{
				Spec: infrav1.OpenStackMachineTemplateSpec{
					Template: infrav1.OpenStackMachineTemplateResource{
						Spec: infrav1.OpenStackMachineSpec{
							Flavor: ptr.To("foo"),
							Image: infrav1.ImageParam{
								Filter: &infrav1.ImageFilter{
									Name: ptr.To("bar"),
								},
							},
						},
					},
				},
			},
			newTemplate: &infrav1.OpenStackMachineTemplate{
				Spec: infrav1.OpenStackMachineTemplateSpec{
					Template: infrav1.OpenStackMachineTemplateResource{
						Spec: infrav1.OpenStackMachineSpec{
							Flavor: ptr.To("foo"),
							Image: infrav1.ImageParam{
								Filter: &infrav1.ImageFilter{
									Name: ptr.To("NewImage"),
								},
							},
						},
					},
				},
			},
			req:     &admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{DryRun: ptr.To(true)}},
			wantErr: true,
		},
		{
			name: "allow modification, dry run, skip immutability annotation set",
			oldTemplate: &infrav1.OpenStackMachineTemplate{
				Spec: infrav1.OpenStackMachineTemplateSpec{
					Template: infrav1.OpenStackMachineTemplateResource{
						Spec: infrav1.OpenStackMachineSpec{
							Flavor: ptr.To("foo"),
							Image: infrav1.ImageParam{
								Filter: &infrav1.ImageFilter{
									Name: ptr.To("bar"),
								},
							},
						},
					},
				},
			},
			newTemplate: &infrav1.OpenStackMachineTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						clusterv1.TopologyDryRunAnnotation: "",
					},
				},
				Spec: infrav1.OpenStackMachineTemplateSpec{
					Template: infrav1.OpenStackMachineTemplateResource{
						Spec: infrav1.OpenStackMachineSpec{
							Flavor: ptr.To("foo"),
							Image: infrav1.ImageParam{
								Filter: &infrav1.ImageFilter{
									Name: ptr.To("NewImage"),
								},
							},
						},
					},
				},
			},
			req: &admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{DryRun: ptr.To(true)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			webhook := &openStackMachineTemplateWebhook{}
			ctx := admission.NewContextWithRequest(context.Background(), *tt.req)

			warn, err := webhook.ValidateUpdate(ctx, tt.oldTemplate, tt.newTemplate)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
			// Nothing emits warnings yet
			g.Expect(warn).To(BeEmpty())
		})
	}
}
