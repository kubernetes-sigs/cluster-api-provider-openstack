# Fetch the OpenStack CCM daemonset manifest corresponding to the current k8s
# release
#
# cloud-provider-openstack versions track k8s versions except that they have a
# major version of 1 instead of 0
#
# Here we get the go module version of (arbitrarily) `k8s.io/api`, and
# substitute its major version with a 1 to get the cloud-provider-openstack
# version
CPO_VERSION=$(shell go list -f '{{ .Module.Version }}' k8s.io/api | sed 's/^v0\./v1\./')

MANIFESTS = openstack-cloud-controller-manager-ds.yaml \
            cloud-controller-manager-roles.yaml \
            cloud-controller-manager-role-bindings.yaml

.PHONY: manifests $(MANIFESTS)
manifests: $(MANIFESTS)

$(MANIFESTS):
	curl -o $@ https://raw.githubusercontent.com/kubernetes/cloud-provider-openstack/$(CPO_VERSION)/manifests/controller-manager/$@
