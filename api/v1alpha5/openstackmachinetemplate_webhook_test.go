/*
Copyright 2021 The Kubernetes Authors.

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

package v1alpha5

import (
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOpenStackMachineTemplate_ValidateUpdate(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name        string
		oldTemplate *OpenStackMachineTemplate
		newTemplate *OpenStackMachineTemplate
		wantErr     bool
	}{
		{
			name: "OpenStackMachineTemplate with immutable spec",
			oldTemplate: &OpenStackMachineTemplate{
				Spec: OpenStackMachineTemplateSpec{
					Template: OpenStackMachineTemplateResource{
						Spec: OpenStackMachineSpec{
							Flavor: "foo",
							Image:  "bar",
						},
					},
				},
			},
			newTemplate: &OpenStackMachineTemplate{
				Spec: OpenStackMachineTemplateSpec{
					Template: OpenStackMachineTemplateResource{
						Spec: OpenStackMachineSpec{
							Flavor: "foo",
							Image:  "NewImage",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "OpenStackMachineTemplate with mutable metadata",
			oldTemplate: &OpenStackMachineTemplate{
				Spec: OpenStackMachineTemplateSpec{
					Template: OpenStackMachineTemplateResource{
						Spec: OpenStackMachineSpec{
							Flavor: "foo",
							Image:  "bar",
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			},
			newTemplate: &OpenStackMachineTemplate{
				Spec: OpenStackMachineTemplateSpec{
					Template: OpenStackMachineTemplateResource{
						Spec: OpenStackMachineSpec{
							Flavor: "foo",
							Image:  "bar",
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "bar",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.newTemplate.ValidateUpdate(tt.oldTemplate)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
		})
	}
}
