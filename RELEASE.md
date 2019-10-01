
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

For version v0.x.y:

1. An issue is proposing a new release with a changelog since the last release
1. Create an annotated tag `git tag -a v0.x.y -m v0.x.y`
    1. To use your GPG signature when pushing the tag, use `git tag -s [...]` instead
1. Push the tag to the GitHub repository `git push origin v0.x.y`
    1. Note: `origin` should be the name of the remote pointing to `github.com/kubernetes-sigs/cluster-api-provider-openstack`
1. Run `make release` to build artifacts and push the images to the staging bucket
1. Follow the [Image Promotion process](https://github.com/kubernetes/k8s.io/tree/master/k8s.gcr.io#image-promoter) to promote the image from the staging repo to `us.gcr.io/k8s-artifacts-prod/capi-openstack`
1. Create a release (with the above mentioned release notes) in GitHub based on the tag created above
1. The release issue is closed
1. An announcement email is sent to `kubernetes-dev@googlegroups.com` with the subject `[ANNOUNCE] cluster-api-provider-openstack $VERSION is released`

<!-- TODO add link to image promote PR after the first release -->

### Permissions

Releasing requires a particular set of permissions.

* Push access to the staging gcr bucket ([kubernetes/k8s.io/k8s.gcr.io/k8s-staging-capi-openstack/OWNERS](https://github.com/kubernetes/k8s.io/blob/master/k8s.gcr.io/k8s-staging-capi-openstack/OWNERS)
* Tag push access to the GitHub repository ([kubernetes/org/config/kubernetes-sigs/sig-cluster-lifecycle/teams.yaml](https://github.com/kubernetes/org/blob/master/config/kubernetes-sigs/sig-cluster-lifecycle/teams.yaml#L136-L137))
* GitHub release creation access ([kubernetes/org/config/kubernetes-sigs/sig-cluster-lifecycle/teams.yaml](https://github.com/kubernetes/org/blob/master/config/kubernetes-sigs/sig-cluster-lifecycle/teams.yaml#L136-L137))

## Staging

There is a post-submit Prow job running after each commit on master which pushes a new image to the staging repo (`gcr.io/k8s-staging-capi-openstack/capi-openstack-controller:latest`). Following configuration is involved:
* staging gcr bucket: [kubernetes/k8s.io/k8s.gcr.io/k8s-staging-capi-openstack/manifest.yaml](https://github.com/kubernetes/k8s.io/blob/master/k8s.gcr.io/k8s-staging-capi-openstack/manifest.yaml)
* post-submit `post-capi-openstack-push-images` Prow job: [kubernetes/test-infra/config/jobs/image-pushing/k8s-staging-capi-openstack.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/image-pushing/k8s-staging-capi-openstack.yaml)) (corresponding dashboard is located at [https://testgrid.k8s.io/sig-cluster-lifecycle-image-pushes#post-capi-openstack-push-images](https://testgrid.k8s.io/sig-cluster-lifecycle-image-pushes#post-capi-openstack-push-images))
* Google Cloud Build configuration which is used by the Prow job: [kubernetes-sigs/cluster-api-provider-openstack/cloudbuild.yaml](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/master/cloudbuild.yaml)
