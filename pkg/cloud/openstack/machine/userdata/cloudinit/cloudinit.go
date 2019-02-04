package cloudinit

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/textproto"
	"text/template"

	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/machine/userdata"
)

// CloudInit represents the entrypoint to create a cloud-config for a distribution
type CloudInit struct {
	Distribution string
}

// Write a cloud config for a Kubernetes Node into an io.Writer (f)
// The result will be a multipart cloud config. For more information on cloud config/init
// visit https://cloudinit.readthedocs.io/en/latest/
func (c *CloudInit) Write(f io.Writer, master bool, params userdata.SetupParams) error {
	w := multipart.NewWriter(f)
	defer w.Close()

	fmt.Fprintf(f, "Content-Type: multipart/mixed; boundary=\"%s\"\n", w.Boundary())
	fmt.Fprintf(f, "MIME-Version: 1.0\n\n")

	switch c.Distribution {
	case "ubuntu":
		dist, err := renderTemplate("ubuntu.yaml", ubuntuCloudConfig, kubelet{Version: params.KubeletVersion})
		if err != nil {
			return err
		}
		if err := writePart(w, "ubuntu.yaml", dist); err != nil {
			return err
		}
	case "centos":
		dist, err := renderTemplate("centos.yaml", centosCloudConfig, kubelet{Version: params.KubeletVersion})
		if err != nil {
			return err
		}
		if err := writePart(w, "centos.yaml", dist); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unsupported distribution: %v", c.Distribution)
	}

	script, err := renderScript(params.ScriptParams)
	if err != nil {
		return err
	}
	if err := writePart(w, "bootstrap-script.yaml", script); err != nil {
		return err
	}

	if master {
		data, err := renderMaster(params.KubernetesParams)
		if err != nil {
			return err
		}
		if err := writePart(w, "kubeadm-master.yaml", data); err != nil {
			return err
		}
	} else {
		data, err := renderWorker(userdata.WorkerParams{
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

func renderTemplate(name string, data string, params interface{}) (io.Reader, error) {
	cfg := template.Must(template.New(name).Parse(data))

	var buf bytes.Buffer
	if err := cfg.Execute(&buf, params); err != nil {
		return nil, err
	}

	return &buf, nil
}

// renderMaster returns an io.Reader of the Kubernetes Master Cloud Config
func renderMaster(params userdata.KubernetesParams) (io.Reader, error) {
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

// renderScript returns an io.Reader of the Script Cloud Config
func renderScript(params userdata.ScriptParams) (io.Reader, error) {
	userdata.DefaultScriptParams(&params)

	return renderTemplate("kubeadm.yaml", genericCloudConfig, params)
}

// renderWorker returns an io.Reader of the Kubernetes Worker Cloud Config
func renderWorker(params userdata.WorkerParams) (io.Reader, error) {
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
