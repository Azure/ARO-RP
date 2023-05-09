package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/form3tech-oss/jwt-go"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclaim"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

type ServicePrincipalValidator interface {
	ValidateServicePrincipal(ctx context.Context, tokenCredential azcore.TokenCredential) error
}

type dynamicServicePrincipal struct {
	log            *logrus.Entry
	authorizerType AuthorizerType
	azEnv          *azureclient.AROEnvironment
}

func NewServicePrincipalValidator(
	log *logrus.Entry,
	azEnv *azureclient.AROEnvironment,
	authorizerType AuthorizerType,
) ServicePrincipalValidator {
	return &dynamicServicePrincipal{
		log:            log,
		authorizerType: authorizerType,
		azEnv:          azEnv,
	}
}

func (dv *dynamicServicePrincipal) ValidateServicePrincipal(ctx context.Context, tokenCredential azcore.TokenCredential) error {
	dv.log.Print("ValidateServicePrincipal")

	tokenRequestOptions := policy.TokenRequestOptions{
		Scopes: []string{dv.azEnv.ActiveDirectoryGraphScope},
	}
	token, err := tokenCredential.GetToken(ctx, tokenRequestOptions)
	if err != nil {
		return err
	}

	p := &jwt.Parser{}
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
