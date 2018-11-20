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

	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type setupParams struct {
	Token   string
	Cluster *clusterv1.Cluster
	Machine *clusterv1.Machine

	PodCIDR        string
	ServiceCIDR    string
	MasterEndpoint string
}

func init() {
}

func masterStartupScript(cluster *clusterv1.Cluster, machine *clusterv1.Machine, script string) (string, error) {
	params := setupParams{
		Cluster:     cluster,
		Machine:     machine,
		PodCIDR:     getSubnet(cluster.Spec.ClusterNetwork.Pods),
		ServiceCIDR: getSubnet(cluster.Spec.ClusterNetwork.Services),
	}

	masterStartUpScript := template.Must(template.New("masterStartUp").Parse(script))

	var buf bytes.Buffer
	if err := masterStartUpScript.Execute(&buf, params); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func nodeStartupScript(cluster *clusterv1.Cluster, machine *clusterv1.Machine, token, script string) (string, error) {
	if len(cluster.Status.APIEndpoints) == 0 {
		return "", errors.New("no cluster status found")
	}
	params := setupParams{
		Token:          token,
		Cluster:        cluster,
		Machine:        machine,
		PodCIDR:        getSubnet(cluster.Spec.ClusterNetwork.Pods),
		ServiceCIDR:    getSubnet(cluster.Spec.ClusterNetwork.Services),
		MasterEndpoint: getEndpoint(cluster.Status.APIEndpoints[0]),
	}

	nodeStartUpScript := template.Must(template.New("nodeStartUp").Parse(script))

	var buf bytes.Buffer
	if err := nodeStartUpScript.Execute(&buf, params); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func getEndpoint(apiEndpoint clusterv1.APIEndpoint) string {
	return fmt.Sprintf("%s:%d", apiEndpoint.Host, apiEndpoint.Port)
}

// Just a temporary hack to grab a single range from the config.
func getSubnet(netRange clusterv1.NetworkRanges) string {
	if len(netRange.CIDRBlocks) == 0 {
		return ""
	}
	return netRange.CIDRBlocks[0]
}
