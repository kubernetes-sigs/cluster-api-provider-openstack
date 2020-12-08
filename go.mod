module sigs.k8s.io/cluster-api-provider-openstack

go 1.13

require (
	github.com/ajeddeloh/go-json v0.0.0-20170920214419-6a2fe990e083 // indirect
	github.com/ajeddeloh/yaml v0.0.0-20170912190910-6b94386aeefd // indirect
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/coreos/container-linux-config-transpiler v0.9.0
	github.com/coreos/go-systemd v0.0.0-20190620071333-e64a0ec8b42a // indirect
	github.com/coreos/ignition v0.33.0 // indirect
	github.com/go-logr/logr v0.2.1
	github.com/gophercloud/gophercloud v0.14.0
	github.com/gophercloud/utils v0.0.0-20201203161420-f41c1768a042
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/openshift/api v0.0.0-20200901182017-7ac89ba6b971
	github.com/openshift/client-go v0.0.0-20200827190008-3062137373b5
	github.com/openshift/machine-api-operator v0.2.1-0.20201002104344-6abfb5440597
	github.com/pkg/errors v0.9.1
	github.com/vincent-petithory/dataurl v0.0.0-20160330182126-9a301d65acbb // indirect
	go4.org v0.0.0-20191010144846-132d2879e1e9 // indirect
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v0.19.0
	k8s.io/cluster-bootstrap v0.0.0-20190202014938-c9acc0c1bea2
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.3.0
	sigs.k8s.io/controller-runtime v0.6.2
	sigs.k8s.io/yaml v1.2.0
)

replace (
	sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20200929152424-eab2e087f366
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20200929220456-04e680e51d03
)
