package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"log"
	"net/http"

	"github.com/form3tech-oss/jwt-go"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/azureclaim"
)

func (dv *dynamic) ValidateServicePrincipal(ctx context.Context, clientID, clientSecret, tenantID string) error {
	// TODO: once aad.GetToken is mockable, write a unit test for this function
	log.Print("ValidateServicePrincipal")

	token, err := aad.GetToken(ctx, dv.log, clientID, clientSecret, tenantID, dv.azEnv.ActiveDirectoryEndpoint, dv.azEnv.GraphEndpoint)
	if err != nil {
		return err
	}

	p := &jwt.Parser{}
	c := &azureclaim.AzureClaim{}
	_, _, err = p.ParseUnverified(token.OAuthToken(), c)
	if err != nil {
		return err
	}

	for _, role := range c.Roles {
		if role == "Application.ReadWrite.OwnedBy" {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalCredentials, "properties.servicePrincipalProfile", "The provided service principal must not have the Application.ReadWrite.OwnedBy permission.")
		}
	}

	return nil
}
