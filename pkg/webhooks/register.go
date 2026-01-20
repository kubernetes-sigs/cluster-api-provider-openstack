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
	"fmt"
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// isSchemeNotRegisteredError checks if the error indicates a type is not registered in the scheme.
func isSchemeNotRegisteredError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no kind is registered for the type")
}

func RegisterAllWithManager(mgr manager.Manager) []error {
	var errs []error

	// Register v1beta1 webhooks for all types with custom validators.
	for _, webhook := range []struct {
		name  string
		setup func(ctrl.Manager) error
	}{
		{"OpenStackCluster (v1beta1)", SetupOpenStackClusterWebhook},
		{"OpenStackClusterTemplate (v1beta1)", SetupOpenStackClusterTemplateWebhook},
		{"OpenStackMachine (v1beta1)", SetupOpenStackMachineWebhook},
		{"OpenStackMachineTemplate (v1beta1)", SetupOpenStackMachineTemplateWebhook},
		{"OpenStackServer", SetupOpenStackServerWebhook},
	} {
		if err := webhook.setup(mgr); err != nil {
			errs = append(errs, fmt.Errorf("creating webhook for %s: %v", webhook.name, err))
		}
	}

	// Register v1beta2 webhooks for all types with custom validators.
	// Skip gracefully if v1beta2 types are not yet registered.
	for _, webhook := range []struct {
		name  string
		setup func(ctrl.Manager) error
	}{
		{"OpenStackCluster (v1beta2)", SetupOpenStackClusterWebhookV1Beta2},
		{"OpenStackClusterTemplate (v1beta2)", SetupOpenStackClusterTemplateWebhookV1Beta2},
		{"OpenStackMachine (v1beta2)", SetupOpenStackMachineWebhookV1Beta2},
		{"OpenStackMachineTemplate (v1beta2)", SetupOpenStackMachineTemplateWebhookV1Beta2},
	} {
		if err := webhook.setup(mgr); err != nil {
			if isSchemeNotRegisteredError(err) {
				continue
			}
			errs = append(errs, fmt.Errorf("creating webhook for %s: %v", webhook.name, err))
		}
	}

	// Register conversion webhooks for List types
	for _, conversionOnlyType := range []conversion.Hub{
		&infrav1.OpenStackClusterList{},
		&infrav1.OpenStackClusterTemplateList{},
		&infrav1.OpenStackMachineList{},
		&infrav1.OpenStackMachineTemplateList{},
	} {
		if err := builder.WebhookManagedBy(mgr).
			For(conversionOnlyType).
			Complete(); err != nil {
			errs = append(errs, fmt.Errorf("creating webhook for %T: %v", conversionOnlyType, err))
		}
	}

	return errs
}
