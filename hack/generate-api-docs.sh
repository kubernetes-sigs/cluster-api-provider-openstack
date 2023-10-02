#!/usr/bin/env bash

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

set -o errexit
set -o nounset
set -o pipefail

__dir__=$(realpath $(dirname $0))
source "${__dir__}/ensure-go.sh"
verify_go_version

go install github.com/ahmetb/gen-crd-api-reference-docs@latest

__root__=$(realpath "${__dir__}/..")
API_VERSIONS=($(ls "${__root__}/api"))

BOOK_DIR="${__root__}/docs/book"

SUMMARY_FILE="${BOOK_DIR}/src/SUMMARY.md"
API_DOC_DIR="${BOOK_DIR}/src/api"

ORIGINAL_SUMMARY_FILE="${BOOK_DIR}/src/SUMMARY_ORIGINAL.md"

cp "${SUMMARY_FILE}" "${ORIGINAL_SUMMARY_FILE}"

mkdir -p "${API_DOC_DIR}"

cd "${__root__}"
echo "- [Reference](./api/index.md)" >> "${SUMMARY_FILE}"
touch "${API_DOC_DIR}/index.md"

for api_version in "${API_VERSIONS[@]}"; do
  gen-crd-api-reference-docs --api-dir="./api/${api_version}" --config=./hack/api/config.json --template-dir=./hack/api/template -out-file="${API_DOC_DIR}/${api_version}.md"
  echo "    - [${api_version}](./api/${api_version}.md)" >> "${SUMMARY_FILE}"
done

# rm -f "${SUMMARY_FILE}"
# mv "${ORIGINAL_SUMMARY_FILE}" "${SUMMARY_FILE}"
