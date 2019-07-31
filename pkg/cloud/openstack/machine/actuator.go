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
	"encoding/json"
	"fmt"
	"net"
	"os"
	"reflect"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"

	clusterv1 "github.com/openshift/cluster-api/pkg/apis/cluster/v1alpha1"
	machinev1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	apierrors "github.com/openshift/cluster-api/pkg/errors"
	"github.com/openshift/cluster-api/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	tokenapi "k8s.io/cluster-bootstrap/token/api"
	tokenutil "k8s.io/cluster-bootstrap/token/util"
	"k8s.io/klog"
	openstackconfigv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/bootstrap"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/clients"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/options"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clconfig "github.com/coreos/container-linux-config-transpiler/config"
)

const (
	CloudConfigPath = "/etc/cloud/cloud_config.yaml"

	UserDataKey          = "userData"
	DisableTemplatingKey = "disableTemplating"
	PostprocessorKey     = "postprocessor"

	TimeoutInstanceCreate       = 5
	TimeoutInstanceDelete       = 5
	RetryIntervalInstanceStatus = 10 * time.Second
)

type OpenstackClient struct {
	params openstack.ActuatorParams
	scheme *runtime.Scheme
	client client.Client
	*openstack.DeploymentClient
}

func NewActuator(params openstack.ActuatorParams) (*OpenstackClient, error) {
	return &OpenstackClient{
		params:           params,
		client:           params.Client,
		scheme:           params.Scheme,
		DeploymentClient: openstack.NewDeploymentClient(),
	}, nil
}

func getTimeout(name string, timeout int) time.Duration {
	if v := os.Getenv(name); v != "" {
		timeout, err := strconv.Atoi(v)
		if err == nil {
			return time.Duration(timeout)
		}
	}
	return time.Duration(timeout)
}

func (oc *OpenstackClient) Create(ctx context.Context, cluster *clusterv1.Cluster, machine *machinev1.Machine) error {
	kubeClient := oc.params.KubeClient

	machineService, err := clients.NewInstanceServiceFromMachine(kubeClient, machine)
	if err != nil {
		return err
	}

	providerSpec, err := openstackconfigv1.MachineSpecFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return oc.handleMachineError(machine, apierrors.InvalidMachineConfiguration(
			"Cannot unmarshal providerSpec field: %v", err))
	}

	if verr := oc.validateMachine(machine, providerSpec); verr != nil {
		return oc.handleMachineError(machine, verr)
	}

	instance, err := oc.instanceExists(machine)
	if err != nil {
		return err
	}
	if instance != nil {
		klog.Infof("Skipped creating a VM that already exists.\n")
		return nil
	}

	// get machine startup script
	var ok bool
	var disableTemplating bool
	var postprocessor string
	var postprocess bool

	userData := []byte{}
	if providerSpec.UserDataSecret != nil {
		namespace := providerSpec.UserDataSecret.Namespace
		if namespace == "" {
			namespace = machine.Namespace
		}

		if providerSpec.UserDataSecret.Name == "" {
			return fmt.Errorf("UserDataSecret name must be provided")
		}

		userDataSecret, err := kubeClient.CoreV1().Secrets(namespace).Get(providerSpec.UserDataSecret.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		userData, ok = userDataSecret.Data[UserDataKey]
		if !ok {
			return fmt.Errorf("Machine's userdata secret %v in namespace %v did not contain key %v", providerSpec.UserDataSecret.Name, namespace, UserDataKey)
		}

		_, disableTemplating = userDataSecret.Data[DisableTemplatingKey]

		var p []byte
		p, postprocess = userDataSecret.Data[PostprocessorKey]

		postprocessor = string(p)
	}

	var userDataRendered string
	if len(userData) > 0 && !disableTemplating {
		// FIXME(mandre) Find the right way to check if machine is part of the control plane
		if machine.ObjectMeta.Name != "" {
			userDataRendered, err = masterStartupScript(cluster, machine, string(userData))
			if err != nil {
				return oc.handleMachineError(machine, apierrors.CreateMachine(
					"error creating Openstack instance: %v", err))
			}
		} else {
			klog.Info("Creating bootstrap token")
			token, err := oc.createBootstrapToken()
			if err != nil {
				return oc.handleMachineError(machine, apierrors.CreateMachine(
					"error creating Openstack instance: %v", err))
			}
			userDataRendered, err = nodeStartupScript(cluster, machine, token, string(userData))
			if err != nil {
				return oc.handleMachineError(machine, apierrors.CreateMachine(
					"error creating Openstack instance: %v", err))
			}
		}
	} else {
		userDataRendered = string(userData)
	}

	// NOTE(shadower): the OpenShift installer does not create the
	// `cluster` object so it's `nil` here. Read the cluster name from
	// the `machine` instead.
	var clusterName string
	// TODO(egarcia): if we ever use the cluster object, this will benifit from reading from it
	var clusterSpec openstackconfigv1.OpenstackClusterProviderSpec
	if cluster == nil {
		// TODO(shadower): if/when we ever reconcile the
		// `machine.openshift.io` bit we should be able to merge this
		// upstream.
		clusterName = fmt.Sprintf("%s-%s", machine.Namespace, machine.Labels["machine.openshift.io/cluster-api-cluster"])
	} else {
		clusterName = fmt.Sprintf("%s-%s", cluster.ObjectMeta.Namespace, cluster.Name)
	}

	if postprocess {
		switch postprocessor {
		// Postprocess with the Container Linux ct transpiler.
		case "ct":
			clcfg, ast, report := clconfig.Parse([]byte(userDataRendered))
			if len(report.Entries) > 0 {
				return fmt.Errorf("Postprocessor error: %s", report.String())
			}

			ignCfg, report := clconfig.Convert(clcfg, "openstack-metadata", ast)
			if len(report.Entries) > 0 {
				return fmt.Errorf("Postprocessor error: %s", report.String())
			}

			ud, err := json.Marshal(&ignCfg)
			if err != nil {
				return fmt.Errorf("Postprocessor error: %s", err)
			}

			userDataRendered = string(ud)

		default:
			return fmt.Errorf("Postprocessor error: unknown postprocessor: '%s'", postprocessor)
		}
	}

	instance, err = machineService.InstanceCreate(clusterName, machine.Name, &clusterSpec, providerSpec, userDataRendered, providerSpec.KeyName, oc.params.ConfigClient)

	if err != nil {
		return oc.handleMachineError(machine, apierrors.CreateMachine(
			"error creating Openstack instance: %v", err))
	}
	instanceCreateTimeout := getTimeout("CLUSTER_API_OPENSTACK_INSTANCE_CREATE_TIMEOUT", TimeoutInstanceCreate)
	instanceCreateTimeout = instanceCreateTimeout * time.Minute
	err = util.PollImmediate(RetryIntervalInstanceStatus, instanceCreateTimeout, func() (bool, error) {
		instance, err := machineService.GetInstance(instance.ID)
		if err != nil {
			return false, nil
		}
		return instance.Status == "ACTIVE", nil
	})
	if err != nil {
		return oc.handleMachineError(machine, apierrors.CreateMachine(
			"error creating Openstack instance: %v", err))
	}

	if providerSpec.FloatingIP != "" {
		err := machineService.AssociateFloatingIP(instance.ID, providerSpec.FloatingIP)
		if err != nil {
			return oc.handleMachineError(machine, apierrors.CreateMachine(
				"Associate floatingIP err: %v", err))
		}

	}

	return oc.updateAnnotation(machine, instance.ID)
}

func (oc *OpenstackClient) Delete(ctx context.Context, cluster *clusterv1.Cluster, machine *machinev1.Machine) error {
	machineService, err := clients.NewInstanceServiceFromMachine(oc.params.KubeClient, machine)
	if err != nil {
		return err
	}

	instance, err := oc.instanceExists(machine)
	if err != nil {
		return err
	}

	if instance == nil {
		klog.Infof("Skipped deleting %s that is already deleted.\n", machine.Name)
		return nil
	}

	id := machine.ObjectMeta.Annotations[openstack.OpenstackIdAnnotationKey]
	err = machineService.InstanceDelete(id)
	if err != nil {
		return oc.handleMachineError(machine, apierrors.DeleteMachine(
			"error deleting Openstack instance: %v", err))
	}

	return nil
}

func (oc *OpenstackClient) Update(ctx context.Context, cluster *clusterv1.Cluster, machine *machinev1.Machine) error {
	status, err := oc.instanceStatus(machine)
	if err != nil {
		return err
	}

	currentMachine := (*machinev1.Machine)(status)
	if currentMachine == nil {
		instance, err := oc.instanceExists(machine)
		if err != nil {
			return err
		}
		if instance != nil && instance.Status == "ACTIVE" {
			klog.Infof("Populating current state for boostrap machine %v", machine.ObjectMeta.Name)
			return oc.updateAnnotation(machine, instance.ID)
		} else {
			return fmt.Errorf("Cannot retrieve current state to update machine %v", machine.ObjectMeta.Name)
		}
	}

	if !oc.requiresUpdate(currentMachine, machine) {
		return nil
	}

	// FIXME(mandre) Find the right way to check if machine is part of the control plane
	if currentMachine.ObjectMeta.Name != "" {
		// TODO: add master inplace
		klog.Errorf("master inplace update failed: not support master in place update now")
	} else {
		klog.Infof("re-creating machine %s for update.", currentMachine.ObjectMeta.Name)
		err = oc.Delete(ctx, cluster, currentMachine)
		if err != nil {
			klog.Errorf("delete machine %s for update failed: %v", currentMachine.ObjectMeta.Name, err)
		} else {
			instanceDeleteTimeout := getTimeout("CLUSTER_API_OPENSTACK_INSTANCE_DELETE_TIMEOUT", TimeoutInstanceDelete)
			instanceDeleteTimeout = instanceDeleteTimeout * time.Minute
			err = util.PollImmediate(RetryIntervalInstanceStatus, instanceDeleteTimeout, func() (bool, error) {
				instance, err := oc.instanceExists(machine)
				if err != nil {
					return false, nil
				}
				return instance == nil, nil
			})
			if err != nil {
				return oc.handleMachineError(machine, apierrors.DeleteMachine(
					"error deleting Openstack instance: %v", err))
			}

			err = oc.Create(ctx, cluster, machine)
			if err != nil {
				klog.Errorf("create machine %s for update failed: %v", machine.ObjectMeta.Name, err)
			}
			klog.Infof("Successfully updated machine %s", currentMachine.ObjectMeta.Name)
		}
	}

	return nil
}

func (oc *OpenstackClient) Exists(ctx context.Context, cluster *clusterv1.Cluster, machine *machinev1.Machine) (bool, error) {
	instance, err := oc.instanceExists(machine)
	if err != nil {
		return false, fmt.Errorf("Error checking if instance exists (machine/actuator.go 346): %v", err)
	}
	return instance != nil, err
}

func getIPFromInstance(instance *clients.Instance) (string, error) {
	if instance.AccessIPv4 != "" && net.ParseIP(instance.AccessIPv4) != nil {
		return instance.AccessIPv4, nil
	}
	type networkInterface struct {
		Address string  `json:"addr"`
		Version float64 `json:"version"`
		Type    string  `json:"OS-EXT-IPS:type"`
	}
	var addrList []string

	for _, b := range instance.Addresses {
		list, err := json.Marshal(b)
		if err != nil {
			return "", fmt.Errorf("extract IP from instance err: %v", err)
		}
		var networks []interface{}
		json.Unmarshal(list, &networks)
		for _, network := range networks {
			var netInterface networkInterface
			b, _ := json.Marshal(network)
			json.Unmarshal(b, &netInterface)
			if netInterface.Version == 4.0 {
				if netInterface.Type == "floating" {
					return netInterface.Address, nil
				}
				addrList = append(addrList, netInterface.Address)
			}
		}
	}
	if len(addrList) != 0 {
		return addrList[0], nil
	}
	return "", fmt.Errorf("extract IP from instance err")
}

// If the OpenstackClient has a client for updating Machine objects, this will set
// the appropriate reason/message on the Machine.Status. If not, such as during
// cluster installation, it will operate as a no-op. It also returns the
// original error for convenience, so callers can do "return handleMachineError(...)".
func (oc *OpenstackClient) handleMachineError(machine *machinev1.Machine, err *apierrors.MachineError) error {
	if oc.client != nil {
		reason := err.Reason
		message := err.Message
		machine.Status.ErrorReason = &reason
		machine.Status.ErrorMessage = &message
		if err := oc.client.Update(nil, machine); err != nil {
			return fmt.Errorf("unable to update machine status: %v", err)
		}
	}

	klog.Errorf("Machine error %s: %v", machine.Name, err.Message)
	return err
}

func (oc *OpenstackClient) updateAnnotation(machine *machinev1.Machine, id string) error {
	if machine.ObjectMeta.Annotations == nil {
		machine.ObjectMeta.Annotations = make(map[string]string)
	}
	machine.ObjectMeta.Annotations[openstack.OpenstackIdAnnotationKey] = id
	instance, _ := oc.instanceExists(machine)
	ip, err := getIPFromInstance(instance)
	if err != nil {
		return err
	}
	machine.ObjectMeta.Annotations[openstack.OpenstackIPAnnotationKey] = ip
	if err := oc.client.Update(nil, machine); err != nil {
		return err
	}

	networkAddresses := []corev1.NodeAddress{}
	networkAddresses = append(networkAddresses, corev1.NodeAddress{
		Type:    corev1.NodeInternalIP,
		Address: ip,
	})

	networkAddresses = append(networkAddresses, corev1.NodeAddress{
		Type:    corev1.NodeHostName,
		Address: machine.Name,
	})

	networkAddresses = append(networkAddresses, corev1.NodeAddress{
		Type:    corev1.NodeInternalDNS,
		Address: machine.Name,
	})

	machineCopy := machine.DeepCopy()
	machineCopy.Status.Addresses = networkAddresses

	if !equality.Semantic.DeepEqual(machine.Status.Addresses, machineCopy.Status.Addresses) {
		if err := oc.client.Status().Update(nil, machineCopy); err != nil {
			return err
		}
	}

	return oc.updateInstanceStatus(machine)
}

func (oc *OpenstackClient) requiresUpdate(a *machinev1.Machine, b *machinev1.Machine) bool {
	if a == nil || b == nil {
		return true
	}
	// Do not want status changes. Do want changes that impact machine provisioning
	return !reflect.DeepEqual(a.Spec.ObjectMeta, b.Spec.ObjectMeta) ||
		!reflect.DeepEqual(a.Spec.ProviderSpec, b.Spec.ProviderSpec) ||
		a.ObjectMeta.Name != b.ObjectMeta.Name
}

func (oc *OpenstackClient) instanceExists(machine *machinev1.Machine) (instance *clients.Instance, err error) {
	machineSpec, err := openstackconfigv1.MachineSpecFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, fmt.Errorf("\nError getting the machine spec from the provider spec (machine/actuator.go 457): %v", err)
	}
	opts := &clients.InstanceListOpts{
		Name:   machine.Name,
		Image:  machineSpec.Image,
		Flavor: machineSpec.Flavor,
	}

	machineService, err := clients.NewInstanceServiceFromMachine(oc.params.KubeClient, machine)
	if err != nil {
		return nil, fmt.Errorf("\nError getting a new instance service from the machine (machine/actuator.go 467): %v", err)
	}

	instanceList, err := machineService.GetInstanceList(opts)
	if err != nil {
		return nil, fmt.Errorf("\nError listing the instances (machine/actuator.go 472): %v", err)
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

	expiration := time.Now().UTC().Add(options.TokenTTL)
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

func (oc *OpenstackClient) validateMachine(machine *machinev1.Machine, config *openstackconfigv1.OpenstackProviderSpec) *apierrors.MachineError {
	// TODO: other validate of openstackCloud
	return nil
}
