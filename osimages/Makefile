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

REPO_ROOT = $(shell git rev-parse --show-toplevel)

ARTIFACTS ?= $(REPO_ROOT)/_artifacts
OSIMAGE_DIR ?= $(ARTIFACTS)/osimages
OSIMAGE_BUILDER ?= registry.k8s.io/scl-image-builder/cluster-node-image-builder-amd64:v0.1.20
STAGING_BUCKET ?= artifacts.k8s-staging-capi-openstack.appspot.com

DISTROS ?= ubuntu-2204 flatcar
KUBERNETES_VERSIONS ?= 1.28.2 1.27.2

# By default, build all distros with all kubernetes versions
all_targets := $(foreach distro,$(DISTROS),$(addprefix osimage-$(distro)-,$(KUBERNETES_VERSIONS)))
all: $(all_targets)

# Set this to 'none' if kvm is not available
ACCELERATOR ?= kvm

ifeq ($(ACCELERATOR),kvm)
	privileged=--privileged
else
	privileged=
endif

$(OSIMAGE_DIR):
	mkdir -p $(OSIMAGE_DIR)

.PHONY: osimage
osimage: DISTRO ?= ubuntu-2204
osimage: KUBERNETES_VERSION ?= v1.28.2
# KUBERNETES_SERIES default to vX.Y from KUBERNETES_VERSION
osimage: KUBERNETES_SERIES ?= $(shell echo $(KUBERNETES_VERSION) | sed 's/\(v[0-9]\+\.[0-9]\+\).[0-9]\+/\1/;tx;q1;:x')
osimage: | $(OSIMAGE_DIR)
	$(MAKE) osimage-$(DISTRO) KUBERNETES_VERSION=$(KUBERNETES_VERSION) KUBERNETES_SERIES=$(KUBERNETES_SERIES)

osimage-flatcar: | $(OSIMAGE_DIR)
	docker run --rm $(privileged) \
		--network=host \
		-v "$(OSIMAGE_DIR):/output" \
		-e PACKER_FLAGS=" \
		    --var 'accelerator=$(ACCELERATOR)' \
		    --var 'kubernetes_semver=$(KUBERNETES_VERSION)' \
		    --var 'kubernetes_series=$(KUBERNETES_SERIES)' \
		" \
		-e OEM_ID=openstack \
		$(OSIMAGE_BUILDER) build-qemu-flatcar

osimage-ubuntu: | $(OSIMAGE_DIR)
	docker run --rm $(privileged) \
		--network=host \
		-v "$(OSIMAGE_DIR):/output" \
		-e PACKER_FLAGS=" \
				--var 'iso_checksum=a4acfda10b18da50e2ec50ccaf860d7f20b389df8765611142305c0e911d16fd' \
				--var 'iso_checksum_type=sha256' \
				--var 'iso_url=https://releases.ubuntu.com/jammy/ubuntu-22.04.3-live-server-amd64.iso' \
		    --var 'accelerator=$(ACCELERATOR)' \
		    --var 'kubernetes_semver=$(KUBERNETES_VERSION)' \
		    --var 'kubernetes_series=$(KUBERNETES_SERIES)' \
		" \
		-e OEM_ID=openstack \
		$(OSIMAGE_BUILDER) build-qemu-ubuntu-2204

# Build a specific image, determined by the name of the target
# e.g. osimage-flatcar-1.26.2 -> DISTRO=flatcar, KUBERNETES_VERSION=v1.26.2
osimage-%: DISTRO=$(shell echo $@ | sed 's/osimage-\(.*\)-\([^-]*\)/\1/')
osimage-%: KUBERNETES_VERSION=v$(shell echo $@ | sed 's/osimage-\(.*\)-\([^-]*\)/\2/')
# KUBERNETES_SERIES default to vX.Y from KUBERNETES_VERSION
osimage-%: KUBERNETES_SERIES ?= $(shell echo $(KUBERNETES_VERSION) | sed 's/\(v[0-9]\+\.[0-9]\+\).[0-9]\+/\1/;tx;q1;:x')
osimage-%: FORCE
	$(MAKE) osimage-$(DISTRO) KUBERNETES_VERSION=$(KUBERNETES_VERSION) KUBERNETES_SERIES=$(KUBERNETES_SERIES)

.PHONY: FORCE
FORCE:

.PHONY: osimage-upload
osimage-upload:
	ls $(OSIMAGE_DIR)
	gsutil cp $(OSIMAGE_DIR)/* gs://$(STAGING_BUCKET)/test
