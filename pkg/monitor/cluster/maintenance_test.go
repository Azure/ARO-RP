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
	ctx := context.Background()

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockEmitter(controller)

	// Unplanned ongoing
	oc := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ProvisioningState: api.ProvisioningStateAdminUpdating,
		},
	}
	mon := &Monitor{
		m:  m,
		oc: oc,
	}
	m.EXPECT().EmitGauge("cluster.maintenance.pucm", int64(1), map[string]string{
		"state": pucmUnplannedOngoing.String(),
	})

	err := mon.emitPucmState(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Planned ongoing
	oc = &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ProvisioningState: api.ProvisioningStateAdminUpdating,
			PucmPending:       true,
		},
	}

	m.EXPECT().EmitGauge("cluster.maintenance.pucm", int64(1), map[string]string{
		"state": pucmPlannedOngoing.String(),
	})

	err = mon.emitPucmState(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Pending
	oc = &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ProvisioningState: api.ProvisioningStateSucceeded,
			PucmPending:       true,
		},
	}

	m.EXPECT().EmitGauge("cluster.maintenance.pucm", int64(1), map[string]string{
		"state": pucmPending.String(),
	})

	err = mon.emitPucmState(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// None
	oc = &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ProvisioningState: api.ProvisioningStateSucceeded,
		},
	}

	m.EXPECT().EmitGauge("cluster.maintenance.pucm", int64(1), map[string]string{
		"state": pucmNone.String(),
	})

	err = mon.emitPucmState(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
