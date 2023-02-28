package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"strings"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
)

type AuthMiddleware struct {
	AdminAuth clientauthorizer.ClientAuthorizer
	ArmAuth   clientauthorizer.ClientAuthorizer
}

func (a AuthMiddleware) Authenticate(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiVersion := r.URL.Query().Get(api.APIVersionKey)
		var clientAuthorizer clientauthorizer.ClientAuthorizer
		if apiVersion == admin.APIVersion || strings.HasPrefix(r.URL.Path, "/admin") {
			clientAuthorizer = a.AdminAuth
		} else {
			clientAuthorizer = a.ArmAuth
		}

		if !clientAuthorizer.IsAuthorized(r.TLS) {
			api.WriteError(w, http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Forbidden.")
			return
		}

		h.ServeHTTP(w, r)
	})
}
