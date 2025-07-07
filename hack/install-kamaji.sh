#!/bin/bash

# Copyright 2025 The Kubernetes Authors.
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

set -euo pipefail

KUBECONFIG_ARG=""
if [ -n "${1-}" ]; then
    echo "Using kubeconfig: ${1}"
    KUBECONFIG_ARG="--kubeconfig ${1}"
fi

# Simple Kamaji installation using Helm
# This script installs Kamaji v1.0.0 in the management cluster

KAMAJI_VERSION="v1.0.0"
KAMAJI_NAMESPACE="kamaji-system"

echo "Installing Kamaji ${KAMAJI_VERSION}..."

# Add Clastix Helm repository
echo "Adding Clastix Helm repository..."
helm repo add clastix https://clastix.github.io/charts
helm repo update

# Install Kamaji
echo "Installing Kamaji in namespace ${KAMAJI_NAMESPACE}..."
helm install kamaji clastix/kamaji \
  --version ${KAMAJI_VERSION} \
  --namespace ${KAMAJI_NAMESPACE} \
  --create-namespace \
  ${KUBECONFIG_ARG} \
  --wait

# Verify installation
echo "Verifying Kamaji installation..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=kamaji -n ${KAMAJI_NAMESPACE} --timeout=300s ${KUBECONFIG_ARG}

# Create default datastore
echo "Creating default datastore..."
cat <<EOF | kubectl apply -f - ${KUBECONFIG_ARG}
apiVersion: kamaji.clastix.io/v1alpha1
kind: DataStore
metadata:
  name: default
  namespace: ${KAMAJI_NAMESPACE}
spec:
  driver: etcd
  endpoints:
  - kamaji-etcd.${KAMAJI_NAMESPACE}.svc.cluster.local:2379
EOF

echo "Kamaji installation completed successfully!"
echo "Ready to create TenantControlPlanes with the 'default' datastore." 