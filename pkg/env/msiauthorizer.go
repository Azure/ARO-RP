package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest"
	"github.com/jongio/azidext/go/azidext"
)

type MSIContext string

const (
	MSIContextRP      MSIContext = "RP"
	MSIContextGateway MSIContext = "GATEWAY"
)

func (c *core) NewMSIAuthorizer(msiContext MSIContext, scopes ...string) (autorest.Authorizer, error) {
	if !c.IsLocalDevelopmentMode() {
		options := c.Environment().ManagedIdentityCredentialOptions()
		tokenCredential, err := azidentity.NewManagedIdentityCredential(options)
		if err != nil {
			return nil, err
		}

		return azidext.NewTokenCredentialAdapter(tokenCredential, scopes), nil
	}

	tenantIdKey := "AZURE_TENANT_ID"
	azureClientIdKey := "AZURE_" + string(msiContext) + "_CLIENT_ID"
	azureClientSecretKey := "AZURE_" + string(msiContext) + "_CLIENT_SECRET"

	if err := ValidateVars(azureClientIdKey, azureClientSecretKey, tenantIdKey); err != nil {
		return nil, fmt.Errorf("%v (development mode)", err.Error())
	}

	tenantId := os.Getenv(tenantIdKey)
	azureClientId := os.Getenv(azureClientIdKey)
	azureClientSecret := os.Getenv(azureClientSecretKey)

	options := c.Environment().ClientSecretCredentialOptions()

	tokenCredential, err := azidentity.NewClientSecretCredential(tenantId, azureClientId, azureClientSecret, options)
	if err != nil {
		return nil, err
	}

	return azidext.NewTokenCredentialAdapter(tokenCredential, scopes), nil
}
