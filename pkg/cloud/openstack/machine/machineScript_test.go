package machine

import (
	"testing"

	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/yaml"
)

const providerSpecYAML = `
value:
  apiVersion: "openstackproviderconfig/v1alpha1"
  kind: "OpenstackProviderSpec"
`

func TestNodeStartupScriptEmpty(t *testing.T) {
	cluster := &clusterv1.Cluster{}
	machine := &clusterv1.Machine{}
	err := yaml.Unmarshal([]byte(providerSpecYAML), &machine.Spec.ProviderSpec)
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	token := ""
	script_template := ""

	// `machine` has no endpoint specified so having `call
	// .GetMasterEndpoint` in the script template would fail. But we
	// don't, so this should succeed.
	script, err := nodeStartupScript(cluster, machine, token, script_template)
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	if script != "" {
		t.Errorf("Expected script, found %q instead", script)
	}
}

func TestNodeStartupScriptEndpointError(t *testing.T) {
	cluster := &clusterv1.Cluster{}
	machine := &clusterv1.Machine{}
	err := yaml.Unmarshal([]byte(providerSpecYAML), &machine.Spec.ProviderSpec)
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	token := ""
	script_template := "{{ call .GetMasterEndpoint }}"
	// `machine` has no endpoint specified so having `call
	// .GetMasterEndpoint` in the template should fail.
	script, err := nodeStartupScript(cluster, machine, token, script_template)
	if err == nil {
		t.Errorf("Expected GetMasterEndpoint to fail, but it succeeded. Startup script %q", script)
	}
}

func TestNodeStartupScriptWithEndpoint(t *testing.T) {
	cluster := clusterv1.Cluster{}
	cluster.Status.APIEndpoints = make([]clusterv1.APIEndpoint, 1)
	cluster.Status.APIEndpoints[0] = clusterv1.APIEndpoint{
		Host: "example.com",
		Port: 8080,
	}

	machine := &clusterv1.Machine{}
	err := yaml.Unmarshal([]byte(providerSpecYAML), &machine.Spec.ProviderSpec)
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	token := ""
	script_template := "{{ call .GetMasterEndpoint }}"
	script, err := nodeStartupScript(&cluster, machine, token, script_template)
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	expected := "example.com:8080"
	if script != expected {
		t.Errorf("Expected %q master endpoint, found %q instead", expected, script)
	}
}
