package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (mon *Monitor) getMachineConfigPoolNodeCounts(ctx context.Context) (int64, error) {
	var cont string
	var count int64

	for {
		mcps, err := mon.mcocli.MachineconfigurationV1().MachineConfigPools().List(ctx, metav1.ListOptions{Limit: 500, Continue: cont})
		if err != nil {
			return 0, err
		}

		for _, mcp := range mcps.Items {
			count += int64(mcp.Status.MachineCount)
		}

		cont = mcps.Continue
		if cont == "" {
			break
		}
	}

	return count, nil
}

func (mon *Monitor) getNodeCounts(ctx context.Context) (int64, error) {
	ns, err := mon.listNodes(ctx)
	if err != nil {
		return 0, err
	}

	return int64(len(ns.Items)), nil
}

// Count the number of nodes available
// Total the nodes under machineconfigpool control
// Alert if different
func (mon *Monitor) emitMachineConfigPoolUnmanagedNodeCounts(ctx context.Context) error {
	mcpcount, err := mon.getMachineConfigPoolNodeCounts(ctx)
	if err != nil {
		return err
	}

	getnodescount, err := mon.getNodeCounts(ctx)
	if err != nil {
		return err
	}

	// unmanagednodescount of 0 is normal (machineconfigpool nodes == nodes)
	// also report if there are missing nodes with too many machineconfigs
	unmanagednodescount := getnodescount - mcpcount

	// emit count of nodes which are not managed by MCP
	// =0 is expected normal (all nodes are managed)
	// >0 mcp isn't managing all nodes
	// <0 nodes are missing from mcp
	if unmanagednodescount != 0 {
		mon.emitGauge("machineconfigpool.unmanagednodescount", unmanagednodescount, nil)
	}

	if mon.hourlyRun && unmanagednodescount != 0 {
		mon.log.Printf("machineconfigpool.unmanagednodescount: %d", unmanagednodescount)
	}

	return nil
}
