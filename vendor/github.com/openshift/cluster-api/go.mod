module github.com/openshift/cluster-api

go 1.12

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/go-log/log v0.0.0-20181211034820-a514cf01a3eb
	github.com/golang/groupcache v0.0.0-20181024230925-c65c006176ff // indirect
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_model v0.0.0-20190115171406-56726106282f // indirect
	github.com/prometheus/common v0.1.0 // indirect
	github.com/prometheus/procfs v0.0.0-20190117184657-bf6a532e95b1 // indirect
	github.com/spf13/pflag v1.0.3
	golang.org/x/net v0.0.0-20190812203447-cdfb69ac37fc
	golang.org/x/sys v0.0.0-20190813064441-fde4db37ae7a // indirect
	k8s.io/api v0.0.0-20190918155943-95b840bb6a1f
	k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655
	k8s.io/apiserver v0.0.0-20190918160949-bfa5e2e684ad
	k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90
	k8s.io/code-generator v0.0.0-20190912054826-cd179ad6a269
	k8s.io/klog v0.4.0
	k8s.io/utils v0.0.0-20190923111123-69764acb6e8e
	sigs.k8s.io/controller-runtime v0.2.0-beta.2
	sigs.k8s.io/controller-tools v0.2.2-0.20190919191502-76a25b63325a
)

replace sigs.k8s.io/controller-runtime => github.com/enxebre/controller-runtime v0.2.0-beta.1.0.20191008062909-08ccc576643f
