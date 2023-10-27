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
var DefaultMinorVersion = 12

// DefaultInstallStreams describes the latest version of our supported streams
var DefaultInstallStreams = map[int]*Stream{
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
	DefaultInstallStreams[11],
	{
		Version:  NewVersion(4, 11, 26),
		PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:1c3913a65b0a10b4a0650f54e545fe928360a94767acea64c0bd10faa52c945a",
	},
	DefaultInstallStreams[12],
}

// FluentbitImage contains the location of the Fluentbit container image
func FluentbitImage(acrDomain string) string {
	return acrDomain + "/fluentbit:1.9.10-cm20231004"
}

// MdmImage contains the location of the MDM container image
// https://eng.ms/docs/products/geneva/collect/references/linuxcontainers
func MdmImage(acrDomain string) string {
	return acrDomain + "/genevamdm:2.2023.928.2134-0de476-20230928t2244"
}

// MdsdImage contains the location of the MDSD container image
// https://eng.ms/docs/products/geneva/collect/references/linuxcontainers
func MdsdImage(acrDomain string) string {
	return acrDomain + "/genevamdsd:mariner_20231024.2"
}

// MUOImage contains the location of the Managed Upgrade Operator container image
func MUOImage(acrDomain string) string {
	return acrDomain + "/app-sre/managed-upgrade-operator:v0.1.952-44b631a"
}

// GateKeeperImage contains the location of the GateKeeper container image
func GateKeeperImage(acrDomain string) string {
	return acrDomain + "/gatekeeper:v3.11.1"
}
