package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/controller-runtime/pkg/client"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
)

func (mon *Monitor) getMachineConfigPoolNodeCounts(ctx context.Context) (int64, error) {
	var cont string
	var count int64
	l := &mcv1.MachineConfigPoolList{}

	for {
		err := mon.ocpclientset.List(ctx, l, client.Continue(cont), client.Limit(mon.queryLimit))
		if err != nil {
			return 0, err
		}

		for _, mcp := range l.Items {
			count += int64(mcp.Status.MachineCount)
		}

		cont = l.Continue
		if cont == "" {
			break
		}
	}

	return count, nil
}

func (mon *Monitor) getNodeCounts(ctx context.Context) (int64, error) {
	var cont string
	var count int

	l := &metav1.PartialObjectMetadataList{}
	l.SetGroupVersionKind(schema.GroupVersionKind{Version: "v1", Group: "", Kind: "NodeList"})

	for {
		err := mon.ocpclientset.List(ctx, l, client.Continue(cont), client.Limit(mon.queryLimit))
		if err != nil {
			return 0, fmt.Errorf("error in Node metadata list operation: %w", err)
		}

		count += len(l.Items)

		cont = l.Continue
		if cont == "" {
			break
		}
	}

	return int64(count), nil
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
