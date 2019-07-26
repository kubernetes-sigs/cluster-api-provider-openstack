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

package controller

import (
	"k8s.io/klog"

	configclient "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m); err != nil {
			return err
		}
	}

	return nil
}

func getActuatorParams(mgr manager.Manager) openstack.ActuatorParams {
	config := mgr.GetConfig()

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Could not create kubernetes client to talk to the apiserver: %v", err)
	}
	configClient, err := configclient.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create a config client to talk to the apiserver: %v", err)
	}

	return openstack.ActuatorParams{
		Client:        mgr.GetClient(),
		KubeClient:    kubeClient,
		ConfigClient:  configClient,
		Scheme:        mgr.GetScheme(),
		EventRecorder: mgr.GetEventRecorderFor("openstack_controller"),
	}

}
