package rpauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"os"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/Azure/ARO-RP/pkg/util/deployment"
)

type RPAuthorizer interface {
	NewRPAuthorizer(resource string) (autorest.Authorizer, error)
}

type devRPAuthorizer struct{}

func (devRPAuthorizer) NewRPAuthorizer(resource string) (autorest.Authorizer, error) {
	config := &auth.ClientCredentialsConfig{
		ClientID:     os.Getenv("AZURE_RP_CLIENT_ID"),
		ClientSecret: os.Getenv("AZURE_RP_CLIENT_SECRET"),
		TenantID:     os.Getenv("AZURE_TENANT_ID"),
		Resource:     resource,
		AADEndpoint:  azure.PublicCloud.ActiveDirectoryEndpoint,
	}

	return config.Authorizer()
}

type prodRPAuthorizer struct{}

func (prodRPAuthorizer) NewRPAuthorizer(resource string) (autorest.Authorizer, error) {
	return auth.NewAuthorizerFromEnvironmentWithResource(resource)
}

func New(deploymentMode deployment.Mode) (RPAuthorizer, error) {
	if deploymentMode == deployment.Development {
		for _, key := range []string{
			"AZURE_RP_CLIENT_ID",
			"AZURE_RP_CLIENT_SECRET",
			"AZURE_TENANT_ID",
		} {
			if _, found := os.LookupEnv(key); !found {
				return nil, fmt.Errorf("environment variable %q unset (development mode)", key)
			}
		}

		return &devRPAuthorizer{}, nil
	}

	return &prodRPAuthorizer{}, nil
}
