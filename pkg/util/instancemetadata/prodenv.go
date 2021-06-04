package instancemetadata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/util/env"
)

type prodenv struct {
	instanceMetadata

	newServicePrincipalTokenFromMSI func(string, string) (ServicePrincipalToken, error)
}

func newProdFromEnv(ctx context.Context) (InstanceMetadata, error) {
	p := &prodenv{
		newServicePrincipalTokenFromMSI: func(msiEndpoint, resource string) (ServicePrincipalToken, error) {
			return adal.NewServicePrincipalTokenFromMSI(msiEndpoint, resource)
		},
	}

	osenv := env.NewOsEnv()
	err := p.populateInstanceMetadata(osenv)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *prodenv) populateInstanceMetadata(osenv env.EnvironmentSource) error {

	for _, key := range []string{
		"AZURE_ENVIRONMENT",
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_TENANT_ID",
		"LOCATION",
		"RESOURCEGROUP",
	} {
		if _, found := osenv.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	// optional env variables
	// * HOSTNAME_OVERRIDE: defaults to os.Hostname()

	envStr := osenv.Getenv("AZURE_ENVIRONMENT")
	environment, err := azure.EnvironmentFromName(envStr)
	if err != nil {
		return err
	}
	p.environment = &environment

	p.subscriptionID = osenv.Getenv("AZURE_SUBSCRIPTION_ID")
	p.tenantID = osenv.Getenv("AZURE_TENANT_ID")
	p.location = osenv.Getenv("LOCATION")
	p.resourceGroup = osenv.Getenv("RESOURCEGROUP")
	p.hostname = osenv.Getenv("HOSTNAME_OVERRIDE") // empty string returned if not set

	if p.hostname == "" {
		hostname, err := os.Hostname()
		if err == nil {
			p.hostname = hostname
		}
	}

	return nil
}
