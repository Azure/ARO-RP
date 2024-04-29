package containerinstall

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/containers/podman/v4/pkg/bindings"
	"github.com/containers/podman/v4/pkg/bindings/containers"
	"github.com/containers/podman/v4/pkg/specgen"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
)

func getConnection(ctx context.Context, _env env.Core) (context.Context, error) {
	socket := _env.GetEnv("ARO_PODMAN_SOCKET")

	if socket == "" {
		sock_dir := _env.GetEnv("XDG_RUNTIME_DIR")
		socket = "unix:" + sock_dir + "/podman/podman.sock"
	}

	return bindings.NewConnection(ctx, socket)
}

func getContainerLogs(ctx context.Context, log *logrus.Entry, containerName string) error {
	stdout, stderr := make(chan string, 1024), make(chan string, 1024)
	go func() {
		for v := range stdout {
			log.Infof("stdout: %s", v)
		}
	}()

	go func() {
		for v := range stderr {
			log.Errorf("stderr: %s", v)
		}
	}()
	err := containers.Logs(
		ctx,
		containerName,
		(&containers.LogOptions{}).WithStderr(true).WithStdout(true),
		stdout,
		stderr,
	)
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
