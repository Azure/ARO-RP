package instancemetadata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"os"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

func NewDev(checkEnv bool) (InstanceMetadata, error) {
	if checkEnv {
		for _, key := range []string{
			"AZURE_SUBSCRIPTION_ID",
			"AZURE_TENANT_ID",
			"LOCATION",
			"RESOURCEGROUP",
		} {
			if _, found := os.LookupEnv(key); !found {
				return nil, fmt.Errorf("environment variable %q unset (development mode)", key)
			}
		}
	}

	environment := azureclient.PublicCloud
	if value, found := os.LookupEnv("AZURE_ENVIRONMENT"); found {
		var err error
		environment, err = azureclient.EnvironmentFromName(value)
		if err != nil {
			return nil, err
		}
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return &instanceMetadata{
		hostname:       hostname,
		tenantID:       os.Getenv("AZURE_TENANT_ID"),
		subscriptionID: os.Getenv("AZURE_SUBSCRIPTION_ID"),
		location:       os.Getenv("LOCATION"),
		resourceGroup:  os.Getenv("RESOURCEGROUP"),
		environment:    &environment,
	}, nil
}
