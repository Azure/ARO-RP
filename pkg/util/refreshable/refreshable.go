package refreshable

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
)

type Authorizer interface {
	autorest.Authorizer
	RefreshWithContext(context.Context, *logrus.Entry) (bool, error)
	LastError() error
	OAuthToken() string
}

type authorizer struct {
	autorest.Authorizer
	sp        *adal.ServicePrincipalToken
	lastError error
}

// RefreshWithContext attempts to refresh a service principal token.  It should
// be called from within a wait.Poll* loop and its return values match
// accordingly.  It requests a retry in the cases below.  Unfortunately there
// doesn't seem to be a way to distinguish whether these cases occur due to
// misconfiguration or AAD propagation delays.
//
// 1. `{"error": "unauthorized_client", "error_description": "AADSTS700016:
// Application with identifier 'xxx' was not found in the directory 'xxx'. This
// can happen if the application has not been installed by the administrator of
// the tenant or consented to by any user in the tenant. You may have sent your
// authentication request to the wrong tenant. ...", "error_codes": [700016]}`.
// This can be an indicator of AAD propagation delay.
//
// 2. Lack of an altsecid, puid or oid claim in the token.  Continuing would
// subsequently cause the ARM error `Code="InvalidAuthenticationToken"
// Message="The received access token is not valid: at least one of the claims
// 'puid' or 'altsecid' or 'oid' should be present. If you are accessing as an
// application please make sure service principal is properly created in the
// tenant."`.  I think this can be returned when the service principal
// associated with the application hasn't yet caught up with the application
// itself.
//
// 3. Network failures.  If the error is not an adal.TokenRefreshError, then
// it's likely a transient failure. For example, connection reset by peer.
//
// 4. If credentials are just created, they might fail with `adal.tokenRefreshError)
// adal: Refresh request failed. Status Code = '401'.  Response body:
// {"error":"invalid_client","error_description":"AADSTS7000215: Invalid client secret is provided.`
// Once aad starts propagating credentials you might see occasional success in authentication.
func (a *authorizer) RefreshWithContext(ctx context.Context, log *logrus.Entry) (bool, error) {
	a.lastError = a.sp.RefreshWithContext(ctx)
	if a.lastError == nil {
		return true, nil
	}

	log.Info(a.lastError)

	if !autorest.IsTokenRefreshError(a.lastError) ||
		azureerrors.IsUnauthorizedClientError(a.lastError) ||
		azureerrors.IsInvalidSecretError(a.lastError) {
		return false, nil
	}
	return false, a.lastError
}

func (a *authorizer) LastError() error {
	return a.lastError
}

func (a *authorizer) OAuthToken() string {
	return a.sp.OAuthToken()
}

func NewAuthorizer(sp *adal.ServicePrincipalToken) Authorizer {
	return &authorizer{
		Authorizer: autorest.NewBearerAuthorizer(sp),
		sp:         sp,
	}
}
