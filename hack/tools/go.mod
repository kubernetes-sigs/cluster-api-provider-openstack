module sigs.k8s.io/cluster-api-provider-openstack/hack/tools

go 1.16

require (
	github.com/a8m/envsubst v1.2.0
	github.com/golang/mock v1.4.4
	github.com/golangci/golangci-lint v1.27.0
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/onsi/ginkgo v1.15.2
	golang.org/x/sys v0.0.0-20210113181707-4bcb84eeeb78 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/code-generator v0.21.0-beta.0
	sigs.k8s.io/cluster-api v0.3.11-0.20210310224224-a9144a861bf4
	sigs.k8s.io/cluster-api/hack/tools v0.0.0-20210313163703-752c727cf58b
	sigs.k8s.io/controller-tools v0.5.0
	sigs.k8s.io/kind v0.9.0
	sigs.k8s.io/kustomize/kustomize/v3 v3.8.6
	sigs.k8s.io/testing_frameworks v0.1.2
)

// pin for now to avoid fixing all the linter issues in the current PR
// TODO(sbueringer): upgrade to current linter and fix the occuring issues
replace github.com/golangci/golangci-lint => github.com/golangci/golangci-lint v1.23.8
