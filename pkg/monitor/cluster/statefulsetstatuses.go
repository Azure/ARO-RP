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

func (mon *Monitor) emitStatefulsetStatuses(ctx context.Context) error {
	// Only fetch in the namespaces we manage
	for _, ns := range mon.namespacesToMonitor {
		var cont string
		l := &appsv1.StatefulSetList{}

		for {
			err := mon.ocpclientset.List(ctx, l, client.InNamespace(ns), client.Continue(cont), client.Limit(mon.queryLimit))
			if err != nil {
				return fmt.Errorf("error in list operation: %w", err)
			}

			for _, ss := range l.Items {
				if ss.Status.Replicas == ss.Status.ReadyReplicas {
					continue
				}

				mon.emitGauge("statefulset.statuses", 1, map[string]string{
					"name":          ss.Name,
					"namespace":     ss.Namespace,
					"replicas":      strconv.Itoa(int(ss.Status.Replicas)),
					"readyReplicas": strconv.Itoa(int(ss.Status.ReadyReplicas)),
				})
			}

			cont = l.Continue
			if cont == "" {
				break
			}
		}
	}

	return nil
}
