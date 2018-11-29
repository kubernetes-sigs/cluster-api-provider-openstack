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

package machine

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/golang/glog"
	errorwrapper "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	tokenapi "k8s.io/cluster-bootstrap/token/api"
	tokenutil "k8s.io/cluster-bootstrap/token/util"
	bootstrap "sigs.k8s.io/cluster-api-provider-openstack/pkg/bootstrap"

	openstackconfigv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/clients"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/machine/machinesetup"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	apierrors "sigs.k8s.io/cluster-api/pkg/errors"
	"sigs.k8s.io/cluster-api/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MachineSetupConfigPath = "/etc/machinesetup/machine_setup_configs.yaml"
	SshPrivateKeyPath      = "/etc/sshkeys/private"
	SshPublicKeyPath       = "/etc/sshkeys/public"
	SshKeyUserPath         = "/etc/sshkeys/user"
	CloudConfigPath        = "/etc/cloud/cloud_config.yaml"

	TimeoutInstanceCreate       = 5 * time.Minute
	RetryIntervalInstanceStatus = 10 * time.Second

	TokenTTL = 60 * time.Minute
)

type SshCreds struct {
	user           string
	privateKeyPath string
	publicKey      string
}

type OpenstackClient struct {
	scheme              *runtime.Scheme
	client              client.Client
	machineSetupWatcher *machinesetup.ConfigWatch
	machineService      *clients.InstanceService
	sshCred             *SshCreds
	*openstack.DeploymentClient
}

func NewActuator(machineClient client.Client, scheme *runtime.Scheme) (*OpenstackClient, error) {
	machineService, err := clients.NewInstanceService()
	if err != nil {
		return nil, err
	}
	var sshCred SshCreds
	b, err := ioutil.ReadFile(SshKeyUserPath)
	if err != nil {
		return nil, err
	}
	sshCred.user = string(b)
	b, err = ioutil.ReadFile(SshPublicKeyPath)
	if err != nil {
		return nil, err
	}
	sshCred.publicKey = string(b)

	keyPairList, err := machineService.GetKeyPairList()
	if err != nil {
		return nil, err
	}
	needCreate := true
	// check whether keypair already exist
	for i := range keyPairList {
		if sshCred.user == keyPairList[i].Name {
			if sshCred.publicKey == keyPairList[i].PublicKey {
				needCreate = false
			} else {
				err = machineService.DeleteKeyPair(keyPairList[i].Name)
				if err != nil {
					return nil, fmt.Errorf("unable to delete keypair: %v", err)
				}
			}
			break
		}
	}
	if needCreate {
		if _, err := os.Stat(SshPrivateKeyPath); err != nil {
			return nil, fmt.Errorf("ssh key pair need to be specified")
		}
		sshCred.privateKeyPath = SshPrivateKeyPath

		err = machineService.CreateKeyPair(sshCred.user, sshCred.publicKey)
		if err != nil {
			return nil, fmt.Errorf("create ssh key pair err: %v", err)
		}
	}

	if err != nil {
		return nil, err
	}

	setupConfigWatcher, err := machinesetup.NewConfigWatch(MachineSetupConfigPath)
	if err != nil {
		return nil, fmt.Errorf("error creating machine setup config watcher: %v", err)
	}

	return &OpenstackClient{
		client:              machineClient,
		machineService:      machineService,
		machineSetupWatcher: setupConfigWatcher,
		scheme:              scheme,
		sshCred:             &sshCred,
		DeploymentClient:    openstack.NewDeploymentClient(),
	}, nil
}

func (oc *OpenstackClient) Create(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	if oc.machineSetupWatcher == nil {
		return errors.New("a valid machine setup config watcher is required!")
	}

	providerConfig, err := openstackconfigv1.ClusterConfigFromProviderConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return oc.handleMachineError(machine, apierrors.InvalidMachineConfiguration(
			"Cannot unmarshal providerConfig field: %v", err))
	}

	if verr := oc.validateMachine(machine, providerConfig); verr != nil {
		return oc.handleMachineError(machine, verr)
	}

	instance, err := oc.instanceExists(machine)
	if err != nil {
		return err
	}
	if instance != nil {
		glog.Infof("Skipped creating a VM that already exists.\n")
		return nil
	}

	// get machine startup script
	machineSetupConfig, err := oc.machineSetupWatcher.GetMachineSetupConfig()
	if err != nil {
		return err
	}
	role, ok := machine.ObjectMeta.Labels["set"]
	if !ok {
		glog.Errorf("Check machine role err, treat as \"node\" by default")
		role = machinesetup.MachineRoleNode
	}
	startupScript, err := machineSetupConfig.GetSetupScript(role)
	if err != nil {
		return err
	}
	if util.IsMaster(machine) {
		startupScript, err = masterStartupScript(cluster, machine, startupScript)
		if err != nil {
			return oc.handleMachineError(machine, apierrors.CreateMachine(
				"error creating Openstack instance: %v", err))
		}
	} else {
		glog.Info("Creating bootstrap token")
		token, err := oc.createBootstrapToken()
		if err != nil {
			return oc.handleMachineError(machine, apierrors.CreateMachine(
				"error creating Openstack instance: %v", err))
		}
		startupScript, err = nodeStartupScript(cluster, machine, token, startupScript)
		if err != nil {
			return oc.handleMachineError(machine, apierrors.CreateMachine(
				"error creating Openstack instance: %v", err))
		}
	}

	instance, err = oc.machineService.InstanceCreate(machine.Name, providerConfig, startupScript, oc.sshCred.user)
	if err != nil {
		return oc.handleMachineError(machine, apierrors.CreateMachine(
			"error creating Openstack instance: %v", err))
	}
	// TODO: wait instance ready
	err = util.PollImmediate(RetryIntervalInstanceStatus, TimeoutInstanceCreate, func() (bool, error) {
		instance, err := oc.machineService.GetInstance(instance.ID)
		if err != nil {
			return false, nil
		}
		return instance.Status == "ACTIVE", nil
	})
	if err != nil {
		return oc.handleMachineError(machine, apierrors.CreateMachine(
			"error creating Openstack instance: %v", err))
	}

	if providerConfig.FloatingIP != "" {
		err := oc.machineService.AssociateFloatingIP(instance.ID, providerConfig.FloatingIP)
		if err != nil {
			return oc.handleMachineError(machine, apierrors.CreateMachine(
				"Associate floatingIP err: %v", err))
		}

	}

	return oc.updateAnnotation(machine, instance, providerConfig)
}

func (oc *OpenstackClient) Delete(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	instance, err := oc.instanceExists(machine)
	if err != nil {
		return err
	}

	if instance == nil {
		glog.Infof("Skipped deleting a VM that is already deleted.\n")
		return nil
	}

	id := machine.ObjectMeta.Annotations[openstack.OpenstackIdAnnotationKey]
	err = oc.machineService.InstanceDelete(id)
	if err != nil {
		return oc.handleMachineError(machine, apierrors.DeleteMachine(
			"error deleting Openstack instance: %v", err))
	}

	return nil
}

// TODO cluster-api is only watching for updates to individual Machines, not MachineSets.. why?
func (oc *OpenstackClient) Update(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	machineConfig, err := openstackconfigv1.MachineConfigFromProviderConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return err
	}

	status, err := oc.instanceStatus(machine)
	if err != nil {
		return err
	}

	currentMachine := (*clusterv1.Machine)(status)
	if currentMachine == nil {
		instance, err := oc.instanceExists(machine)
		if err != nil {
			return err
		}
		if instance != nil && instance.Status == "ACTIVE" {
			glog.Infof("Populating current state for boostrap machine %v", machine.ObjectMeta.Name)
			return oc.updateAnnotation(machine, instance, machineConfig)
		} else {
			return fmt.Errorf("Cannot retrieve current state to update machine %v", machine.ObjectMeta.Name)
		}
	}

	if err := oc.updateSecurityGroups(cluster, machine, machineConfig); err != nil {
		return errorwrapper.Wrap(err, "failed to ensure security groups")
	}

	// TODO updatedSecurityGroups is a part of Spec.ProviderConfig so it will trigger machine re-build as is
	if !oc.requiresUpdate(currentMachine, machine) {
		return nil
	}

	// TODO not all changes require recreating machine from scratch, need to differentiate.
	if util.IsMaster(currentMachine) {
		// TODO: add master inplace
		glog.Errorf("master inplace update failed: %v", err)
	} else {
		glog.Infof("re-creating machine %s for update.", currentMachine.ObjectMeta.Name)
		err = oc.Delete(cluster, currentMachine)
		if err != nil {
			glog.Errorf("delete machine %s for update failed: %v", currentMachine.ObjectMeta.Name, err)
		} else {
			err = oc.Create(cluster, machine)
			if err != nil {
				glog.Errorf("create machine %s for update failed: %v", machine.ObjectMeta.Name, err)
			}
		}
	}

	return nil
}

func (oc *OpenstackClient) Exists(cluster *clusterv1.Cluster, machine *clusterv1.Machine) (bool, error) {
	instance, err := oc.instanceExists(machine)
	if err != nil {
		return false, err
	}
	return instance != nil, err
}

func (oc *OpenstackClient) GetKubeConfig(cluster *clusterv1.Cluster, master *clusterv1.Machine) (string, error) {
	if oc.sshCred == nil {
		return "", fmt.Errorf("Get kubeConfig failed, don't have ssh keypair information")
	}
	ip, err := oc.GetIP(cluster, master)
	if err != nil {
		return "", err
	}

	machineConfig, err := openstackconfigv1.MachineConfigFromProviderConfig(master.Spec.ProviderConfig)
	if err != nil {
		return "", err
	}

	result := strings.TrimSpace(util.ExecCommand(
		"ssh", "-i", oc.sshCred.privateKeyPath,
		"-o", "StrictHostKeyChecking no",
		"-o", "UserKnownHostsFile /dev/null",
		fmt.Sprintf("%s@%s", machineConfig.SshUserName, ip),
		"echo STARTFILE; sudo cat /etc/kubernetes/admin.conf"))
	parts := strings.Split(result, "STARTFILE")
	if len(parts) != 2 {
		return "", nil
	}
	return strings.TrimSpace(parts[1]), nil
}

// If the OpenstackClient has a client for updating Machine objects, this will set
// the appropriate reason/message on the Machine.Status. If not, such as during
// cluster installation, it will operate as a no-op. It also returns the
// original error for convenience, so callers can do "return handleMachineError(...)".
func (oc *OpenstackClient) handleMachineError(machine *clusterv1.Machine, err *apierrors.MachineError) error {
	if oc.client != nil {
		reason := err.Reason
		message := err.Message
		machine.Status.ErrorReason = &reason
		machine.Status.ErrorMessage = &message
		if err := oc.client.Update(nil, machine); err != nil {
			return fmt.Errorf("unable to update machine status: %v", err)
		}
	}

	glog.Errorf("Machine error: %v", err.Message)
	return err
}

func (oc *OpenstackClient) requiresUpdate(a *clusterv1.Machine, b *clusterv1.Machine) bool {
	if a == nil || b == nil {
		return true
	}
	// Do not want status changes. Do want changes that impact machine provisioning
	return !reflect.DeepEqual(a.Spec.ObjectMeta, b.Spec.ObjectMeta) ||
		!reflect.DeepEqual(a.Spec.ProviderConfig, b.Spec.ProviderConfig) ||
		!reflect.DeepEqual(a.Spec.Versions, b.Spec.Versions) ||
		a.ObjectMeta.Name != b.ObjectMeta.Name
}

func (oc *OpenstackClient) instanceExists(machine *clusterv1.Machine) (instance *clients.Instance, err error) {
	machineConfig, err := openstackconfigv1.MachineConfigFromProviderConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, err
	}
	opts := &clients.InstanceListOpts{
		Name:   machine.Name,
		Image:  machineConfig.Image,
		Flavor: machineConfig.Flavor,
	}
	instanceList, err := oc.machineService.GetInstanceList(opts)
	if err != nil {
		return nil, err
	}
	if len(instanceList) == 0 {
		return nil, nil
	}
	return instanceList[0], nil
}

func (oc *OpenstackClient) createBootstrapToken() (string, error) {
	token, err := tokenutil.GenerateBootstrapToken()
	if err != nil {
		return "", err
	}

	expiration := time.Now().UTC().Add(TokenTTL)
	tokenSecret, err := bootstrap.GenerateTokenSecret(token, expiration)
	if err != nil {
		panic(fmt.Sprintf("unable to create token. there might be a bug somwhere: %v", err))
	}

	err = oc.client.Create(context.TODO(), tokenSecret)
	if err != nil {
		return "", err
	}

	return tokenutil.TokenFromIDAndSecret(
		string(tokenSecret.Data[tokenapi.BootstrapTokenIDKey]),
		string(tokenSecret.Data[tokenapi.BootstrapTokenSecretKey]),
	), nil
}

func (oc *OpenstackClient) validateMachine(machine *clusterv1.Machine, config *openstackconfigv1.OpenstackProviderConfig) *apierrors.MachineError {
	if machine.Spec.Versions.Kubelet == "" {
		return apierrors.InvalidMachineConfiguration("spec.versions.kubelet can't be empty")
	}
	// TODO: other validate of openstackCloud
	return nil
}
