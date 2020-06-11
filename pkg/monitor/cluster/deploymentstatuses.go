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
	ds, err := mon.cli.AppsV1().Deployments("").List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, d := range ds.Items {
		if !namespace.IsOpenShift(d.Namespace) {
			continue
		}

		if d.Status.Replicas == d.Status.AvailableReplicas && !mon.hourlyRun {
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
