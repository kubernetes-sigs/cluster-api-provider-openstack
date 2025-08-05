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

package ipam

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ipamv1 "sigs.k8s.io/cluster-api/api/ipam/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func processIfClaimReferencesPoolKind(gk metav1.GroupKind, obj client.Object) bool {
	var claim *ipamv1.IPAddressClaim
	var ok bool
	if claim, ok = obj.(*ipamv1.IPAddressClaim); !ok {
		return false
	}

	if claim.Spec.PoolRef.Kind != gk.Kind || claim.Spec.PoolRef.APIGroup == "" || claim.Spec.PoolRef.APIGroup != gk.Group {
		return false
	}

	return true
}

// ClaimReferencesPoolKind is a predicate that ensures an ipamv1.IPAddressClaim references a specified pool kind.
func ClaimReferencesPoolKind(gk metav1.GroupKind) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return processIfClaimReferencesPoolKind(gk, e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return processIfClaimReferencesPoolKind(gk, e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return processIfClaimReferencesPoolKind(gk, e.ObjectNew)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return processIfClaimReferencesPoolKind(gk, e.Object)
		},
	}
}

func processIfAddressReferencesPoolKind(gk metav1.GroupKind, obj client.Object) bool {
	var addr *ipamv1.IPAddress
	var ok bool
	if addr, ok = obj.(*ipamv1.IPAddress); !ok {
		return false
	}

	if addr.Spec.PoolRef.Kind != gk.Kind || addr.Spec.PoolRef.APIGroup == "" || addr.Spec.PoolRef.APIGroup != gk.Group {
		return false
	}

	return true
}

// AddressReferencesPoolKind is a predicate that ensures an ipamv1.IPAddress references a specified pool kind.
func AddressReferencesPoolKind(gk metav1.GroupKind) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return processIfAddressReferencesPoolKind(gk, e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return processIfAddressReferencesPoolKind(gk, e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return processIfAddressReferencesPoolKind(gk, e.ObjectNew)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return processIfAddressReferencesPoolKind(gk, e.Object)
		},
	}
}

func HasFinalizerAndIsDeleting(finalizer string) predicate.Funcs {
	return predicate.NewPredicateFuncs(func(obj client.Object) bool {
		return !obj.GetDeletionTimestamp().IsZero() && controllerutil.ContainsFinalizer(obj, finalizer)
	})
}
