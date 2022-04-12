package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/namespace"
)

func (mon *Monitor) emitDeploymentStatuses(ctx context.Context) error {
	var cont string
	var count int64
	for {
		ds, err := mon.cli.AppsV1().Deployments("").List(ctx, metav1.ListOptions{Limit: 500, Continue: cont})
		if err != nil {
			return err
		}

		count += int64(len(ds.Items))

		for _, d := range ds.Items {
			if !namespace.IsOpenShiftNamespace(d.Namespace) {
				continue
			}

			if d.Status.Replicas == d.Status.AvailableReplicas {
				continue
			}

			mon.emitGauge("deployment.statuses", 1, map[string]string{
				"availableReplicas": strconv.Itoa(int(d.Status.AvailableReplicas)),
				"name":              d.Name,
				"namespace":         d.Namespace,
				"replicas":          strconv.Itoa(int(d.Status.Replicas)),
			})
		}

		cont = ds.Continue
		if cont == "" {
			break
		}
	}
	mon.emitGauge("deployment.count", count, nil)
	return nil
}
