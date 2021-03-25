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

// TODO(sbuerin): should be removed after: https://github.com/kubernetes-sigs/kustomize/issues/2825 is fixed
// copied from: cluster-api/test/framework/kubernetesversions/template.go
// currently we have to skip the CAPI generated debian patch because it would overwrite our files

package shared

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path"

	"sigs.k8s.io/cluster-api/test/framework"
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

	kustomization := []byte(`
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: default
resources:
  - ci-artifacts-source-template.yaml
patchesStrategicMerge:
  - platform-kustomization.yaml
`)

	if err := ioutil.WriteFile(path.Join(overlayDir, "kustomization.yaml"), kustomization, 0o600); err != nil {
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
