package azureclient

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
)

// keys must be lower case
var apiVersions = map[string]string{
	"microsoft.authorization":                 "2018-09-01-preview",
	"microsoft.authorization/denyassignments": "2018-07-01-preview",
	"microsoft.authorization/roledefinitions": "2018-01-01-preview",
	"microsoft.compute":                       "2020-12-01",
	"microsoft.compute/disks":                 "2019-03-01",
	"microsoft.compute/snapshots":             "2020-05-01",
	"microsoft.compute/diskencryptionsets":    "2021-04-01",
	"microsoft.containerregistry":             "2020-11-01-preview",
	"microsoft.documentdb":                    "2021-01-15",
	"microsoft.insights":                      "2018-03-01",
	"microsoft.keyvault":                      "2019-09-01",
	"microsoft.managedidentity":               "2018-11-30",
	"microsoft.network":                       "2020-08-01",
	"microsoft.network/dnszones":              "2018-05-01",
	"microsoft.network/privatednszones":       "2018-09-01",
	"microsoft.storage":                       "2019-04-01",
}

// APIVersion gets the APIVersion from a full resource type
func APIVersion(typ string) string {
	t := strings.ToLower(typ)

	for {
		if apiVersion, ok := apiVersions[t]; ok {
			return apiVersion
		}

		i := strings.LastIndexByte(t, '/')
		if i == -1 {
			break
		}

		t = t[:i]
	}

	return ""
}
