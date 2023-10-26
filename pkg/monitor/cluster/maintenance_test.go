package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
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
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			controller := gomock.NewController(t)
			defer controller.Finish()

			m := mock_metrics.NewMockEmitter(controller)
			oc := &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ProvisioningState:    tt.provisioningState,
					MaintenanceState:     tt.maintenanceState,
					LastAdminUpdateError: tt.adminUpdateErr,
				},
			}
			mon := &Monitor{
				m:  m,
				oc: oc,
			}

			m.EXPECT().EmitGauge("cluster.maintenance.pucm", int64(1), map[string]string{
				"state": tt.expectedState.String(),
			})

			err := mon.emitMaintenanceState(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
