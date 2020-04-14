package azureclient

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"strings"
)

var APIVersions = map[string]string{
	"Microsoft.Authorization":                 "2018-09-01-preview",
	"Microsoft.Authorization/denyAssignments": "2018-07-01-preview",
	"Microsoft.Authorization/roleDefinitions": "2018-01-01-preview",
	"Microsoft.Compute":                       "2019-03-01",
	"Microsoft.ContainerRegistry":             "2019-05-01",
	"Microsoft.DocumentDB":                    "2019-08-01",
	"Microsoft.Insights":                      "2018-03-01",
	"Microsoft.KeyVault":                      "2016-10-01",
	"Microsoft.ManagedIdentity":               "2018-11-30",
	"Microsoft.Network":                       "2019-07-01",
	"Microsoft.Network/dnsZones":              "2018-05-01",
	"Microsoft.Network/privateDnsZones":       "2018-09-01",
	"Microsoft.Storage":                       "2019-04-01",
}

// APIVersionForType gets the APIVersion from a full resource type
func APIVersionForType(typ string) (string, error) {
	t := typ

	for {
		if apiVersion, ok := APIVersions[t]; ok {
			return apiVersion, nil
		}

		i := strings.LastIndexByte(t, '/')
		if i == -1 {
			break
		}

		t = t[:i]
	}

	return "", fmt.Errorf("API version not found for type %s", typ)
}
