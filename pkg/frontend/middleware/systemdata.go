package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
)

func SystemData(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := r.Header.Get("X-Ms-Arm-Resource-System-Data")
		if data != "" {
			var systemData *api.SystemData
			err := json.Unmarshal([]byte(data), &systemData)
			if err != nil {
				if log, ok := r.Context().Value(ContextKeyLog).(*logrus.Entry); ok {
					log.Warnf("failed to unmarshal systemData: %v", err)
				}
			}

			r = r.WithContext(context.WithValue(r.Context(), ContextKeySystemData, systemData))
		}

		h.ServeHTTP(w, r)
	})
}
