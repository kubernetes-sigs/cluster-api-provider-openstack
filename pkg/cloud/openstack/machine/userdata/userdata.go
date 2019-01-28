package userdata

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/textproto"
	"text/template"
)

var (
	supportedDistributions = map[string]bool{
		"ubuntu": true,
		"centos": true,
	}
)

// SetupParams contains all necessary information to create cloud-config files
type SetupParams struct {
	KubernetesParams
	ScriptParams

	Token string
}

// WriteCloudConfig writes a cloud config for a Kubernetes Node into an io.Writer (f)
func WriteCloudConfig(f io.Writer, distribution string, master bool, params SetupParams) error {
	w := multipart.NewWriter(f)
	defer w.Close()

	fmt.Fprintf(f, "Content-Type: multipart/mixed; boundary=\"%s\"\n", w.Boundary())
	fmt.Fprintf(f, "MIME-Version: 1.0\n\n")

	switch distribution {
	case "ubuntu":
		dist, err := NewUbuntu(params.KubeletVersion)
		if err != nil {
			return err
		}
		if err := writePart(w, "ubuntu.yaml", dist); err != nil {
			return err
		}
	case "centos":
		dist, err := NewCentOS(params.KubeletVersion)
		if err != nil {
			return err
		}
		if err := writePart(w, "centos.yaml", dist); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unsupported distribution: %v", distribution)
	}

	script, err := NewScript(params.ScriptParams)
	if err != nil {
		return err
	}
	if err := writePart(w, "bootstrap-script.yaml", script); err != nil {
		return err
	}

	if master {
		data, err := NewMaster(params.KubernetesParams)
		if err != nil {
			return err
		}
		if err := writePart(w, "kubeadm-master.yaml", data); err != nil {
			return err
		}
	} else {
		data, err := NewWorker(WorkerParams{
			ControlPlaneEndpoint: params.ControlPlaneEndpoint,
			Token:                params.Token,
		})
		if err != nil {
			return err
		}
		if err := writePart(w, "kubeadm-worker.yaml", data); err != nil {
			return err
		}
	}

	return nil
}

func writePart(w *multipart.Writer, name string, r io.Reader) error {
	partHeaders := textproto.MIMEHeader{
		"Content-Type":              []string{"text/cloud-config; charset=\"us-ascii\""},
		"MIME-Version":              []string{"1.0"},
		"Content-Transfer-Encoding": []string{"7bit"},
		"Content-Disposition":       []string{fmt.Sprintf("attachment; filename=\"%s\"", name)},
	}

	part, err := w.CreatePart(partHeaders)
	if err != nil {
		return err
	}

	_, err = io.Copy(part, r)
	return err
}

type kubelet struct {
	Version string
}

// NewUbuntu returns an io.Reader of the Ubuntu Cloud Config
func NewUbuntu(version string) (io.Reader, error) {
	return renderTemplate("ubuntu.yaml", ubuntuCloudConfig, kubelet{Version: version})
}

// NewCentOS returns an io.Reader of the CentOS Cloud Config
func NewCentOS(version string) (io.Reader, error) {
	return renderTemplate("ubuntu.yaml", centosCloudConfig, kubelet{Version: version})
}

func renderTemplate(name string, data string, params interface{}) (io.Reader, error) {
	cfg := template.Must(template.New(name).Parse(data))

	var buf bytes.Buffer
	if err := cfg.Execute(&buf, params); err != nil {
		return nil, err
	}

	return &buf, nil
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

// NewMaster returns an io.Reader of the Kubernetes Master Cloud Config
func NewMaster(params KubernetesParams) (io.Reader, error) {
	kubeadm, err := renderTemplate("kubeadm_config.yaml", masterKubeadmCloudConfig, params)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(kubeadm)
	if err != nil {
		return nil, err
	}
	params.KubeadmConfig = base64.StdEncoding.EncodeToString(data)
	return renderTemplate("master.yaml", masterCloudConfig, params)
}

// ScriptParams contains all parametes needed in the Bootstrap Script
type ScriptParams struct {
	Namespace        string
	Name             string
	BootstrapScript  string
	BootstrapService string
}

// NewScript returns an io.Reader of the Script Cloud Config
func NewScript(params ScriptParams) (io.Reader, error) {
	if params.BootstrapScript == "" {
		params.BootstrapScript = base64.StdEncoding.EncodeToString([]byte(bootstrapScript))
	}
	if params.BootstrapService == "" {
		params.BootstrapService = base64.StdEncoding.EncodeToString([]byte(bootstrapService))
	}
	return renderTemplate("kubeadm.yaml", genericCloudConfig, params)
}

// WorkerParams contains all parameters needed on Worker Nodes
type WorkerParams struct {
	ControlPlaneEndpoint string
	Token                string
	KubeadmConfig        string
}

// NewWorker returns an io.Reader of the Kubernetes Worker Cloud Config
func NewWorker(params WorkerParams) (io.Reader, error) {
	kubeadm, err := renderTemplate("kubeadm_config.yaml", workerKubeadmCloudConfig, params)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(kubeadm)
	if err != nil {
		return nil, err
	}
	params.KubeadmConfig = base64.StdEncoding.EncodeToString(data)
	return renderTemplate("worker.yaml", workerCloudConfig, params)
}

// IsSupported returns true if the passed in distribution is supported by the current
// implementation of userdata
func IsSupported(distri string) bool {
	_, ok := supportedDistributions[distri]
	return ok
}

// GetSupported returns a list of supported distributions
func GetSupported() []string {
	s := make([]string, len(supportedDistributions))
	for k := range supportedDistributions {
		s = append(s, k)
	}
	return s
}
