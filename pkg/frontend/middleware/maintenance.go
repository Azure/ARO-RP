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
func (mm MaintenanceMiddleware) EmitUnplannedMaintenanceSignal(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		resourceID := strings.TrimPrefix(filepath.Dir(r.URL.Path), "/admin")

		// Use a do-while loop to ensure we emit the metric at least once
		mm.EmitGauge("frontend.maintenance.unplanned", 1, map[string]string{
			"resourceID": resourceID,
		})
		go func(ctx context.Context, resourceID string) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					time.Sleep(1 * time.Minute)
					mm.EmitGauge("frontend.maintenance.unplanned", 1, map[string]string{
						"resourceID": resourceID,
					})
				}
			}
		}(ctx, resourceID)

		h.ServeHTTP(w, r)
	})
}
