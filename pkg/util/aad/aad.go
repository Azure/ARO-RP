package aad

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
)

// TokenMaker is an interface which has an AuthenticateAndGetToken method
type TokenMaker interface {
	AuthenticateAndGetToken(ctx context.Context, log *logrus.Entry, oc *api.OpenShiftCluster,
		resource string) (instancemetadata.ServicePrincipalToken, error)
}

// TokenFactory contains a NewToken function used to create a new token
type TokenFactory struct {
	NewToken      func(conf auth.ClientCredentialsConfig) (instancemetadata.ServicePrincipalToken, error)
	RetryInterval time.Duration
	Timeout       time.Duration
}

// AuthenticateAndGetToken authenticates in the customer's tenant as the cluster service
// principal and returns a token.  It retries in the cases below.  Unfortunately
// there doesn't seem to be a way to distinguish whether these cases occur due
// to misconfiguration or AAD propagation delays.
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
func (tf TokenFactory) AuthenticateAndGetToken(ctx context.Context, log *logrus.Entry, oc *api.OpenShiftCluster,
	resource string) (instancemetadata.ServicePrincipalToken, error) {

	spp := &oc.Properties.ServicePrincipalProfile

	conf := auth.NewClientCredentialsConfig(spp.ClientID, string(spp.ClientSecret), spp.TenantID)
	conf.Resource = resource

	token, err := tf.NewToken(conf)
	if err != nil {
		return nil, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, tf.Timeout)
	defer cancel()

	// NOTE: Do not override err with the error returned by wait.PollImmediateUntil.
	// Doing this will not propagate the latest error to the user in case when wait exceeds the timeout
	wait.PollImmediateUntil(tf.RetryInterval, func() (bool, error) {
		err = token.RefreshWithContext(ctx)
		if err != nil {
			isAADSTS700016 := strings.Contains(err.Error(), "AADSTS700016")

			// populate err with a user-facing error that will be visible if
			// we're not successful.
			log.Info(err)
			err = api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalCredentials, "properties.servicePrincipalProfile", "The provided service principal credentials are invalid.")

			if isAADSTS700016 {
				return false, nil
			}

			return false, err
		}

		p := &jwt.Parser{}
		claims := jwt.MapClaims{}
		_, _, err = p.ParseUnverified(token.OAuthToken(), claims)
		if err != nil {
			return false, err
		}

		for _, claim := range []string{"altsecid", "oid", "puid"} {
			if _, found := claims[claim]; found {
				return true, nil
			}
		}

		// populate err with a user-facing error that will be visible if we're
		// not successful.
		log.Info("token does not contain the required claims")
		err = api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalClaims, "properties.servicePrincipalProfile", "The provided service principal does not give an access token with at least one of the claims 'altsecid', 'oid' or 'puid'.")

		return false, nil
	}, timeoutCtx.Done())
	if err != nil {
		return nil, err
	}

	return token, nil
}
