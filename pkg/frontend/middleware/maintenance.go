package middleware

type MaintenanceMiddleware struct {
	metrics.Emitter
}

// Emit metric for unplanned maintenance
func (mm MaintenanceMiddleware) EmitUnplannedMaintenanceSignal(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		resourceID := strings.TrimPrefix(filepath.Dir(r.URL.Path), "/admin")

		go func(ctx context.Contetxt, resourceID string) {
			for {
				mm.EmitGauge("frontend.maintenance.unplanned", 1, map[string]string{
					"resource_id": resourceID,
				})
				select {
				case <-ctx.Done():
					return
				default:
					time.Sleep(1 * time.Minute)
				}
			}
		}(ctx, resourceID)

		h.ServeHTTP(w, r)
	})
}
