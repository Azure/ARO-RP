package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	jwt "github.com/golang-jwt/jwt/v4"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclaim"
)

func (dv *dynamic) ValidateServicePrincipal(ctx context.Context, spTokenCredential azcore.TokenCredential) error {
	dv.log.Print("ValidateServicePrincipal")

	tokenRequestOptions := policy.TokenRequestOptions{
		Scopes: []string{dv.azEnv.MicrosoftGraphScope},
	}
	token, err := spTokenCredential.GetToken(ctx, tokenRequestOptions)
	if err != nil {
		return err
	}

	p := jwt.NewParser()
	c := &azureclaim.AzureClaim{}
	_, _, err = p.ParseUnverified(token.Token, c)
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalCredentials, "properties.servicePrincipalProfile", err.Error())
	}

	for _, role := range c.Roles {
		if role == "Application.ReadWrite.OwnedBy" {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalCredentials, "properties.servicePrincipalProfile", "The provided service principal must not have the Application.ReadWrite.OwnedBy permission.")
		}
	}

	return nil
}
