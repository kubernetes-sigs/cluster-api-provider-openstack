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
	"errors"
	"text/template"

	"fmt"

<<<<<<< HEAD
	machinev1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	openstackconfigv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
=======
	openstackconfigv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
>>>>>>> 564ebf706608601f4c653a5e2c1092332feb661e
)

type setupParams struct {
	Token       string
<<<<<<< HEAD
	Cluster     *machinev1.Cluster
	Machine     *machinev1.Machine
=======
	Cluster     *clusterv1.Cluster
	Machine     *clusterv1.Machine
>>>>>>> 564ebf706608601f4c653a5e2c1092332feb661e
	MachineSpec *openstackconfigv1.OpenstackProviderSpec

	PodCIDR           string
	ServiceCIDR       string
	GetMasterEndpoint func() (string, error)
}

func init() {
}

<<<<<<< HEAD
func masterStartupScript(cluster *machinev1.Cluster, machine *machinev1.Machine, script string) (string, error) {
=======
func masterStartupScript(cluster *clusterv1.Cluster, machine *clusterv1.Machine, script string) (string, error) {
>>>>>>> 564ebf706608601f4c653a5e2c1092332feb661e
	machineSpec, err := openstackconfigv1.MachineSpecFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return "", err
	}

	params := setupParams{
		Cluster:     cluster,
		Machine:     machine,
		MachineSpec: machineSpec,
<<<<<<< HEAD
	}

	if cluster != nil {
		params.PodCIDR = getSubnet(cluster.Spec.ClusterNetwork.Pods)
		params.ServiceCIDR = getSubnet(cluster.Spec.ClusterNetwork.Services)
=======
		PodCIDR:     getSubnet(cluster.Spec.ClusterNetwork.Pods),
		ServiceCIDR: getSubnet(cluster.Spec.ClusterNetwork.Services),
>>>>>>> 564ebf706608601f4c653a5e2c1092332feb661e
	}

	masterStartUpScript := template.Must(template.New("masterStartUp").Parse(script))

	var buf bytes.Buffer
	if err := masterStartUpScript.Execute(&buf, params); err != nil {
		return "", err
	}
	return buf.String(), nil
}

<<<<<<< HEAD
func nodeStartupScript(cluster *machinev1.Cluster, machine *machinev1.Machine, token, script string) (string, error) {
=======
func nodeStartupScript(cluster *clusterv1.Cluster, machine *clusterv1.Machine, token, script string) (string, error) {
>>>>>>> 564ebf706608601f4c653a5e2c1092332feb661e
	machineSpec, err := openstackconfigv1.MachineSpecFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return "", err
	}

	GetMasterEndpoint := func() (string, error) {
<<<<<<< HEAD
		if cluster == nil {
			return "", nil
		} else if len(cluster.Status.APIEndpoints) == 0 {
=======
		if len(cluster.Status.APIEndpoints) == 0 {
>>>>>>> 564ebf706608601f4c653a5e2c1092332feb661e
			return "", errors.New("no cluster status found")
		}
		return getEndpoint(cluster.Status.APIEndpoints[0]), nil
	}

	params := setupParams{
		Token:             token,
		Cluster:           cluster,
		Machine:           machine,
		MachineSpec:       machineSpec,
<<<<<<< HEAD
		GetMasterEndpoint: GetMasterEndpoint,
	}

	if cluster != nil {
		params.PodCIDR = getSubnet(cluster.Spec.ClusterNetwork.Pods)
		params.ServiceCIDR = getSubnet(cluster.Spec.ClusterNetwork.Services)
	}

=======
		PodCIDR:           getSubnet(cluster.Spec.ClusterNetwork.Pods),
		ServiceCIDR:       getSubnet(cluster.Spec.ClusterNetwork.Services),
		GetMasterEndpoint: GetMasterEndpoint,
	}

>>>>>>> 564ebf706608601f4c653a5e2c1092332feb661e
	nodeStartUpScript := template.Must(template.New("nodeStartUp").Parse(script))

	var buf bytes.Buffer
	if err := nodeStartUpScript.Execute(&buf, params); err != nil {
		return "", err
	}
	return buf.String(), nil
}

<<<<<<< HEAD
func getEndpoint(apiEndpoint machinev1.APIEndpoint) string {
=======
func getEndpoint(apiEndpoint clusterv1.APIEndpoint) string {
>>>>>>> 564ebf706608601f4c653a5e2c1092332feb661e
	return fmt.Sprintf("%s:%d", apiEndpoint.Host, apiEndpoint.Port)
}

// Just a temporary hack to grab a single range from the config.
<<<<<<< HEAD
func getSubnet(netRange machinev1.NetworkRanges) string {
=======
func getSubnet(netRange clusterv1.NetworkRanges) string {
>>>>>>> 564ebf706608601f4c653a5e2c1092332feb661e
	if len(netRange.CIDRBlocks) == 0 {
		return ""
	}
	return netRange.CIDRBlocks[0]
}
