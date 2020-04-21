package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pgknamespace "github.com/Azure/ARO-RP/pkg/util/namespace"
)

func (mon *Monitor) emitReplicaSetsConditions(ctx context.Context) error {
	rss, err := mon.cli.AppsV1().ReplicaSets("").List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, rs := range rss.Items {
		if !pgknamespace.IsOpenShift(rs.Namespace) {
			continue
		}

		if rs.Status.Replicas == rs.Status.AvailableReplicas {
			continue
		}

		mon.emitGauge("replicasets.conditions", 1, map[string]string{
			"availableReplicas": strconv.Itoa(int(rs.Status.AvailableReplicas)),
			"name":              rs.Name,
			"namespace":         rs.Namespace,
			"replicas":          strconv.Itoa(int(rs.Status.Replicas)),
		})
	}

	return nil
}
