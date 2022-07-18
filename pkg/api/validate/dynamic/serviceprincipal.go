package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/form3tech-oss/jwt-go"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/azureclaim"
)

type (
	GetTokenWrapper func(ctx context.Context, l *logrus.Entry, clientID, clientSecret, tenantID, ActiveDirectoryEndpoint, GraphEndpoint string) (*adal.ServicePrincipalToken, error) // Allows us to mock the getToken function
)

func GetServicePrincipalToken(ctx context.Context, log *logrus.Entry, clientID, clientSecret, tenantID, ActiveDirectoryEndpoint, GraphEndpoint string) (*adal.ServicePrincipalToken, error) {
	return aad.GetToken(ctx, log, clientID, clientSecret, tenantID, ActiveDirectoryEndpoint, GraphEndpoint)
}

// ValidateServicePrincipal ensures the provided service principal does not have Application.ReadWrite.OwnedBy permission.
func (dv *dynamic) ValidateServicePrincipal(ctx context.Context, clientID, clientSecret, tenantID string, azClaim azureclaim.AzureClaim, tokenWrapper GetTokenWrapper) error {
	dv.log.Print("ValidateServicePrincipal")

	token, err := tokenWrapper(ctx, dv.log, clientID, clientSecret, tenantID, dv.azEnv.ActiveDirectoryEndpoint, dv.azEnv.GraphEndpoint)
	if err != nil {
		return err
	}

	p := &jwt.Parser{}
	_, _, err = p.ParseUnverified(token.OAuthToken(), &azClaim)
	if err != nil {
		return err
	}

	for _, role := range azClaim.Roles {
		if role == "Application.ReadWrite.OwnedBy" {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalCredentials, "properties.servicePrincipalProfile", "The provided service principal must not have the Application.ReadWrite.OwnedBy permission.")
		}
	}
	return nil
}
