package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"

	"golang.org/x/crypto/ssh/agent"
)

// agent runs an SSH agent with the cluster's private key and spawns a shell
// with the environment set up to use the agent
func (s *sshTool) agent(ctx context.Context) error {
	key, err := x509.ParsePKCS1PrivateKey(s.oc.Properties.SSHKey)
	if err != nil {
		return err
	}

	keyring := agent.NewKeyring()

	err = keyring.Add(agent.AddedKey{
		PrivateKey: key,
	})
	if err != nil {
		return err
	}

	dir, err := ioutil.TempDir("", "sshtool-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	p := path.Join(dir, "agent")

	l, err := net.Listen("unix", p)
	if err != nil {
		return err
	}

	fmt.Printf("ssh -A -p 2200 core@%s\n", s.oc.Properties.NetworkProfile.PrivateEndpointIP)

	go func() {
		c := &exec.Cmd{
			Path:   "/bin/bash",
			Env:    append(os.Environ(), fmt.Sprintf("SSH_AUTH_SOCK=%s", p)),
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}

		c.Run()

		err = l.Close()
		if err != nil {
			s.log.Error(err)
		}
	}()

	for {
		c, err := l.Accept()
		if err != nil {
			break
		}

		go agent.ServeAgent(keyring, c)
	}

	return nil
}
