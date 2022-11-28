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
# usage: ci-conformance.sh
#  This program runs the clusterctl conformance e2e tests.
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

# Ensure that python3-pip is installed.
apt-get update -y
apt-get install -y python3-pip
rm -rf /var/lib/apt/lists/*

# Install/upgrade pip and requests module explicitly for HTTP calls.
python3 -m pip install --upgrade pip requests

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

"hack/ci/create_devstack.sh"

export OPENSTACK_CLOUD_YAML_FILE
OPENSTACK_CLOUD_YAML_FILE="$(pwd)/clouds.yaml"
make test-e2e
test_status="${?}"

# If Boskos is being used then release the resource back to Boskos.
[ -z "${BOSKOS_HOST:-}" ] || python3 hack/boskos.py --release >> "$ARTIFACTS/logs/boskos.log" 2>&1

exit "${test_status}"
