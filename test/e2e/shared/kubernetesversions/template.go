/*
Copyright 2021 The Kubernetes Authors.

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

package kubernetesversions

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/yaml"
)

var (
	//go:embed data/debian_injection_script_control_plane.envsubst.sh
	debianInjectionScriptControlPlaneBytes string

	//go:embed data/debian_injection_script_worker.envsubst.sh
	debianInjectionScriptWorkerBytes string

	//go:embed data/kustomization.yaml.tpl
	kustomizationYAMLBytes string

	kustomizationTemplate = template.Must(template.New("kustomization").Parse(kustomizationYAMLBytes))
)

type GenerateCIArtifactsInjectedTemplateForDebianInput struct {
	// ArtifactsDirectory is where conformance suite output will go. Defaults to _artifacts
	ArtifactsDirectory string
	// SourceTemplate is an input YAML clusterctl template which is to have
	// the CI artifact script injection
	SourceTemplate []byte
	// PlatformKustomization is an SMP (strategic-merge-style) patch for adding
	// platform specific kustomizations required for use with CI, such as
	// referencing a specific image
	PlatformKustomization []byte
	// KubeadmConfigTemplateName is the name of the KubeadmConfigTemplate resource
	// that needs to have the Debian install script injected. Defaults to "${CLUSTER_NAME}-md-0".
	KubeadmConfigTemplateName string
	// KubeadmControlPlaneName is the name of the KubeadmControlPlane resource
	// that needs to have the Debian install script injected. Defaults to "${CLUSTER_NAME}-control-plane".
	KubeadmControlPlaneName string
	// KubeadmConfigName is the name of a KubeadmConfig that needs kustomizing. To be used in conjunction with MachinePools. Optional.
	KubeadmConfigName string
}

// GenerateCIArtifactsInjectedTemplateForDebian takes a source clusterctl template
// and a platform-specific Kustomize SMP patch and injects a bash script to download
// and install the debian packages for the given Kubernetes version, returning the
// location of the outputted file.
func GenerateCIArtifactsInjectedTemplateForDebian(input GenerateCIArtifactsInjectedTemplateForDebianInput) (string, error) {
	if input.SourceTemplate == nil {
		return "", errors.New("SourceTemplate must be provided")
	}
	input.ArtifactsDirectory = framework.ResolveArtifactsDirectory(input.ArtifactsDirectory)
	if input.KubeadmConfigTemplateName == "" {
		input.KubeadmConfigTemplateName = "${CLUSTER_NAME}-md-0"
	}
	if input.KubeadmControlPlaneName == "" {
		input.KubeadmControlPlaneName = "${CLUSTER_NAME}-control-plane"
	}
	templateDir := path.Join(input.ArtifactsDirectory, "templates")
	overlayDir := path.Join(input.ArtifactsDirectory, "overlay")

	if err := os.MkdirAll(templateDir, 0o750); err != nil {
		return "", err
	}
	if err := os.MkdirAll(overlayDir, 0o750); err != nil {
		return "", err
	}

	kustomizedTemplate := path.Join(templateDir, "cluster-template-conformance-ci-artifacts.yaml")

	kustomizationYamlBytes, err := generateKustomizationYAML(input)
	if err != nil {
		return "", err
	}

	if err := ioutil.WriteFile(path.Join(overlayDir, "kustomization.yaml"), kustomizationYamlBytes, 0o600); err != nil {
		return "", err
	}

	patch, err := generateDebianInjectionScriptJSONPatch(input.SourceTemplate, "KubeadmControlPlane", input.KubeadmControlPlaneName, "/spec/kubeadmConfigSpec", debianInjectionScriptControlPlaneBytes)
	if err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(path.Join(overlayDir, "kubeadmcontrolplane-patch.yaml"), patch, 0o600); err != nil {
		return "", err
	}

	patch, err = generateDebianInjectionScriptJSONPatch(input.SourceTemplate, "KubeadmConfigTemplate", input.KubeadmConfigTemplateName, "/spec/template/spec", debianInjectionScriptWorkerBytes)
	if err != nil {
		return "", err
	}

	if err := ioutil.WriteFile(path.Join(overlayDir, "kubeadmconfigtemplate-patch.yaml"), patch, 0o600); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(path.Join(overlayDir, "ci-artifacts-source-template.yaml"), input.SourceTemplate, 0o600); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(path.Join(overlayDir, "platform-kustomization.yaml"), input.PlatformKustomization, 0o600); err != nil {
		return "", err
	}
	cmd := exec.Command("kustomize", "build", overlayDir)
	data, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(kustomizedTemplate, data, 0o600); err != nil {
		return "", err
	}
	return kustomizedTemplate, nil
}

func generateKustomizationYAML(input GenerateCIArtifactsInjectedTemplateForDebianInput) ([]byte, error) {
	var kustomizationYamlBytes bytes.Buffer
	if err := kustomizationTemplate.Execute(&kustomizationYamlBytes, input); err != nil {
		return nil, err
	}
	return kustomizationYamlBytes.Bytes(), nil
}

type jsonPatch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

func generateDebianInjectionScriptJSONPatch(sourceTemplate []byte, kind, name, path, content string) ([]byte, error) {
	filesPathExists, preKubeadmCommandsPathExists, err := checkExistingPaths(sourceTemplate, kind, name, path)
	if err != nil {
		return nil, err
	}

	var patches []jsonPatch
	if !filesPathExists {
		patches = append(patches, jsonPatch{
			Op:    "add",
			Path:  fmt.Sprintf("%s/files", path),
			Value: []interface{}{},
		})
	}
	patches = append(patches, jsonPatch{
		Op:   "add",
		Path: fmt.Sprintf("%s/files/0", path),
		Value: map[string]string{
			"content":     content,
			"owner":       "root:root",
			"path":        "/usr/local/bin/ci-artifacts.sh",
			"permissions": "0750",
		},
	})
	if !preKubeadmCommandsPathExists {
		patches = append(patches, jsonPatch{
			Op:    "add",
			Path:  fmt.Sprintf("%s/preKubeadmCommands", path),
			Value: []string{},
		})
	}
	patches = append(patches, jsonPatch{
		Op:    "add",
		Path:  fmt.Sprintf("%s/preKubeadmCommands/0", path),
		Value: "/usr/local/bin/ci-artifacts.sh",
	})

	return yaml.Marshal(patches)
}

func checkExistingPaths(sourceTemplate []byte, kind, name, path string) (bool, bool, error) {
	yamlDocs := strings.Split(string(sourceTemplate), "---")
	for _, yamlDoc := range yamlDocs {
		if yamlDoc == "" {
			continue
		}
		var obj unstructured.Unstructured
		if err := yaml.Unmarshal([]byte(yamlDoc), &obj); err != nil {
			return false, false, err
		}

		if obj.GetKind() != kind {
			continue
		}
		if obj.GetName() != name {
			continue
		}

		pathSplit := strings.Split(strings.TrimPrefix(path, "/"), "/")
		filesPath := append(pathSplit, "files")
		preKubeadmCommandsPath := append(pathSplit, "preKubeadmCommands")
		_, filesPathExists, err := unstructured.NestedFieldCopy(obj.Object, filesPath...)
		if err != nil {
			return false, false, err
		}
		_, preKubeadmCommandsPathExists, err := unstructured.NestedFieldCopy(obj.Object, preKubeadmCommandsPath...)
		if err != nil {
			return false, false, err
		}
		return filesPathExists, preKubeadmCommandsPathExists, nil
	}
	return false, false, fmt.Errorf("could not find document with kind %q and name %q", kind, name)
}
