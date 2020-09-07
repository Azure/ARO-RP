package instancemetadata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"os"
)

func NewDev() (InstanceMetadata, error) {
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

	return &instanceMetadata{
		tenantID:       os.Getenv("AZURE_TENANT_ID"),
		subscriptionID: os.Getenv("AZURE_SUBSCRIPTION_ID"),
		location:       os.Getenv("LOCATION"),
		resourceGroup:  os.Getenv("RESOURCEGROUP"),
	}, nil
}
