package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
)

func (mon *Monitor) emitSyncSetStatus(ctx context.Context) error {
	cs, error := mon.hiveClusterManager.GetSyncSetResources(ctx, mon.doc)
	if error != nil {
		return nil
	}
	if cs.Status.SyncSets != nil {
		mon.emitGauge("syncsets.count", int64(len(cs.Status.SyncSets)), nil)

		for _, s := range cs.Status.SyncSets {
			mon.emitGauge("hive.syncsets", 1, map[string]string{
				"name":               s.Name,
				"result":             string(s.Result),
				"firstSuccessTime":   s.FirstSuccessTime.String(),
				"lastTransitionTime": s.LastTransitionTime.String(),
				"failureMessage":     s.FailureMessage,
			})
		}
	}

	if cs.Status.SelectorSyncSets != nil {
		mon.emitGauge("selectorsyncsets.count", int64(len(cs.Status.SelectorSyncSets)), nil)

		for _, s := range cs.Status.SelectorSyncSets {
			mon.emitGauge("hive.selectorsyncsets", 1, map[string]string{
				"name":               s.Name,
				"result":             string(s.Result),
				"firstSuccessTime":   s.FirstSuccessTime.String(),
				"lastTransitionTime": s.LastTransitionTime.String(),
				"failureMessage":     s.FailureMessage,
			})
		}
	}
	return nil
}
