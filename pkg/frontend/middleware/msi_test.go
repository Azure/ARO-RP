package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMockMSIMiddleware(t *testing.T) {
	mockTenantID := "test-tenant-id"
	t.Setenv(mockTenantIDEnvVar, mockTenantID)

	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := MockMSIMiddleware(mockHandler)

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if req.Header.Get(MsiIdentityURLHeader) != MockIdentityURL {
		t.Errorf("Expected %s, got %s", MockIdentityURL, req.Header.Get(MsiIdentityURLHeader))
	}
	if req.Header.Get(MsiTenantHeader) != mockTenantID {
		t.Errorf("Expected %s, got %s", mockTenantID, req.Header.Get(MsiTenantHeader))
	}
}
