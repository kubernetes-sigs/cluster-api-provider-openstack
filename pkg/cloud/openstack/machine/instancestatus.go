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
	"bytes"
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/serializer/json"

	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Long term, we should retrieve the current status by asking k8s, openstack etc. for all the needed info.
// For now, it is stored in the matching CRD under an annotation. This is similar to
// the spec and status concept where the machine CRD is the instance spec and the annotation is the instance status.

const InstanceStatusAnnotationKey = "instance-status"

type instanceStatus *machinev1.Machine

// Get the status of the instance identified by the given machine
func (oc *OpenstackClient) instanceStatus(machine *machinev1.Machine) (instanceStatus, error) {
	currentMachine, err := GetMachineIfExists(oc.client, machine.Namespace, machine.Name)

	if err != nil {
		return nil, err
	}

	if currentMachine == nil {
		// The current status no longer exists because the matching CRD has been deleted (or does not exist yet ie. bootstrapping)
		return nil, nil
	}
	return oc.machineInstanceStatus(currentMachine)
}

// Get a `machinev1.Machine` matching the specified name and namespace.
//
// Same as cluster-api's `util.GetMachineIfExists`, but works with
// `machinev1` instead of `clusterv1`. The latter does not work with
// OpenShift deployments.
func GetMachineIfExists(c client.Client, namespace, name string) (*machinev1.Machine, error) {
	if c == nil {
		// Being called before k8s is setup as part of control plane VM creation
		return nil, nil
	}

	machine := &machinev1.Machine{}
	key := client.ObjectKey{Namespace: namespace, Name: name}
	err := c.Get(context.Background(), key, machine)

	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return machine, nil
}

// Sets the status of the instance identified by the given machine to the given machine
func (oc *OpenstackClient) updateInstanceStatus(machine *machinev1.Machine) error {
	status := instanceStatus(machine)
	currentMachine, err := GetMachineIfExists(oc.client, machine.Namespace, machine.Name)
	if err != nil {
		return err
	}

	if currentMachine == nil {
		// The current status no longer exists because the matching CRD has been deleted.
		return fmt.Errorf("Machine has already been deleted. Cannot update current instance status for machine %v", machine.ObjectMeta.Name)
	}

	m, err := oc.setMachineInstanceStatus(currentMachine, status)
	if err != nil {
		return err
	}

	return oc.client.Update(context.TODO(), m)
}

// Gets the state of the instance stored on the given machine CRD
func (oc *OpenstackClient) machineInstanceStatus(machine *machinev1.Machine) (instanceStatus, error) {
	if machine.ObjectMeta.Annotations == nil {
		// No state
		return nil, nil
	}

	a := machine.ObjectMeta.Annotations[InstanceStatusAnnotationKey]
	if a == "" {
		// No state
		return nil, nil
	}

	serializer := json.NewSerializer(json.DefaultMetaFactory, oc.scheme, oc.scheme, false)
	var status machinev1.Machine
	_, _, err := serializer.Decode([]byte(a), &schema.GroupVersionKind{Group: "machine.openshift.io", Version: "v1beta1", Kind: "Machine"}, &status)
	if err != nil {
		return nil, fmt.Errorf("decoding failure: %v", err)
	}

	return instanceStatus(&status), nil
}

// Applies the state of an instance onto a given machine CRD
func (oc *OpenstackClient) setMachineInstanceStatus(machine *machinev1.Machine, status instanceStatus) (*machinev1.Machine, error) {
	// Avoid status within status within status ...
	status.ObjectMeta.Annotations[InstanceStatusAnnotationKey] = ""

	serializer := json.NewSerializer(json.DefaultMetaFactory, oc.scheme, oc.scheme, false)
	b := []byte{}
	buff := bytes.NewBuffer(b)
	err := serializer.Encode((*machinev1.Machine)(status), buff)
	if err != nil {
		return nil, fmt.Errorf("encoding failure: %v", err)
	}

	if machine.ObjectMeta.Annotations == nil {
		machine.ObjectMeta.Annotations = make(map[string]string)
	}
	machine.ObjectMeta.Annotations[InstanceStatusAnnotationKey] = buff.String()
	return machine, nil
}
