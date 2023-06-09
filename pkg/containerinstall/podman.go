package containerinstall

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"os"

	"github.com/containers/podman/v4/pkg/bindings"
	"github.com/containers/podman/v4/pkg/bindings/containers"
	"github.com/containers/podman/v4/pkg/bindings/images"
	"github.com/containers/podman/v4/pkg/specgen"
	"github.com/sirupsen/logrus"
)

func getConnection(ctx context.Context, isDevelopment bool) (context.Context, error) {
	socket := os.Getenv("ARO_PODMAN_SOCKET")

	if socket == "" && isDevelopment {
		sock_dir := os.Getenv("XDG_RUNTIME_DIR")
		socket = "unix:" + sock_dir + "/podman/podman.sock"
	}

	if socket == "" {
		return nil, errors.New("no podman socket defined")
	}

	return bindings.NewConnection(ctx, socket)
}

func getContainerLogs(ctx context.Context, log *logrus.Entry, containerName string) error {
	stdout, stderr := make(chan string, 1024), make(chan string, 1024)
	go func() {
		for {
			v, ok := <-stdout
			if !ok {
				return
			}
			log.Infof("stdout: %s", v)
		}
	}()

	go func() {
		for {
			v, ok := <-stderr
			if !ok {
				return
			}
			log.Errorf("stderr: %s", v)
		}
	}()
	err := containers.Logs(ctx, containerName, nil, stdout, stderr)
	return err
}

func pullContainer(ctx context.Context, pullspec string, options *images.PullOptions) error {
	_, err := images.Pull(ctx, pullspec, options)
	return err
}

func runContainer(ctx context.Context, log *logrus.Entry, s *specgen.SpecGenerator) (string, error) {
	createResponse, err := containers.CreateWithSpec(ctx, s, nil)
	if err != nil {
		return "", err
	}
	log.Infof("created container %s with ID %s", s.Name, createResponse.ID)

	err = containers.Start(ctx, createResponse.ID, nil)
	if err != nil {
		return createResponse.ID, err
	}
	log.Infof("started container %s", createResponse.ID)
	return createResponse.ID, nil
}
