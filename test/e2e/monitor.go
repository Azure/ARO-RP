package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/monitor/cluster"
)

var _ = Describe("Monitor", func() {
	Specify("a monitor run should not return any errors", func() {
		ctx := context.Background()

		mon, err := cluster.NewMonitor(ctx, log, clients.RestConfig, &api.OpenShiftCluster{
			ID: resourceIDFromEnv(),
		}, &noop.Noop{}, nil, true)
		Expect(err).NotTo(HaveOccurred())

		errs := mon.Monitor(ctx)
		Expect(errs).To(HaveLen(0))
	})
})
