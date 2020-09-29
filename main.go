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
	_ "net/http/pprof"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	cgrecord "k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha3"
	"sigs.k8s.io/cluster-api-provider-openstack/controllers"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
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
		leaderElectionNamespace     string
		watchNamespace              string
		profilerAddress             string
		openStackClusterConcurrency int
		openStackMachineConcurrency int
		syncPeriod                  time.Duration
		webhookPort                 int
		healthAddr                  string
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
		&leaderElectionNamespace,
		"leader-election-namespace",
		"",
		"Namespace that the controller performs leader election in. If unspecified, the controller will discover which namespace it is running in.",
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
		10,
		"Number of OpenStackMachines to process simultaneously",
	)

	flag.DurationVar(&syncPeriod,
		"sync-period",
		10*time.Minute,
		"The minimum interval at which watched resources are reconciled (e.g. 15m)",
	)

	flag.IntVar(&webhookPort,
		"webhook-port",
		0,
		"Webhook Server port, disabled by default. When enabled, the manager will only work as webhook server, no reconcilers are installed.",
	)

	flag.StringVar(&healthAddr,
		"health-addr",
		":9440",
		"The address the health endpoint binds to.",
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
	// Machine and cluster operations can create enough events to trigger the event recorder spam filter
	// Setting the burst size higher ensures all events will be recorded and submitted to the API
	broadcaster := cgrecord.NewBroadcasterWithCorrelatorOptions(cgrecord.CorrelatorOptions{
		BurstSize: 100,
	})

	cfg, err := config.GetConfigWithContext(os.Getenv("KUBECONTEXT"))
	if err != nil {
		setupLog.Error(err, "unable to get kubeconfig")
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      metricsAddr,
		LeaderElection:          enableLeaderElection,
		LeaderElectionID:        "controller-leader-election-capo",
		LeaderElectionNamespace: leaderElectionNamespace,
		SyncPeriod:              &syncPeriod,
		Namespace:               watchNamespace,
		EventBroadcaster:        broadcaster,
		Port:                    webhookPort,
		HealthProbeBindAddress:  healthAddr,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Initialize event recorder.
	record.InitFromRecorder(mgr.GetEventRecorderFor("openstack-controller"))

	if webhookPort == 0 {
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
	} else {
		if err = (&infrav1.OpenStackMachineTemplate{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "OpenStackMachineTemplate")
			os.Exit(1)
		}
		if err = (&infrav1.OpenStackMachineTemplateList{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "OpenStackMachineTemplateList")
			os.Exit(1)
		}
		if err = (&infrav1.OpenStackCluster{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "OpenStackCluster")
			os.Exit(1)
		}
		if err = (&infrav1.OpenStackMachine{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "OpenStackMachine")
			os.Exit(1)
		}
		if err = (&infrav1.OpenStackMachineList{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "OpenStackMachineList")
			os.Exit(1)
		}
		if err = (&infrav1.OpenStackClusterList{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "OpenStackClusterList")
			os.Exit(1)
		}
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddReadyzCheck("ping", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to create ready check")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to create health check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
