package containerinstall

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/containers/podman/v4/pkg/bindings/containers"
	"github.com/containers/podman/v4/pkg/specgen"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/uuid"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

const TEST_PULLSPEC = "registry.access.redhat.com/ubi8/go-toolset:1.18.4"

func TestPodman(t *testing.T) {
	ctx := context.Background()
	conn, err := getConnection(ctx, true)
	if err != nil {
		t.Skipf("unable to access podman: %v", err)
	}

	hook, log := testlog.New()
	randomName := uuid.DefaultGenerator.Generate()

	s := specgen.NewSpecGenerator(TEST_PULLSPEC, false)
	s.Name = randomName
	s.Entrypoint = []string{"/bin/bash", "-c", "echo 'hello'"}

	err = pullContainer(conn, TEST_PULLSPEC, "missing")
	if err != nil {
		t.Fatal(err)
	}

	id, err := runContainer(conn, log, s)
	if err != nil {
		t.Fatal(err)
	}

	exit, err := containers.Wait(conn, id, nil)
	if err != nil {
		t.Error(err)
	}
	if exit != 0 {
		t.Errorf("exit code was %d, not 0", exit)
	}

	err = getContainerLogs(conn, log, id)
	if err != nil {
		t.Error(err)
	}

	_, err = containers.Remove(conn, id, &containers.RemoveOptions{Force: to.BoolPtr(true)})
	if err != nil {
		t.Error(err)
	}

	entries := []map[string]types.GomegaMatcher{

		{
			"msg":   gomega.Equal("created container " + randomName + " with ID " + id),
			"level": gomega.Equal(logrus.InfoLevel),
		},
		{
			"msg":   gomega.Equal("started container " + id),
			"level": gomega.Equal(logrus.InfoLevel),
		},
		{
			"msg":   gomega.Equal("stdout: hello\n"),
			"level": gomega.Equal(logrus.InfoLevel),
		},
	}

	err = testlog.AssertLoggingOutput(hook, entries)
	if err != nil {
		t.Fatal(err)
	}
}
