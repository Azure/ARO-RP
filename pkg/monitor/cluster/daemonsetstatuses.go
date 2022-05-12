package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/namespace"
)

func (mon *Monitor) emitDaemonsetStatuses(ctx context.Context) error {
	var cont string
	var count int64
	for {
		dss, err := mon.cli.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{Limit: 500, Continue: cont})
		if err != nil {
			return err
		}

		count += int64(len(dss.Items))

		for _, ds := range dss.Items {
			if !namespace.IsOpenShiftNamespace(ds.Namespace) {
				continue
			}

			if ds.Status.DesiredNumberScheduled == ds.Status.NumberAvailable {
				continue
			}

			mon.emitGauge("daemonset.statuses", 1, map[string]string{
				"desiredNumberScheduled": strconv.Itoa(int(ds.Status.DesiredNumberScheduled)),
				"name":                   ds.Name,
				"namespace":              ds.Namespace,
				"numberAvailable":        strconv.Itoa(int(ds.Status.NumberAvailable)),
			})
		}

		cont = dss.Continue
		if cont == "" {
			break
		}
	}

	mon.emitGauge("daemonset.count", count, nil)

	return nil
}
