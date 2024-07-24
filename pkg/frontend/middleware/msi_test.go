package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Azure/msi-dataplane/pkg/dataplane"
)

func TestMockMSIMiddleware(t *testing.T) {
	// Set the environment variable for the test
	mockTenantID := "test-tenant-id"
	t.Setenv(mockTenantIDEnvVar, mockTenantID)

	// Create a mock handler to be wrapped by the middleware
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This handler does nothing, just a placeholder
	})

	// Wrap the mock handler with the MockMSIMiddleware
	handler := MockMSIMiddleware(mockHandler)

	// Create a new HTTP request
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	// Create a response recorder to capture the response
	rr := httptest.NewRecorder()

	// Serve the HTTP request using the wrapped handler
	handler.ServeHTTP(rr, req)

	// Check the response headers
	if req.Header.Get(dataplane.MsiIdentityURLHeader) != mockIdentityURL {
		t.Errorf("Expected %s, got %s", mockIdentityURL, req.Header.Get(dataplane.MsiIdentityURLHeader))
	}
	if req.Header.Get(dataplane.MsiTenantHeader) != mockTenantID {
		t.Errorf("Expected %s, got %s", mockTenantID, req.Header.Get(dataplane.MsiTenantHeader))
	}
}
