package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"
)

const (
	masterRoleLabel = "node-role.kubernetes.io/master"
	workerRoleLabel = "node-role.kubernetes.io/worker"
)

// emitSummary emits joined metric to be able to report better on all clusters
// state in single dashboard
func (mon *Monitor) emitSummary(ctx context.Context) error {
	if !mon.hourlyRun {
		return nil
	}

	cv, err := mon.getClusterVersion(ctx)
	if err != nil {
		return err
	}

	ns, err := mon.listNodes(ctx)
	if err != nil {
		return err
	}

	var masterCount, workerCount int
	for _, node := range ns.Items {
		if _, ok := node.Labels[masterRoleLabel]; ok {
			masterCount++
		}
		if _, ok := node.Labels[workerRoleLabel]; ok {
			workerCount++
		}
	}

	mon.emitGauge("cluster.summary", 1, map[string]string{
		"actualVersion":     actualVersion(cv),
		"desiredVersion":    desiredVersion(cv),
		"masterCount":       strconv.Itoa(masterCount),
		"workerCount":       strconv.Itoa(workerCount),
		"provisioningState": mon.oc.Properties.ProvisioningState.String(),
		"createdAt":         mon.oc.Properties.CreatedAt.String(),
	})

	return nil
}
