/*
Copyright 2018 The Kubernetes Authors.

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

package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/controller"

	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	_ "net/http/pprof"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha2"
	"sigs.k8s.io/cluster-api-provider-openstack/controllers"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha2"
	ctrl "sigs.k8s.io/controller-runtime"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = infrav1.AddToScheme(scheme)
	_ = clusterv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	klog.InitFlags(nil)

	var (
		metricsAddr                 string
		enableLeaderElection        bool
		watchNamespace              string
		profilerAddress             string
		openStackClusterConcurrency int
		openStackMachineConcurrency int
		syncPeriod                  time.Duration
	)

	flag.StringVar(
		&metricsAddr,
		"metrics-addr",
		":8080",
		"The address the metric endpoint binds to.",
	)

	flag.BoolVar(
		&enableLeaderElection,
		"enable-leader-election",
		false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.",
	)

	flag.StringVar(
		&watchNamespace,
		"namespace",
		"",
		"Namespace that the controller watches to reconcile cluster-api objects. If unspecified, the controller watches for cluster-api objects across all namespaces.",
	)

	flag.StringVar(
		&profilerAddress,
		"profiler-address",
		"",
		"Bind address to expose the pprof profiler (e.g. localhost:6060)",
	)

	flag.IntVar(&openStackClusterConcurrency,
		"openstackcluster-concurrency",
		1,
		"Number of OpenStackClusters to process simultaneously",
	)

	flag.IntVar(&openStackMachineConcurrency,
		"openstackmachine-concurrency",
		1,
		"Number of OpenStackMachines to process simultaneously",
	)

	flag.DurationVar(&syncPeriod,
		"sync-period",
		10*time.Minute,
		"The minimum interval at which watched resources are reconciled (e.g. 15m)",
	)

	flag.Parse()

	if watchNamespace != "" {
		setupLog.Info("Watching cluster-api objects only in namespace for reconciliation", "namespace", watchNamespace)
	}

	if profilerAddress != "" {
		setupLog.Info("Profiler listening for requests", "profiler-address", profilerAddress)
		go func() {
			setupLog.Error(http.ListenAndServe(profilerAddress, nil), "listen and serve error")
		}()
	}

	ctrl.SetLogger(klogr.New())

	cfg, err := config.GetConfigWithContext(os.Getenv("KUBECONTEXT"))
	if err != nil {
		setupLog.Error(err, "unable to get kubeconfig")
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		SyncPeriod:         &syncPeriod,
		Namespace:          watchNamespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Initialize event recorder.
	record.InitFromRecorder(mgr.GetEventRecorderFor("openstack-controller"))

	if err = (&controllers.OpenStackMachineReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("OpenStackMachine"),
		Recorder: mgr.GetEventRecorderFor("openstackmachine-controller"),
	}).SetupWithManager(mgr, controller.Options{MaxConcurrentReconciles: openStackMachineConcurrency}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OpenStackMachine")
		os.Exit(1)
	}
	if err = (&controllers.OpenStackClusterReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("OpenStackCluster"),
		Recorder: mgr.GetEventRecorderFor("openstackcluster-controller"),
	}).SetupWithManager(mgr, controller.Options{MaxConcurrentReconciles: openStackClusterConcurrency}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OpenStackCluster")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
