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

func (mon *Monitor) emitReplicasetStatuses(ctx context.Context) error {
	// Only fetch in the namespaces we manage
	for _, ns := range mon.namespacesToMonitor {
		var cont string
		l := &appsv1.ReplicaSetList{}

		for {
			err := mon.ocpclientset.List(ctx, l, client.InNamespace(ns), client.Continue(cont), client.Limit(mon.queryLimit))
			if err != nil {
				return fmt.Errorf("error in list operation: %w", err)
			}

			for _, rs := range l.Items {
				if rs.Status.Replicas == rs.Status.AvailableReplicas {
					continue
				}

				mon.emitGauge("replicaset.statuses", 1, map[string]string{
					"availableReplicas": strconv.Itoa(int(rs.Status.AvailableReplicas)),
					"name":              rs.Name,
					"namespace":         rs.Namespace,
					"replicas":          strconv.Itoa(int(rs.Status.Replicas)),
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
