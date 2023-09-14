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

// DefaultMinorVersion describes the minor OpenShift version to default to
var DefaultMinorVersion = 11

// DefaultInstallStreams describes the latest version of our supported streams
var DefaultInstallStreams = map[int]*Stream{
	10: {
		Version:  NewVersion(4, 10, 63),
		PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:340091aefa0bba06bbb99cc58cb1f2b73404c832f72b83c526b8e7677efbecef",
	},
	11: {
		Version:  NewVersion(4, 11, 44),
		PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:52cbfbbeb9cc03b49c2788ac7333e63d3dae14673e01a9d8e59270f3a8390ed3",
	},
	12: {
		Version:  NewVersion(4, 12, 25),
		PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:5a4fb052cda1d14d1e306ce87e6b0ded84edddaa76f1cf401bcded99cef2ad84",
	},
}

// DefaultInstallStream describes stream we are defaulting to for all new clusters
var DefaultInstallStream = DefaultInstallStreams[DefaultMinorVersion]

var AvailableInstallStreams = []*Stream{
	DefaultInstallStreams[10],
	{
		Version:  NewVersion(4, 10, 54),
		PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:7e44fa5f6aa15f9492341c4714bba4dc5089c968f2bf77fb8d4cdf189634f8f5",
	},
	{
		Version:  NewVersion(4, 10, 40),
		PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:b9fad814fb4442e7e852b0614d9bb4e2ebc5e1a2fa51623aa838b4ee0e4a5369",
	},
	DefaultInstallStreams[11],
	{
		Version:  NewVersion(4, 11, 26),
		PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:1c3913a65b0a10b4a0650f54e545fe928360a94767acea64c0bd10faa52c945a",
	},
	DefaultInstallStreams[12],
}

// FluentbitImage contains the location of the Fluentbit container image
func FluentbitImage(acrDomain string) string {
	return acrDomain + "/fluentbit:1.9.10-cm20230805"
}

// MdmImage contains the location of the MDM container image
// https://eng.ms/docs/products/geneva/collect/references/linuxcontainers
func MdmImage(acrDomain string) string {
	return acrDomain + "/genevamdm:2.2023.721.1630-e50918-20230721t1737"
}

// MdsdImage contains the location of the MDSD container image
// https://eng.ms/docs/products/geneva/collect/references/linuxcontainers
func MdsdImage(acrDomain string) string {
	return acrDomain + "/genevamdsd:mariner_20230727.1"
}

// MUOImage contains the location of the Managed Upgrade Operator container image
func MUOImage(acrDomain string) string {
	return acrDomain + "/app-sre/managed-upgrade-operator:v0.1.952-44b631a"
}

// GateKeeperImage contains the location of the GateKeeper container image
func GateKeeperImage(acrDomain string) string {
	return acrDomain + "/gatekeeper:v3.11.1"
}
