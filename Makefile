# Copyright 2019 The Kubernetes Authors.
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

# If you update this file, please follow
# https://suva.sh/posts/well-documented-makefiles

# Ensure Make is run with bash shell as some syntax below is bash-specific
SHELL:=/usr/bin/env bash

.DEFAULT_GOAL:=help

# Use GOPROXY environment variable if set
GOPROXY := $(shell go env GOPROXY)
ifeq ($(GOPROXY),)
GOPROXY := https://proxy.golang.org
endif
export GOPROXY

# Activate module mode, as we use go modules to manage dependencies
export GO111MODULE=on

# Directories.
TOOLS_DIR := hack/tools
TOOLS_BIN_DIR := $(TOOLS_DIR)/bin
BIN_DIR := bin

# Binaries.
KUSTOMIZE := $(TOOLS_BIN_DIR)/kustomize
CLUSTERCTL := $(BIN_DIR)/clusterctl
CONTROLLER_GEN := $(TOOLS_BIN_DIR)/controller-gen
ENVSUBST := $(TOOLS_BIN_DIR)/envsubst
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/golangci-lint
MOCKGEN := $(TOOLS_BIN_DIR)/mockgen
CONVERSION_GEN := $(TOOLS_BIN_DIR)/conversion-gen
RELEASE_NOTES_BIN := bin/release-notes
RELEASE_NOTES := $(TOOLS_DIR)/$(RELEASE_NOTES_BIN)
GINKGO := $(abspath $(TOOLS_BIN_DIR)/ginkgo)

# Define Docker related variables. Releases should modify and double check these vars.
REGISTRY ?= gcr.io/k8s-staging-capi-openstack
STAGING_REGISTRY := gcr.io/k8s-staging-capi-openstack
PROD_REGISTRY := us.gcr.io/k8s-artifacts-prod/capi-openstack
IMAGE_NAME ?= capi-openstack-controller
CONTROLLER_IMG ?= $(REGISTRY)/$(IMAGE_NAME)
TAG ?= dev
ARCH ?= amd64
ALL_ARCH = amd64 arm arm64 ppc64le s390x
CAPI_VERSION = 0.3.10

# Allow overriding manifest generation destination directory
MANIFEST_ROOT ?= config
CRD_ROOT ?= $(MANIFEST_ROOT)/crd/bases
WEBHOOK_ROOT ?= $(MANIFEST_ROOT)/webhook
RBAC_ROOT ?= $(MANIFEST_ROOT)/rbac

# Allow overriding the imagePullPolicy
PULL_POLICY ?= IfNotPresent

# Hosts running SELinux need :z added to volume mounts
SELINUX_ENABLED := $(shell cat /sys/fs/selinux/enforce 2> /dev/null || echo 0)

ifeq ($(SELINUX_ENABLED),1)
  DOCKER_VOL_OPTS?=:z
endif

# Set build time variables including version details
LDFLAGS := $(shell source ./hack/version.sh; version::ldflags)

## --------------------------------------
## Help
## --------------------------------------

help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## --------------------------------------
## Define targets for prow
## --------------------------------------


.PHONY: images
images: docker-build ## Build all images

.PHONY: check
check: modules generate lint test verify

## --------------------------------------
## Testing
## --------------------------------------

.PHONY: test
test: generate lint ## Run tests
	$(MAKE) test-go

.PHONY: test-go
test-go: ## Run golang tests
	go test -v ./...

## --------------------------------------
## Binaries
## --------------------------------------
$(GINKGO): $(TOOLS_DIR)/go.mod
	cd $(TOOLS_DIR) && go build -tags=tools -o $(BIN_DIR)/ginkgo github.com/onsi/ginkgo/ginkgo

.PHONY: binaries
binaries: manager ## Builds and installs all binaries

.PHONY: manager
manager: ## Build manager binary.
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/manager .

## --------------------------------------
## Tooling Binaries
## --------------------------------------

$(KUSTOMIZE): $(TOOLS_DIR)/go.mod # Build kustomize from tools folder.
	cd $(TOOLS_DIR); go build -tags=tools -o $(BIN_DIR)/kustomize sigs.k8s.io/kustomize/kustomize/v3

$(CONTROLLER_GEN): $(TOOLS_DIR)/go.mod # Build controller-gen from tools folder.
	cd $(TOOLS_DIR); go build -tags=tools -o $(BIN_DIR)/controller-gen sigs.k8s.io/controller-tools/cmd/controller-gen

$(ENVSUBST): $(TOOLS_DIR)/go.mod # Build envsubst from tools folder.
	cd $(TOOLS_DIR); go build -tags=tools -o $(BIN_DIR)/envsubst github.com/a8m/envsubst/cmd/envsubst

$(GOLANGCI_LINT): $(TOOLS_DIR)/go.mod # Build golangci-lint from tools folder.
	cd $(TOOLS_DIR); go build -tags=tools -o $(BIN_DIR)/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

$(MOCKGEN): $(TOOLS_DIR)/go.mod # Build mockgen from tools folder.
	cd $(TOOLS_DIR); go build -tags=tools -o $(BIN_DIR)/mockgen github.com/golang/mock/mockgen

$(CONVERSION_GEN): $(TOOLS_DIR)/go.mod
	cd $(TOOLS_DIR); go build -tags=tools -o $(BIN_DIR)/conversion-gen k8s.io/code-generator/cmd/conversion-gen

$(RELEASE_NOTES) : $(TOOLS_DIR)/go.mod
	cd $(TOOLS_DIR) && go build -tags tools -o $(BIN_DIR)/release-notes sigs.k8s.io/cluster-api/hack/tools/release

## --------------------------------------
## Linting
## --------------------------------------

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Lint codebase
	$(GOLANGCI_LINT) run -v --fast=false

lint-fast: $(GOLANGCI_LINT) ## Run only faster linters to detect possible issues
	$(GOLANGCI_LINT) run -v --fast=true

## --------------------------------------
## Generate
## --------------------------------------

.PHONY: modules
modules: ## Runs go mod to ensure proper vendoring.
	go mod tidy
	cd $(TOOLS_DIR); go mod tidy

.PHONY: generate
generate: ## Generate code
	$(MAKE) generate-go
	$(MAKE) generate-manifests

.PHONY: generate-go
generate-go: $(CONTROLLER_GEN) $(CONVERSION_GEN) $(MOCKGEN) ## Runs Go related generate targets
	$(CONTROLLER_GEN) \
		paths=./api/... \
		object:headerFile=./hack/boilerplate/boilerplate.generatego.txt

	$(CONVERSION_GEN) \
		--input-dirs=./api/v1alpha3 \
		--output-file-base=zz_generated.conversion \
		--go-header-file=./hack/boilerplate/boilerplate.generatego.txt
	go generate ./...

.PHONY: generate-manifests
generate-manifests: $(CONTROLLER_GEN) ## Generate manifests e.g. CRD, RBAC etc.
	$(CONTROLLER_GEN) \
		paths=./api/... \
		crd:crdVersions=v1 \
		output:crd:dir=$(CRD_ROOT) \
		output:webhook:dir=$(WEBHOOK_ROOT) \
		webhook
	$(CONTROLLER_GEN) \
		paths=./controllers/... \
		output:rbac:dir=$(RBAC_ROOT) \
		rbac:roleName=manager-role

## --------------------------------------
## Docker
## --------------------------------------

.PHONY: docker-build
docker-build: ## Build the docker image for controller-manager
	docker build --pull --build-arg ARCH=$(ARCH) --build-arg LDFLAGS="$(LDFLAGS)" . -t $(CONTROLLER_IMG)-$(ARCH):$(TAG)
	MANIFEST_IMG=$(CONTROLLER_IMG)-$(ARCH) MANIFEST_TAG=$(TAG) $(MAKE) set-manifest-image
	$(MAKE) set-manifest-pull-policy

.PHONY: docker-push
docker-push: ## Push the docker image
	docker push $(CONTROLLER_IMG)-$(ARCH):$(TAG)

## --------------------------------------
## Docker — All ARCH
## --------------------------------------

.PHONY: docker-build-all ## Build all the architecture docker images
docker-build-all: $(addprefix docker-build-,$(ALL_ARCH))

docker-build-%:
	$(MAKE) ARCH=$* docker-build

.PHONY: docker-push-all ## Push all the architecture docker images
docker-push-all: $(addprefix docker-push-,$(ALL_ARCH))
	$(MAKE) docker-push-manifest

docker-push-%:
	$(MAKE) ARCH=$* docker-push

.PHONY: docker-push-manifest
docker-push-manifest: ## Push the fat manifest docker image.
	## Minimum docker version 18.06.0 is required for creating and pushing manifest images.
	docker manifest create --amend $(CONTROLLER_IMG):$(TAG) $(shell echo $(ALL_ARCH) | sed -e "s~[^ ]*~$(CONTROLLER_IMG)\-&:$(TAG)~g")
	@for arch in $(ALL_ARCH); do docker manifest annotate --arch $${arch} ${CONTROLLER_IMG}:${TAG} ${CONTROLLER_IMG}-$${arch}:${TAG}; done
	docker manifest push --purge ${CONTROLLER_IMG}:${TAG}
	MANIFEST_IMG=$(CONTROLLER_IMG) MANIFEST_TAG=$(TAG) $(MAKE) set-manifest-image
	$(MAKE) set-manifest-pull-policy

.PHONY: set-manifest-image
set-manifest-image:
	$(info Updating kustomize image patch file for manager resource)
	sed -i'' -e 's@image: .*@image: '"${MANIFEST_IMG}:$(MANIFEST_TAG)"'@' ./config/manager/manager_image_patch.yaml


.PHONY: set-manifest-pull-policy
set-manifest-pull-policy:
	$(info Updating kustomize pull policy file for manager resource)
	sed -i'' -e 's@imagePullPolicy: .*@imagePullPolicy: '"$(PULL_POLICY)"'@' ./config/manager/manager_pull_policy.yaml

## --------------------------------------
## Release
## --------------------------------------

RELEASE_TAG := $(shell git describe --abbrev=0 2>/dev/null)
RELEASE_DIR := out

$(RELEASE_DIR):
	mkdir -p $(RELEASE_DIR)/

.PHONY: release
release: clean-release  ## Builds and push container images using the latest git tag for the commit.
	@if [ -z "${RELEASE_TAG}" ]; then echo "RELEASE_TAG is not set"; exit 1; fi
	@if ! [ -z "$$(git status --porcelain)" ]; then echo "Your local git repository contains uncommitted changes, use git clean before proceeding."; exit 1; fi
	git checkout "${RELEASE_TAG}"
	# Set the manifest image to the production bucket.
	$(MAKE) set-manifest-image MANIFEST_IMG=$(PROD_REGISTRY)/$(IMAGE_NAME) MANIFEST_TAG=$(RELEASE_TAG)
	$(MAKE) set-manifest-pull-policy PULL_POLICY=IfNotPresent
	$(MAKE) release-manifests

.PHONY: release-manifests
release-manifests: $(RELEASE_DIR) $(KUSTOMIZE) ## Builds the manifests to publish with a release
	$(KUSTOMIZE) build config > $(RELEASE_DIR)/infrastructure-components.yaml

.PHONY: release-staging
release-staging: ## Builds and push container images to the staging bucket.
	REGISTRY=$(STAGING_REGISTRY) $(MAKE) docker-build-all docker-push-all release-alias-tag

RELEASE_ALIAS_TAG=$(PULL_BASE_REF)

.PHONY: release-alias-tag
release-alias-tag: # Adds the tag to the last build tag.
	gcloud container images add-tag $(CONTROLLER_IMG):$(TAG) $(CONTROLLER_IMG):$(RELEASE_ALIAS_TAG)

.PHONY: release-notes
release-notes: $(RELEASE_NOTES)
	$(RELEASE_NOTES) $(ARGS)

## --------------------------------------
## Development
## --------------------------------------

# Properties for create-cluster
OPENSTACK_FAILURE_DOMAIN ?= "nova"
OPENSTACK_CLOUD ?= "capi-quickstart"
OPENSTACK_CLOUD_CACERT_B64 ?= "Cg=="
OPENSTACK_CLOUD_PROVIDER_CONF_B64 ?= ""
OPENSTACK_CLOUD_YAML_B64 ?= ""
OPENSTACK_DNS_NAMESERVERS ?= ""
OPENSTACK_IMAGE_NAME ?= "ubuntu-1910-kube-v1.17.3"
OPENSTACK_BASTION_IMAGE_NAME ?= "cirros"
OPENSTACK_NODE_MACHINE_FLAVOR ?= "m1.small"
OPENSTACK_CONTROL_PLANE_MACHINE_FLAVOR ?= "m1.medium"
OPENSTACK_BASTION_MACHINE_FLAVOR ?= "m1.tiny"
CLUSTER_NAME ?= "capi-quickstart"
OPENSTACK_SSH_KEY_NAME ?= "${CLUSTER_NAME}-key"
OPENSTACK_CLUSTER_TEMPLATE ?= "./templates/cluster-template-without-lb.yaml"
KUBERNETES_VERSION ?= "v1.17.3"
CONTROL_PLANE_MACHINE_COUNT ?= "1"
WORKER_MACHINE_COUNT ?= "3"
LOAD_IMAGE=$(CONTROLLER_IMG)-$(ARCH):$(TAG)

.PHONY: create-cluster
create-cluster: $(KUSTOMIZE) $(ENVSUBST) ## Create a development Kubernetes cluster on OpenStack in a KIND management cluster.
	mkdir $(BIN_DIR)
	wget https://github.com/kubernetes-sigs/cluster-api/releases/download/v${CAPI_VERSION}/clusterctl-linux-amd64
	mv ./clusterctl-linux-amd64 $(CLUSTERCTL)
	mkdir ~/.cluster-api
	chmod +x $(CLUSTERCTL)
	# Create clusterctl.yaml to use local OpenStack provider
	mkdir -p ./out/infrastructure-openstack/v0.3.1
	echo "providers:" > ./out/clusterctl.yaml
	echo "- name: openstack" >> ./out/clusterctl.yaml
	echo "  url: $(PWD)/out/infrastructure-openstack/v0.3.1/infrastructure-components.yaml" >> ./out/clusterctl.yaml
	echo "  type: InfrastructureProvider" >> ./out/clusterctl.yaml

	echo "releaseSeries:" > ./out/infrastructure-openstack/v0.3.1/metadata.yaml
	echo "- major: 0" >> ./out/infrastructure-openstack/v0.3.1/metadata.yaml
	echo "  minor: 3" >> ./out/infrastructure-openstack/v0.3.1/metadata.yaml
	echo "  contract: v1alpha3" >> ./out/infrastructure-openstack/v0.3.1/metadata.yaml

	@if [ -z `kind get clusters | grep clusterapi` ]; then \
		kind create cluster --name=clusterapi; \
	fi
	@if [ ! -z "${LOAD_IMAGE}" ]; then \
		echo "loading ${LOAD_IMAGE} into kind cluster ..." && \
		kind --name="clusterapi" load docker-image "${LOAD_IMAGE}"; \
	fi

	# (Re-)install Core providers
	$(CLUSTERCTL) delete --all

	# (Re-)deploy CAPO provider
	MANIFEST_IMG=$(CONTROLLER_IMG)-$(ARCH) MANIFEST_TAG=$(TAG) $(MAKE) set-manifest-image
	$(KUSTOMIZE) build config > ./out/infrastructure-openstack/v0.3.1/infrastructure-components.yaml
	$(CLUSTERCTL) delete --infrastructure openstack --include-namespace --namespace capo-system || true
	kubectl wait --for=delete ns/capo-system || true
	$(CLUSTERCTL) init --config ./out/clusterctl.yaml --infrastructure openstack --core cluster-api:v${CAPI_VERSION} --bootstrap kubeadm:v${CAPI_VERSION} --control-plane kubeadm:v${CAPI_VERSION}

	# Wait for CAPI pods
	kubectl wait --for=condition=Ready --timeout=5m -n capi-system pod --all
	kubectl wait --for=condition=Ready --timeout=5m -n capi-webhook-system pod --all
	kubectl wait --for=condition=Ready --timeout=5m -n capi-kubeadm-bootstrap-system pod --all
	kubectl wait --for=condition=Ready --timeout=5m -n capi-kubeadm-control-plane-system pod --all
	kubectl wait --for=condition=Ready --timeout=5m -n capo-system pod --all

	# Wait for CAPO CRDs
	kubectl wait --for condition=established --timeout=60s crds/openstackmachines.infrastructure.cluster.x-k8s.io
	kubectl wait --for condition=established --timeout=60s crds/openstackmachinetemplates.infrastructure.cluster.x-k8s.io
	kubectl wait --for condition=established --timeout=60s crds/openstackclusters.infrastructure.cluster.x-k8s.io

    # Wait until everything is really ready, as we had some problems with pods being ready but not yet
    # available when deploying the cluster.
	sleep 5

	# Create Cluster.
	kubectl create ns $(CLUSTER_NAME) || true
	PULL_POLICY=$(PULL_POLICY) \
	OPENSTACK_FAILURE_DOMAIN=$(OPENSTACK_FAILURE_DOMAIN) \
	OPENSTACK_CLOUD=$(OPENSTACK_CLOUD) \
	OPENSTACK_CLOUD_CACERT_B64=$(OPENSTACK_CLOUD_CACERT_B64) \
	OPENSTACK_CLOUD_PROVIDER_CONF_B64=$(OPENSTACK_CLOUD_PROVIDER_CONF_B64) \
	OPENSTACK_CLOUD_YAML_B64=$(OPENSTACK_CLOUD_YAML_B64) \
	OPENSTACK_DNS_NAMESERVERS=$(OPENSTACK_DNS_NAMESERVERS) \
	OPENSTACK_IMAGE_NAME=$(OPENSTACK_IMAGE_NAME) \
	OPENSTACK_SSH_KEY_NAME=$(OPENSTACK_SSH_KEY_NAME) \
	OPENSTACK_NODE_MACHINE_FLAVOR=$(OPENSTACK_NODE_MACHINE_FLAVOR) \
	OPENSTACK_CONTROL_PLANE_MACHINE_FLAVOR=$(OPENSTACK_CONTROL_PLANE_MACHINE_FLAVOR) \
	  $(CLUSTERCTL) config cluster $(CLUSTER_NAME) \
	    --from=$(OPENSTACK_CLUSTER_TEMPLATE) \
	    --kubernetes-version $(KUBERNETES_VERSION) \
	    --control-plane-machine-count=$(CONTROL_PLANE_MACHINE_COUNT) \
	    --worker-machine-count=$(WORKER_MACHINE_COUNT) > ./hack/ci/e2e-conformance/cluster.yaml

    # Patch Kubernetes version
	cat ./hack/ci/e2e-conformance/e2e-conformance_patch.yaml.tpl | \
	  sed "s|\$${OPENSTACK_CLOUD_PROVIDER_CONF_B64}|$(OPENSTACK_CLOUD_PROVIDER_CONF_B64)|" | \
	  sed "s|\$${OPENSTACK_CLOUD_CACERT_B64}|$(OPENSTACK_CLOUD_CACERT_B64)|" | \
	  sed "s|\$${KUBERNETES_VERSION}|$(KUBERNETES_VERSION)|" | \
	  sed "s|\$${CLUSTER_NAME}|$(CLUSTER_NAME)|" | \
	  sed "s|\$${OPENSTACK_BASTION_MACHINE_FLAVOR}|$(OPENSTACK_BASTION_MACHINE_FLAVOR)|" | \
	  sed "s|\$${OPENSTACK_BASTION_IMAGE_NAME}|$(OPENSTACK_BASTION_IMAGE_NAME)|" | \
	  sed "s|\$${OPENSTACK_SSH_KEY_NAME}|$(OPENSTACK_SSH_KEY_NAME)|" \
	   > ./hack/ci/e2e-conformance/e2e-conformance_patch.yaml
	$(KUSTOMIZE) build --reorder=none hack/ci/e2e-conformance  > ./out/cluster.yaml

	# Deploy cluster
	kubectl apply -f ./out/cluster.yaml

	# Wait for the kubeconfig to become available.
	timeout 300 bash -c "while ! kubectl get secrets | grep $(CLUSTER_NAME)-kubeconfig; do sleep 10; done"
	# Get kubeconfig and store it locally.
	kubectl get secrets $(CLUSTER_NAME)-kubeconfig -o json | jq -r .data.value | base64 --decode > ./kubeconfig
	timeout 900 bash -c "while ! kubectl --kubeconfig=./kubeconfig get nodes | grep master; do sleep 10; done"

	# Deploy calico
	curl https://docs.projectcalico.org/manifests/calico.yaml | sed "s/veth_mtu:.*/veth_mtu: \"1430\"/g" | \
		kubectl --kubeconfig=./kubeconfig apply -f -

.PHONY: delete-cluster
delete-cluster:
	kubectl delete cluster --all --ignore-not-found

	kubectl get machinedeployment,kubeadmcontrolplane,cluster

	@if [[ `kubectl get machinedeployment,kubeadmcontrolplane,cluster | wc -l` -gt 0 ]]; then \
	  echo "Error: not all resources have been deleted correctly"; \
	  exit 1; \
	fi

.PHONY: kind-reset
kind-reset: ## Destroys the "clusterapi" kind cluster.
	kind delete cluster --name=clusterapi || true

## --------------------------------------
## Cleanup / Verification
## --------------------------------------

.PHONY: clean
clean: ## Remove all generated files
	$(MAKE) clean-bin
	$(MAKE) clean-temporary

.PHONY: clean-bin
clean-bin: ## Remove all generated binaries
	rm -rf bin
	rm -rf hack/tools/bin

.PHONY: clean-temporary
clean-temporary: ## Remove all temporary files and folders
	rm -f minikube.kubeconfig
	rm -f kubeconfig

.PHONY: clean-release
clean-release: ## Remove the release folder
	rm -rf $(RELEASE_DIR)

.PHONY: verify
verify: verify-boilerplate verify-modules verify-gen

.PHONY: verify-boilerplate
verify-boilerplate:
	./hack/verify-boilerplate.sh

.PHONY: verify-modules
verify-modules: modules
	@if !(git diff --quiet HEAD -- go.sum go.mod hack/tools/go.mod hack/tools/go.sum); then \
		git diff; \
		echo "go module files are out of date"; exit 1; \
	fi

verify-gen: generate
	@if !(git diff --quiet HEAD); then \
		git diff; \
		echo "generated files are out of date, run make generate"; exit 1; \
	fi
