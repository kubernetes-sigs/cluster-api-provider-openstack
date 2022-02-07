module sigs.k8s.io/cluster-api-provider-openstack

go 1.16

require (
	github.com/go-logr/logr v0.4.0
	github.com/golang/mock v1.6.0
	github.com/google/gofuzz v1.2.0
	github.com/gophercloud/gophercloud v0.16.0
	github.com/gophercloud/utils v0.0.0-20210323225332-7b186010c04f
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	gopkg.in/ini.v1 v1.63.2
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	k8s.io/component-base v0.22.2
	k8s.io/klog/v2 v2.9.0
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b
	sigs.k8s.io/cluster-api v1.0.4
	// cluster-api/test by commit to pull in kubernetes-sigs/cluster-api#6080
	sigs.k8s.io/cluster-api/test v1.0.5-0.20220209113217-e5b9f43cba6b
	sigs.k8s.io/controller-runtime v0.10.3
	sigs.k8s.io/yaml v1.3.0
)

replace sigs.k8s.io/cluster-api => sigs.k8s.io/cluster-api v1.0.4
