package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	testmonitor "github.com/Azure/ARO-RP/test/util/monitor"
)

func TestEmitMaintenanceState(t *testing.T) {
	for _, tt := range []struct {
		name              string
		provisioningState api.ProvisioningState
		maintenanceState  api.MaintenanceState
		adminUpdateErr    string
		expectedState     maintenanceState
	}{
		{
			name:              "state none - empty maintenance state",
			provisioningState: api.ProvisioningStateSucceeded,
			expectedState:     none,
		},
		{
			name:              "state none - no maintenance state set",
			provisioningState: api.ProvisioningStateSucceeded,
			maintenanceState:  api.MaintenanceStateNone,
			expectedState:     none,
		},
		{
			name:              "state pending",
			provisioningState: api.ProvisioningStateSucceeded,
			maintenanceState:  api.MaintenanceStatePending,
			expectedState:     pending,
		},
		{
			name:              "state unplanned",
			provisioningState: api.ProvisioningStateAdminUpdating,
			maintenanceState:  api.MaintenanceStateUnplanned,
			expectedState:     unplanned,
		},
		{
			name:              "state planned",
			provisioningState: api.ProvisioningStateAdminUpdating,
			maintenanceState:  api.MaintenanceStatePlanned,
			expectedState:     planned,
		},
		{
			name:              "state customer action needed",
			provisioningState: api.ProvisioningStateSucceeded,
			adminUpdateErr:    "error",
			maintenanceState:  api.MaintenanceStateCustomerActionNeeded,
			expectedState:     customerActionNeeded,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			oc := &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ProvisioningState:    tt.provisioningState,
					MaintenanceState:     tt.maintenanceState,
					LastAdminUpdateError: tt.adminUpdateErr,
				},
			}

			m := testmonitor.NewFakeEmitter(t)
			mon := &Monitor{
				m:  m,
				oc: oc,
			}

			err := mon.emitMaintenanceState(ctx)
			if err != nil {
				t.Fatal(err)
			}

			m.VerifyEmittedMetrics(testmonitor.Metric("cluster.maintenance.pucm", int64(1), map[string]string{
				"state": tt.expectedState.String(),
			}))
		})
	}
}
