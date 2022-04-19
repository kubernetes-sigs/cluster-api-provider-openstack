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
)

func TestOpenStackCluster_ValidateUpdate(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name        string
		oldTemplate *OpenStackCluster
		newTemplate *OpenStackCluster
		wantErr     bool
	}{
		{
			name: "OpenStackCluster.Spec.IdentityRef.Kind must always be Secret",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &OpenStackIdentityReference{
						Kind: "Secret",
						Name: "foobar",
					},
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &OpenStackIdentityReference{
						Kind: "foobar",
						Name: "foobar",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Changing OpenStackCluster.Spec.IdentityRef.Name is allowed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &OpenStackIdentityReference{
						Kind: "Secret",
						Name: "foobar",
					},
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &OpenStackIdentityReference{
						Kind: "Secret",
						Name: "foobarbaz",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "OpenStackCluster.Spec.IdentityRef can be changed if it was unset",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &OpenStackIdentityReference{
						Kind: "Secret",
						Name: "foobar",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "OpenStackCluster.Spec.IdentityRef must not be removed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &OpenStackIdentityReference{
						Kind: "Secret",
						Name: "foobar",
					},
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
				},
			},
			wantErr: true,
		},
		{
			name: "Toggle OpenStackCluster.Spec.Bastion.Enabled flag is allowed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					Bastion: &Bastion{
						Instance: OpenStackMachineSpec{
							CloudName: "foobar",
							Image:     "foobar",
							Flavor:    "minimal",
						},
						Enabled: false,
					},
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					Bastion: &Bastion{
						Instance: OpenStackMachineSpec{
							CloudName: "foobarbaz",
							Image:     "foobarbaz",
							Flavor:    "medium",
						},
						Enabled: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Changing empty OpenStackCluster.Spec.Bastion is allowed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					Bastion: &Bastion{
						Instance: OpenStackMachineSpec{
							CloudName: "foobar",
							Image:     "foobar",
							Flavor:    "medium",
						},
						Enabled: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Changing OpenStackCluster.Spec.Bastion with no deployed bastion host is allowed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					Bastion: &Bastion{
						Instance: OpenStackMachineSpec{
							CloudName: "foobar",
							Image:     "foobar",
							Flavor:    "minimal",
						},
						Enabled: false,
					},
				},
				Status: OpenStackClusterStatus{},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					Bastion: &Bastion{
						Instance: OpenStackMachineSpec{
							CloudName: "foobarbaz",
							Image:     "foobarbaz",
							Flavor:    "medium",
						},
						Enabled: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Changing OpenStackCluster.Spec.Bastion with deployed bastion host is not allowed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					Bastion: &Bastion{
						Instance: OpenStackMachineSpec{
							CloudName: "foobar",
							Image:     "foobar",
							Flavor:    "minimal",
						},
						Enabled: true,
					},
				},
				Status: OpenStackClusterStatus{
					Bastion: &Instance{
						Name: "foobar",
					},
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					Bastion: &Bastion{
						Instance: OpenStackMachineSpec{
							CloudName: "foobarbaz",
							Image:     "foobarbaz",
							Flavor:    "medium",
						},
						Enabled: true,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Disabling the OpenStackCluster.Spec.Bastion while it's running is allowed",
			oldTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					Bastion: &Bastion{
						Instance: OpenStackMachineSpec{
							CloudName: "foobar",
							Image:     "foobar",
							Flavor:    "minimal",
						},
						Enabled: true,
					},
				},
				Status: OpenStackClusterStatus{
					Bastion: &Instance{
						Name: "foobar",
					},
				},
			},
			newTemplate: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					Bastion: &Bastion{
						Instance: OpenStackMachineSpec{
							CloudName: "foobar",
							Image:     "foobar",
							Flavor:    "minimal",
						},
						Enabled: false,
					},
				},
				Status: OpenStackClusterStatus{
					Bastion: &Instance{
						Name: "foobar",
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.newTemplate.ValidateUpdate(tt.oldTemplate)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
		})
	}
}

func TestOpenStackCluster_ValidateCreate(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name     string
		template *OpenStackCluster
		wantErr  bool
	}{
		{
			name: "OpenStackCluster.Spec.IdentityRef with correct spec on create",
			template: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &OpenStackIdentityReference{
						Kind: "Secret",
						Name: "foobar",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "OpenStackCluster.Spec.IdentityRef with faulty spec on create",
			template: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					CloudName: "foobar",
					IdentityRef: &OpenStackIdentityReference{
						Kind: "foobar",
						Name: "foobar",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.template.ValidateCreate()
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
		})
	}
}
