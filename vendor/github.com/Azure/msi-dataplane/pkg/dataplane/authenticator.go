package dataplane

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/gofrs/uuid"
)

var (
	errInvalidAuthHeader = errors.New("could not parse the provided WWW-Authenticate header")
	errInvalidTenantID   = errors.New("the provided tenantID is invalid")
)

// Authenticating with MSI: https://eng.ms/docs/products/arm/rbac/managed_identities/msionboardinginteractionwithmsi .
func NewAuthenticatorPolicy(cred azcore.TokenCredential, audience string) policy.Policy {
	return runtime.NewBearerTokenPolicy(cred, nil, &policy.BearerTokenOptions{
		AuthorizationHandler: policy.AuthorizationHandler{
			// Make an unauthenticated request
			OnRequest: func(*policy.Request, func(policy.TokenRequestOptions) error) error {
				return nil
			},
			// Inspect WWW-Authenticate header returned from challenge
			OnChallenge: func(req *policy.Request, resp *http.Response, authenticateAndAuthorize func(policy.TokenRequestOptions) error) error {
				authHeader := resp.Header.Get(headerWWWAuthenticate)

				// Parse the returned challenge
				parts := strings.Split(authHeader, " ")
				vals := map[string]string{}
				for _, part := range parts {
					subParts := strings.Split(part, "=")
					if len(subParts) == 2 {
						stripped := strings.ReplaceAll(subParts[1], "\"", "")
						stripped = strings.TrimSuffix(stripped, ",")
						vals[subParts[0]] = stripped
					}
				}

				u, err := url.Parse(vals[headerAuthorization])
				if err != nil {
					return fmt.Errorf("%w: %w", errInvalidAuthHeader, err)
				}
				tenantID := strings.ToLower(strings.Trim(u.Path, "/"))

				// check if valid tenantId
				if _, err = uuid.FromString(tenantID); err != nil {
					return fmt.Errorf("%w: %w", errInvalidTenantID, err)
				}

				// Note: "In api versions prior to 2023-09-30, the audience is included in the bearer challenge, but we recommend that partners
				// rely on hard-configuring the explicit values above for security reasons."

				// Authenticate from tenantID and audience
				return authenticateAndAuthorize(policy.TokenRequestOptions{
					Scopes:   []string{audience + "/.default"},
					TenantID: tenantID,
				})
			},
		},
	})
}
