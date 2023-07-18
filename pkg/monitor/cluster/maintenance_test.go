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

func TestEmitPucmState(t *testing.T) {
	for _, tt := range []struct {
		name              string
		provisioningState api.ProvisioningState
		pucmPending       bool
		adminUpdateErr    string
		expectedPucmState pucmState
	}{
		{
			name:              "state none",
			provisioningState: api.ProvisioningStateSucceeded,
			expectedPucmState: pucmNone,
		},
		{
			name:              "state pending",
			provisioningState: api.ProvisioningStateSucceeded,
			pucmPending:       true,
			expectedPucmState: pucmPending,
		},
		{
			name:              "state unplanned - admin updating in flight and no admin update error",
			provisioningState: api.ProvisioningStateAdminUpdating,
			expectedPucmState: pucmUnplanned,
		},
		{
			name:              "state planned - admin updating in flight and no admin update error",
			provisioningState: api.ProvisioningStateAdminUpdating,
			pucmPending:       true,
			expectedPucmState: pucmPlanned,
		},
		{
			name:              "state unplanned - not admin updating but admin update error",
			provisioningState: api.ProvisioningStateFailed,
			adminUpdateErr:    "PUCM failed",
			expectedPucmState: pucmUnplanned,
		},
		{
			name:              "state planned - not admin updating but admin update error",
			provisioningState: api.ProvisioningStateFailed,
			pucmPending:       true,
			adminUpdateErr:    "PUCM failed",
			expectedPucmState: pucmPlanned,
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
					PucmPending:          tt.pucmPending,
					LastAdminUpdateError: tt.adminUpdateErr,
				},
			}
			mon := &Monitor{
				m:  m,
				oc: oc,
			}

			m.EXPECT().EmitGauge("cluster.maintenance.pucm", int64(1), map[string]string{
				"state": tt.expectedPucmState.String(),
			})

			err := mon.emitPucmState(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
