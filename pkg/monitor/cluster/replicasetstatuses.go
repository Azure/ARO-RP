package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/namespace"
)

func (mon *Monitor) emitReplicasetStatuses(ctx context.Context) error {
	rss, err := mon.cli.AppsV1().ReplicaSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, rs := range rss.Items {
		if !namespace.IsOpenShift(rs.Namespace) {
			continue
		}

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

	return nil
}
