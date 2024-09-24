package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/portal/util/responsewriter"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestUnplannedMaintenanceSignal(t *testing.T) {
	for _, tt := range []struct {
		name           string
		resourceID     string
		adminOperation string
	}{
		{
			name:           "emit unplanned maintenance signal",
			resourceID:     "/subscriptions/123/resourcegroups/456/providers/Microsoft.RedHatOpenShift/openShiftClusters/789",
			adminOperation: "/startvm",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			m := mock_metrics.NewMockEmitter(controller)

			maintenanceMiddleware := MaintenanceMiddleware{m}

			m.EXPECT().EmitGauge("frontend.maintenance.unplanned", int64(1), map[string]string{
				"resourceId": tt.resourceID,
			})

			handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			handler := maintenanceMiddleware.UnplannedMaintenanceSignal(handlerFunc)

			path := "/admin" + tt.resourceID + tt.adminOperation
			r, err := http.NewRequest(http.MethodPost, path, nil)
			if err != nil {
				t.Fatal(err)
			}
			w := responsewriter.New(r)

			handler.ServeHTTP(w, r)
		})
	}
}
