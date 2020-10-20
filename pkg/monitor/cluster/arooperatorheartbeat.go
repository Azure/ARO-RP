package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (mon *Monitor) emitAroOperatorHeartbeat(ctx context.Context) error {
	aroDeployments, err := mon.cli.AppsV1().Deployments("openshift-azure-operator").List(metav1.ListOptions{})

	if err != nil {
		return err
	}

	for _, d := range aroDeployments.Items {
		mon.emitGauge("arooperator.heartbeat", 1, map[string]string{
			"name":      d.Name,
			"available": strconv.FormatBool(d.Status.AvailableReplicas == d.Status.Replicas),
		})
	}

	return nil
}
