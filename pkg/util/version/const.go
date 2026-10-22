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

	MUOImageTag = "v0.1.1445-g512fbce"
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
	Version:  NewVersion(4, 19, 24),
	PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:3ef832b8bb0d56331035ba54af36c36be46d6c6dc1a41e300055692f02bb001d",
}

// MdmImage contains the location of the MDM container image
// https://eng.ms/docs/products/geneva/collect/references/linuxcontainers
func MdmImage(acrDomain string) string {
	return acrDomain + "/geneva/distroless/mdm:2.202610022014.0-20261005-1@sha256:cad8c6b5faf6c11f7b2c3096e1bc229e28cb6b1b0b2dbc5945302f806b1ce3f9"
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
	return acrDomain + "/msftonly/mise:1.45.1-azurelinux3.0-distroless"
}

// OtelImage contains the location of the OpenTelemetry container image for MISE
func OTelImage(acrDomain string) string {
	return "mcr.microsoft.com/oss/otel/opentelemetry-collector-contrib:0.95.0-linux-amd64"
}

// TelemetryExporterImage contains the location of the telemetry exporter container image
func TelemetryExporterImage(acrDomain string) string {
	return acrDomain + "/telemetryexporter:v20260622.00@sha256:a946e947f46a83f0515acb7fdd5be878a4300328f37f13e57f69c829c83a3fde"
}

// HolmesImage contains the location of the HolmesGPT investigation container image
func HolmesImage(acrDomain string) string {
	return acrDomain + "/holmesgpt:latest"
}
