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
	Version:  NewVersion(4, 16, 30),
	PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:7aacace57ab6ec468dd98b0b3e0f3fc440b29afce21b90bd716fed0db487e9e9",
}

// FluentbitImage contains the location of the Fluentbit container image
func FluentbitImage(acrDomain string) string {
	// https://github.com/microsoft/azurelinux/releases
	return acrDomain + "/fluentbit:1.9.10-cm20250429@sha256:a115a12041f48404a21fb678e3f853b531ca6d9b9935482b67b400427ffbdfef"
}

// MdmImage contains the location of the MDM container image
// https://eng.ms/docs/products/geneva/collect/references/linuxcontainers
func MdmImage(acrDomain string) string {
	return acrDomain + "/distroless/genevamdm:2.2025.404.1254-77220c-20250406t1133@sha256:8e89bfec19c81398afa0ec51a97d748cc6b7b85cf9440dd1c7ea75b24302fe55"
}

// MdsdImage contains the location of the MDSD container image
// https://eng.ms/docs/products/geneva/collect/references/linuxcontainers
func MdsdImage(acrDomain string) string {
	return acrDomain + "/distroless/genevamdsd:mariner_20250225.2@sha256:da59ef7cfe3b0b9b6b453930cc629605cf3528ed11dbb88cdc50a38633198add"
}

// MUOImage contains the location of the Managed Upgrade Operator container image
func MUOImage(acrDomain string) string {
	return acrDomain + "/app-sre/managed-upgrade-operator:v0.1.1202-g118c178"
}

// GateKeeperImage contains the location of the GateKeeper container image
func GateKeeperImage(acrDomain string) string {
	return acrDomain + "/gatekeeper:v3.19.1"
}

// MiseImage contains the location of the Mise container image
func MiseImage(acrDomain string) string {
	return acrDomain + "/mise:1.0.02773.115-mise-cbl-mariner2.0-distroless"
}

func OTelImage(acrDomain string) string {
	return "mcr.microsoft.com/oss/otel/opentelemetry-collector-contrib:0.95.0-linux-amd64"
}
