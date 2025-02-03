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
	Version      *Version   `json:"version"`
	PullSpec     string     `json:"-"`
	MasterVmSize api.VMSize `json:"masterVmSize"`
	WorkerVmSize api.VMSize `json:"workervMSize"`
}

// Install stream data for production and INT has moved to RP-Config.
// This default is left here ONLY for use by local development mode,
// until we can come up with a better solution.
var DefaultInstallStream = Stream{
	Version:      NewVersion(4, 15, 35),
	PullSpec:     "quay.io/openshift-release-dev/ocp-release@sha256:8c8433f95d09b051e156ff638f4ccc95543918c3aed92b8c09552a8977a2a1a2",
	MasterVmSize: api.VMSizeStandardD8sV3,
	WorkerVmSize: api.VMSizeStandardD4sV3,
}

// FluentbitImage contains the location of the Fluentbit container image
func FluentbitImage(acrDomain string) string {
	// https://github.com/microsoft/azurelinux/releases
	return acrDomain + "/fluentbit:1.9.10-cm20241208@sha256:fa35a491542b1e531b73658da83e47f0f549786a186f00b0cdaffec86100c980"
}

// MdmImage contains the location of the MDM container image
// https://eng.ms/docs/products/geneva/collect/references/linuxcontainers
func MdmImage(acrDomain string) string {
	return acrDomain + "/distroless/genevamdm:2.2024.1115.1908-5b4aed-20241230t1713@sha256:249a57801d76244f722a739c8bb03cb519cbfbc3ca8356b7da36ffe9084afecd"
}

// MdsdImage contains the location of the MDSD container image
// https://eng.ms/docs/products/geneva/collect/references/linuxcontainers
func MdsdImage(acrDomain string) string {
	return acrDomain + "/distroless/genevamdsd:mariner_20241212.2@sha256:a7a71af5b631ea5a8ad587a09d8680b17719cae25b95de81e8a4d71f2cc55f45"
}

// MUOImage contains the location of the Managed Upgrade Operator container image
func MUOImage(acrDomain string) string {
	return acrDomain + "/app-sre/managed-upgrade-operator:v0.1.952-44b631a"
}

// GateKeeperImage contains the location of the GateKeeper container image
func GateKeeperImage(acrDomain string) string {
	return acrDomain + "/gatekeeper:v3.15.1"
}
