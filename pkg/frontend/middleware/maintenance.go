package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/ARO-RP/pkg/metrics"
)

type MaintenanceMiddleware struct {
	metrics.Emitter
}

// Emit metric for unplanned maintenance
func (mm MaintenanceMiddleware) UnplannedMaintenanceSignal(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		resourceID := strings.TrimPrefix(filepath.Dir(r.URL.Path), "/admin")

		// Use a do-while loop to ensure we emit the metric at least once
		mm.emitMaintenanceSignal("unplanned", resourceID)
		go func(ctx context.Context, resourceID string) {
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Minute):
					mm.emitMaintenanceSignal("unplanned", resourceID)
				}
			}
		}(ctx, resourceID)

		h.ServeHTTP(w, r)
	})
}

func (mm MaintenanceMiddleware) emitMaintenanceSignal(maintenanceType, resourceID string) {
	maintenanceMetric := "frontend.maintenance." + maintenanceType
	mm.EmitGauge(maintenanceMetric, 1, map[string]string{
		"resourceId": resourceID,
	})
}
