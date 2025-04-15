package containerinstall

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/containers/podman/v5/libpod/define"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/bindings/images"
	"github.com/containers/podman/v5/pkg/bindings/secrets"
	"github.com/containers/podman/v5/pkg/domain/entities"
	"github.com/containers/podman/v5/pkg/specgen"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Azure/ARO-RP/pkg/util/uuid"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

const TEST_PULLSPEC = "registry.access.redhat.com/golang-builder--partner-share:rhel-9-golang-1.22-openshift-4.19"

var _ = Describe("Podman", Ordered, func() {
	var err error
	var cancel context.CancelFunc
	var conn context.Context
	var hook *test.Hook
	var log *logrus.Entry
	var containerName string
	var containerID string
	var secret *entities.SecretCreateReport

	BeforeAll(func() {
		var err error
		var outerconn context.Context
		outerconn, cancel = context.WithCancel(context.Background())
		conn, err = getConnection(outerconn)
		if err != nil {
			Skip("unable to access podman: %v")
		}

		hook, log = testlog.New()
		containerName = uuid.DefaultGenerator.Generate()
	})

	It("can pull images", func() {
		_, err = images.Pull(conn, TEST_PULLSPEC, (&images.PullOptions{}).WithPolicy("missing"))
		Expect(err).ToNot(HaveOccurred())
	})

	It("can create a secret", func() {
		secret, err = secrets.Create(
			conn, bytes.NewBufferString("hello\n"),
			(&secrets.CreateOptions{}).WithName(containerName))
		Expect(err).ToNot(HaveOccurred())
	})

	It("can start a container", func() {
		s := specgen.NewSpecGenerator(TEST_PULLSPEC, false)
		s.Name = containerName
		s.Secrets = []specgen.Secret{
			{
				Source: containerName,
				Target: "/.azure/testfile",
				Mode:   0o644,
			},
		}
		s.Mounts = append(s.Mounts, specs.Mount{
			Destination: "/.azure",
			Type:        "tmpfs",
			Source:      "",
		})
		s.WorkDir = "/.azure"
		s.Entrypoint = []string{"/bin/bash", "-c", "cat testfile"}

		containerID, err = runContainer(conn, log, s)
		Expect(err).ToNot(HaveOccurred())
	})

	It("can wait for completion", func() {
		exit, err := containers.Wait(conn, containerID, (&containers.WaitOptions{}).WithCondition([]define.ContainerStatus{define.ContainerStateExited}))
		Expect(err).ToNot(HaveOccurred())
		Expect(exit).To(BeEquivalentTo(0), "exit code was %d, not 0", exit)
	})

	It("can fetch container logs", func() {
		// sometimes logs take a few seconds to flush to disk, so just retry for 10s
		Eventually(func(g Gomega) {
			hook.Reset()
			err = getContainerLogs(conn, log, containerID)
			g.Expect(err).ToNot(HaveOccurred())
			entries := []map[string]types.GomegaMatcher{
				{
					"msg":   Equal("stdout: hello\n"),
					"level": Equal(logrus.InfoLevel),
				},
			}

			err = testlog.AssertLoggingOutput(hook, entries)
			g.Expect(err).ToNot(HaveOccurred())
		}).WithTimeout(10 * time.Second).WithPolling(time.Second).Should(Succeed())
	})

	AfterAll(func() {
		if containerID != "" {
			_, err = containers.Remove(conn, containerID, (&containers.RemoveOptions{}).WithForce(true))
			Expect(err).ToNot(HaveOccurred())
		}

		if secret != nil {
			err = secrets.Remove(conn, secret.ID)
			Expect(err).ToNot(HaveOccurred())
		}

		if cancel != nil {
			cancel()
		}
	})
})

func TestContainerInstall(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ContainerInstall Suite")
}
