package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"testing"
)

func TestLowercase(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/TEST", nil)
	if err != nil {
		t.Fatal(err)
	}

	Lowercase(http.HandlerFunc(func(w http.ResponseWriter, _r *http.Request) {
		r = _r
	})).ServeHTTP(nil, r)

	if r.URL.Path != "/test" {
		t.Error(r.URL.Path)
	}

	originalPath := r.Context().Value(ContextKeyOriginalPath).(string)
	if originalPath != "/TEST" {
		t.Error(originalPath)
	}
}
