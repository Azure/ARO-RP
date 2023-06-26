package cluster

import "context"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func (mon *Monitor) emitClusterMaintenanceState(ctx context.Context) error {
	mon.emitGauge("cluster.maintenanceState", 1, map[string]string{
		"maintenanceState": mon.oc.Properties.MaintenanceState.String(),
	})

	return nil
}
