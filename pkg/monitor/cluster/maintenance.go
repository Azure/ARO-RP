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
		- Conditions:
			* Field pucmPending is true
			* Don't meet below conditions for in progress maintenance

	(2) Planned PUCM in progress
		- Emit a planned maintenance in progress signal.
		- If first PUCM attempt fails, leave cluster in this state
		  because we will need to retry PUCM in at a later time.
		- Conditions:
			* Field pucmPending is true
			* One of: (a) provisoning state AdminUpdate or (2) AdminUpdate err is not nil

	(3) Unplanned PUCM in progress
		- Emit an unplanned maintenance in progress signal.
		- If first PUCM attempt fails, leave cluster in this state
		  because we will need to retry PUCM in at a later time.
		- Conditions:
			* Field pucmPending is false
			* One of: (a) provisoning state AdminUpdate or (2) AdminUpdate err is not nil

	(4) No ongoinig or scheduled PUCM
		- Don't emit a signal
		- Conditions:
			* Field pucmPending is false
			* Provisioning state is not AdminUpdate and AdminUpdate err is not nil
**************************************************************/

type pucmState string

func (p pucmState) String() string {
	return string(p)
}

const (
	pucmNone             pucmState = "none"
	pucmPending          pucmState = "pending"
	pucmPlannedOngoing   pucmState = "planned_ongoing"
	pucmUnplannedOngoing pucmState = "unplanned_ongoing"
)

func (mon *Monitor) emitPucmState(ctx context.Context) error {
	state := getPucmState(mon.oc.Properties)
	mon.emitGauge("cluster.maintenance.pucm", 1, map[string]string{
		"state": state.String(),
	})

	return nil
}

func getPucmState(clusterProperties api.OpenShiftClusterProperties) pucmState {
	if pucmOngoing(clusterProperties) {
		if clusterProperties.PucmPending {
			return pucmPlannedOngoing
		} else {
			return pucmUnplannedOngoing
		}
	} else {
		if clusterProperties.PucmPending {
			return pucmPending
		}
	}

	return pucmNone
}

func pucmOngoing(clusterProperties api.OpenShiftClusterProperties) bool {
	return clusterProperties.ProvisioningState == api.ProvisioningStateAdminUpdating ||
		clusterProperties.LastAdminUpdateError != ""
}
