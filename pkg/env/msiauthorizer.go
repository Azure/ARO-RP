package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest"
	"github.com/jongio/azidext/go/azidext"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

type MSIContext string

const (
	MSIContextRP      MSIContext = "RP"
	MSIContextGateway MSIContext = "GATEWAY"
)

func (c *core) NewMSITokenCredential(msiContext MSIContext, scopes ...string) (azcore.TokenCredential, error) {
	if !c.IsLocalDevelopmentMode() {
		options := c.Environment().ManagedIdentityCredentialOptions()
		return azidentity.NewManagedIdentityCredential(options)
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

	return azidentity.NewClientSecretCredential(tenantId, azureClientId, azureClientSecret, options)
}

func (c *core) NewMSIAuthorizer(msiContext MSIContext, scopes ...string) (autorest.Authorizer, error) {
	token, err := c.NewMSITokenCredential(msiContext, scopes...)
	if err != nil {
		return nil, err
	}
	return azidext.NewTokenCredentialAdapter(token, scopes), nil
}
