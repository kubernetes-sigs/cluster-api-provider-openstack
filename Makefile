

# Allow overriding manifest generation destination directory
MANIFEST_ROOT ?= "config"
CRD_ROOT ?= "$(MANIFEST_ROOT)/crd/bases"
WEBHOOK_ROOT ?= "$(MANIFEST_ROOT)/webhook"
RBAC_ROOT ?= "$(MANIFEST_ROOT)/rbac"



GIT_HOST = sigs.k8s.io
PWD := $(shell pwd)
BASE_DIR := $(shell basename $(PWD))

HAS_LINT := $(shell command -v golint;)
HAS_GOX := $(shell command -v gox;)
HAS_YQ := $(shell command -v yq;)
HAS_KUSTOMIZE := $(shell command -v kustomize;)
HAS_ENVSUBST := $(shell command -v envsubst;)
GOX_PARALLEL ?= 3
TARGETS ?= darwin/amd64 linux/amd64 linux/386 linux/arm linux/arm64 linux/ppc64le
DIST_DIRS         = find * -type d -exec

GOOS ?= $(shell go env GOOS)
VERSION ?= $(shell git describe --exact-match 2> /dev/null || \
                 git describe --match=$(git rev-parse --short=8 HEAD) --always --dirty --abbrev=8)
GOFLAGS   :=
TAGS      :=
LDFLAGS   := "-w -s -X 'main.version=${VERSION}'"
REGISTRY ?= k8scloudprovider

MANAGER_IMAGE_NAME ?= cluster-api-provider-openstack
MANAGER_IMAGE_TAG ?= dev
PULL_POLICY ?= Always

# Used in docker-* targets.
MANAGER_IMAGE ?= $(REGISTRY)/$(MANAGER_IMAGE_NAME):$(MANAGER_IMAGE_TAG)


build: binary images

binary: manager clusterctl

manager:
	CGO_ENABLED=0 GOOS=$(GOOS) go build -v \
		-ldflags $(LDFLAGS) \
		-o bin/manager \
		cmd/manager/main.go

clusterctl:
	CGO_ENABLED=0 GOOS=$(GOOS) go build \
		-ldflags $(LDFLAGS) \
		-o bin/clusterctl \
		cmd/clusterctl/main.go

check: vendor fmt vet lint

fmt:
	hack/verify-gofmt.sh

lint:
ifndef HAS_LINT
		go get -u golang.org/x/lint/golint
		echo "installing golint"
endif
	hack/verify-golint.sh

vet:
	go vet ./pkg/... ./cmd/...

cover: generate vendor
	go test -tags=unit ./pkg/... ./cmd/... -cover

docs:
	@echo "$@ not yet implemented"

godoc:
	@echo "$@ not yet implemented"

releasenotes:
	@echo "Reno not yet implemented for this repo"

translation:
	@echo "$@ not yet implemented"

# Do the work here

# Set up the development environment
env:
	@echo "PWD: $(PWD)"
	@echo "BASE_DIR: $(BASE_DIR)"
	go version
	go env

shell:
	$(SHELL) -i

images: docker-build

# Build the docker image
.PHONY: docker-build
docker-build:
	docker build . -t ${MANAGER_IMAGE}

upload-images: images
	@echo "push images to $(REGISTRY)"
	docker login -u="$(DOCKER_USERNAME)" -p="$(DOCKER_PASSWORD)";
	docker push $(REGISTRY)/openstack-cluster-api-controller:$(VERSION)
	docker push $(REGISTRY)/openstack-cluster-api-clusterctl:$(VERSION)

version:
	@echo ${VERSION}

.PHONY: build-cross
build-cross: LDFLAGS += -extldflags "-static"
build-cross: vendor
ifndef HAS_GOX
	go get -u github.com/mitchellh/gox
endif
	CGO_ENABLED=0 gox -parallel=$(GOX_PARALLEL) -output="_dist/{{.OS}}-{{.Arch}}/{{.Dir}}" -osarch='$(TARGETS)' $(GOFLAGS) $(if $(TAGS),-tags '$(TAGS)',) -ldflags '$(LDFLAGS)' $(GIT_HOST)/$(BASE_DIR)/cmd/openstack-machine-controller/

.PHONY: dist
dist: build-cross
	( \
		cd _dist && \
		$(DIST_DIRS) cp ../LICENSE {} \; && \
		$(DIST_DIRS) cp ../README.md {} \; && \
		$(DIST_DIRS) tar -zcf cluster-api-provider-openstack-$(VERSION)-{}.tar.gz {} \; && \
		$(DIST_DIRS) zip -r cluster-api-provider-openstack-$(VERSION)-{}.zip {} \; \
	)

# TODO(sbueringer) target below are already cleaned up after v1alpha2 refactoring
# targets above have to be cleaned up

## --------------------------------------
## Testing
## --------------------------------------

.PHONY: test
test: generate lint ## Run tests
	$(MAKE) test-go
	$(MAKE) test-generate-examples

.PHONY: test-go
test-go: ## Run tests
	go test -v -tags=unit ./api/... ./pkg/... ./controllers/...

test-generate-examples:
ifndef HAS_YQ
	go get github.com/mikefarah/yq
	echo "installing yq"
endif
ifndef HAS_KUSTOMIZE
	GO111MODULE=on go get sigs.k8s.io/kustomize/v3/cmd/kustomize
	echo "installing kustomize"
endif
ifndef HAS_ENVSUBST
	go get github.com/a8m/envsubst/cmd/envsubst
	echo "installing envsubst"
endif
	# Create a dummy file for test only
	mkdir tmp
	echo 'clouds' > tmp/dummy-clouds-test.yaml
	examples/generate.sh -f tmp/dummy-clouds-test.yaml openstack tmp/dummy-make-auto-test
	# the folder will be generated under same folder of examples
	rm -rf tmp/dummy-make-auto-test
	rm tmp/dummy-clouds-test.yaml

## --------------------------------------
## Generate
## --------------------------------------

.PHONY: vendor
vendor: ## Runs go mod to ensure proper vendoring.
	./hack/update-vendor.sh

.PHONY: generate
generate: ## Generate code
	$(MAKE) generate-go
	$(MAKE) generate-manifests
	$(MAKE) generate-deepcopy

.PHONY: generate-go
generate-go: ## Runs go generate
	go generate ./pkg/... ./cmd/...

.PHONY: generate-manifests
generate-manifests: ## Generate manifests e.g. CRD, RBAC etc.
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go \
		paths=./api/... \
		crd:trivialVersions=true \
		output:crd:dir=$(CRD_ROOT) \
		output:webhook:dir=$(WEBHOOK_ROOT) \
		webhook
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go \
		paths=./controllers/... \
		output:rbac:dir=$(RBAC_ROOT) \
		rbac:roleName=manager-role

.PHONY: generate-deepcopy
generate-deepcopy: ## Runs controller-gen to generate deepcopy files.
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go \
		paths=./api/... \
		object:headerFile=./hack/boilerplate/boilerplate.generatego.txt

.PHONY: generate-examples
generate-examples: clean-examples ## Generate examples configurations to run a cluster.
	./examples/generate.sh

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

.PHONY: clean-temporary
clean-temporary: ## Remove all temporary files and folders
	rm -f minikube.kubeconfig
	rm -f kubeconfig
	rm -rf out/

.PHONY: clean-examples
clean-examples: ## Remove all the temporary files generated in the examples folder
	rm -rf examples/_out/
	rm -f examples/provider-components/provider-components-*.yaml


.PHONY: build clean cover vendor docs fmt functional lint \
	translation version build-cross dist manifests
