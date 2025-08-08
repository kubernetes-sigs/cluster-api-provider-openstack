---
name: New release
about: "[Only for maintainers] Create an issue to track release activities"
title: Tasks for v<release-tag> release cycle

---

## Tasks

Tasks for a new release `vX.Y.Z` of the Cluster API Provider OpenStack.
For details, see [RELEASE.md](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/main/RELEASE.md).

- [ ] [When bumping `X` or `Y`] Create a new release branch called `release-X.Y`.
- [ ] [When bumping `X` or `Y`] Add a new entry to [metadata.yaml](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/main/metadata.yaml)
  as [described in the CAPI book](https://cluster-api.sigs.k8s.io/clusterctl/provider-contract.html#metadata-yaml)
  on the release branch prior to release.
- [ ] Push tag to the repository.
- [ ] Promote the [staging image](https://console.cloud.google.com/cloud-build/builds?project=k8s-staging-capi-openstack) by
  adding the new sha=>tag mapping to [images.yaml](https://github.com/kubernetes/k8s.io/blob/main/registry.k8s.io/images/k8s-staging-capi-openstack/images.yaml).
- [ ] Verify that the new draft release looks good and make changes if necessary.
- [ ] Verify that the image was promoted sucessfully.
- [ ] Publish the release.
  Mark the release as "latest" if it is the most recent minor release.
  E.g. if both v1.1 and v1.2 are supported with patch releases, then only v1.2.z should be marked as "latest".

## Post-release tasks

- [ ] [When bumping `X` or `Y`] Update the [periodic jobs](https://github.com/kubernetes/test-infra/tree/master/config/jobs/kubernetes-sigs/cluster-api-provider-openstack).
  Make sure there are periodic jobs for the new release branch, and clean up jobs for branches that are no longer supported.
- [ ] [When bumping `X` or `Y`] Update the [clusterctl upgrade tests](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/main/test/e2e/suites/e2e/clusterctl_upgrade_test.go)
  and the [e2e config](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/main/test/e2e/data/e2e_conf.yaml)
  to include the new release branch.
  It is also a good idea to update the Cluster API versions we test against and to clean up older versions that we no longer want
  to test.
