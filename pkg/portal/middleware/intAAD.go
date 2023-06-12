package middleware

import (
	"context"
	"net/http"
)

const (
	IntUsernameKey = "INT_OAUTH_USERNAME"
	IntGroupsKey   = "INT_OAUTH_GROUPS"
)

// IntAAD effectively disable authentication for testing purposes
type IntAAD struct {
}

func (a IntAAD) Callback(w http.ResponseWriter, r *http.Request) {
}

func (a IntAAD) Login(w http.ResponseWriter, r *http.Request) {
}

func (a IntAAD) AAD(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		groups := ""
		username := ""

		//here we use cookies because selenium doesn't allow us to set headers
		for _, v := range r.Cookies() {
			switch v.Name {
			case IntGroupsKey:
				groups = v.Value

			case IntUsernameKey:
				username = v.Value
			}
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, ContextKeyUsername, username)
		ctx = context.WithValue(ctx, ContextKeyGroups, groups)
		r = r.WithContext(ctx)

		h.ServeHTTP(w, r)
	})
}

func (a IntAAD) Logout(url string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: OIDCCookie, MaxAge: -1})
		http.Redirect(w, r, url, http.StatusSeeOther)
	})
}
