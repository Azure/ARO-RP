package middleware

import (
	"net/http"
	"os"

	"github.com/Azure/msi-dataplane/pkg/dataplane"
)

const (
	mockIdentityURL    = "https://bogus.identity.azure-int.net/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/rg/providers/Microsoft.ApiManagement/service/test/credentials?tid=00000000-0000-0000-0000-000000000000&oid=00000000-0000-0000-0000-000000000000&aid=00000000-0000-0000-0000-000000000000"
	mockTenantIDEnvVar = "MOCK_MSI_TENANT_ID"
)

// MockMSIMiddleware is used to mock MSI headers for development purposes
// Do not use this in production code!

func GetMockMSIMiddleware(next http.Handler) func(http.ResponseWriter, *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set(dataplane.MsiIdentityURLHeader, mockIdentityURL)
		r.Header.Set(dataplane.MsiTenantHeader, os.Getenv(mockTenantIDEnvVar))
	})
}
