package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"strings"
)

func Headers(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.EqualFold(r.Header.Get("X-Ms-Return-Client-Request-Id"), "true") {
			w.Header().Set("X-Ms-Client-Request-Id", r.Header.Get("X-Ms-Client-Request-Id"))
		}

		// In production, ARM sets the Referer header (see
		// https://github.com/Azure/azure-resource-manager-rpc).  We use this to
		// construct Azure-AsyncOperation headers for polling of long-running
		// operations.  In development, fake this, otherwise `az aro create`
		// polling wouldn't work against a development RP.
		if r.Header.Get("Referer") == "" {
			r.Header.Set("Referer", "https://localhost:8443"+r.URL.String())
		}

		h.ServeHTTP(w, r)
	})
}
