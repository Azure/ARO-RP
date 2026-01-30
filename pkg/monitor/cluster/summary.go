package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"

	corev1 "k8s.io/api/core/v1"
)

const (
	masterRoleLabel = "node-role.kubernetes.io/master"
	workerRoleLabel = "node-role.kubernetes.io/worker"
	infraRoleLabel  = "node-role.kubernetes.io/infra"
)

// emitSummary emits joined metric to be able to report better on all clusters
// state in single dashboard
func (mon *Monitor) emitSummary(ctx context.Context) error {
	if !mon.hourlyRun {
		return nil
	}

	var desiredVersion, actualVersion, actualMinorVersion string
	if mon.clusterActualVersion != nil {
		actualVersion = mon.clusterActualVersion.String()
		actualMinorVersion = mon.clusterActualVersion.MinorVersion()
	}

	if mon.clusterDesiredVersion != nil {
		desiredVersion = mon.clusterDesiredVersion.String()
	}

	var masterCount, workerCount int
	err := mon.iterateOverNodes(ctx, func(n *corev1.Node) {
		if _, ok := n.Labels[masterRoleLabel]; ok {
			masterCount++
		}
		if _, ok := n.Labels[workerRoleLabel]; ok {
			workerCount++
		}
	})
	if err != nil {
		return err
	}

	mon.emitGauge("cluster.summary", 1, map[string]string{
		"actualVersion":      actualVersion,
		"actualMinorVersion": actualMinorVersion,
		"desiredVersion":     desiredVersion,
		"masterCount":        strconv.Itoa(masterCount),
		"workerCount":        strconv.Itoa(workerCount),
		"provisioningState":  mon.oc.Properties.ProvisioningState.String(),
		"createdAt":          mon.oc.Properties.CreatedAt.String(),
	})

	return nil
}
