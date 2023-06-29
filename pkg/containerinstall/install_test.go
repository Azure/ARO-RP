package containerinstall

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/containers/podman/v4/pkg/bindings/containers"
	"github.com/containers/podman/v4/pkg/bindings/images"
	"github.com/containers/podman/v4/pkg/specgen"
	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Azure/ARO-RP/pkg/util/uuid"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

const TEST_PULLSPEC = "registry.access.redhat.com/ubi8/go-toolset:1.18.4"

var _ = Describe("Podman", Ordered, func() {
	var err error
	var conn context.Context
	var hook *test.Hook
	var log *logrus.Entry
	var containerName string
	var containerID string

	BeforeAll(func(ctx context.Context) {
		var err error
		conn, err = getConnection(ctx)
		if err != nil {
			Skip("unable to access podman: %v")
		}

		hook, log = testlog.New()
		containerName = uuid.DefaultGenerator.Generate()
	})

	It("can pull images", func() {
		_, err = images.Pull(conn, TEST_PULLSPEC, &images.PullOptions{Policy: to.StringPtr("missing")})
		Expect(err).To(BeNil())
	})

	It("can start a container", func() {
		s := specgen.NewSpecGenerator(TEST_PULLSPEC, false)
		s.Name = containerName
		s.Entrypoint = []string{"/bin/bash", "-c", "echo 'hello'"}

		containerID, err = runContainer(conn, log, s)
		Expect(err).To(BeNil())
	})

	It("can wait for completion", func() {
		exit, err := containers.Wait(conn, containerID, nil)
		Expect(err).To(BeNil())
		Expect(exit).To(Equal(0), "exit code was %d, not 0", exit)
	})

	It("can fetch container logs", func() {
		err = getContainerLogs(conn, log, containerID)
		Expect(err).To(BeNil())

		entries := []map[string]types.GomegaMatcher{
			{
				"msg":   Equal("created container " + containerName + " with ID " + containerID),
				"level": Equal(logrus.InfoLevel),
			},
			{
				"msg":   Equal("started container " + containerID),
				"level": Equal(logrus.InfoLevel),
			},
			{
				"msg":   Equal("stdout: hello\n"),
				"level": Equal(logrus.InfoLevel),
			},
		}

		err = testlog.AssertLoggingOutput(hook, entries)
		Expect(err).To(BeNil())
	})

	AfterAll(func() {
		if containerID != "" {
			_, err = containers.Remove(conn, containerID, &containers.RemoveOptions{Force: to.BoolPtr(true)})
			Expect(err).To(BeNil())
		}
	})
})

func TestContainerInstall(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ContainerInstall Suite")
}
