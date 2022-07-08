#!/bin/bash
# Copyright 2022 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

CPO_VERSION=1.24.2
CNI=calico


if [ "$#" -le 1 ]; then
    echo "One param for cluster-name needed"
    echo "The usage: external_cloud_setup.sh <cluster-name>"
    exit 1
fi

if ! command -v clusterctl &> /dev/null
then
    echo "clusterctl could not be found, make sure you download it and put into right place"
    exit 1
fi

if ! command -v ./create_cloud_conf.sh &> /dev/null
then
    echo "./create_cloud_conf.sh could not be found, make sure you run this script inside desired folder"
    exit 1
fi



CLUSTER_NAME=$1


echo "##########################################"
echo "Create kubeconfig file ${CLUSTER_NAME}.kubeconfig for cluster ${CLUSTER_NAME}"
clusterctl get kubeconfig ${CLUSTER_NAME} --namespace default > ./${CLUSTER_NAME}.kubeconfig

echo "##########################################"
echo "Create secret cloud-config"
tmpfile=`mktemp`
./create_cloud_conf.sh clouds.yaml openstack > $tmpfile
kubectl --kubeconfig=./${CLUSTER_NAME}.kubeconfig create secret -n kube-system generic cloud-config --from-file=$tmpfile
rm $tmpfile

echo "##########################################"
echo "Apply OCCM manifests"
kubectl --kubeconfig=./${CLUSTER_NAME}.kubeconfig apply -f https://raw.githubusercontent.com/kubernetes/cloud-provider-openstack/$CPO_VERSION/manifests/controller-manager/cloud-controller-manager-roles.yaml
kubectl --kubeconfig=./${CLUSTER_NAME}.kubeconfig apply -f https://raw.githubusercontent.com/kubernetes/cloud-provider-openstack/$CPO_VERSION/manifests/controller-manager/cloud-controller-manager-role-bindings.yaml
kubectl --kubeconfig=./${CLUSTER_NAME}.kubeconfig apply -f https://raw.githubusercontent.com/kubernetes/cloud-provider-openstack/$CPO_VERSION/manifests/controller-manager/openstack-cloud-controller-manager-ds.yaml

echo "##########################################"

# we need install CNI in order to make whole cluster working
echo "Install CNI, currently it's calico"
kubectl --kubeconfig=./${CLUSTER_NAME}.kubeconfig apply -f https://docs.projectcalico.org/v3.23/manifests/calico.yaml
