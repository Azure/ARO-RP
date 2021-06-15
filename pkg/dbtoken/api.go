package dbtoken

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest/azure"
)

// Resource returns the dbtoken domain for the provided cloud name
func Resource(cloudName string) (string, error) {
	switch cloudName {
	case azure.PublicCloud.Name:
		return "https://dbtoken.aro.azure.com/", nil
	case azure.USGovernmentCloud.Name:
		return "https://dbtoken.aro.azure.us/", nil
	}
	return "", fmt.Errorf("unsupported cloud environment")
}

type tokenResponse struct {
	Token string `json:"token,omitempty"`
}
