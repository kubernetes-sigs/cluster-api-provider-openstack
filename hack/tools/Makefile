# Copyright 2021 The Kubernetes Authors.
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

ROOT_DIR_RELATIVE := ../..
include $(ROOT_DIR_RELATIVE)/common.mk

UNAME := $(shell uname -s)

# Directories.
BIN_DIR := bin

OS := $(shell go env GOOS)

## --------------------------------------
## Tooling Binaries
## --------------------------------------

$(BIN_DIR):
	mkdir -p $@

CONTROLLER_GEN := $(BIN_DIR)/controller-gen
$(CONTROLLER_GEN): $(BIN_DIR) go.mod go.sum # Build controller-gen from tools folder.
	go build -tags=tools -o $@ sigs.k8s.io/controller-tools/cmd/controller-gen

CONVERSION_GEN := $(BIN_DIR)/conversion-gen
$(CONVERSION_GEN): $(BIN_DIR) go.mod go.sum
	go build -tags=tools -o $@ k8s.io/code-generator/cmd/conversion-gen

DEFAULTER_GEN := $(BIN_DIR)/defaulter-gen
$(DEFAULTER_GEN): $(BIN_DIR) go.mod go.sum
	go build -tags=tools -o $@ k8s.io/code-generator/cmd/defaulter-gen

ENVSUBST := $(BIN_DIR)/envsubst
$(ENVSUBST): $(BIN_DIR) go.mod go.sum # Build envsubst from tools folder.
	go build -tags=tools -o $@ github.com/a8m/envsubst/cmd/envsubst

GINKGO := $(BIN_DIR)/ginkgo
$(GINKGO): $(BIN_DIR) go.mod go.sum
	go build -tags=tools -o $@ github.com/onsi/ginkgo/ginkgo

GOJQ := $(BIN_DIR)/gojq
$(GOJQ): $(BIN_DIR) go.mod go.sum
	go build -tags=tools -o $@ github.com/itchyny/gojq/cmd/gojq

GOLANGCI_LINT := $(BIN_DIR)/golangci-lint
$(GOLANGCI_LINT): $(BIN_DIR) go.mod go.sum # Build golangci-lint from tools folder.
	go build -tags=tools -o $@ github.com/golangci/golangci-lint/cmd/golangci-lint

KIND := $(BIN_DIR)/kind
$(KIND): $(BIN_DIR) go.mod go.sum
	go build -tags tools -o $@ sigs.k8s.io/kind

KUSTOMIZE := $(BIN_DIR)/kustomize
$(KUSTOMIZE): $(BIN_DIR) go.mod go.sum # Build kustomize from tools folder.
	CGO_ENABLED=0 go build -tags=tools -o $@ sigs.k8s.io/kustomize/kustomize/v3

MOCKGEN := $(BIN_DIR)/mockgen
$(MOCKGEN): $(BIN_DIR) go.mod go.sum # Build mockgen from tools folder.
	go build -tags=tools -o $@ github.com/golang/mock/mockgen

RELEASE_NOTES := $(BIN_DIR)/release-notes
$(RELEASE_NOTES): $(BIN_DIR) go.mod go.sum
	go build -tags tools -o $@ sigs.k8s.io/cluster-api/hack/tools/release

.PHONY: clean
clean: ## Remove all tools
	rm -rf bin
	rm -rf share
