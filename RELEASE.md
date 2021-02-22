
# Releasing

## Output

### Expected artifacts

1. A container image of the `cluster-api-provider-openstack` controller manager

### Artifact locations

1. The container image is found in the registry `us.gcr.io/k8s-artifacts-prod/capi-openstack/` with an image
   name of `capi-openstack-controller` and a tag that matches the release version. For
   example, in the `v0.2.0` release, the container image location is
   `us.gcr.io/k8s-artifacts-prod/capi-openstack/capi-openstack-controller:v0.2.0`


## Process


1. Make sure your repo is clean by git's standards
2. If this is a new minor release, create a new release branch and push to github, otherwise switch to it, for example `release-0.2`
3. Run `make release-notes` to gather changes since the last revision. If you need to specify a specific tag to look for changes
   since, use `make release-notes ARGS="--from <tag>"` Pay close attention to the `## :question: Sort these by hand` section, as it contains items that need to be manually sorted.
4. Tag the repository and push the tag `git tag -s -m $VERSION $VERSION; git push upstream $VERSION`
5. Create a draft release in github and associate it with the tag that was just created, copying the generated release notes into
   the draft.
6. Checkout the tag you've just created and make sure git is in a clean state
7. Run `make release`
8. Attach the files to the drafted release:
    1. `./out/infrastructure-components.yaml`
    2. `./templates/cluster-template.yaml`
    3. `./templates/cluster-template-without-lb.yaml`
    4. `./templates/cluster-template-external-cloud-provider.yaml`
    5. `./metadata.yaml` (clusterctl >0.3.1 will include hardcoded metadata for CAPO. But let's keep the `metadata.yaml` file for our v0.3.* releases to be compatible with clusterctl <=0.3.1)
9.  Perform the [image promotion process](https://github.com/kubernetes/k8s.io/tree/main/k8s.gcr.io#image-promoter).
    The staging repository is at https://console.cloud.google.com/gcr/images/k8s-staging-capi-openstack/GLOBAL. Be
    sure to choose the top level `capi-openstack-controller`, which will provide the multi-arch manifest, rather than one for a specific architecture.
    The image build logs are available in [Cloud Build](https://console.cloud.google.com/cloud-build/builds?project=k8s-staging-capi-openstack).
    Add the new sha=>tag mapping to the [images.yaml](https://github.com/kubernetes/k8s.io/edit/main/k8s.gcr.io/images/k8s-staging-capi-openstack/images.yaml) (use the sha of the image with the corresponding tag)
10.  Finalise the release notes
11.  Publish release. Use the pre-release option for release
    candidate versions of Cluster API Provider OpenStack.

### Permissions

Releasing requires a particular set of permissions.

* Approver role for the image promoter process ([kubernetes/k8s.io/blob/master/k8s.gcr.io/images/k8s-staging-capi-openstack/OWNERS](https://github.com/kubernetes/k8s.io/blob/master/k8s.gcr.io/images/k8s-staging-capi-openstack/OWNERS))
* Tag push and release creation rights to the GitHub repository (team `cluster-api-provider-openstack-maintainers` in [kubernetes/org/config/kubernetes-sigs/sig-cluster-lifecycle/teams.yaml](https://github.com/kubernetes/org/blob/master/config/kubernetes-sigs/sig-cluster-lifecycle/teams.yaml))

## Staging

There is a post-submit Prow job running after each commit on master which pushes a new image to the staging repo (`gcr.io/k8s-staging-capi-openstack/capi-openstack-controller:latest`). Following configuration is involved:
* staging gcr bucket: [kubernetes/k8s.io/blob/master/k8s.gcr.io/manifests/k8s-staging-capi-openstack/promoter-manifest.yaml](https://github.com/kubernetes/k8s.io/blob/master/k8s.gcr.io/manifests/k8s-staging-capi-openstack/promoter-manifest.yaml)
* post-submit `post-capi-openstack-push-images` Prow job: [kubernetes/test-infra/blob/master/config/jobs/image-pushing/k8s-staging-cluster-api.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/image-pushing/k8s-staging-cluster-api.yaml)) (corresponding dashboard is located at [https://testgrid.k8s.io/sig-cluster-lifecycle-image-pushes#post-capi-openstack-push-images](https://testgrid.k8s.io/sig-cluster-lifecycle-image-pushes#post-capi-openstack-push-images))
* Google Cloud Build configuration which is used by the Prow job: [kubernetes-sigs/cluster-api-provider-openstack/cloudbuild.yaml](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/master/cloudbuild.yaml)
