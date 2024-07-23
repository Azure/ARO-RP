package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
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
	DevRPGenevaLoggingConfigVersion      = "4.3"
	DevRPGenevaLoggingNamespace          = "ARORPLogs"
	DevRPGenevaMetricsAccount            = "AzureRedHatOpenShiftRP"

	DevGatewayGenevaLoggingConfigVersion = "4.3"
)

var GitCommit = "unknown"

type Stream struct {
	Version  *Version `json:"version"`
	PullSpec string   `json:"-"`
}

// Install stream data for production and INT has moved to RP-Config.
// This default is left here ONLY for use by local development mode,
// until we can come up with a better solution.
var DefaultInstallStream = Stream{
	Version:  NewVersion(4, 13, 40),
	PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:c1f69e6137bc9cda2c6da56bafbc7ea969900acb5e5c349b1ebb2103b10b424f",
}

// FluentbitImage contains the location of the Fluentbit container image
func FluentbitImage(acrDomain string) string {
	// Latest stable release is 2.0.20240628
	// https://github.com/microsoft/azurelinux/releases
	return acrDomain + "/fluentbit:1.9.10-cm20240628@sha256:1d0b1789f4eb06f5c74fa5def306bd5095852882f63d91578ef589e8eab18765"
}

// MdmImage contains the location of the MDM container image
// https://eng.ms/docs/products/geneva/collect/references/linuxcontainers
func MdmImage(acrDomain string) string {
	return acrDomain + "/distroless/genevamdm:2.2024.626.1539-d1a6e7-20240715t0935@sha256:372fbc981bbfdf2b9a9d0ffdca2c51ed389b291a3bcff0401e9afb0c01605823"
}

// MdsdImage contains the location of the MDSD container image
// https://eng.ms/docs/products/geneva/collect/references/linuxcontainers
func MdsdImage(acrDomain string) string {
	return acrDomain + "/distroless/genevamdsd:mariner_20240711.1@sha256:20266be8d3de71ceaf6b864e70e295de5eecb5f796aad92bc303efcaf179a365"
}

// MUOImage contains the location of the Managed Upgrade Operator container image
func MUOImage(acrDomain string) string {
	return acrDomain + "/app-sre/managed-upgrade-operator:v0.1.952-44b631a"
}

// GateKeeperImage contains the location of the GateKeeper container image
func GateKeeperImage(acrDomain string) string {
	return acrDomain + "/gatekeeper:v3.15.1"
}
