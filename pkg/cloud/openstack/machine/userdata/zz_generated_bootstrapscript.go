package userdata

/*
This file is auto-generated DO NOT TOUCH!
*/

const (
	bootstrapScript = `#!/bin/bash
# Copyright 2019 by the contributors
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.
set -o verbose
set -o errexit
set -o nounset
set -o pipefail

if [[ -f /etc/kubernetes/kubelet.conf ]]; then
    echo "kubernetes already bootstrapped."
    exit 0
fi
: ${MACHINE_NAME:=$(hostname -s)}
MACHINE="$NAMESPACE/$MACHINE_NAME"
if grep "InitConfiguration" /etc/kubernetes/kubeadm_config.yaml; then
kubeadm init --config /etc/kubernetes/kubeadm_config.yaml

# By default, use calico for container network plugin, should make this configurable.
for tries in $(seq 1 60); do
    kubectl --kubeconfig /etc/kubernetes/admin.conf apply -f https://docs.projectcalico.org/v3.1/getting-started/kubernetes/installation/hosted/rbac-kdd.yaml && break
    sleep 1
done
for tries in $(seq 1 60); do
    kubectl --kubeconfig /etc/kubernetes/admin.conf apply -f https://docs.projectcalico.org/v3.1/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/1.7/calico.yaml && break
    sleep 1
done

else
kubeadm join --ignore-preflight-errors=all --config /etc/kubernetes/kubeadm_config.yaml
fi

for tries in $(seq 1 60); do
    kubectl --kubeconfig /etc/kubernetes/kubelet.conf annotate --overwrite node $(hostname -s) machine=${MACHINE} && break
    sleep 1
done

echo done.
`
)
