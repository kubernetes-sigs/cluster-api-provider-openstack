/*
Copyright 2020 The Kubernetes Authors.

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

package compute

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/pkg/errors"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha3"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
)

const (
	TimeoutInstanceCreate       = 5
	RetryIntervalInstanceStatus = 10 * time.Second
)

func (s *Service) ReconcileBastion(clusterName string, cluster *clusterv1.Cluster, openStackCluster *infrav1.OpenStackCluster) (*Instance, error) {

	openStackMachine := &infrav1.OpenStackMachine{}
	openStackMachine.Name = fmt.Sprintf("%s-bastion", clusterName)

	instance, err := s.InstanceExists(openStackMachine)
	if err != nil {
		return nil, err
	}

	if instance == nil {
		instance, err = s.createBastion(clusterName, openStackCluster)
		if err != nil {
			return nil, errors.Errorf("error creating Openstack instance: %v", err)
		}
		instanceCreateTimeout := getTimeout("CLUSTER_API_OPENSTACK_INSTANCE_CREATE_TIMEOUT", TimeoutInstanceCreate)
		instanceCreateTimeout *= time.Minute
		err = util.PollImmediate(RetryIntervalInstanceStatus, instanceCreateTimeout, func() (bool, error) {
			instance, err = s.GetInstance(instance.ID)
			if err != nil {
				return false, nil
			}
			return instance.Status == "ACTIVE", nil
		})
		if err != nil {
			return nil, errors.Errorf("error creating Openstack instance: %v", err)
		}
	}

	return instance, nil
}

func (s *Service) DeleteBastion(clusterName string, openStackCluster *infrav1.OpenStackCluster) error {

	openStackMachine := &infrav1.OpenStackMachine{}
	openStackMachine.Name = fmt.Sprintf("%s-bastion", clusterName)

	instance, err := s.InstanceExists(openStackMachine)
	if err != nil {
		return err
	}
	if instance == nil {
		return nil
	}

	err = servers.Delete(s.computeClient, instance.ID).ExtractErr()
	if err != nil {
		record.Warnf(openStackCluster, "FailedDeleteServer", "Failed to delete bastion: %v", err)
		return err
	}
	record.Eventf(openStackCluster, "SuccessfulTerminate", "Terminated instance %q", instance.ID)
	return nil
}

func (s *Service) createBastion(clusterName string, openStackCluster *infrav1.OpenStackCluster) (*Instance, error) {

	name := fmt.Sprintf("%s-bastion", clusterName)
	// Get image ID
	imageID, err := getImageID(s, openStackCluster.Spec.Bastion.Image)
	if err != nil {
		return nil, fmt.Errorf("create new server err: %v", err)
	}
	flavorName := openStackCluster.Spec.Bastion.Flavor
	keyName := openStackCluster.Spec.Bastion.SSHKeyName
	networks := []servers.Network{}
	networks = append(networks,
		servers.Network{
			UUID: openStackCluster.Status.Network.ID,
		})
	securityGroups, err := getSecurityGroups(s, openStackCluster.Spec.Bastion.SecurityGroups)
	if err != nil {
		return nil, err
	}
	if openStackCluster.Spec.ManagedSecurityGroups {
		securityGroups = append(securityGroups, openStackCluster.Status.BastionSecurityGroup.ID)
	}

	var serverCreateOpts servers.CreateOptsBuilder = servers.CreateOpts{
		Name:           name,
		ImageRef:       imageID,
		FlavorName:     flavorName,
		Networks:       networks,
		SecurityGroups: securityGroups,
		ServiceClient:  s.computeClient,
	}

	server, err := servers.Create(s.computeClient, keypairs.CreateOptsExt{
		CreateOptsBuilder: serverCreateOpts,
		KeyName:           keyName,
	}).Extract()
	if err != nil {
		record.Warnf(openStackCluster, "FailedCreateServer", "Failed to create bastion: %v", err)
		return nil, fmt.Errorf("create new server err: %v", err)
	}
	record.Eventf(openStackCluster, "SuccessfulCreateServer", "Created server %s with id %s", name, server.ID)

	return &Instance{Server: *server, State: infrav1.InstanceState(server.Status)}, nil

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
