package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pgknamespace "github.com/Azure/ARO-RP/pkg/util/namespace"
)

func (mon *Monitor) emitDaemonsetsConditions(ctx context.Context) error {
	dss, err := mon.cli.AppsV1().DaemonSets("").List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ds := range dss.Items {
		if !pgknamespace.IsOpenShift(ds.Namespace) {
			continue
		}

		if ds.Status.DesiredNumberScheduled == ds.Status.NumberAvailable {
			continue
		}

		mon.emitGauge("daemonsets.conditions", 1, map[string]string{
			"desiredNumberScheduled": strconv.Itoa(int(ds.Status.DesiredNumberScheduled)),
			"name":                   ds.Name,
			"namespace":              ds.Namespace,
			"numberAvailable":        strconv.Itoa(int(ds.Status.NumberAvailable)),
		})
	}

	return nil
}
