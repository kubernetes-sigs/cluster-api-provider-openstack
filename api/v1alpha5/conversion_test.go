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
	"runtime/debug"
	"testing"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	testhelpers "sigs.k8s.io/cluster-api-provider-openstack/test/helpers"
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
			name:  "cluster conversion must have conversion-data annotation",
			spoke: &OpenStackCluster{},
			hub: &infrav1.OpenStackCluster{
				Spec: infrav1.OpenStackClusterSpec{},
			},
			want: &OpenStackCluster{
				Spec: OpenStackClusterSpec{
					IdentityRef: &OpenStackIdentityReference{},
				},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"cluster.x-k8s.io/conversion-data": "{\"spec\":{\"identityRef\":{\"cloudName\":\"\",\"name\":\"\"}},\"status\":{\"ready\":false}}",
					},
				},
			},
		},
		{
			name:  "cluster template conversion must have conversion-data annotation",
			spoke: &OpenStackClusterTemplate{},
			hub: &infrav1.OpenStackClusterTemplate{
				Spec: infrav1.OpenStackClusterTemplateSpec{},
			},
			want: &OpenStackClusterTemplate{
				Spec: OpenStackClusterTemplateSpec{
					Template: OpenStackClusterTemplateResource{
						Spec: OpenStackClusterSpec{
							IdentityRef: &OpenStackIdentityReference{},
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"cluster.x-k8s.io/conversion-data": "{\"spec\":{\"template\":{\"spec\":{\"identityRef\":{\"cloudName\":\"\",\"name\":\"\"}}}}}",
					},
				},
			},
		},
		{
			name:  "machine conversion must have conversion-data annotation",
			spoke: &OpenStackMachine{},
			hub: &infrav1.OpenStackMachine{
				Spec: infrav1.OpenStackMachineSpec{},
			},
			want: &OpenStackMachine{
				Spec: OpenStackMachineSpec{},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"cluster.x-k8s.io/conversion-data": "{\"spec\":{\"flavor\":\"\",\"image\":{}},\"status\":{\"ready\":false}}",
					},
				},
			},
		},
		{
			name:  "machine template conversion must have conversion-data annotation",
			spoke: &OpenStackMachineTemplate{},
			hub: &infrav1.OpenStackMachineTemplate{
				Spec: infrav1.OpenStackMachineTemplateSpec{},
			},
			want: &OpenStackMachineTemplate{
				Spec: OpenStackMachineTemplateSpec{},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"cluster.x-k8s.io/conversion-data": "{\"spec\":{\"template\":{\"spec\":{\"flavor\":\"\",\"image\":{}}}}}",
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

func TestConvert_v1alpha5_OpenStackClusterSpec_To_v1beta1_OpenStackClusterSpec(t *testing.T) {
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
			err := Convert_v1alpha5_OpenStackClusterSpec_To_v1beta1_OpenStackClusterSpec(tt.in, out, nil)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(out).To(gomega.Equal(tt.expectedOut))
		})
	}
}

type convertiblePointer[T any] interface {
	ctrlconversion.Convertible
	*T
}

type hubPointer[T any] interface {
	ctrlconversion.Hub
	*T
}

func test_ObjectConvert[SP convertiblePointer[S], HP hubPointer[H], S, H any](tb testing.TB) {
	tb.Helper()

	fuzzerFuncs := func(_ runtimeserializer.CodecFactory) []interface{} {
		return testhelpers.InfraV1FuzzerFuncs()
	}
	f := utilconversion.GetFuzzer(scheme.Scheme, fuzzerFuncs)
	g := gomega.NewWithT(tb)

	for i := 0; i < 10000; i++ {
		var hub HP = new(H)
		f.Fuzz(hub)
		var spoke SP = new(S)

		func() {
			defer func() {
				if r := recover(); r != nil {
					tb.Errorf("PANIC! Down-converting:\n%s\n%s", format.Object(hub, 1), debug.Stack())
					tb.FailNow()
				}
			}()
			g.Expect(spoke.ConvertFrom(hub)).To(gomega.Succeed())
		}()

		spoke = new(S)
		f.Fuzz(spoke)
		hub = new(H)

		func() {
			defer func() {
				if r := recover(); r != nil {
					tb.Errorf("PANIC! Up-converting:\n%s\n%s", format.Object(spoke, 1), debug.Stack())
					tb.FailNow()
				}
			}()
			g.Expect(spoke.ConvertTo(hub)).To(gomega.Succeed())
		}()
	}
}

func Test_OpenStackClusterConvert(t *testing.T) {
	test_ObjectConvert[*OpenStackCluster, *infrav1.OpenStackCluster](t)
}

func Test_OpenStackClusterTemplate(t *testing.T) {
	test_ObjectConvert[*OpenStackClusterTemplate, *infrav1.OpenStackClusterTemplate](t)
}

func Test_OpenStackMachineConvert(t *testing.T) {
	test_ObjectConvert[*OpenStackMachine, *infrav1.OpenStackMachine](t)
}

func Test_OpenStackMachineTemplateConvert(t *testing.T) {
	test_ObjectConvert[*OpenStackMachineTemplate, *infrav1.OpenStackMachineTemplate](t)
}
