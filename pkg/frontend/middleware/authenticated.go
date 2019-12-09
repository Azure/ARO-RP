package middleware

import (
	"net/http"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/env"
)

func Authenticated(env env.Interface) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !env.IsAuthorized(r.TLS) {
				api.WriteError(w, http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Forbidden.")
				return
			}

			h.ServeHTTP(w, r)
		})
	}
}
