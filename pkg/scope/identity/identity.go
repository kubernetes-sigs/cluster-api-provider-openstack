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

// package identity provides services around the provider's identity.
// In particular, we want to have a create X Y Z, delete X Y Z model
// that is compatible with Helm, ArgoCD etc.  To do this we need to
// keep the identity alive until provider resources have been
// deprovisioned.  Owner references cannot be used, as they keep the
// identity alive until all owners are deleted, which evidently means
// the timestamp has been set, not necessarily the finalizers removed
// and resources removed from etcd.
package identity

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha6"
)

// finalizerName returns the fully qualified finalizer name for the
// resource in question.  For example, there are many machines, and they
// all need a unique finalizer or else the deletion of the first may result
// in the deletion of the identity.  Where it gets tricky is Kubernetes
// will reject non-standard finalizers, non-fully qualified e.g.
// domain/thing, or exceed 63 characters, so we have to hash the name
// into a fixed length string.
func finalizerName(finalizer, name string) string {
	// If we use "openstackmachine.infrastructure.cluster.x-k8s.io" as
	// the baseline, that's 48 characters...
	hash := sha256.Sum256([]byte(name))

	// ...then 4 bytes of hex and the slash take us to 57.
	return fmt.Sprintf("%s/%x", finalizer, hash[:4])
}

type Scope struct {
	// finalizer is the unique finalizer for the resource kind and name.
	finalizer string

	// client is used to access the identity.
	client client.Client

	// namespace is where the cluster, and by extension, the identity
	// lives.
	namespace string

	// identity is the identity reference.  This should not be changed
	// during the lifetime of the cluster (e.g. be the same resource),
	// or the finalizers will diverge from reality.  However you can
	// modify it to rotate the credentials.
	identity *infrav1.OpenStackIdentityReference
}

// NewForCluster configures the identity scope for an OpenStackCluster.
func NewForCluster(ctx context.Context, ctrlClient client.Client, r *infrav1.OpenStackCluster) (*Scope, error) {
	scope := &Scope{
		finalizer: finalizerName(infrav1.ClusterFinalizer, r.Name),
		client:    ctrlClient,
		namespace: r.Namespace,
		identity:  r.Spec.IdentityRef,
	}

	if err := scope.init(ctx); err != nil {
		return nil, err
	}

	return scope, nil
}

// NewForMachine configures the identity scope for an OpenStackMachine.
func NewForMachine(ctx context.Context, ctrlClient client.Client, r *infrav1.OpenStackMachine) (*Scope, error) {
	scope := &Scope{
		finalizer: finalizerName(infrav1.MachineFinalizer, r.Name),
		client:    ctrlClient,
		namespace: r.Namespace,
		identity:  r.Spec.IdentityRef,
	}

	if err := scope.init(ctx); err != nil {
		return nil, err
	}

	return scope, nil
}

// init gets the identity and sets up any additional clients.
func (s *Scope) init(ctx context.Context) error {
	if s.identity == nil {
		return fmt.Errorf("identity is not set")
	}

	if s.identity.Kind != "Secret" {
		return fmt.Errorf("unsupported identity kind %v", s.identity.Kind)
	}

	return nil
}

// AddFinalizer adds the finalizer to the identity.
func (s Scope) AddFinalizer(ctx context.Context) error {
	return s.retryPatch(ctx, controllerutil.AddFinalizer)
}

// RemoveFinalizer removes the finalizer from the identity.
func (s Scope) RemoveFinalizer(ctx context.Context) error {
	return s.retryPatch(ctx, controllerutil.RemoveFinalizer)
}

// retryPatch recognizes that adding and removing multiple finalizers is a race condition.
// It will reload the identity, mutate it and commit the result.  It'll do this in a loop
// to avoid CAS (complare-and-swap) conflicts.
func (s Scope) retryPatch(ctx context.Context, mutator func(client.Object, string) bool) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var err error

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("%w: %s", timeoutCtx.Err(), err.Error())
		case <-ticker.C:
			if err = s.patch(ctx, mutator); err == nil {
				return nil
			}
		}
	}
}

// patch does the actual patch operation, standard read/modify/write to avoid conflicts.
func (s Scope) patch(ctx context.Context, mutator func(client.Object, string) bool) error {
	key := client.ObjectKey{
		Namespace: s.namespace,
		Name:      s.identity.Name,
	}

	secret := &corev1.Secret{}
	if err := s.client.Get(ctx, key, secret); err != nil {
		return err
	}

	patchHelper, err := patch.NewHelper(secret, s.client)
	if err != nil {
		return err
	}

	mutator(secret, s.finalizer)

	if err := patchHelper.Patch(ctx, secret); err != nil {
		return err
	}

	return nil
}
