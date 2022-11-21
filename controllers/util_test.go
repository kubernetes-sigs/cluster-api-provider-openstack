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

	"github.com/go-logr/logr"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

func getManager(ctx context.Context, logger logr.Logger, scheme *runtime.Scheme) (manager.Manager, func(), func()) {
	ctrl.SetLogger(logger)

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	ctx, cancel := context.WithCancel(ctx)
	run := func() {
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred(), "failed to run manager")
	}

	return mgr, run, cancel
}

func addMachineController(mgr manager.Manager, k8sClient client.Client, scopeFactory scope.Factory) { //nolint:unused
	err := (&OpenStackMachineReconciler{
		Client:       k8sClient,
		Recorder:     mgr.GetEventRecorderFor("openstackmachine-controller"),
		ScopeFactory: scopeFactory,
	}).SetupWithManager(ctx, mgr, controller.Options{})
	Expect(err).ToNot(HaveOccurred())
}

func addClusterController(mgr manager.Manager, k8sClient client.Client, scopeFactory scope.Factory) {
	err := (&OpenStackClusterReconciler{
		Client:       k8sClient,
		Recorder:     mgr.GetEventRecorderFor("openstackcluster-controller"),
		ScopeFactory: scopeFactory,
	}).SetupWithManager(ctx, mgr, controller.Options{})
	Expect(err).ToNot(HaveOccurred())
}
