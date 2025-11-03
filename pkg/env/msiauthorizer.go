package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"os"

	"github.com/jongio/azidext/go/azidext"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest"
)

type MSIContext string

const (
	MSIContextRP      MSIContext = "RP"
	MSIContextGateway MSIContext = "GATEWAY"
)

const EnvUseWorkloadIdentity = "ARO_RP_WORKLOAD_IDENTITY"

func (c *core) NewMSITokenCredential() (azcore.TokenCredential, error) {
	if !c.IsLocalDevelopmentMode() {
		// If ARO_RP_WORKLOAD_IDENTITY is set, use a WorkloadIdentity credential
		// for RP authentication to FPSP keyvault instead
		useWorkloadIdentity := os.Getenv(EnvUseWorkloadIdentity)
		if useWorkloadIdentity != "" {
			options := c.Environment().WorkloadIdentityCredentialOptions()
			return azidentity.NewWorkloadIdentityCredential(options)
		}

		options := c.Environment().ManagedIdentityCredentialOptions("")
		return azidentity.NewManagedIdentityCredential(options)
	}

	var msiContext string
	if c.service == SERVICE_GATEWAY {
		msiContext = string(MSIContextGateway)
	} else {
		msiContext = string(MSIContextRP)
	}

	tenantIdKey := "AZURE_TENANT_ID"
	azureClientIdKey := "AZURE_" + msiContext + "_CLIENT_ID"
	azureClientSecretKey := "AZURE_" + msiContext + "_CLIENT_SECRET"

	if err := ValidateVars(azureClientIdKey, azureClientSecretKey, tenantIdKey); err != nil {
		return nil, fmt.Errorf("%v (development mode)", err.Error())
	}

	tenantId := os.Getenv(tenantIdKey)
	azureClientId := os.Getenv(azureClientIdKey)
	azureClientSecret := os.Getenv(azureClientSecretKey)

	options := c.Environment().ClientSecretCredentialOptions()

	return azidentity.NewClientSecretCredential(tenantId, azureClientId, azureClientSecret, options)
}

func (c *core) NewMSIAuthorizer(scope string) (autorest.Authorizer, error) {
	// To prevent creating multiple authorisers with independent token
	// refreshes, store them in a cache per-scope when created
	auth, ok := c.msiAuthorizers[scope]
	if ok {
		return auth, nil
	}

	token, err := c.NewMSITokenCredential()
	if err != nil {
		return nil, err
	}
	auth = azidext.NewTokenCredentialAdapter(token, []string{scope})
	c.msiAuthorizers[scope] = auth
	return auth, nil
}
