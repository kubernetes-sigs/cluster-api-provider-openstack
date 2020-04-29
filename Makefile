
GIT_HOST = sigs.k8s.io
PWD := $(shell pwd)
BASE_DIR := $(shell basename $(PWD))
# Keep an existing GOPATH, make a private one if it is undefined
GOPATH_DEFAULT := $(PWD)/.go
export GOPATH ?= $(GOPATH_DEFAULT)
GOBIN_DEFAULT := $(GOPATH)/bin
export GOBIN ?= $(GOBIN_DEFAULT)
TESTARGS_DEFAULT := "-v"
export TESTARGS ?= $(TESTARGS_DEFAULT)
DEST := $(GOPATH)/src/$(GIT_HOST)/$(BASE_DIR)
SOURCES := $(shell find $(DEST) -name '*.go')

HAS_LINT := $(shell command -v golint;)
HAS_GOX := $(shell command -v gox;)
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

ifneq ("$(realpath $(DEST))", "$(realpath $(PWD))")
    $(error Please run 'make' from $(DEST). Current directory is $(PWD))
endif

# CTI targets

$(GOBIN):
	echo "create gobin"
	mkdir -p $(GOBIN)

work: $(GOBIN)

build: manager

manager:
	CGO_ENABLED=0 GOOS=$(GOOS) go build \
		-ldflags $(LDFLAGS) \
		-o bin/manager \
		cmd/manager/main.go

test: unit functional

check: fmt vet lint

unit:
	go test -tags=unit $(shell go list ./...) $(TESTARGS)

functional:
	@echo "$@ not yet implemented"

fmt:
	hack/verify-gofmt.sh

lint:
ifndef HAS_LINT
		go get -u golang.org/x/lint/golint
		echo "installing golint"
endif
	hack/verify-golint.sh

vet:
	go vet ./...

cover:
	go test -tags=unit $(shell go list ./...) -cover

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
	@echo "GOPATH: $(GOPATH)"
	@echo "GOROOT: $(GOROOT)"
	@echo "DEST: $(DEST)"
	go version
	go env

clean:
	rm -rf _dist bin/manager

realclean: clean
	rm -rf vendor
	if [ "$(GOPATH)" = "$(GOPATH_DEFAULT)" ]; then \
		rm -rf $(GOPATH); \
	fi

shell:
	$(SHELL) -i

images: openstack-cluster-api-controller

openstack-cluster-api-controller: manager
ifeq ($(GOOS),linux)
	cp bin/manager cmd/manager
	docker build -t $(REGISTRY)/openstack-cluster-api-controller:$(VERSION) cmd/manager
	rm cmd/manager/manager
else
	$(error Please set GOOS=linux for building the image)
endif

upload-images: images
	@echo "push images to $(REGISTRY)"
	docker login -u="$(DOCKER_USERNAME)" -p="$(DOCKER_PASSWORD)";
	docker push $(REGISTRY)/openstack-cluster-api-controller:$(VERSION)

version:
	@echo ${VERSION}

.PHONY: build-cross
build-cross: LDFLAGS += -extldflags "-static"
build-cross:
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

.PHONY: build clean cover docs fmt functional lint realclean \
	relnotes test translation version build-cross dist
