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
var (
	OCPv4190 = NewVersion(4, 19, 0)
	OCPv440  = NewVersion(4, 4, 0)
)

var GitCommit = "unknown"

type Stream struct {
	Version  Version `json:"version"`
	PullSpec string  `json:"-"`
}

// Install stream data for production and INT has moved to RP-Config.
// This default is left here ONLY for use by local development mode,
// until we can come up with a better solution.
var DefaultInstallStream = Stream{
	Version:  NewVersion(4, 17, 44),
	PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:e3d5a7ccc804f95867a4fa9b9802739898be8814a429368521b12d7822de51a0",
}

// FluentbitImage contains the location of the Fluentbit container image
func FluentbitImage(acrDomain string) string {
	// https://github.com/microsoft/azurelinux/releases
	return acrDomain + "/fluentbit:4.2.2-cm20260102@sha256:48e57180d5d56c8170e850fbd0e8abb1c95805e3f7e1f84340c5ba8426109bbf"
}

// MdmImage contains the location of the MDM container image
// https://eng.ms/docs/products/geneva/collect/references/linuxcontainers
func MdmImage(acrDomain string) string {
	return acrDomain + "/geneva/distroless/mdm:2.202601131623.0-20260113-1@sha256:ca99b167e2463f0ba4008b65661524694340f23666959fbf06d6d1c169d5d699"
}

// MdsdImage contains the location of the MDSD container image
// https://eng.ms/docs/products/geneva/collect/references/linuxcontainers
func MdsdImage(acrDomain string) string {
	return acrDomain + "/geneva/distroless/mdsd:1.37.3-20251121-1@sha256:d088afef6e2614448c46e528de818b314fd52e42e9eb4eaa5cc1ceb70d86a204"
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
