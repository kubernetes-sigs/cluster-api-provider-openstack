/*
Copyright 2026 The Kubernetes Authors.

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

package conversion

import (
	"slices"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"

	infrav1beta1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
	testhelpers "sigs.k8s.io/cluster-api-provider-openstack/test/helpers"
)

// Setting this to false to avoid running tests in parallel. Only for use in development.
const parallel = true

func runParallel(f func(t *testing.T)) func(t *testing.T) {
	if parallel {
		return func(t *testing.T) {
			t.Helper()
			t.Parallel()
			f(t)
		}
	}
	return f
}

func TestFuzzyConversion(t *testing.T) {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := infrav1beta1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := infrav1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	fuzzerFuncs := func(_ runtimeserializer.CodecFactory) []interface{} {
		return slices.Concat(
			testhelpers.InfraV1beta1FuzzerFuncs(),
			testhelpers.InfraV1Beta2FuzzerFuncs(),
		)
	}

	// ignoreDataAnnotation removes the data annotation that is added by
	// MarshalData during conversion to the hub. It is not present on the
	// original hub object, so it must be removed before comparison.
	ignoreDataAnnotation := func(obj metav1.Object) {
		annotations := obj.GetAnnotations()
		delete(annotations, utilconversion.DataAnnotation)
		if len(annotations) == 0 {
			obj.SetAnnotations(nil)
		}
	}

	// clusterHubAfterMutation removes the data annotation and sorts
	// FailureDomains by Name to normalize non-deterministic map iteration
	// order from the FailureDomains map↔slice conversion.
	clusterHubAfterMutation := func(cluster *infrav1.OpenStackCluster) {
		annotations := cluster.GetAnnotations()
		delete(annotations, utilconversion.DataAnnotation)
		if len(annotations) == 0 {
			cluster.SetAnnotations(nil)
		}
		slices.SortFunc(cluster.Status.FailureDomains, func(a, b clusterv1.FailureDomain) int {
			return strings.Compare(a.Name, b.Name)
		})
	}

	t.Run("for OpenStackCluster", runParallel(utilconversion.SpokeConverterFuzzTestFunc(utilconversion.SpokeConverterFuzzTestFuncInput[*infrav1.OpenStackCluster, *infrav1beta1.OpenStackCluster]{
		Scheme:                scheme,
		ConvertHubToSpokeFunc: ConvertOpenStackClusterHubToV1Beta1,
		ConvertSpokeToHubFunc: ConvertOpenStackClusterV1Beta1ToHub,
		HubAfterMutation:      clusterHubAfterMutation,
		FuzzerFuncs:           []fuzzer.FuzzerFuncs{fuzzerFuncs},
	})))

	t.Run("for OpenStackCluster with mutate", runParallel(testhelpers.FuzzMutateTestFunc(testhelpers.FuzzMutateTestFuncInput[*infrav1.OpenStackCluster, *infrav1beta1.OpenStackCluster]{
		SpokeConverterFuzzTestFuncInput: utilconversion.SpokeConverterFuzzTestFuncInput[*infrav1.OpenStackCluster, *infrav1beta1.OpenStackCluster]{
			Scheme:                scheme,
			ConvertHubToSpokeFunc: ConvertOpenStackClusterHubToV1Beta1,
			ConvertSpokeToHubFunc: ConvertOpenStackClusterV1Beta1ToHub,
			HubAfterMutation:      clusterHubAfterMutation,
			FuzzerFuncs:           []fuzzer.FuzzerFuncs{fuzzerFuncs},
		},
		MutateFuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzerFuncs},
	})))

	t.Run("for OpenStackClusterTemplate", runParallel(utilconversion.SpokeConverterFuzzTestFunc(utilconversion.SpokeConverterFuzzTestFuncInput[*infrav1.OpenStackClusterTemplate, *infrav1beta1.OpenStackClusterTemplate]{
		Scheme:                scheme,
		ConvertHubToSpokeFunc: ConvertOpenStackClusterTemplateHubToV1Beta1,
		ConvertSpokeToHubFunc: ConvertOpenStackClusterTemplateV1Beta1ToHub,
		HubAfterMutation:      func(hub *infrav1.OpenStackClusterTemplate) { ignoreDataAnnotation(hub) },
		FuzzerFuncs:           []fuzzer.FuzzerFuncs{fuzzerFuncs},
	})))

	t.Run("for OpenStackClusterTemplate with mutate", runParallel(testhelpers.FuzzMutateTestFunc(testhelpers.FuzzMutateTestFuncInput[*infrav1.OpenStackClusterTemplate, *infrav1beta1.OpenStackClusterTemplate]{
		SpokeConverterFuzzTestFuncInput: utilconversion.SpokeConverterFuzzTestFuncInput[*infrav1.OpenStackClusterTemplate, *infrav1beta1.OpenStackClusterTemplate]{
			Scheme:                scheme,
			ConvertHubToSpokeFunc: ConvertOpenStackClusterTemplateHubToV1Beta1,
			ConvertSpokeToHubFunc: ConvertOpenStackClusterTemplateV1Beta1ToHub,
			HubAfterMutation:      func(hub *infrav1.OpenStackClusterTemplate) { ignoreDataAnnotation(hub) },
			FuzzerFuncs:           []fuzzer.FuzzerFuncs{fuzzerFuncs},
		},
		MutateFuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzerFuncs},
	})))

	t.Run("for OpenStackMachine", runParallel(utilconversion.SpokeConverterFuzzTestFunc(utilconversion.SpokeConverterFuzzTestFuncInput[*infrav1.OpenStackMachine, *infrav1beta1.OpenStackMachine]{
		Scheme:                scheme,
		ConvertHubToSpokeFunc: ConvertOpenStackMachineHubToV1Beta1,
		ConvertSpokeToHubFunc: ConvertOpenStackMachineV1Beta1ToHub,
		HubAfterMutation:      func(hub *infrav1.OpenStackMachine) { ignoreDataAnnotation(hub) },
		FuzzerFuncs:           []fuzzer.FuzzerFuncs{fuzzerFuncs},
	})))

	t.Run("for OpenStackMachine with mutate", runParallel(testhelpers.FuzzMutateTestFunc(testhelpers.FuzzMutateTestFuncInput[*infrav1.OpenStackMachine, *infrav1beta1.OpenStackMachine]{
		SpokeConverterFuzzTestFuncInput: utilconversion.SpokeConverterFuzzTestFuncInput[*infrav1.OpenStackMachine, *infrav1beta1.OpenStackMachine]{
			Scheme:                scheme,
			ConvertHubToSpokeFunc: ConvertOpenStackMachineHubToV1Beta1,
			ConvertSpokeToHubFunc: ConvertOpenStackMachineV1Beta1ToHub,
			HubAfterMutation:      func(hub *infrav1.OpenStackMachine) { ignoreDataAnnotation(hub) },
			FuzzerFuncs:           []fuzzer.FuzzerFuncs{fuzzerFuncs},
		},
		MutateFuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzerFuncs},
	})))

	t.Run("for OpenStackMachineTemplate", runParallel(utilconversion.SpokeConverterFuzzTestFunc(utilconversion.SpokeConverterFuzzTestFuncInput[*infrav1.OpenStackMachineTemplate, *infrav1beta1.OpenStackMachineTemplate]{
		Scheme:                scheme,
		ConvertHubToSpokeFunc: ConvertOpenStackMachineTemplateHubToV1Beta1,
		ConvertSpokeToHubFunc: ConvertOpenStackMachineTemplateV1Beta1ToHub,
		HubAfterMutation:      func(hub *infrav1.OpenStackMachineTemplate) { ignoreDataAnnotation(hub) },
		FuzzerFuncs:           []fuzzer.FuzzerFuncs{fuzzerFuncs},
	})))

	t.Run("for OpenStackMachineTemplate with mutate", runParallel(testhelpers.FuzzMutateTestFunc(testhelpers.FuzzMutateTestFuncInput[*infrav1.OpenStackMachineTemplate, *infrav1beta1.OpenStackMachineTemplate]{
		SpokeConverterFuzzTestFuncInput: utilconversion.SpokeConverterFuzzTestFuncInput[*infrav1.OpenStackMachineTemplate, *infrav1beta1.OpenStackMachineTemplate]{
			Scheme:                scheme,
			ConvertHubToSpokeFunc: ConvertOpenStackMachineTemplateHubToV1Beta1,
			ConvertSpokeToHubFunc: ConvertOpenStackMachineTemplateV1Beta1ToHub,
			HubAfterMutation:      func(hub *infrav1.OpenStackMachineTemplate) { ignoreDataAnnotation(hub) },
			FuzzerFuncs:           []fuzzer.FuzzerFuncs{fuzzerFuncs},
		},
		MutateFuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzerFuncs},
	})))
}
