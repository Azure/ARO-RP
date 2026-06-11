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
	"otel-collector-master": {},
	"otel-collector-worker": {},
}

func (mon *Monitor) emitDaemonsetStatuses(ctx context.Context) error {
	// Only fetch in the namespaces we manage
	for _, ns := range mon.namespacesToMonitor {
		var cont string
		l := &appsv1.DaemonSetList{}

		for {
			err := mon.ocpclientset.List(ctx, l, client.InNamespace(ns), client.Continue(cont), client.Limit(mon.queryLimit))
			if err != nil {
				return fmt.Errorf("error in list operation: %w", err)
			}

			for _, ds := range l.Items {
				if ds.Status.DesiredNumberScheduled == ds.Status.NumberAvailable {
					continue
				}

				mon.emitGauge("daemonset.statuses", 1, map[string]string{
					"desiredNumberScheduled": strconv.Itoa(int(ds.Status.DesiredNumberScheduled)),
					"name":                   ds.Name,
					"namespace":              ds.Namespace,
					"numberAvailable":        strconv.Itoa(int(ds.Status.NumberAvailable)),
				})

				if isGenevaLoggingOTelDaemonSet(ds) && ds.Status.DesiredNumberScheduled > 0 && ds.Status.NumberAvailable == 0 {
					mon.emitGauge("genevalogging.otel.cannotstart", 1, map[string]string{
						"desiredNumberScheduled": strconv.Itoa(int(ds.Status.DesiredNumberScheduled)),
						"name":                   ds.Name,
						"namespace":              ds.Namespace,
						"numberAvailable":        strconv.Itoa(int(ds.Status.NumberAvailable)),
					})
				}
			}

			cont = l.Continue
			if cont == "" {
				break
			}
		}
	}
	return nil
}

func isGenevaLoggingOTelDaemonSet(ds appsv1.DaemonSet) bool {
	if ds.Namespace != genevaLoggingNamespace {
		return false
	}

	_, ok := genevaLoggingOTelDaemonSets[ds.Name]
	return ok
}
