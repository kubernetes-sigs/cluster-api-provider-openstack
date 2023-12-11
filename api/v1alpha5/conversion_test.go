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

	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha8"
)

func TestConvertFrom(t *testing.T) {
	g := gomega.NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(AddToScheme(scheme)).To(gomega.Succeed())
	g.Expect(infrav1.AddToScheme(scheme)).To(gomega.Succeed())

	tests := []struct {
		name  string
		spoke ctrlconversion.Convertible
		hub   ctrlconversion.Hub
		want  ctrlconversion.Convertible
	}{
		{
			name:  "conversion must have conversion-data annotation",
			spoke: &OpenStackCluster{},
			hub: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{},
			},
			want: &OpenStackCluster{
				Spec: OpenStackClusterSpec{},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"cluster.x-k8s.io/conversion-data": "{\"spec\":{\"apiServerLoadBalancer\":{},\"cloudName\":\"\",\"controlPlaneEndpoint\":{\"host\":\"\",\"port\":0},\"disableAPIServerFloatingIP\":false,\"disableExternalNetwork\":false,\"externalNetwork\":{},\"managedSecurityGroups\":null,\"network\":{}},\"status\":{\"ready\":false}}",
					},
				},
			},
		},
		{
			name:  "conversion must have conversion-data annotation",
			spoke: &OpenStackClusterTemplate{},
			hub: &infrav1.OpenStackClusterTemplate{
				Spec: infrav1.OpenStackClusterTemplateSpec{},
			},
			want: &OpenStackClusterTemplate{
				Spec: OpenStackClusterTemplateSpec{},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"cluster.x-k8s.io/conversion-data": "{\"spec\":{\"template\":{\"spec\":{\"apiServerLoadBalancer\":{},\"cloudName\":\"\",\"controlPlaneEndpoint\":{\"host\":\"\",\"port\":0},\"disableAPIServerFloatingIP\":false,\"disableExternalNetwork\":false,\"externalNetwork\":{},\"managedSecurityGroups\":null,\"network\":{}}}}}",
					},
				},
			},
		},
		{
			name:  "conversion must have conversion-data annotation",
			spoke: &OpenStackMachine{},
			hub: &infrav1.OpenStackMachine{
				Spec: infrav1.OpenStackMachineSpec{},
			},
			want: &OpenStackMachine{
				Spec: OpenStackMachineSpec{},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"cluster.x-k8s.io/conversion-data": "{\"spec\":{\"cloudName\":\"\",\"flavor\":\"\",\"image\":{}},\"status\":{\"dependentResources\":{},\"ready\":false,\"referencedResources\":{}}}",
					},
				},
			},
		},
		{
			name:  "conversion must have conversion-data annotation",
			spoke: &OpenStackMachineTemplate{},
			hub: &infrav1.OpenStackMachineTemplate{
				Spec: infrav1.OpenStackMachineTemplateSpec{},
			},
			want: &OpenStackMachineTemplate{
				Spec: OpenStackMachineTemplateSpec{},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"cluster.x-k8s.io/conversion-data": "{\"spec\":{\"template\":{\"spec\":{\"cloudName\":\"\",\"flavor\":\"\",\"image\":{}}}}}",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spoke.ConvertFrom(tt.hub)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(tt.spoke).To(gomega.Equal(tt.want))
		})
	}
}

func TestConvert_v1alpha5_OpenStackClusterSpec_To_v1alpha8_OpenStackClusterSpec(t *testing.T) {
	tests := []struct {
		name        string
		in          *OpenStackClusterSpec
		expectedOut *infrav1.OpenStackClusterSpec
	}{
		{
			name:        "empty",
			in:          &OpenStackClusterSpec{},
			expectedOut: &infrav1.OpenStackClusterSpec{},
		},
		{
			name: "with managed security groups and not allow all in cluster traffic",
			in: &OpenStackClusterSpec{
				ManagedSecurityGroups:    true,
				AllowAllInClusterTraffic: false,
			},
			expectedOut: &infrav1.OpenStackClusterSpec{
				ManagedSecurityGroups: &infrav1.ManagedSecurityGroups{
					AllNodesSecurityGroupRules: infrav1.LegacyCalicoSecurityGroupRules(),
				},
			},
		},
		{
			name: "with managed security groups and allow all in cluster traffic",
			in: &OpenStackClusterSpec{
				ManagedSecurityGroups:    true,
				AllowAllInClusterTraffic: true,
			},
			expectedOut: &infrav1.OpenStackClusterSpec{
				ManagedSecurityGroups: &infrav1.ManagedSecurityGroups{
					AllowAllInClusterTraffic: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			out := &infrav1.OpenStackClusterSpec{}
			err := Convert_v1alpha5_OpenStackClusterSpec_To_v1alpha8_OpenStackClusterSpec(tt.in, out, nil)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(out).To(gomega.Equal(tt.expectedOut))
		})
	}
}
