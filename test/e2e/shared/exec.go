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

// executeCommands opens a terminal connection
// and executes the given commands, outputting the results to a file for each.
func executeCommands(ctx context.Context, artifactsFolder string, debug bool, logDir, machineIP, bastionIP string, commands []command) {
	privateKey, err := os.ReadFile(filepath.Join(artifactsFolder, "ssh", DefaultSSHKeyPairName))
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "could not load private key from artifacts folder: %s\n", err)
		return
	}
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "could not parse private key from artifacts folder: %s\n", err)
		return
	}

	for _, cmd := range commands {
		if err := executeCommand(ctx, signer, debug, logDir, bastionIP, machineIP, cmd); err != nil {
			_, _ = fmt.Fprintln(GinkgoWriter, err.Error())
		}
	}
}

func executeCommand(ctx context.Context, signer ssh.Signer, debug bool, logDir, bastionIP, machineIP string, cmd command) error {
	cfg := &ssh.ClientConfig{
		User: "cirros",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil },
		Timeout:         60 * time.Second,
	}
	cfg.SetDefaults()
	Debugf(debug, "dialing from local to bastion host %s", bastionIP)
	bastionConn, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", bastionIP), cfg)
	if err != nil {
		return fmt.Errorf("couldn't dial from local to bastion host %s: %s", bastionIP, err)
	}
	defer bastionConn.Close()

	// Dial a connection to the service host, from the bastion host
	Debugf(debug, "dialing from bastion host %s to machine %s", bastionIP, machineIP)
	// we have to timeout this connection
	// * there is no way to set a timeout in the Dial func
	// * sometimes the server are deleted when we try this and we would be stuck infinitely
	timeout, timeoutCancel := context.WithTimeout(ctx, 60*time.Second)
	defer timeoutCancel()
	go func() {
		<-timeout.Done()
		bastionConn.Close()
	}()
	conn, err := bastionConn.Dial("tcp", fmt.Sprintf("%s:22", machineIP))
	if err != nil {
		return fmt.Errorf("couldn't dial from bastion host %s to machine %s: %s", bastionIP, machineIP, err)
	}
	defer conn.Close()

	cfg = &ssh.ClientConfig{
		User: "ubuntu",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil },
		Timeout:         60 * time.Second,
	}
	cfg.SetDefaults()
	Debugf(debug, "dialing from local to machine %s (via tunnel)", machineIP)
	clientConn, channels, reqs, err := ssh.NewClientConn(conn, machineIP, cfg)
	if err != nil {
		return fmt.Errorf("couldn't dial from local to machine %s: %s", machineIP, err)
	}
	defer clientConn.Close()

	sshClient := ssh.NewClient(clientConn, channels, reqs)

	Debugf(debug, "executing cmd %q on machine %s", cmd.cmd, machineIP)
	session, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("couldn't open session from local to machine %s to execute cmd %q: %s", cmd.cmd, machineIP, err)
	}
	defer session.Close()

	logFile := path.Join(logDir, cmd.title+".log")
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf
	if err := session.Run("sudo " + cmd.cmd + "\n"); err != nil {
		return fmt.Errorf("unable to send command %q: %s", "sudo "+cmd.cmd, err)
	}
	result := strings.TrimSuffix(stdoutBuf.String(), "\n") + "\n" + strings.TrimSuffix(stderrBuf.String(), "\n")
	if err := os.WriteFile(logFile, []byte(result), os.ModePerm); err != nil {
		return fmt.Errorf("error writing log file: %s", err)
	}
	Debugf(debug, "finished executing cmd %q on machine %s", cmd.cmd, machineIP)

	return nil
}
