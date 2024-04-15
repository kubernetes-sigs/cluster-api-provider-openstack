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

package apivalidations

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

var _ = Describe("Filter API validations", func() {
	var (
		namespace *corev1.Namespace
		cluster   *infrav1.OpenStackCluster
		machine   *infrav1.OpenStackMachine
	)

	BeforeEach(func() {
		namespace = createNamespace()

		// Initialise a basic machine object in the correct namespace
		machine = &infrav1.OpenStackMachine{
			Spec: infrav1.OpenStackMachineSpec{
				Image: infrav1.ImageParam{Filter: &infrav1.ImageFilter{Name: ptr.To("test-image")}},
			},
		}
		machine.Namespace = namespace.Name
		machine.GenerateName = "machine-"

		// Initialise a basic cluster object in the correct namespace
		cluster = &infrav1.OpenStackCluster{}
		cluster.Namespace = namespace.Name
		cluster.GenerateName = clusterNamePrefix
	})

	DescribeTable("Allow valid neutron filter tags", func(tags []infrav1.FilterByNeutronTags) {
		// Specify the given neutron tags in every filter it is
		// possible to specify them in, then create the
		// resulting object. It should be valid.

		securityGroups := make([]infrav1.SecurityGroupParam, len(tags))
		for i := range tags {
			securityGroups[i].Filter = &infrav1.SecurityGroupFilter{FilterByNeutronTags: tags[i]}
		}
		machine.Spec.SecurityGroups = securityGroups

		ports := make([]infrav1.PortOpts, len(tags))
		for i := range tags {
			port := &ports[i]
			port.Network = &infrav1.NetworkParam{Filter: &infrav1.NetworkFilter{FilterByNeutronTags: tags[i]}}
			port.FixedIPs = []infrav1.FixedIP{{Subnet: &infrav1.SubnetParam{
				Filter: &infrav1.SubnetFilter{FilterByNeutronTags: tags[i]},
			}}}
			port.SecurityGroups = []infrav1.SecurityGroupParam{{Filter: &infrav1.SecurityGroupFilter{FilterByNeutronTags: tags[i]}}}
		}
		Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "OpenStackMachine creation should succeed")

		// Maximum of 2 subnets are supported
		nSubnets := min(len(tags), 2)
		subnets := make([]infrav1.SubnetParam, nSubnets)
		for i := 0; i < nSubnets; i++ {
			subnets[i].Filter = &infrav1.SubnetFilter{FilterByNeutronTags: tags[i]}
		}
		cluster.Spec.Subnets = subnets
		if len(tags) > 0 {
			cluster.Spec.Network = &infrav1.NetworkParam{Filter: &infrav1.NetworkFilter{FilterByNeutronTags: tags[0]}}
			cluster.Spec.ExternalNetwork = &infrav1.NetworkParam{Filter: &infrav1.NetworkFilter{FilterByNeutronTags: tags[0]}}
			cluster.Spec.Router = &infrav1.RouterParam{Filter: &infrav1.RouterFilter{FilterByNeutronTags: tags[0]}}
		}
		Expect(k8sClient.Create(ctx, cluster)).To(Succeed(), "OpenStackCluster creation should succeed")
	},
		Entry("empty list", nil),
		Entry("single tag", []infrav1.FilterByNeutronTags{
			{Tags: []infrav1.NeutronTag{"foo"}},
		}),
		Entry("multiple filters, multiple tags", []infrav1.FilterByNeutronTags{
			{Tags: []infrav1.NeutronTag{"foo", "bar"}},
			{TagsAny: []infrav1.NeutronTag{"foo", "bar"}},
			{NotTags: []infrav1.NeutronTag{"foo", "bar"}},
			{NotTagsAny: []infrav1.NeutronTag{"foo", "bar"}},
		}),
	)

	DescribeTable("Disallow invalid neutron filter tags", func(tags []infrav1.FilterByNeutronTags) {
		{
			machine := machine.DeepCopy()
			securityGroups := make([]infrav1.SecurityGroupParam, len(tags))
			for i := range tags {
				securityGroups[i].Filter = &infrav1.SecurityGroupFilter{FilterByNeutronTags: tags[i]}
			}
			machine.Spec.SecurityGroups = securityGroups
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail with invalid security group neutron tags")
		}

		for i := range tags {
			{
				machine := machine.DeepCopy()
				machine.Spec.Ports = []infrav1.PortOpts{
					{Network: &infrav1.NetworkParam{Filter: &infrav1.NetworkFilter{FilterByNeutronTags: tags[i]}}},
				}
				Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail with invalid port network neutron tags")
			}
			{
				machine := machine.DeepCopy()
				machine.Spec.Ports = []infrav1.PortOpts{
					{FixedIPs: []infrav1.FixedIP{{Subnet: &infrav1.SubnetParam{
						Filter: &infrav1.SubnetFilter{FilterByNeutronTags: tags[i]},
					}}}},
				}
				Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail with invalid port subnet neutron tags")
			}
			{
				machine := machine.DeepCopy()
				machine.Spec.Ports = []infrav1.PortOpts{
					{SecurityGroups: []infrav1.SecurityGroupParam{{Filter: &infrav1.SecurityGroupFilter{FilterByNeutronTags: tags[i]}}}},
				}
				Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail with invalid port security group neutron tags")
			}
		}

		if len(tags) > 0 {
			tag := tags[0]

			{
				cluster := cluster.DeepCopy()
				cluster.Spec.Subnets = []infrav1.SubnetParam{{Filter: &infrav1.SubnetFilter{FilterByNeutronTags: tag}}}
				Expect(k8sClient.Create(ctx, cluster)).NotTo(Succeed(), "OpenStackCluster creation should fail with invalid subnet neutron tags")
			}

			{
				cluster := cluster.DeepCopy()
				cluster.Spec.Network = &infrav1.NetworkParam{Filter: &infrav1.NetworkFilter{FilterByNeutronTags: tag}}
				Expect(k8sClient.Create(ctx, cluster)).NotTo(Succeed(), "OpenStackCluster creation should fail with invalid network neutron tags")
			}

			{
				cluster := cluster.DeepCopy()
				cluster.Spec.ExternalNetwork = &infrav1.NetworkParam{Filter: &infrav1.NetworkFilter{FilterByNeutronTags: tag}}
				Expect(k8sClient.Create(ctx, cluster)).NotTo(Succeed(), "OpenStackCluster creation should fail with invalid external network neutron tags")
			}

			{
				cluster := cluster.DeepCopy()
				cluster.Spec.Router = &infrav1.RouterParam{Filter: &infrav1.RouterFilter{FilterByNeutronTags: tag}}
				Expect(k8sClient.Create(ctx, cluster)).NotTo(Succeed(), "OpenStackCluster creation should fail with invalid router neutron tags")
			}
		}
	},
		Entry("contains leading comma", []infrav1.FilterByNeutronTags{
			{Tags: []infrav1.NeutronTag{",foo"}},
		}),
		Entry("contains trailing comma", []infrav1.FilterByNeutronTags{
			{Tags: []infrav1.NeutronTag{"foo,"}},
		}),
		Entry("contains comma in middle", []infrav1.FilterByNeutronTags{
			{Tags: []infrav1.NeutronTag{"foo,bar"}},
		}),
		Entry("contains multiple commas", []infrav1.FilterByNeutronTags{
			{Tags: []infrav1.NeutronTag{"foo,,bar"}},
		}),
		Entry("empty tag", []infrav1.FilterByNeutronTags{
			{Tags: []infrav1.NeutronTag{""}},
		}),
		Entry("second tag is invalid", []infrav1.FilterByNeutronTags{
			{Tags: []infrav1.NeutronTag{"foo", ""}},
		}),
	)

	Context("ImageParam", func() {
		const imageUUID = "5a78f794-cdc3-48d2-8d9f-0fd472fdd743"

		It("should not allow both ID and Filter to be set", func() {
			machine.Spec.Image = infrav1.ImageParam{
				ID: ptr.To(imageUUID),
				Filter: &infrav1.ImageFilter{
					Name: ptr.To("bar"),
				},
			}
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail")
		})

		It("should not allow both ID and Tags of ImageFilter to be set", func() {
			machine.Spec.Image = infrav1.ImageParam{
				ID: ptr.To(imageUUID),
				Filter: &infrav1.ImageFilter{
					Tags: []string{"bar", "baz"},
				},
			}
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail")
		})

		It("should allow UUID ID of ImageFilter to be set", func() {
			machine.Spec.Image = infrav1.ImageParam{
				ID: ptr.To(imageUUID),
			}
			Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "OpenStackMachine creation should succeed")
		})

		It("should not allow non-UUID ID of ImageFilter to be set", func() {
			machine.Spec.Image = infrav1.ImageParam{
				ID: ptr.To("foo"),
			}
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail")
		})

		It("should allow Name and Tags of ImageFilter to be set", func() {
			machine.Spec.Image = infrav1.ImageParam{
				Filter: &infrav1.ImageFilter{
					Name: ptr.To("bar"),
					Tags: []string{"bar", "baz"},
				},
			}
			Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "OpenStackMachine creation should succeed")
		})

		It("should not allow a non-nil, empty image filter", func() {
			machine.Spec.Image = infrav1.ImageParam{
				Filter: &infrav1.ImageFilter{},
			}
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail")
		})
	})

	Context("NetworkParam", func() {
		It("should allow setting ID", func() {
			cluster.Spec.Network = &infrav1.NetworkParam{
				ID: ptr.To("06c32c52-f207-4f6a-a769-bbcbe5a43f5c"),
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed(), "OpenStackCluster creation should succeed")
		})

		It("should allow setting non-empty Filter", func() {
			cluster.Spec.Network = &infrav1.NetworkParam{
				Filter: &infrav1.NetworkFilter{Name: "foo"},
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed(), "OpenStackCluster creation should succeed")
		})

		It("should not allow setting empty param", func() {
			cluster.Spec.Network = &infrav1.NetworkParam{}
			Expect(k8sClient.Create(ctx, cluster)).NotTo(Succeed(), "OpenStackCluster creation should fail")
		})

		It("should not allow setting invalid id", func() {
			cluster.Spec.Network = &infrav1.NetworkParam{
				ID: ptr.To("foo"),
			}
			Expect(k8sClient.Create(ctx, cluster)).NotTo(Succeed(), "OpenStackCluster creation should fail")
		})

		It("should not allow setting empty Filter", func() {
			cluster.Spec.Network = &infrav1.NetworkParam{
				Filter: &infrav1.NetworkFilter{},
			}
			Expect(k8sClient.Create(ctx, cluster)).NotTo(Succeed(), "OpenStackCluster creation should fail")
		})

		It("should not allow setting both ID and Filter", func() {
			cluster.Spec.Network = &infrav1.NetworkParam{
				ID: ptr.To("06c32c52-f207-4f6a-a769-bbcbe5a43f5c"), Filter: &infrav1.NetworkFilter{Name: "foo"},
			}
			Expect(k8sClient.Create(ctx, cluster)).NotTo(Succeed(), "OpenStackCluster creation should fail")
		})
	})

	Context("SubnetParam", func() {
		It("should allow setting ID", func() {
			cluster.Spec.Subnets = []infrav1.SubnetParam{
				{ID: ptr.To("06c32c52-f207-4f6a-a769-bbcbe5a43f5c")},
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed(), "OpenStackCluster creation should succeed")
		})

		It("should allow setting non-empty Filter", func() {
			cluster.Spec.Subnets = []infrav1.SubnetParam{
				{Filter: &infrav1.SubnetFilter{Name: "foo"}},
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed(), "OpenStackCluster creation should succeed")
		})

		It("should not allow setting empty SubnetParam", func() {
			cluster.Spec.Subnets = []infrav1.SubnetParam{{}}
			Expect(k8sClient.Create(ctx, cluster)).NotTo(Succeed(), "OpenStackCluster creation should fail")
		})

		It("should not allow setting invalid id", func() {
			cluster.Spec.Subnets = []infrav1.SubnetParam{
				{ID: ptr.To("foo")},
			}
			Expect(k8sClient.Create(ctx, cluster)).NotTo(Succeed(), "OpenStackCluster creation should fail")
		})

		It("should not allow setting empty Filter", func() {
			cluster.Spec.Subnets = []infrav1.SubnetParam{
				{Filter: &infrav1.SubnetFilter{}},
			}
			Expect(k8sClient.Create(ctx, cluster)).NotTo(Succeed(), "OpenStackCluster creation should fail")
		})

		It("should not allow setting both ID and Filter", func() {
			cluster.Spec.Subnets = []infrav1.SubnetParam{
				{ID: ptr.To("06c32c52-f207-4f6a-a769-bbcbe5a43f5c"), Filter: &infrav1.SubnetFilter{Name: "foo"}},
			}
			Expect(k8sClient.Create(ctx, cluster)).NotTo(Succeed(), "OpenStackCluster creation should fail")
		})
	})

	Context("SecurityGroupParam", func() {
		It("should allow setting ID", func() {
			machine.Spec.SecurityGroups = []infrav1.SecurityGroupParam{
				{ID: ptr.To("06c32c52-f207-4f6a-a769-bbcbe5a43f5c")},
			}
			Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "OpenStackMachine creation should succeed")
		})

		It("should allow setting non-empty Filter", func() {
			machine.Spec.SecurityGroups = []infrav1.SecurityGroupParam{
				{Filter: &infrav1.SecurityGroupFilter{Name: "foo"}},
			}
			Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "OpenStackMachine creation should succeed")
		})

		It("should not allow setting empty param", func() {
			machine.Spec.SecurityGroups = []infrav1.SecurityGroupParam{{}}
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail")
		})

		It("should not allow setting invalid id", func() {
			machine.Spec.SecurityGroups = []infrav1.SecurityGroupParam{
				{ID: ptr.To("foo")},
			}
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail")
		})

		It("should not allow setting empty Filter", func() {
			machine.Spec.SecurityGroups = []infrav1.SecurityGroupParam{
				{Filter: &infrav1.SecurityGroupFilter{}},
			}
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail")
		})

		It("should not allow setting both ID and Filter", func() {
			machine.Spec.SecurityGroups = []infrav1.SecurityGroupParam{
				{ID: ptr.To("06c32c52-f207-4f6a-a769-bbcbe5a43f5c"), Filter: &infrav1.SecurityGroupFilter{Name: "foo"}},
			}
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail")
		})
	})

	Context("RouterParam", func() {
		It("should allow setting ID", func() {
			cluster.Spec.Router = &infrav1.RouterParam{
				ID: ptr.To("06c32c52-f207-4f6a-a769-bbcbe5a43f5c"),
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed(), "OpenStackCluster creation should succeed")
		})

		It("should allow setting non-empty Filter", func() {
			cluster.Spec.Router = &infrav1.RouterParam{
				Filter: &infrav1.RouterFilter{Name: "foo"},
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed(), "OpenStackCluster creation should succeed")
		})

		It("should not allow setting empty RouterParam", func() {
			cluster.Spec.Router = &infrav1.RouterParam{}
			Expect(k8sClient.Create(ctx, cluster)).NotTo(Succeed(), "OpenStackCluster creation should fail")
		})

		It("should not allow setting invalid id", func() {
			cluster.Spec.Router = &infrav1.RouterParam{
				ID: ptr.To("foo"),
			}
			Expect(k8sClient.Create(ctx, cluster)).NotTo(Succeed(), "OpenStackCluster creation should fail")
		})

		It("should not allow setting empty Filter", func() {
			cluster.Spec.Router = &infrav1.RouterParam{
				Filter: &infrav1.RouterFilter{},
			}
			Expect(k8sClient.Create(ctx, cluster)).NotTo(Succeed(), "OpenStackCluster creation should fail")
		})

		It("should not allow setting both ID and Filter", func() {
			cluster.Spec.Router = &infrav1.RouterParam{
				ID:     ptr.To("06c32c52-f207-4f6a-a769-bbcbe5a43f5c"),
				Filter: &infrav1.RouterFilter{Name: "foo"},
			}
			Expect(k8sClient.Create(ctx, cluster)).NotTo(Succeed(), "OpenStackCluster creation should fail")
		})
	})

	Context("ServerGroupParam", func() {
		It("should allow setting ID", func() {
			machine.Spec.ServerGroup = &infrav1.ServerGroupParam{ID: ptr.To("06c32c52-f207-4f6a-a769-bbcbe5a43f5c")}
			Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "OpenStackMachine creation should succeed")
		})

		It("should allow setting non-empty Filter", func() {
			machine.Spec.ServerGroup = &infrav1.ServerGroupParam{Filter: &infrav1.ServerGroupFilter{Name: ptr.To("foo")}}
			Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "OpenStackMachine creation should succeed")
		})

		It("should not allow setting empty param", func() {
			machine.Spec.ServerGroup = &infrav1.ServerGroupParam{}
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail")
		})

		It("should not allow setting invalid id", func() {
			machine.Spec.ServerGroup = &infrav1.ServerGroupParam{ID: ptr.To("foo")}
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail")
		})

		It("should not allow setting empty Filter", func() {
			machine.Spec.ServerGroup = &infrav1.ServerGroupParam{Filter: &infrav1.ServerGroupFilter{}}
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail")
		})

		It("should not allow setting both ID and Filter", func() {
			machine.Spec.ServerGroup = &infrav1.ServerGroupParam{
				ID:     ptr.To("06c32c52-f207-4f6a-a769-bbcbe5a43f5c"),
				Filter: &infrav1.ServerGroupFilter{Name: ptr.To("foo")},
			}
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail")
		})
	})
})
