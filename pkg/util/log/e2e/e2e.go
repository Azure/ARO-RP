package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/metrics"
	e2estatsd "github.com/Azure/ARO-RP/pkg/metrics/statsd/e2e"
)

type e2EHook struct {
	logToMetrics e2estatsd.E2ELogToMetrics
}

func NewE2EHook(m metrics.Emitter) e2EHook {
	return e2EHook{
		logToMetrics: e2estatsd.NewE2ELogToMetrics(m),
	}
}

func (e2EHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *e2EHook) Fire(entry *logrus.Entry) error {
	h.logToMetrics.PostMetricsFromLogEntry(entry)
	return nil
}
