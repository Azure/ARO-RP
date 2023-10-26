package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
)

/**************************************************************
	Possible PUCM states:

	(1) PUCM pending
		- We will do PUCM, so emit a maintenance pending signal

	(2) Planned PUCM in progress
		- Emit a planned maintenance in progress signal.
		- If first PUCM attempt fails, leave cluster in this state because
		  we will need to either retry PUCM or have an SRE update the state to now.

	(3) Unplanned PUCM in progress
		- Emit an unplanned maintenance in progress signal.
		- If first PUCM attempt fails, leave cluster in this state because
		  we will need to either retry PUCM or have an SRE update the state to now.

	(4) No ongoinig or scheduled PUCM
		- Emit the none signal.
**************************************************************/

type maintenanceState string

func (m maintenanceState) String() string {
	return string(m)
}

const (
	none      maintenanceState = "none"
	pending   maintenanceState = "pending"
	planned   maintenanceState = "planned"
	unplanned maintenanceState = "unplanned"
)

func (mon *Monitor) emitMaintenanceState(ctx context.Context) error {
	state := getMaintenanceState(mon.oc.Properties)
	mon.emitGauge("cluster.maintenance.pucm", 1, map[string]string{
		"state": state.String(),
	})

	return nil
}

func getMaintenanceState(clusterProperties api.OpenShiftClusterProperties) maintenanceState {
	switch clusterProperties.MaintenanceState {
	case api.MaintenanceStatePending:
		return pending
	case api.MaintenanceStatePlanned:
		return planned
	case api.MaintenanceStateUnplanned:
		return unplanned
	case api.MaintenanceStateNone:
		fallthrough
	// For new clusters, no maintenance state has been set yet
	default:
		return none
	}
}
