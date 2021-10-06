module sigs.k8s.io/cluster-api-provider-openstack

go 1.16

require (
	github.com/go-logr/logr v0.4.0
	github.com/golang/mock v1.6.0
	github.com/gophercloud/gophercloud v0.16.0
	github.com/gophercloud/utils v0.0.0-20210323225332-7b186010c04f
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	gopkg.in/ini.v1 v1.62.0
	k8s.io/api v0.21.4
	k8s.io/apimachinery v0.21.4
	k8s.io/client-go v0.21.4
	k8s.io/component-base v0.21.4
	k8s.io/klog/v2 v2.9.0
	k8s.io/utils v0.0.0-20210802155522-efc7438f0176
	sigs.k8s.io/cluster-api v0.4.3
	sigs.k8s.io/cluster-api/test v0.4.3
	sigs.k8s.io/controller-runtime v0.9.7
	sigs.k8s.io/yaml v1.2.0
)

replace sigs.k8s.io/cluster-api => sigs.k8s.io/cluster-api v0.4.3
