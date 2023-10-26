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

type pucmState string

func (p pucmState) String() string {
	return string(p)
}

const (
	pucmNone      pucmState = "none"
	pucmPending   pucmState = "pending"
	pucmPlanned   pucmState = "planned"
	pucmUnplanned pucmState = "unplanned"
)

func (mon *Monitor) emitMaintenanceState(ctx context.Context) error {
	state := getPucmState(mon.oc.Properties)
	mon.emitGauge("cluster.maintenance.pucm", 1, map[string]string{
		"state": state.String(),
	})

	return nil
}

func getPucmState(clusterProperties api.OpenShiftClusterProperties) pucmState {
	switch clusterProperties.MaintenanceState {
	case api.MaintenanceStatePending:
		return pucmPending
	case api.MaintenanceStatePlanned:
		return pucmPlanned
	case api.MaintenanceStateUnplanned:
		return pucmUnplanned
	case api.MaintenanceStateNone:
		fallthrough
	// For new clusters, no maintenance state has been set yet
	default:
		return pucmNone
	}
}
