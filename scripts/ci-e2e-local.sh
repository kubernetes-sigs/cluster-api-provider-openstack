#!/bin/bash

# Copyright 2023 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# 	http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

################################################################################
# usage: ci-e2e-local.sh
#  This program runs the e2e tests locally and also starts
#  the create_devstack.sh script which is optimized for Debian based Distros
################################################################################

set -o nounset
set -o pipefail

# Check prereq binaries.
allBinariesInstalled=true
for binary in ipcalc yq genisoimage; do
	if ! command -v $binary > /dev/null; then
		echo "Binary $binary is not installed, please install it first."
		allBinariesInstalled=false
	fi
done
$allBinariesInstalled || exit 1

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${REPO_ROOT}" || exit 1

# shellcheck source=../hack/ensure-go.sh
source "${REPO_ROOT}/hack/ensure-go.sh"
# shellcheck source=../hack/ensure-kubectl.sh
source "${REPO_ROOT}/hack/ensure-kubectl.sh"

export RESOURCE_TYPE="${RESOURCE_TYPE:-"local-kvm"}"

CLUSTER_NAME=${CLUSTER_NAME:-"capo-e2e"}
ARTIFACTS=${ARTIFACTS:-/tmp/${CLUSTER_NAME}-artifacts}
devstackdir="${ARTIFACTS}/devstack"

CLEANUP_DEVSTACK=${CLEANUP_DEVSTACK:-""}

# These tests can use more than 2 machines per cluster and even 2 clusters
# per test for clusterctl upgrade tests, so we limit the parallel jobs
# to avoid capacity issues.
export E2E_GINKGO_PARALLEL=1

hack/ci/create_devstack.sh "${CLEANUP_DEVSTACK}"

# Upload image for e2e clusterctl upgrade tests
source "${REPO_ROOT}/hack/ci/${RESOURCE_TYPE}.sh"
CONTAINER_ARCHIVE="${ARTIFACTS}/capo-e2e-image.tar"
SSH_KEY="$(get_ssh_private_key_file)"
SSH_ARGS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o IdentitiesOnly=yes -o PasswordAuthentication=no"
CONTROLLER_IP=${CONTROLLER_IP:-"10.0.3.15"}

make e2e-image

# Restart OpenStack instances that are in state SHUTOFF. Due too the fact that our e2e tests needs alot of resources it's possible
# that the instances may fail due too OOM kills.
# This really dirty workaround kind of fixes this issue -- at least the e2e tests will succeed.
source "${REPO_ROOT}/templates/env.rc" "${REPO_ROOT}/clouds.yaml" "capo-e2e-admin"
export OS_CLOUD="capo-e2e-admin"
while :; do
	openstack server list --all -fjson | jq -r '.[] | select(.Status == "SHUTOFF") | .ID' | while read id; do
		echo openstack server start $id
		openstack server start $id
	done
	sleep 1
done &

export OPENSTACK_CLOUD_YAML_FILE
OPENSTACK_CLOUD_YAML_FILE="$(pwd)/clouds.yaml"
make test-e2e
test_status="${?}"

exit "${test_status}"
