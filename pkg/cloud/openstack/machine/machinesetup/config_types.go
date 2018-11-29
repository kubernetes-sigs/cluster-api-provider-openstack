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

package machinesetup

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/yaml"
)

const (
	MachineRoleMaster = "master"
	MachineRoleNode   = "node"
)

type MachineSetupConfig interface {
	GetYaml() (string, error)
	GetSetupScript(role string) (string, error)
}

// Config Watch holds the path to the machine setup configs yaml file.
// This works directly with a yaml file is used instead of a ConfigMap object so that we don't take a dependency on the API Server.
type ConfigWatch struct {
	path string
}

// The valid machine setup configs parsed out of the machine setup configs yaml file held in ConfigWatch.
type ValidConfigs struct {
	configList *configList
}

type configList struct {
	Items []config `json:"items"`
}

// A single valid machine setup config that maps a machine's version to the corresponding startupScript.
// TODO need more elements for setup openstack instance
type config struct {
	// A list of the valid combinations of ConfigParams that will
	// map to the given Image and Metadata.
	Versions clusterv1.MachineVersionInfo

	StartupScript string `json:"startupScript"`
}

func NewConfigWatch(path string) (*ConfigWatch, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	return &ConfigWatch{path: path}, nil
}

func (cw *ConfigWatch) GetMachineSetupConfig() (MachineSetupConfig, error) {
	file, err := os.Open(cw.path)
	if err != nil {
		return nil, err
	}
	return parseMachineSetupYaml(file)
}

func parseMachineSetupYaml(reader io.Reader) (*ValidConfigs, error) {
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	configList := &configList{}
	err = yaml.Unmarshal(bytes, configList)
	if err != nil {
		return nil, err
	}

	return &ValidConfigs{configList}, nil
}

func (vc *ValidConfigs) GetYaml() (string, error) {
	bytes, err := yaml.Marshal(vc.configList)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (vc *ValidConfigs) GetSetupScript(role string) (string, error) {
	machineSetupConfig, err := vc.matchMachineSetupConfig(role)
	if err != nil {
		return "", err
	}
	return machineSetupConfig.StartupScript, nil
}

func (vc *ValidConfigs) matchMachineSetupConfig(role string) (*config, error) {
	matchingConfigs := make([]config, 0)
	for _, conf := range vc.configList.Items {
		if role == MachineRoleMaster && conf.Versions.ControlPlane != "" {
			matchingConfigs = append(matchingConfigs, conf)
		}
		if role == MachineRoleNode && conf.Versions.ControlPlane == "" {
			matchingConfigs = append(matchingConfigs, conf)
		}
	}

	if len(matchingConfigs) == 1 {
		return &matchingConfigs[0], nil
	} else if len(matchingConfigs) == 0 {
		return nil, fmt.Errorf("could not find a matching machine setup config for role %q", role)
	} else {
		return nil, fmt.Errorf("found multiple matching machine setup configs for role %q", role)
	}
}
