This helper module allows us to:
1. Avoid adding openapi generator dependencies to the main CAPO module
1. Import a specific k8s.io/code-generator commit without messing up the main
   CAPO module or tools dependencies

It may be possible to simplify this configuration in the future when CAPO is
using at least k/k v0.31.

We are very specifically pulling:
```
k8s.io/code-generator 030791bd8d60de2141f3b7f57c751787ee468ac9
```

This commit contains a fix to openapi-gen which prevents it running when
imported as a module. Later commits pull in changes which prevent the generated
applyconfiguration from building against k/k v0.29, so we can't pull those in
yet.

Do not bump the version of code-generator from this specific commit until we
also bump CAPO to k/k v0.31. At this point we should:
* Delete cmd/magnet from this directory
* Run go mod tidy

With these changes, the go.work in this module will result in us pulling the
version of code-generator corresponding to the version of k/k used by the main
CAPO module.
