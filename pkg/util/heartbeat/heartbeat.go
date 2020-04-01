package heartbeat

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

// EmitHeartbeat sends a heartbeat metric every 60 seconds
func EmitHeartbeat(log *logrus.Entry, m metrics.Interface, metricName string, stop <-chan struct{}, checkFunc func() bool) {
	defer recover.Panic(log)

	t := time.NewTicker(time.Minute)
	defer t.Stop()

	log.Print("starting heartbeat")

	for {
		select {
		case <-t.C:
			if checkFunc() {
				m.EmitGauge(metricName, 1, nil)
			}

		case <-stop:
			return
		}
	}
}
