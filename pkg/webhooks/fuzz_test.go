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

package webhooks

import (
	"context"
	"runtime/debug"
	"testing"

	"github.com/onsi/gomega/format"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	infrav1alpha1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

type pointerToObject[T any] interface {
	*T
	runtime.Object
}

// fuzzCustomValidator fuzzes a CustomValidator with objects of the validator's expected type.
func fuzzCustomValidator[O any, PO pointerToObject[O]](t *testing.T, name string, validator webhook.CustomValidator) {
	t.Helper()
	fuzz := utilconversion.GetFuzzer(scheme.Scheme)
	ctx := context.TODO()

	t.Run(name, func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			var previous PO = new(O)
			var dst PO = new(O)
			fuzz.Fill(previous)
			fuzz.Fill(dst)

			checkPanic := func(f func(), name string, args ...runtime.Object) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("PANIC in %s", name)
						for i, arg := range args {
							t.Errorf("arg %d:\n%s", i, format.Object(arg, 1))
						}
						t.Errorf("Stack trace:\n%s", debug.Stack())
						t.FailNow()
					}
				}()
				f()
			}

			checkPanic(func() {
				_, _ = validator.ValidateCreate(ctx, dst)
			}, "ValidateCreate()", dst)
			checkPanic(func() {
				_, _ = validator.ValidateUpdate(ctx, previous, dst)
			}, "ValidateUpdate()", previous, dst)
			checkPanic(func() {
				_, _ = validator.ValidateDelete(ctx, previous)
			}, "ValidateDelete()", previous)
		}
	})
}

func Test_FuzzClusterWebhook(t *testing.T) {
	fuzzCustomValidator[infrav1.OpenStackCluster](t, "OpenStackCluster", &openStackClusterWebhook{})
}

func Test_FuzzClusterTemplateWebhook(t *testing.T) {
	fuzzCustomValidator[infrav1.OpenStackClusterTemplate](t, "OpenStackClusterTemplate", &openStackClusterTemplateWebhook{})
}

func Test_FuzzMachineWebhook(t *testing.T) {
	fuzzCustomValidator[infrav1.OpenStackMachine](t, "OpenStackMachine", &openStackMachineWebhook{})
}

func Test_FuzzMachineTemplateWebhook(t *testing.T) {
	fuzzCustomValidator[infrav1.OpenStackMachineTemplate](t, "OpenStackMachineTemplate", &openStackMachineTemplateWebhook{})
}

func Test_FuzzServerWebhook(t *testing.T) {
	fuzzCustomValidator[infrav1alpha1.OpenStackServer](t, "OpenStackServer", &openStackServerWebhook{})
}
