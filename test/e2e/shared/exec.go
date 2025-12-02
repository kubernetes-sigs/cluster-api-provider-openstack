//go:build e2e

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

	. "github.com/onsi/ginkgo/v2"
	"golang.org/x/crypto/ssh"
)

type command struct {
	title string
	cmd   string
}

type commandParameter struct {
	signer      ssh.Signer
	debug       bool
	logDir      string
	bastionIP   string
	machineIP   string
	machineUser string
	cmd         command
}

// executeCommands opens a terminal connection
// and executes the given commands, outputting the results to a file for each.
func executeCommands(ctx context.Context, artifactsFolder string, debug bool, logDir, machineIP, bastionIP, machineUser string, commands []command) {
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
		if err := executeCommand(ctx, commandParameter{
			signer:      signer,
			debug:       debug,
			logDir:      logDir,
			bastionIP:   bastionIP,
			machineIP:   machineIP,
			machineUser: machineUser,
			cmd:         cmd,
		}); err != nil {
			_, _ = fmt.Fprintln(GinkgoWriter, err.Error())
		}
	}
}

func executeCommand(ctx context.Context, p commandParameter) error {
	cfg := &ssh.ClientConfig{
		User: "cirros",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(p.signer),
		},
		HostKeyCallback: func(string, net.Addr, ssh.PublicKey) error { return nil },
		Timeout:         60 * time.Second,
	}
	cfg.SetDefaults()
	Debugf(p.debug, "dialing from local to bastion host %s", p.bastionIP)
	bastionConn, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", p.bastionIP), cfg)
	if err != nil {
		return fmt.Errorf("couldn't dial from local to bastion host %s: %s", p.bastionIP, err)
	}
	defer bastionConn.Close()

	// Dial a connection to the service host, from the bastion host
	Debugf(p.debug, "dialing from bastion host %s to machine %s", p.bastionIP, p.machineIP)
	// we have to timeout this connection
	// * there is no way to set a timeout in the Dial func
	// * sometimes the server are deleted when we try this and we would be stuck infinitely
	timeout, timeoutCancel := context.WithTimeout(ctx, 60*time.Second)
	defer timeoutCancel()
	go func() {
		<-timeout.Done()
		bastionConn.Close()
	}()
	conn, err := bastionConn.Dial("tcp", fmt.Sprintf("%s:22", p.machineIP))
	if err != nil {
		return fmt.Errorf("couldn't dial from bastion host %s to machine %s: %s", p.bastionIP, p.machineIP, err)
	}
	defer conn.Close()

	cfg = &ssh.ClientConfig{
		User: p.machineUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(p.signer),
		},
		HostKeyCallback: func(string, net.Addr, ssh.PublicKey) error { return nil },
		Timeout:         60 * time.Second,
	}
	cfg.SetDefaults()
	Debugf(p.debug, "dialing from local to machine %s (via tunnel)", p.machineIP)
	clientConn, channels, reqs, err := ssh.NewClientConn(conn, p.machineIP, cfg)
	if err != nil {
		return fmt.Errorf("couldn't dial from local to machine %s: %s", p.machineIP, err)
	}
	defer clientConn.Close()

	sshClient := ssh.NewClient(clientConn, channels, reqs)

	Debugf(p.debug, "executing cmd %q on machine %s", p.cmd.cmd, p.machineIP)
	session, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("couldn't open session from local to machine %s to execute cmd %q: %s", p.cmd.cmd, p.machineIP, err)
	}
	defer session.Close()

	logFile := path.Join(p.logDir, p.cmd.title+".log")
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf
	if err := session.Run("sudo " + p.cmd.cmd + "\n"); err != nil {
		return fmt.Errorf("unable to send command %q: %s", "sudo "+p.cmd.cmd, err)
	}
	result := strings.TrimSuffix(stdoutBuf.String(), "\n") + "\n" + strings.TrimSuffix(stderrBuf.String(), "\n")
	if err := os.WriteFile(logFile, []byte(result), 0o600); err != nil {
		return fmt.Errorf("error writing log file: %s", err)
	}
	Debugf(p.debug, "finished executing cmd %q on machine %s", p.cmd.cmd, p.machineIP)

	return nil
}
