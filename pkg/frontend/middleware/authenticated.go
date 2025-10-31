package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/miseadapter"
)

type AuthMiddleware struct {
	Log *logrus.Entry

	EnableMISE  bool
	EnforceMISE bool

	AdminAuth clientauthorizer.ClientAuthorizer
	ArmAuth   clientauthorizer.ClientAuthorizer
	MiseAuth  miseadapter.MISEAdapter
}

func (a AuthMiddleware) Authenticate(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// grab the logger from the request context to get the correlation data
		_log := r.Context().Value(ContextKeyLog)
		log, ok := _log.(*logrus.Entry)
		if !ok {
			log = a.Log
		}

		var err error
		var authenticated bool
		// Admin API authorisation (Geneva Actions) is performed via mutual TLS
		// authentication
		apiVersion := r.URL.Query().Get(api.APIVersionKey)
		if apiVersion == admin.APIVersion || strings.HasPrefix(r.URL.Path, "/admin") {
			authenticated = a.AdminAuth.IsAuthorized(r.TLS)
		} else {
			// ARM traffic is authenticated via either MISE or mutual TLS
			// authentication
			if a.EnableMISE {
				authenticated, err = a.MiseAuth.IsAuthorized(log, r)
				if authenticated {
					log.Infoln("MISE authorization successful")
				} else {
					log.Errorf("MISE authorization unsuccessful, enforcing: %t, error: %s", a.EnforceMISE, err)
				}
			}

			// If we do not enforce MISE, then fall back to checking the TLS
			// certificate
			if !a.EnforceMISE && !authenticated {
				log.Warnln("MISE authorization unsuccessful/disabled, fallback to TLS certificate authentication")
				authenticated = a.ArmAuth.IsAuthorized(r.TLS)
			}
		}

		if !authenticated {
			log.Errorf("Authentication Failed")
			api.WriteError(w, http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Forbidden.")
			return
		}

		h.ServeHTTP(w, r)
	})
}
