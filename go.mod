module sigs.k8s.io/cluster-api-provider-openstack

go 1.12

require (
	github.com/ajeddeloh/go-json v0.0.0-20170920214419-6a2fe990e083 // indirect
	github.com/ajeddeloh/yaml v0.0.0-20170912190910-6b94386aeefd // indirect
	github.com/coreos/container-linux-config-transpiler v0.9.0
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/coreos/ignition v0.33.0 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/gophercloud/gophercloud v0.3.0
	github.com/gophercloud/utils v0.0.0-20190527093828-25f1b77b8c03
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pkg/errors v0.8.1
	github.com/vincent-petithory/dataurl v0.0.0-20160330182126-9a301d65acbb // indirect
	go4.org v0.0.0-20190313082347-94abd6928b1d // indirect
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190711103429-37c3b8b1ca65
	k8s.io/apimachinery v0.0.0-20190711103026-7bf792636534
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/cluster-bootstrap v0.0.0-20190711112844-b7409fb13d1b
	k8s.io/code-generator v0.0.0-20190311093542-50b561225d70
	k8s.io/gengo v0.0.0-20190813173942-955ffa8fcfc9 // indirect
	k8s.io/klog v0.4.0
	k8s.io/kubernetes v1.14.2
	k8s.io/utils v0.0.0-20190506122338-8fab8cb257d5
	sigs.k8s.io/cluster-api v0.0.0-20190821154522-636a336cc6b5
	sigs.k8s.io/controller-runtime v0.2.0-rc.0
	sigs.k8s.io/controller-tools v0.2.0-rc.0
	sigs.k8s.io/testing_frameworks v0.1.2-0.20190130140139-57f07443c2d4
)

replace (
	// Workaround for https://github.com/kubernetes/test-infra/pull/13918#issuecomment-522807578
	gomodules.xyz/jsonpatch/v2 => gomodules.xyz/jsonpatch/v2 v2.0.0-20190626003512-87910169748d
	k8s.io/api => k8s.io/api v0.0.0-20190704095032-f4ca3d3bdf1d
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190704094733-8f6ac2502e51
	sigs.k8s.io/cluster-api => sigs.k8s.io/cluster-api v0.0.0-20190821154522-636a336cc6b5
)
