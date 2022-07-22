package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strings"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

// Bearer validates a Bearer token and adds the corresponding username to the
// context if it checks out.  It lets the request through regardless (this is so
// that failures can be logged).
func Bearer(dbPortal database.Portal) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			authorization := r.Header.Get("Authorization")
			if !strings.HasPrefix(authorization, "Bearer ") {
				h.ServeHTTP(w, r)
				return
			}

			token, err := uuid.FromString(strings.TrimPrefix(authorization, "Bearer "))
			if err != nil {
				h.ServeHTTP(w, r)
				return
			}

			portalDoc, err := dbPortal.Get(ctx, token.String())
			if err != nil {
				h.ServeHTTP(w, r)
				return
			}

			ctx = context.WithValue(ctx, ContextKeyUsername, portalDoc.Portal.Username)
			ctx = context.WithValue(ctx, ContextKeyPortalDoc, portalDoc)

			r = r.WithContext(ctx)

			h.ServeHTTP(w, r)
		})
	}
}
