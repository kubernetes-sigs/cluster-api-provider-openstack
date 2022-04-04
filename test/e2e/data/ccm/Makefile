# The version of cloud-provider-openstack to install
# This should correspond to the KUBERNETES_VERSION defined in
# test/e2e/data/e2e_conf
CPO_VERSION := v1.23.0

# MANIFESTS must also be listed as resources in kustomization.yaml
MANIFESTS = openstack-cloud-controller-manager-ds.yaml \
	    cloud-controller-manager-role-bindings.yaml \
	    cloud-controller-manager-roles.yaml

.PHONY: manifests $(MANIFESTS)
manifests: $(MANIFESTS)

$(MANIFESTS):
	curl -o $@ https://raw.githubusercontent.com/kubernetes/cloud-provider-openstack/$(CPO_VERSION)/manifests/controller-manager/$@
