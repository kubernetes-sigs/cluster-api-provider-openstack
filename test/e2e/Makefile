# Copyright 2020 The Kubernetes Authors.
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

# If you update this file, please follow:
# https://suva.sh/posts/well-documented-makefiles/

REPO_ROOT := $(shell git rev-parse --show-toplevel)
include $(REPO_ROOT)/common.mk

# Use GOPROXY environment variable if set

.DEFAULT_GOAL:=help

GOPROXY := $(shell go env GOPROXY)
ifeq ($(GOPROXY),)
GOPROXY := https://proxy.golang.org
endif
export GOPROXY

help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## --------------------------------------
## Binaries
## --------------------------------------

GINGKO_VER := v1.16.5
GINKGO_BIN := ginkgo
GINKGO := $(abspath $(TOOLS_BIN_DIR)/$(GINKGO_BIN)-$(GINGKO_VER))
GINKGO_PKG := github.com/onsi/ginkgo/ginkgo
GO_INSTALL := $(REPO_ROOT)/scripts/go_install.sh
KUSTOMIZE_BIN := kustomize
KUSTOMIZE_VER := $(shell grep "^KUSTOMIZE_VER" $(REPO_ROOT)/Makefile | awk '{print $$NF}')
KUSTOMIZE := $(abspath $(TOOLS_BIN_DIR)/$(KUSTOMIZE_BIN)-$(KUSTOMIZE_VER))
KUSTOMIZE_PKG := sigs.k8s.io/kustomize/kustomize/v4

$(GINKGO): # Build ginkgo from tools folder.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) $(GINKGO_PKG) $(GINKGO_BIN) $(GINGKO_VER)

$(KUSTOMIZE): # Build kustomize from tools folder.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) $(KUSTOMIZE_PKG) $(KUSTOMIZE_BIN) $(KUSTOMIZE_VER)

.PHONY: $(GINKGO_BIN)
$(GINKGO_BIN): $(GINKGO) ## Build a local copy of ginkgo

.PHONY: $(KUSTOMIZE_BIN)
$(KUSTOMIZE_BIN): $(KUSTOMIZE) ## Build a local copy of kustomize

## --------------------------------------
## Templates
## --------------------------------------

TEMPLATES_DIR := $(REPO_ROOT)/test/e2e/data/infrastructure-openstack
BASES_DIR := $(TEMPLATES_DIR)/kustomize_sources

.PHONY: cluster-templates
cluster-templates: $(KUSTOMIZE)
	$(KUSTOMIZE) build $(BASES_DIR)/default > $(TEMPLATES_DIR)/cluster-template.yaml
	$(KUSTOMIZE) build $(BASES_DIR)/external-cloud-provider > $(TEMPLATES_DIR)/cluster-template-external-cloud-provider.yaml
	$(KUSTOMIZE) build $(BASES_DIR)/multi-az > $(TEMPLATES_DIR)/cluster-template-multi-az.yaml
	$(KUSTOMIZE) build $(BASES_DIR)/multi-network > $(TEMPLATES_DIR)/cluster-template-multi-network.yaml
	$(KUSTOMIZE) build $(BASES_DIR)/without-lb > $(TEMPLATES_DIR)/cluster-template-without-lb.yaml

## --------------------------------------
## Testing
## --------------------------------------

GINKGO_FOCUS  ?=
GINKGO_SKIP ?=
GINKGO_NODES  ?= 1
E2E_DATA_DIR ?= ${REPO_ROOT}/test/e2e/data
E2E_CONF_FILE  ?= ${E2E_DATA_DIR}/e2e_conf.yaml
ARTIFACTS ?= ${REPO_ROOT}/_artifacts
SKIP_RESOURCE_CLEANUP ?= false
USE_EXISTING_CLUSTER ?= false
GINKGO_NOCOLOR ?= false
SUITE ?= e2e

# to set multiple ginkgo skip flags, if any
ifneq ($(strip $(GINKGO_SKIP)),)
_SKIP_ARGS := $(foreach arg,$(strip $(GINKGO_SKIP)),-skip="$(arg)")
endif

.PHONY: run
run: $(GINKGO) cluster-templates ## Run the end-to-end tests
	$(GINKGO) -v --failFast -trace -tags=e2e -focus="$(GINKGO_FOCUS)" $(_SKIP_ARGS) -nodes=$(GINKGO_NODES) --noColor=$(GINKGO_NOCOLOR) $(GINKGO_ARGS) ./suites/$(SUITE)/... -- \
	    -artifacts-folder="$(ARTIFACTS)" -config-path="$(E2E_CONF_FILE)" -data-folder="$(E2E_DATA_DIR)" $(TEST_ARGS)
