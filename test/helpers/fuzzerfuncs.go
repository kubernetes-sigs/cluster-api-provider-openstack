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

package helpers

import (
	"slices"
	"strings"

	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/randfill"

	infrav1beta1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/utils/optional"
)

// ensureNonEmptyIDWhenSet ensures the ID field is non-empty when any field of the struct
// is set, since conversion uses ID != "" to detect whether the status object is present.
func ensureNonEmptyID(id *string, c randfill.Continue) {
	if *id == "" {
		*id = nonEmptyString(c)
	}
}

// fixupSchedulerHints ensures SchedulerHintAdditionalProperties have valid
// union values for v1beta2. When Type is "String", String must be non-empty.
// When Type is not "String", String must be "" (zero value).
func fixupSchedulerHintsV1Beta2(hints []infrav1.SchedulerHintAdditionalProperty, c randfill.Continue) {
	for i := range hints {
		// Ensure Name is non-empty (MinLength=1 validation)
		if hints[i].Name == "" {
			hints[i].Name = nonEmptyString(c)
		}
		v := &hints[i].Value
		switch c.Intn(3) {
		case 0:
			v.Type = "Bool"
			v.Bool = ptr.To(c.Bool())
			v.Number = nil
			v.String = ""
		case 1:
			v.Type = "String"
			v.Bool = nil
			v.Number = nil
			v.String = nonEmptyString(c)
		case 2:
			v.Type = "Number"
			v.Bool = nil
			v.Number = ptr.To(c.Int31())
			v.String = ""
		}
	}
}

// fixupSecurityGroupRulesV1Beta2 ensures SecurityGroupRuleSpec fields are
// valid for conversion round-trip.
func fixupSecurityGroupRulesV1Beta2(rules []infrav1.SecurityGroupRuleSpec, c randfill.Continue) {
	for i := range rules {
		if rules[i].Name == "" {
			rules[i].Name = nonEmptyString(c)
		}
		if rules[i].Direction == "" {
			if c.Bool() {
				rules[i].Direction = "ingress"
			} else {
				rules[i].Direction = "egress"
			}
		}
		// EtherType: string in v1beta2, *string in v1beta1.
		// Empty string is fine (represents "not set" via omitempty),
		// but non-empty must be a valid value.
		if rules[i].EtherType != "" && rules[i].EtherType != "IPv4" && rules[i].EtherType != "IPv6" {
			if c.Bool() {
				rules[i].EtherType = "IPv4"
			} else {
				rules[i].EtherType = "IPv6"
			}
		}
	}
}

// fixupOpenStackMachineSpecV1Beta2 ensures OpenStackMachineSpec fields are
// valid for conversion round-trip.
func fixupOpenStackMachineSpecV1Beta2(spec *infrav1.OpenStackMachineSpec, c randfill.Continue) {
	// Fix SchedulerHintAdditionalProperties for round-trip
	fixupSchedulerHintsV1Beta2(spec.SchedulerHintAdditionalProperties, c)
}

// fixupAPIServerLoadBalancerV1Beta2 ensures APIServerLoadBalancer fields
// round-trip correctly through v1beta1.
func fixupAPIServerLoadBalancerV1Beta2(lb *infrav1.APIServerLoadBalancer, c randfill.Continue) {
	// Monitor.Delay/Timeout/MaxRetries are *int32 in v1beta2 but int in v1beta1.
	// nil → 0 → &0, so nil doesn't round-trip. Ensure non-nil.
	if lb.Monitor != nil {
		if lb.Monitor.Delay == nil {
			lb.Monitor.Delay = ptr.To(c.Int31())
		}
		if lb.Monitor.Timeout == nil {
			lb.Monitor.Timeout = ptr.To(c.Int31())
		}
		if lb.Monitor.MaxRetries == nil {
			lb.Monitor.MaxRetries = ptr.To(c.Int31())
		}
	}

	// Network is NetworkParam (value type with omitzero).
	// Ensure valid structure: either zero or has ID/Filter.
	if lb.Network != (infrav1.NetworkParam{}) {
		if lb.Network.ID == nil && lb.Network.Filter == nil {
			lb.Network = infrav1.NetworkParam{ID: ptr.To(nonEmptyString(c))}
		}
	}

	// Subnets: each SubnetParam must have valid ID or Filter.
	for i := range lb.Subnets {
		if lb.Subnets[i].ID == nil && lb.Subnets[i].Filter == nil {
			lb.Subnets[i].ID = ptr.To(nonEmptyString(c))
		}
	}
}

// filterInvalidTags removes tags that are empty or contain commas, which are
// rejected by API validation in both v1beta1 and v1beta2.
func filterInvalidTags[T ~string](tags []T) []T {
	var ret []T
	for i := range tags {
		s := string(tags[i])
		if s != "" && !strings.Contains(s, ",") {
			ret = append(ret, tags[i])
		}
	}
	return ret
}

func nonEmptyString(c randfill.Continue) string {
	for {
		if s := c.String(20); s != "" {
			return s
		}
	}
}

type isZeroer[T any] interface {
	IsZero() bool
	*T
}

func fuzzFilterParam[Z isZeroer[T], T any](id *optional.String, filter *Z, c randfill.Continue) {
	if c.Bool() {
		*id = ptr.To(nonEmptyString(c))
		*filter = nil
	} else {
		*filter = new(T)
		for (*filter).IsZero() {
			c.Fill(*filter)
		}
		*id = nil
	}
}

// fuzzVolumeAZ generates a valid VolumeAvailabilityZone configuration.
// Extracted as a helper to share between v1beta1 and v1beta2 fuzzer funcs.
func fuzzVolumeAZ[FromT ~string, NameT ~string](from *FromT, name **NameT, c randfill.Continue) {
	stringWithoutSpaces := func() string {
		for {
			s := c.String(20)
			if !strings.Contains(s, " ") && s != "" {
				return s
			}
		}
	}

	// From is defaulted
	if c.Bool() {
		n := NameT(stringWithoutSpaces())
		*name = &n
		return
	}

	// From is Name
	if c.Bool() {
		*from = FromT("Name")
		n := NameT(stringWithoutSpaces())
		*name = &n
		return
	}

	// From is Machine
	*from = FromT("Machine")
}

// ensureValidFlavorParam sets exactly one of ID or Filter.Name to a non-empty
// value, ensuring the FlavorParam satisfies conversion requirements.
func ensureValidFlavorParam(param *infrav1.FlavorParam, c randfill.Continue) {
	if c.Bool() {
		*param = infrav1.FlavorParam{
			ID: ptr.To(nonEmptyString(c)),
		}
	} else {
		*param = infrav1.FlavorParam{
			Filter: infrav1.FlavorFilter{
				Name: ptr.To(nonEmptyString(c)),
			},
		}
	}
}

// InfraV1beta1FuzzerFuncs returns fuzzer funcs for v1beta1 OpenStack types which:
// * Constrain the output in ways which are validated by the API server
// * Add additional test coverage where it is not generated by the default fuzzer.
func InfraV1beta1FuzzerFuncs() []any {
	return []any{
		func(spec *infrav1beta1.OpenStackClusterSpec, c randfill.Continue) {
			c.FillNoCustom(spec)

			// The fuzzer only seems to generate Subnets of
			// length 1, but we need to also test length 2.
			// Ensure it is occasionally generated.
			if len(spec.Subnets) == 1 && c.Bool() {
				subnet := infrav1beta1.SubnetParam{}
				c.Fill(&subnet)
				spec.Subnets = append(spec.Subnets, subnet)
			}

			// Fix SecurityGroupRuleSpec fields that don't round-trip
			// when *string is nil (nil → "" → &"" after round-trip).
			if spec.ManagedSecurityGroups != nil {
				for i := range spec.ManagedSecurityGroups.AllNodesSecurityGroupRules {
					rule := &spec.ManagedSecurityGroups.AllNodesSecurityGroupRules[i]
					// EtherType is *string in v1beta1, string in v1beta2.
					// nil doesn't round-trip (nil → "" → &""), so ensure non-nil.
					if rule.EtherType == nil {
						if c.Bool() {
							rule.EtherType = ptr.To("IPv4")
						} else {
							rule.EtherType = ptr.To("IPv6")
						}
					}
				}
			}
		},

		func(spec *infrav1beta1.SubnetSpec, c randfill.Continue) {
			c.FillNoCustom(spec)

			// CIDR is required and API validates that it's present, so
			// we force it to always be set.
			for spec.CIDR == "" {
				spec.CIDR = c.String(20)
			}
		},

		func(pool *infrav1beta1.AllocationPool, c randfill.Continue) {
			c.FillNoCustom(pool)

			// Start and End are required properties, let's make sure both are set
			for pool.Start == "" {
				pool.Start = c.String(20)
			}

			for pool.End == "" {
				pool.End = c.String(20)
			}
		},

		// v1beta1 filter tags cannot contain commas and can't be empty.
		func(filter *infrav1beta1.FilterByNeutronTags, c randfill.Continue) {
			c.FillNoCustom(filter)

			// Sometimes add an additional tag to ensure we get test coverage of multiple tags
			if c.Bool() {
				filter.Tags = append(filter.Tags, infrav1beta1.NeutronTag(c.String(20)))
			}
			if c.Bool() {
				filter.TagsAny = append(filter.TagsAny, infrav1beta1.NeutronTag(c.String(20)))
			}
			if c.Bool() {
				filter.NotTags = append(filter.NotTags, infrav1beta1.NeutronTag(c.String(20)))
			}
			if c.Bool() {
				filter.NotTagsAny = append(filter.NotTagsAny, infrav1beta1.NeutronTag(c.String(20)))
			}

			// Remove empty tags and tags with commas
			filter.Tags = filterInvalidTags(filter.Tags)
			filter.TagsAny = filterInvalidTags(filter.TagsAny)
			filter.NotTags = filterInvalidTags(filter.NotTags)
			filter.NotTagsAny = filterInvalidTags(filter.NotTagsAny)
		},

		// v1beta1 filter params contain exactly one of ID or filter
		func(param *infrav1beta1.NetworkParam, c randfill.Continue) {
			fuzzFilterParam(&param.ID, &param.Filter, c)
		},

		func(param *infrav1beta1.SubnetParam, c randfill.Continue) {
			fuzzFilterParam(&param.ID, &param.Filter, c)
		},

		func(param *infrav1beta1.SecurityGroupParam, c randfill.Continue) {
			fuzzFilterParam(&param.ID, &param.Filter, c)
		},

		func(param *infrav1beta1.ImageParam, c randfill.Continue) {
			fuzzFilterParam(&param.ID, &param.Filter, c)
		},

		func(param *infrav1beta1.RouterParam, c randfill.Continue) {
			fuzzFilterParam(&param.ID, &param.Filter, c)
		},

		// Ensure at least one of Flavor or FlavorID is set (required for conversion)
		func(spec *infrav1beta1.OpenStackMachineSpec, c randfill.Continue) {
			c.FillNoCustom(spec)

			// Ensure non-empty if set (MinLength=1 validation)
			if spec.Flavor != nil && *spec.Flavor == "" {
				s := nonEmptyString(c)
				spec.Flavor = &s
			}
			if spec.FlavorID != nil && *spec.FlavorID == "" {
				s := nonEmptyString(c)
				spec.FlavorID = &s
			}
			// Ensure at least one is set
			if spec.Flavor == nil && spec.FlavorID == nil {
				if c.Bool() {
					s := nonEmptyString(c)
					spec.FlavorID = &s
				} else {
					s := nonEmptyString(c)
					spec.Flavor = &s
				}
			}

			// Fix SchedulerHintAdditionalValue.String: *string in v1beta1, string in v1beta2.
			// nil doesn't round-trip (nil → "" → &""), so ensure non-nil.
			for i := range spec.SchedulerHintAdditionalProperties {
				v := &spec.SchedulerHintAdditionalProperties[i].Value
				if v.String == nil {
					v.String = ptr.To("")
				}
				// Ensure Number fits in int32 for conversion
				if v.Number != nil {
					n := int(int32(*v.Number)) //nolint:gosec // intentional truncation to constrain fuzzer output to int32 range
					v.Number = &n
				}
			}
		},

		// Ensure APIServerLoadBalancer is nil in v1beta1 status to avoid
		// triggering the unsafe.Pointer cast that reads past struct bounds
		// (v1beta1.LoadBalancer is smaller than v1beta2.LoadBalancer due to
		// LoadBalancerNetwork changing from pointer to value type).
		func(status *infrav1beta1.OpenStackClusterStatus, c randfill.Continue) {
			c.FillNoCustom(status)
			status.APIServerLoadBalancer = nil
		},

		// Ensure VolumeAZ type is valid
		func(az *infrav1beta1.VolumeAvailabilityZone, c randfill.Continue) {
			fuzzVolumeAZ(&az.From, &az.Name, c)
		},

		// SecurityGroupRuleSpec: ensure EtherType is never nil
		func(rule *infrav1beta1.SecurityGroupRuleSpec, c randfill.Continue) {
			c.FillNoCustom(rule)
			// EtherType is *string in v1beta1, string in v1beta2.
			// nil doesn't round-trip, so always set it.
			if rule.EtherType == nil {
				if c.Bool() {
					rule.EtherType = ptr.To("IPv4")
				} else {
					rule.EtherType = ptr.To("IPv6")
				}
			}
			// Ensure Number fits in int32
			if rule.PortRangeMin != nil {
				n := int(int32(*rule.PortRangeMin)) //nolint:gosec // intentional truncation to constrain fuzzer output to int32 range
				rule.PortRangeMin = &n
			}
			if rule.PortRangeMax != nil {
				n := int(int32(*rule.PortRangeMax)) //nolint:gosec // intentional truncation to constrain fuzzer output to int32 range
				rule.PortRangeMax = &n
			}
		},

		// SchedulerHintAdditionalValue: ensure String is never nil
		func(val *infrav1beta1.SchedulerHintAdditionalValue, c randfill.Continue) {
			c.FillNoCustom(val)
			// String is *string in v1beta1, string in v1beta2.
			// nil doesn't round-trip (nil → "" → &""), use &"" instead.
			if val.String == nil {
				val.String = ptr.To("")
			}
			// Ensure Number fits in int32
			if val.Number != nil {
				n := int(int32(*val.Number)) //nolint:gosec // intentional truncation to constrain fuzzer output to int32 range
				val.Number = &n
			}
		},
	}
}

func fuzzManagedNetwork(mn **infrav1.ManagedNetwork, c randfill.Continue) {
	if c.Bool() {
		*mn = nil
		return
	}
	m := &infrav1.ManagedNetwork{}
	c.Fill(m)
	if m.MTU == nil && m.EnablePortSecurity == nil {
		m.MTU = ptr.To(c.Int31())
	}
	*mn = m
}

func fuzzManagedRouter(mr **infrav1.ManagedRouter, c randfill.Continue) {
	if c.Bool() {
		*mr = nil
		return
	}
	m := &infrav1.ManagedRouter{}
	c.Fill(m)
	if len(m.ExternalIPs) == 0 {
		ip := infrav1.ExternalRouterIPParam{}
		c.Fill(&ip)
		m.ExternalIPs = []infrav1.ExternalRouterIPParam{ip}
	}
	*mr = m
}

func fuzzAPIServer(as **infrav1.APIServer, c randfill.Continue) {
	if c.Bool() {
		*as = nil
		return
	}
	a := &infrav1.APIServer{}
	c.Fill(a)
	if a.FloatingIP == nil &&
		a.FixedIP == nil &&
		a.Port == nil &&
		a.EnableFloatingIP == nil &&
		a.ManagedLoadBalancer == nil {
		a.FloatingIP = ptr.To(nonEmptyString(c))
	}
	*as = a
}

// InfraV1Beta2FuzzerFuncs returns fuzzer funcs for v1beta2 OpenStack types which:
// * Constrain the output in ways which are validated by the API server
// * Constrain fields that are not preserved during v1beta2 <-> v1beta1 round-trip conversion
// * Add additional test coverage where it is not generated by the default fuzzer.
func InfraV1Beta2FuzzerFuncs() []any { //nolint:gocyclo,cyclop
	return []any{
		// Normalize OpenStackCluster fields that are not preserved during
		// hub-spoke-hub conversion:
		// - ObservedGeneration on conditions is set from ObjectMeta.Generation during ConvertTo
		// - FailureDomains ordering is lost during map<->slice conversion
		func(cluster *infrav1.OpenStackCluster, c randfill.Continue) {
			c.FillNoCustom(cluster)

			// FillNoCustom does not invoke custom fuzzers for nested types,
			// so we explicitly ensure Flavor fields are valid for conversion.
			if cluster.Spec.Bastion != nil {
				ensureValidFlavorParam(&cluster.Spec.Bastion.Spec.Flavor, c)
				fixupOpenStackMachineSpecV1Beta2(&cluster.Spec.Bastion.Spec, c)
			}

			// Fix APIServer.ManagedLoadBalancer fields
			if cluster.Spec.APIServer != nil && cluster.Spec.APIServer.ManagedLoadBalancer != nil {
				fixupAPIServerLoadBalancerV1Beta2(cluster.Spec.APIServer.ManagedLoadBalancer, c)
			}

			// Fix ManagedSecurityGroups rules
			if cluster.Spec.ManagedSecurityGroups != nil {
				fixupSecurityGroupRulesV1Beta2(cluster.Spec.ManagedSecurityGroups.ClusterNodesSecurityGroupRules, c)
			}

			// Fix status fields: conversion uses .ID != "" to detect presence
			ensureNonEmptyID(&cluster.Status.Network.ID, c)
			ensureNonEmptyID(&cluster.Status.ExternalNetwork.ID, c)
			ensureNonEmptyID(&cluster.Status.Router.ID, c)
			ensureNonEmptyID(&cluster.Status.ControlPlaneSecurityGroup.ID, c)
			ensureNonEmptyID(&cluster.Status.WorkerSecurityGroup.ID, c)
			ensureNonEmptyID(&cluster.Status.BastionSecurityGroup.ID, c)
			// APIServerManagedLoadBalancer has an unsafe cast issue with
			// LoadBalancerNetwork (pointer vs value type layout mismatch).
			cluster.Status.APIServerManagedLoadBalancer = infrav1.LoadBalancer{}

			for i := range cluster.Status.Conditions {
				cluster.Status.Conditions[i].ObservedGeneration = cluster.Generation
			}
			slices.SortFunc(cluster.Status.FailureDomains, func(a, b clusterv1.FailureDomain) int {
				return strings.Compare(a.Name, b.Name)
			})
		},

		// Normalize OpenStackMachine ObservedGeneration (set from ObjectMeta.Generation during ConvertTo).
		func(machine *infrav1.OpenStackMachine, c randfill.Continue) {
			c.FillNoCustom(machine)

			// FillNoCustom does not invoke custom fuzzers for nested types,
			// so we explicitly ensure Flavor is valid for conversion.
			ensureValidFlavorParam(&machine.Spec.Flavor, c)
			fixupOpenStackMachineSpecV1Beta2(&machine.Spec, c)

			for i := range machine.Status.Conditions {
				machine.Status.Conditions[i].ObservedGeneration = machine.Generation
			}
		},

		// Normalize OpenStackMachineTemplate ObservedGeneration.
		// The template ConvertTo does not set ObservedGeneration, so it is
		// always zero after a hub-spoke-hub round-trip.
		func(tmpl *infrav1.OpenStackMachineTemplate, c randfill.Continue) {
			c.FillNoCustom(tmpl)

			// FillNoCustom does not invoke custom fuzzers for nested types,
			// so we explicitly ensure Flavor is valid for conversion.
			ensureValidFlavorParam(&tmpl.Spec.Template.Spec.Flavor, c)
			fixupOpenStackMachineSpecV1Beta2(&tmpl.Spec.Template.Spec, c)

			for i := range tmpl.Status.Conditions {
				tmpl.Status.Conditions[i].ObservedGeneration = 0
			}
		},

		func(spec *infrav1.OpenStackClusterSpec, c randfill.Continue) {
			c.FillNoCustom(spec)

			// The fuzzer only seems to generate Subnets of
			// length 1, but we need to also test length 2.
			// Ensure it is occasionally generated.
			if len(spec.Subnets) == 1 && c.Bool() {
				subnet := infrav1.SubnetParam{}
				c.Fill(&subnet)
				spec.Subnets = append(spec.Subnets, subnet)
			}

			// Fix APIServer.ManagedLoadBalancer fields
			if spec.APIServer != nil && spec.APIServer.ManagedLoadBalancer != nil {
				fixupAPIServerLoadBalancerV1Beta2(spec.APIServer.ManagedLoadBalancer, c)
			}

			// Fix ManagedSecurityGroups rules
			if spec.ManagedSecurityGroups != nil {
				fixupSecurityGroupRulesV1Beta2(spec.ManagedSecurityGroups.ClusterNodesSecurityGroupRules, c)
			}

			// Fix Bastion.Spec if present
			if spec.Bastion != nil {
				ensureValidFlavorParam(&spec.Bastion.Spec.Flavor, c)
				fixupOpenStackMachineSpecV1Beta2(&spec.Bastion.Spec, c)
			}
		},

		func(spec *infrav1.SubnetSpec, c randfill.Continue) {
			c.FillNoCustom(spec)

			// CIDR is required and API validates that it's present, so
			// we force it to always be set.
			for spec.CIDR == "" {
				spec.CIDR = c.String(20)
			}
		},

		func(pool *infrav1.AllocationPool, c randfill.Continue) {
			c.FillNoCustom(pool)

			// Start and End are required properties, let's make sure both are set
			for pool.Start == "" {
				pool.Start = c.String(20)
			}

			for pool.End == "" {
				pool.End = c.String(20)
			}
		},

		// v1beta2 filter tags cannot contain commas and can't be empty.
		func(filter *infrav1.FilterByNeutronTags, c randfill.Continue) {
			c.FillNoCustom(filter)

			// Sometimes add an additional tag to ensure we get test coverage of multiple tags
			if c.Bool() {
				filter.Tags = append(filter.Tags, infrav1.NeutronTag(c.String(20)))
			}
			if c.Bool() {
				filter.TagsAny = append(filter.TagsAny, infrav1.NeutronTag(c.String(20)))
			}
			if c.Bool() {
				filter.NotTags = append(filter.NotTags, infrav1.NeutronTag(c.String(20)))
			}
			if c.Bool() {
				filter.NotTagsAny = append(filter.NotTagsAny, infrav1.NeutronTag(c.String(20)))
			}

			// Remove empty tags and tags with commas
			filter.Tags = filterInvalidTags(filter.Tags)
			filter.TagsAny = filterInvalidTags(filter.TagsAny)
			filter.NotTags = filterInvalidTags(filter.NotTags)
			filter.NotTagsAny = filterInvalidTags(filter.NotTagsAny)
		},

		// v1beta2 filter params contain exactly one of ID or filter
		func(param *infrav1.NetworkParam, c randfill.Continue) {
			fuzzFilterParam(&param.ID, &param.Filter, c)
		},

		func(param *infrav1.SubnetParam, c randfill.Continue) {
			fuzzFilterParam(&param.ID, &param.Filter, c)
		},

		func(param *infrav1.SecurityGroupParam, c randfill.Continue) {
			fuzzFilterParam(&param.ID, &param.Filter, c)
		},

		func(param *infrav1.ImageParam, c randfill.Continue) {
			if c.Bool() {
				param.ID = ptr.To(nonEmptyString(c))
				param.Filter = infrav1.ImageFilter{}
			} else {
				param.ID = nil
				for (&param.Filter).IsZero() {
					c.Fill(&param.Filter)
				}
			}
		},

		func(param *infrav1.RouterParam, c randfill.Continue) {
			fuzzFilterParam(&param.ID, &param.Filter, c)
		},

		// Ensure FlavorParam has exactly one of ID or Filter with non-empty Name.
		// This mirrors the MinProperties/MaxProperties validation on FlavorParam.
		func(param *infrav1.FlavorParam, c randfill.Continue) {
			ensureValidFlavorParam(param, c)
		},

		// Ensure VolumeAZ type is valid
		func(az *infrav1.VolumeAvailabilityZone, c randfill.Continue) {
			stringWithoutSpaces := func() string {
				for {
					s := c.String(20)
					if !strings.Contains(s, " ") && s != "" {
						return s
					}
				}
			}

			// From is defaulted
			if c.Bool() {
				az.Name = infrav1.VolumeAZName(stringWithoutSpaces())
				return
			}

			// From is Name
			if c.Bool() {
				az.From = infrav1.VolumeAZSource("Name")
				az.Name = infrav1.VolumeAZName(stringWithoutSpaces())
				return
			}

			// From is Machine
			az.From = infrav1.VolumeAZSource("Machine")
		},

		// FailureDomains are converted via a map keyed by Name, so names
		// must be unique and non-empty. ControlPlane must be non-nil because
		// nil is converted to *false via ptr.Deref/ptr.To and won't
		// round-trip back to nil.
		func(fd *clusterv1.FailureDomain, c randfill.Continue) {
			c.FillNoCustom(fd)

			for fd.Name == "" {
				fd.Name = c.String(20)
			}
			if fd.ControlPlane == nil {
				fd.ControlPlane = ptr.To(c.Bool())
			}
		},

		// Deduplicate FailureDomain names at the status level since map
		// conversion collapses duplicates.
		func(status *infrav1.OpenStackClusterStatus, c randfill.Continue) {
			c.FillNoCustom(status)

			seen := map[string]bool{}
			unique := make([]clusterv1.FailureDomain, 0, len(status.FailureDomains))
			for _, fd := range status.FailureDomains {
				if !seen[fd.Name] {
					seen[fd.Name] = true
					unique = append(unique, fd)
				}
			}

			// Normalize empty slice to nil: v1beta1 round-trip cannot
			// distinguish between nil and empty slice.
			if len(unique) == 0 {
				status.FailureDomains = nil
			} else {
				status.FailureDomains = unique
			}

			// Conversion uses .ID != "" to detect whether status objects are set.
			// Ensure IDs are non-empty when other fields are populated.
			ensureNonEmptyID(&status.Network.ID, c)
			ensureNonEmptyID(&status.ExternalNetwork.ID, c)
			ensureNonEmptyID(&status.Router.ID, c)
			ensureNonEmptyID(&status.ControlPlaneSecurityGroup.ID, c)
			ensureNonEmptyID(&status.WorkerSecurityGroup.ID, c)
			ensureNonEmptyID(&status.BastionSecurityGroup.ID, c)

			// APIServerManagedLoadBalancer has an unsafe cast issue:
			// v1beta1.LoadBalancer and v1beta2.LoadBalancer have different
			// sizes due to LoadBalancerNetwork changing from pointer to value.
			status.APIServerManagedLoadBalancer = infrav1.LoadBalancer{}
		},

		func(mn **infrav1.ManagedNetwork, c randfill.Continue) {
			fuzzManagedNetwork(mn, c)
		},
		// ...
		func(mr **infrav1.ManagedRouter, c randfill.Continue) {
			fuzzManagedRouter(mr, c)
		},
		// ...
		func(as **infrav1.APIServer, c randfill.Continue) {
			fuzzAPIServer(as, c)
		},
	}
}
