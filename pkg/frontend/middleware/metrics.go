package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics"
)

type MetricsMiddleware struct {
	metrics.Emitter
}

// Metric records request metrics for tracking
func (mm MetricsMiddleware) Metrics(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiVersion := r.URL.Query().Get(api.APIVersionKey)
		t := time.Now()

		w = &logResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		h.ServeHTTP(w, r)

		//get the route pattern that matched
		rctx := chi.RouteContext(r.Context())
		routePattern := strings.Join(rctx.RoutePatterns, "")
		mm.EmitGauge("frontend.count", 1, map[string]string{
			"verb":        r.Method,
			"api-version": apiVersion,
			"code":        strconv.Itoa(w.(*logResponseWriter).statusCode),
			"route":       routePattern,
		})

		mm.EmitGauge("frontend.duration", time.Since(t).Milliseconds(), map[string]string{
			"verb":        r.Method,
			"api-version": apiVersion,
			"code":        strconv.Itoa(w.(*logResponseWriter).statusCode),
			"route":       routePattern,
		})
	})
}
