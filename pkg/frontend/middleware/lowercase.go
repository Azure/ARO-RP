package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strings"
)

func Lowercase(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), ContextKeyOriginalPath, r.URL.Path))
		r.URL.Path = strings.ToLower(r.URL.Path)

		h.ServeHTTP(w, r)
	})
}
