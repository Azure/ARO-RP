package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const genevaLoggingNamespace = "openshift-azure-logging"

var genevaLoggingOTelDaemonSets = map[string]struct{}{
	"otel-exporter-master": {},
	"otel-exporter-worker": {},
}

func (mon *Monitor) emitDaemonsetStatuses(ctx context.Context) error {
	for name := range genevaLoggingOTelDaemonSets {
		var ds appsv1.DaemonSet
		err := mon.ocpclientset.Get(ctx, client.ObjectKey{Namespace: genevaLoggingNamespace, Name: name}, &ds)
		if err != nil {
			if client.IgnoreNotFound(err) == nil {
				continue
			}
			return fmt.Errorf("error getting DaemonSet %s/%s: %w", genevaLoggingNamespace, name, err)
		}

		if ds.Status.DesiredNumberScheduled == ds.Status.NumberAvailable {
			continue
		}

		dimensions := map[string]string{
			"desiredNumberScheduled": strconv.Itoa(int(ds.Status.DesiredNumberScheduled)),
			"name":                   ds.Name,
			"namespace":              ds.Namespace,
			"numberAvailable":        strconv.Itoa(int(ds.Status.NumberAvailable)),
		}

		mon.emitGauge("daemonset.statuses", 1, dimensions)

		if ds.Status.NumberAvailable == 0 {
			mon.emitGauge("genevalogging.otel.cannotstart", 1, dimensions)
		}
	}
	return nil
}
