/*
Copyright 2022 The Kubernetes Authors.

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

package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha6"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

// getIdentitySecret retrieves the identity secret for either a machine, or a cluster.
// This can be used to attach a finalizer so the secret is not deleted until both machine
// and cluster deprovisioning have completed.
func getIdentitySecret(ctx context.Context, ctrlClient client.Client, namespace string, identity *infrav1.OpenStackIdentityReference) (*corev1.Secret, error) {
	if identity == nil {
		return nil, fmt.Errorf("identity is not set")
	}

	if identity.Kind != "Secret" {
		return nil, fmt.Errorf("unsupported identity kind %v", identity.Kind)
	}

	secret := &corev1.Secret{}
	if err := ctrlClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: identity.Name}, secret); err != nil {
		return nil, err
	}

	return secret, nil
}

func getIdentitySecretFromCluster(ctx context.Context, ctrlClient client.Client, cluster *infrav1.OpenStackCluster) (*corev1.Secret, error) {
	return getIdentitySecret(ctx, ctrlClient, cluster.Namespace, cluster.Spec.IdentityRef)
}

func getIdentitySecretFromMachine(ctx context.Context, ctrlClient client.Client, machine *infrav1.OpenStackMachine) (*corev1.Secret, error) {
	return getIdentitySecret(ctx, ctrlClient, machine.Namespace, machine.Spec.IdentityRef)
}

// identitySecretFinalizerName creates unique finalizers for each machine, this assumes
// there is only one cluster using the secret.
func identitySecretFinalizerName(scope *scope.Scope, finalizer string) string {
	name := finalizer

	if scope.Identity.Machine != "" {
		name += "-" + scope.Identity.Machine
	}

	return name
}

// addIdentitySecretFinalizer adds a finalizer for either a machine or cluster to the identity
// secret.
func addIdentitySecretFinalizer(ctx context.Context, scope *scope.Scope, patchHelper *patch.Helper, finalizer string) error {
	controllerutil.AddFinalizer(scope.Identity.Secret, identitySecretFinalizerName(scope, finalizer))

	return patchHelper.Patch(ctx, scope.Identity.Secret)
}

// removeIdentitySecretFinalizer removes a finalizer from the identity secret for either
// a machine or cluster.
func removeIdentitySecretFinalizer(ctx context.Context, scope *scope.Scope, patchHelper *patch.Helper, finalizer string) error {
	controllerutil.RemoveFinalizer(scope.Identity.Secret, identitySecretFinalizerName(scope, finalizer))

	return patchHelper.Patch(ctx, scope.Identity.Secret)
}
