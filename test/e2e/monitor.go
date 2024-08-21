package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"

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
		var wg sync.WaitGroup
		wg.Add(1)
		mon, err := cluster.NewMonitor(log, clients.RestConfig, &api.OpenShiftCluster{
			ID: resourceIDFromEnv(),
		}, &api.OpenShiftClusterDocument{
			ID: resourceIDFromEnv(),
		}, &noop.Noop{}, nil, true, &wg, nil)
		Expect(err).NotTo(HaveOccurred())

		By("running the monitor once")
		errs := mon.Monitor(ctx)
		Expect(errs).To(BeEmpty())
	})
})
