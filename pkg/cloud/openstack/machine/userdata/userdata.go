package userdata

import (
	"encoding/base64"
	"io"
)

var (
	supportedDistributions = map[string]bool{
		"ubuntu": true,
		"centos": true,
	}
)

// UserData is an interface to provide userdata to VMs
// e.g. a cloud config, or ignition
type UserData interface {
	Write(f io.Writer, master bool, params SetupParams) error
}

// SetupParams contains all necessary information to create cloud-config files
type SetupParams struct {
	KubernetesParams
	ScriptParams

	Token string
}

// KubernetesParams contains all parameters relevant for Kubernetes
type KubernetesParams struct {
	ControlPlaneVersion  string
	ControlPlaneEndpoint string
	KubeletVersion       string
	PodCIDR              string
	ServiceCIDR          string
	KubeadmConfig        string
}

// ScriptParams contains all parametes needed in the Bootstrap Script
type ScriptParams struct {
	Namespace        string
	Name             string
	BootstrapScript  string
	BootstrapService string
}

// DefaultScriptParams sets ScriptParams to defaults if needed
func DefaultScriptParams(params *ScriptParams) {
	if params.BootstrapScript == "" {
		params.BootstrapScript = base64.StdEncoding.EncodeToString([]byte(bootstrapScript))
	}
	if params.BootstrapService == "" {
		params.BootstrapService = base64.StdEncoding.EncodeToString([]byte(bootstrapService))
	}
}

// WorkerParams contains all parameters needed on Worker Nodes
type WorkerParams struct {
	ControlPlaneEndpoint string
	Token                string
	KubeadmConfig        string
}

// IsSupported returns true if the passed in distribution is supported by the current
// implementation of userdata
func IsSupported(distri string) bool {
	_, ok := supportedDistributions[distri]
	return ok
}

// GetSupported returns a list of supported distributions
func GetSupported() []string {
	s := []string{}
	for k := range supportedDistributions {
		s = append(s, k)
	}
	return s
}
