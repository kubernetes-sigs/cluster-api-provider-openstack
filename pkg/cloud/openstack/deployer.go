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

package openstack

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"

	"k8s.io/klog"
	openstackconfigv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
)

const ProviderName = "openstack"
const (
	OpenstackIPAnnotationKey = "openstack-ip-address"
	OpenstackIdAnnotationKey = "openstack-resourceId"
)

type DeploymentClient struct{}

func NewDeploymentClient() *DeploymentClient {
	return &DeploymentClient{}
}

func (*DeploymentClient) GetIP(machine *machinev1.Machine) (string, error) {
	if machine.ObjectMeta.Annotations != nil {
		if ip, ok := machine.ObjectMeta.Annotations[OpenstackIPAnnotationKey]; ok {
			klog.Infof("Returning IP from machine annotation %s", ip)
			return ip, nil
		}
	}

	return "", errors.New("could not get IP")
}

// execCommand executes a local command in the current shell.
func execCommand(name string, args ...string) string {
	cmdOut, err := exec.Command(name, args...).Output()
	if err != nil {
		s := strings.Join(append([]string{name}, args...), " ")
		klog.Errorf("error executing command %q: %v", s, err)
	}
	return string(cmdOut)
}

func (d *DeploymentClient) GetKubeConfig(master *machinev1.Machine) (string, error) {
	ip, err := d.GetIP(master)
	if err != nil {
		return "", err
	}

	homeDir, ok := os.LookupEnv("HOME")
	if !ok {
		return "", fmt.Errorf("unable to use HOME environment variable to find SSH key: %v", err)
	}

	machineSpec, err := openstackconfigv1.MachineSpecFromProviderSpec(master.Spec.ProviderSpec)
	if err != nil {
		return "", err
	}

	result := strings.TrimSpace(execCommand(
		"ssh", "-i", homeDir+"/.ssh/openstack_tmp",
		"-o", "StrictHostKeyChecking no",
		"-o", "UserKnownHostsFile /dev/null",
		"-o", "BatchMode=yes",
		fmt.Sprintf("%s@%s", machineSpec.SshUserName, ip),
		"echo STARTFILE; sudo cat /etc/kubernetes/admin.conf"))
	parts := strings.Split(result, "STARTFILE")
	if len(parts) != 2 {
		return "", nil
	}
	return strings.TrimSpace(parts[1]), nil
}
