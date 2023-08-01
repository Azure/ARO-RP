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

func TestEmitSpExpiration(t *testing.T) {
	for _, tt := range []struct {
		appId      string
		expiryDate string
	}{
		{
			appId:      "1245-2533-52114",
			expiryDate: "2023-10-20",
		},
		{
			appId:      "1245-2533-52114",
			expiryDate: "2023-10-20",
		},
	} {
		t.Run(tt.appId, func(t *testing.T) {
			ctx := context.Background()

			controller := gomock.NewController(t)
			defer controller.Finish()

			m := mock_metrics.NewMockEmitter(controller)
			oc := &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: api.ServicePrincipalProfile{
						ClientID: tt.appId,
					},
				},
			}
			mon := &Monitor{
				m:  m,
				oc: oc,
			}

			m.EXPECT().EmitGauge("cluster.serviceprincipal.expiration", int64(1), map[string]string{
				"expiryDate": tt.expiryDate,
			})

			err := mon.emitSpExpiration(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
