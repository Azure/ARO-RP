package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pgknamespace "github.com/Azure/ARO-RP/pkg/util/namespace"
)

func (mon *Monitor) emitStatefulSetsConditions(ctx context.Context) error {
	sss, err := mon.cli.AppsV1().StatefulSets("").List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ss := range sss.Items {
		if !pgknamespace.IsOpenShift(ss.Namespace) {
			continue
		}

		if ss.Status.Replicas == ss.Status.ReadyReplicas {
			continue
		}

		mon.emitGauge("statefulsets.conditions", 1, map[string]string{
			"name":          ss.Name,
			"namespace":     ss.Namespace,
			"replicas":      strconv.Itoa(int(ss.Status.Replicas)),
			"readyReplicas": strconv.Itoa(int(ss.Status.ReadyReplicas)),
		})
	}

	return nil
}
