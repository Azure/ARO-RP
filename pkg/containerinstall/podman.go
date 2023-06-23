package containerinstall

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"

	"github.com/containers/podman/v4/pkg/bindings"
	"github.com/containers/podman/v4/pkg/bindings/containers"
	"github.com/containers/podman/v4/pkg/specgen"
	"github.com/sirupsen/logrus"
)

func getConnection(ctx context.Context) (context.Context, error) {
	socket := os.Getenv("ARO_PODMAN_SOCKET")

	if socket == "" {
		sock_dir := os.Getenv("XDG_RUNTIME_DIR")
		socket = "unix:" + sock_dir + "/podman/podman.sock"
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

func runContainer(ctx context.Context, log *logrus.Entry, s *specgen.SpecGenerator) (string, error) {
	container, err := containers.CreateWithSpec(ctx, s, nil)
	if err != nil {
		return "", err
	}
	log.Infof("created container %s with ID %s", s.Name, container.ID)

	err = containers.Start(ctx, container.ID, nil)
	if err != nil {
		return container.ID, err
	}
	log.Infof("started container %s", container.ID)
	return container.ID, nil
}
