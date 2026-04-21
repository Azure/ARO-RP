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
	Version:  NewVersion(4, 19, 24),
	PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:3ef832b8bb0d56331035ba54af36c36be46d6c6dc1a41e300055692f02bb001d",
}

// FluentbitImage contains the location of the Fluentbit container image
func FluentbitImage(acrDomain string) string {
	// https://github.com/microsoft/azurelinux/releases
	return acrDomain + "/fluentbit:5.0.2-cm20260311@sha256:6b62024a5d92814b6eb2e2ee6ce5b292f1ef9ec20a0c2329b9bb2813c0eb2666"
}

// MdmImage contains the location of the MDM container image
// https://eng.ms/docs/products/geneva/collect/references/linuxcontainers
func MdmImage(acrDomain string) string {
	return acrDomain + "/geneva/distroless/mdm:2.202604071548.0-20260407-1@sha256:390a13ab26a4c90baa9d1a47ef2b502b7ec635840587d89d05120b6952fe680b"
}

// MdsdImage contains the location of the MDSD container image
// https://eng.ms/docs/products/geneva/collect/references/linuxcontainers
func MdsdImage(acrDomain string) string {
	return acrDomain + "/geneva/distroless/mdsd:1.40.3-20260409-1@sha256:1fb51857a0a34e7e7445a91c0a1082d97df235349a66a166a58c86029c80ea89"
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
