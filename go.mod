module sigs.k8s.io/cluster-api-provider-openstack

go 1.13

require (
	github.com/go-logr/logr v0.1.0
	github.com/gophercloud/gophercloud v0.8.0
	github.com/gophercloud/utils v0.0.0-20200302155035-0565566533e4
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/pkg/errors v0.9.1
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.17.9
	k8s.io/apimachinery v0.17.9
	k8s.io/client-go v0.17.9
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20200619165400-6e3d28b6ed19
	sigs.k8s.io/cluster-api v0.3.12
	sigs.k8s.io/controller-runtime v0.5.14
)
