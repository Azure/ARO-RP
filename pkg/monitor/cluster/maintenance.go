package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
)

/**************************************************************
	Possible maintenance states:

	(1) Maintenance pending
		- We will do maintenance, so emit a maintenance pending signal

	(2) Planned maintenance in progress
		- Emit a planned maintenance in progress signal.
		- If first attempt fails, leave cluster in this state because
		  we will need to either retry or have an SRE update the state to none.

	(3) Unplanned maintenance in progress
		- Emit an unplanned maintenance in progress signal.
		- If first attempt fails, leave cluster in this state because
		  we will need to either retry or have an SRE update the state to none.

	(4) No ongoinig or scheduled maintenance
		- Emit the none signal.
**************************************************************/

type maintenanceState string

func (m maintenanceState) String() string {
	return string(m)
}

const (
	none                 maintenanceState = "none"
	pending              maintenanceState = "pending"
	planned              maintenanceState = "planned"
	unplanned            maintenanceState = "unplanned"
	customerActionNeeded maintenanceState = "customerActionNeeded"
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
	case api.MaintenanceStateCustomerActionNeeded:
		return customerActionNeeded
	case api.MaintenanceStateNone:
		fallthrough
	// For new clusters, no maintenance state has been set yet
	default:
		return none
	}
}
