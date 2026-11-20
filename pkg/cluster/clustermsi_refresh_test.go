package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/Azure/msi-dataplane/pkg/dataplane"

	"github.com/Azure/ARO-RP/pkg/api"
	utilarm "github.com/Azure/ARO-RP/pkg/util/arm"
	utilazcore "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
	mock_azsecrets "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azsecrets"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

type msiDataplaneCredentialFunc func(context.Context, policy.TokenRequestOptions) (azcore.AccessToken, error)

func (f msiDataplaneCredentialFunc) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return f(ctx, options)
}

type msiDataplaneTransportFunc func(*http.Request) (*http.Response, error)

func (f msiDataplaneTransportFunc) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestEnsureClusterMsiCertificateCredentialRefresh(t *testing.T) {
	originalBackoff := utilarm.TransientBackoff
	utilarm.TransientBackoff = wait.Backoff{Steps: 4, Duration: time.Microsecond}
	t.Cleanup(func() { utilarm.TransientBackoff = originalBackoff })

	const clusterID = "00000000-0000-0000-0000-000000000000"
	const identityID = "/subscriptions/" + clusterID + "/resourceGroups/test/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test"
	cluster := &api.OpenShiftCluster{
		Identity: &api.ManagedServiceIdentity{
			IdentityURL: "https://msi.example/identity?routing=preserved",
			UserAssignedIdentities: map[string]api.UserAssignedIdentity{
				identityID: {},
			},
		},
	}

	rebuildError := errors.New("credential rebuild failed")
	rebuildAuthorizationError := &azcore.ResponseError{StatusCode: http.StatusForbidden, ErrorCode: "AuthorizationFailed"}
	for _, tt := range []struct {
		name                 string
		statuses             []int
		code                 string
		rebuildError         error
		rebuildFailures      int
		cancelOnError        bool
		cancelOnRebuildError bool
		wantError            error
		wantTokens           []string
		wantBuilds           int
		wantStatusCode       int
	}{
		{
			name:       "normal challenge does not rebuild",
			statuses:   []int{http.StatusOK},
			wantTokens: []string{"token-1"},
			wantBuilds: 1,
		},
		{
			name:       "forbidden rebuilds before retrying",
			statuses:   []int{http.StatusForbidden, http.StatusOK},
			wantTokens: []string{"token-1", "token-2"},
			wantBuilds: 2,
		},
		{
			name:       "final unauthorized rebuilds after challenge handling",
			statuses:   []int{http.StatusUnauthorized, http.StatusOK},
			wantTokens: []string{"token-1", "token-2"},
			wantBuilds: 2,
		},
		{
			name:       "authorization code without forbidden rebuilds",
			statuses:   []int{http.StatusBadRequest, http.StatusOK},
			code:       "AuthorizationFailed",
			wantTokens: []string{"token-1", "token-2"},
			wantBuilds: 2,
		},
		{
			name:           "persistent authorization failure is bounded",
			statuses:       []int{http.StatusForbidden, http.StatusForbidden, http.StatusForbidden, http.StatusForbidden},
			wantTokens:     []string{"token-1", "token-2", "token-3", "token-4"},
			wantBuilds:     4,
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:            "rebuild failure recovers within retry budget",
			statuses:        []int{http.StatusForbidden, http.StatusOK},
			rebuildError:    rebuildError,
			rebuildFailures: 1,
			wantTokens:      []string{"token-1", "token-3"},
			wantBuilds:      3,
		},
		{
			name:            "persistent rebuild failure is bounded",
			statuses:        []int{http.StatusForbidden},
			rebuildError:    rebuildError,
			rebuildFailures: 3,
			wantError:       rebuildError,
			wantTokens:      []string{"token-1"},
			wantBuilds:      4,
		},
		{
			name:            "authorization failure during rebuild is bounded",
			statuses:        []int{http.StatusForbidden},
			rebuildError:    rebuildAuthorizationError,
			rebuildFailures: 3,
			wantError:       rebuildAuthorizationError,
			wantTokens:      []string{"token-1"},
			wantBuilds:      4,
		},
		{
			name:            "non authorization fetch failure after rebuild recovery stops",
			statuses:        []int{http.StatusForbidden, http.StatusBadRequest},
			rebuildError:    rebuildError,
			rebuildFailures: 1,
			wantTokens:      []string{"token-1", "token-3"},
			wantBuilds:      3,
			wantStatusCode:  http.StatusBadRequest,
		},
		{
			name:                 "cancellation during rebuild failure prevents retrying",
			statuses:             []int{http.StatusForbidden},
			rebuildError:         rebuildError,
			rebuildFailures:      2,
			cancelOnRebuildError: true,
			wantError:            context.Canceled,
			wantTokens:           []string{"token-1"},
			wantBuilds:           2,
		},
		{
			name:          "cancellation prevents rebuilding and retrying",
			statuses:      []int{http.StatusForbidden},
			cancelOnError: true,
			wantTokens:    []string{"token-1"},
			wantBuilds:    1,
		},
		{
			name:           "non authorization failure does not rebuild",
			statuses:       []int{http.StatusBadRequest},
			wantTokens:     []string{"token-1"},
			wantBuilds:     1,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "throttling does not rebuild",
			statuses:       []int{http.StatusTooManyRequests},
			wantTokens:     []string{"token-1"},
			wantBuilds:     1,
			wantStatusCode: http.StatusTooManyRequests,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			builds := 0
			credential, err := refreshable.NewTokenCredential(func() (utilazcore.TokenCredential, error) {
				builds++
				if builds > 1 && builds <= tt.rebuildFailures+1 {
					if tt.cancelOnRebuildError {
						cancel()
					}
					return nil, tt.rebuildError
				}
				token := fmt.Sprintf("token-%d", builds)
				return msiDataplaneCredentialFunc(func(_ context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
					require.Equal(t, "challenge-tenant", options.TenantID)
					require.Equal(t, []string{"https://msi.example/.default"}, options.Scopes)
					return azcore.AccessToken{Token: token, ExpiresOn: time.Now().Add(time.Hour)}, nil
				}), nil
			})
			require.NoError(t, err)

			var tokens, bodies []string
			challenges := 0
			options := &policy.ClientOptions{
				Retry: policy.RetryOptions{MaxRetries: -1},
				Transport: msiDataplaneTransportFunc(func(request *http.Request) (*http.Response, error) {
					require.Equal(t, http.MethodPost, request.Method)
					require.Equal(t, "/identity", request.URL.Path)
					require.Equal(t, "preserved", request.URL.Query().Get("routing"))
					header := http.Header{"Content-Type": []string{"application/json"}}
					status := http.StatusUnauthorized
					body := `{"error":{"code":"Unauthorized","message":"challenge"}}`
					if authorization := request.Header.Get("Authorization"); authorization == "" {
						challenges++
						header.Set("WWW-Authenticate", `Bearer authorization="https://login.microsoftonline.com/challenge-tenant"`)
					} else {
						tokens = append(tokens, strings.TrimPrefix(authorization, "Bearer "))
						requestBody, err := io.ReadAll(request.Body)
						require.NoError(t, err)
						bodies = append(bodies, string(requestBody))
						require.LessOrEqual(t, len(tokens), len(tt.statuses), "unexpected extra credential fetch")
						status = tt.statuses[len(tokens)-1]
						body = fmt.Sprintf(`{"explicit_identities":[{"resource_id":%q,"client_id":"test-client","client_secret":"test-certificate","tenant_id":"challenge-tenant","not_before":"2026-01-01T00:00:00Z","not_after":"2027-01-01T00:00:00Z","renew_after":"2026-12-01T00:00:00Z","cannot_renew_after":"2027-01-01T00:00:00Z"}]}`, identityID)
						if status != http.StatusOK {
							body = fmt.Sprintf(`{"error":{"code":%q,"message":"credential fetch failed"}}`, tt.code)
							if tt.cancelOnError {
								cancel()
							}
						}
					}
					return &http.Response{
						StatusCode: status,
						Header:     header,
						Body:       io.NopCloser(strings.NewReader(body)),
						Request:    request,
					}, nil
				}),
			}
			_, log := testlog.LogForTesting(t)
			factory := dataplane.NewClientFactory(credential, "https://msi.example", options)
			vault := mock_azsecrets.NewMockClient(gomock.NewController(t))
			secretName := dataplane.IdentifierForManagedIdentityCredentials(clusterID)
			vault.EXPECT().GetSecret(ctx, secretName, "", nil).Return(azsecrets.GetSecretResponse{}, &azcore.ResponseError{StatusCode: http.StatusNotFound})
			if tt.wantStatusCode == 0 && tt.wantError == nil && !tt.cancelOnError {
				vault.EXPECT().SetSecret(ctx, secretName, gomock.Any(), nil).DoAndReturn(
					func(context.Context, string, azsecrets.SetSecretParameters, *azsecrets.SetSecretOptions) (azsecrets.SetSecretResponse, error) {
						require.Equal(t, tt.wantTokens, tokens, "persist only after successful credential fetch")
						return azsecrets.SetSecretResponse{}, nil
					},
				)
			}

			result, err := EnsureClusterMsiCertificateWithParams(ctx, clusterID, cluster, time.Now, vault, factory, credential, log)
			switch {
			case tt.cancelOnError:
				require.ErrorIs(t, err, context.Canceled)
			case tt.wantError != nil:
				require.ErrorIs(t, err, tt.wantError)
			case tt.wantStatusCode != 0:
				var responseError *azcore.ResponseError
				require.ErrorAs(t, err, &responseError)
				require.Equal(t, tt.wantStatusCode, responseError.StatusCode)
			default:
				require.NoError(t, err)
				require.Equal(t, MsiCertificateRefreshResultCreated, result)
			}
			require.Equal(t, tt.wantTokens, tokens)
			require.Equal(t, tt.wantBuilds, builds)
			require.Equal(t, len(tokens), challenges)
			for _, body := range bodies {
				require.JSONEq(t, fmt.Sprintf(`{"identityIds":[%q]}`, identityID), body)
			}
		})
	}
}
