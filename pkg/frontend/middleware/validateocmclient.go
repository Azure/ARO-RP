package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
)

type OCMValidator struct {
	Env env.Interface
}

func (o OCMValidator) ValidateOCMClient(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ocmResourceType := chi.URLParam(r, "ocmResourceType"); ocmResourceType != "" {
			if systemDataHeader := r.Header.Get(ArmSystemDataHeaderKey); !o.Env.ValidateOCMClientID(systemDataHeader) {
				api.WriteError(w, http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Forbidden.")
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}
