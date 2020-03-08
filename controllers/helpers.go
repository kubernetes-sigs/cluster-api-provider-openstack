/*
Copyright 2020 The Kubernetes Authors.

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

package controllers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	clusterutil "sigs.k8s.io/cluster-api/util"
)

// TODO: Fix up Cluster API's clusterutil.IsPaused function
func isPaused(cluster *clusterv1.Cluster, v metav1.Object) bool {
	if cluster == nil {
		cluster = &clusterv1.Cluster{
			Spec: clusterv1.ClusterSpec{},
		}
	}
	return clusterutil.IsPaused(cluster, v)
}
