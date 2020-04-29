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

package machine

import (
	"bytes"
	"text/template"

	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	openstackconfigv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
)

type setupParams struct {
	Token       string
	Machine     *machinev1.Machine
	MachineSpec *openstackconfigv1.OpenstackProviderSpec

	PodCIDR           string
	ServiceCIDR       string
	GetMasterEndpoint func() (string, error)
}

func init() {
}

func masterStartupScript(machine *machinev1.Machine, script string) (string, error) {
	machineSpec, err := openstackconfigv1.MachineSpecFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return "", err
	}

	params := setupParams{
		Machine:     machine,
		MachineSpec: machineSpec,
	}

	masterStartUpScript := template.Must(template.New("masterStartUp").Parse(script))

	var buf bytes.Buffer
	if err := masterStartUpScript.Execute(&buf, params); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func nodeStartupScript(machine *machinev1.Machine, token, script string) (string, error) {
	machineSpec, err := openstackconfigv1.MachineSpecFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return "", err
	}

	params := setupParams{
		Token:       token,
		Machine:     machine,
		MachineSpec: machineSpec,
	}

	nodeStartUpScript := template.Must(template.New("nodeStartUp").Parse(script))

	var buf bytes.Buffer
	if err := nodeStartUpScript.Execute(&buf, params); err != nil {
		return "", err
	}
	return buf.String(), nil
}
