package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"os"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

// RPAuthorizer returns an authorizer for the specified resource from the RP
// MSI.
func RPAuthorizer(resource string) (autorest.Authorizer, error) {
	if strings.ToLower(os.Getenv("RP_MODE")) == "development" {
		config := &auth.ClientCredentialsConfig{
			ClientID:     os.Getenv("AZURE_RP_CLIENT_ID"),
			ClientSecret: os.Getenv("AZURE_RP_CLIENT_SECRET"),
			TenantID:     os.Getenv("AZURE_TENANT_ID"),
			Resource:     resource,
			AADEndpoint:  azure.PublicCloud.ActiveDirectoryEndpoint,
		}

		return config.Authorizer()
	}

	return auth.NewAuthorizerFromEnvironmentWithResource(resource)
}
