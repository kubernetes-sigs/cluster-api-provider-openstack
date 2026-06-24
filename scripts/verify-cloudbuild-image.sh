#!/usr/bin/env bash

# Copyright 2026 The Kubernetes Authors.
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

# Verifies the gcb-docker-gcloud builder image referenced in cloudbuild.yaml:
#   1. Exists in the registry (can be pulled)
#   2. Can build the CAPO container image (make docker-build succeeds)
#
# This catches the case where a pruned builder image would only be discovered
# at release time. See https://github.com/kubernetes/k8s.io/issues/9599.

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${REPO_ROOT}"

CLOUDBUILD_FILE="cloudbuild.yaml"

# Extract the pinned gcb-docker-gcloud image digest from cloudbuild.yaml.
IMAGE=$(grep -oE "gcr\.io/k8s-staging-test-infra/gcb-docker-gcloud@sha256:[a-f0-9]+" "${CLOUDBUILD_FILE}" | head -1)

if [[ -z "${IMAGE}" ]]; then
    echo "ERROR: Could not extract gcb-docker-gcloud image from ${CLOUDBUILD_FILE}" >&2
    exit 1
fi

echo "==> Verifying cloudbuild image exists: ${IMAGE}"
docker pull "${IMAGE}"
echo "==> Image pull succeeded."

echo "==> Verifying cloudbuild image can build the CAPO container image..."
# Mount the host Docker socket so that 'make docker-build' inside the
# gcb-docker-gcloud container can reach the daemon. By this point docker pull
# has already succeeded, so the socket is guaranteed to exist.
docker run --rm \
    -v "$(pwd):/workspace" \
    -w /workspace \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -e DOCKER_CLI_EXPERIMENTAL=enabled \
    -e DOCKER_BUILDKIT=1 \
    -e TAG=ci-verify \
    -e PULL_BASE_REF="${PULL_BASE_REF:-main}" \
    "${IMAGE}" \
    make docker-build

echo "==> SUCCESS: cloudbuild image ${IMAGE} exists and can build the CAPO container image."
