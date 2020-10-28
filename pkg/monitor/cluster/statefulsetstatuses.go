package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/namespace"
)

func (mon *Monitor) emitStatefulsetStatuses(ctx context.Context) error {
	sss, err := mon.cli.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ss := range sss.Items {
		if !namespace.IsOpenShift(ss.Namespace) {
			continue
		}

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

	return nil
}
