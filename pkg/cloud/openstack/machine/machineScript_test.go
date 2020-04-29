package machine

import (
	"testing"

	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	"sigs.k8s.io/yaml"
)

const providerSpecYAML = `
value:
  apiVersion: "openstackproviderconfig/v1alpha1"
  kind: "OpenstackProviderSpec"
`

func TestNodeStartupScriptEmpty(t *testing.T) {
	machine := &machinev1.Machine{}
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
	script, err := nodeStartupScript(machine, token, script_template)
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	if script != "" {
		t.Errorf("Expected script, found %q instead", script)
	}
}

func TestNodeStartupScriptEndpointError(t *testing.T) {
	machine := &machinev1.Machine{}
	err := yaml.Unmarshal([]byte(providerSpecYAML), &machine.Spec.ProviderSpec)
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	token := ""
	script_template := "{{ call .GetMasterEndpoint }}"
	// `machine` has no endpoint specified so having `call
	// .GetMasterEndpoint` in the template should fail.
	script, err := nodeStartupScript(machine, token, script_template)
	if err == nil {
		t.Errorf("Expected GetMasterEndpoint to fail, but it succeeded. Startup script %q", script)
	}
}
