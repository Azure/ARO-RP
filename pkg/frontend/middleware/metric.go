package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/Azure/ARO-RP/pkg/metrics"
)

type frontendMetric struct {
	Start      time.Time
	Verb       string
	RouteName  string
	ApiVersion string
	Code       string
}

// Metric records request metrics for tracking
func Metric(m metrics.Interface) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			t := time.Now()
			var routeName string
			if route := mux.CurrentRoute(r); route != nil {
				routeName = route.GetName()
			} else {
				routeName = "unknown"
			}

			w = &logResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			h.ServeHTTP(w, r)

			// TODO: Request count metric

			defer m.EmitFloat("frontend.duration",
				float64(time.Now().Sub(t)/time.Millisecond),
				"verb", r.Method,
				"api-version", vars["api-version"],
				"code", strconv.Itoa(w.(*logResponseWriter).statusCode),
				"route", routeName)

		})
	}
}
