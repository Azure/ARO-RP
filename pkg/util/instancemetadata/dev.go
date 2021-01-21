package instancemetadata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"os"

	"github.com/Azure/go-autorest/autorest/azure"
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

	environment := azure.PublicCloud
	if value, found := os.LookupEnv("AZURE_ENVIRONMENT"); found {
		var err error
		environment, err = azure.EnvironmentFromName(value)
		if err != nil {
			return nil, err
		}
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	// awful heuristics, remove when the value of CLUSTER_RESOURCEGROUP in the
	// pipelines has been updated to the cluster's RG and we can assume
	// RESOURCEGROUP is the CI environment.
	clusterRG, exists := os.LookupEnv("CLUSTER_RESOURCEGROUP")
	if !exists {
		clusterRG = os.Getenv("CLUSTER")
	}

	return &instanceMetadata{
		hostname:       hostname,
		tenantID:       os.Getenv("AZURE_TENANT_ID"),
		subscriptionID: os.Getenv("AZURE_SUBSCRIPTION_ID"),
		location:       os.Getenv("LOCATION"),
		resourceGroup:  clusterRG,
		environment:    &environment,
	}, nil
}
