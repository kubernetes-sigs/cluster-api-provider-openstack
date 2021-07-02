
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


1. Make sure your repo is clean by git's standards.
1. Make sure you are on the correct branch (`master` for the current release and `release-0.x` for older releases).
1. Create an annotated tag
    - `git tag -s -a $VERSION -m $VERSION`.
1. Push the tag to the GitHub repository:
   > NOTE: `upstream` should be the name of the remote pointing to `github.com/kubernetes-sigs/cluster-api-provider-openstack`
    - `git push upstream $VERSION`
1. Run `make release` to build artifacts (the image is automatically built by CI)
1. Follow the [image promotion process](https://github.com/kubernetes/k8s.io/tree/main/k8s.gcr.io#image-promoter) to promote the image from the staging repo to `k8s.gcr.io/capi-openstack`.
   The staging repository can be inspected at https://console.cloud.google.com/gcr/images/k8s-staging-capi-openstack/GLOBAL. Be
   sure to choose the top level `capi-openstack-controller`, which will provide the multi-arch manifest, rather than one for a specific architecture.
   The image build logs are available at [Cloud Build](https://console.cloud.google.com/cloud-build/builds?project=k8s-staging-capi-openstack).
   Add the new sha=>tag mapping to the [images.yaml](https://github.com/kubernetes/k8s.io/edit/main/k8s.gcr.io/images/k8s-staging-capi-openstack/images.yaml) (use the sha of the image with the corresponding tag)
1. Create a draft release in GitHub based on the tag created above
1. Generate and finalize the release notes and add them to the draft release:
    - Run `make release-notes` to gather changes since the last revision. If you need to specify a specific tag to look for changes
      since, use `make release-notes RELEASE_NOTES_ARGS="--from <tag>"`.
    - Pay close attention to the `## :question: Sort these by hand` section, as it contains items that need to be manually sorted.
1. Attach the following files to the draft release:
    - `./out/infrastructure-components.yaml`
    - `./out/cluster-template.yaml`
    - `./out/cluster-template-external-cloud-provider.yaml`
    - `./out/cluster-template-without-lb.yaml`
    - `./out/metadata.yaml`
1.  Publish release. Use the pre-release option for release candidate or beta versions.

### Permissions

Releasing requires a particular set of permissions.

* Approver role for the image promoter process ([kubernetes/k8s.io/blob/main/k8s.gcr.io/images/k8s-staging-capi-openstack/OWNERS](https://github.com/kubernetes/k8s.io/blob/main/k8s.gcr.io/images/k8s-staging-capi-openstack/OWNERS))
* Tag push and release creation rights to the GitHub repository (team `cluster-api-provider-openstack-maintainers` in [kubernetes/org/config/kubernetes-sigs/sig-cluster-lifecycle/teams.yaml](https://github.com/kubernetes/org/blob/master/config/kubernetes-sigs/sig-cluster-lifecycle/teams.yaml))

## Staging

There is a post-submit Prow job running after each commit on master which pushes a new image to the staging repo (`gcr.io/k8s-staging-capi-openstack/capi-openstack-controller:latest`). Following configuration is involved:
* staging gcr bucket: [kubernetes/k8s.io/blob/main/k8s.gcr.io/manifests/k8s-staging-capi-openstack/promoter-manifest.yaml](https://github.com/kubernetes/k8s.io/blob/main/k8s.gcr.io/manifests/k8s-staging-capi-openstack/promoter-manifest.yaml)
* post-submit `post-capi-openstack-push-images` Prow job: [kubernetes/test-infra/blob/master/config/jobs/image-pushing/k8s-staging-cluster-api.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/image-pushing/k8s-staging-cluster-api.yaml)) (corresponding dashboard is located at [https://testgrid.k8s.io/sig-cluster-lifecycle-image-pushes#post-capi-openstack-push-images](https://testgrid.k8s.io/sig-cluster-lifecycle-image-pushes#post-capi-openstack-push-images))
* Google Cloud Build configuration which is used by the Prow job: [kubernetes-sigs/cluster-api-provider-openstack/cloudbuild.yaml](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/master/cloudbuild.yaml)
