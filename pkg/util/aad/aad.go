package aad

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"time"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
)

const (
	defaultTimeout = 5 * time.Minute
)

type TokenClient interface {
	GetToken(ctx context.Context, log *logrus.Entry, clientID, clientSecret, tenantID string, aadEndpoint, resource string) (*adal.ServicePrincipalToken, error)
}

type tokenClient struct{}

func NewTokenClient() TokenClient {
	return &tokenClient{}
}

func (tc *tokenClient) GetToken(ctx context.Context, log *logrus.Entry, clientID, clientSecret, tenantID, aadEndpoint, resource string) (*adal.ServicePrincipalToken, error) {
	spToken, err := newServicePrincipalToken(clientID, clientSecret, tenantID, aadEndpoint, resource)
	if err != nil {
		return spToken, err
	}

	tokenAuthorizer := refreshable.NewAuthorizer(spToken)

	err = authenticateServicePrincipalToken(ctx, log, tokenAuthorizer, defaultTimeout)
	return spToken, err
}

func refreshContext(ctx context.Context, authorizer refreshable.Authorizer, log *logrus.Entry) (bool, error) {
	done, err := authorizer.RefreshWithContext(ctx, log)
	if err != nil {
		err = api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalCredentials, "properties.servicePrincipalProfile", "The provided service principal credentials are invalid.")
	}
	return done, err
}

// GetToken authenticates in the customer's tenant as the cluster service
// principal and returns a token.
func newServicePrincipalToken(clientID, clientSecret, tenantID, aadEndpoint, resource string) (*adal.ServicePrincipalToken, error) {
	conf := auth.ClientCredentialsConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TenantID:     tenantID,
		Resource:     resource,
		AADEndpoint:  aadEndpoint,
	}

	spToken, err := conf.ServicePrincipalToken()
	if err != nil {
		return nil, err
	}

	return spToken, nil
}

// AuthenticateServicePrincipalToken authenticates in the customer's tenant as the cluster service principal and returns a token.
func authenticateServicePrincipalToken(ctx context.Context, log *logrus.Entry, authorizer refreshable.Authorizer, timeout time.Duration) error {
	// during credentials rotation this can take time to propagate
	// it is overridable so we can have unit tests pass/fail quicker
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var err error
	done := false
	// NOTE: Do not override err with the error returned by
	// wait.PollImmediateUntil. Doing this will not propagate the latest error
	// to the user in case when wait exceeds the timeout
	_ = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		done, err = refreshContext(ctx, authorizer, log)
		if !done || err != nil {
			return false, err
		}

		p := &jwt.Parser{}
		claims := jwt.MapClaims{}
		_, _, err = p.ParseUnverified(authorizer.OAuthToken(), claims)
		if err != nil {
			err = api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalToken, "properties.servicePrincipalProfile", "The provided service principal generated an invalid token.")
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
		return err
	}

	if !done && authorizer.LastError() != nil {
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidServicePrincipalCredentials,
			"properties.servicePrincipalProfile",
			"The provided service principal credentials are invalid.")
	}

	return nil
}

// GetObjectId extracts the "oid" claim from a given access jwtToken
func GetObjectId(jwtToken string) (string, error) {
	p := jwt.NewParser(jwt.WithoutClaimsValidation())
	c := &custom{}
	_, _, err := p.ParseUnverified(jwtToken, c)
	if err != nil {
		return "", err
	}
	return c.ObjectId, nil
}

type custom struct {
	ObjectId string `json:"oid"`
	jwt.StandardClaims
}
