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

// InstallStream describes stream we are defaulting to for all new clusters
var InstallStream = &Stream{
	Version:  NewVersion(4, 10, 15),
	PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:ddcb70ce04a01ce487c0f4ad769e9e36a10c8c832a34307c1b1eb8e03a5b7ddb",
}

// UpgradeStreams describes list of streams we support for upgrades
var (
	UpgradeStreams = []*Stream{
		InstallStream,
		{
			Version:  NewVersion(4, 9, 28),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:4084d94969b186e20189649b5affba7da59f7d1943e4e5bc7ef78b981eafb7a8",
		},
		{
			Version:  NewVersion(4, 8, 18),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:321aae3d3748c589bc2011062cee9fd14e106f258807dc2d84ced3f7461160ea",
		},
		{
			Version:  NewVersion(4, 7, 30),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:aba54b293dc151f5c0fd96d4353ced6ced3e7da6620c1c10714ab32d0577486f",
		},
		{
			Version:  NewVersion(4, 6, 44),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:d042aa235b538721a39989b13d7d9d3537af9b57e9fd10f485dd04461932ec85",
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
	return acrDomain + "/genevamdm:master_20220419.1"
}

// MdsdImage contains the location of the MDSD container image
func MdsdImage(acrDomain string) string {
	return acrDomain + "/genevamdsd:master_20220419.1"
}

// MUOImage contains the location of the Managed Upgrade Operator container image
func MUOImage(acrDomain string) string {
	return acrDomain + "/managed-upgrade-operator:aro-b1"
}
