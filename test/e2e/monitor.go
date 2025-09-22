package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/monitor/cluster"
)

var _ = Describe("Monitor", func() {
	// This is more of an integration test rather than E2E.
	It("must run and must not return any errors", func(ctx context.Context) {
		By("creating a new monitor instance for the test cluster")
		mon, err := cluster.NewMonitor(log, clients.RestConfig, &api.OpenShiftCluster{
			ID: resourceIDFromEnv(),
		}, nil, "", &noop.Noop{}, true)
		Expect(err).NotTo(HaveOccurred())

		By("running the monitor once")
		err = mon.Monitor(ctx)
		Expect(err).NotTo(HaveOccurred())
	})
})
