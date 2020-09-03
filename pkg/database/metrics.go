package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

func EmitQueueLengthMetrics(ctx context.Context, log *logrus.Entry, openShiftClusters OpenShiftClusters, m metrics.Interface) {
	defer recover.Panic(log)
	t := time.NewTicker(time.Minute)
	defer t.Stop()

	for range t.C {
		i, err := openShiftClusters.QueueLength(ctx)
		if err != nil {
			log.Error(err)
		} else {
			m.EmitGauge("database.openshiftclusters.queue.length", int64(i), nil)
		}
	}
}
