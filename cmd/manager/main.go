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
	"log"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog"

	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/apis"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/controller"
	clusterapis "sigs.k8s.io/cluster-api/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

var RootCmd = &cobra.Command{
	Use:   "manager",
	Short: "Cluster API provider OpenStack manager",
	Long:  `Manager for the openstack cluster manager provider`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Glog requires this otherwise it complains.
		flag.CommandLine.Parse(nil)
		pflag.Parse()

		// This is a temporary hack to enable proper logging until upstream dependencies
		// are migrated to fully utilize klog instead of glog.
		klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
		klog.InitFlags(klogFlags)

		// Sync the glog and klog flags.
		cmd.Flags().VisitAll(func(f1 *pflag.Flag) {
			f2 := klogFlags.Lookup(f1.Name)
			if f2 != nil {
				value := f1.Value.String()
				f2.Value.Set(value)
			}
		})
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
		cmd.Help()
	},
}

var logFlushFreq = pflag.Duration("log-flush-frequency", 5*time.Second, "Maximum number of seconds between log flushes")

// KlogWriter serves as a bridge between the standard log package and the klog package.
type KlogWriter struct{}

// Write implements the io.Writer interface.
func (writer KlogWriter) Write(data []byte) (n int, err error) {
	klog.Info(string(data))
	return len(data), nil
}

// InitLogs initializes logs the way we want for kubernetes.
func InitLogs() {
	flag.Set("logtostderr", "true")
	log.SetOutput(KlogWriter{})
	log.SetFlags(0)
	// The default klog flush interval is 30 seconds, which is frighteningly long.
	go wait.Until(klog.Flush, *logFlushFreq, wait.NeverStop)
}

func main() {
	flag.Parse()
	RootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	InitLogs()

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		glog.Fatal(err)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		glog.Fatal(err)
	}

	glog.Infof("Initializing Dependencies.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatal(err)
	}

	if err := clusterapis.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatal(err)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		glog.Fatal(err)
	}

	log.Printf("Starting the Cmd.")

	// Start the Cmd
	log.Fatal(mgr.Start(signals.SetupSignalHandler()))
}
