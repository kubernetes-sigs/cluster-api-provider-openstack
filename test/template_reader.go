/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package test

import (
	"bytes"
	"embed"
	"encoding/base64"
	"fmt"
	"path"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

type TemplateVars struct {
	ClusterName                        string `template:"CLUSTER_NAME"`
	CNIResources                       string `template:"CNI_RESOURCES"`
	ControlPlaneMachineCount           string `template:"CONTROL_PLANE_MACHINE_COUNT"`
	KubernetesVersion                  string `template:"KUBERNETES_VERSION"`
	OpenStackBastionImageName          string `template:"OPENSTACK_BASTION_IMAGE_NAME"`
	OpenStackBastionMachineFlavor      string `template:"OPENSTACK_BASTION_MACHINE_FLAVOR"`
	OpenStackCloud                     string `template:"OPENSTACK_CLOUD"`
	OpenStackCloudCACert               string `template:"OPENSTACK_CLOUD_CACERT_B64"`
	OpenStackCloudProviderConf         string `template:"OPENSTACK_CLOUD_PROVIDER_CONF_B64"`
	OpenStackCloudYAML                 string `template:"OPENSTACK_CLOUD_YAML_B64"`
	OpenStackControlPlaneMachineFlavor string `template:"OPENSTACK_CONTROL_PLANE_MACHINE_FLAVOR"`
	OpenStackDNSNameservers            string `template:"OPENSTACK_DNS_NAMESERVERS"`
	OpenStackExternalNetworkID         string `template:"OPENSTACK_EXTERNAL_NETWORK_ID"`
	OpenStackFailureDomain             string `template:"OPENSTACK_FAILURE_DOMAIN"`
	OpenStackImageName                 string `template:"OPENSTACK_IMAGE_NAME"`
	OpenStackNodeMachineFlavor         string `template:"OPENSTACK_NODE_MACHINE_FLAVOR"`
	OpenStackSSHKeyName                string `template:"OPENSTACK_SSH_KEY_NAME"`
	WorkerMachineCount                 string `template:"WORKER_MACHINE_COUNT"`
}

//go:embed e2e/data/infrastructure-openstack/*.yaml
var fs embed.FS

func ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(path.Join("e2e/data/infrastructure-openstack", name))
}

func Substitute(b []byte, vars TemplateVars) []byte {
	s := string(b)

	t := reflect.TypeOf(vars)
	v := reflect.ValueOf(vars)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("template")
		value := v.Field(i).String()

		if strings.HasSuffix(tag, "_B64") {
			b64 := base64.StdEncoding.EncodeToString([]byte(value))
			value = fmt.Sprintf("%q", b64)
		}

		s = strings.Replace(s, fmt.Sprintf("${%s}", tag), value, -1)
	}

	return []byte(s)
}

func SplitYAML(b []byte) [][]byte {
	const separator = "\n---"

	ret := [][]byte{}
	pos := 0
	for {
		i := bytes.Index(b[pos:], []byte(separator))
		if i > 0 {
			ret = append(ret, b[pos:pos+i])
			pos += i + len(separator)
		} else {
			return append(ret, b[pos:])
		}
	}
}

func ReadObject(b []byte, scheme *runtime.Scheme) (runtime.Object, error) {
	codecs := serializer.NewCodecFactory(scheme)
	decoder := codecs.UniversalDeserializer()

	obj, _, err := decoder.Decode(b, nil, nil)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func ReadTemplatedObjects(name string, vars TemplateVars, scheme *runtime.Scheme) ([]runtime.Object, error) {
	b, err := ReadFile(name)
	if err != nil {
		return nil, err
	}

	templated := Substitute(b, vars)
	yamls := SplitYAML(templated)

	objs := []runtime.Object{}
	for _, yaml := range yamls {
		o, err := ReadObject(yaml, scheme)
		if err != nil {
			return nil, err
		}

		objs = append(objs, o)
	}

	return objs, nil
}
