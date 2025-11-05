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

ROOT_DIR_RELATIVE := ../..
include $(ROOT_DIR_RELATIVE)/common.mk

GOLANGCI_LINT_VERSION ?= v2.6.1

# GOTESTSUM version without the leading 'v'
GOTESTSUM_VERSION ?= 1.12.0

# Directories.
BIN_DIR := bin
SHARE_DIR := share

OS := $(shell go env GOOS)
ARCH ?= $(shell go env GOARCH)
RUST_TARGET := unknown-$(OS)-gnu

ifeq ($(OS), darwin)
	RUST_TARGET := apple-darwin
	GH_ARCH_SUFFIX := macOS_amd64
	GTAR := gtar
endif

ifeq ($(OS), linux)
	GH_ARCH_SUFFIX := linux_amd64
	GTAR := tar
endif

MDBOOK_EXTRACT_COMMAND := tar xfvz $(SHARE_DIR)/mdbook.tar.gz -C $(BIN_DIR)
MDBOOK_ARCHIVE_EXT := .tar.gz

ifeq ($(OS), windows)
	RUST_TARGET := pc-windows-msvc
	MDBOOK_ARCHIVE_EXT := .zip
	MDBOOK_EXTRACT_COMMAND := unzip -d /tmp
endif

## --------------------------------------
## Tooling Binaries
## --------------------------------------

$(BIN_DIR):
	mkdir -p $@

$(SHARE_DIR):
	mkdir -p $@

.PHONY: $(GTAR)
$(GTAR):
	@$(GTAR) --version > /dev/null || (echo Install GNU Tar with brew install gnu-tar && exit -1)


CONTROLLER_GEN := $(BIN_DIR)/controller-gen
$(CONTROLLER_GEN): go.mod go.sum | $(BIN_DIR) # Build controller-gen from tools folder.
	go build -tags=tools -o $@ sigs.k8s.io/controller-tools/cmd/controller-gen

CONVERSION_GEN := $(BIN_DIR)/conversion-gen
$(CONVERSION_GEN): go.mod go.sum | $(BIN_DIR)
	go build -tags=tools -o $@ k8s.io/code-generator/cmd/conversion-gen

OPENAPI_GEN := $(BIN_DIR)/openapi-gen
$(OPENAPI_GEN): go.mod go.sum | $(BIN_DIR)
	go build -tags=tools -o $@ k8s.io/kube-openapi/cmd/openapi-gen

APPLYCONFIGURATION_GEN := $(BIN_DIR)/applyconfiguration-gen
$(APPLYCONFIGURATION_GEN): go.mod go.sum | $(BIN_DIR)
	go build -tags=tools -o $@ k8s.io/code-generator/cmd/applyconfiguration-gen

CLIENT_GEN := $(BIN_DIR)/client-gen
$(CLIENT_GEN): go.mod go.sum | $(BIN_DIR)
	go build -tags=tools -o $@ k8s.io/code-generator/cmd/client-gen

LISTER_GEN := $(BIN_DIR)/lister-gen
$(LISTER_GEN): go.mod go.sum | $(BIN_DIR)
	go build -tags=tools -o $@ k8s.io/code-generator/cmd/lister-gen

INFORMER_GEN := $(BIN_DIR)/informer-gen
$(INFORMER_GEN): go.mod go.sum | $(BIN_DIR)
	go build -tags=tools -o $@ k8s.io/code-generator/cmd/informer-gen

ENVSUBST := $(BIN_DIR)/envsubst
$(ENVSUBST): go.mod go.sum | $(BIN_DIR) # Build envsubst from tools folder.
	go build -tags=tools -o $@ github.com/a8m/envsubst/cmd/envsubst

SETUP_ENVTEST := $(BIN_DIR)/setup-envtest
$(SETUP_ENVTEST): go.mod go.sum | $(BIN_DIR)
	go build -tags=tools -o $@ sigs.k8s.io/controller-runtime/tools/setup-envtest

GOTESTSUM := $(BIN_DIR)/gotestsum
$(GOTESTSUM): | $(BIN_DIR)
	curl -L https://github.com/gotestyourself/gotestsum/releases/download/v$(GOTESTSUM_VERSION)/gotestsum_$(GOTESTSUM_VERSION)_$(OS)_$(ARCH).tar.gz | \
		tar -C $(BIN_DIR) -zxvf - gotestsum

GH_SHARE := $(SHARE_DIR)/gh

$(GH_SHARE): $(SHARE_DIR)
	mkdir -p $@

$(GH_SHARE)/gh.tar.gz: $(GH_SHARE)
	curl -L "https://github.com/cli/cli/releases/download/v$(GH_VERSION)/gh_$(GH_VERSION)_$(GH_ARCH_SUFFIX).tar.gz" -o $@

GH := $(BIN_DIR)/gh
$(GH): $(GTAR) $(GH_SHARE)/gh.tar.gz | $(BIN_DIR)
	$(GTAR) -xvf $(SHARE_DIR)/gh/gh.tar.gz gh_$(GH_VERSION)_$(GH_ARCH_SUFFIX)/bin/gh --strip-components 1 --directory $(BIN_DIR)
	chmod +x $@
	touch -m $@

GINKGO := $(BIN_DIR)/ginkgo
$(GINKGO): go.mod go.sum | $(BIN_DIR)
	go build -tags=tools -o $@ github.com/onsi/ginkgo/v2/ginkgo

GOJQ := $(BIN_DIR)/gojq
$(GOJQ): go.mod go.sum | $(BIN_DIR)
	go build -tags=tools -o $@ github.com/itchyny/gojq/cmd/gojq

GOLANGCI_LINT := $(BIN_DIR)/golangci-lint
$(GOLANGCI_LINT): Makefile ensure-golangci-lint.sh | $(BIN_DIR)
	./ensure-golangci-lint.sh -b $(BIN_DIR) $(GOLANGCI_LINT_VERSION)

KUSTOMIZE := $(BIN_DIR)/kustomize
$(KUSTOMIZE): go.mod go.sum | $(BIN_DIR) # Build kustomize from tools folder.
	CGO_ENABLED=0 go build -tags=tools -o $@ sigs.k8s.io/kustomize/kustomize/v5

MDBOOK_SHARE := $(SHARE_DIR)/mdbook$(MDBOOK_ARCHIVE_EXT)
$(MDBOOK_SHARE): ../../versions.mk $(SHARE_DIR)
	curl -sL -o $(MDBOOK_SHARE) "https://github.com/rust-lang/mdBook/releases/download/$(MDBOOK_VERSION)/mdBook-$(MDBOOK_VERSION)-x86_64-$(RUST_TARGET)$(MDBOOK_ARCHIVE_EXT)"

MDBOOK := $(BIN_DIR)/mdbook
$(MDBOOK): $(MDBOOK_SHARE) | $(BIN_DIR)
	$(MDBOOK_EXTRACT_COMMAND)
	chmod +x $@
	touch -m $@

MDBOOK_EMBED := $(BIN_DIR)/mdbook-embed
$(MDBOOK_EMBED): go.mod go.sum | $(BIN_DIR)
	go build -tags=tools -o $(BIN_DIR)/mdbook-embed sigs.k8s.io/cluster-api/hack/tools/mdbook/embed

MDBOOK_RELEASELINK := $(BIN_DIR)/mdbook-releaselink
$(MDBOOK_RELEASELINK): go.mod go.sum | $(BIN_DIR)
	go build -tags=tools -o $(BIN_DIR)/mdbook-releaselink sigs.k8s.io/cluster-api/hack/tools/mdbook/releaselink

MDBOOK_TABULATE := $(BIN_DIR)/mdbook-tabulate
$(MDBOOK_TABULATE): go.mod go.sum | $(BIN_DIR)
	go build -tags=tools -o $(BIN_DIR)/mdbook-tabulate sigs.k8s.io/cluster-api/hack/tools/mdbook/tabulate

MOCKGEN := $(BIN_DIR)/mockgen
$(MOCKGEN): go.mod go.sum | $(BIN_DIR) # Build mockgen from tools folder.
	go build -tags=tools -o $@ go.uber.org/mock/mockgen

RELEASE_NOTES := $(BIN_DIR)/release-notes
$(RELEASE_NOTES): go.mod go.sum | $(BIN_DIR)
	go build -tags tools -o $@ sigs.k8s.io/cluster-api/hack/tools/release/notes

PLANTUML := $(BIN_DIR)/plantuml-sentinal
$(PLANTUML): plantuml.Dockerfile ../../versions.mk | $(BIN_DIR)
	docker build --build-arg PLANTUML_VERSION=$(PLANTUML_VERSION)  . -f plantuml.Dockerfile -t "plantuml-builder"
	touch $@

GEN_CRD_API_REFERENCE_DOCS := $(BIN_DIR)/gen-crd-api-reference-docs
$(GEN_CRD_API_REFERENCE_DOCS):
	go build -tags tools -o $@ github.com/ahmetb/gen-crd-api-reference-docs

.PHONY: clean
clean: ## Remove all tools
	rm -rf $(BIN_DIR)
	rm -rf $(SHARE_DIR)
