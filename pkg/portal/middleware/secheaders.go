package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
)

// SecurityHeaders attaches a restrictive Content-Security-Policy and related
// hardening headers to every admin portal response.
//
// The portal proxies content from customer clusters (for example the cluster's
// Prometheus), and a customer with cluster-admin on their own cluster can
// author arbitrary markup.  A strict script-src with no 'unsafe-inline' is the
// control that stops injected inline <script> elements and event-handler
// attributes (onerror=, onclick=, ...) from executing on the portal's origin.
//
// A fresh nonce is minted per request and placed on the context so that
// server-rendered templates which must emit a legitimate inline <script> (the
// embedded Prometheus UI) can opt that single element in via a matching
// nonce attribute.  'unsafe-inline' is permitted for style-src only: the React
// (Fluent UI) portal injects styles at runtime, and style injection is far less
// dangerous than script execution, which remains strictly gated.
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nonce, err := cspNonce()
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Security-Policy", fmt.Sprintf(
				"default-src 'self'; "+
					"script-src 'self' 'nonce-%s'; "+
					"style-src 'self' 'unsafe-inline'; "+
					"img-src 'self' data:; "+
					"connect-src 'self'; "+
					"font-src 'self'; "+
					"object-src 'none'; "+
					"base-uri 'self'; "+
					"frame-ancestors 'none'", nonce))
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "no-referrer")

			r = r.WithContext(context.WithValue(r.Context(), ContextKeyCSPNonce, nonce))

			h.ServeHTTP(w, r)
		})
	}
}

// CSPNonce returns the per-request Content-Security-Policy nonce set by
// SecurityHeaders, or the empty string if none is present.
func CSPNonce(ctx context.Context) string {
	nonce, _ := ctx.Value(ContextKeyCSPNonce).(string)
	return nonce
}

func cspNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
