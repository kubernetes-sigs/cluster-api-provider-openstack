module sigs.k8s.io/cluster-api-provider-openstack

go 1.16

require (
	github.com/go-logr/logr v0.4.0
	github.com/gophercloud/gophercloud v0.15.0
	github.com/gophercloud/utils v0.0.0-20201203161420-f41c1768a042
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.10.5
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.21.0-beta.0
	k8s.io/apimachinery v0.21.0-beta.0
	k8s.io/client-go v0.21.0-beta.0
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	sigs.k8s.io/cluster-api v0.3.11-0.20210305093021-046ab290ba3c
	sigs.k8s.io/controller-runtime v0.8.2
)
