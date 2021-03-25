// +build e2e

/*
Copyright 2021 The Kubernetes Authors.

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

package shared

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	. "github.com/onsi/ginkgo"
	"golang.org/x/crypto/ssh"

	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/compute"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/provider"
)

type instance struct {
	name string
	id   string
	ip   string
}

// allMachines gets all OpenStack servers at once, to save on DescribeInstances
// calls
func allMachines(_ context.Context, e2eCtx *E2EContext) ([]instance, error) {
	openStackCloudYAMLFile := e2eCtx.E2EConfig.GetVariable(OpenStackCloudYAMLFile)
	openstackCloud := e2eCtx.E2EConfig.GetVariable(OpenStackCloud)

	clouds := getParsedOpenStackCloudYAML(openStackCloudYAMLFile)
	cloud := clouds.Clouds[openstackCloud]

	providerClient, clientOpts, err := provider.NewClient(cloud, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating provider client: %v", err)
	}

	computeClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{Region: clientOpts.RegionName})
	if err != nil {
		return nil, fmt.Errorf("error creating compute client: %v", err)
	}

	serverListOpts := &servers.ListOpts{}
	allPages, err := servers.List(computeClient, serverListOpts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("error listing server: %v", err)
	}

	serverList, err := servers.ExtractServers(allPages)
	if err != nil {
		return nil, fmt.Errorf("error extracting server: %v", err)
	}

	instances := make([]instance, len(serverList))
	for _, server := range serverList {
		addrMap, err := compute.GetIPFromInstance(server)
		if err != nil {
			return nil, fmt.Errorf("error getting ip for server %s: %v", server.Name, err)
		}
		ip, ok := addrMap["internal"]
		if !ok {
			return nil, fmt.Errorf("error geting internal ip for server %s: %v", server.Name, err)
		}

		instances = append(instances, instance{
			name: server.Name,
			id:   server.ID,
			ip:   ip,
		})
	}
	return instances, nil
}

type command struct {
	title string
	cmd   string
}

// commandsForMachine opens a terminal connection
// and executes the given commands, outputting the results to a file for each.
func commandsForMachine(f *os.File, machineIP, bastionIP string, commands []command) {
	// TODO(sbuerin) try to access via ssh key pair as soon as it's clear how to do that
	// Issue: https://github.com/kubernetes-sigs/cluster-api-provider-openstack/issues/784
	//providerClient, clientOpts, err := getProviderClient(e2eCtx)
	//computeClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{
	//	Region: clientOpts.RegionName,
	//})
	//keyPair, err := keypairs.Get(computeClient, DefaultSSHKeyPairName).Extract()
	// Get a signer from the private key file
	//signer, err := ssh.ParsePrivateKey([]byte(keyPair.PrivateKey))

	cfg := &ssh.ClientConfig{
		User:            "cirros",
		Auth:            []ssh.AuthMethod{ssh.Password("gocubsgo")},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil },
	}
	cfg.SetDefaults()

	// connect to the bastion host
	bastionConn, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", bastionIP), cfg)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "couldn't connect to bastion host %s: %s", bastionIP, err)
		return
	}
	defer bastionConn.Close()

	// Dial a connection to the service host, from the bastion host
	conn, err := bastionConn.Dial("tcp", fmt.Sprintf("%s:22", machineIP))
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "couldn't connect from the bastion host %s to the target instance %s: %s", bastionIP, machineIP, err)
		return
	}
	defer conn.Close()

	// connect to the machineInstance via hte bastion host
	cfg = &ssh.ClientConfig{
		User:            "capi",
		Auth:            []ssh.AuthMethod{ssh.Password("capi")},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil },
	}
	cfg.SetDefaults()
	clientConn, channels, reqs, err := ssh.NewClientConn(conn, machineIP, cfg)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "couldn't connect from local to the target instance %s: %s", machineIP, err)
		return
	}
	defer clientConn.Close()

	sshClient := ssh.NewClient(clientConn, channels, reqs)

	for _, c := range commands {
		session, err := sshClient.NewSession()
		if err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "couldn't open session from local to the target instance %s: %s", machineIP, err)
			continue
		}
		defer session.Close()

		logFile := path.Join(filepath.Dir(f.Name()), c.title+".log")
		var stdoutBuf bytes.Buffer
		var stderrBuf bytes.Buffer
		session.Stdout = &stdoutBuf
		session.Stderr = &stderrBuf
		if err := session.Run("sudo " + c.cmd + "\n"); err != nil {
			_, _ = fmt.Fprintf(f, "unable to send command %q: %s\n", c.cmd, err)
			continue
		}
		result := strings.TrimSuffix(stdoutBuf.String(), "\n") + "\n" + strings.TrimSuffix(stderrBuf.String(), "\n")
		if err := os.WriteFile(logFile, []byte(result), os.ModePerm); err != nil {
			_, _ = fmt.Fprintf(f, "error writing log file: %s", err)
			continue
		}
	}
}
