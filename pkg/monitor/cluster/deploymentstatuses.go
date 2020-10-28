package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"

	"github.com/Azure/ARO-RP/pkg/util/namespace"
)

func (mon *Monitor) emitDeploymentStatuses(ctx context.Context) error {
	ds, err := mon.listDeployments(ctx)
	if err != nil {
		return err
	}

	for _, d := range ds.Items {
		if !namespace.IsOpenShift(d.Namespace) {
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

	return nil
}
