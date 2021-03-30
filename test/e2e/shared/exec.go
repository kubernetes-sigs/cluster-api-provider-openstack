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
	"time"

	. "github.com/onsi/ginkgo"
	"golang.org/x/crypto/ssh"
)

type server struct {
	name string
	id   string
	ip   string
}

type command struct {
	title string
	cmd   string
}

// commandsForMachine opens a terminal connection
// and executes the given commands, outputting the results to a file for each.
func commandsForMachine(ctx context.Context, debug bool, f *os.File, machineIP, bastionIP string, commands []command) {
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
		Timeout:         10 * time.Second,
	}
	cfg.SetDefaults()
	Debugf(debug, "dialing to bastion host %s", bastionIP)
	bastionConn, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", bastionIP), cfg)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "couldn't connect to bastion host %s: %s\n", bastionIP, err)
		return
	}
	defer bastionConn.Close()

	// Dial a connection to the service host, from the bastion host
	Debugf(debug, "dialing from bastion host %s to machine %s", bastionIP, machineIP)
	timeout, timeoutCancel := context.WithTimeout(ctx, 30*time.Second)
	defer timeoutCancel()
	go func() {
		<-timeout.Done()
		bastionConn.Close()
	}()
	conn, err := bastionConn.Dial("tcp", fmt.Sprintf("%s:22", machineIP))
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "couldn't connect from the bastion host %s to the target instance %s: %s\n", bastionIP, machineIP, err)
		return
	}
	defer conn.Close()

	cfg = &ssh.ClientConfig{
		User:            "capi",
		Auth:            []ssh.AuthMethod{ssh.Password("capi")},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil },
		Timeout:         10 * time.Second,
	}
	cfg.SetDefaults()
	Debugf(debug, "dialing to machine %s (via tunnel)", machineIP)
	clientConn, channels, reqs, err := ssh.NewClientConn(conn, machineIP, cfg)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "couldn't connect from local to the target instance %s: %s\n", machineIP, err)
		return
	}
	defer clientConn.Close()

	sshClient := ssh.NewClient(clientConn, channels, reqs)

	for _, c := range commands {
		Debugf(debug, "executing cmd %q on machine %s", c.cmd, machineIP)
		session, err := sshClient.NewSession()
		if err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "couldn't open session from local to the target instance %s: %s\n", machineIP, err)
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
			_, _ = fmt.Fprintf(f, "error writing log file: %s\n", err)
			continue
		}
		Debugf(debug, "finished executing cmd %q on machine %s", c.cmd, machineIP)
	}
}
