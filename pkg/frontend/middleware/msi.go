package middleware

import (
	"net/http"

	"github.com/Azure/msi-dataplane/pkg/dataplane"
)

const (
	mockIdentityURL    = "https://bogus.identity.azure-int.net/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/rg/providers/Microsoft.ApiManagement/service/test/credentials?tid=00000000-0000-0000-0000-000000000000&oid=00000000-0000-0000-0000-000000000000&aid=00000000-0000-0000-0000-000000000000"
	mockTenantIDEnvVar = "MOCK_MSI_TENANT_ID"
)

// MockMSIMiddleware is used to mock MSI headers for development purposes
func GetMockMSIMiddleware() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set(dataplane.MsiIdentityURLHeader, mockIdentityURL)
			r.Header.Set(dataplane.MsiTenantHeader, mockTenantIDEnvVar)

			h.ServeHTTP(w, r)
		})
	}
}
