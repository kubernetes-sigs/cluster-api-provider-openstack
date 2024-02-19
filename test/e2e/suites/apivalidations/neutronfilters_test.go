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

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

var _ = Describe("Neutron filter API validations", func() {
	var (
		namespace *corev1.Namespace
		cluster   *infrav1.OpenStackCluster
		machine   *infrav1.OpenStackMachine
	)

	BeforeEach(func() {
		namespace = createNamespace()

		// Initialise a basic machine object in the correct namespace
		machine = &infrav1.OpenStackMachine{}
		machine.Namespace = namespace.Name
		machine.GenerateName = "machine-"

		// Initialise a basic cluster object in the correct namespace
		cluster = &infrav1.OpenStackCluster{}
		cluster.Namespace = namespace.Name
		cluster.GenerateName = "cluster-"
	})

	DescribeTable("Allow valid neutron filter tags", func(tags []infrav1.FilterByNeutronTags) {
		// Specify the given neutron tags in every filter it is
		// possible to specify them in, then create the
		// resulting object. It should be valid.

		securityGroups := make([]infrav1.SecurityGroupFilter, len(tags))
		for i := range tags {
			securityGroups[i].FilterByNeutronTags = tags[i]
		}
		machine.Spec.SecurityGroups = securityGroups

		ports := make([]infrav1.PortOpts, len(tags))
		for i := range tags {
			port := &ports[i]
			port.Network = &infrav1.NetworkFilter{FilterByNeutronTags: tags[i]}
			port.FixedIPs = []infrav1.FixedIP{{Subnet: &infrav1.SubnetFilter{FilterByNeutronTags: tags[i]}}}
			port.SecurityGroups = []infrav1.SecurityGroupFilter{{FilterByNeutronTags: tags[i]}}
		}
		Expect(k8sClient.Create(ctx, machine)).To(Succeed(), "OpenStackMachine creation should succeed")

		// Maximum of 2 subnets are supported
		nSubnets := min(len(tags), 2)
		subnets := make([]infrav1.SubnetFilter, nSubnets)
		for i := 0; i < nSubnets; i++ {
			subnets[i].FilterByNeutronTags = tags[i]
		}
		cluster.Spec.Subnets = subnets
		if len(tags) > 0 {
			cluster.Spec.Network = &infrav1.NetworkFilter{FilterByNeutronTags: tags[0]}
			cluster.Spec.ExternalNetwork = &infrav1.NetworkFilter{FilterByNeutronTags: tags[0]}
			cluster.Spec.Router = &infrav1.RouterFilter{FilterByNeutronTags: tags[0]}
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
			securityGroups := make([]infrav1.SecurityGroupFilter, len(tags))
			for i := range tags {
				securityGroups[i].FilterByNeutronTags = tags[i]
			}
			machine.Spec.SecurityGroups = securityGroups
			Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail with invalid security group neutron tags")
		}

		for i := range tags {
			{
				machine := machine.DeepCopy()
				machine.Spec.Ports = []infrav1.PortOpts{
					{Network: &infrav1.NetworkFilter{FilterByNeutronTags: tags[i]}},
				}
				Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail with invalid port network neutron tags")
			}
			{
				machine := machine.DeepCopy()
				machine.Spec.Ports = []infrav1.PortOpts{
					{FixedIPs: []infrav1.FixedIP{{Subnet: &infrav1.SubnetFilter{FilterByNeutronTags: tags[i]}}}},
				}
				Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail with invalid port subnet neutron tags")
			}
			{
				machine := machine.DeepCopy()
				machine.Spec.Ports = []infrav1.PortOpts{
					{SecurityGroups: []infrav1.SecurityGroupFilter{{FilterByNeutronTags: tags[i]}}},
				}
				Expect(k8sClient.Create(ctx, machine)).NotTo(Succeed(), "OpenStackMachine creation should fail with invalid port security group neutron tags")
			}
		}

		if len(tags) > 0 {
			tag := tags[0]

			{
				cluster := cluster.DeepCopy()
				cluster.Spec.Subnets = []infrav1.SubnetFilter{{FilterByNeutronTags: tag}}
				Expect(k8sClient.Create(ctx, cluster)).NotTo(Succeed(), "OpenStackCluster creation should fail with invalid subnet neutron tags")
			}

			{
				cluster := cluster.DeepCopy()
				cluster.Spec.Network = &infrav1.NetworkFilter{FilterByNeutronTags: tag}
				Expect(k8sClient.Create(ctx, cluster)).NotTo(Succeed(), "OpenStackCluster creation should fail with invalid network neutron tags")
			}

			{
				cluster := cluster.DeepCopy()
				cluster.Spec.ExternalNetwork = &infrav1.NetworkFilter{FilterByNeutronTags: tag}
				Expect(k8sClient.Create(ctx, cluster)).NotTo(Succeed(), "OpenStackCluster creation should fail with invalid external network neutron tags")
			}

			{
				cluster := cluster.DeepCopy()
				cluster.Spec.Router = &infrav1.RouterFilter{FilterByNeutronTags: tag}
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
})
