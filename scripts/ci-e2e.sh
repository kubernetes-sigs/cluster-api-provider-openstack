#!/bin/bash

# Copyright 2021 The Kubernetes Authors.
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
# usage: ci-e2e.sh
#  This program runs the e2e tests.
################################################################################

set -x
set -o nounset
set -o pipefail

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${REPO_ROOT}" || exit 1

# shellcheck source=../hack/ensure-go.sh
source "${REPO_ROOT}/hack/ensure-go.sh"
# shellcheck source=../hack/ensure-kubectl.sh
source "${REPO_ROOT}/hack/ensure-kubectl.sh"

export RESOURCE_TYPE="${RESOURCE_TYPE:-"gce-project"}"

ARTIFACTS="${ARTIFACTS:-${PWD}/_artifacts}"
mkdir -p "${ARTIFACTS}/logs/"

# our exit handler (trap)
cleanup() {
  # stop boskos heartbeat
  [[ -z ${HEART_BEAT_PID:-} ]] || kill -9 "${HEART_BEAT_PID}"

  # will be started by the devstack installation script
  pkill sshuttle
}
trap cleanup EXIT

apt-get update -y
# Install requests module explicitly for HTTP calls.
# libffi required for pip install cffi (caracal dependency)
apt-get install -y python3-requests libffi-dev
rm -rf /var/lib/apt/lists/*

# If BOSKOS_HOST is set then acquire a resource of type ${RESOURCE_TYPE} from Boskos.
if [ -n "${BOSKOS_HOST:-}" ]; then
  # Check out the account from Boskos and store the produced environment
  # variables in a temporary file.
  account_env_var_file="$(mktemp)"
  python3 hack/boskos.py --get --resource-type="${RESOURCE_TYPE}" 1>"${account_env_var_file}"
  checkout_account_status="${?}"

  # If the checkout process was a success then load the account's
  # environment variables into this process.
  # shellcheck disable=SC1090
  [ "${checkout_account_status}" = "0" ] && . "${account_env_var_file}"

  # Always remove the account environment variable file. It contains
  # sensitive information.
  rm -f "${account_env_var_file}"

  if [ ! "${checkout_account_status}" = "0" ]; then
    echo "error getting account from boskos" 1>&2
    exit "${checkout_account_status}"
  fi

  # run the heart beat process to tell boskos that we are still
  # using the checked out account periodically
  python3 -u hack/boskos.py --heartbeat >> "$ARTIFACTS/logs/boskos.log" 2>&1 &
  HEART_BEAT_PID=$!
fi

# Run e2e prerequisites concurrently with devstack build to save time
prerequisites_log="${ARTIFACTS}/logs/e2e-prerequisites.log"
(
    # Run prerequisites at low priority to avoid slowing down devstack tasks,
    # which generally take much longer
    ionice nice make e2e-image e2e-prerequisites build-e2e-tests

    container_archive=$(mktemp --suffix=.tar)
    ionice nice docker save -o "${container_archive}" gcr.io/k8s-staging-capi-openstack/capi-openstack-controller:e2e

    # Wait for SSH to become available in the provisioning devstack
    # infrastructure before uploading image archive
    CONTROLLER_IP=${CONTROLLER_IP:-"10.0.3.15"}
    source "${REPO_ROOT}/hack/ci/${RESOURCE_TYPE}.sh"
    source "${REPO_ROOT}/hack/ci/common.sh"
    wait_for_ssh "${CONTROLLER_IP}"

    retry 10 10 scp $(get_ssh_common_args) "${container_archive}" "cloud@${CONTROLLER_IP}:capo-e2e-image.tar"
    retry 10 10 $(get_ssh_cmd) ${CONTROLLER_IP} -- sudo chown root:root capo-e2e-image.tar
    retry 10 10 $(get_ssh_cmd) ${CONTROLLER_IP} -- sudo chmod u=rw,g=r,o=r capo-e2e-image.tar
    retry 10 10 $(get_ssh_cmd) ${CONTROLLER_IP} -- sudo mkdir -p /var/www/html
    retry 10 10 $(get_ssh_cmd) ${CONTROLLER_IP} -- sudo mv capo-e2e-image.tar /var/www/html/capo-e2e-image.tar
) >"$prerequisites_log" 2>&1 &
build_pid=$!

# Build the devstack environment
hack/ci/create_devstack.sh

# Check exit of prerequisites build, waiting for it to complete if necessary
if ! wait $build_pid; then
    echo "Building e2e prerequisites failed"
    cat "$prerequisites_log"
    exit 1
fi

make test-e2e OPENSTACK_CLOUD_YAML_FILE="$(pwd)/clouds.yaml"
test_status="${?}"

# If Boskos is being used then release the resource back to Boskos.
[ -z "${BOSKOS_HOST:-}" ] || python3 hack/boskos.py --release >> "$ARTIFACTS/logs/boskos.log" 2>&1

exit "${test_status}"
