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
// state in single dashboard.
// NOTE: Do not return early. We want to report this metric whatever it takes
// so we can have a view of all clusters with, at least, provisioning states.
func (mon *Monitor) emitSummary(ctx context.Context) error {
	if !mon.hourlyRun {
		return nil
	}

	dims := map[string]string{
		"provisioningState":       mon.oc.Properties.ProvisioningState.String(),
		"failedProvisioningState": mon.oc.Properties.FailedProvisioningState.String(),
		"actualVersion":           "unknown",
		"desiredVersion":          "unknown",
		"masterCount":             "unknown",
		"workerCount":             "unknown",
	}

	cv, err := mon.getClusterVersion(ctx)
	if err != nil {
		mon.log.Warn(err)
	} else {
		dims["actualVersion"] = actualVersion(cv)
		dims["desiredVersion"] = desiredVersion(cv)
	}

	ns, err := mon.listNodes(ctx)
	if err != nil {
		mon.log.Warn(err)
	} else {
		var masterCount, workerCount int
		for _, node := range ns.Items {
			if _, ok := node.Labels[masterRoleLabel]; ok {
				masterCount++
			}
			if _, ok := node.Labels[workerRoleLabel]; ok {
				workerCount++
			}
		}
		dims["masterCount"] = strconv.Itoa(masterCount)
		dims["workerCount"] = strconv.Itoa(workerCount)
	}

	mon.emitGauge("cluster.summary", 1, dims)

	return nil
}
