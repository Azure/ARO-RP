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

	MUOImageTag = "v0.1.1297-ge922e64"
)

// OCP versions - declared as major, minor, z
var OCPv4190 = NewVersion(4, 19, 0)
var OCPv440 = NewVersion(4, 4, 0)

var GitCommit = "unknown"

type Stream struct {
	Version  Version `json:"version"`
	PullSpec string  `json:"-"`
}

// Install stream data for production and INT has moved to RP-Config.
// This default is left here ONLY for use by local development mode,
// until we can come up with a better solution.
var DefaultInstallStream = Stream{
	Version:  NewVersion(4, 17, 45),
	PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:e3d5a7ccc804f95867a4fa9b9802739898be8814a429368521b12d7822de51a0",
}

// FluentbitImage contains the location of the Fluentbit container image
func FluentbitImage(acrDomain string) string {
	// https://github.com/microsoft/azurelinux/releases
	return acrDomain + "/fluentbit:4.0.4-cm20250701@sha256:72e56529c56b43eb6e375807dc1924b24705138ec3f3788c8a6cdf7c4ad36e63"
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
	return acrDomain + "/app-sre/managed-upgrade-operator:" + MUOImageTag
}

// GateKeeperImage contains the location of the GateKeeper container image
func GateKeeperImage(acrDomain string) string {
	return acrDomain + "/gatekeeper:v3.19.2"
}

// MiseImage contains the location of the Mise container image
func MiseImage(acrDomain string) string {
	return acrDomain + "/mise:1.0.03103.537-mise-cbl-mariner2.0-distroless"
}

func OTelImage(acrDomain string) string {
	return "mcr.microsoft.com/oss/otel/opentelemetry-collector-contrib:0.95.0-linux-amd64"
}
