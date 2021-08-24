package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"os"

	"github.com/Azure/ARO-RP/pkg/api"
)

const InstallArchitectureVersion = api.ArchitectureVersionV2

const (
	DevClusterGenevaLoggingAccount       = "AROClusterLogs"
	DevClusterGenevaLoggingConfigVersion = "2.4"
	DevClusterGenevaLoggingNamespace     = "AROClusterLogs"
	DevClusterGenevaMetricsAccount       = "AzureRedHatOpenShiftCluster"
	DevGenevaLoggingEnvironment          = "Test"
	DevRPGenevaLoggingAccount            = "ARORPLogs"
	DevRPGenevaLoggingConfigVersion      = "3.7"
	DevRPGenevaLoggingNamespace          = "ARORPLogs"
	DevRPGenevaMetricsAccount            = "AzureRedHatOpenShiftRP"
)

var GitCommit = "unknown"

// InstallStream describes stream we are defaulting to for all new clusters
var InstallStream = &Stream{
	Version:  NewVersion(4, 7, 24),
	PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:06c95d1212f977af6270d5683ae4d963f7215a75177893335a96433de87ffefe",
}

// UpgradeStreams describes list of streams we support for upgrades
var (
	UpgradeStreams = []*Stream{
		InstallStream,
		{
			Version:  NewVersion(4, 6, 40),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:a8bc2a472611d499d99f347b3e2a7c385107e9c4e44b45d765a99338b566dd12",
		},
		{
			Version:  NewVersion(4, 5, 39),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:c4b9eb565c64df97afe7841bbcc0469daec7973e46ae588739cc30ea9062172b",
		},
		{
			Version:  NewVersion(4, 4, 33),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:a035dddd8a5e5c99484138951ef4aba021799b77eb9046f683a5466c23717738",
		},
	}
)

// FluentbitImage contains the location of the Fluentbit container image
func FluentbitImage(acrDomain string) string {
	return acrDomain + "/fluentbit:1.7.8-1"
}

// MdmImage contains the location of the MDM container image
func MdmImage(acrDomain string) string {
	// for the latest version see https://genevamondocs.azurewebsites.net/collect/references/linuxcontainers.html?q=container
	if os.Getenv("GENEVA_MDM_IMAGE_OVERRIDE") != "" {
		return os.Getenv("GENEVA_MDM_IMAGE_OVERRIDE")
	}

	return acrDomain + "/genevamdm:master_20210808.1"
}

// MdsdImage contains the location of the MDSD container image
func MdsdImage(acrDomain string) string {
	// for the latest version see https://genevamondocs.azurewebsites.net/collect/references/linuxcontainers.html?q=container
	if os.Getenv("GENEVA_MDSD_IMAGE_OVERRIDE") != "" {
		return os.Getenv("GENEVA_MDSD_IMAGE_OVERRIDE")
	}

	return acrDomain + "/genevamdsd:master_20210808.1"
}
