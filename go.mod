module sigs.k8s.io/cluster-api-provider-openstack

go 1.12

require (
	cloud.google.com/go v0.40.0 // indirect
	github.com/Azure/go-autorest/autorest v0.5.0 // indirect
	github.com/ajeddeloh/go-json v0.0.0-20170920214419-6a2fe990e083 // indirect
	github.com/ajeddeloh/yaml v0.0.0-20170912190910-6b94386aeefd // indirect
	github.com/appscode/jsonpatch v0.0.0-20190108182946-7c0e3b262f30 // indirect
	github.com/coreos/container-linux-config-transpiler v0.9.0
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20190321100706-95778dfbb74e // indirect
	github.com/coreos/ignition v0.32.0 // indirect
	github.com/go-logr/logr v0.1.0 // indirect
	github.com/go-logr/zapr v0.1.1 // indirect
	github.com/gobuffalo/envy v1.7.0 // indirect
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/gofuzz v1.0.0 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/googleapis/gnostic v0.3.0 // indirect
	github.com/gophercloud/gophercloud v0.3.0
	github.com/gophercloud/utils v0.0.0-20190527093828-25f1b77b8c03
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/markbates/inflect v1.0.4 // indirect
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.4 // indirect
	github.com/rogpeppe/go-internal v1.3.0 // indirect
	github.com/spf13/cobra v0.0.5 // indirect
	github.com/vincent-petithory/dataurl v0.0.0-20160330182126-9a301d65acbb // indirect
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.10.0 // indirect
	go4.org v0.0.0-20190313082347-94abd6928b1d // indirect
	golang.org/x/crypto v0.0.0-20190611184440-5c40567a22f8 // indirect
	golang.org/x/net v0.0.0-20190613194153-d28f0bde5980 // indirect
	golang.org/x/sys v0.0.0-20190616124812-15dcb6c0061f // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	golang.org/x/tools v0.0.0-20190614205625-5aca471b1d59 // indirect
	google.golang.org/appengine v1.6.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0
	k8s.io/apiextensions-apiserver v0.0.0-00010101000000-000000000000 // indirect
	k8s.io/apimachinery v0.0.0
	k8s.io/client-go v0.0.0
	k8s.io/cluster-bootstrap v0.0.0
	k8s.io/code-generator v0.0.0
	k8s.io/component-base v0.0.0-00010101000000-000000000000 // indirect
	k8s.io/gengo v0.0.0-20190327210449-e17681d19d3a // indirect
	k8s.io/klog v0.3.3
	k8s.io/kube-openapi v0.0.0-20190603182131-db7b694dc208 // indirect
	k8s.io/kubernetes v1.13.4
	k8s.io/utils v0.0.0-20190801114015-581e00157fb1 // indirect
	sigs.k8s.io/cluster-api v0.1.4
	sigs.k8s.io/controller-runtime v0.1.12
	sigs.k8s.io/controller-tools v0.1.11
	sigs.k8s.io/testing_frameworks v0.1.1
	sigs.k8s.io/yaml v1.1.0
)

replace (
	k8s.io/api => k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190228180357-d002e88f6236
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190228174905-79427f02047f
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190228180923-a9e421a79326
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190228174230-b40b2a5939e4
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190228181926-f531db154915
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20190228181738-e96ff33745e4
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20181117043124-c2090bec4d9b
	k8s.io/component-base => k8s.io/component-base v0.0.0-20190620085130-185d68e6e6ea
	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190531030430-6117653b35f1
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20190620090116-299a7b270edc
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190228175259-3e0149950b0e
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20190228181625-2d4c0aa6bcf3
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20190228181220-93400f02525d
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20190228181501-284499a24813
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20190228181339-b2baec5a487a
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20190620090156-2138f2c9de18
	k8s.io/metrics => k8s.io/metrics v0.0.0-20190228180609-34472d076c30
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20190228175645-7167784fdc46
)
