package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/sirupsen/logrus"
)

func SystemData(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := r.Header.Get("x-ms-arm-resource-system-data")
		if data == "" {
			h.ServeHTTP(w, r)
			return
		}

		var systemData *api.SystemData
		err := json.Unmarshal([]byte(data), systemData)
		if err != nil {
			if log, ok := r.Context().Value(ContextKeyLog).(*logrus.Entry); ok {
				log.Warn("failed to read systemData")
			}
		}

		r = r.WithContext(context.WithValue(r.Context(), ContextKeySystemData, systemData))

		h.ServeHTTP(w, r)
	})
}
