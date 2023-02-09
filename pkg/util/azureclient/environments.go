package azureclient

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest/azure"
)

// AROEnvironment contains additional, cloud-specific information needed by ARO.
type AROEnvironment struct {
	azure.Environment
	ActualCloudName          string
	GenevaMonitoringEndpoint string
	AppSuffix                string
	AppLensEndpoint          string
	AppLensScope             string
	AppLensTenantID          string
	AuthzRemotePDPEndPoint   string
	azidentity.AuthorityHost
	AzureRbacPDPEnvironment
}

// AzureRbacPDPEnvironment contains cloud specific instance of Authz RBAC PDP Remote Server
type AzureRbacPDPEnvironment struct {
	Endpoint   string
	OAuthScope string
}

var (
	// PublicCloud contains additional ARO information for the public Azure cloud environment.
	PublicCloud = AROEnvironment{
		Environment:              azure.PublicCloud,
		ActualCloudName:          "AzureCloud",
		GenevaMonitoringEndpoint: "https://gcs.prod.monitoring.core.windows.net/",
		AppSuffix:                "aro.azure.com",
		AppLensEndpoint:          "https://diag-runtimehost-prod.trafficmanager.net/api/invoke",
		AppLensScope:             "b9a1efcd-32ee-4330-834c-c04eb00f4b33",
		AppLensTenantID:          "72f988bf-86f1-41af-91ab-2d7cd011db47",
		AuthorityHost:            azidentity.AzurePublicCloud,
		AzureRbacPDPEnvironment: AzureRbacPDPEnvironment{
			Endpoint:   "https://%s.authorization.azure.net/providers/Microsoft.Authorization/checkAccess",
			OAuthScope: "https://authorization.azure.net/.default",
		},
	}

	// USGovernmentCloud contains additional ARO information for the US Gov cloud environment.
	USGovernmentCloud = AROEnvironment{
		Environment:              azure.USGovernmentCloud,
		ActualCloudName:          "AzureUSGovernment",
		GenevaMonitoringEndpoint: "https://gcs.monitoring.core.usgovcloudapi.net/",
		AppSuffix:                "aro.azure.us",
		AppLensEndpoint:          "https://diag-runtimehost-prod-bn1-001.azurewebsites.us/api/invoke",
		AppLensScope:             "https://microsoft.onmicrosoft.com/runtimehost",
		AppLensTenantID:          "cab8a31a-1906-4287-a0d8-4eef66b95f6e",
		AuthorityHost:            azidentity.AzureGovernment,
		AzureRbacPDPEnvironment: AzureRbacPDPEnvironment{
			Endpoint:   "https://%s.authorization.azure.us/providers/Microsoft.Authorization/checkAccess",
			OAuthScope: "https://authorization.azure.us/.default",
		},
	}
)

// EnvironmentFromName returns the AROEnvironment corresponding to the common name specified.
func EnvironmentFromName(name string) (AROEnvironment, error) {
	switch strings.ToUpper(name) {
	case "AZUREPUBLICCLOUD":
		return PublicCloud, nil
	case "AZUREUSGOVERNMENTCLOUD":
		return USGovernmentCloud, nil
	}
	return AROEnvironment{}, fmt.Errorf("cloud environment %q is unsupported by ARO", name)
}
