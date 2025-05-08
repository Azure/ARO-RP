package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"os"
)

const (
	MsiIdentityURLHeader = "x-ms-identity-url"
	MsiPrincipalIDHeader = "x-ms-identity-principal-id"
	MsiTenantHeader      = "x-ms-home-tenant-id"

	MockIdentityURL    = "https://bogus.identity.azure.net/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/rg/providers/Microsoft.ApiManagement/service/test/credentials?tid=00000000-0000-0000-0000-000000000000&oid=00000000-0000-0000-0000-000000000000&aid=00000000-0000-0000-0000-000000000000"
	mockTenantIDEnvVar = "MOCK_MSI_TENANT_ID"
)

func MockMSIMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set(MsiIdentityURLHeader, MockIdentityURL)
		r.Header.Set(MsiTenantHeader, os.Getenv(mockTenantIDEnvVar))
		h.ServeHTTP(w, r)
	})
}
