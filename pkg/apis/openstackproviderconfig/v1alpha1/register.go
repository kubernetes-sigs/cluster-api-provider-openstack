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

package v1alpha1

import (
	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"

	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

const GroupName = "openstackproviderconfig"

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "openstackproviderconfig.k8s.io", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)

func ClusterConfigFromProviderConfig(providerConfig clusterv1.ProviderConfig) (*OpenstackProviderConfig, error) {
	var config *OpenstackProviderConfig
	if err := yaml.Unmarshal(providerConfig.Value.Raw, &config); err != nil {
		return nil, err
	}
	return config, nil
}

func ClusterStatusFromProviderStatus(extension *runtime.RawExtension) (*OpenstackClusterProviderStatus, error) {
	if extension == nil {
		return &OpenstackClusterProviderStatus{}, nil
	}

	status := new(OpenstackClusterProviderStatus)
	if err := yaml.Unmarshal(extension.Raw, status); err != nil {
		return nil, err
	}

	return status, nil
}

// This is the same as ClusterConfigFromProviderConfig but we
// expect there to be a specific Config type for Machines soon
func MachineConfigFromProviderConfig(providerConfig clusterv1.ProviderConfig) (*OpenstackProviderConfig, error) {
	var config OpenstackProviderConfig
	if err := yaml.Unmarshal(providerConfig.Value.Raw, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func EncodeClusterStatus(status *OpenstackClusterProviderStatus) (*runtime.RawExtension, error) {
	if status == nil {
		return &runtime.RawExtension{}, nil
	}

	var rawBytes []byte
	var err error

	//  TODO: use apimachinery conversion https://godoc.org/k8s.io/apimachinery/pkg/runtime#Convert_runtime_Object_To_runtime_RawExtension
	if rawBytes, err = json.Marshal(status); err != nil {
		return nil, err
	}

	return &runtime.RawExtension{
		Raw: rawBytes,
	}, nil
}
