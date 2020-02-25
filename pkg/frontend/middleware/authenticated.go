package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
)

func Authenticated(env env.Interface) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)

			var clientAuthorizer clientauthorizer.ClientAuthorizer
			switch vars["api-version"] {
			case admin.APIVersion:
				clientAuthorizer = env.AdminClientAuthorizer()
			default:
				clientAuthorizer = env.ArmClientAuthorizer()
			}

			if !clientAuthorizer.IsAuthorized(r.TLS) {
				api.WriteError(w, http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Forbidden.")
				return
			}

			h.ServeHTTP(w, r)
		})
	}
}
