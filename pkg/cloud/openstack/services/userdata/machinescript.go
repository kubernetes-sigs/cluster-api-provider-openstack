/*
Copyright 2018 The Kubernetes Authors.

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

package userdata

import (
	"bytes"
	"context"
	"errors"
	werrors "github.com/pkg/errors"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"path"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/options"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/certificates"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/compute"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/provider"
	"sigs.k8s.io/cluster-api/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"text/template"
	"time"

	"fmt"

	"encoding/json"
	clconfig "github.com/coreos/container-linux-config-transpiler/config"
	tokenapi "k8s.io/cluster-bootstrap/token/api"
	tokenutil "k8s.io/cluster-bootstrap/token/util"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha2"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/bootstrap"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha2"
	apierrors "sigs.k8s.io/cluster-api/pkg/errors"
)

const (
	UserDataKey          = "userData"
	DisableTemplatingKey = "disableTemplating"
	PostprocessorKey     = "postprocessor"
)

type setupParams struct {
	Machine *clusterv1.Machine

	CACert           string
	CAKey            string
	EtcdCACert       string
	EtcdCAKey        string
	FrontProxyCACert string
	FrontProxyCAKey  string
	SaCert           string
	SaKey            string

	KubeadmConfig string
}

func GetUserData(ctrlClient client.Client, machine *clusterv1.Machine, openStackMachine *infrav1.OpenStackMachine, cluster *clusterv1.Cluster, openStackCluster *infrav1.OpenStackCluster) (string, error) {
	ctx := context.TODO()
	// get machine startup script
	var ok bool
	var disableTemplating bool
	var postprocessor string
	var postprocess bool
	var err error

	var userData []byte
	if openStackMachine.Spec.UserDataSecret != nil {
		namespace := openStackMachine.Spec.UserDataSecret.Namespace
		if namespace == "" {
			namespace = machine.Namespace
		}

		if openStackMachine.Spec.UserDataSecret.Name == "" {
			return "", fmt.Errorf("UserDataSecret name must be provided")
		}

		userDataSecret := &v1.Secret{}
		err := ctrlClient.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      openStackMachine.Spec.UserDataSecret.Name,
		}, userDataSecret)
		if err != nil {
			return "", err
		}

		userData, ok = userDataSecret.Data[UserDataKey]
		if !ok {
			return "", fmt.Errorf("machine's userdata secret %v in namespace %v did not contain key %v", openStackMachine.Spec.UserDataSecret.Name, namespace, UserDataKey)
		}

		_, disableTemplating = userDataSecret.Data[DisableTemplatingKey]

		var p []byte
		p, postprocess = userDataSecret.Data[PostprocessorKey]

		postprocessor = string(p)
	} else if options.UserDataFolder != "" {
		userData, err = ioutil.ReadFile(path.Join(options.UserDataFolder, fmt.Sprintf("%s.yaml", machine.Name)))
		if err != nil {
			return "", fmt.Errorf("could not load local userdata files: %v", err)
		}
		postprocessor = options.UserDataPostprocessor
		if postprocessor != "" {
			postprocess = true
		}
	}

	var userDataRendered string
	if len(userData) > 0 && !disableTemplating {
		isNodeJoin, err := isNodeJoin(ctrlClient, openStackCluster, machine)
		if err != nil {
			return "", apierrors.CreateMachine("error creating Openstack instance: %v", err)
		}

		var bootstrapToken string
		if isNodeJoin {
			klog.Info("Creating bootstrap token")
			bootstrapToken, err = createBootstrapToken(openStackCluster)
			if err != nil {
				return "", apierrors.CreateMachine("error creating Openstack instance: %v", err)
			}
		}

		userDataRendered, err = startupScript(machine, openStackMachine, cluster, openStackCluster, string(userData), bootstrapToken)
		if err != nil {
			return "", apierrors.CreateMachine("error creating Openstack instance: %v", err)
		}
		if util.IsControlPlaneMachine(machine) && isNodeJoin {
			// A little bit hacky but maybe good enough until v1alpha2. The alternative would be to template
			// either the kubeadm command or the whole kubeadm service file. But I think the 2nd option would
			// reduce the flexibility too much.
			userDataRendered = strings.ReplaceAll(userDataRendered, "kubeadm init", "kubeadm join")
		}
	} else {
		userDataRendered = string(userData)
	}

	if postprocess {
		switch postprocessor {
		// Postprocess with the Container Linux ct transpiler.
		case "ct":
			clCfg, ast, report := clconfig.Parse([]byte(userDataRendered))
			if len(report.Entries) > 0 {
				return "", fmt.Errorf("postprocessor error: %s", report.String())
			}

			ignCfg, report := clconfig.Convert(clCfg, "openstack-metadata", ast)
			if len(report.Entries) > 0 {
				return "", fmt.Errorf("postprocessor error: %s", report.String())
			}

			ud, err := json.Marshal(&ignCfg)
			if err != nil {
				return "", fmt.Errorf("postprocessor error: %s", err)
			}

			userDataRendered = string(ud)

		default:
			return "", fmt.Errorf("postprocessor error: unknown postprocessor: '%s'", postprocessor)
		}
	}
	return userDataRendered, nil
}

func isNodeJoin(ctrlClient client.Client, openStackCluster *infrav1.OpenStackCluster, machine *clusterv1.Machine) (bool, error) {

	// Worker machines always join
	if !util.IsControlPlaneMachine(machine) {
		klog.Infof("Worker machine %s is joining the cluster\n", machine.Name)
		return true, nil
	}

	// Get control plane machines and return false if none found
	controlPlaneMachines, err := getControlPlaneMachines(ctrlClient)
	if err != nil {
		return false, apierrors.CreateMachine("error retrieving control plane machines: %v", err)
	}
	if len(controlPlaneMachines) == 0 {
		klog.Infof("Could not find control plane machine: creating first control plane machine %s\n", machine.Name)
		return false, nil
	}

	// Get control plane machine instances and return false if none found
	osProviderClient, clientOpts, err := provider.NewClientFromCluster(ctrlClient, openStackCluster)
	if err != nil {
		return false, err
	}
	computeService, err := compute.NewService(osProviderClient, clientOpts)
	if err != nil {
		return false, err
	}
	instanceList, err := computeService.GetInstanceList(&compute.InstanceListOpts{})
	if err != nil {
		return false, err
	}
	if len(instanceList) == 0 {
		klog.Infof("Could not find control plane machine: creating first control plane machine %s\n", machine.Name)
		return false, nil
	}

	for _, controlPlaneMachine := range controlPlaneMachines {
		for _, instance := range instanceList {
			if controlPlaneMachine.Name == instance.Name {
				klog.Infof("Found control plane machine %s: control plane machine %s is joining the cluster\n", controlPlaneMachine.Name, machine.Name)
				return true, nil
			}
		}
	}
	klog.Infof("Could not find control plane machine: creating first control plane machine %s\n", machine.Name)
	return false, nil
}

func getControlPlaneMachines(ctrlClient client.Client) ([]*clusterv1.Machine, error) {
	var controlPlaneMachines []*clusterv1.Machine
	msList := &clusterv1.MachineList{}
	err := ctrlClient.List(context.TODO(), msList)
	if err != nil {
		return nil, fmt.Errorf("error retrieving machines: %v", err)
	}
	for _, m := range msList.Items {
		if util.IsControlPlaneMachine(&m) {
			// we need DeepCopy because if we append the Pointer it will all be
			// the same machine
			controlPlaneMachines = append(controlPlaneMachines, m.DeepCopy())
		}
	}
	return controlPlaneMachines, nil
}

func createBootstrapToken(openStackCluster *infrav1.OpenStackCluster) (string, error) {
	token, err := tokenutil.GenerateBootstrapToken()
	if err != nil {
		return "", err
	}

	expiration := time.Now().UTC().Add(options.TokenTTL)
	tokenSecret, err := bootstrap.GenerateTokenSecret(token, expiration)
	if err != nil {
		panic(fmt.Sprintf("unable to create token. there might be a bug somwhere: %v", err))
	}

	kubeClient, err := getKubeClient(openStackCluster)
	if err != nil {
		return "", err
	}

	err = kubeClient.Create(context.TODO(), tokenSecret)
	if err != nil {
		return "", err
	}

	return tokenutil.TokenFromIDAndSecret(
		string(tokenSecret.Data[tokenapi.BootstrapTokenIDKey]),
		string(tokenSecret.Data[tokenapi.BootstrapTokenSecretKey]),
	), nil
}

func getKubeClient(openStackCluster *infrav1.OpenStackCluster) (client.Client, error) {
	kubeConfig, err := GetKubeConfig(openStackCluster)
	if err != nil {
		return nil, err
	}

	apiConfig, err := clientcmd.Load([]byte(kubeConfig))
	if err != nil {
		return nil, err
	}

	cfg, err := clientcmd.NewDefaultClientConfig(*apiConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, err
	}

	cl, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("unable to create client for restConfig: %v", err)
	}

	return cl, nil
}

func startupScript(machine *clusterv1.Machine, openStackMachine *infrav1.OpenStackMachine, cluster *clusterv1.Cluster, openStackCluster *infrav1.OpenStackCluster, userdata, bootstrapToken string) (string, error) {
	if err := validateCertificates(openStackCluster); err != nil {
		return "", err
	}

	kubeadmConfig, err := generateKubeadmConfig(util.IsControlPlaneMachine(machine), bootstrapToken, machine, openStackMachine, cluster, openStackCluster)
	if err != nil {
		return "", err
	}

	params := setupParams{
		CACert:           string(openStackCluster.Spec.CAKeyPair.Cert),
		CAKey:            string(openStackCluster.Spec.CAKeyPair.Key),
		EtcdCACert:       string(openStackCluster.Spec.EtcdCAKeyPair.Cert),
		EtcdCAKey:        string(openStackCluster.Spec.EtcdCAKeyPair.Key),
		FrontProxyCACert: string(openStackCluster.Spec.FrontProxyCAKeyPair.Cert),
		FrontProxyCAKey:  string(openStackCluster.Spec.FrontProxyCAKeyPair.Key),
		SaCert:           string(openStackCluster.Spec.SAKeyPair.Cert),
		SaKey:            string(openStackCluster.Spec.SAKeyPair.Key),
		Machine:          machine,
		KubeadmConfig:    kubeadmConfig,
	}

	fMap := map[string]interface{}{
		"EscapeNewLines": templateEscapeNewLines,
		"Indent":         templateYAMLIndent,
	}
	startUpScript := template.Must(template.New("startUp").Funcs(fMap).Parse(userdata))

	var buf bytes.Buffer
	if err := startUpScript.Execute(&buf, params); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func validateCertificates(openStackCluster *infrav1.OpenStackCluster) error {
	if !isKeyPairValid(openStackCluster.Spec.CAKeyPair.Cert, openStackCluster.Spec.CAKeyPair.Key) {
		return errors.New("CA cert material in the openStackCluster.Spec is missing cert/key")
	}

	if !isKeyPairValid(openStackCluster.Spec.EtcdCAKeyPair.Cert, openStackCluster.Spec.EtcdCAKeyPair.Key) {
		return errors.New("ETCD CA cert material in the openStackCluster.Spec is  missing cert/key")
	}

	if !isKeyPairValid(openStackCluster.Spec.FrontProxyCAKeyPair.Cert, openStackCluster.Spec.FrontProxyCAKeyPair.Key) {
		return errors.New("FrontProxy CA cert material in openStackCluster.Spec is  missing cert/key")
	}

	if !isKeyPairValid(openStackCluster.Spec.SAKeyPair.Cert, openStackCluster.Spec.SAKeyPair.Key) {
		return errors.New("ServiceAccount cert material in openStackCluster.Spec is  missing cert/key")
	}
	return nil
}

func isKeyPairValid(cert, key []byte) bool {
	return len(cert) > 0 && len(key) > 0
}

func templateEscapeNewLines(s string) string {
	return strings.ReplaceAll(s, "\n", "\\n")
}

func templateYAMLIndent(i int, input string) string {
	split := strings.Split(input, "\n")
	ident := "\n" + strings.Repeat(" ", i)
	return strings.Repeat(" ", i) + strings.Join(split, ident)
}

// GetKubeConfig returns the kubeConfig after the bootstrap process is complete.
func GetKubeConfig(openStackCluster *infrav1.OpenStackCluster) (string, error) {

	cert, err := certificates.DecodeCertPEM(openStackCluster.Spec.CAKeyPair.Cert)
	if err != nil {
		return "", werrors.Wrap(err, "failed to decode CA Cert")
	} else if cert == nil {
		return "", errors.New("certificate not found in clusterProviderSpec")
	}

	key, err := certificates.DecodePrivateKeyPEM(openStackCluster.Spec.CAKeyPair.Key)
	if err != nil {
		return "", werrors.Wrap(err, "failed to decode private key")
	} else if key == nil {
		return "", errors.New("key not found in clusterProviderSpec")
	}

	cfg, err := certificates.NewKubeconfig(openStackCluster.Name, fmt.Sprintf("https://%s", openStackCluster.Spec.ClusterConfiguration.ControlPlaneEndpoint), cert, key)
	if err != nil {
		return "", werrors.Wrap(err, "failed to generate a kubeconfig")
	}

	yaml, err := clientcmd.Write(*cfg)
	if err != nil {
		return "", werrors.Wrap(err, "failed to serialize config to yaml")
	}

	return string(yaml), nil
}
